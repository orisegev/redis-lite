package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func newTestEngine() *Engine {
	return NewEngine(Options{})
}

func TestSetGet(t *testing.T) {
	e := newTestEngine()
	defer e.Close()
	e.Set("k", "v", 0)
	val, ok := e.Get("k")
	if !ok || val != "v" {
		t.Errorf("expected v, got %q (ok=%v)", val, ok)
	}
}

func TestGet_Missing(t *testing.T) {
	e := newTestEngine()
	defer e.Close()
	if _, ok := e.Get("nope"); ok {
		t.Error("expected miss on empty engine")
	}
}

func TestDelete(t *testing.T) {
	e := newTestEngine()
	defer e.Close()
	e.Set("k", "v", 0)
	e.Delete("k")
	if _, ok := e.Get("k"); ok {
		t.Error("key should be deleted")
	}
}

func TestListKeys(t *testing.T) {
	e := newTestEngine()
	defer e.Close()
	e.Set("a", "1", 0)
	e.Set("b", "2", 0)
	if got := len(e.ListKeys()); got != 2 {
		t.Errorf("expected 2 keys, got %d", got)
	}
}

func TestListKeys_Empty(t *testing.T) {
	e := newTestEngine()
	defer e.Close()
	if got := len(e.ListKeys()); got != 0 {
		t.Errorf("expected 0 keys, got %d", got)
	}
}

func TestSetEx_Expiry(t *testing.T) {
	e := newTestEngine()
	defer e.Close()
	e.Set("k", "v", 50*time.Millisecond)

	if _, ok := e.Get("k"); !ok {
		t.Fatal("key should exist before expiry")
	}
	time.Sleep(60 * time.Millisecond)
	if _, ok := e.Get("k"); ok {
		t.Error("key should be expired")
	}
}

func TestBackgroundCleanup(t *testing.T) {
	e := &Engine{
		data:   make(map[string]string),
		expiry: make(map[string]time.Time),
		done:   make(chan struct{}),
	}
	go func() {
		ticker := time.NewTicker(20 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				e.deleteExpired()
			case <-e.done:
				return
			}
		}
	}()
	defer e.Close()

	e.Set("k", "v", 50*time.Millisecond)
	time.Sleep(120 * time.Millisecond)

	e.mu.RLock()
	_, inData := e.data["k"]
	_, inExpiry := e.expiry["k"]
	e.mu.RUnlock()

	if inData || inExpiry {
		t.Error("background cleanup should have removed the expired key from memory")
	}
}

func TestTTL_NoExpiry(t *testing.T) {
	e := newTestEngine()
	defer e.Close()
	e.Set("k", "v", 0)
	if got := e.TTL("k"); got != -1 {
		t.Errorf("expected -1 for key with no expiry, got %d", got)
	}
}

func TestTTL_Missing(t *testing.T) {
	e := newTestEngine()
	defer e.Close()
	if got := e.TTL("nope"); got != -2 {
		t.Errorf("expected -2 for missing key, got %d", got)
	}
}

func TestTTL_WithExpiry(t *testing.T) {
	e := newTestEngine()
	defer e.Close()
	e.Set("k", "v", 10*time.Second)
	ttl := e.TTL("k")
	if ttl < 9 || ttl > 10 {
		t.Errorf("expected TTL ~10, got %d", ttl)
	}
}

func TestTTL_AfterExpiry(t *testing.T) {
	e := newTestEngine()
	defer e.Close()
	e.Set("k", "v", 50*time.Millisecond)
	time.Sleep(60 * time.Millisecond)
	if got := e.TTL("k"); got != -2 {
		t.Errorf("expected -2 after expiry, got %d", got)
	}
}

func TestSet_OverwriteClearsTTL(t *testing.T) {
	e := newTestEngine()
	defer e.Close()
	e.Set("k", "v", 50*time.Millisecond)
	e.Set("k", "v2", 0)
	time.Sleep(60 * time.Millisecond)
	if _, ok := e.Get("k"); !ok {
		t.Error("key should persist after TTL was cleared by overwrite")
	}
}

// --- Persistence tests ---

func tempSnapshotPath(t *testing.T) string {
	t.Helper()
	f, err := os.CreateTemp("", "redis-lite-test-*.json")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	os.Remove(f.Name()) // start with no file; NewEngine will create it on save
	t.Cleanup(func() { os.Remove(f.Name()) })
	return f.Name()
}

func TestPersistence_SaveLoad(t *testing.T) {
	path := tempSnapshotPath(t)

	e1 := NewEngine(Options{SnapshotPath: path})
	e1.Set("name", "ori", 0)
	e1.Set("city", "tlv", 0)
	e1.Close() // triggers final save

	e2 := NewEngine(Options{SnapshotPath: path})
	defer e2.Close()

	if v, ok := e2.Get("name"); !ok || v != "ori" {
		t.Errorf("expected name=ori after reload, got %q ok=%v", v, ok)
	}
	if v, ok := e2.Get("city"); !ok || v != "tlv" {
		t.Errorf("expected city=tlv after reload, got %q ok=%v", v, ok)
	}
}

func TestPersistence_SkipsExpiredOnLoad(t *testing.T) {
	path := tempSnapshotPath(t)

	e1 := NewEngine(Options{SnapshotPath: path})
	e1.Set("gone", "value", 50*time.Millisecond)
	time.Sleep(60 * time.Millisecond) // let it expire
	e1.Close()                        // snapshot written — expired key filtered out

	e2 := NewEngine(Options{SnapshotPath: path})
	defer e2.Close()

	if _, ok := e2.Get("gone"); ok {
		t.Error("expired key should not be loaded from snapshot")
	}
}

func TestPersistence_PreservesTTL(t *testing.T) {
	path := tempSnapshotPath(t)

	e1 := NewEngine(Options{SnapshotPath: path})
	e1.Set("session", "abc", 10*time.Second)
	e1.Close()

	e2 := NewEngine(Options{SnapshotPath: path})
	defer e2.Close()

	ttl := e2.TTL("session")
	if ttl < 8 || ttl > 10 {
		t.Errorf("expected TTL ~10 after reload, got %d", ttl)
	}
}

func TestPersistence_NoFileIsOK(t *testing.T) {
	path := filepath.Join(os.TempDir(), "redis-lite-nonexistent-test.json")
	os.Remove(path) // ensure it doesn't exist before the test
	e := NewEngine(Options{SnapshotPath: path})
	defer e.Close()
	defer os.Remove(path)
	// Should not panic or error — just start empty
	if keys := e.ListKeys(); len(keys) != 0 {
		t.Errorf("expected empty engine, got %d keys", len(keys))
	}
}

func TestPersistence_SaveOnClose(t *testing.T) {
	path := tempSnapshotPath(t)

	e := NewEngine(Options{SnapshotPath: path})
	e.Set("k", "v", 0)
	e.Close()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("snapshot file should exist after Close()")
	}
}
