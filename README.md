# distributed-homework

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

