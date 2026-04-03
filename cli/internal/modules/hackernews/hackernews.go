// SPDX-License-Identifier: AGPL-3.0-or-later

package hackernews

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/KyleDerZweite/basalt/internal/graph"
	"github.com/KyleDerZweite/basalt/internal/httpclient"
	"github.com/KyleDerZweite/basalt/internal/modules"
)

const hackerNewsURL = "https://hacker-news.firebaseio.com"

// Module extracts profile data from Hacker News.
type Module struct {
	baseURL string
}

// New creates a HackerNews module.
func New() *Module {
	return &Module{baseURL: hackerNewsURL}
}

func (m *Module) Name() string                   { return "hackernews" }
func (m *Module) Description() string            { return "Extract profile data from Hacker News" }
func (m *Module) CanHandle(nodeType string) bool { return nodeType == "username" }

func (m *Module) Extract(ctx context.Context, node *graph.Node, client *httpclient.Client) ([]*graph.Node, []*graph.Edge, error) {
	username := node.Label
	url := fmt.Sprintf("%s/v0/user/%s.json", m.baseURL, username)

	resp, err := client.Do(ctx, url, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("hackernews request: %w", err)
	}
	if resp.StatusCode != 200 || resp.Body == "null" {
		return nil, nil, nil
	}

	var profile struct {
		ID      string `json:"id"`
		Created int64  `json:"created"`
		Karma   int    `json:"karma"`
		About   string `json:"about"`
	}
	if err := json.Unmarshal([]byte(resp.Body), &profile); err != nil {
		return nil, nil, fmt.Errorf("parsing hackernews response: %w", err)
	}

	profileURL := fmt.Sprintf("https://news.ycombinator.com/user?id=%s", username)
	account := graph.NewAccountNode("hackernews", username, profileURL, "hackernews")
	account.Confidence = 0.95
	account.Properties["karma"] = profile.Karma
	account.Properties["created"] = profile.Created
	if profile.About != "" {
		account.Properties["about"] = profile.About
	}

	nodes := []*graph.Node{account}
	edges := []*graph.Edge{
		graph.NewEdge(0, node.ID, account.ID, graph.EdgeTypeHasAccount, "hackernews"),
	}

	return nodes, edges, nil
}

func (m *Module) Verify(ctx context.Context, client *httpclient.Client) (modules.HealthStatus, string) {
	url := fmt.Sprintf("%s/v0/user/dang.json", m.baseURL)
	resp, err := client.Do(ctx, url, nil)
	if err != nil {
		return modules.Offline, fmt.Sprintf("hackernews: %v", err)
	}
	if resp.StatusCode != 200 {
		return modules.Degraded, fmt.Sprintf("hackernews: unexpected status %d", resp.StatusCode)
	}

	var profile struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(resp.Body), &profile); err == nil && profile.ID == "dang" {
		return modules.Healthy, "hackernews: OK"
	}
	return modules.Degraded, "hackernews: unexpected response format"
}
