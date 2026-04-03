// SPDX-License-Identifier: AGPL-3.0-or-later

package stackexchange

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/KyleDerZweite/basalt/internal/graph"
	"github.com/KyleDerZweite/basalt/internal/httpclient"
	"github.com/KyleDerZweite/basalt/internal/modules"
)

const stackExchangeAPI = "https://api.stackexchange.com"

// Module extracts profile data from StackExchange (Stack Overflow).
type Module struct {
	baseURL string
}

// New creates a StackExchange module.
func New() *Module {
	return &Module{baseURL: stackExchangeAPI}
}

func (m *Module) Name() string { return "stackexchange" }
func (m *Module) Description() string {
	return "Extract profile data from Stack Overflow / StackExchange"
}
func (m *Module) CanHandle(nodeType string) bool { return nodeType == "username" }

func (m *Module) Extract(ctx context.Context, node *graph.Node, client *httpclient.Client) ([]*graph.Node, []*graph.Edge, error) {
	apiURL := fmt.Sprintf("%s/2.3/users?inname=%s&site=stackoverflow", m.baseURL, url.QueryEscape(node.Label))

	headers := map[string]string{
		"Accept-Encoding": "identity",
	}

	resp, err := client.Do(ctx, apiURL, headers)
	if err != nil {
		return nil, nil, fmt.Errorf("stackexchange request: %w", err)
	}
	if resp.StatusCode != 200 {
		return nil, nil, fmt.Errorf("stackexchange returned %d", resp.StatusCode)
	}

	var data struct {
		Items []struct {
			DisplayName string `json:"display_name"`
			Link        string `json:"link"`
			WebsiteURL  string `json:"website_url"`
			Location    string `json:"location"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(resp.Body), &data); err != nil {
		return nil, nil, fmt.Errorf("parsing stackexchange response: %w", err)
	}
	if len(data.Items) == 0 {
		return nil, nil, nil
	}

	// Find the first item with a case-insensitive display_name match.
	target := strings.ToLower(node.Label)
	idx := -1
	for i, item := range data.Items {
		if strings.ToLower(item.DisplayName) == target {
			idx = i
			break
		}
	}
	if idx == -1 {
		return nil, nil, nil
	}

	user := data.Items[idx]
	var nodes []*graph.Node
	var edges []*graph.Edge

	account := graph.NewAccountNode("stackexchange", user.DisplayName, user.Link, "stackexchange")
	account.Confidence = 0.80
	if user.Location != "" {
		account.Properties["location"] = user.Location
	}
	nodes = append(nodes, account)
	edges = append(edges, graph.NewEdge(0, node.ID, account.ID, graph.EdgeTypeHasAccount, "stackexchange"))

	if user.WebsiteURL != "" {
		websiteNode := graph.NewNode(graph.NodeTypeWebsite, user.WebsiteURL, "stackexchange")
		websiteNode.Pivot = true
		websiteNode.Confidence = 0.75
		nodes = append(nodes, websiteNode)
		edges = append(edges, graph.NewEdge(0, account.ID, websiteNode.ID, graph.EdgeTypeHasDomain, "stackexchange"))
	}

	return nodes, edges, nil
}

func (m *Module) Verify(ctx context.Context, client *httpclient.Client) (modules.HealthStatus, string) {
	apiURL := fmt.Sprintf("%s/2.3/users?inname=jonathon&site=stackoverflow", m.baseURL)

	headers := map[string]string{
		"Accept-Encoding": "identity",
	}

	resp, err := client.Do(ctx, apiURL, headers)
	if err != nil {
		return modules.Offline, fmt.Sprintf("stackexchange: %v", err)
	}
	if resp.StatusCode == 200 {
		return modules.Healthy, "stackexchange: OK"
	}
	return modules.Degraded, fmt.Sprintf("stackexchange: status %d", resp.StatusCode)
}
