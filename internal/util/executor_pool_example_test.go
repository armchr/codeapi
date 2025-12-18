package util

import (
	"fmt"
	"sync/atomic"
	"time"
)

func ExampleExecutorPool() {
	var processedCount int64

	pool := NewExecutorPool(3, 10, func(data string) {
		fmt.Printf("Processing: %s\n", data)
		atomic.AddInt64(&processedCount, 1)
		time.Sleep(100 * time.Millisecond)
	})

	tasks := []string{"task1", "task2", "task3", "task4", "task5"}

	for _, task := range tasks {
		pool.Submit(task)
	}

	pool.Close()

	fmt.Printf("Total processed: %d\n", atomic.LoadInt64(&processedCount))
}
