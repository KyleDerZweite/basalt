// SPDX-License-Identifier: AGPL-3.0-or-later

package trello

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kyle/basalt/internal/graph"
	"github.com/kyle/basalt/internal/httpclient"
	"github.com/kyle/basalt/internal/modules"
)

const trelloURL = "https://trello.com"

// Module extracts profile data from Trello.
type Module struct {
	baseURL string
}

// New creates a Trello module.
func New() *Module {
	return &Module{baseURL: trelloURL}
}

func (m *Module) Name() string                   { return "trello" }
func (m *Module) Description() string            { return "Extract profile data from Trello" }
func (m *Module) CanHandle(nodeType string) bool { return nodeType == "username" }

func (m *Module) Extract(ctx context.Context, node *graph.Node, client *httpclient.Client) ([]*graph.Node, []*graph.Edge, error) {
	username := node.Label
	url := fmt.Sprintf("%s/1/members/%s", m.baseURL, username)

	resp, err := client.Do(ctx, url, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("trello request: %w", err)
	}
	if resp.StatusCode != 200 {
		return nil, nil, nil
	}

	var profile struct {
		ID         string `json:"id"`
		Username   string `json:"username"`
		FullName   string `json:"fullName"`
		AvatarUrl  string `json:"avatarUrl"`
		Bio        string `json:"bio"`
		Url        string `json:"url"`
		MemberType string `json:"memberType"`
		Confirmed  bool   `json:"confirmed"`
	}
	if err := json.Unmarshal([]byte(resp.Body), &profile); err != nil {
		return nil, nil, fmt.Errorf("parsing trello response: %w", err)
	}

	profileURL := fmt.Sprintf("https://trello.com/%s", username)
	account := graph.NewAccountNode("trello", username, profileURL, "trello")
	account.Confidence = 0.90
	account.Properties["id"] = profile.ID
	account.Properties["confirmed"] = profile.Confirmed
	if profile.FullName != "" {
		account.Properties["full_name"] = profile.FullName
	}
	if profile.AvatarUrl != "" {
		account.Properties["avatar_url"] = profile.AvatarUrl
	}
	if profile.Bio != "" {
		account.Properties["bio"] = profile.Bio
	}
	if profile.Url != "" {
		account.Properties["url"] = profile.Url
	}
	if profile.MemberType != "" {
		account.Properties["member_type"] = profile.MemberType
	}

	nodes := []*graph.Node{account}
	edges := []*graph.Edge{
		graph.NewEdge(0, node.ID, account.ID, graph.EdgeTypeHasAccount, "trello"),
	}

	// Create FullName node if present and non-empty
	if profile.FullName != "" {
		fullNameNode := graph.NewNode(graph.NodeTypeFullName, profile.FullName, "trello")
		nodes = append(nodes, fullNameNode)
		edges = append(edges, graph.NewEdge(0, account.ID, fullNameNode.ID, graph.EdgeTypeLinkedTo, "trello"))
	}

	return nodes, edges, nil
}

func (m *Module) Verify(ctx context.Context, client *httpclient.Client) (modules.HealthStatus, string) {
	url := fmt.Sprintf("%s/1/members/trello", m.baseURL)
	resp, err := client.Do(ctx, url, nil)
	if err != nil {
		return modules.Offline, fmt.Sprintf("trello: %v", err)
	}
	if resp.StatusCode != 200 {
		return modules.Degraded, fmt.Sprintf("trello: unexpected status %d", resp.StatusCode)
	}

	var profile struct {
		Username string `json:"username"`
	}
	if err := json.Unmarshal([]byte(resp.Body), &profile); err == nil && profile.Username == "trello" {
		return modules.Healthy, "trello: OK"
	}
	return modules.Degraded, "trello: unexpected response format"
}
