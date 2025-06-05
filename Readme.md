# hub_channel_pattern

An efficient, production-grade implementation of the singleton pattern in Go, leveraging Redis and channels for scalable, concurrent WebSocket communication.

## Overview

`hub_channel_pattern` is a Go server that acts as a scalable WebSocket hub, allowing multiple clients to connect, send, and receive messages in real time. The core of this design is a concurrency-safe, singleton `RedisHub` that manages client registrations, message broadcasting, and session state using Redis as a backend. Channels and goroutines are used extensively to ensure efficient message delivery and heartbeat monitoring.

This pattern is ideal for chat applications, live dashboards, or any system requiring real-time, distributed client communication with robust connection management.

---

## Architecture

### Main Components

- **main.go**: Entry point. Initializes `RedisHub`, starts the HTTP/WebSocket server, and manages client connections.
- **hub.go**: Implements the singleton `RedisHub`, encapsulating all Redis interactions, client registration, state management, and message broadcasting.
- **client.go**: Defines the `Client` structure and logic for handling WebSocket communication, including reading, writing, and broadcasting messages.

### Key Patterns & Technologies

- **Singleton Pattern**: Only one `RedisHub` is instantiated, ensuring a single point of coordination for all client activity.
- **Channels & Goroutines**: Used for concurrent message passing and asynchronous client operations.
- **Redis Pub/Sub**: Enables message broadcasting across distributed server instances, supporting horizontal scaling.
- **Heartbeat Mechanism**: Each client connection sends/receives heartbeats to ensure liveness and enable cleanup of stale connections.

---

## Detailed Workflow

### 1. Server Startup

- The application reads the Redis connection URL (from `REDIS_URL` environment variable, defaulting to `redis://localhost:6379`).
- It creates a singleton instance of `RedisHub`, connecting to Redis and subscribing to the `"broadcast"` channel.
- HTTP server is started on port `8080`, with a `/ws` endpoint for WebSocket upgrades.

### 2. Client Connection Lifecycle

- On new WebSocket connection:
  - A unique `clientID` is generated.
  - A `Client` instance is created and registered with the `RedisHub`.
  - Connection metadata (e.g., IP, timestamp) is stored in Redis.
  - Reader and writer goroutines (`readPump`/`writePump`) are started for this client.

### 3. Message Handling

- **Reading**: The client’s `readPump` goroutine listens for incoming WebSocket messages, parses them (supporting both JSON and plain text), and calls `RedisHub.BroadcastMessage`.
- **Broadcasting**: The `RedisHub` serializes the message and publishes it to the `"broadcast"` Redis channel.
- **Receiving**: All connected clients have a `MessageBroadcaster` listening to `"broadcast"`; messages are delivered to all clients except the sender.
- **Writing**: The client’s `writePump` goroutine sends messages (from `send` channel) to the WebSocket connection and periodically sends pings to maintain the connection.

### 4. Client State & Heartbeat

- On every ping/pong, the client’s heartbeat is updated in Redis (`heartbeat:<clientID>`), allowing for detection and cleanup of disconnected clients.
- On disconnect, the client is unregistered and its Redis state is cleaned up.

---

## Code Example

```go
// main.go (excerpt)
func main() {
    redisURL := os.Getenv("REDIS_URL")
    hub := NewRedisHub(redisURL)
    defer hub.Close()
    http.HandleFunc("/ws", wsHandler(hub))
    log.Println("Server is running on port 8080")
    http.ListenAndServe(":8080", nil)
}
```

---

## Features

- **Concurrency-safe singleton hub** for all client management and broadcasting.
- **WebSocket server** with hot client registration/unregistration.
- **Redis-backed state** for scalable, distributed deployments.
- **Heartbeat and cleanup** for reliable connection management.
- **Simple, extensible codebase** suitable for real-time systems.

---

## Getting Started

### Prerequisites

- [Go](https://golang.org/dl/) 1.18+
- [Redis](https://redis.io/) server

### Installation

```bash
git clone https://github.com/LikhithMar14/hub_channel_pattern.git
cd hub_channel_pattern
```

### Running

```bash
export REDIS_URL=redis://localhost:6379  # or your redis instance
go run main.go
```
Open `ws://localhost:8080/ws` with a WebSocket client.

---

## File Structure

```
main.go      // Entry point and server
hub.go       // Singleton RedisHub implementation
client.go    // Client connection and messaging logic
```

---

## Contributing

Contributions are welcome! Please open issues or submit PRs for improvements and new features.

## License

*No license specified.*  
To open source this project, consider adding a license file ([choose a license](https://choosealicense.com/)).

## Author

- [LikhithMar14](https://github.com/LikhithMar14)

---

⭐️ Star this repo if you found it useful!