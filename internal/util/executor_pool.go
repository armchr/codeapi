package util

import (
	"sync"
)

type ExecutorPool[T any] struct {
	maxConcurrent int
	workerFunc    func(T)
	buffer        chan T
	workerSem     chan struct{}
	wg            sync.WaitGroup
	closed        bool
	closeMutex    sync.Mutex
	done          chan struct{}
}

func NewExecutorPool[T any](maxConcurrent int, bufferSize int, workerFunc func(T)) *ExecutorPool[T] {
	pool := &ExecutorPool[T]{
		maxConcurrent: maxConcurrent,
		workerFunc:    workerFunc,
		buffer:        make(chan T, bufferSize),
		workerSem:     make(chan struct{}, maxConcurrent),
		done:          make(chan struct{}),
	}

	pool.start()
	return pool
}

func (p *ExecutorPool[T]) start() {
	go func() {
		defer close(p.done)
		for item := range p.buffer {
			p.workerSem <- struct{}{}
			p.wg.Add(1)

			go func(data T) {
				defer func() {
					<-p.workerSem
					p.wg.Done()
				}()
				p.workerFunc(data)
			}(item)
		}
		p.wg.Wait()
	}()
}

func (p *ExecutorPool[T]) Submit(item T) {
	p.closeMutex.Lock()
	defer p.closeMutex.Unlock()

	if p.closed {
		return
	}

	p.buffer <- item
}

func (p *ExecutorPool[T]) Close() {
	p.closeMutex.Lock()
	if p.closed {
		p.closeMutex.Unlock()
		return
	}

	p.closed = true
	close(p.buffer)
	p.closeMutex.Unlock()

	<-p.done
}
