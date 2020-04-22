# distributed-homework

## Requirements
- Server
	- 
- Client
	- Cache (+ update algorithm (will require clock sync))
- Support for both at-most-once and at-least-once

## Basic bitch
- Distribute server (among clients? yay) and implement eventual consistency (BASE)
	- DHT with electric routing
	- Make server run over WebRTC and let people on web browsers become clients by just visiting a website
- Implement QUIC?
- Add mixnet
- Use sphinx?

## Usage
### Client
```
cd client
python main.py # Run client
python tests.py # Run client tests
```

### Server
```
cd server
go run main.go # Run server
```
