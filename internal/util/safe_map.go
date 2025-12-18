package util

import (
	"sync"
)

type SafeMap[V any] struct {
	mu   sync.RWMutex
	data map[string]V
}

func NewSafeMap[V any]() *SafeMap[V] {
	return &SafeMap[V]{
		data: make(map[string]V),
	}
}

func (sm *SafeMap[V]) Set(key string, value V) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.data[key] = value
}

func (sm *SafeMap[V]) Get(key string) (V, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	val, ok := sm.data[key]
	return val, ok
}
