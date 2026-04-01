// SPDX-License-Identifier: AGPL-3.0-or-later

package opgg

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/kyle/basalt/internal/graph"
	"github.com/kyle/basalt/internal/httpclient"
	"github.com/kyle/basalt/internal/modules"
)

const defaultBaseURL = "https://www.op.gg"

// regions to check for each username. Most popular first.
var regions = []string{"euw", "na", "eune", "kr", "br", "oce", "jp", "tr", "lan", "las"}

// Module extracts League of Legends profile data from OP.GG.
type Module struct {
	baseURL string
}

// New creates an OP.GG module.
func New() *Module {
	return &Module{baseURL: defaultBaseURL}
}

func (m *Module) Name() string                   { return "opgg" }
func (m *Module) Description() string            { return "Extract League of Legends profile data from OP.GG" }
func (m *Module) CanHandle(nodeType string) bool { return nodeType == "username" }

func (m *Module) Extract(ctx context.Context, node *graph.Node, client *httpclient.Client) ([]*graph.Node, []*graph.Edge, error) {
	username := node.Label

	// OP.GG search endpoint resolves Riot ID tags automatically.
	// /summoners/search?q={name}&region={region} returns the profile
	// page with og:title containing the full Riot ID if found.
	for _, region := range regions {
		select {
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		default:
		}

		searchURL := fmt.Sprintf("%s/summoners/search?q=%s&region=%s", m.baseURL, url.QueryEscape(username), region)
		resp, err := client.Do(ctx, searchURL, nil)
		if err != nil {
			continue
		}
		if resp.StatusCode != 200 {
			continue
		}

		doc, err := goquery.NewDocumentFromReader(strings.NewReader(resp.Body))
		if err != nil {
			continue
		}

		title, hasTitle := doc.Find(`meta[property="og:title"]`).Attr("content")
		if !hasTitle || title == "" {
			continue
		}

		// Title format: "Name#Tag - Summoner Stats - League of Legends"
		// If it doesn't contain " - Summoner Stats", it's not a real profile.
		if !strings.Contains(title, " - Summoner Stats") {
			continue
		}

		// Get the canonical profile URL from og:url.
		profileURL := searchURL
		if ogURL, exists := doc.Find(`meta[property="og:url"]`).Attr("content"); exists && ogURL != "" {
			profileURL = ogURL
		}

		var nodes []*graph.Node
		var edges []*graph.Edge

		account := graph.NewAccountNode("opgg", username, profileURL, "opgg")
		account.Confidence = 0.90
		account.Properties["region"] = region

		// Extract summoner name (includes tag) from title.
		summonerName := strings.Split(title, " - Summoner Stats")[0]
		if summonerName != "" {
			account.Properties["summoner_name"] = summonerName
		}

		if desc, exists := doc.Find(`meta[property="og:description"]`).Attr("content"); exists && desc != "" {
			account.Properties["description"] = desc
			// Description often contains level: "Name#Tag / Lv. 120"
			if parts := strings.Split(desc, " / Lv. "); len(parts) == 2 {
				account.Properties["level"] = parts[1]
			}
		}

		nodes = append(nodes, account)
		edges = append(edges, graph.NewEdge(0, node.ID, account.ID, graph.EdgeTypeHasAccount, "opgg"))

		return nodes, edges, nil
	}

	return nil, nil, nil
}

func (m *Module) Verify(ctx context.Context, client *httpclient.Client) (modules.HealthStatus, string) {
	// Use the search endpoint to look up a known player.
	searchURL := fmt.Sprintf("%s/summoners/search?q=Faker&region=kr", m.baseURL)
	resp, err := client.Do(ctx, searchURL, nil)
	if err != nil {
		return modules.Offline, fmt.Sprintf("opgg: %v", err)
	}
	if resp.StatusCode != 200 {
		return modules.Degraded, fmt.Sprintf("opgg: status %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(resp.Body))
	if err != nil {
		return modules.Degraded, "opgg: failed to parse HTML"
	}
	title, _ := doc.Find(`meta[property="og:title"]`).Attr("content")
	if strings.Contains(title, "Summoner Stats") {
		return modules.Healthy, "opgg: OK"
	}
	return modules.Degraded, "opgg: unexpected response"
}
