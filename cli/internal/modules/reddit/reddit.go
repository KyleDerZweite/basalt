// SPDX-License-Identifier: AGPL-3.0-or-later

package reddit

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kyle/basalt/internal/graph"
	"github.com/kyle/basalt/internal/httpclient"
	"github.com/kyle/basalt/internal/modules"
)

const redditURL = "https://www.reddit.com"

// Module extracts profile data from Reddit's public JSON API.
type Module struct {
	baseURL string
}

// New creates a Reddit module.
func New() *Module {
	return &Module{baseURL: redditURL}
}

func (m *Module) Name() string                  { return "reddit" }
func (m *Module) Description() string           { return "Extract profile data from Reddit user pages" }
func (m *Module) CanHandle(nodeType string) bool { return nodeType == "username" }

func (m *Module) Extract(ctx context.Context, node *graph.Node, client *httpclient.Client) ([]*graph.Node, []*graph.Edge, error) {
	username := node.Label
	url := fmt.Sprintf("%s/user/%s/about.json", m.baseURL, username)

	resp, err := client.Do(ctx, url, map[string]string{
		"User-Agent": "basalt/2.0",
	})
	if err != nil {
		return nil, nil, fmt.Errorf("reddit request: %w", err)
	}
	if resp.StatusCode == 404 {
		return nil, nil, nil
	}
	if resp.StatusCode != 200 {
		return nil, nil, fmt.Errorf("reddit returned %d", resp.StatusCode)
	}

	var payload struct {
		Data struct {
			Name       string  `json:"name"`
			CreatedUTC float64 `json:"created_utc"`
			Subreddit  struct {
				PublicDescription string `json:"public_description"`
			} `json:"subreddit"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(resp.Body), &payload); err != nil {
		return nil, nil, fmt.Errorf("parsing reddit response: %w", err)
	}

	var nodes []*graph.Node
	var edges []*graph.Edge

	profileURL := fmt.Sprintf("%s/user/%s", m.baseURL, username)
	account := graph.NewAccountNode("reddit", username, profileURL, "reddit")
	account.Confidence = 0.90
	if payload.Data.CreatedUTC > 0 {
		account.Properties["created_utc"] = payload.Data.CreatedUTC
	}
	if payload.Data.Subreddit.PublicDescription != "" {
		account.Properties["description"] = payload.Data.Subreddit.PublicDescription
	}
	nodes = append(nodes, account)
	edges = append(edges, graph.NewEdge(0, node.ID, account.ID, graph.EdgeTypeHasAccount, "reddit"))

	return nodes, edges, nil
}

func (m *Module) Verify(ctx context.Context, client *httpclient.Client) (modules.HealthStatus, string) {
	url := fmt.Sprintf("%s/user/spez/about.json", m.baseURL)
	resp, err := client.Do(ctx, url, map[string]string{
		"User-Agent": "basalt/2.0",
	})
	if err != nil {
		return modules.Offline, fmt.Sprintf("reddit: %v", err)
	}
	if resp.StatusCode != 200 {
		return modules.Degraded, fmt.Sprintf("reddit: unexpected status %d", resp.StatusCode)
	}

	var payload struct {
		Data struct {
			Name string `json:"name"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(resp.Body), &payload); err == nil && payload.Data.Name == "spez" {
		return modules.Healthy, "reddit: OK"
	}
	return modules.Degraded, "reddit: unexpected response format"
}
