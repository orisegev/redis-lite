package storage

import (
	"sync"
	"time"
)

const cleanupInterval = time.Second

type Engine struct {
	data   map[string]string
	expiry map[string]time.Time
	mu     sync.RWMutex
	done   chan struct{}
}

func NewEngine() *Engine {
	e := &Engine{
		data:   make(map[string]string),
		expiry: make(map[string]time.Time),
		done:   make(chan struct{}),
	}
	go e.cleanupLoop()
	return e
}

// Close stops the background cleanup goroutine.
func (e *Engine) Close() {
	close(e.done)
}

func (e *Engine) cleanupLoop() {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			e.deleteExpired()
		case <-e.done:
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
