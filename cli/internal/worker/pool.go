// SPDX-License-Identifier: AGPL-3.0-or-later

package worker

import (
	"context"
	"sync"
)

// Pool manages a bounded set of goroutines for concurrent work.
type Pool struct {
	concurrency int
}

// New creates a worker pool with the given concurrency limit.
func New(concurrency int) *Pool {
	if concurrency < 1 {
		concurrency = 1
	}
	return &Pool{concurrency: concurrency}
}

// Task is a unit of work to execute.
type Task func(ctx context.Context)

// Run executes all tasks with bounded concurrency.
// It blocks until all tasks complete or the context is cancelled.
func (p *Pool) Run(ctx context.Context, tasks []Task) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, p.concurrency)

	for _, task := range tasks {
		// Check for cancellation before starting new work.
		select {
		case <-ctx.Done():
			break
		default:
		}

		sem <- struct{}{} // acquire slot
		wg.Add(1)

		go func(t Task) {
			defer wg.Done()
			defer func() { <-sem }() // release slot
			t(ctx)
		}(task)
	}

	wg.Wait()
}
