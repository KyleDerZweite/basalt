// SPDX-License-Identifier: AGPL-3.0-or-later

package modules

import (
	"context"

	"github.com/kyle/basalt/internal/graph"
)

// Module is the interface every OSINT extraction module must implement.
type Module interface {
	// Name returns a human-readable name (e.g., "GitHub Profile Scraper").
	Name() string

	// CanHandle checks if this module can process the given node type.
	CanHandle(nodeType string) bool

	// Extract processes a node and returns new nodes and edges.
	// Must respect context cancellation.
	Extract(ctx context.Context, node *graph.Node) ([]*graph.Node, []*graph.Edge, error)
}
