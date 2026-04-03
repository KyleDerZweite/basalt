// SPDX-License-Identifier: AGPL-3.0-or-later

package bento

import (
	"context"
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/KyleDerZweite/basalt/internal/graph"
	"github.com/KyleDerZweite/basalt/internal/httpclient"
	"github.com/KyleDerZweite/basalt/internal/modules"
)

const bentoURL = "https://bento.me/"

// Module scrapes Bento.me profile pages for linked accounts.
type Module struct {
	baseURL string
}

func New() *Module { return &Module{baseURL: bentoURL} }

func (m *Module) Name() string { return "bento" }
func (m *Module) Description() string {
	return "Scrape Bento.me profiles for linked accounts and websites"
}
func (m *Module) CanHandle(nodeType string) bool { return nodeType == "username" }

func (m *Module) Extract(ctx context.Context, node *graph.Node, client *httpclient.Client) ([]*graph.Node, []*graph.Edge, error) {
	username := node.Label
	url := m.baseURL + username

	resp, err := client.Do(ctx, url, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("bento request: %w", err)
	}
	if resp.StatusCode == 404 {
		return nil, nil, nil
	}
	if resp.StatusCode != 200 {
		return nil, nil, fmt.Errorf("bento returned %d", resp.StatusCode)
	}
	// Bento redirects non-existent profiles to a different domain.
	// If the final URL differs from what we requested, it's a redirect.
	if resp.FinalURL != "" && resp.FinalURL != url {
		return nil, nil, nil
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(resp.Body))
	if err != nil {
		return nil, nil, fmt.Errorf("parsing bento HTML: %w", err)
	}

	var nodes []*graph.Node
	var edges []*graph.Edge

	account := graph.NewAccountNode("bento", username, url, "bento")
	account.Confidence = 0.90
	nodes = append(nodes, account)
	edges = append(edges, graph.NewEdge(0, node.ID, account.ID, graph.EdgeTypeHasAccount, "bento"))

	doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists || href == "" || href == "#" {
			return
		}
		if !strings.HasPrefix(href, "http") {
			return
		}
		if strings.Contains(href, "bento.me") {
			return
		}

		websiteNode := graph.NewNode(graph.NodeTypeWebsite, href, "bento")
		websiteNode.Confidence = 0.75
		websiteNode.Pivot = false
		nodes = append(nodes, websiteNode)
		edges = append(edges, graph.NewEdge(0, account.ID, websiteNode.ID, graph.EdgeTypeMentions, "bento"))
	})

	return nodes, edges, nil
}

func (m *Module) Verify(ctx context.Context, client *httpclient.Client) (modules.HealthStatus, string) {
	resp, err := client.Do(ctx, m.baseURL+"bento", nil)
	if err != nil {
		return modules.Offline, fmt.Sprintf("bento: %v", err)
	}
	if resp.StatusCode == 200 {
		return modules.Healthy, "bento: OK"
	}
	if resp.StatusCode == 403 || resp.StatusCode == 429 {
		return modules.Degraded, fmt.Sprintf("bento: %d (may be rate limited)", resp.StatusCode)
	}
	return modules.Offline, fmt.Sprintf("bento: status %d", resp.StatusCode)
}
