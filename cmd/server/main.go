package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
)

func main() {
	port := ":6379"

	listener, err := net.Listen("tcp", "127.0.0.1"+port)

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

	isAuthenticated := false

	const password = "mysecretpassword"

	conn.Write([]byte("Connected to Redis-lite please authenticate with AUTH <passwword>"))

	conn.Write([]byte("Connected to Redis-lite!\n"))

	scanner := bufio.NewScanner(conn)

	for scanner.Scan() {
		text := scanner.Text()
		parts := strings.Fields(text)

		if len(parts) == 0 {
			continue
		}

		command := strings.ToUpper(parts[0])

		if command == "exit" {
			conn.Write([]byte("Goodbye!\n"))
			break
		}

		if command == "AUTH" {
			if len(parts) > 1 && parts[1] == password {
				isAuthenticated = true
				conn.Write([]byte("OK\n"))
			} else {
				conn.Write([]byte("ERR invalid password\n"))
			}
			continue
		}

		if !isAuthenticated {
			conn.Write([]byte("ERR Use AUTH <password>\n"))
			continue
		}
		fmt.Printf("Received: %s\n", text)

		conn.Write([]byte("You said: " + text + "\n"))

	}
}
