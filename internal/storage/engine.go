package storage

import (
	"sync"
	"time"
)

type Engine struct {
	data   map[string]string
	expiry map[string]time.Time
	mu     sync.RWMutex
}

func NewEngine() *Engine {
	return &Engine{
		data:   make(map[string]string),
		expiry: make(map[string]time.Time),
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
