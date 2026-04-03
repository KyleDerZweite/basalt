// SPDX-License-Identifier: AGPL-3.0-or-later

package chesscom

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/KyleDerZweite/basalt/internal/graph"
	"github.com/KyleDerZweite/basalt/internal/httpclient"
	"github.com/KyleDerZweite/basalt/internal/modules"
)

const chesscomBaseURL = "https://api.chess.com"

// Module extracts profile data from Chess.com.
type Module struct {
	baseURL string
}

// New creates a Chess.com module.
func New() *Module {
	return &Module{baseURL: chesscomBaseURL}
}

func (m *Module) Name() string        { return "chesscom" }
func (m *Module) Description() string { return "Extract profile data from Chess.com" }

func (m *Module) CanHandle(nodeType string) bool {
	return nodeType == "username"
}

func (m *Module) Extract(ctx context.Context, node *graph.Node, client *httpclient.Client) ([]*graph.Node, []*graph.Edge, error) {
	username := node.Label
	apiURL := fmt.Sprintf("%s/pub/player/%s", m.baseURL, username)

	resp, err := client.Do(ctx, apiURL, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("chesscom user request: %w", err)
	}
	if resp.StatusCode == 404 {
		return nil, nil, nil
	}
	if resp.StatusCode != 200 {
		return nil, nil, fmt.Errorf("chesscom returned %d", resp.StatusCode)
	}

	var profile struct {
		Username   string `json:"username"`
		Name       string `json:"name"`
		Avatar     string `json:"avatar"`
		URL        string `json:"url"`
		Country    string `json:"country"`
		Location   string `json:"location"`
		Title      string `json:"title"`
		Followers  int    `json:"followers"`
		Joined     int64  `json:"joined"`
		LastOnline int64  `json:"last_online"`
		Status     string `json:"status"`
		IsStreamer bool   `json:"is_streamer"`
		TwitchURL  string `json:"twitch_url"`
	}
	if err := json.Unmarshal([]byte(resp.Body), &profile); err != nil {
		return nil, nil, fmt.Errorf("parsing chesscom response: %w", err)
	}

	var nodes []*graph.Node
	var edges []*graph.Edge

	// Create account node
	account := graph.NewAccountNode("chesscom", username, profile.URL, "chesscom")
	account.Confidence = 0.90
	if profile.Name != "" {
		account.Properties["name"] = profile.Name
	}
	if profile.Avatar != "" {
		account.Properties["avatar"] = profile.Avatar
	}
	if profile.Country != "" {
		account.Properties["country"] = profile.Country
	}
	if profile.Location != "" {
		account.Properties["location"] = profile.Location
	}
	if profile.Title != "" {
		account.Properties["title"] = profile.Title
	}
	if profile.Followers > 0 {
		account.Properties["followers"] = profile.Followers
	}
	if profile.Joined > 0 {
		account.Properties["joined"] = profile.Joined
	}
	if profile.LastOnline > 0 {
		account.Properties["last_online"] = profile.LastOnline
	}
	if profile.Status != "" {
		account.Properties["status"] = profile.Status
	}
	if profile.IsStreamer {
		account.Properties["is_streamer"] = profile.IsStreamer
	}
	nodes = append(nodes, account)
	edges = append(edges, graph.NewEdge(0, node.ID, account.ID, graph.EdgeTypeHasAccount, "chesscom"))

	// Create Twitch website node if available (valuable for pivoting)
	if profile.TwitchURL != "" {
		twitchNode := graph.NewNode(graph.NodeTypeWebsite, profile.TwitchURL, "chesscom")
		twitchNode.Confidence = 0.85
		twitchNode.Pivot = false
		nodes = append(nodes, twitchNode)
		edges = append(edges, graph.NewEdge(0, account.ID, twitchNode.ID, graph.EdgeTypeLinkedTo, "chesscom"))
	}

	return nodes, edges, nil
}

func (m *Module) Verify(ctx context.Context, client *httpclient.Client) (modules.HealthStatus, string) {
	apiURL := fmt.Sprintf("%s/pub/player/hikaru", m.baseURL)
	resp, err := client.Do(ctx, apiURL, nil)
	if err != nil {
		return modules.Offline, fmt.Sprintf("chesscom: %v", err)
	}
	if resp.StatusCode == 200 {
		var user struct {
			Username string `json:"username"`
		}
		if err := json.Unmarshal([]byte(resp.Body), &user); err == nil && user.Username == "hikaru" {
			return modules.Healthy, "chesscom: OK"
		}
		return modules.Degraded, "chesscom: unexpected response format"
	}
	return modules.Offline, fmt.Sprintf("chesscom: status %d", resp.StatusCode)
}
