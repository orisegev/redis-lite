package storage

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

const cleanupInterval = time.Second

// Options configures the storage engine.
type Options struct {
	SnapshotPath     string        // file path for snapshots; "" disables persistence
	SnapshotInterval time.Duration // how often to auto-save; 0 disables periodic saves
}

type Engine struct {
	data   map[string]string
	expiry map[string]time.Time
	mu     sync.RWMutex
	done   chan struct{}
	wg     sync.WaitGroup
	opts   Options
}

func NewEngine(opts Options) *Engine {
	e := &Engine{
		data:   make(map[string]string),
		expiry: make(map[string]time.Time),
		done:   make(chan struct{}),
		opts:   opts,
	}
	if opts.SnapshotPath != "" {
		if err := e.load(); err != nil && !os.IsNotExist(err) {
			log.Printf("snapshot load: %v", err)
		}
	}
	e.wg.Add(1)
	go e.cleanupLoop()
	return e
}

// Close stops background goroutines, waits for the final snapshot to finish, then returns.
func (e *Engine) Close() {
	close(e.done)
	e.wg.Wait()
}

func (e *Engine) cleanupLoop() {
	defer e.wg.Done()
	cleanupTicker := time.NewTicker(cleanupInterval)
	defer cleanupTicker.Stop()

	var snapshotC <-chan time.Time
	if e.opts.SnapshotPath != "" && e.opts.SnapshotInterval > 0 {
		t := time.NewTicker(e.opts.SnapshotInterval)
		defer t.Stop()
		snapshotC = t.C
	}

	for {
		select {
		case <-cleanupTicker.C:
			e.deleteExpired()
		case <-snapshotC:
			if err := e.save(); err != nil {
				log.Printf("snapshot save: %v", err)
			}
		case <-e.done:
			if e.opts.SnapshotPath != "" {
				if err := e.save(); err != nil {
					log.Printf("snapshot final save: %v", err)
				}
			}
			return
		}
	}
}

func (e *Engine) deleteExpired() {
	now := time.Now()
	e.mu.Lock()
	defer e.mu.Unlock()
	for k, exp := range e.expiry {
		if now.After(exp) {
			delete(e.data, k)
			delete(e.expiry, k)
		}
	}
}

// snapshotEntry is the JSON representation of a single key in the snapshot file.
type snapshotEntry struct {
	Key    string `json:"key"`
	Value  string `json:"value"`
	Expiry int64  `json:"expiry"` // Unix nanoseconds; 0 = no expiry
}

func (e *Engine) save() error {
	now := time.Now()

	e.mu.RLock()
	entries := make([]snapshotEntry, 0, len(e.data))
	for k, v := range e.data {
		var expiryNano int64
		if exp, ok := e.expiry[k]; ok {
			if now.After(exp) {
				continue // skip already-expired keys
			}
			expiryNano = exp.UnixNano()
		}
		entries = append(entries, snapshotEntry{Key: k, Value: v, Expiry: expiryNano})
	}
	e.mu.RUnlock()

	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	if err = os.WriteFile(e.opts.SnapshotPath, data, 0600); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	return nil
}

func (e *Engine) load() error {
	data, err := os.ReadFile(e.opts.SnapshotPath)
	if err != nil {
		return err
	}
	var entries []snapshotEntry
	if err = json.Unmarshal(data, &entries); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}

	now := time.Now()
	for _, entry := range entries {
		if entry.Expiry != 0 {
			exp := time.Unix(0, entry.Expiry)
			if now.After(exp) {
				continue // expired while the process was down — skip
			}
			e.Set(entry.Key, entry.Value, time.Until(exp))
		} else {
			e.Set(entry.Key, entry.Value, 0)
		}
	}
	log.Printf("snapshot loaded %d keys from %s", len(entries), e.opts.SnapshotPath)
	return nil
}

// Set stores a key-value pair. ttl > 0 sets an expiry; ttl == 0 means no expiry.
func (e *Engine) Set(key, value string, ttl time.Duration) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.data[key] = value
	if ttl > 0 {
		e.expiry[key] = time.Now().Add(ttl)
	} else {
		delete(e.expiry, key)
	}
}

func (e *Engine) Get(key string) (string, bool) {
	e.mu.RLock()
	val, exists := e.data[key]
	exp, hasExpiry := e.expiry[key]
	e.mu.RUnlock()

	if !exists {
		return "", false
	}
	if hasExpiry && time.Now().After(exp) {
		e.mu.Lock()
		delete(e.data, key)
		delete(e.expiry, key)
		e.mu.Unlock()
		return "", false
	}
	return val, true
}

func (e *Engine) Delete(key string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.data, key)
	delete(e.expiry, key)
}

// TTL returns remaining seconds for a key: -1 if no expiry, -2 if key does not exist.
func (e *Engine) TTL(key string) int {
	e.mu.RLock()
	defer e.mu.RUnlock()

	_, exists := e.data[key]
	if !exists {
		return -2
	}
	exp, hasExpiry := e.expiry[key]
	if !hasExpiry {
		return -1
	}
	remaining := time.Until(exp)
	if remaining <= 0 {
		return -2
	}
	return int(remaining.Seconds())
}

func (e *Engine) ListKeys() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	keys := make([]string, 0, len(e.data))
	for k := range e.data {
		keys = append(keys, k)
	}
	return keys
}
