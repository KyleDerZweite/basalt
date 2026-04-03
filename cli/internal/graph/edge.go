// SPDX-License-Identifier: AGPL-3.0-or-later

package graph

import "fmt"

// Edge types.
const (
	EdgeTypeHasAccount   = "has_account"
	EdgeTypeHasEmail     = "has_email"
	EdgeTypeHasDomain    = "has_domain"
	EdgeTypeHasUsername  = "has_username"
	EdgeTypeRegisteredTo = "registered_to"
	EdgeTypeResolvesTo   = "resolves_to"
	EdgeTypeLinkedTo     = "linked_to"
	EdgeTypeMentions     = "mentions"
)

// Edge represents a directed relationship between two nodes.
type Edge struct {
	ID           string                 `json:"id"`
	Source       string                 `json:"source"`
	Target       string                 `json:"target"`
	Type         string                 `json:"type"`
	SourceModule string                 `json:"source_module"`
	Properties   map[string]interface{} `json:"properties,omitempty"`
}

// NewEdge creates an edge between two nodes.
func NewEdge(id int, source, target, edgeType, sourceModule string) *Edge {
	return &Edge{
		ID:           fmt.Sprintf("e%d", id),
		Source:       source,
		Target:       target,
		Type:         edgeType,
		SourceModule: sourceModule,
	}
}
