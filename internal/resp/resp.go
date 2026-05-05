package resp

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Value represents a single RESP value.
type Value struct {
	Type  byte    // '+' simple string  '-' error  ':' integer  '$' bulk string  '*' array
	Str   string  // simple string, error message, or bulk string content
	Num   int     // integer value
	Elems []Value // array elements
	Null  bool    // null bulk string or null array
}

// --- Reader ---

type Reader struct {
	br *bufio.Reader
}

func NewReader(r io.Reader) *Reader {
	return &Reader{br: bufio.NewReader(r)}
}

// ReadValue reads one complete RESP value from the stream.
// Inline commands (no type prefix) are returned as a '*' array for uniform handling.
func (r *Reader) ReadValue() (Value, error) {
	line, err := r.readLine()
	if err != nil {
		return Value{}, err
	}
	if len(line) == 0 {
		return Value{}, fmt.Errorf("empty line")
	}

	switch line[0] {
	case '+':
		return Value{Type: '+', Str: line[1:]}, nil
	case '-':
		return Value{Type: '-', Str: line[1:]}, nil
	case ':':
		n, err := strconv.Atoi(line[1:])
		if err != nil {
			return Value{}, fmt.Errorf("invalid integer %q", line[1:])
		}
		return Value{Type: ':', Num: n}, nil
	case '$':
		return r.readBulkString(line[1:])
	case '*':
		return r.readArray(line[1:])
	default:
		// Inline command: treat the whole line as space-separated args
		parts := strings.Fields(line)
		elems := make([]Value, len(parts))
		for i, p := range parts {
			elems[i] = Value{Type: '$', Str: p}
		}
		return Value{Type: '*', Elems: elems}, nil
	}
}

func (r *Reader) readBulkString(lenStr string) (Value, error) {
	length, err := strconv.Atoi(lenStr)
	if err != nil {
		return Value{}, fmt.Errorf("invalid bulk string length %q", lenStr)
	}
	if length == -1 {
		return Value{Type: '$', Null: true}, nil
	}
	buf := make([]byte, length+2) // +2 for \r\n
	if _, err = io.ReadFull(r.br, buf); err != nil {
		return Value{}, err
	}
	return Value{Type: '$', Str: string(buf[:length])}, nil
}

func (r *Reader) readArray(countStr string) (Value, error) {
	count, err := strconv.Atoi(countStr)
	if err != nil {
		return Value{}, fmt.Errorf("invalid array length %q", countStr)
	}
	if count == -1 {
		return Value{Type: '*', Null: true}, nil
	}
	elems := make([]Value, count)
	for i := range elems {
		elems[i], err = r.ReadValue()
		if err != nil {
			return Value{}, err
		}
	}
	return Value{Type: '*', Elems: elems}, nil
}

func (r *Reader) readLine() (string, error) {
	line, err := r.br.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

// --- Writer ---

type Writer struct {
	bw *bufio.Writer
}

func NewWriter(w io.Writer) *Writer {
	return &Writer{bw: bufio.NewWriter(w)}
}

func (w *Writer) WriteSimpleString(s string) error {
	_, err := fmt.Fprintf(w.bw, "+%s\r\n", s)
	if err != nil {
		return err
	}
	return w.bw.Flush()
}

func (w *Writer) WriteError(msg string) error {
	_, err := fmt.Fprintf(w.bw, "-ERR %s\r\n", msg)
	if err != nil {
		return err
	}
	return w.bw.Flush()
}

func (w *Writer) WriteInteger(n int) error {
	_, err := fmt.Fprintf(w.bw, ":%d\r\n", n)
	if err != nil {
		return err
	}
	return w.bw.Flush()
}

func (w *Writer) WriteBulkString(s string) error {
	_, err := fmt.Fprintf(w.bw, "$%d\r\n%s\r\n", len(s), s)
	if err != nil {
		return err
	}
	return w.bw.Flush()
}

func (w *Writer) WriteNull() error {
	_, err := fmt.Fprint(w.bw, "$-1\r\n")
	if err != nil {
		return err
	}
	return w.bw.Flush()
}

func (w *Writer) WriteArray(items []string) error {
	if _, err := fmt.Fprintf(w.bw, "*%d\r\n", len(items)); err != nil {
		return err
	}
	for _, item := range items {
		if _, err := fmt.Fprintf(w.bw, "$%d\r\n%s\r\n", len(item), item); err != nil {
			return err
		}
	}
	return w.bw.Flush()
}
