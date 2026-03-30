// SPDX-License-Identifier: AGPL-3.0-or-later

package linktree

import (
	"context"
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/kyle/basalt/internal/graph"
	"github.com/kyle/basalt/internal/httpclient"
	"github.com/kyle/basalt/internal/modules"
)

const linktreeURL = "https://linktr.ee/"

// Module scrapes Linktree profile pages for linked accounts.
type Module struct {
	baseURL string
}

func New() *Module { return &Module{baseURL: linktreeURL} }

func (m *Module) Name() string                  { return "linktree" }
func (m *Module) Description() string            { return "Scrape Linktree profiles for linked social accounts and websites" }
func (m *Module) CanHandle(nodeType string) bool { return nodeType == "username" }

func (m *Module) Extract(ctx context.Context, node *graph.Node, client *httpclient.Client) ([]*graph.Node, []*graph.Edge, error) {
	username := node.Label
	url := m.baseURL + username

	resp, err := client.Do(ctx, url, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("linktree request: %w", err)
	}
	if resp.StatusCode == 404 {
		return nil, nil, nil
	}
	if resp.StatusCode != 200 {
		return nil, nil, fmt.Errorf("linktree returned %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(resp.Body))
	if err != nil {
		return nil, nil, fmt.Errorf("parsing linktree HTML: %w", err)
	}

	var nodes []*graph.Node
	var edges []*graph.Edge

	account := graph.NewAccountNode("linktree", username, url, "linktree")
	account.Confidence = 0.90
	nodes = append(nodes, account)
	edges = append(edges, graph.NewEdge(0, node.ID, account.ID, graph.EdgeTypeHasAccount, "linktree"))

	doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists || href == "" || href == "#" {
			return
		}
		if !strings.HasPrefix(href, "http") {
			return
		}
		if strings.Contains(href, "linktr.ee") {
			return
		}

		websiteNode := graph.NewNode(graph.NodeTypeWebsite, href, "linktree")
		websiteNode.Confidence = 0.75
		websiteNode.Pivot = false
		nodes = append(nodes, websiteNode)
		edges = append(edges, graph.NewEdge(0, account.ID, websiteNode.ID, graph.EdgeTypeMentions, "linktree"))
	})

	return nodes, edges, nil
}

func (m *Module) Verify(ctx context.Context, client *httpclient.Client) (modules.HealthStatus, string) {
	resp, err := client.Do(ctx, m.baseURL+"linktree", nil)
	if err != nil {
		return modules.Offline, fmt.Sprintf("linktree: %v", err)
	}
	if resp.StatusCode == 200 {
		return modules.Healthy, "linktree: OK"
	}
	if resp.StatusCode == 403 || resp.StatusCode == 429 {
		return modules.Degraded, fmt.Sprintf("linktree: %d (may be rate limited)", resp.StatusCode)
	}
	return modules.Offline, fmt.Sprintf("linktree: status %d", resp.StatusCode)
}
