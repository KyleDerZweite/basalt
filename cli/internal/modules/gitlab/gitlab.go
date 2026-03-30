// SPDX-License-Identifier: AGPL-3.0-or-later

package gitlab

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/kyle/basalt/internal/graph"
	"github.com/kyle/basalt/internal/httpclient"
	"github.com/kyle/basalt/internal/modules"
)

const gitlabAPI = "https://gitlab.com"

// Module extracts profile data from GitLab's REST API.
type Module struct {
	baseURL string
}

// New creates a GitLab module.
func New() *Module {
	return &Module{baseURL: gitlabAPI}
}

func (m *Module) Name() string                  { return "gitlab" }
func (m *Module) Description() string            { return "Extract profile data from GitLab" }
func (m *Module) CanHandle(nodeType string) bool { return nodeType == "username" }

func (m *Module) Extract(ctx context.Context, node *graph.Node, client *httpclient.Client) ([]*graph.Node, []*graph.Edge, error) {
	apiURL := fmt.Sprintf("%s/api/v4/users?username=%s", m.baseURL, url.QueryEscape(node.Label))

	resp, err := client.Do(ctx, apiURL, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("gitlab request: %w", err)
	}
	if resp.StatusCode != 200 {
		return nil, nil, fmt.Errorf("gitlab returned %d", resp.StatusCode)
	}

	var users []struct {
		Username   string `json:"username"`
		Name       string `json:"name"`
		WebURL     string `json:"web_url"`
		AvatarURL  string `json:"avatar_url"`
		Bio        string `json:"bio"`
		WebsiteURL string `json:"website_url"`
	}
	if err := json.Unmarshal([]byte(resp.Body), &users); err != nil {
		return nil, nil, fmt.Errorf("parsing gitlab response: %w", err)
	}
	if len(users) == 0 {
		return nil, nil, nil
	}

	user := users[0]
	var nodes []*graph.Node
	var edges []*graph.Edge

	account := graph.NewAccountNode("gitlab", user.Username, user.WebURL, "gitlab")
	account.Confidence = 0.90
	if user.Bio != "" {
		account.Properties["bio"] = user.Bio
	}
	nodes = append(nodes, account)
	edges = append(edges, graph.NewEdge(0, node.ID, account.ID, graph.EdgeTypeHasAccount, "gitlab"))

	if user.Name != "" {
		nameNode := graph.NewNode(graph.NodeTypeFullName, user.Name, "gitlab")
		nameNode.Confidence = 0.85
		nodes = append(nodes, nameNode)
		edges = append(edges, graph.NewEdge(0, account.ID, nameNode.ID, graph.EdgeTypeLinkedTo, "gitlab"))
	}

	if user.AvatarURL != "" {
		avatarNode := graph.NewNode(graph.NodeTypeAvatarURL, user.AvatarURL, "gitlab")
		avatarNode.Confidence = 0.90
		nodes = append(nodes, avatarNode)
		edges = append(edges, graph.NewEdge(0, account.ID, avatarNode.ID, graph.EdgeTypeLinkedTo, "gitlab"))
	}

	if user.WebsiteURL != "" {
		websiteNode := graph.NewNode(graph.NodeTypeWebsite, user.WebsiteURL, "gitlab")
		websiteNode.Pivot = true
		websiteNode.Confidence = 0.80
		nodes = append(nodes, websiteNode)
		edges = append(edges, graph.NewEdge(0, account.ID, websiteNode.ID, graph.EdgeTypeHasDomain, "gitlab"))
	}

	return nodes, edges, nil
}

func (m *Module) Verify(ctx context.Context, client *httpclient.Client) (modules.HealthStatus, string) {
	apiURL := fmt.Sprintf("%s/api/v4/users?username=root", m.baseURL)
	resp, err := client.Do(ctx, apiURL, nil)
	if err != nil {
		return modules.Offline, fmt.Sprintf("gitlab: %v", err)
	}
	if resp.StatusCode == 200 {
		return modules.Healthy, "gitlab: OK"
	}
	return modules.Degraded, fmt.Sprintf("gitlab: status %d", resp.StatusCode)
}
