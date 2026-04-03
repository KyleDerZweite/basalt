// SPDX-License-Identifier: AGPL-3.0-or-later

package codeforces

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

const codeforcesBaseURL = "https://codeforces.com"

// Module extracts profile data from Codeforces.
type Module struct {
	baseURL string
}

// New creates a Codeforces module.
func New() *Module {
	return &Module{baseURL: codeforcesBaseURL}
}

func (m *Module) Name() string        { return "codeforces" }
func (m *Module) Description() string { return "Extract profile data from Codeforces" }

func (m *Module) CanHandle(nodeType string) bool {
	return nodeType == "username"
}

func (m *Module) Extract(ctx context.Context, node *graph.Node, client *httpclient.Client) ([]*graph.Node, []*graph.Edge, error) {
	apiURL := fmt.Sprintf("%s/api/user.info?handles=%s", m.baseURL, url.QueryEscape(node.Label))

	resp, err := client.Do(ctx, apiURL, map[string]string{"Accept": "application/json"})
	if err != nil {
		return nil, nil, fmt.Errorf("codeforces request: %w", err)
	}
	if resp.StatusCode != 200 && resp.StatusCode != 400 {
		return nil, nil, fmt.Errorf("codeforces returned %d", resp.StatusCode)
	}

	var payload struct {
		Status  string `json:"status"`
		Comment string `json:"comment"`
		Result  []struct {
			Handle                  string `json:"handle"`
			FirstName               string `json:"firstName"`
			LastName                string `json:"lastName"`
			Country                 string `json:"country"`
			City                    string `json:"city"`
			Avatar                  string `json:"avatar"`
			TitlePhoto              string `json:"titlePhoto"`
			Organization            string `json:"organization"`
			Rank                    string `json:"rank"`
			MaxRank                 string `json:"maxRank"`
			Rating                  int    `json:"rating"`
			MaxRating               int    `json:"maxRating"`
			Contribution            int    `json:"contribution"`
			FriendOfCount           int    `json:"friendOfCount"`
			LastOnlineTimeSeconds   int64  `json:"lastOnlineTimeSeconds"`
			RegistrationTimeSeconds int64  `json:"registrationTimeSeconds"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(resp.Body), &payload); err != nil {
		return nil, nil, fmt.Errorf("parsing codeforces response: %w", err)
	}

	if resp.StatusCode == 400 && payload.Status == "FAILED" && strings.Contains(strings.ToLower(payload.Comment), "not found") {
		return nil, nil, nil
	}
	if resp.StatusCode != 200 {
		return nil, nil, fmt.Errorf("codeforces returned %d", resp.StatusCode)
	}
	if payload.Status == "FAILED" {
		if strings.Contains(strings.ToLower(payload.Comment), "not found") {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("codeforces API error: %s", payload.Comment)
	}
	if len(payload.Result) == 0 {
		return nil, nil, nil
	}

	user := payload.Result[0]
	if user.Handle == "" {
		return nil, nil, nil
	}

	var nodes []*graph.Node
	var edges []*graph.Edge

	account := graph.NewAccountNode("codeforces", user.Handle, fmt.Sprintf("%s/profile/%s", m.baseURL, url.PathEscape(user.Handle)), "codeforces")
	account.Confidence = 0.90
	if user.Country != "" {
		account.Properties["country"] = user.Country
	}
	if user.City != "" {
		account.Properties["city"] = user.City
	}
	if user.Rank != "" {
		account.Properties["rank"] = user.Rank
	}
	if user.MaxRank != "" {
		account.Properties["max_rank"] = user.MaxRank
	}
	if user.Rating > 0 {
		account.Properties["rating"] = user.Rating
	}
	if user.MaxRating > 0 {
		account.Properties["max_rating"] = user.MaxRating
	}
	if user.Contribution != 0 {
		account.Properties["contribution"] = user.Contribution
	}
	if user.FriendOfCount > 0 {
		account.Properties["friend_of_count"] = user.FriendOfCount
	}
	if user.LastOnlineTimeSeconds > 0 {
		account.Properties["last_online"] = user.LastOnlineTimeSeconds
	}
	if user.RegistrationTimeSeconds > 0 {
		account.Properties["registered_at"] = user.RegistrationTimeSeconds
	}
	nodes = append(nodes, account)
	edges = append(edges, graph.NewEdge(0, node.ID, account.ID, graph.EdgeTypeHasAccount, "codeforces"))

	fullName := strings.TrimSpace(strings.TrimSpace(user.FirstName) + " " + strings.TrimSpace(user.LastName))
	if fullName != "" {
		nameNode := graph.NewNode(graph.NodeTypeFullName, fullName, "codeforces")
		nameNode.Confidence = 0.85
		nodes = append(nodes, nameNode)
		edges = append(edges, graph.NewEdge(0, account.ID, nameNode.ID, graph.EdgeTypeLinkedTo, "codeforces"))
	}

	avatarURL := user.TitlePhoto
	if avatarURL == "" {
		avatarURL = user.Avatar
	}
	if avatarURL != "" {
		avatarNode := graph.NewNode(graph.NodeTypeAvatarURL, avatarURL, "codeforces")
		avatarNode.Confidence = 0.90
		nodes = append(nodes, avatarNode)
		edges = append(edges, graph.NewEdge(0, account.ID, avatarNode.ID, graph.EdgeTypeLinkedTo, "codeforces"))
	}

	if user.Organization != "" {
		orgNode := graph.NewNode(graph.NodeTypeOrganization, user.Organization, "codeforces")
		orgNode.Confidence = 0.80
		nodes = append(nodes, orgNode)
		edges = append(edges, graph.NewEdge(0, account.ID, orgNode.ID, graph.EdgeTypeLinkedTo, "codeforces"))
	}

	return nodes, edges, nil
}

func (m *Module) Verify(ctx context.Context, client *httpclient.Client) (modules.HealthStatus, string) {
	apiURL := fmt.Sprintf("%s/api/user.info?handles=tourist", m.baseURL)

	resp, err := client.Do(ctx, apiURL, map[string]string{"Accept": "application/json"})
	if err != nil {
		return modules.Offline, fmt.Sprintf("codeforces: %v", err)
	}
	if resp.StatusCode != 200 {
		return modules.Offline, fmt.Sprintf("codeforces: status %d", resp.StatusCode)
	}

	var payload struct {
		Status string `json:"status"`
		Result []struct {
			Handle string `json:"handle"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(resp.Body), &payload); err == nil &&
		payload.Status == "OK" &&
		len(payload.Result) > 0 &&
		payload.Result[0].Handle == "tourist" {
		return modules.Healthy, "codeforces: OK"
	}
	return modules.Degraded, "codeforces: unexpected response format"
}
