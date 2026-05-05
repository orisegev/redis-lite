package server

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"testing"

	"github.com/orisegev/redis-lite/internal/config"
)

func newTestServer() *Server {
	return New(config.Config{Port: "0", Password: "testpass"})
}

func startPipe(t *testing.T, srv *Server) (client net.Conn, cleanup func()) {
	t.Helper()
	client, serverSide := net.Pipe()
	go srv.handleConnection(serverSide)
	return client, func() { client.Close() }
}

func send(t *testing.T, r *bufio.Reader, w net.Conn, cmd string) string {
	t.Helper()
	fmt.Fprintln(w, cmd)
	resp, err := r.ReadString('\n')
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	return strings.TrimSpace(resp)
}

func authenticate(t *testing.T, r *bufio.Reader, w net.Conn) {
	t.Helper()
	r.ReadString('\n') // consume greeting
	if got := send(t, r, w, "AUTH testpass"); got != "OK" {
		t.Fatalf("auth failed: %q", got)
	}
}

func TestAuth_Valid(t *testing.T) {
	srv := newTestServer()
	client, cleanup := startPipe(t, srv)
	defer cleanup()
	r := bufio.NewReader(client)
	r.ReadString('\n')

	if got := send(t, r, client, "AUTH testpass"); got != "OK" {
		t.Errorf("expected OK, got %q", got)
	}
}

func TestAuth_Invalid(t *testing.T) {
	srv := newTestServer()
	client, cleanup := startPipe(t, srv)
	defer cleanup()
	r := bufio.NewReader(client)
	r.ReadString('\n')

	if got := send(t, r, client, "AUTH wrong"); !strings.HasPrefix(got, "ERR") {
		t.Errorf("expected ERR, got %q", got)
	}
}

func TestUnauthenticated_Blocked(t *testing.T) {
	srv := newTestServer()
	client, cleanup := startPipe(t, srv)
	defer cleanup()
	r := bufio.NewReader(client)
	r.ReadString('\n')

	if got := send(t, r, client, "SET foo bar"); !strings.HasPrefix(got, "ERR") {
		t.Errorf("unauthenticated SET should return ERR, got %q", got)
	}
}

func TestSetGet(t *testing.T) {
	srv := newTestServer()
	client, cleanup := startPipe(t, srv)
	defer cleanup()
	r := bufio.NewReader(client)
	authenticate(t, r, client)

	send(t, r, client, "SET name redis-lite")
	if got := send(t, r, client, "GET name"); got != "redis-lite" {
		t.Errorf("expected redis-lite, got %q", got)
	}
}

func TestGet_Missing(t *testing.T) {
	srv := newTestServer()
	client, cleanup := startPipe(t, srv)
	defer cleanup()
	r := bufio.NewReader(client)
	authenticate(t, r, client)

	if got := send(t, r, client, "GET missing"); got != "(nil)" {
		t.Errorf("expected (nil), got %q", got)
	}
}

func TestDel(t *testing.T) {
	srv := newTestServer()
	client, cleanup := startPipe(t, srv)
	defer cleanup()
	r := bufio.NewReader(client)
	authenticate(t, r, client)

	send(t, r, client, "SET k v")
	send(t, r, client, "DEL k")
	if got := send(t, r, client, "GET k"); got != "(nil)" {
		t.Errorf("expected (nil) after DEL, got %q", got)
	}
}

func TestUnknownCommand(t *testing.T) {
	srv := newTestServer()
	client, cleanup := startPipe(t, srv)
	defer cleanup()
	r := bufio.NewReader(client)
	authenticate(t, r, client)

	if got := send(t, r, client, "PING"); !strings.HasPrefix(got, "ERR") {
		t.Errorf("expected ERR for unknown command, got %q", got)
	}
}
