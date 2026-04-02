// SPDX-License-Identifier: AGPL-3.0-or-later

package chesscom

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
		"username":    "testplayer",
		"name":        "Test Player",
		"avatar":      "https://www.chess.com/avatar/testplayer.png",
		"url":         "https://www.chess.com/member/testplayer",
		"country":     "/country/US",
		"location":    "New York, USA",
		"title":       "GM",
		"followers":   1500,
		"joined":      1609459200,
		"last_online": 1704067200,
		"status":      "premium",
		"is_streamer": true,
		"twitch_url":  "https://twitch.tv/testplayer",
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(profile)
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

	// Expect: account + twitch website = 2 nodes
	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(nodes))
	}
	// Expect: seed->account + account->twitch = 2 edges
	if len(edges) != 2 {
		t.Fatalf("expected 2 edges, got %d", len(edges))
	}

	// Verify account node.
	account := nodes[0]
	if account.Type != graph.NodeTypeAccount {
		t.Errorf("expected account node, got %s", account.Type)
	}
	if account.Confidence != 0.90 {
		t.Errorf("expected confidence 0.90, got %f", account.Confidence)
	}
	if account.Properties["name"] != "Test Player" {
		t.Errorf("expected name 'Test Player', got %v", account.Properties["name"])
	}
	if account.Properties["avatar"] != "https://www.chess.com/avatar/testplayer.png" {
		t.Errorf("expected avatar URL, got %v", account.Properties["avatar"])
	}
	if account.Properties["country"] != "/country/US" {
		t.Errorf("expected country '/country/US', got %v", account.Properties["country"])
	}
	if account.Properties["location"] != "New York, USA" {
		t.Errorf("expected location 'New York, USA', got %v", account.Properties["location"])
	}
	if account.Properties["title"] != "GM" {
		t.Errorf("expected title 'GM', got %v", account.Properties["title"])
	}
	if account.Properties["followers"] != 1500 {
		t.Errorf("expected followers 1500, got %v", account.Properties["followers"])
	}
	if account.Properties["joined"] != int64(1609459200) {
		t.Errorf("expected joined 1609459200, got %v", account.Properties["joined"])
	}
	if account.Properties["last_online"] != int64(1704067200) {
		t.Errorf("expected last_online 1704067200, got %v", account.Properties["last_online"])
	}
	if account.Properties["status"] != "premium" {
		t.Errorf("expected status 'premium', got %v", account.Properties["status"])
	}
	if account.Properties["is_streamer"] != true {
		t.Errorf("expected is_streamer true, got %v", account.Properties["is_streamer"])
	}

	// Verify Twitch website node.
	twitchNode := nodes[1]
	if twitchNode.Type != graph.NodeTypeWebsite {
		t.Errorf("expected website node, got %s", twitchNode.Type)
	}
	if twitchNode.Label != "https://twitch.tv/testplayer" {
		t.Errorf("expected label 'https://twitch.tv/testplayer', got %s", twitchNode.Label)
	}
	if twitchNode.Pivot {
		t.Error("twitch website node should not be pivotable")
	}
	if twitchNode.Confidence != 0.85 {
		t.Errorf("expected confidence 0.85, got %f", twitchNode.Confidence)
	}

	// Verify edge types.
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

func TestExtractWithoutTwitchURL(t *testing.T) {
	profile := map[string]interface{}{
		"username":    "basicplayer",
		"name":        "Basic Player",
		"avatar":      "https://www.chess.com/avatar/basicplayer.png",
		"url":         "https://www.chess.com/member/basicplayer",
		"country":     "/country/GB",
		"location":    "London, UK",
		"title":       "",
		"followers":   100,
		"joined":      1609459200,
		"last_online": 1704067200,
		"status":      "free",
		"is_streamer": false,
		"twitch_url":  "",
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(profile)
	}))
	defer srv.Close()

	m := New()
	m.baseURL = srv.URL

	node := graph.NewNode("username", "basicplayer", "seed")
	client := httpclient.New()

	nodes, edges, err := m.Extract(context.Background(), node, client)
	if err != nil {
		t.Fatal(err)
	}

	// Expect: account only (no twitch URL) = 1 node
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}
	// Expect: seed->account = 1 edge
	if len(edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(edges))
	}

	// Verify account node.
	account := nodes[0]
	if account.Type != graph.NodeTypeAccount {
		t.Errorf("expected account node, got %s", account.Type)
	}
	if account.Properties["name"] != "Basic Player" {
		t.Errorf("expected name 'Basic Player', got %v", account.Properties["name"])
	}
}

func TestVerifyHealthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"username": "hikaru",
			"name":     "Hikaru Nakamura",
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
