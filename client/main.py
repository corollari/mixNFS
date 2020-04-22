import socket
import re
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
        else:
            items[i] = str(items[i])
    return ",".join(items).encode()

cache = {}

def getLocalTime(t):
    # TODO
    return t

'''
example messages
[1, 'chmod', 'file', 511]
[1, 'write', 'file', 0, "fr"]
[1, 'append', 'file', "fr"]
[1, 'read', 'file', 0, 2]
[1, 'subscribe', 'file', 1000]
'''

def sendMessage(msg):
    msgId = randint(0, 2**20)
    msg = [msgId] + msg
    encodedMsg = encodeMsg(msg)
    print("message to send", encodedMsg)
    lastSend = 0
    answered = False
    startingTime = time()
    while True:
        currentTime = time()
        if (currentTime - lastSend) > TIMEOUT_INTERVAL and not answered:
            sock.sendto(encodedMsg, (SERVER_UDP_IP, SERVER_UDP_PORT))
            lastSend = currentTime
        try:
            data, addr = sock.recvfrom(1024) # buffer size is 1024 bytes
            parsedMsg = parseMsg(data)
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
    sock.setblocking(0)
    while True:
        #msg = input("Input message: ")
        msg = ['read', 'file', 0, 2]
        #msg = [msgId, 'subscribe', 'file', 5000]
        if msg[0] == "read":
            handleCache(msg)
            handleCache(msg)
        else:
            sendMessage(msg)
        break

if __name__ == '__main__':
    main()
