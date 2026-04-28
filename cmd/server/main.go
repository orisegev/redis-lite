package main

import (
	"fmt"
	"log"
	"net"
	"os"

	"github.com/joho/godotenv"
	"github.com/orisegev/redis-lite/internal/storage"
)

func main() {

	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}
	port := os.Getenv("PORT")
	password := os.Getenv("AUTH_PASSWORD")

	if password == "" {
		password = "defaultpassword"
		log.Println("Warning: No AUTH_PASSWORD set, using default")
	}

	address := net.JoinHostPort("127.0.0.1", port)

	listener, err := net.Listen("tcp", address)

	if err != nil {
		log.Fatalf("Unable to start server: %v", err)
	}

	defer listener.Close()

	db := storage.NewEngine()

	defer listener.Close()

	fmt.Printf("Redis-lite server is running on %s\n", address)

	for {
		conn, err := listener.Accept()

		if err != nil {
			fmt.Printf("Error accepting connection: %v\n", err)
		}

		go handleConnection(conn, db, password)
	}
}
