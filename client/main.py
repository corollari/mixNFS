import socket
import re
import sys
from random import randint
from time import time

CLIENT_UDP_IP = "127.0.0.1"
CLIENT_UDP_PORT = 5005
SERVER_UDP_IP = "127.0.0.1"
SERVER_UDP_PORT = 5006
TIMEOUT_INTERVAL = 1 # 1 sec

sock = socket.socket(socket.AF_INET, # Internet
                     socket.SOCK_DGRAM) # UDP
sock.bind((CLIENT_UDP_IP, CLIENT_UDP_PORT))

def parseMsg(msg):
    if msg == b"":
        return []
    if msg[0]==ord('"'):
        msg = msg[1:]
        endingDelimiter = re.search(b'[^\\\]"', msg).start()+1
        return [msg[:endingDelimiter].replace(b'\\"', b'"')]+parseMsg(msg[endingDelimiter+2:])
    else:
        try:
            comma = msg.index(b',')
            return [int(msg[:comma])]+parseMsg(msg[comma+1:])
        except:
            return [int(msg)]

def encodeMsg(msg):
    items = msg.copy()
    for i in range(len(items)):
        if type(items[i]) == str:
            items[i] = '"' + items[i].replace('"', '\\"') + '"'
        elif type(items[i]) == bytes:
            items[i] = '"' + items[i].decode("utf-8").replace('"', '\\"') + '"'
        elif type(items[i]) == int:
            items[i] = str(items[i])
        else:
            print("error type", type(items[i]), "cannot be encoded")
    return ",".join(items).encode()

cache = {}

'''
example messages
['chmod', 'file', 511]
['write', 'file', 0, "fr"]
['append', 'file', "fr"]
['read', 'file', 0, 2]
['subscribe', 'file', 1000]
'''

mixnet = False

mixnetNodes = [{
    "ip":"127.0.0.1",
    "port":5100
    },
    {
    "ip":"127.0.0.1",
    "port":5101
    }]

def buildHops(hops):
    hops.reverse()
    encoded = encodeMsg([hops[0]])
    for h in hops[1:]:
        encoded = encodeMsg([h, encoded])
    return encoded

def strIP(ip, port):
    return ip + ":" + str(port)

def sendServer(encodedMsg):
    if mixnet:
        firstHopIndex = randint(0, 1)
        firstHop = mixnetNodes[firstHopIndex]
        secondHop = mixnetNodes[(firstHopIndex+1)%2]
        hops = buildHops([
            strIP(firstHop["ip"], firstHop["port"]),
            strIP(secondHop["ip"], secondHop["port"]),
            strIP(SERVER_UDP_IP, SERVER_UDP_PORT),
            strIP(secondHop["ip"], secondHop["port"]),
            strIP(firstHop["ip"], firstHop["port"])
            ])
        encodedMsg = encodeMsg([hops, encodedMsg])
        print("sent:", encodedMsg)
        sock.sendto(encodedMsg, (firstHop["ip"], firstHop["port"]))
    else:
        print("sent:", encodedMsg)
        sock.sendto(encodedMsg, (SERVER_UDP_IP, SERVER_UDP_PORT))

def getMsgId():
    return randint(0, 2**20)

def sendMessage(msg):
    msgId = getMsgId()
    msg = [msgId] + msg
    encodedMsg = encodeMsg(msg)
    lastSend = 0
    answered = False
    startingTime = time()
    while True:
        currentTime = time()
        if (currentTime - lastSend) > TIMEOUT_INTERVAL and not answered:
            sendServer(encodedMsg)
            lastSend = currentTime
        try:
            data, addr = sock.recvfrom(1024) # buffer size is 1024 bytes
            parsedMsg = parseMsg(data)
            if parsedMsg[1] == b"subscriptionupdate":
                # This goes before any checks because it may be a message from a previous subscription which still hasn't been ack'd
                sendServer(encodeMsg([parsedMsg[0], "ack"]))
            print("received message:", parsedMsg)
            if parsedMsg[0] != msgId:
                continue # Older message, outdated
            answered = True
            if msg[1] == "read":
                cache[msg[2]] = {
                        "offset": msg[3],
                        "length": msg[4],
                        "content": parsedMsg[2],
                        "lastValidityCheck": currentTime,
                        "lastWrite": parsedMsg[3]
                        }
            if msg[1] == "subscribe":
                if (currentTime - startingTime) > (msg[3]/1000):
                    return
            else:
                return parsedMsg
        except:
            pass

CACHE_INTERVAL = 30

def handleCache(msg):
    if msg[1] in cache:
        cachedFile = cache[msg[1]]
        if cachedFile["offset"]<=msg[2] and (cachedFile["offset"]+cachedFile["length"])>=(msg[2]+msg[3]): # Cache contains a subset of request
            if (time()-cachedFile["lastValidityCheck"]) < CACHE_INTERVAL:
                print("content cached:", cachedFile["content"])
            else:
                lastWrite = sendMessage(["lastWrite", msg[1]])[2]
                if lastWrite == cachedFile["lastWrite"]:
                    cachedFile["lastValidityCheck"] = time()
                    print("content cached:", cachedFile["content"])
                else:
                    sendMessage(msg)
    else:
        sendMessage(msg)

def main():
    if len(sys.argv)>1 and sys.argv[1]=="mixnet":
        global mixnet
        mixnet = True
        print("Mixnet mode")
    sock.setblocking(0)
    while True:
        #msg = input("Input message: ")
        msg = ['read', '.', 0, 2]
        msg = ['subscribe', 'file', 5000]
        if msg[0] == "read":
            handleCache(msg)
            handleCache(msg)
        else:
            sendMessage(msg)
        break

if __name__ == '__main__':
    main()
