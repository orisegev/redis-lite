package server

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"strings"

	"github.com/orisegev/redis-lite/internal/config"
	"github.com/orisegev/redis-lite/internal/storage"
)

type Server struct {
	cfg     config.Config
	storage *storage.Engine
}

func New(cfg config.Config) *Server {
	return &Server{
		cfg:     cfg,
		storage: storage.NewEngine(),
	}
}

func (s *Server) Start(ctx context.Context) error {
	addr := net.JoinHostPort("0.0.0.0", s.cfg.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", addr, err)
	}

	go func() {
		<-ctx.Done()
		listener.Close()
	}()

	log.Printf("redis-lite listening on %s", addr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			log.Printf("accept: %v", err)
			continue
		}
		go s.handleConnection(conn)
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	authenticated := false
	fmt.Fprint(conn, "Connected to redis-lite. Authenticate with AUTH <password>\n")

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		parts := strings.Fields(scanner.Text())
		if len(parts) == 0 {
			continue
		}
		cmd := strings.ToUpper(parts[0])

		switch cmd {
		case "EXIT":
			fmt.Fprint(conn, "Goodbye!\n")
			return
		case "AUTH":
			if len(parts) > 1 && parts[1] == s.cfg.Password {
				authenticated = true
				fmt.Fprint(conn, "OK\n")
			} else {
				fmt.Fprint(conn, "ERR invalid password\n")
			}
		default:
			if !authenticated {
				fmt.Fprint(conn, "ERR authenticate with AUTH <password>\n")
				continue
			}
			s.dispatch(conn, cmd, parts)
		}
	}
}

func (s *Server) dispatch(conn net.Conn, cmd string, parts []string) {
	switch cmd {
	case "SET":
		if len(parts) < 3 {
			fmt.Fprint(conn, "ERR usage: SET <key> <value>\n")
			return
		}
		s.storage.Set(parts[1], parts[2])
		fmt.Fprint(conn, "OK\n")

	case "GET":
		if len(parts) < 2 {
			fmt.Fprint(conn, "ERR usage: GET <key>\n")
			return
		}
		val, ok := s.storage.Get(parts[1])
		if !ok {
			fmt.Fprint(conn, "(nil)\n")
		} else {
			fmt.Fprint(conn, val+"\n")
		}

	case "DEL":
		if len(parts) < 2 {
			fmt.Fprint(conn, "ERR usage: DEL <key>\n")
			return
		}
		_, ok := s.storage.Get(parts[1])
		if !ok {
			fmt.Fprint(conn, "(integer) 0\n")
			return
		}
		s.storage.Delete(parts[1])
		fmt.Fprint(conn, "(integer) 1\n")

	case "KEYS":
		keys := s.storage.ListKeys()
		if len(keys) == 0 {
			fmt.Fprint(conn, "(empty list)\n")
		} else {
			for i, k := range keys {
				fmt.Fprintf(conn, "%d) %s\n", i+1, k)
			}
		}

	default:
		fmt.Fprintf(conn, "ERR unknown command '%s'\n", cmd)
	}
}
