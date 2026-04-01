// SPDX-License-Identifier: AGPL-3.0-or-later

package opgg

import (
	"context"
	"fmt"
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

	// OP.GG URL format: /lol/summoners/{region}/{name}-{tag}
	// Try the username as both the name and the tag (common pattern).
	// Also try without tag for exact vanity URLs.
	for _, region := range regions {
		select {
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		default:
		}

		url := fmt.Sprintf("%s/lol/summoners/%s/%s-%s", m.baseURL, region, username, strings.ToUpper(region))
		resp, err := client.Do(ctx, url, nil)
		if err != nil {
			continue
		}
		if resp.StatusCode == 404 {
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

		var nodes []*graph.Node
		var edges []*graph.Edge

		account := graph.NewAccountNode("opgg", username, url, "opgg")
		account.Confidence = 0.90
		account.Properties["region"] = region

		// Extract summoner name from title.
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
	// Check that OP.GG is reachable by looking up Faker (famous LoL player).
	url := fmt.Sprintf("%s/lol/summoners/kr/Hide%%20on%%20bush-KR1", m.baseURL)
	resp, err := client.Do(ctx, url, nil)
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
