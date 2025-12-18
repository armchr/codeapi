package util

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestExecutorPool_BasicExecution(t *testing.T) {
	var counter int64

	pool := NewExecutorPool(3, 10, func(item int) {
		atomic.AddInt64(&counter, int64(item))
	})

	time.Sleep(10 * time.Millisecond)

	for i := 1; i <= 5; i++ {
		pool.Submit(i)
	}

	time.Sleep(50 * time.Millisecond)

	pool.Close()

	expected := int64(1 + 2 + 3 + 4 + 5)
	if atomic.LoadInt64(&counter) != expected {
		t.Errorf("Expected counter to be %d, got %d", expected, atomic.LoadInt64(&counter))
	}
}

func TestExecutorPool_ConcurrentExecution(t *testing.T) {
	var counter int64
	var mu sync.Mutex
	var activeWorkers int64
	var maxActiveWorkers int64

	pool := NewExecutorPool(3, 10, func(item int) {
		current := atomic.AddInt64(&activeWorkers, 1)

		mu.Lock()
		if current > maxActiveWorkers {
			maxActiveWorkers = current
		}
		mu.Unlock()

		time.Sleep(20 * time.Millisecond)
		atomic.AddInt64(&counter, int64(item))
		atomic.AddInt64(&activeWorkers, -1)
	})

	time.Sleep(10 * time.Millisecond)

	for i := 1; i <= 10; i++ {
		pool.Submit(i)
	}

	pool.Close()

	expected := int64(55)
	if atomic.LoadInt64(&counter) != expected {
		t.Errorf("Expected counter to be %d, got %d", expected, atomic.LoadInt64(&counter))
	}

	if maxActiveWorkers > 3 {
		t.Errorf("Expected max concurrent workers to be 3, got %d", maxActiveWorkers)
	}
}

func TestExecutorPool_ProcessingOrder(t *testing.T) {
	var processed []int
	var mu sync.Mutex

	pool := NewExecutorPool(2, 5, func(item int) {
		mu.Lock()
		processed = append(processed, item)
		mu.Unlock()
		time.Sleep(10 * time.Millisecond)
	})

	time.Sleep(10 * time.Millisecond)

	for i := 1; i <= 5; i++ {
		pool.Submit(i)
	}

	pool.Close()

	mu.Lock()
	defer mu.Unlock()

	if len(processed) != 5 {
		t.Errorf("Expected 5 items processed, got %d", len(processed))
	}

	processedMap := make(map[int]bool)
	for _, item := range processed {
		processedMap[item] = true
	}

	for i := 1; i <= 5; i++ {
		if !processedMap[i] {
			t.Errorf("Item %d was not processed", i)
		}
	}
}
