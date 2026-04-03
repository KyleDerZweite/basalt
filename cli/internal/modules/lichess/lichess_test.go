// SPDX-License-Identifier: AGPL-3.0-or-later

package lichess

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
	payload := map[string]interface{}{
		"id":        "testplayer",
		"username":  "TestPlayer",
		"url":       "https://lichess.org/@/TestPlayer",
		"verified":  true,
		"patron":    true,
		"flair":     "activity.lichess",
		"createdAt": int64(1700000000000),
		"seenAt":    int64(1700001000000),
		"count": map[string]interface{}{
			"all":   200,
			"rated": 150,
		},
		"playTime": map[string]interface{}{
			"total": 3600,
			"tv":    60,
		},
		"perfs": map[string]interface{}{
			"bullet":    map[string]interface{}{"games": 50, "rating": 1800},
			"blitz":     map[string]interface{}{"games": 100, "rating": 1900},
			"rapid":     map[string]interface{}{"games": 25, "rating": 2000},
			"classical": map[string]interface{}{"games": 10, "rating": 2100},
		},
		"profile": map[string]interface{}{
			"firstName": "Test",
			"lastName":  "Player",
			"bio":       "Chess player",
			"location":  "Berlin",
			"links":     "https://example.com\n twitch.tv/testplayer \nhttps://example.com",
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(payload)
	}))
	defer srv.Close()

	m := New()
	m.baseURL = srv.URL

	node := graph.NewNode("username", "testplayer", "seed")
	client := httpclient.New()

	nodes, edges, err := m.Extract(context.Background(), node, client)
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 4 {
		t.Fatalf("expected 4 nodes, got %d", len(nodes))
	}
	if len(edges) != 4 {
		t.Fatalf("expected 4 edges, got %d", len(edges))
	}

	account := nodes[0]
	if account.Type != graph.NodeTypeAccount {
		t.Fatalf("expected account node, got %s", account.Type)
	}
	if account.Properties["location"] != "Berlin" {
		t.Errorf("expected location Berlin, got %v", account.Properties["location"])
	}
	if account.Properties["bio"] != "Chess player" {
		t.Errorf("expected bio, got %v", account.Properties["bio"])
	}
	if account.Properties["verified"] != true {
		t.Errorf("expected verified true, got %v", account.Properties["verified"])
	}
	if account.Properties["blitz_rating"] != 1900 {
		t.Errorf("expected blitz rating 1900, got %v", account.Properties["blitz_rating"])
	}

	if nodes[1].Type != graph.NodeTypeFullName || nodes[1].Label != "Test Player" {
		t.Errorf("expected full name node, got %s %q", nodes[1].Type, nodes[1].Label)
	}

	if nodes[2].Type != graph.NodeTypeWebsite || nodes[2].Label != "https://example.com" {
		t.Errorf("expected website node https://example.com, got %s %q", nodes[2].Type, nodes[2].Label)
	}
	if !nodes[2].Pivot {
		t.Error("expected website node to be pivotable")
	}

	if nodes[3].Type != graph.NodeTypeWebsite || nodes[3].Label != "https://twitch.tv/testplayer" {
		t.Errorf("expected normalized twitch website, got %s %q", nodes[3].Type, nodes[3].Label)
	}
}

func TestExtractNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	m := New()
	m.baseURL = srv.URL

	node := graph.NewNode("username", "missing", "seed")
	client := httpclient.New()

	nodes, edges, err := m.Extract(context.Background(), node, client)
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 0 || len(edges) != 0 {
		t.Errorf("expected no results for 404, got %d nodes and %d edges", len(nodes), len(edges))
	}
}

func TestVerifyHealthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"id": "lichess"})
	}))
	defer srv.Close()

	m := New()
	m.baseURL = srv.URL

	client := httpclient.New()
	status, msg := m.Verify(context.Background(), client)
	if status != modules.Healthy {
		t.Fatalf("expected Healthy, got %d: %s", status, msg)
	}
}
