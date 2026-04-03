// SPDX-License-Identifier: AGPL-3.0-or-later

package myanimelist

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/KyleDerZweite/basalt/internal/graph"
	"github.com/KyleDerZweite/basalt/internal/httpclient"
	"github.com/KyleDerZweite/basalt/internal/modules"
)

const defaultBaseURL = "https://api.jikan.moe"

// Module extracts profile data from MyAnimeList using the Jikan API.
type Module struct {
	baseURL string
}

// New creates a MyAnimeList module.
func New() *Module {
	return &Module{baseURL: defaultBaseURL}
}

func (m *Module) Name() string                   { return "myanimelist" }
func (m *Module) Description() string            { return "Extract profile data from MyAnimeList via Jikan API" }
func (m *Module) CanHandle(nodeType string) bool { return nodeType == "username" }

func (m *Module) Extract(ctx context.Context, node *graph.Node, client *httpclient.Client) ([]*graph.Node, []*graph.Edge, error) {
	username := node.Label
	url := fmt.Sprintf("%s/v4/users/%s", m.baseURL, username)

	resp, err := client.Do(ctx, url, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("myanimelist request: %w", err)
	}
	if resp.StatusCode == 404 {
		return nil, nil, nil
	}
	if resp.StatusCode != 200 {
		return nil, nil, fmt.Errorf("myanimelist returned %d", resp.StatusCode)
	}

	// Jikan API wraps the response in a "data" field
	var respWrapper struct {
		Data struct {
			Username string `json:"username"`
			URL      string `json:"url"`
			Images   struct {
				JPG struct {
					ImageURL string `json:"image_url"`
				} `json:"jpg"`
			} `json:"images"`
			LastOnline string `json:"last_online"`
			Gender     string `json:"gender"`
			Birthday   string `json:"birthday"`
			Location   string `json:"location"`
			Joined     string `json:"joined"`
			MalID      int    `json:"mal_id"`
		} `json:"data"`
	}

	if err := json.Unmarshal([]byte(resp.Body), &respWrapper); err != nil {
		return nil, nil, fmt.Errorf("parsing myanimelist response: %w", err)
	}

	data := respWrapper.Data
	profileURL := data.URL
	if profileURL == "" {
		profileURL = fmt.Sprintf("https://myanimelist.net/profile/%s", username)
	}

	account := graph.NewAccountNode("myanimelist", username, profileURL, "myanimelist")
	account.Confidence = 0.90

	// Store available profile properties
	if data.Gender != "" {
		account.Properties["gender"] = data.Gender
	}
	if data.Birthday != "" {
		account.Properties["birthday"] = data.Birthday
	}
	if data.Location != "" {
		account.Properties["location"] = data.Location
	}
	if data.Joined != "" {
		account.Properties["joined"] = data.Joined
	}
	if data.LastOnline != "" {
		account.Properties["last_online"] = data.LastOnline
	}
	if data.MalID > 0 {
		account.Properties["mal_id"] = data.MalID
	}
	if data.Images.JPG.ImageURL != "" {
		account.Properties["avatar"] = data.Images.JPG.ImageURL
	}

	nodes := []*graph.Node{account}
	edges := []*graph.Edge{
		graph.NewEdge(0, node.ID, account.ID, graph.EdgeTypeHasAccount, "myanimelist"),
	}

	return nodes, edges, nil
}

func (m *Module) Verify(ctx context.Context, client *httpclient.Client) (modules.HealthStatus, string) {
	url := fmt.Sprintf("%s/v4/users/Nekomata1037", m.baseURL)
	resp, err := client.Do(ctx, url, nil)
	if err != nil {
		return modules.Offline, fmt.Sprintf("myanimelist: %v", err)
	}
	if resp.StatusCode != 200 {
		return modules.Offline, fmt.Sprintf("myanimelist: status %d", resp.StatusCode)
	}

	var respWrapper struct {
		Data struct {
			Username string `json:"username"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(resp.Body), &respWrapper); err != nil {
		return modules.Degraded, fmt.Sprintf("myanimelist: parse error: %v", err)
	}

	if respWrapper.Data.Username == "Nekomata1037" {
		return modules.Healthy, "myanimelist: OK"
	}
	return modules.Degraded, "myanimelist: unexpected response format"
}
