// SPDX-License-Identifier: AGPL-3.0-or-later

package dockerhub

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/kyle/basalt/internal/graph"
	"github.com/kyle/basalt/internal/httpclient"
	"github.com/kyle/basalt/internal/modules"
)

const defaultBaseURL = "https://hub.docker.com"

// Module extracts profile data from Docker Hub.
type Module struct {
	baseURL string
}

// New creates a Docker Hub module.
func New() *Module {
	return &Module{baseURL: defaultBaseURL}
}

func (m *Module) Name() string        { return "dockerhub" }
func (m *Module) Description() string { return "Extract profile data from Docker Hub" }

func (m *Module) CanHandle(nodeType string) bool {
	return nodeType == "username"
}

func (m *Module) Extract(ctx context.Context, node *graph.Node, client *httpclient.Client) ([]*graph.Node, []*graph.Edge, error) {
	username := node.Label
	apiURL := fmt.Sprintf("%s/v2/users/%s", m.baseURL, url.PathEscape(username))

	resp, err := client.Do(ctx, apiURL, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("dockerhub user request: %w", err)
	}
	if resp.StatusCode == 404 {
		return nil, nil, nil
	}
	if resp.StatusCode != 200 {
		return nil, nil, fmt.Errorf("dockerhub returned %d", resp.StatusCode)
	}

	var profile struct {
		ID         string `json:"id"`
		Username   string `json:"username"`
		FullName   string `json:"full_name"`
		Location   string `json:"location"`
		Company    string `json:"company"`
		DateJoined string `json:"date_joined"`
		ProfileURL string `json:"profile_url"`
	}
	if err := json.Unmarshal([]byte(resp.Body), &profile); err != nil {
		return nil, nil, fmt.Errorf("parsing dockerhub response: %w", err)
	}

	var nodes []*graph.Node
	var edges []*graph.Edge

	profileURL := fmt.Sprintf("https://hub.docker.com/u/%s", url.PathEscape(username))
	account := graph.NewAccountNode("dockerhub", username, profileURL, "dockerhub")
	account.Confidence = 0.90

	if profile.FullName != "" {
		account.Properties["full_name"] = profile.FullName
	}
	if profile.Location != "" {
		account.Properties["location"] = profile.Location
	}
	if profile.Company != "" {
		account.Properties["company"] = profile.Company
	}
	if profile.DateJoined != "" {
		account.Properties["date_joined"] = profile.DateJoined
	}

	nodes = append(nodes, account)
	edges = append(edges, graph.NewEdge(0, node.ID, account.ID, graph.EdgeTypeHasAccount, "dockerhub"))

	if profile.FullName != "" {
		nameNode := graph.NewNode(graph.NodeTypeFullName, profile.FullName, "dockerhub")
		nameNode.Confidence = 0.70
		nameNode.Pivot = false
		nodes = append(nodes, nameNode)
		edges = append(edges, graph.NewEdge(0, account.ID, nameNode.ID, graph.EdgeTypeLinkedTo, "dockerhub"))
	}

	return nodes, edges, nil
}

func (m *Module) Verify(ctx context.Context, client *httpclient.Client) (modules.HealthStatus, string) {
	apiURL := fmt.Sprintf("%s/v2/users/thajeztah", m.baseURL)
	resp, err := client.Do(ctx, apiURL, nil)
	if err != nil {
		return modules.Offline, fmt.Sprintf("dockerhub: %v", err)
	}
	if resp.StatusCode == 200 {
		var user struct {
			Username string `json:"username"`
			Orgname  string `json:"orgname"`
		}
		if err := json.Unmarshal([]byte(resp.Body), &user); err == nil && (user.Username != "" || user.Orgname != "") {
			return modules.Healthy, "dockerhub: OK"
		}
		return modules.Degraded, "dockerhub: unexpected response format"
	}
	return modules.Offline, fmt.Sprintf("dockerhub: status %d", resp.StatusCode)
}
