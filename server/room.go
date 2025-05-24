package server

type Room struct {
	RoomName string
	RoomId   int
	Users    map[int]*ConnectionData
}

func newRoom(roomName string, roomId int) *Room {
	return &Room{
		RoomName: roomName,
		RoomId:   roomId,
		Users:    make(map[int]*ConnectionData),
	}
}

func (r *Room) addUser(connData *ConnectionData) {
	r.Users[connData.ConnId] = connData
}
