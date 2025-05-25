package server

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"sync"
	"time"
)

type Server struct {
	port        string
	connections map[int]*ConnectionData
	rooms       map[int]*Room
	roomsMap    map[string]*Room
	upgrader    websocket.Upgrader
	register    chan *ConnectionData
	unregister  chan *ConnectionData
	broadcast   chan []byte
	mutex       sync.RWMutex
	nextConnID  int
	nextRoomID  int
}

func NewServer() *Server {
	return &Server{
		connections: make(map[int]*ConnectionData),
		rooms:       make(map[int]*Room),
		roomsMap:    make(map[string]*Room),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		register:   make(chan *ConnectionData, 256),
		unregister: make(chan *ConnectionData, 256),
		broadcast:  make(chan []byte, 256),
		nextConnID: 1,
		nextRoomID: 1,
	}
}

func (s *Server) Start(port string) {
	s.port = port
	go s.run()
	http.HandleFunc("/ws", s.handleWebSocket)
	go func() {
		log.Printf("Starting server on port %s", port)
		http.ListenAndServe(":"+port, nil)
	}()
}

func (s *Server) run() {
	for {
		select {
		case conn := <-s.register:
			s.handleRegister(conn)

		case message := <-s.broadcast:
			s.handleBroadcast(message)

		case conn := <-s.unregister:
			s.handleUnregister(conn)
		}
	}
}

func (s *Server) handleRegister(conn *ConnectionData) {
	s.mutex.Lock()
	s.connections[conn.ConnId] = conn
	s.mutex.Unlock()

	log.Printf("Client {name: %s, id: %d} connected", conn.Username, conn.ConnId)
	welcome := NewMessage("info", "Welcome to the Gocket Room Chatter!")
	s.sendToConnection(conn, welcome)
}

func (s *Server) handleUnregister(conn *ConnectionData) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if conn.RoomId != -1 {
		if room, exists := s.rooms[conn.RoomId]; exists {
			delete(room.Users, conn.ConnId)
			leftMsg := NewMessage("user_left", map[string]interface{}{
				"username": conn.Username,
				"roomId":   conn.RoomId,
			})
			s.broadcastToRoom(room, leftMsg)
		}
	}
	close(conn.Send)
	delete(s.connections, conn.ConnId)
}

func (s *Server) handleBroadcast(message []byte) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	for _, conn := range s.connections {
		if conn.Send != nil {
			conn.Send <- message
		}
	}
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error upgrading to WebSocket: %v", err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}
	username := r.URL.Query().Get("username")
	connId := s.getNextConnID()
	connData := NewConnectionData(username, conn, connId, -1)
	s.register <- connData

	go s.readMessages(connData)
	go s.writeMessages(connData)
}

func (s *Server) readMessages(conn *ConnectionData) {
	for {
		_, rawMessage, err := conn.Conn.ReadMessage()
		if err != nil {
			log.Printf("Error reading message: %v", err)
			break
		}
		log.Printf("Message [%s] received from {name: %s, id: %d}", string(rawMessage), conn.Username, conn.ConnId)

		var msg Message
		json.Unmarshal(rawMessage, &msg)
		s.handleMessage(conn, msg)
	}
}

func (s *Server) writeMessages(conn *ConnectionData) {
	defer conn.Conn.Close()
	for {
		message, ok := <-conn.Send
		if !ok {
			conn.Conn.WriteMessage(websocket.CloseMessage, []byte{})
			return
		}
		conn.Conn.WriteMessage(websocket.TextMessage, message)
	}
}

func (s *Server) handleMessage(conn *ConnectionData, msg Message) {
	log.Printf("Received message of type %s from %s", msg.Type, conn.Username)

	switch msg.Type {
	case "get_rooms":
		s.handleGetRooms(conn)
	case "create_room":
		s.handleCreateRoom(conn, msg.Data)
	case "join_room":
		s.handleJoinRoom(conn, msg.Data)
	case "leave_room":
		s.handleLeaveRoom(conn)
	case "chat_message":
		s.handleChatMessage(conn, msg.Data)
	default:
		log.Printf("Invalid message type: %s", msg.Type)
	}
}

func (s *Server) handleGetRooms(conn *ConnectionData) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	roomsList := make([]map[string]interface{}, 0)

	for _, room := range s.roomsMap {
		roomData := map[string]interface{}{
			"roomName":  room.RoomName,
			"roomId":    room.RoomId,
			"userCount": len(room.Users),
		}
		roomsList = append(roomsList, roomData)
	}

	response := Message{
		Type: "rooms_list",
		Data: roomsList,
	}
	s.sendToConnection(conn, response)
}

func (s *Server) handleCreateRoom(conn *ConnectionData, data interface{}) {
	dataMap, ok := data.(map[string]interface{})

	roomName, ok := dataMap["roomName"].(string)
	if !ok || roomName == "" {
		s.sendErrorToConnection(conn, "Invalid room name")
		return
	}

	s.mutex.Lock()
	roomId := s.getNextRoomID()
	room := NewRoom(roomName, roomId)
	s.rooms[roomId] = room
	s.roomsMap[roomName] = room
	s.mutex.Unlock()

	response := NewMessage("room_created", map[string]interface{}{
		"roomId":   roomId,
		"roomName": roomName,
	})

	log.Printf("Room created: %s with ID: %d", roomName, roomId)
	s.sendToConnection(conn, response)
}

func (s *Server) sendErrorToConnection(conn *ConnectionData, errorMsg string) {
	errorResponse := NewMessage("error", errorMsg)
	s.sendToConnection(conn, errorResponse)
}

func (s *Server) handleJoinRoom(conn *ConnectionData, data interface{}) {
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		s.sendErrorToConnection(conn, "Invalid data format")
		return
	}

	roomName, ok := dataMap["roomName"].(string)
	if !ok || roomName == "" {
		s.sendErrorToConnection(conn, "Invalid room name")
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	room, exists := s.roomsMap[roomName]
	if !exists {
		s.sendErrorToConnection(conn, "Room does not exist")
		return
	}

	if conn.RoomId != -1 {
		if currentRoom, exists := s.rooms[conn.RoomId]; exists {
			delete(currentRoom.Users, conn.ConnId)

		}
	}

	conn.RoomId = room.RoomId
	room.Users[conn.ConnId] = conn

	response := NewMessage("joined_room", map[string]interface{}{
		"roomId":   room.RoomId,
		"roomName": room.RoomName,
	})

	s.sendToConnection(conn, response)

	joinMsg := NewMessage("user_joined", map[string]interface{}{
		"username": conn.Username,
		"roomId":   room.RoomId,
	})

	s.broadcastToRoom(room, joinMsg)
}

func (s *Server) handleLeaveRoom(conn *ConnectionData) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if room, exists := s.rooms[conn.RoomId]; exists {
		delete(room.Users, conn.ConnId)
		leaveMsg := NewMessage("user_left", map[string]interface{}{
			"username": conn.Username,
			"roomId":   conn.RoomId,
		})
		s.broadcastToRoom(room, leaveMsg)
	}

	conn.RoomId = -1
	response := NewMessage("left_room", map[string]interface{}{
		"message": "Left room successfully",
	})

	s.sendToConnection(conn, response)
}

func (s *Server) handleChatMessage(conn *ConnectionData, data interface{}) {
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		s.sendErrorToConnection(conn, "Invalid data format")
		return
	}

	message, ok := dataMap["message"].(string)
	if !ok {
		s.sendErrorToConnection(conn, "Invalid message")
		return
	}

	s.mutex.RLock()
	room, exists := s.rooms[conn.RoomId]
	s.mutex.RUnlock()

	if !exists {
		s.sendErrorToConnection(conn, "Not in a room")
		return
	}

	chatMsg := NewMessage("chat_message", map[string]interface{}{
		"roomId":   conn.RoomId,
		"username": conn.Username,
		"message":  message,
	})

	s.broadcastToRoom(room, chatMsg)
}

func (s *Server) sendToConnection(conn *ConnectionData, msg Message) {
	data, _ := json.Marshal(msg)

	log.Printf("Sending to {username: %s, id: %d} message: %s", conn.Username, conn.ConnId, string(data))

	select {
	case conn.Send <- data:
		log.Printf("Message sent to  {username: %s, id: %d} successfully", conn.Username, conn.ConnId)
	case <-time.After(5 * time.Second):
		log.Printf("Timeout sending message to {username: %s, id: %d}", conn.Username, conn.ConnId)
		s.unregister <- conn
	default:
		log.Printf("Failed to send message to {username: %s, id: %d}", conn.Username, conn.ConnId)
		s.unregister <- conn
	}
}

func (s *Server) broadcastToRoom(room *Room, msg Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error marshaling broadcast message: %v", err)
		return
	}

	for _, conn := range room.Users {
		select {
		case conn.Send <- data:
		default:
			close(conn.Send)
			delete(room.Users, conn.ConnId)
			s.mutex.Lock()
			delete(s.connections, conn.ConnId)
			s.mutex.Unlock()
		}
	}
}

func (s *Server) getNextConnID() int {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	id := s.nextConnID
	s.nextConnID++
	return id
}

func (s *Server) getNextRoomID() int {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	id := s.nextRoomID
	s.nextRoomID++
	return id
}
