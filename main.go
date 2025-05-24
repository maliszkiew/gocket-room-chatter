package main

import (
	"gocket-room-chatter/server"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	s := server.NewServer()
	s.Start(":8080")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
}
