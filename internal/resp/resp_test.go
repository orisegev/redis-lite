package resp

import (
	"bytes"
	"strings"
	"testing"
)

// roundTrip writes with the Writer then reads back with the Reader.
func roundTrip(t *testing.T, write func(*Writer) error) Value {
	t.Helper()
	var buf bytes.Buffer
	w := NewWriter(&buf)
	if err := write(w); err != nil {
		t.Fatalf("write: %v", err)
	}
	v, err := NewReader(&buf).ReadValue()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	return v
}

func TestSimpleString(t *testing.T) {
	v := roundTrip(t, func(w *Writer) error { return w.WriteSimpleString("OK") })
	if v.Type != '+' || v.Str != "OK" {
		t.Errorf("got type=%c str=%q", v.Type, v.Str)
	}
}

func TestError(t *testing.T) {
	v := roundTrip(t, func(w *Writer) error { return w.WriteError("unknown command") })
	if v.Type != '-' || !strings.Contains(v.Str, "unknown command") {
		t.Errorf("got type=%c str=%q", v.Type, v.Str)
	}
}

func TestInteger(t *testing.T) {
	v := roundTrip(t, func(w *Writer) error { return w.WriteInteger(42) })
	if v.Type != ':' || v.Num != 42 {
		t.Errorf("got type=%c num=%d", v.Type, v.Num)
	}
}

func TestInteger_Negative(t *testing.T) {
	v := roundTrip(t, func(w *Writer) error { return w.WriteInteger(-2) })
	if v.Type != ':' || v.Num != -2 {
		t.Errorf("got type=%c num=%d", v.Type, v.Num)
	}
}

func TestBulkString(t *testing.T) {
	v := roundTrip(t, func(w *Writer) error { return w.WriteBulkString("hello world") })
	if v.Type != '$' || v.Str != "hello world" {
		t.Errorf("got type=%c str=%q", v.Type, v.Str)
	}
}

func TestNull(t *testing.T) {
	v := roundTrip(t, func(w *Writer) error { return w.WriteNull() })
	if v.Type != '$' || !v.Null {
		t.Errorf("got type=%c null=%v", v.Type, v.Null)
	}
}

func TestArray(t *testing.T) {
	v := roundTrip(t, func(w *Writer) error { return w.WriteArray([]string{"foo", "bar"}) })
	if v.Type != '*' || len(v.Elems) != 2 {
		t.Fatalf("got type=%c elems=%d", v.Type, len(v.Elems))
	}
	if v.Elems[0].Str != "foo" || v.Elems[1].Str != "bar" {
		t.Errorf("got elems %q %q", v.Elems[0].Str, v.Elems[1].Str)
	}
}

func TestArray_Empty(t *testing.T) {
	v := roundTrip(t, func(w *Writer) error { return w.WriteArray([]string{}) })
	if v.Type != '*' || len(v.Elems) != 0 {
		t.Errorf("got type=%c elems=%d", v.Type, len(v.Elems))
	}
}

func TestReader_ArrayCommand(t *testing.T) {
	// Simulate redis-cli sending: SET foo bar
	raw := "*3\r\n$3\r\nSET\r\n$3\r\nfoo\r\n$3\r\nbar\r\n"
	v, err := NewReader(strings.NewReader(raw)).ReadValue()
	if err != nil {
		t.Fatal(err)
	}
	if v.Type != '*' || len(v.Elems) != 3 {
		t.Fatalf("expected 3-elem array, got type=%c elems=%d", v.Type, len(v.Elems))
	}
	if v.Elems[0].Str != "SET" || v.Elems[1].Str != "foo" || v.Elems[2].Str != "bar" {
		t.Errorf("unexpected elems: %v", v.Elems)
	}
}

func TestReader_InlineCommand(t *testing.T) {
	raw := "AUTH mypassword\r\n"
	v, err := NewReader(strings.NewReader(raw)).ReadValue()
	if err != nil {
		t.Fatal(err)
	}
	if v.Type != '*' || len(v.Elems) != 2 {
		t.Fatalf("expected 2-elem array, got type=%c elems=%d", v.Type, len(v.Elems))
	}
	if v.Elems[0].Str != "AUTH" || v.Elems[1].Str != "mypassword" {
		t.Errorf("unexpected elems: %v", v.Elems)
	}
}
