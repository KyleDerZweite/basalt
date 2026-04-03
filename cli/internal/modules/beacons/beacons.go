// SPDX-License-Identifier: AGPL-3.0-or-later

package beacons

import (
	"context"
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/KyleDerZweite/basalt/internal/graph"
	"github.com/KyleDerZweite/basalt/internal/httpclient"
	"github.com/KyleDerZweite/basalt/internal/modules"
)

const beaconsURL = "https://beacons.ai/"

// Module scrapes Beacons.ai profile pages for linked accounts.
type Module struct {
	baseURL string
}

func New() *Module { return &Module{baseURL: beaconsURL} }

func (m *Module) Name() string { return "beacons" }
func (m *Module) Description() string {
	return "Scrape Beacons.ai profiles for linked accounts and websites"
}
func (m *Module) CanHandle(nodeType string) bool { return nodeType == "username" }

func (m *Module) Extract(ctx context.Context, node *graph.Node, client *httpclient.Client) ([]*graph.Node, []*graph.Edge, error) {
	username := node.Label
	url := m.baseURL + username

	resp, err := client.Do(ctx, url, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("beacons request: %w", err)
	}
	if resp.StatusCode == 404 {
		return nil, nil, nil
	}
	if resp.StatusCode != 200 {
		return nil, nil, fmt.Errorf("beacons returned %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(resp.Body))
	if err != nil {
		return nil, nil, fmt.Errorf("parsing beacons HTML: %w", err)
	}

	var nodes []*graph.Node
	var edges []*graph.Edge

	account := graph.NewAccountNode("beacons", username, url, "beacons")
	account.Confidence = 0.90
	nodes = append(nodes, account)
	edges = append(edges, graph.NewEdge(0, node.ID, account.ID, graph.EdgeTypeHasAccount, "beacons"))

	doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists || href == "" || href == "#" {
			return
		}
		if !strings.HasPrefix(href, "http") {
			return
		}
		if strings.Contains(href, "beacons.ai") {
			return
		}

		websiteNode := graph.NewNode(graph.NodeTypeWebsite, href, "beacons")
		websiteNode.Confidence = 0.75
		websiteNode.Pivot = false
		nodes = append(nodes, websiteNode)
		edges = append(edges, graph.NewEdge(0, account.ID, websiteNode.ID, graph.EdgeTypeMentions, "beacons"))
	})

	return nodes, edges, nil
}

func (m *Module) Verify(ctx context.Context, client *httpclient.Client) (modules.HealthStatus, string) {
	resp, err := client.Do(ctx, m.baseURL+"linkinbio", nil)
	if err != nil {
		return modules.Offline, fmt.Sprintf("beacons: %v", err)
	}
	if resp.StatusCode == 200 {
		return modules.Healthy, "beacons: OK"
	}
	if resp.StatusCode == 403 || resp.StatusCode == 429 {
		return modules.Offline, fmt.Sprintf("beacons: blocked by Cloudflare or rate limit (%d)", resp.StatusCode)
	}
	return modules.Offline, fmt.Sprintf("beacons: status %d", resp.StatusCode)
}
