// SPDX-License-Identifier: AGPL-3.0-or-later

package keybase

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kyle/basalt/internal/graph"
	"github.com/kyle/basalt/internal/httpclient"
	"github.com/kyle/basalt/internal/modules"
)

func TestCanHandle(t *testing.T) {
	m := New()
	if !m.CanHandle("username") {
		t.Error("should handle username")
	}
	if m.CanHandle("email") {
		t.Error("should not handle email")
	}
}

func TestExtractFound(t *testing.T) {
	payload := map[string]interface{}{
		"them": []interface{}{
			map[string]interface{}{
				"basics": map[string]interface{}{
					"username": "testuser",
				},
				"profile": map[string]interface{}{
					"full_name": "Test User",
					"bio":       "I am a test user",
					"location":  "Internet",
				},
				"proofs_summary": map[string]interface{}{
					"all": []interface{}{
						map[string]interface{}{
							"proof_type":  "twitter",
							"nametag":     "testuser_tw",
							"service_url": "https://twitter.com/testuser_tw",
							"human_url":   "https://twitter.com/testuser_tw",
							"state":       1,
						},
						map[string]interface{}{
							"proof_type":  "github",
							"nametag":     "testuser_gh",
							"service_url": "https://github.com/testuser_gh",
							"human_url":   "https://github.com/testuser_gh",
							"state":       1,
						},
						map[string]interface{}{
							"proof_type":  "generic_web_site",
							"nametag":     "testuser.com",
							"service_url": "https://testuser.com/.well-known/keybase.txt",
							"human_url":   "https://testuser.com",
							"state":       1,
						},
						map[string]interface{}{
							"proof_type":  "dns",
							"nametag":     "example.org",
							"service_url": "",
							"human_url":   "http://example.org",
							"state":       1,
						},
						map[string]interface{}{
							"proof_type":  "twitter",
							"nametag":     "unverified_tw",
							"service_url": "https://twitter.com/unverified_tw",
							"human_url":   "https://twitter.com/unverified_tw",
							"state":       0,
						},
					},
				},
			},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(payload)
	}))
	defer srv.Close()

	m := New()
	m.baseURL = srv.URL

	node := graph.NewNode("username", "testuser", "seed")
	client := httpclient.New()

	nodes, edges, err := m.Extract(context.Background(), node, client)
	if err != nil {
		t.Fatal(err)
	}

	// Expect: 1 account + 2 username (twitter, github) + 1 website + 1 domain = 5 nodes
	// The unverified twitter proof (state=0) should be excluded.
	if len(nodes) != 5 {
		t.Fatalf("expected 5 nodes, got %d", len(nodes))
	}

	// First node is the account node.
	account := nodes[0]
	if account.Type != graph.NodeTypeAccount {
		t.Errorf("expected account node, got %s", account.Type)
	}
	if account.Confidence != 0.95 {
		t.Errorf("expected confidence 0.95, got %f", account.Confidence)
	}
	if account.Properties["full_name"] != "Test User" {
		t.Errorf("expected full_name 'Test User', got %v", account.Properties["full_name"])
	}
	if account.Properties["bio"] != "I am a test user" {
		t.Errorf("expected bio 'I am a test user', got %v", account.Properties["bio"])
	}
	if account.Properties["location"] != "Internet" {
		t.Errorf("expected location 'Internet', got %v", account.Properties["location"])
	}

	// Twitter proof node.
	twitter := nodes[1]
	if twitter.Type != graph.NodeTypeUsername {
		t.Errorf("expected username node for twitter, got %s", twitter.Type)
	}
	if twitter.Label != "testuser_tw" {
		t.Errorf("expected label 'testuser_tw', got %s", twitter.Label)
	}
	if !twitter.Pivot {
		t.Error("expected twitter proof to be pivotable")
	}
	if twitter.Confidence != 0.90 {
		t.Errorf("expected confidence 0.90, got %f", twitter.Confidence)
	}

	// GitHub proof node.
	gh := nodes[2]
	if gh.Type != graph.NodeTypeUsername {
		t.Errorf("expected username node for github, got %s", gh.Type)
	}
	if gh.Label != "testuser_gh" {
		t.Errorf("expected label 'testuser_gh', got %s", gh.Label)
	}
	if !gh.Pivot {
		t.Error("expected github proof to be pivotable")
	}

	// Website proof node.
	website := nodes[3]
	if website.Type != graph.NodeTypeWebsite {
		t.Errorf("expected website node, got %s", website.Type)
	}
	if website.Label != "testuser.com" {
		t.Errorf("expected label 'testuser.com', got %s", website.Label)
	}
	if website.Pivot {
		t.Error("expected website proof not to be pivotable")
	}

	// DNS proof node.
	dns := nodes[4]
	if dns.Type != graph.NodeTypeDomain {
		t.Errorf("expected domain node, got %s", dns.Type)
	}
	if dns.Label != "example.org" {
		t.Errorf("expected label 'example.org', got %s", dns.Label)
	}
	if dns.Pivot {
		t.Error("expected dns proof not to be pivotable")
	}

	// Edges: 1 has_account + 4 linked_to = 5 edges.
	if len(edges) != 5 {
		t.Fatalf("expected 5 edges, got %d", len(edges))
	}
	if edges[0].Type != graph.EdgeTypeHasAccount {
		t.Errorf("expected has_account edge, got %s", edges[0].Type)
	}
	for i := 1; i < 5; i++ {
		if edges[i].Type != graph.EdgeTypeLinkedTo {
			t.Errorf("expected linked_to edge at index %d, got %s", i, edges[i].Type)
		}
	}
}

func TestExtractNotFound(t *testing.T) {
	payload := map[string]interface{}{
		"them": []interface{}{nil},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(payload)
	}))
	defer srv.Close()

	m := New()
	m.baseURL = srv.URL

	node := graph.NewNode("username", "nonexistent", "seed")
	client := httpclient.New()

	nodes, edges, err := m.Extract(context.Background(), node, client)
	if err != nil {
		t.Fatal(err)
	}
	if nodes != nil {
		t.Errorf("expected nil nodes for not found, got %d", len(nodes))
	}
	if edges != nil {
		t.Errorf("expected nil edges for not found, got %d", len(edges))
	}
}

func TestVerifyHealthy(t *testing.T) {
	payload := map[string]interface{}{
		"them": []interface{}{
			map[string]interface{}{
				"basics": map[string]interface{}{
					"username": "max",
				},
			},
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(payload)
	}))
	defer srv.Close()

	m := New()
	m.baseURL = srv.URL

	client := httpclient.New()
	status, msg := m.Verify(context.Background(), client)
	if status != modules.Healthy {
		t.Errorf("expected Healthy, got %d: %s", status, msg)
	}
}
