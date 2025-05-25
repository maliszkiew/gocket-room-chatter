package server

import "github.com/gorilla/websocket"

type ConnectionData struct {
	Username string
	Conn     *websocket.Conn
	ConnId   int
	RoomId   int
	Send     chan []byte
}

func NewConnectionData(username string, conn *websocket.Conn, connId int, roomId int) *ConnectionData {
	return &ConnectionData{
		Username: username,
		Conn:     conn,
		ConnId:   connId,
		RoomId:   roomId,
		Send:     make(chan []byte, 256),
	}
}
