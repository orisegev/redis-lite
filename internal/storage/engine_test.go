package storage

import (
	"testing"
	"time"
)

func TestSetGet(t *testing.T) {
	e := NewEngine()
	e.Set("k", "v", 0)
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
	e.Set("k", "v", 0)
	e.Delete("k")
	if _, ok := e.Get("k"); ok {
		t.Error("key should be deleted")
	}
}

func TestListKeys(t *testing.T) {
	e := NewEngine()
	e.Set("a", "1", 0)
	e.Set("b", "2", 0)
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

func TestSetEx_Expiry(t *testing.T) {
	e := NewEngine()
	e.Set("k", "v", 50*time.Millisecond)

	if _, ok := e.Get("k"); !ok {
		t.Fatal("key should exist before expiry")
	}

	time.Sleep(60 * time.Millisecond)

	if _, ok := e.Get("k"); ok {
		t.Error("key should be expired")
	}
}

func TestTTL_NoExpiry(t *testing.T) {
	e := NewEngine()
	e.Set("k", "v", 0)
	if got := e.TTL("k"); got != -1 {
		t.Errorf("expected -1 for key with no expiry, got %d", got)
	}
}

func TestTTL_Missing(t *testing.T) {
	e := NewEngine()
	if got := e.TTL("nope"); got != -2 {
		t.Errorf("expected -2 for missing key, got %d", got)
	}
}

func TestTTL_WithExpiry(t *testing.T) {
	e := NewEngine()
	e.Set("k", "v", 10*time.Second)
	ttl := e.TTL("k")
	if ttl < 9 || ttl > 10 {
		t.Errorf("expected TTL ~10, got %d", ttl)
	}
}

func TestTTL_AfterExpiry(t *testing.T) {
	e := NewEngine()
	e.Set("k", "v", 50*time.Millisecond)
	time.Sleep(60 * time.Millisecond)
	if got := e.TTL("k"); got != -2 {
		t.Errorf("expected -2 after expiry, got %d", got)
	}
}

func TestSet_OverwriteClearsTTL(t *testing.T) {
	e := NewEngine()
	e.Set("k", "v", 50*time.Millisecond)
	e.Set("k", "v2", 0) // overwrite without TTL

	time.Sleep(60 * time.Millisecond)

	if _, ok := e.Get("k"); !ok {
		t.Error("key should persist after TTL was cleared by overwrite")
	}
}
