package workerpool

import (
	"context"
	"sync"
)

type Pool[T any] struct {
	jobs   chan T
	worker func(context.Context, T)
	wg     sync.WaitGroup
}

func New[T any](workers, queueSize int, worker func(context.Context, T)) *Pool[T] {
	p := &Pool[T]{
		jobs:   make(chan T, queueSize),
		worker: worker,
	}

	for i := 0; i < workers; i++ {
		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			for job := range p.jobs {
				p.worker(context.Background(), job)
			}
		}()
	}

	return p
}

func (p *Pool[T]) Submit(job T) bool {
	select {
	case p.jobs <- job:
		return true
	default:
		return false
	}
}

func (p *Pool[T]) Close() {
	close(p.jobs)
	p.wg.Wait()
}
