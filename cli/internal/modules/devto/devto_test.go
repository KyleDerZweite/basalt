// SPDX-License-Identifier: AGPL-3.0-or-later

package devto

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
	if m.CanHandle("domain") {
		t.Error("should not handle domain")
	}
}

func TestExtractFound(t *testing.T) {
	profile := map[string]interface{}{
		"username":         "testuser",
		"name":             "Test User",
		"summary":          "A developer",
		"joined_at":        "Jan 1, 2020",
		"profile_image":    "https://dev.to/avatar/testuser.png",
		"twitter_username": "testtwitter",
		"github_username":  "testgithub",
		"website_url":      "https://example.com",
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(profile)
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

	// Expect: account + github username + twitter username + website = 4 nodes
	if len(nodes) != 4 {
		t.Fatalf("expected 4 nodes, got %d", len(nodes))
	}
	// Expect: seed->account + account->github + account->twitter + account->website = 4 edges
	if len(edges) != 4 {
		t.Fatalf("expected 4 edges, got %d", len(edges))
	}

	// Verify account node.
	account := nodes[0]
	if account.Type != graph.NodeTypeAccount {
		t.Errorf("expected account node, got %s", account.Type)
	}
	if account.Confidence != 0.90 {
		t.Errorf("expected confidence 0.90, got %f", account.Confidence)
	}
	if account.Properties["name"] != "Test User" {
		t.Errorf("expected name 'Test User', got %v", account.Properties["name"])
	}
	if account.Properties["summary"] != "A developer" {
		t.Errorf("expected summary 'A developer', got %v", account.Properties["summary"])
	}
	if account.Properties["joined_at"] != "Jan 1, 2020" {
		t.Errorf("expected joined_at 'Jan 1, 2020', got %v", account.Properties["joined_at"])
	}
	if account.Properties["profile_image"] != "https://dev.to/avatar/testuser.png" {
		t.Errorf("expected profile_image URL, got %v", account.Properties["profile_image"])
	}

	// Verify GitHub username node.
	ghNode := nodes[1]
	if ghNode.Type != graph.NodeTypeUsername {
		t.Errorf("expected username node for github, got %s", ghNode.Type)
	}
	if ghNode.Label != "testgithub" {
		t.Errorf("expected label 'testgithub', got %s", ghNode.Label)
	}
	if !ghNode.Pivot {
		t.Error("github username node should be pivotable")
	}
	if ghNode.Confidence != 0.85 {
		t.Errorf("expected confidence 0.85, got %f", ghNode.Confidence)
	}

	// Verify Twitter username node.
	twNode := nodes[2]
	if twNode.Type != graph.NodeTypeUsername {
		t.Errorf("expected username node for twitter, got %s", twNode.Type)
	}
	if twNode.Label != "testtwitter" {
		t.Errorf("expected label 'testtwitter', got %s", twNode.Label)
	}
	if !twNode.Pivot {
		t.Error("twitter username node should be pivotable")
	}
	if twNode.Confidence != 0.85 {
		t.Errorf("expected confidence 0.85, got %f", twNode.Confidence)
	}

	// Verify website node.
	webNode := nodes[3]
	if webNode.Type != graph.NodeTypeWebsite {
		t.Errorf("expected website node, got %s", webNode.Type)
	}
	if webNode.Label != "https://example.com" {
		t.Errorf("expected label 'https://example.com', got %s", webNode.Label)
	}
	if webNode.Pivot {
		t.Error("website node should not be pivotable")
	}
	if webNode.Confidence != 0.80 {
		t.Errorf("expected confidence 0.80, got %f", webNode.Confidence)
	}

	// Verify edge types.
	if edges[0].Type != graph.EdgeTypeHasAccount {
		t.Errorf("expected has_account edge, got %s", edges[0].Type)
	}
	for _, e := range edges[1:] {
		if e.Type != graph.EdgeTypeLinkedTo {
			t.Errorf("expected linked_to edge, got %s", e.Type)
		}
	}
}

func TestExtractNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
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
	if len(nodes) != 0 {
		t.Errorf("expected no nodes for 404, got %d", len(nodes))
	}
	if len(edges) != 0 {
		t.Errorf("expected no edges for 404, got %d", len(edges))
	}
}

func TestVerifyHealthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"username": "ben",
			"name":     "Ben Halpern",
		})
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
