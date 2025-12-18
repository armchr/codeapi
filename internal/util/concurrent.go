package util

import "sync"

func DoWorkList[T any, R any](list []T, work func(T) R) []R {
	results := make([]R, len(list))
	var wg sync.WaitGroup

	for i, item := range list {
		wg.Add(1)
		go func(index int, value T) {
			defer wg.Done()
			results[index] = work(value)
		}(i, item)
	}

	wg.Wait()
	return results
}
