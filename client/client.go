package main

/*import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"net/url"
	"sync"
)*/

type Client struct {
	Username string
	Ip       string
	Port     string
	Send     chan []byte
	Recv     chan []byte
}

func (c *Client) connect(ip string, port string, username string) {
	/*	url := url.URL{Scheme: "ws", Host: ip + ":" + port, Path: "/ws"}*/

}
