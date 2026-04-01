// SPDX-License-Identifier: AGPL-3.0-or-later

package keybase

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kyle/basalt/internal/graph"
	"github.com/kyle/basalt/internal/httpclient"
	"github.com/kyle/basalt/internal/modules"
)

const keybaseURL = "https://keybase.io"

// Module extracts identity proofs and PGP keys from Keybase.
type Module struct {
	baseURL string
}

// New creates a Keybase module.
func New() *Module {
	return &Module{baseURL: keybaseURL}
}

func (m *Module) Name() string                  { return "keybase" }
func (m *Module) Description() string           { return "Extract identity proofs and PGP keys from Keybase" }
func (m *Module) CanHandle(nodeType string) bool { return nodeType == "username" }

func (m *Module) Extract(ctx context.Context, node *graph.Node, client *httpclient.Client) ([]*graph.Node, []*graph.Edge, error) {
	username := node.Label
	url := fmt.Sprintf("%s/_/api/1.0/user/lookup.json?usernames=%s", m.baseURL, username)

	resp, err := client.Do(ctx, url, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("keybase request: %w", err)
	}
	if resp.StatusCode != 200 {
		return nil, nil, fmt.Errorf("keybase returned %d", resp.StatusCode)
	}

	var payload struct {
		Them []json.RawMessage `json:"them"`
	}
	if err := json.Unmarshal([]byte(resp.Body), &payload); err != nil {
		return nil, nil, fmt.Errorf("parsing keybase response: %w", err)
	}

	if len(payload.Them) == 0 || string(payload.Them[0]) == "null" {
		return nil, nil, nil
	}

	var profile struct {
		Basics struct {
			Username string `json:"username"`
		} `json:"basics"`
		Profile struct {
			FullName string `json:"full_name"`
			Bio      string `json:"bio"`
			Location string `json:"location"`
		} `json:"profile"`
		ProofsSummary struct {
			All []struct {
				ProofType  string `json:"proof_type"`
				Nametag    string `json:"nametag"`
				ServiceURL string `json:"service_url"`
				HumanURL   string `json:"human_url"`
				State      int    `json:"state"`
			} `json:"all"`
		} `json:"proofs_summary"`
	}
	if err := json.Unmarshal(payload.Them[0], &profile); err != nil {
		return nil, nil, fmt.Errorf("parsing keybase profile: %w", err)
	}

	var nodes []*graph.Node
	var edges []*graph.Edge

	profileURL := fmt.Sprintf("%s/%s", m.baseURL, username)
	account := graph.NewAccountNode("keybase", username, profileURL, "keybase")
	account.Confidence = 0.95

	if profile.Profile.FullName != "" {
		account.Properties["full_name"] = profile.Profile.FullName
	}
	if profile.Profile.Bio != "" {
		account.Properties["bio"] = profile.Profile.Bio
	}
	if profile.Profile.Location != "" {
		account.Properties["location"] = profile.Profile.Location
	}

	nodes = append(nodes, account)
	edges = append(edges, graph.NewEdge(0, node.ID, account.ID, graph.EdgeTypeHasAccount, "keybase"))

	edgeID := 1
	for _, proof := range profile.ProofsSummary.All {
		if proof.State != 1 {
			continue
		}

		var proofNode *graph.Node

		switch proof.ProofType {
		case "twitter", "github", "reddit", "hackernews", "mastodon":
			proofNode = graph.NewNode(graph.NodeTypeUsername, proof.Nametag, "keybase")
			proofNode.Pivot = true
			proofNode.Confidence = 0.90
		case "generic_web_site":
			proofNode = graph.NewNode(graph.NodeTypeWebsite, proof.Nametag, "keybase")
			proofNode.Confidence = 0.90
		case "dns":
			proofNode = graph.NewNode(graph.NodeTypeDomain, proof.Nametag, "keybase")
			proofNode.Confidence = 0.90
		default:
			continue
		}

		proofNode.Properties["proof_type"] = proof.ProofType
		if proof.HumanURL != "" {
			proofNode.Properties["human_url"] = proof.HumanURL
		}
		if proof.ServiceURL != "" {
			proofNode.Properties["service_url"] = proof.ServiceURL
		}

		nodes = append(nodes, proofNode)
		edges = append(edges, graph.NewEdge(edgeID, account.ID, proofNode.ID, graph.EdgeTypeLinkedTo, "keybase"))
		edgeID++
	}

	return nodes, edges, nil
}

func (m *Module) Verify(ctx context.Context, client *httpclient.Client) (modules.HealthStatus, string) {
	url := fmt.Sprintf("%s/_/api/1.0/user/lookup.json?usernames=max", m.baseURL)
	resp, err := client.Do(ctx, url, nil)
	if err != nil {
		return modules.Offline, fmt.Sprintf("keybase: %v", err)
	}
	if resp.StatusCode != 200 {
		return modules.Degraded, fmt.Sprintf("keybase: unexpected status %d", resp.StatusCode)
	}

	var payload struct {
		Them []json.RawMessage `json:"them"`
	}
	if err := json.Unmarshal([]byte(resp.Body), &payload); err != nil {
		return modules.Degraded, "keybase: unexpected response format"
	}
	if len(payload.Them) == 0 || string(payload.Them[0]) == "null" {
		return modules.Degraded, "keybase: user 'max' not found"
	}

	var profile struct {
		Basics struct {
			Username string `json:"username"`
		} `json:"basics"`
	}
	if err := json.Unmarshal(payload.Them[0], &profile); err == nil && profile.Basics.Username == "max" {
		return modules.Healthy, "keybase: OK"
	}
	return modules.Degraded, "keybase: unexpected response format"
}
