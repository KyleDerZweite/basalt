// SPDX-License-Identifier: AGPL-3.0-or-later

package carrd

import (
	"context"
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/kyle/basalt/internal/graph"
	"github.com/kyle/basalt/internal/httpclient"
	"github.com/kyle/basalt/internal/modules"
)

const carrdSuffix = ".carrd.co"

// Module scrapes Carrd.co profile pages for linked accounts.
type Module struct {
	baseURL string

	// overrideURL, if set, is used as a prefix instead of subdomain construction.
	// This allows tests to point at a local httptest server.
	overrideURL string

	// verifyURL, if set, overrides the default verify endpoint.
	verifyURL string
}

func New() *Module { return &Module{baseURL: carrdSuffix} }

func (m *Module) Name() string                  { return "carrd" }
func (m *Module) Description() string            { return "Scrape Carrd.co profiles for linked accounts and websites" }
func (m *Module) CanHandle(nodeType string) bool { return nodeType == "username" }

// profileURL builds the profile URL for the given username.
func (m *Module) profileURL(username string) string {
	if m.overrideURL != "" {
		return m.overrideURL + username
	}
	return "https://" + username + m.baseURL
}

func (m *Module) Extract(ctx context.Context, node *graph.Node, client *httpclient.Client) ([]*graph.Node, []*graph.Edge, error) {
	username := node.Label
	url := m.profileURL(username)

	resp, err := client.Do(ctx, url, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("carrd request: %w", err)
	}
	if resp.StatusCode == 404 {
		return nil, nil, nil
	}
	if resp.StatusCode != 200 {
		return nil, nil, fmt.Errorf("carrd returned %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(resp.Body))
	if err != nil {
		return nil, nil, fmt.Errorf("parsing carrd HTML: %w", err)
	}

	var nodes []*graph.Node
	var edges []*graph.Edge

	// Use the canonical subdomain URL for the account node, even during tests.
	canonicalURL := "https://" + username + carrdSuffix
	account := graph.NewAccountNode("carrd", username, canonicalURL, "carrd")
	account.Confidence = 0.90
	nodes = append(nodes, account)
	edges = append(edges, graph.NewEdge(0, node.ID, account.ID, graph.EdgeTypeHasAccount, "carrd"))

	doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists || href == "" || href == "#" {
			return
		}
		if !strings.HasPrefix(href, "http") {
			return
		}
		if strings.Contains(href, "carrd.co") {
			return
		}

		websiteNode := graph.NewNode(graph.NodeTypeWebsite, href, "carrd")
		websiteNode.Confidence = 0.75
		websiteNode.Pivot = false
		nodes = append(nodes, websiteNode)
		edges = append(edges, graph.NewEdge(0, account.ID, websiteNode.ID, graph.EdgeTypeMentions, "carrd"))
	})

	return nodes, edges, nil
}

func (m *Module) Verify(ctx context.Context, client *httpclient.Client) (modules.HealthStatus, string) {
	url := "https://carrd.co"
	if m.verifyURL != "" {
		url = m.verifyURL
	}

	resp, err := client.Do(ctx, url, nil)
	if err != nil {
		return modules.Offline, fmt.Sprintf("carrd: %v", err)
	}
	if resp.StatusCode == 200 {
		return modules.Healthy, "carrd: OK"
	}
	if resp.StatusCode == 403 || resp.StatusCode == 429 {
		return modules.Degraded, fmt.Sprintf("carrd: %d (may be rate limited)", resp.StatusCode)
	}
	return modules.Offline, fmt.Sprintf("carrd: status %d", resp.StatusCode)
}
