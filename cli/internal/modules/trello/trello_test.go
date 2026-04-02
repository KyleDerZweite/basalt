// SPDX-License-Identifier: AGPL-3.0-or-later

package trello

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
	payload := map[string]interface{}{
		"id":         "5f1a2b3c4d5e6f7g8h9i0j1k",
		"username":   "testuser",
		"fullName":   "Test User",
		"avatarUrl":  "https://trello-members.s3.amazonaws.com/5f1a2b3c4d5e6f7g8h9i0j1k/abc123/170.png",
		"bio":        "I love organizing tasks",
		"url":        "https://trello.com/testuser",
		"memberType": "normal",
		"confirmed":  true,
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

	// Should have account node + fullName node = 2 nodes
	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(nodes))
	}

	// Check account node
	accountNode := nodes[0]
	if accountNode.Type != graph.NodeTypeAccount {
		t.Errorf("expected account node, got %s", accountNode.Type)
	}
	if accountNode.Confidence != 0.90 {
		t.Errorf("expected confidence 0.90, got %f", accountNode.Confidence)
	}
	if accountNode.Properties["profile_url"] != "https://trello.com/testuser" {
		t.Errorf("unexpected profile_url: %v", accountNode.Properties["profile_url"])
	}
	if accountNode.Properties["id"] != "5f1a2b3c4d5e6f7g8h9i0j1k" {
		t.Errorf("expected id 5f1a2b3c4d5e6f7g8h9i0j1k, got %v", accountNode.Properties["id"])
	}
	if accountNode.Properties["full_name"] != "Test User" {
		t.Errorf("expected full_name Test User, got %v", accountNode.Properties["full_name"])
	}
	if accountNode.Properties["avatar_url"] != "https://trello-members.s3.amazonaws.com/5f1a2b3c4d5e6f7g8h9i0j1k/abc123/170.png" {
		t.Errorf("expected avatar_url, got %v", accountNode.Properties["avatar_url"])
	}
	if accountNode.Properties["bio"] != "I love organizing tasks" {
		t.Errorf("expected bio, got %v", accountNode.Properties["bio"])
	}
	if accountNode.Properties["url"] != "https://trello.com/testuser" {
		t.Errorf("expected url, got %v", accountNode.Properties["url"])
	}
	if accountNode.Properties["member_type"] != "normal" {
		t.Errorf("expected member_type normal, got %v", accountNode.Properties["member_type"])
	}
	if confirmed, ok := accountNode.Properties["confirmed"].(bool); !ok || !confirmed {
		t.Errorf("expected confirmed true, got %v", accountNode.Properties["confirmed"])
	}

	// Check fullName node
	fullNameNode := nodes[1]
	if fullNameNode.Type != graph.NodeTypeFullName {
		t.Errorf("expected full_name node, got %s", fullNameNode.Type)
	}
	if fullNameNode.Label != "Test User" {
		t.Errorf("expected label Test User, got %s", fullNameNode.Label)
	}

	// Should have 2 edges: seed->account and account->fullName
	if len(edges) != 2 {
		t.Fatalf("expected 2 edges, got %d", len(edges))
	}
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
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"message": "model not found"}`))
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
		t.Errorf("expected no nodes for 404 response, got %d", len(nodes))
	}
	if len(edges) != 0 {
		t.Errorf("expected no edges for 404 response, got %d", len(edges))
	}
}

func TestVerifyHealthy(t *testing.T) {
	payload := map[string]interface{}{
		"id":         "4d4fe8e9d6bb0c2d6dbf0c90",
		"username":   "trello",
		"fullName":   "Trello",
		"avatarUrl":  "https://trello-members.s3.amazonaws.com/4d4fe8e9d6bb0c2d6dbf0c90/abc/170.png",
		"bio":        "Official Trello account",
		"url":        "https://trello.com/trello",
		"memberType": "normal",
		"confirmed":  true,
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
