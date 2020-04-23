package main

import (
	"net"
	"fmt"
	"strconv"
	"github.com/fsnotify/fsnotify"
	"os"
	"time"
	"sync"
	"math/rand"
	"github.com/corollari/distributed-homework/server/onepiece"
)

const maxMessageSize = 1000
const maxNumItems = 10
const serverAddr = "0.0.0.0:5006"

var responses = map[int]([]byte){}
var responsesMux sync.Mutex
var acks = map[int](chan int){}
var acksMux sync.RWMutex
var rateSendFailures = 0

func main(){
	/*
	b := []byte("\"abc\\\\\"d\",156")
	fmt.Println(parseMsg(b, 2))
	u := make([]interface{}, 2)
	u[0]=1
	u[1]=b
	fmt.Println(string(encodeMsg(u)))
	*/
	startServer(os.Args[1:])
}

func startServer(args []string){
	filterDuplicates := false
	if len(args)>0 && args[0] == "at-most-once" {
		filterDuplicates = true
		fmt.Println("Invocation semantics: At most once")
	} else {
		fmt.Println("Invocation semantics: At least once")
	}
	if len(args)>1 {
		rateArg, err := strconv.Atoi(args[1])
		rateSendFailures = rateArg
		if err != nil {
			panic(err)
		}
		if rateSendFailures > 100 || rateSendFailures < 0 {
			panic("Failure rate should be in [0, 100]")
		}
	}
	fmt.Printf("Rate of send failures set to %v%%\n", rateSendFailures)
	pc, err := net.ListenPacket("udp", serverAddr)
	if err != nil {
		panic(err)
	}
	defer pc.Close()
	fmt.Println("Server started at", serverAddr)

	for {
		buffer := make([]byte, maxMessageSize) // It's not possible to re-use the same buffer because then it could get overwritten by a different message while on answer()
		n, addr, err := pc.ReadFrom(buffer)
		fmt.Println("received message:", string(buffer[:n]))
		if err != nil {
			continue
		}
		go answer(pc, addr, buffer[:n], filterDuplicates)
	}
}

func checkDuplicates(msgId int) (response []byte, exists bool){
	responsesMux.Lock()
	defer responsesMux.Unlock()
	response, exists = responses[msgId]
	return
}

// answer format
// error -> "error","explanation of error"
// success -> "ok",params.. 
func answer(pc net.PacketConn, addr net.Addr, buffer []byte, filterDuplicates bool) {
	defer func(){
		if r:=recover(); r!=nil{
			//If this gets triggered -> error parsing, just ignore the message then
			fmt.Println("Parsing error triggered")
		}
	}()

	numbers, bytearrays := onepiece.ParseMsg(buffer, maxNumItems)
	msgId := numbers[0]

	defer func(){
		// If an error is triggered send error msg to client
		if errorMessage := recover(); errorMessage != nil {
			sendError(msgId, pc, addr, errorMessage.(string))
			fmt.Println("Error triggered:", errorMessage)
		}
	}()
	if filterDuplicates {
		response, exists := checkDuplicates(msgId)
		if exists {
			fmt.Println("Received duplicate message with id:", msgId)
			send(pc, addr, response)
			return
		}
	}
	operation := string(onepiece.GetBytearray(numbers, bytearrays, 1))
	if operation == "ack" {
		acksMux.RLock()
		defer acksMux.RUnlock()
		acks[msgId]<-1 // If this fails because acks[msgId] doesn't exist or is closed a panic will be triggered and the goroutine cleaned, no harm
		return
	}
	pathname := string(onepiece.GetBytearray(numbers, bytearrays, 2))
	f, err := os.OpenFile(pathname, os.O_RDWR, 0777)
	checkError(err, "error opening file (does it exist?)")
	defer f.Close()
	switch operation {
	case "read":
		offset := numbers[3]
		length := numbers[4]
		_, err := f.Seek(int64(offset), 0)
		checkError(err, "error seeking")
		fileBuffer := make([]byte, length) //Not secure, length must be bounded or this could take up a huge amount of memory. Not impelmenting checks here to make the code cleaner
		n, err := f.Read(fileBuffer)
		checkError(err, "error reading file")
		lastWrite := getLastWrite(f)
		sendMessage(msgId, pc, addr, []interface{}{fileBuffer[:n], lastWrite})
	case "write":
		offset := numbers[3]
		content := onepiece.GetBytearray(numbers, bytearrays, 4)
		_, err := f.Seek(int64(offset), 0)
		checkError(err, "error seeking")
		_, err = f.Write(content)
		checkError(err, "error writing file")
		sendMessage(msgId, pc, addr, []interface{}{})
	case "lastWrite":
		info, err := f.Stat()
		checkError(err, "failed to stat file")
		sendMessage(msgId, pc, addr, []interface{}{int(info.ModTime().Unix())})
	case "chmod":
		//idempotent
		err = f.Chmod(os.FileMode(numbers[3]))
		checkError(err, "mode cannot be changed")
		sendMessage(msgId, pc, addr, []interface{}{})
	case "append":
		//non-idempotent
		content := onepiece.GetBytearray(numbers, bytearrays, 3)
		_, err = f.Seek(0, 2)
		checkError(err, "error seeking")
		_, err = f.Write(content)
		checkError(err, "error writing file")
		sendMessage(msgId, pc, addr, []interface{}{})
	case "subscribe":
		duration := numbers[3]
		end := time.After(time.Duration(duration) * time.Millisecond)
		watcher, err := fsnotify.NewWatcher()
		checkError(err, "watcher error")
		defer watcher.Close()
		err = watcher.Add(pathname)
		checkError(err, "file cannot be watched")
		sendMessage(msgId, pc, addr, []interface{}{}) // send "ok" message, subscription sucessful
		for {
			select {
				case event, ok := <-watcher.Events:
					if !ok {
						return
					}
					if event.Op&fsnotify.Write == fsnotify.Write {
						info, err := f.Stat()
						checkError(err, "file cannot be stat'd")
						fileBuffer := make([]byte, info.Size())
						f.Seek(0, 0)
						n, err := f.Read(fileBuffer)
						checkError(err, "read failure")
						go sendRecurrentMessage(pc, addr, []interface{}{"subscriptionupdate", fileBuffer[:n]}) //Possibly sending less because the file might have changed since we stat'd it
					}
				case <- end:
					return
			}
		}
	}
}

// Doesn't add an ok like other functions
func sendRecurrentMessage(pc net.PacketConn, addr net.Addr, msg []interface{}) {
	msgId := rand.Intn(1048576) // Equal to 2**20, just a big number to make sure there are no collisions
	acked := make(chan int)
	acksMux.Lock()
	acks[msgId] = acked // Doesn't require mutexes 
	acksMux.Unlock()
	resendTimeout := time.Tick(1 * time.Second)
	stopTimeout := time.After(5 * time.Second)
	msg = append([]interface{}{msgId}, msg...)
	encodedMsg := onepiece.EncodeMsg(msg)
	send(pc, addr, encodedMsg)
	for {
		select {
		case <-acked:
			close(acked) // Make sure that any attempts to write on it fail so no thread gets hanged up
			return
		case <-resendTimeout:
			send(pc, addr, encodedMsg)
		case <-stopTimeout:
			return
		}
	}
}

func getLastWrite(f *os.File) int {
	info, err := f.Stat()
	checkError(err, "failed to stat file")
	return int(info.ModTime().Unix())
}

func checkError(err error, errorMessage string) {
	if err != nil {
		panic(errorMessage)
	}
}

func saveResponse(msgId int, response []byte){
	responsesMux.Lock()
	defer responsesMux.Unlock()
	responses[msgId] = response
	return
}

func send(pc net.PacketConn, addr net.Addr, encodedMsg []byte){
	if rateSendFailures < rand.Intn(100) {
		pc.WriteTo(encodedMsg, addr)
		fmt.Println("sent message:", encodedMsg)
	} else {
		fmt.Println("Failed message send:", encodedMsg)
	}
}

func sendError(msgId int, pc net.PacketConn, addr net.Addr, err string){
	encodedMsg := onepiece.EncodeMsg([]interface{}{msgId, "error", err})
	saveResponse(msgId, encodedMsg)
	send(pc, addr, encodedMsg)
}

func sendMessage(msgId int, pc net.PacketConn, addr net.Addr, msg []interface{}){
	msg = append([]interface{}{msgId, "ok"}, msg...)
	encodedMsg := onepiece.EncodeMsg(msg)
	saveResponse(msgId, encodedMsg)
	send(pc, addr, encodedMsg)
}

