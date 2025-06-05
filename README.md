# gocket-room-chatter

A simple WIP chat application written in Go, featuring a WebSocket-based server and a basic client.

## Features

- WebSocket server for real-time communication
- Room-based chat structure
- Simple client and server separation

## Project Structure

```
client/   # Client-side code (WIP)
server/   # Server-side code
```

## Server

- Handles WebSocket connections
- Manages chat rooms and users
- Uses [gorilla/websocket](https://github.com/gorilla/websocket)

## Client

- Basic structure for connecting to the server (WIP)

## How to Run

1. Build and run the server:

   ```sh
   cd server
   go run main.go
   ```

2. (WIP) Build and run the client:

   ```sh
   cd client
   go run main.go
   ```
