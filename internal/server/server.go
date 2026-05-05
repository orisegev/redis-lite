package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/orisegev/redis-lite/internal/config"
	"github.com/orisegev/redis-lite/internal/resp"
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

// Close stops the engine's background cleanup goroutine.
func (s *Server) Close() {
	s.storage.Close()
}

func (s *Server) Start(ctx context.Context) error {
	defer s.storage.Close()

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

	reader := resp.NewReader(conn)
	writer := resp.NewWriter(conn)
	authenticated := false

	for {
		val, err := reader.ReadValue()
		if err != nil {
			return
		}

		if val.Type != '*' || len(val.Elems) == 0 {
			writer.WriteError("invalid command format")
			continue
		}

		parts := make([]string, len(val.Elems))
		for i, e := range val.Elems {
			parts[i] = e.Str
		}
		cmd := strings.ToUpper(parts[0])

		switch cmd {
		case "EXIT":
			writer.WriteSimpleString("Goodbye!")
			return
		case "AUTH":
			if len(parts) > 1 && parts[1] == s.cfg.Password {
				authenticated = true
				writer.WriteSimpleString("OK")
			} else {
				writer.WriteError("invalid password")
			}
		default:
			if !authenticated {
				writer.WriteError("authenticate with AUTH <password>")
				continue
			}
			s.dispatch(writer, cmd, parts)
		}
	}
}

func (s *Server) dispatch(w *resp.Writer, cmd string, parts []string) {
	switch cmd {
	case "SET":
		if len(parts) < 3 {
			w.WriteError("usage: SET <key> <value> [EX <seconds>]")
			return
		}
		var ttl time.Duration
		if len(parts) >= 5 && strings.ToUpper(parts[3]) == "EX" {
			secs, err := strconv.Atoi(parts[4])
			if err != nil || secs <= 0 {
				w.WriteError("invalid expire time")
				return
			}
			ttl = time.Duration(secs) * time.Second
		}
		s.storage.Set(parts[1], parts[2], ttl)
		w.WriteSimpleString("OK")

	case "GET":
		if len(parts) < 2 {
			w.WriteError("usage: GET <key>")
			return
		}
		val, ok := s.storage.Get(parts[1])
		if !ok {
			w.WriteNull()
		} else {
			w.WriteBulkString(val)
		}

	case "DEL":
		if len(parts) < 2 {
			w.WriteError("usage: DEL <key>")
			return
		}
		_, ok := s.storage.Get(parts[1])
		if !ok {
			w.WriteInteger(0)
			return
		}
		s.storage.Delete(parts[1])
		w.WriteInteger(1)

	case "KEYS":
		keys := s.storage.ListKeys()
		w.WriteArray(keys)

	case "TTL":
		if len(parts) < 2 {
			w.WriteError("usage: TTL <key>")
			return
		}
		w.WriteInteger(s.storage.TTL(parts[1]))

	default:
		w.WriteError(fmt.Sprintf("unknown command '%s'", cmd))
	}
}
