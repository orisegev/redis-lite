package server

import (
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/orisegev/redis-lite/internal/config"
	"github.com/orisegev/redis-lite/internal/resp"
)

func newTestServer() *Server {
	return New(config.Config{Port: "0", Password: "testpass"})
}

func startPipe(t *testing.T, srv *Server) (net.Conn, *resp.Reader, func()) {
	t.Helper()
	client, serverSide := net.Pipe()
	go srv.handleConnection(serverSide)
	return client, resp.NewReader(client), func() {
		client.Close()
		srv.Close()
	}
}

// send encodes args as a RESP array, writes to w, reads and returns one Value.
func send(t *testing.T, r *resp.Reader, w net.Conn, args ...string) resp.Value {
	t.Helper()
	var sb strings.Builder
	fmt.Fprintf(&sb, "*%d\r\n", len(args))
	for _, arg := range args {
		fmt.Fprintf(&sb, "$%d\r\n%s\r\n", len(arg), arg)
	}
	if _, err := fmt.Fprint(w, sb.String()); err != nil {
		t.Fatalf("write command: %v", err)
	}
	v, err := r.ReadValue()
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	return v
}

func authenticate(t *testing.T, r *resp.Reader, w net.Conn) {
	t.Helper()
	v := send(t, r, w, "AUTH", "testpass")
	if v.Type != '+' || v.Str != "OK" {
		t.Fatalf("auth failed: type=%c str=%q", v.Type, v.Str)
	}
}

func TestAuth_Valid(t *testing.T) {
	srv := newTestServer()
	client, r, cleanup := startPipe(t, srv)
	defer cleanup()

	v := send(t, r, client, "AUTH", "testpass")
	if v.Type != '+' || v.Str != "OK" {
		t.Errorf("expected +OK, got type=%c str=%q", v.Type, v.Str)
	}
}

func TestAuth_Invalid(t *testing.T) {
	srv := newTestServer()
	client, r, cleanup := startPipe(t, srv)
	defer cleanup()

	v := send(t, r, client, "AUTH", "wrong")
	if v.Type != '-' {
		t.Errorf("expected error, got type=%c str=%q", v.Type, v.Str)
	}
}

func TestUnauthenticated_Blocked(t *testing.T) {
	srv := newTestServer()
	client, r, cleanup := startPipe(t, srv)
	defer cleanup()

	v := send(t, r, client, "SET", "foo", "bar")
	if v.Type != '-' {
		t.Errorf("unauthenticated SET should return error, got type=%c", v.Type)
	}
}

func TestSetGet(t *testing.T) {
	srv := newTestServer()
	client, r, cleanup := startPipe(t, srv)
	defer cleanup()
	authenticate(t, r, client)

	send(t, r, client, "SET", "name", "redis-lite")
	v := send(t, r, client, "GET", "name")
	if v.Type != '$' || v.Str != "redis-lite" {
		t.Errorf("expected bulk 'redis-lite', got type=%c str=%q", v.Type, v.Str)
	}
}

func TestGet_Missing(t *testing.T) {
	srv := newTestServer()
	client, r, cleanup := startPipe(t, srv)
	defer cleanup()
	authenticate(t, r, client)

	v := send(t, r, client, "GET", "missing")
	if !v.Null {
		t.Errorf("expected null bulk string, got type=%c null=%v", v.Type, v.Null)
	}
}

func TestDel(t *testing.T) {
	srv := newTestServer()
	client, r, cleanup := startPipe(t, srv)
	defer cleanup()
	authenticate(t, r, client)

	send(t, r, client, "SET", "k", "v")
	v := send(t, r, client, "DEL", "k")
	if v.Type != ':' || v.Num != 1 {
		t.Errorf("expected :1, got type=%c num=%d", v.Type, v.Num)
	}
	if got := send(t, r, client, "GET", "k"); !got.Null {
		t.Error("expected null after DEL")
	}
}

func TestDel_Missing(t *testing.T) {
	srv := newTestServer()
	client, r, cleanup := startPipe(t, srv)
	defer cleanup()
	authenticate(t, r, client)

	v := send(t, r, client, "DEL", "nope")
	if v.Type != ':' || v.Num != 0 {
		t.Errorf("expected :0, got type=%c num=%d", v.Type, v.Num)
	}
}

func TestKeys(t *testing.T) {
	srv := newTestServer()
	client, r, cleanup := startPipe(t, srv)
	defer cleanup()
	authenticate(t, r, client)

	send(t, r, client, "SET", "a", "1")
	send(t, r, client, "SET", "b", "2")
	v := send(t, r, client, "KEYS")
	if v.Type != '*' || len(v.Elems) != 2 {
		t.Errorf("expected array of 2, got type=%c elems=%d", v.Type, len(v.Elems))
	}
}

func TestKeys_Empty(t *testing.T) {
	srv := newTestServer()
	client, r, cleanup := startPipe(t, srv)
	defer cleanup()
	authenticate(t, r, client)

	v := send(t, r, client, "KEYS")
	if v.Type != '*' || len(v.Elems) != 0 {
		t.Errorf("expected empty array, got type=%c elems=%d", v.Type, len(v.Elems))
	}
}

func TestTTL_NoExpiry(t *testing.T) {
	srv := newTestServer()
	client, r, cleanup := startPipe(t, srv)
	defer cleanup()
	authenticate(t, r, client)

	send(t, r, client, "SET", "k", "v")
	v := send(t, r, client, "TTL", "k")
	if v.Type != ':' || v.Num != -1 {
		t.Errorf("expected :-1, got type=%c num=%d", v.Type, v.Num)
	}
}

func TestTTL_MissingKey(t *testing.T) {
	srv := newTestServer()
	client, r, cleanup := startPipe(t, srv)
	defer cleanup()
	authenticate(t, r, client)

	v := send(t, r, client, "TTL", "nope")
	if v.Type != ':' || v.Num != -2 {
		t.Errorf("expected :-2, got type=%c num=%d", v.Type, v.Num)
	}
}

func TestTTL_WithExpiry(t *testing.T) {
	srv := newTestServer()
	client, r, cleanup := startPipe(t, srv)
	defer cleanup()
	authenticate(t, r, client)

	send(t, r, client, "SET", "k", "v", "EX", "60")
	v := send(t, r, client, "TTL", "k")
	if v.Type != ':' || (v.Num != 59 && v.Num != 60) {
		t.Errorf("expected TTL ~60, got type=%c num=%d", v.Type, v.Num)
	}
}

func TestSetEx_ExpiresKey(t *testing.T) {
	srv := newTestServer()
	client, r, cleanup := startPipe(t, srv)
	defer cleanup()
	authenticate(t, r, client)

	send(t, r, client, "SET", "temp", "value", "EX", "1")
	if v := send(t, r, client, "GET", "temp"); v.Type != '$' || v.Str != "value" {
		t.Fatalf("expected value before expiry, got type=%c str=%q", v.Type, v.Str)
	}

	time.Sleep(1100 * time.Millisecond)

	if v := send(t, r, client, "GET", "temp"); !v.Null {
		t.Errorf("expected null after expiry, got type=%c str=%q", v.Type, v.Str)
	}
}

func TestSetEx_InvalidExpiry(t *testing.T) {
	srv := newTestServer()
	client, r, cleanup := startPipe(t, srv)
	defer cleanup()
	authenticate(t, r, client)

	if v := send(t, r, client, "SET", "k", "v", "EX", "notanumber"); v.Type != '-' {
		t.Errorf("expected error for invalid EX, got type=%c", v.Type)
	}
	if v := send(t, r, client, "SET", "k", "v", "EX", "0"); v.Type != '-' {
		t.Errorf("expected error for zero EX, got type=%c", v.Type)
	}
}

func TestUnknownCommand(t *testing.T) {
	srv := newTestServer()
	client, r, cleanup := startPipe(t, srv)
	defer cleanup()
	authenticate(t, r, client)

	v := send(t, r, client, "PING")
	if v.Type != '-' {
		t.Errorf("expected error for unknown command, got type=%c", v.Type)
	}
}
