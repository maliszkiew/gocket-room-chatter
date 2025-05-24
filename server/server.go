package server

import (
	"container/list"
	"encoding/json"
	"github.com/gorilla/websocket"
	"net/http"
)

type Server struct {
	port             string
	broadcast        chan []byte
	connections      map[int]*ConnectionData
	maxConnections   int
	availableConnIds list.List
	availableRoomIds list.List
	rooms            map[int]*Room
	roomsMap         map[string]*Room
	upgrader         websocket.Upgrader
	register         chan *ConnectionData
	unregister       chan *ConnectionData
}

func NewServer() *Server {
	const maxConns = 2048
	const maxRooms = 1024
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
		maxConnections: maxConns,
		availableConnIds: func(max int) list.List {
			l := list.List{}
			for i := 0; i < max; i++ {
				l.PushBack(i)
			}
			return l
		}(maxConns),
		availableRoomIds: func(max int) list.List {
			l := list.List{}
			for i := 0; i < max; i++ {
				l.PushBack(i)
			}
			return l
		}(maxRooms),
		broadcast:  make(chan []byte, 1024),
		register:   make(chan *ConnectionData, 1024),
		unregister: make(chan *ConnectionData, 1024),
	}
}

func (s *Server) Start(port string) {
	s.port = port
	http.HandleFunc("/ws", s.handleWebSocket)
	http.HandleFunc("/rooms", s.handleRooms)
	http.HandleFunc("/join", s.handleJoinRoom)
	http.HandleFunc("/leave", s.handleLeaveRoom)
	http.HandleFunc("/create", s.handleCreateRoom)
	go func() {
		if err := http.ListenAndServe(s.port, nil); err != nil {
			panic(err)
		}
	}()

}

func (s *Server) assignConnId() int {
	connId := s.availableConnIds.Front()
	s.availableConnIds.Remove(connId)
	return connId.Value.(int)
}

func (s *Server) assignRoomId() int {
	roomId := s.availableRoomIds.Front()
	s.availableRoomIds.Remove(roomId)
	return roomId.Value.(int)
}

func (s *Server) handleWebSocket(writer http.ResponseWriter, request *http.Request) {
	conn, err := s.upgrader.Upgrade(writer, request, nil)
	if err != nil {
		http.Error(writer, "Connection failed. Upgrading WebSocket fatal error.", http.StatusInternalServerError)
		return
	}
	username := request.URL.Query().Get("username")
	connId := s.assignConnId()
	connData := newConnectionData(username, conn, connId, -1) // RoomId is -1 initially
	s.connections[connId] = connData
	s.register <- connData
	defer conn.Close()
}

func (s *Server) handleRooms(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "application/json")
	if len(s.roomsMap) == 0 {
		writer.Write([]byte("[]"))
		return
	}
	var roomsList []map[string]interface{}
	for _, room := range s.roomsMap {
		roomData := map[string]interface{}{
			"roomName":  room.RoomName,
			"roomId":    room.RoomId,
			"userCount": len(room.Users),
		}
		roomsList = append(roomsList, roomData)
	}
	response, err := json.Marshal(roomsList)
	if err != nil {
		http.Error(writer, "Failed to fetch Room list. Server fatal error.", http.StatusInternalServerError)
		return
	}
	writer.Write(response)
}
func (s *Server) handleCreateRoom(writer http.ResponseWriter, request *http.Request) {
	roomName := request.URL.Query().Get("roomName")
	roomId := s.assignRoomId()
	s.rooms[roomId] = newRoom(roomName, roomId)
	s.roomsMap[roomName] = s.rooms[roomId]
	writer.WriteHeader(http.StatusCreated)
}
func (s *Server) handleJoinRoom(writer http.ResponseWriter, request *http.Request) {

}

func (s *Server) handleLeaveRoom(writer http.ResponseWriter, request *http.Request) {

}
