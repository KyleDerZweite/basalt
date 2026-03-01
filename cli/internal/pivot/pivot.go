// SPDX-License-Identifier: AGPL-3.0-or-later
//
// PivotController manages recursive seed discovery during scans.
// It deduplicates seeds, respects maximum pivot depth, and queues
// newly discovered seeds for subsequent engine runs.

package pivot

import (
	"sync"

	"github.com/kyle/basalt/internal/engine"
)

// Controller tracks discovered seeds and prevents cycles.
type Controller struct {
	mu       sync.Mutex
	seen     map[string]struct{} // "type:value" → already queued
	pending  []engine.Seed       // seeds waiting to be processed
	maxDepth int
}

// NewController creates a pivot controller with the given max depth.
// A maxDepth of 0 disables pivoting (only the initial seed is processed).
func NewController(maxDepth int) *Controller {
	return &Controller{
		seen:     make(map[string]struct{}),
		maxDepth: maxDepth,
	}
}

// MarkSeen records a seed as already processed. Returns true if this is
// the first time the seed has been seen.
func (c *Controller) MarkSeen(seed engine.Seed) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := string(seed.Type) + ":" + seed.Value
	if _, exists := c.seen[key]; exists {
		return false
	}
	c.seen[key] = struct{}{}
	return true
}

// Enqueue adds discovered seeds to the pending queue.
// Seeds that have already been seen or exceed max depth are skipped.
// Returns the number of seeds actually enqueued.
func (c *Controller) Enqueue(seeds []engine.Seed) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	count := 0
	for _, seed := range seeds {
		if seed.Depth > c.maxDepth {
			continue
		}

		key := string(seed.Type) + ":" + seed.Value
		if _, exists := c.seen[key]; exists {
			continue
		}

		c.seen[key] = struct{}{}
		c.pending = append(c.pending, seed)
		count++
	}
	return count
}

// Drain returns all pending seeds and clears the queue.
func (c *Controller) Drain() []engine.Seed {
	c.mu.Lock()
	defer c.mu.Unlock()

	seeds := c.pending
	c.pending = nil
	return seeds
}

// Enabled returns whether pivoting is active.
func (c *Controller) Enabled() bool {
	return c.maxDepth > 0
}
