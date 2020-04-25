# MixNFS

A simple Network File System (NFS) implementation that uses it's own custom fixed-time parseable serialization format and a mixnet for traffic anonymization

## Serialization format (One Piece)
Based on my previous experiences designing protocols and the trends I’ve seen play out in the industry I decided to optimize for the following characteristics when architecting the format:
- **Simple**: A major source of inspiration in this design comes from the essay “Worse is Better”^[1], which posits that simplicity, especially implementation simplicity,  is a key contributor to the success of a system, and should be held over other qualities such as completeness or consistency. I believe that this is especially true in the case of a serialization protocol, as one of its major requirements for success is its availability on a plethora of systems^[2], which requires a lot of implementations. Furthermore implementation complexity is usually correlated with number of bugs, and minimizing that is a must for a system connected to the internet that has to handle inputs that can be directly crafted by an attacker.
- **Human-readable**: Human-readable formats have seen usage orders of magnitude greater than the usage of binary formats^[3], which attests to the idea that most developers find them preferable, as, while binary formats might be more efficient, human-readable formats are much more easier to consume and the applications using them can be debugged with ease due to the fact that artifacts can be inspected directly.
- **Parseable in fixed time**: Most, if not all, of the parsers used by other serialization formats have been coded with branching conditions and other structures which create a dependence between the time spent decoding a message and the contents of the message itself. This opens a security hole that could be exploited by an attacker with network-level access, as the attacker could monitor the packets coming and going from the server and then use timing information to derive some knowledge on the contents of the packets. Clearly, this problem falls out of the scope of the security models of the vast majority of protocols, but there will be some that need to consider this, and while they could write a fixed-time parser for a common serialization format like json, this will be much harder and error-prone than using a serialization format that has been created with that in mind.

With these goals in mind, I came up with a system that serializes messages as lists of two fundamental types, positive integers and bytearrays. That is, all messages are encoded as a comma separated list of integers, represented in the decimal form, and bytearrays, represented by their raw data and delimited by ‘”’ (‘”’ symbols inside the bytearray are escaped with ‘\’). The following examples should help illustrate the concept:

| Json                                          | One Piece                               |
|-----------------------------------------------|-----------------------------------------|
| [1]                                           | 1                                       |
| [1, “Hello world!”]                           | 1,“Hello world!”                        |
| [“Non-string bytes are encoded like:”, “\\t”] | “Non-string bytes are encoded like:”,“	” |

As you can see, the format resembles array declarations in most languages, making it highly intuitive to anyone that has been exposed to them previously.  Also note that this scheme is easily extensible, as new items can be directly added at the end of the list while maintaining backwards compatibility. For example, if suddenly a protocol wanted to add an extra parameter to its answers, it could simply append it at the end of the list and the clients consuming its responses wouldn’t need to be upgraded.
Furthermore, messages using this scheme could be padded with an arbitrary number of characters through the addition, somewhere in the list, of an extra number composed of an arbitrary number of zeroes. This trick can be used to hide the real length of the message, which is leaked by network packets and the parsing algorithm. Example:

| Json   | One Piece                            |
|--------|--------------------------------------|
| [4, 0] | 4,0000000000000000000000000000000000 |

## Usage
### Client
```
cd client
python main.py # Run client in client-server mode
python main.py mixnet # Run client in mixnet mode
python tests.py # Run client tests
```

### Server
```
cd server
go run main.go # Run server with at-least-once semantics
go run main.go at-most-once # Run server with at-most-once semantics
go run main.go at-most-once n # Run with n% packet loss (artificial)
go run main.go at-most-once n mixnet # Run in mixnet mode
go test # Run tests
```

### Mixnet
```
cd mixnet
go run main.go 5000 # Run node (argument is port number)
```

## Why is the serialization format called onepiece?
Because it encodes messages as an unlimited list of items, the same cardinal that the list of one piece episodes has (unlimited).

## Ideas that would be cool to implement
- Distribute server (among clients? yay) and implement eventual consistency (BASE)
	- DHT with electric routing
	- Make server run over WebRTC and let people on web browsers become clients by just visiting a website
- Implement QUIC?
- Add mixnet
- Use sphinx?

