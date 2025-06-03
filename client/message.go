package main

import "encoding/json"

type Message struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

func NewMessage(msgType string, data interface{}) Message {
	return Message{
		Type: msgType,
		Data: data,
	}
}

func (m Message) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}
