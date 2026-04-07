// SPDX-License-Identifier: AGPL-3.0-or-later

package dockerhub

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/KyleDerZweite/basalt/internal/graph"
	"github.com/KyleDerZweite/basalt/internal/httpclient"
	"github.com/KyleDerZweite/basalt/internal/modules"
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
	profile := map[string]interface{}{
		"id":          "abc123",
		"username":    "testuser",
		"full_name":   "Test User",
		"location":    "Berlin",
		"company":     "ACME Corp",
		"date_joined": "2020-01-15T10:30:00Z",
		"profile_url": "https://hub.docker.com/u/testuser",
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

	// Expect 2 nodes: account + full_name
	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes (account + full_name), got %d", len(nodes))
	}
	// Expect 2 edges: seed->account + account->full_name
	if len(edges) != 2 {
		t.Fatalf("expected 2 edges, got %d", len(edges))
	}

	// Verify account node
	account := nodes[0]
	if account.Type != graph.NodeTypeAccount {
		t.Errorf("expected account node type, got %s", account.Type)
	}
	if account.Confidence != 0.90 {
		t.Errorf("expected confidence 0.90, got %f", account.Confidence)
	}
	if account.Properties["full_name"] != "Test User" {
		t.Errorf("expected full_name property 'Test User', got %v", account.Properties["full_name"])
	}
	if account.Properties["location"] != "Berlin" {
		t.Errorf("expected location property 'Berlin', got %v", account.Properties["location"])
	}
	if account.Properties["company"] != "ACME Corp" {
		t.Errorf("expected company property 'ACME Corp', got %v", account.Properties["company"])
	}
	if account.Properties["date_joined"] != "2020-01-15T10:30:00Z" {
		t.Errorf("expected date_joined property, got %v", account.Properties["date_joined"])
	}

	// Verify full_name node
	nameNode := nodes[1]
	if nameNode.Type != graph.NodeTypeFullName {
		t.Errorf("expected full_name node type, got %s", nameNode.Type)
	}
	if nameNode.Label != "Test User" {
		t.Errorf("expected label 'Test User', got %s", nameNode.Label)
	}
	if nameNode.Confidence != 0.70 {
		t.Errorf("expected confidence 0.70, got %f", nameNode.Confidence)
	}
	if nameNode.Pivot {
		t.Error("full_name node should not be pivotable")
	}

	// Verify edges
	if edges[0].Type != graph.EdgeTypeHasAccount {
		t.Errorf("expected has_account edge, got %s", edges[0].Type)
	}
	if edges[1].Type != graph.EdgeTypeLinkedTo {
		t.Errorf("expected linked_to edge, got %s", edges[1].Type)
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

func TestExtractInfersUsernameFromSingleTokenFullName(t *testing.T) {
	profile := map[string]interface{}{
		"id":          "abc123",
		"username":    "kylederzweite",
		"full_name":   "VispLP",
		"profile_url": "https://hub.docker.com/u/kylederzweite",
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(profile)
	}))
	defer srv.Close()

	m := New()
	m.baseURL = srv.URL

	node := graph.NewNode("username", "kylederzweite", "seed")
	client := httpclient.New()

	nodes, edges, err := m.Extract(context.Background(), node, client)
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 3 {
		t.Fatalf("expected 3 nodes (account + full_name + inferred username), got %d", len(nodes))
	}
	if len(edges) != 3 {
		t.Fatalf("expected 3 edges, got %d", len(edges))
	}

	inferred := nodes[2]
	if inferred.Type != graph.NodeTypeUsername {
		t.Fatalf("expected inferred username node, got %s", inferred.Type)
	}
	if inferred.Label != "VispLP" {
		t.Fatalf("expected inferred username VispLP, got %q", inferred.Label)
	}
	if !inferred.Pivot {
		t.Fatal("expected inferred username to be pivotable")
	}
	if inferred.Confidence != 0.65 {
		t.Fatalf("expected inferred confidence 0.65, got %f", inferred.Confidence)
	}
	if inferred.Properties["inferred_from"] != graph.NodeTypeFullName {
		t.Fatalf("expected inferred_from full_name, got %#v", inferred.Properties["inferred_from"])
	}
	if edges[2].Type != graph.EdgeTypeHasUsername {
		t.Fatalf("expected inferred edge type has_username, got %s", edges[2].Type)
	}
}

func TestVerifyHealthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"username": "library",
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
