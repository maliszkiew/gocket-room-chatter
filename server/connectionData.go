package server

import "github.com/gorilla/websocket"

type ConnectionData struct {
	Username string
	Conn     *websocket.Conn
	ConnId   int
	RoomId   int
}

func newConnectionData(username string, conn *websocket.Conn, connId int, roomId int) *ConnectionData {
	return &ConnectionData{
		Username: username,
		Conn:     conn,
		ConnId:   connId,
		RoomId:   roomId,
	}
}
