// SPDX-License-Identifier: AGPL-3.0-or-later

package graph

import "fmt"

// Edge types.
const (
	EdgeTypeDiscoveredOn  = "discovered_on"
	EdgeTypeExtractedSeed = "extracted_seed"
)

// Edge represents a directed relationship between two nodes.
type Edge struct {
	ID         string                 `json:"id"`
	Source     string                 `json:"source"`
	Target     string                 `json:"target"`
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties"`
}

// NewDiscoveredEdge creates an edge from a seed node to an account node.
func NewDiscoveredEdge(id int, sourceNodeID, targetNodeID, engineName string, confidence float64) *Edge {
	return &Edge{
		ID:     fmt.Sprintf("e%d", id),
		Source: sourceNodeID,
		Target: targetNodeID,
		Type:   EdgeTypeDiscoveredOn,
		Properties: map[string]interface{}{
			"engine":     engineName,
			"confidence": confidence,
		},
	}
}

// NewExtractedSeedEdge creates an edge from an account node to a discovered seed node.
func NewExtractedSeedEdge(id int, sourceNodeID, targetNodeID, field, method string) *Edge {
	return &Edge{
		ID:     fmt.Sprintf("e%d", id),
		Source: sourceNodeID,
		Target: targetNodeID,
		Type:   EdgeTypeExtractedSeed,
		Properties: map[string]interface{}{
			"extraction_field":  field,
			"extraction_method": method,
		},
	}
}
