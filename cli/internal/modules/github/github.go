// SPDX-License-Identifier: AGPL-3.0-or-later

package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/kyle/basalt/internal/graph"
	"github.com/kyle/basalt/internal/httpclient"
	"github.com/kyle/basalt/internal/modules"
)

const githubAPI = "https://api.github.com"

// Module extracts profile data from GitHub's REST API.
type Module struct {
	baseURL string
	token   string
}

// New creates a GitHub module. Token is optional (higher rate limits).
func New(token string) *Module {
	return &Module{baseURL: githubAPI, token: token}
}

func (m *Module) Name() string        { return "github" }
func (m *Module) Description() string { return "Extract profile, email, domain, and social links from GitHub" }

func (m *Module) CanHandle(nodeType string) bool {
	return nodeType == "username" || nodeType == "email"
}

func (m *Module) Extract(ctx context.Context, node *graph.Node, client *httpclient.Client) ([]*graph.Node, []*graph.Edge, error) {
	switch node.Type {
	case "username":
		return m.extractByUsername(ctx, node, client)
	case "email":
		return m.extractByEmail(ctx, node, client)
	default:
		return nil, nil, nil
	}
}

func (m *Module) extractByUsername(ctx context.Context, node *graph.Node, client *httpclient.Client) ([]*graph.Node, []*graph.Edge, error) {
	username := node.Label
	apiURL := fmt.Sprintf("%s/users/%s", m.baseURL, url.PathEscape(username))

	resp, err := client.Do(ctx, apiURL, m.headers())
	if err != nil {
		return nil, nil, fmt.Errorf("github user request: %w", err)
	}
	if resp.StatusCode == 404 {
		return nil, nil, nil
	}
	if resp.StatusCode != 200 {
		return nil, nil, fmt.Errorf("github returned %d", resp.StatusCode)
	}

	var user struct {
		Login           string `json:"login"`
		Name            string `json:"name"`
		Email           string `json:"email"`
		Blog            string `json:"blog"`
		Company         string `json:"company"`
		Location        string `json:"location"`
		Bio             string `json:"bio"`
		HTMLURL         string `json:"html_url"`
		AvatarURL       string `json:"avatar_url"`
		TwitterUsername string `json:"twitter_username"`
	}
	if err := json.Unmarshal([]byte(resp.Body), &user); err != nil {
		return nil, nil, fmt.Errorf("parsing github response: %w", err)
	}

	var nodes []*graph.Node
	var edges []*graph.Edge

	account := graph.NewAccountNode("github", user.Login, user.HTMLURL, "github")
	account.Confidence = 0.95
	account.Properties["company"] = user.Company
	account.Properties["location"] = user.Location
	account.Properties["bio"] = user.Bio
	nodes = append(nodes, account)
	edges = append(edges, graph.NewEdge(0, node.ID, account.ID, graph.EdgeTypeHasAccount, "github"))

	if user.Name != "" {
		nameNode := graph.NewNode(graph.NodeTypeFullName, user.Name, "github")
		nameNode.Confidence = 0.90
		nodes = append(nodes, nameNode)
		edges = append(edges, graph.NewEdge(0, account.ID, nameNode.ID, graph.EdgeTypeLinkedTo, "github"))
	}

	if user.Email != "" {
		emailNode := graph.NewNode(graph.NodeTypeEmail, user.Email, "github")
		emailNode.Pivot = true
		emailNode.Confidence = 0.90
		nodes = append(nodes, emailNode)
		edges = append(edges, graph.NewEdge(0, account.ID, emailNode.ID, graph.EdgeTypeHasEmail, "github"))
	}

	if user.Blog != "" {
		blog := user.Blog
		if !strings.HasPrefix(blog, "http") {
			blog = "https://" + blog
		}
		websiteNode := graph.NewNode(graph.NodeTypeWebsite, blog, "github")
		websiteNode.Pivot = true
		websiteNode.Confidence = 0.85
		nodes = append(nodes, websiteNode)
		edges = append(edges, graph.NewEdge(0, account.ID, websiteNode.ID, graph.EdgeTypeHasDomain, "github"))

		if parsed, err := url.Parse(blog); err == nil && parsed.Host != "" {
			domainNode := graph.NewNode(graph.NodeTypeDomain, parsed.Host, "github")
			domainNode.Pivot = true
			domainNode.Confidence = 0.85
			nodes = append(nodes, domainNode)
			edges = append(edges, graph.NewEdge(0, account.ID, domainNode.ID, graph.EdgeTypeHasDomain, "github"))
		}
	}

	if user.AvatarURL != "" {
		avatarNode := graph.NewNode(graph.NodeTypeAvatarURL, user.AvatarURL, "github")
		avatarNode.Confidence = 0.95
		nodes = append(nodes, avatarNode)
		edges = append(edges, graph.NewEdge(0, account.ID, avatarNode.ID, graph.EdgeTypeLinkedTo, "github"))
	}

	if user.TwitterUsername != "" {
		twitterNode := graph.NewNode(graph.NodeTypeUsername, user.TwitterUsername, "github")
		twitterNode.Pivot = true
		twitterNode.Confidence = 0.80
		twitterNode.Properties["platform_hint"] = "twitter"
		nodes = append(nodes, twitterNode)
		edges = append(edges, graph.NewEdge(0, account.ID, twitterNode.ID, graph.EdgeTypeHasUsername, "github"))
	}

	return nodes, edges, nil
}

func (m *Module) extractByEmail(ctx context.Context, node *graph.Node, client *httpclient.Client) ([]*graph.Node, []*graph.Edge, error) {
	email := node.Label
	apiURL := fmt.Sprintf("%s/search/users?q=%s+in:email", m.baseURL, url.QueryEscape(email))

	resp, err := client.Do(ctx, apiURL, m.headers())
	if err != nil {
		return nil, nil, fmt.Errorf("github email search: %w", err)
	}
	if resp.StatusCode != 200 {
		return nil, nil, fmt.Errorf("github search returned %d", resp.StatusCode)
	}

	var result struct {
		TotalCount int `json:"total_count"`
		Items      []struct {
			Login   string `json:"login"`
			HTMLURL string `json:"html_url"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(resp.Body), &result); err != nil {
		return nil, nil, fmt.Errorf("parsing github search: %w", err)
	}

	var nodes []*graph.Node
	var edges []*graph.Edge

	for _, item := range result.Items {
		usernameNode := graph.NewNode(graph.NodeTypeUsername, item.Login, "github")
		usernameNode.Pivot = true
		usernameNode.Confidence = 0.85
		nodes = append(nodes, usernameNode)
		edges = append(edges, graph.NewEdge(0, node.ID, usernameNode.ID, graph.EdgeTypeHasUsername, "github"))
	}

	return nodes, edges, nil
}

func (m *Module) Verify(ctx context.Context, client *httpclient.Client) (modules.HealthStatus, string) {
	apiURL := fmt.Sprintf("%s/users/octocat", m.baseURL)
	resp, err := client.Do(ctx, apiURL, m.headers())
	if err != nil {
		return modules.Offline, fmt.Sprintf("github: %v", err)
	}
	if resp.StatusCode == 200 {
		var user struct {
			Login string `json:"login"`
		}
		if err := json.Unmarshal([]byte(resp.Body), &user); err == nil && user.Login == "octocat" {
			return modules.Healthy, "github: OK"
		}
		return modules.Degraded, "github: unexpected response format"
	}
	if resp.StatusCode == 403 {
		return modules.Degraded, "github: rate limited (consider setting GITHUB_TOKEN)"
	}
	return modules.Offline, fmt.Sprintf("github: status %d", resp.StatusCode)
}

func (m *Module) headers() map[string]string {
	h := map[string]string{
		"Accept": "application/vnd.github+json",
	}
	if m.token != "" {
		h["Authorization"] = "Bearer " + m.token
	}
	return h
}
