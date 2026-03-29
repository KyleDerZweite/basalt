// SPDX-License-Identifier: AGPL-3.0-or-later

package gravatar

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kyle/basalt/internal/graph"
	"github.com/kyle/basalt/internal/httpclient"
	"github.com/kyle/basalt/internal/modules"
)

const gravatarAPI = "https://en.gravatar.com"

// Module extracts profile data from Gravatar via email MD5 hash.
type Module struct {
	baseURL string
}

// New creates a Gravatar module.
func New() *Module {
	return &Module{baseURL: gravatarAPI}
}

func (m *Module) Name() string                  { return "gravatar" }
func (m *Module) Description() string            { return "Extract profile data from Gravatar via email hash" }
func (m *Module) CanHandle(nodeType string) bool { return nodeType == "email" }

func (m *Module) Extract(ctx context.Context, node *graph.Node, client *httpclient.Client) ([]*graph.Node, []*graph.Edge, error) {
	email := strings.ToLower(strings.TrimSpace(node.Label))
	hash := fmt.Sprintf("%x", md5.Sum([]byte(email)))
	url := fmt.Sprintf("%s/%s.json", m.baseURL, hash)

	resp, err := client.Do(ctx, url, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("gravatar request: %w", err)
	}
	if resp.StatusCode == 404 {
		return nil, nil, nil
	}
	if resp.StatusCode != 200 {
		return nil, nil, fmt.Errorf("gravatar returned %d", resp.StatusCode)
	}

	var data struct {
		Entry []struct {
			DisplayName      string `json:"displayName"`
			PreferredUsername string `json:"preferredUsername"`
			ThumbnailURL     string `json:"thumbnailUrl"`
			URLs             []struct {
				Value string `json:"value"`
			} `json:"urls"`
		} `json:"entry"`
	}

	// The single-profile endpoint returns a flat object, not wrapped in "entry".
	// Try to parse as-is first (for test server), then try the real API format.
	var profile struct {
		DisplayName      string `json:"displayName"`
		PreferredUsername string `json:"preferredUsername"`
		ThumbnailURL     string `json:"thumbnailUrl"`
		URLs             []struct {
			Value string `json:"value"`
		} `json:"urls"`
	}

	if err := json.Unmarshal([]byte(resp.Body), &data); err == nil && len(data.Entry) > 0 {
		profile = data.Entry[0]
	} else if err := json.Unmarshal([]byte(resp.Body), &profile); err != nil {
		return nil, nil, fmt.Errorf("parsing gravatar response: %w", err)
	}

	var nodes []*graph.Node
	var edges []*graph.Edge

	// Account node.
	account := graph.NewAccountNode("gravatar", email, fmt.Sprintf("https://gravatar.com/%s", hash), "gravatar")
	account.Confidence = 0.95
	nodes = append(nodes, account)

	edges = append(edges, graph.NewEdge(0, node.ID, account.ID, graph.EdgeTypeHasAccount, "gravatar"))

	// Display name.
	if profile.DisplayName != "" {
		nameNode := graph.NewNode(graph.NodeTypeFullName, profile.DisplayName, "gravatar")
		nameNode.Confidence = 0.85
		nodes = append(nodes, nameNode)
		edges = append(edges, graph.NewEdge(0, account.ID, nameNode.ID, graph.EdgeTypeLinkedTo, "gravatar"))
	}

	// Username.
	if profile.PreferredUsername != "" {
		usernameNode := graph.NewNode(graph.NodeTypeUsername, profile.PreferredUsername, "gravatar")
		usernameNode.Pivot = true
		usernameNode.Confidence = 0.85
		nodes = append(nodes, usernameNode)
		edges = append(edges, graph.NewEdge(0, account.ID, usernameNode.ID, graph.EdgeTypeHasUsername, "gravatar"))
	}

	// Avatar.
	if profile.ThumbnailURL != "" {
		avatarNode := graph.NewNode(graph.NodeTypeAvatarURL, profile.ThumbnailURL, "gravatar")
		avatarNode.Confidence = 0.95
		nodes = append(nodes, avatarNode)
		edges = append(edges, graph.NewEdge(0, account.ID, avatarNode.ID, graph.EdgeTypeLinkedTo, "gravatar"))
	}

	// Linked URLs (websites/domains).
	for _, u := range profile.URLs {
		if u.Value != "" {
			websiteNode := graph.NewNode(graph.NodeTypeWebsite, u.Value, "gravatar")
			websiteNode.Pivot = true
			websiteNode.Confidence = 0.80
			nodes = append(nodes, websiteNode)
			edges = append(edges, graph.NewEdge(0, account.ID, websiteNode.ID, graph.EdgeTypeHasDomain, "gravatar"))
		}
	}

	return nodes, edges, nil
}

func (m *Module) Verify(ctx context.Context, client *httpclient.Client) (modules.HealthStatus, string) {
	// Check a known email hash (test@example.com).
	hash := fmt.Sprintf("%x", md5.Sum([]byte("test@example.com")))
	url := fmt.Sprintf("%s/%s.json", m.baseURL, hash)

	resp, err := client.Do(ctx, url, nil)
	if err != nil {
		return modules.Offline, fmt.Sprintf("gravatar: %v", err)
	}
	// 404 is acceptable (means API is up, hash not found).
	if resp.StatusCode == 200 || resp.StatusCode == 404 {
		return modules.Healthy, "gravatar: OK"
	}
	return modules.Degraded, fmt.Sprintf("gravatar: unexpected status %d", resp.StatusCode)
}
