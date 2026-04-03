// SPDX-License-Identifier: AGPL-3.0-or-later

package lichess

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/KyleDerZweite/basalt/internal/graph"
	"github.com/KyleDerZweite/basalt/internal/httpclient"
	"github.com/KyleDerZweite/basalt/internal/modules"
)

const lichessBaseURL = "https://lichess.org"

// Module extracts profile data from Lichess.
type Module struct {
	baseURL string
}

// New creates a Lichess module.
func New() *Module {
	return &Module{baseURL: lichessBaseURL}
}

func (m *Module) Name() string        { return "lichess" }
func (m *Module) Description() string { return "Extract profile data and linked websites from Lichess" }

func (m *Module) CanHandle(nodeType string) bool {
	return nodeType == "username"
}

func (m *Module) Extract(ctx context.Context, node *graph.Node, client *httpclient.Client) ([]*graph.Node, []*graph.Edge, error) {
	apiURL := fmt.Sprintf("%s/api/user/%s", m.baseURL, url.PathEscape(node.Label))

	resp, err := client.Do(ctx, apiURL, map[string]string{"Accept": "application/json"})
	if err != nil {
		return nil, nil, fmt.Errorf("lichess request: %w", err)
	}
	if resp.StatusCode == 404 {
		return nil, nil, nil
	}
	if resp.StatusCode != 200 {
		return nil, nil, fmt.Errorf("lichess returned %d", resp.StatusCode)
	}

	var profile struct {
		ID        string `json:"id"`
		Username  string `json:"username"`
		URL       string `json:"url"`
		Verified  bool   `json:"verified"`
		Patron    bool   `json:"patron"`
		Flair     string `json:"flair"`
		CreatedAt int64  `json:"createdAt"`
		SeenAt    int64  `json:"seenAt"`
		Count     struct {
			All   int `json:"all"`
			Rated int `json:"rated"`
		} `json:"count"`
		PlayTime struct {
			Total int `json:"total"`
			TV    int `json:"tv"`
		} `json:"playTime"`
		Perfs struct {
			Bullet    lichessPerf `json:"bullet"`
			Blitz     lichessPerf `json:"blitz"`
			Rapid     lichessPerf `json:"rapid"`
			Classical lichessPerf `json:"classical"`
		} `json:"perfs"`
		Profile struct {
			FirstName string `json:"firstName"`
			LastName  string `json:"lastName"`
			Bio       string `json:"bio"`
			Location  string `json:"location"`
			Links     string `json:"links"`
		} `json:"profile"`
	}
	if err := json.Unmarshal([]byte(resp.Body), &profile); err != nil {
		return nil, nil, fmt.Errorf("parsing lichess response: %w", err)
	}
	if profile.ID == "" {
		return nil, nil, nil
	}

	username := node.Label
	if profile.Username != "" {
		username = profile.Username
	}

	profileURL := profile.URL
	if profileURL == "" {
		profileURL = fmt.Sprintf("%s/@/%s", m.baseURL, url.PathEscape(username))
	}

	var nodes []*graph.Node
	var edges []*graph.Edge

	account := graph.NewAccountNode("lichess", username, profileURL, "lichess")
	account.Confidence = 0.90
	if profile.Profile.Location != "" {
		account.Properties["location"] = profile.Profile.Location
	}
	if profile.Profile.Bio != "" {
		account.Properties["bio"] = profile.Profile.Bio
	}
	if profile.Flair != "" {
		account.Properties["flair"] = profile.Flair
	}
	if profile.Verified {
		account.Properties["verified"] = true
	}
	if profile.Patron {
		account.Properties["patron"] = true
	}
	if profile.CreatedAt > 0 {
		account.Properties["created_at"] = profile.CreatedAt
	}
	if profile.SeenAt > 0 {
		account.Properties["seen_at"] = profile.SeenAt
	}
	if profile.Count.All > 0 {
		account.Properties["games_total"] = profile.Count.All
	}
	if profile.Count.Rated > 0 {
		account.Properties["games_rated"] = profile.Count.Rated
	}
	if profile.PlayTime.Total > 0 {
		account.Properties["play_time_total"] = profile.PlayTime.Total
	}
	setPerfProperties(account.Properties, "bullet", profile.Perfs.Bullet)
	setPerfProperties(account.Properties, "blitz", profile.Perfs.Blitz)
	setPerfProperties(account.Properties, "rapid", profile.Perfs.Rapid)
	setPerfProperties(account.Properties, "classical", profile.Perfs.Classical)

	nodes = append(nodes, account)
	edges = append(edges, graph.NewEdge(0, node.ID, account.ID, graph.EdgeTypeHasAccount, "lichess"))

	fullName := strings.TrimSpace(strings.TrimSpace(profile.Profile.FirstName) + " " + strings.TrimSpace(profile.Profile.LastName))
	if fullName != "" {
		nameNode := graph.NewNode(graph.NodeTypeFullName, fullName, "lichess")
		nameNode.Confidence = 0.85
		nodes = append(nodes, nameNode)
		edges = append(edges, graph.NewEdge(0, account.ID, nameNode.ID, graph.EdgeTypeLinkedTo, "lichess"))
	}

	seenLinks := make(map[string]struct{})
	for _, rawLink := range strings.FieldsFunc(profile.Profile.Links, func(r rune) bool {
		return r == '\n' || r == '\r'
	}) {
		link := normalizeWebsite(rawLink)
		if link == "" {
			continue
		}
		if _, ok := seenLinks[link]; ok {
			continue
		}
		seenLinks[link] = struct{}{}

		websiteNode := graph.NewNode(graph.NodeTypeWebsite, link, "lichess")
		websiteNode.Confidence = 0.85
		websiteNode.Pivot = true
		nodes = append(nodes, websiteNode)
		edges = append(edges, graph.NewEdge(0, account.ID, websiteNode.ID, graph.EdgeTypeLinkedTo, "lichess"))
	}

	return nodes, edges, nil
}

func (m *Module) Verify(ctx context.Context, client *httpclient.Client) (modules.HealthStatus, string) {
	apiURL := fmt.Sprintf("%s/api/user/lichess", m.baseURL)

	resp, err := client.Do(ctx, apiURL, map[string]string{"Accept": "application/json"})
	if err != nil {
		return modules.Offline, fmt.Sprintf("lichess: %v", err)
	}
	if resp.StatusCode != 200 {
		return modules.Offline, fmt.Sprintf("lichess: status %d", resp.StatusCode)
	}

	var profile struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(resp.Body), &profile); err == nil && strings.EqualFold(profile.ID, "lichess") {
		return modules.Healthy, "lichess: OK"
	}
	return modules.Degraded, "lichess: unexpected response format"
}

type lichessPerf struct {
	Games  int `json:"games"`
	Rating int `json:"rating"`
}

func setPerfProperties(props map[string]interface{}, prefix string, perf lichessPerf) {
	if perf.Games > 0 {
		props[prefix+"_games"] = perf.Games
	}
	if perf.Rating > 0 && perf.Games > 0 {
		props[prefix+"_rating"] = perf.Rating
	}
}

func normalizeWebsite(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}
	if !strings.Contains(value, "://") {
		value = "https://" + value
	}

	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	return parsed.String()
}
