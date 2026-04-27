package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
)

func main() {
	port := ":6379"

	listener, err := net.Listen("tcp", port)

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

		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	fmt.Printf("Client connected: %s\n", conn.RemoteAddr().String())

	conn.Write([]byte("Connected to Redis-lite!\n"))

	scanner := bufio.NewScanner(conn)

	for scanner.Scan() {
		text := scanner.Text()

		fmt.Printf("Received: %s\n", text)

		conn.Write([]byte("You said: " + text + "\n"))

		if text == "exit" {
			conn.Write([]byte("Goodbye!\n"))
			break
		}

	}
}
