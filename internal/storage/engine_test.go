package storage

import (
	"testing"
)

func TestSetGet(t *testing.T) {
	e := NewEngine()
	e.Set("k", "v")
	val, ok := e.Get("k")
	if !ok || val != "v" {
		t.Errorf("expected v, got %q (ok=%v)", val, ok)
	}
}

func TestGet_Missing(t *testing.T) {
	e := NewEngine()
	if _, ok := e.Get("nope"); ok {
		t.Error("expected miss on empty engine")
	}
}

func TestDelete(t *testing.T) {
	e := NewEngine()
	e.Set("k", "v")
	e.Delete("k")
	if _, ok := e.Get("k"); ok {
		t.Error("key should be deleted")
	}
}

func TestListKeys(t *testing.T) {
	e := NewEngine()
	e.Set("a", "1")
	e.Set("b", "2")
	if got := len(e.ListKeys()); got != 2 {
		t.Errorf("expected 2 keys, got %d", got)
	}
}

func TestListKeys_Empty(t *testing.T) {
	e := NewEngine()
	if got := len(e.ListKeys()); got != 0 {
		t.Errorf("expected 0 keys, got %d", got)
	}
}
