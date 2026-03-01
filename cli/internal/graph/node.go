// SPDX-License-Identifier: AGPL-3.0-or-later

package graph

import (
	"fmt"
	"strings"
)

// Node types.
const (
	NodeTypeSeed    = "seed"
	NodeTypeAccount = "account"
)

// Node represents an entity in the intelligence graph.
type Node struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Label      string                 `json:"label"`
	Properties map[string]interface{} `json:"properties"`
}

// NewSeedNode creates a node representing an input seed.
func NewSeedNode(seedType, value string, isInitial bool, parentNodeID string, depth int) *Node {
	props := map[string]interface{}{
		"seed_type":  seedType,
		"value":      value,
		"is_initial": isInitial,
	}
	if parentNodeID != "" {
		props["discovered_from"] = parentNodeID
		props["pivot_depth"] = depth
	}

	return &Node{
		ID:         SeedNodeID(seedType, value),
		Type:       NodeTypeSeed,
		Label:      value,
		Properties: props,
	}
}

// NewAccountNode creates a node representing a discovered account on a platform.
func NewAccountNode(siteName, profileURL, category string, confidence float64, exists bool, signals interface{}, metadata map[string]string, httpStatus int, responseTimeMs int64, seedValue string) *Node {
	props := map[string]interface{}{
		"site_name":        siteName,
		"profile_url":      profileURL,
		"category":         category,
		"confidence":       confidence,
		"exists":           exists,
		"signals":          signals,
		"http_status":      httpStatus,
		"response_time_ms": responseTimeMs,
	}
	if metadata != nil && len(metadata) > 0 {
		props["metadata"] = metadata
	}

	return &Node{
		ID:         AccountNodeID(siteName, seedValue),
		Type:       NodeTypeAccount,
		Label:      fmt.Sprintf("%s - %s", siteName, seedValue),
		Properties: props,
	}
}

// SeedNodeID generates a deterministic node ID for a seed.
func SeedNodeID(seedType, value string) string {
	return fmt.Sprintf("seed:%s:%s", seedType, strings.ToLower(value))
}

// AccountNodeID generates a deterministic node ID for an account.
func AccountNodeID(siteName, seedValue string) string {
	return fmt.Sprintf("account:%s:%s", strings.ToLower(siteName), strings.ToLower(seedValue))
}
