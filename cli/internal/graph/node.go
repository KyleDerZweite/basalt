// SPDX-License-Identifier: AGPL-3.0-or-later

package graph

import (
	"fmt"
	"strings"
)

// Node types.
const (
	NodeTypeSeed         = "seed"
	NodeTypeAccount      = "account"
	NodeTypeUsername     = "username"
	NodeTypeEmail        = "email"
	NodeTypeDomain       = "domain"
	NodeTypeIP           = "ip"
	NodeTypeOrganization = "organization"
	NodeTypePhone        = "phone"
	NodeTypeFullName     = "full_name"
	NodeTypeAvatarURL    = "avatar_url"
	NodeTypeWebsite      = "website"
)

// Node represents an entity in the intelligence graph.
type Node struct {
	ID           string                 `json:"id"`
	Type         string                 `json:"type"`
	Label        string                 `json:"label"`
	SourceModule string                 `json:"source_module"`
	Pivot        bool                   `json:"pivot"`
	Wave         int                    `json:"wave"`
	Confidence   float64                `json:"confidence"`
	Properties   map[string]interface{} `json:"properties,omitempty"`
}

// NewNode creates a node with the given type, value, and source module.
func NewNode(nodeType, value, sourceModule string) *Node {
	return &Node{
		ID:           fmt.Sprintf("%s:%s", nodeType, strings.ToLower(value)),
		Type:         nodeType,
		Label:        value,
		SourceModule: sourceModule,
		Properties:   make(map[string]interface{}),
	}
}

// NewSeedNode creates a node representing an input seed.
func NewSeedNode(seedType, value string) *Node {
	return &Node{
		ID:           SeedNodeID(seedType, value),
		Type:         NodeTypeSeed,
		Label:        value,
		SourceModule: "seed",
		Pivot:        true,
		Wave:         0,
		Properties: map[string]interface{}{
			"seed_type": seedType,
		},
	}
}

// NewAccountNode creates a node representing a discovered account on a platform.
func NewAccountNode(platform, seedValue, profileURL, sourceModule string) *Node {
	return &Node{
		ID:           AccountNodeID(platform, seedValue),
		Type:         NodeTypeAccount,
		Label:        fmt.Sprintf("%s - %s", platform, seedValue),
		SourceModule: sourceModule,
		Properties: map[string]interface{}{
			"site_name":   platform,
			"profile_url": profileURL,
		},
	}
}

// SeedNodeID generates a deterministic node ID for a seed.
func SeedNodeID(seedType, value string) string {
	return fmt.Sprintf("seed:%s:%s", seedType, strings.ToLower(value))
}

// AccountNodeID generates a deterministic node ID for an account.
func AccountNodeID(platform, seedValue string) string {
	return fmt.Sprintf("account:%s:%s", strings.ToLower(platform), strings.ToLower(seedValue))
}

// Seed represents a seed entity for scanning.
type Seed struct {
	Type  string
	Value string
}

// ParseSeed parses a seed string in format "type:value".
func ParseSeed(s string) (Seed, error) {
	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return Seed{}, fmt.Errorf("invalid seed format %q (expected type:value)", s)
	}
	return Seed{Type: parts[0], Value: parts[1]}, nil
}
