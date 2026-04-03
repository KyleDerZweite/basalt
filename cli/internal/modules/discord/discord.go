// SPDX-License-Identifier: AGPL-3.0-or-later

package discord

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/KyleDerZweite/basalt/internal/graph"
	"github.com/KyleDerZweite/basalt/internal/httpclient"
	"github.com/KyleDerZweite/basalt/internal/modules"
)

const discordURL = "https://discord.com"

// Module checks username availability on Discord to infer account existence.
type Module struct {
	baseURL string
}

// New creates a Discord module.
func New() *Module {
	return &Module{baseURL: discordURL}
}

func (m *Module) Name() string                   { return "discord" }
func (m *Module) Description() string            { return "Check Discord username existence via validation API" }
func (m *Module) CanHandle(nodeType string) bool { return nodeType == "username" }

func (m *Module) Extract(ctx context.Context, node *graph.Node, client *httpclient.Client) ([]*graph.Node, []*graph.Edge, error) {
	username := node.Label
	url := fmt.Sprintf("%s/api/v9/unique-username/validate", m.baseURL)

	reqBody := fmt.Sprintf(`{"username": %q}`, username)
	headers := map[string]string{
		"Content-Type": "application/json",
	}

	resp, err := client.DoRequest(ctx, "POST", url, strings.NewReader(reqBody), headers)
	if err != nil {
		return nil, nil, fmt.Errorf("discord request: %w", err)
	}
	if resp.StatusCode == 404 {
		return nil, nil, nil
	}
	if resp.StatusCode != 200 {
		return nil, nil, fmt.Errorf("discord returned %d", resp.StatusCode)
	}

	var result struct {
		Taken bool `json:"taken"`
	}
	if err := json.Unmarshal([]byte(resp.Body), &result); err != nil {
		return nil, nil, fmt.Errorf("parsing discord response: %w", err)
	}

	if !result.Taken {
		return nil, nil, nil
	}

	var nodes []*graph.Node
	var edges []*graph.Edge

	profileURL := fmt.Sprintf("https://discord.com/users/%s", username)
	account := graph.NewAccountNode("discord", username, profileURL, "discord")
	account.Confidence = 0.70
	nodes = append(nodes, account)
	edges = append(edges, graph.NewEdge(0, node.ID, account.ID, graph.EdgeTypeHasAccount, "discord"))

	return nodes, edges, nil
}

func (m *Module) Verify(ctx context.Context, client *httpclient.Client) (modules.HealthStatus, string) {
	url := fmt.Sprintf("%s/api/v9/unique-username/validate", m.baseURL)

	reqBody := `{"username": "discord"}`
	headers := map[string]string{
		"Content-Type": "application/json",
	}

	resp, err := client.DoRequest(ctx, "POST", url, strings.NewReader(reqBody), headers)
	if err != nil {
		return modules.Offline, fmt.Sprintf("discord: %v", err)
	}
	if resp.StatusCode == 200 {
		return modules.Healthy, "discord: OK"
	}
	if resp.StatusCode == 404 {
		return modules.Offline, "discord: validation endpoint removed (404)"
	}
	if resp.StatusCode == 403 || resp.StatusCode == 429 {
		return modules.Offline, fmt.Sprintf("discord: blocked (%d)", resp.StatusCode)
	}
	return modules.Degraded, fmt.Sprintf("discord: unexpected status %d", resp.StatusCode)
}
