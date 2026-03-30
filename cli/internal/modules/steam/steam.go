// SPDX-License-Identifier: AGPL-3.0-or-later

package steam

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/kyle/basalt/internal/graph"
	"github.com/kyle/basalt/internal/httpclient"
	"github.com/kyle/basalt/internal/modules"
)

const steamAPI = "https://api.steampowered.com"

// Module extracts profile data from Steam via API key.
type Module struct {
	baseURL string
	apiKey  string
}

// New creates a Steam module. Requires an API key.
func New(apiKey string) *Module {
	return &Module{baseURL: steamAPI, apiKey: apiKey}
}

func (m *Module) Name() string                    { return "steam" }
func (m *Module) Description() string             { return "Extract profile, aliases, and friends from Steam (requires API key)" }
func (m *Module) CanHandle(nodeType string) bool { return nodeType == "username" }

func (m *Module) Extract(ctx context.Context, node *graph.Node, client *httpclient.Client) ([]*graph.Node, []*graph.Edge, error) {
	// Step 1: Resolve vanity URL to SteamID.
	resolveURL := fmt.Sprintf("%s/ISteamUser/ResolveVanityURL/v1/?key=%s&vanityurl=%s",
		m.baseURL, url.QueryEscape(m.apiKey), url.QueryEscape(node.Label))

	resp, err := client.Do(ctx, resolveURL, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("steam resolve: %w", err)
	}
	if resp.StatusCode != 200 {
		return nil, nil, fmt.Errorf("steam resolve returned %d", resp.StatusCode)
	}

	var resolve struct {
		Response struct {
			Success int    `json:"success"`
			SteamID string `json:"steamid"`
		} `json:"response"`
	}
	if err := json.Unmarshal([]byte(resp.Body), &resolve); err != nil {
		return nil, nil, fmt.Errorf("parsing steam resolve: %w", err)
	}
	if resolve.Response.Success != 1 {
		return nil, nil, nil
	}

	// Step 2: Get player summary.
	summaryURL := fmt.Sprintf("%s/ISteamUser/GetPlayerSummaries/v2/?key=%s&steamids=%s",
		m.baseURL, url.QueryEscape(m.apiKey), url.QueryEscape(resolve.Response.SteamID))

	resp, err = client.Do(ctx, summaryURL, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("steam summary: %w", err)
	}
	if resp.StatusCode != 200 {
		return nil, nil, fmt.Errorf("steam summary returned %d", resp.StatusCode)
	}

	var summary struct {
		Response struct {
			Players []struct {
				SteamID        string `json:"steamid"`
				PersonaName    string `json:"personaname"`
				RealName       string `json:"realname"`
				ProfileURL     string `json:"profileurl"`
				AvatarFull     string `json:"avatarfull"`
				LocCountryCode string `json:"loccountrycode"`
			} `json:"players"`
		} `json:"response"`
	}
	if err := json.Unmarshal([]byte(resp.Body), &summary); err != nil {
		return nil, nil, fmt.Errorf("parsing steam summary: %w", err)
	}
	if len(summary.Response.Players) == 0 {
		return nil, nil, nil
	}

	player := summary.Response.Players[0]
	var nodes []*graph.Node
	var edges []*graph.Edge

	account := graph.NewAccountNode("steam", node.Label, player.ProfileURL, "steam")
	account.Confidence = 0.95
	account.Properties["steamid"] = player.SteamID
	account.Properties["persona_name"] = player.PersonaName
	account.Properties["country"] = player.LocCountryCode
	nodes = append(nodes, account)
	edges = append(edges, graph.NewEdge(0, node.ID, account.ID, graph.EdgeTypeHasAccount, "steam"))

	if player.RealName != "" {
		nameNode := graph.NewNode(graph.NodeTypeFullName, player.RealName, "steam")
		nameNode.Confidence = 0.80
		nodes = append(nodes, nameNode)
		edges = append(edges, graph.NewEdge(0, account.ID, nameNode.ID, graph.EdgeTypeLinkedTo, "steam"))
	}

	if player.AvatarFull != "" {
		avatarNode := graph.NewNode(graph.NodeTypeAvatarURL, player.AvatarFull, "steam")
		avatarNode.Confidence = 0.95
		nodes = append(nodes, avatarNode)
		edges = append(edges, graph.NewEdge(0, account.ID, avatarNode.ID, graph.EdgeTypeLinkedTo, "steam"))
	}

	return nodes, edges, nil
}

func (m *Module) Verify(ctx context.Context, client *httpclient.Client) (modules.HealthStatus, string) {
	if m.apiKey == "" {
		return modules.Offline, "steam: no API key configured (set STEAM_API_KEY)"
	}

	resolveURL := fmt.Sprintf("%s/ISteamUser/ResolveVanityURL/v1/?key=%s&vanityurl=valve",
		m.baseURL, url.QueryEscape(m.apiKey))

	resp, err := client.Do(ctx, resolveURL, nil)
	if err != nil {
		return modules.Offline, fmt.Sprintf("steam: %v", err)
	}
	if resp.StatusCode == 200 {
		return modules.Healthy, "steam: OK"
	}
	if resp.StatusCode == 403 {
		return modules.Offline, "steam: invalid API key"
	}
	return modules.Degraded, fmt.Sprintf("steam: unexpected status %d", resp.StatusCode)
}
