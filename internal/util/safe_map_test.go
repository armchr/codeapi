package util

import (
	"fmt"
	"sync"
	"testing"
)

func TestNewSafeMap(t *testing.T) {
	m := NewSafeMap[int]()
	if m == nil {
		t.Error("NewSafeMap() returned nil")
	}
	if m.data == nil {
		t.Error("NewSafeMap() did not initialize internal map")
	}
}

func TestSafeMap_SetAndGet(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		value    int
		wantOk   bool
	}{
		{"set and get value", "key1", 42, true},
		{"get non-existent key", "missing", 0, false},
		{"set zero value", "zero", 0, true},
		{"set negative value", "negative", -100, true},
	}

	m := NewSafeMap[int]()

	// First, set the values that should exist
	m.Set("key1", 42)
	m.Set("zero", 0)
	m.Set("negative", -100)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := m.Get(tt.key)
			if ok != tt.wantOk {
				t.Errorf("Get(%q) ok = %v, want %v", tt.key, ok, tt.wantOk)
			}
			if ok && got != tt.value {
				t.Errorf("Get(%q) = %v, want %v", tt.key, got, tt.value)
			}
		})
	}
}

func TestSafeMap_Overwrite(t *testing.T) {
	m := NewSafeMap[string]()
	
	m.Set("key", "first")
	got, ok := m.Get("key")
	if !ok || got != "first" {
		t.Errorf("initial Set/Get failed: got %q, ok=%v", got, ok)
	}
	
	m.Set("key", "second")
	got, ok = m.Get("key")
	if !ok || got != "second" {
		t.Errorf("overwrite Set/Get failed: got %q, ok=%v, want 'second'", got, ok)
	}
}

func TestSafeMap_StringValues(t *testing.T) {
	m := NewSafeMap[string]()
	
	m.Set("greeting", "hello")
	m.Set("empty", "")
	m.Set("unicode", "héllo wörld 日本語")
	
	tests := []struct {
		key   string
		want  string
		wantOk bool
	}{
		{"greeting", "hello", true},
		{"empty", "", true},
		{"unicode", "héllo wörld 日本語", true},
		{"missing", "", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got, ok := m.Get(tt.key)
			if ok != tt.wantOk {
				t.Errorf("Get(%q) ok = %v, want %v", tt.key, ok, tt.wantOk)
			}
			if got != tt.want {
				t.Errorf("Get(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

func TestSafeMap_StructValues(t *testing.T) {
	type Person struct {
		Name string
		Age  int
	}
	
	m := NewSafeMap[Person]()
	
	alice := Person{Name: "Alice", Age: 30}
	m.Set("alice", alice)
	
	got, ok := m.Get("alice")
	if !ok {
		t.Fatal("Get('alice') returned ok=false")
	}
	if got != alice {
		t.Errorf("Get('alice') = %+v, want %+v", got, alice)
	}
	
	_, ok = m.Get("bob")
	if ok {
		t.Error("Get('bob') should return ok=false for missing key")
	}
}

func TestSafeMap_Concurrent(t *testing.T) {
	m := NewSafeMap[int]()
	var wg sync.WaitGroup
	numGoroutines := 100
	
	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			m.Set(fmt.Sprintf("key%d", n), n)
		}(i)
	}
	
	// Concurrent reads (while writes are happening)
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			m.Get(fmt.Sprintf("key%d", n))
		}(i)
	}
	
	wg.Wait()
	
	// Verify all values were written
	for i := 0; i < numGoroutines; i++ {
		key := fmt.Sprintf("key%d", i)
		val, ok := m.Get(key)
		if !ok {
			t.Errorf("key %s not found after concurrent writes", key)
		}
		if val != i {
			t.Errorf("Get(%s) = %d, want %d", key, val, i)
		}
	}
}

func TestSafeMap_ConcurrentReadWrite(t *testing.T) {
	m := NewSafeMap[int]()
	var wg sync.WaitGroup
	
	// Pre-populate some keys
	for i := 0; i < 10; i++ {
		m.Set(fmt.Sprintf("key%d", i), i)
	}
	
	// Concurrent mixed read/write operations
	for i := 0; i < 50; i++ {
		wg.Add(2)
		
		// Writer
		go func(n int) {
			defer wg.Done()
			m.Set(fmt.Sprintf("key%d", n%10), n)
		}(i)
		
		// Reader
		go func(n int) {
			defer wg.Done()
			m.Get(fmt.Sprintf("key%d", n%10))
		}(i)
	}
	
	wg.Wait()
	// If we get here without deadlock or panic, the test passes
}

func TestSafeMap_EmptyKey(t *testing.T) {
	m := NewSafeMap[string]()
	
	m.Set("", "empty key value")
	got, ok := m.Get("")
	if !ok {
		t.Error("Get('') should find empty key")
	}
	if got != "empty key value" {
		t.Errorf("Get('') = %q, want 'empty key value'", got)
	}
}
