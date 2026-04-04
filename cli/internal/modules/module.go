// SPDX-License-Identifier: AGPL-3.0-or-later

package modules

import (
	"context"

	"github.com/KyleDerZweite/basalt/internal/graph"
	"github.com/KyleDerZweite/basalt/internal/httpclient"
)

// HealthStatus represents the operational state of a module.
type HealthStatus int

const (
	// Healthy means verify passed, results scored normally.
	Healthy HealthStatus = iota
	// Degraded means verify got a response but data was unexpected. Confidence * 0.5.
	Degraded
	// Offline means verify failed entirely. Module skipped.
	Offline
)

// String returns the wire-safe name for the health status.
func (s HealthStatus) String() string {
	switch s {
	case Healthy:
		return "healthy"
	case Degraded:
		return "degraded"
	case Offline:
		return "offline"
	default:
		return "unknown"
	}
}

// Module is the interface every OSINT extraction module must implement.
type Module interface {
	// Name returns a human-readable identifier (e.g., "github").
	Name() string

	// Description returns what this module does.
	Description() string

	// CanHandle reports whether this module can process the given node type.
	CanHandle(nodeType string) bool

	// Extract processes a node and returns discovered nodes and edges.
	// Must respect context cancellation.
	Extract(ctx context.Context, node *graph.Node, client *httpclient.Client) ([]*graph.Node, []*graph.Edge, error)

	// Verify runs a lightweight self-test against a known entity.
	// Called once at startup. Returns status and a human-readable message.
	Verify(ctx context.Context, client *httpclient.Client) (HealthStatus, string)
}
