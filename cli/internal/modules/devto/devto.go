// SPDX-License-Identifier: AGPL-3.0-or-later

package devto

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/KyleDerZweite/basalt/internal/graph"
	"github.com/KyleDerZweite/basalt/internal/httpclient"
	"github.com/KyleDerZweite/basalt/internal/modules"
)

const devtoBaseURL = "https://dev.to"

// Module extracts profile data from DEV.to.
type Module struct {
	baseURL string
}

// New creates a DEV.to module.
func New() *Module {
	return &Module{baseURL: devtoBaseURL}
}

func (m *Module) Name() string        { return "devto" }
func (m *Module) Description() string { return "Extract profile data from DEV.to" }

func (m *Module) CanHandle(nodeType string) bool {
	return nodeType == "username"
}

func (m *Module) Extract(ctx context.Context, node *graph.Node, client *httpclient.Client) ([]*graph.Node, []*graph.Edge, error) {
	username := node.Label
	apiURL := fmt.Sprintf("%s/api/users/by_username?url=%s", m.baseURL, username)

	resp, err := client.Do(ctx, apiURL, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("devto user request: %w", err)
	}
	if resp.StatusCode == 404 {
		return nil, nil, nil
	}
	if resp.StatusCode != 200 {
		return nil, nil, fmt.Errorf("devto returned %d", resp.StatusCode)
	}

	var profile struct {
		Username        string `json:"username"`
		Name            string `json:"name"`
		Summary         string `json:"summary"`
		JoinedAt        string `json:"joined_at"`
		ProfileImage    string `json:"profile_image"`
		TwitterUsername string `json:"twitter_username"`
		GithubUsername  string `json:"github_username"`
		WebsiteURL      string `json:"website_url"`
	}
	if err := json.Unmarshal([]byte(resp.Body), &profile); err != nil {
		return nil, nil, fmt.Errorf("parsing devto response: %w", err)
	}

	var nodes []*graph.Node
	var edges []*graph.Edge

	profileURL := fmt.Sprintf("https://dev.to/%s", profile.Username)
	account := graph.NewAccountNode("devto", username, profileURL, "devto")
	account.Confidence = 0.90
	if profile.Name != "" {
		account.Properties["name"] = profile.Name
	}
	if profile.Summary != "" {
		account.Properties["summary"] = profile.Summary
	}
	if profile.JoinedAt != "" {
		account.Properties["joined_at"] = profile.JoinedAt
	}
	if profile.ProfileImage != "" {
		account.Properties["profile_image"] = profile.ProfileImage
	}
	nodes = append(nodes, account)
	edges = append(edges, graph.NewEdge(0, node.ID, account.ID, graph.EdgeTypeHasAccount, "devto"))

	if profile.GithubUsername != "" {
		ghNode := graph.NewNode(graph.NodeTypeUsername, profile.GithubUsername, "devto")
		ghNode.Pivot = true
		ghNode.Confidence = 0.85
		nodes = append(nodes, ghNode)
		edges = append(edges, graph.NewEdge(0, account.ID, ghNode.ID, graph.EdgeTypeLinkedTo, "devto"))
	}

	if profile.TwitterUsername != "" {
		twNode := graph.NewNode(graph.NodeTypeUsername, profile.TwitterUsername, "devto")
		twNode.Pivot = true
		twNode.Confidence = 0.85
		nodes = append(nodes, twNode)
		edges = append(edges, graph.NewEdge(0, account.ID, twNode.ID, graph.EdgeTypeLinkedTo, "devto"))
	}

	if profile.WebsiteURL != "" {
		webNode := graph.NewNode(graph.NodeTypeWebsite, profile.WebsiteURL, "devto")
		webNode.Confidence = 0.80
		nodes = append(nodes, webNode)
		edges = append(edges, graph.NewEdge(0, account.ID, webNode.ID, graph.EdgeTypeLinkedTo, "devto"))
	}

	return nodes, edges, nil
}

func (m *Module) Verify(ctx context.Context, client *httpclient.Client) (modules.HealthStatus, string) {
	apiURL := fmt.Sprintf("%s/api/users/by_username?url=ben", m.baseURL)
	resp, err := client.Do(ctx, apiURL, nil)
	if err != nil {
		return modules.Offline, fmt.Sprintf("devto: %v", err)
	}
	if resp.StatusCode == 200 {
		var user struct {
			Username string `json:"username"`
		}
		if err := json.Unmarshal([]byte(resp.Body), &user); err == nil && user.Username == "ben" {
			return modules.Healthy, "devto: OK"
		}
		return modules.Degraded, "devto: unexpected response format"
	}
	return modules.Offline, fmt.Sprintf("devto: status %d", resp.StatusCode)
}
