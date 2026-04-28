package main

import (
	"fmt"
	"log"
	"net"

	"github.com/orisegev/redis-lite/internal/storage"
)

func main() {
	port := ":6379"

	listener, err := net.Listen("tcp", "127.0.0.1"+port)

	db := storage.NewEngine()

	if err != nil {
		log.Fatalf("Unable to start server: %v", err)
	}

	defer listener.Close()

	fmt.Printf("Redits-lite server is running on %s\n", port)

	for {
		conn, err := listener.Accept()

		if err != nil {
			fmt.Printf("Error accepting connection: %v\n", err)
		}

		go handleConnection(conn, db)
	}
}
