package main

import (
	"bufio"
	"fmt"
	"net"
	"strings"

	"github.com/orisegev/redis-lite/internal/storage"
)

func handleConnection(conn net.Conn, db *storage.Engine, password string) {
	defer conn.Close()

	isAuthenticated := false

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

		if command == "EXIT" {
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

		switch command {
		case "SET":
			if len(parts) < 3 {
				conn.Write([]byte("ERR Usage: SET <key> <value>\\n"))
				continue
			}

			db.Set(parts[1], parts[2])
			conn.Write([]byte("OK\n"))

		case "GET":
			if len(parts) < 2 {
				conn.Write([]byte("ERR usage: GET <key>\n"))
				continue
			}
			val, exists := db.Get(parts[1])

			if !exists {
				conn.Write([]byte("(nill)\n"))
			} else {
				conn.Write([]byte(val + "\n"))
			}
		case "DEL":
			if len(parts) < 2 {
				conn.Write([]byte("ERR Usage: DEL <key>\n"))
				continue
			}

			_, exists := db.Get(parts[1])

			if !exists {
				conn.Write([]byte("Key Not exists\n"))
				continue
			}

			db.Delete(parts[1])
			conn.Write([]byte("OK\n"))

		case "KEYS":
			keys := db.ListKeys()

			if len(keys) == 0 {
				conn.Write([]byte("(empty list or set)\n"))
			} else {
				for i, key := range keys {
					fmt.Fprintf(conn, "%d) %s\n", i+1, key)
				}
			}
		default:
			conn.Write([]byte("ERR unknown command '" + command + "'\n"))
		}

	}
}
