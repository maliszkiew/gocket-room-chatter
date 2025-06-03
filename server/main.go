package server

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	s := NewServer()
	s.Start(":8080")

	log.Println("Server started on port 8080")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Server shutting down...")
}
