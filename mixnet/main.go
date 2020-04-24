package main

import (
	"net"
	"os"
	"github.com/corollari/distributed-homework/onepiece"
	"crypto/rand"
	"math/big"
	"time"
	"sync"
	"fmt"
	// TODO: Add onion encryption
	// ecies "github.com/ecies/go"
)

const maxMessageSize = 1000

// Used to send decoy messages
var knownMixNodes = []string{
	"localhost:5100",
	"localhost:5101",
}

type ForwardMsg struct{
	nextHop string
	msg []byte
}

var (
	nextMsgBatch = []ForwardMsg{}
	MsgBatchMux sync.Mutex
)


func main() {
	if len(os.Args) < 1 {
		panic("provide port number as argument")
	}
	port := os.Args[1]
	pc, err := net.ListenPacket("udp", "0.0.0.0:" + port)
	panicError(err)
	defer pc.Close()

	go scheduleSending()

	for {
		buffer := make([]byte, maxMessageSize)
		n, _, err := pc.ReadFrom(buffer)
        fmt.Println("received message:", string(buffer[:n]))
        if err != nil {
            continue
        }
        go handleRequest(buffer[:n])
	}
}

// Message format
// Decoy: 000000000000
// Real: "hops","msg"
//       ""next_hop",""hop","...""","msg"
func handleRequest(buffer []byte) {
	defer recover() // If there's a problem just ignore this message
	// Decrypt here
	numbers, bytearrays := onepiece.ParseMsg(buffer, 2)
	if len(numbers)==1 && numbers[0] == 0 {
		return //It's a decoy message, drop it
	}
	// Both hops and msg should be padded to prevent any information leaks, but this is not implemented here
	hops := onepiece.GetBytearray(numbers, bytearrays, 0)
	msg := onepiece.GetBytearray(numbers, bytearrays, 1)
	numbers2, bytearrays2 := onepiece.ParseMsg(hops, 2)
	nextHop := onepiece.GetBytearray(numbers2, bytearrays2, 0)
	var nextMsg []interface{}
	if numbers[1] == 0 {
		nextMsg = []interface{}{msg}
	} else {
		otherHops := onepiece.GetBytearray(numbers2, bytearrays2, 1)
		nextMsg = append([]interface{}{otherHops}, msg)
	}
	msgToAdd := ForwardMsg{
		string(nextHop),
		onepiece.EncodeMsg(nextMsg),
	}
	MsgBatchMux.Lock()
	nextMsgBatch = append(nextMsgBatch, msgToAdd)
	MsgBatchMux.Unlock()
}

func scheduleSending(){
	clock := time.Tick(3 * time.Second)
	for {
		<-clock
		MsgBatchMux.Lock()
		go sendMessages(nextMsgBatch)
		nextMsgBatch = []ForwardMsg{} // Empty queue
		MsgBatchMux.Unlock()
	}
}

func sendMessages(batch []ForwardMsg) {
	if len(batch) == 0 {
		return
	}
	defer recover()
	// Add decoy messages
	// Decoy messages are set to only have 1 hop in order to avoid exponential blow-up
	for i:=0; i<getRandom(2*len(batch)); i++ {
		batch = append(batch, ForwardMsg{
			knownMixNodes[getRandom(2)],
			[]byte("000000000000000000"), // Again length of all packets should be made equal but not doing it for now
		})
	}
	shuffle(batch)
	for _, v := range batch {
		go send(v.nextHop, v.msg)
	}
}

func send(recipient string, msg []byte){
	conn, err := net.Dial("udp", recipient)
	if err != nil {
		return
	}
	fmt.Println("sent message:", msg)
	conn.Write(msg)
}

func getRandom(max int) int {
	r, _ := rand.Int(rand.Reader, big.NewInt(int64(max)))
	return int(r.Int64())
}

func panicError(err error){
	if err != nil {
		panic(err)
	}
}

// Implementation of Fisher-Yates shuffle
// https://en.wikipedia.org/wiki/Fisher%E2%80%93Yates_shuffle
// Uses crypto/rand for random numbers because math/rand generates the numbers based on a fixed seed, therefore an attacker could send messages to a mixnet node, observe where their outgoing order, and from a few observations derive the internal state of the PRNG therefore deanonymizing the mixing
func shuffle(msgs []ForwardMsg) {
	for i := range msgs {
		j:= getRandom(i + 1)
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
}
