// SPDX-License-Identifier: AGPL-3.0-or-later

package roblox

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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
	if m.CanHandle("domain") {
		t.Error("should not handle domain")
	}
}

func TestExtractFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == "POST" && r.URL.Path == "/v1/usernames/users":
			// Username lookup response
			response := map[string]interface{}{
				"data": []map[string]interface{}{
					{
						"requestedUsername": "testuser",
						"hasVerifiedBadge":  false,
						"id":                123456,
						"name":              "testuser",
						"displayName":       "Test User",
					},
				},
			}
			json.NewEncoder(w).Encode(response)

		case r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/v1/users/"):
			// Profile response
			response := map[string]interface{}{
				"description":            "This is a test profile",
				"created":                "2019-01-01T00:00:00Z",
				"isBanned":               false,
				"externalAppDisplayName": nil,
				"hasVerifiedBadge":       false,
				"id":                     123456,
				"name":                   "testuser",
				"displayName":            "Test User",
			}
			json.NewEncoder(w).Encode(response)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	m := New()
	m.usersBaseURL = srv.URL

	node := graph.NewNode("username", "testuser", "seed")
	client := httpclient.New()

	nodes, edges, err := m.Extract(context.Background(), node, client)
	if err != nil {
		t.Fatal(err)
	}

	if len(nodes) == 0 {
		t.Fatal("expected at least one node")
	}

	// Check that we got an account node
	var foundAccount bool
	for _, n := range nodes {
		if n.Type == graph.NodeTypeAccount {
			foundAccount = true
			if n.Confidence < 0.85 {
				t.Errorf("expected high confidence for found profile, got %f", n.Confidence)
			}
			// Verify properties are stored
			if n.Properties["display_name"] != "Test User" {
				t.Errorf("expected display_name property, got %v", n.Properties["display_name"])
			}
			if n.Properties["description"] != "This is a test profile" {
				t.Errorf("expected description property, got %v", n.Properties["description"])
			}
		}
	}
	if !foundAccount {
		t.Error("expected an account node")
	}

	if len(edges) == 0 {
		t.Error("expected at least one edge")
	}

	// Verify edge from seed to account
	var foundEdge bool
	for _, e := range edges {
		if e.Type == graph.EdgeTypeHasAccount {
			foundEdge = true
			break
		}
	}
	if !foundEdge {
		t.Error("expected EdgeTypeHasAccount edge")
	}
}

func TestExtractNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == "POST" && r.URL.Path == "/v1/usernames/users":
			// Return empty data array (user not found)
			response := map[string]interface{}{
				"data": []interface{}{},
			}
			json.NewEncoder(w).Encode(response)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	m := New()
	m.usersBaseURL = srv.URL

	node := graph.NewNode("username", "nonexistentuser", "seed")
	client := httpclient.New()

	nodes, edges, err := m.Extract(context.Background(), node, client)
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 0 {
		t.Errorf("expected no nodes for user not found, got %d", len(nodes))
	}
	if len(edges) != 0 {
		t.Errorf("expected no edges for user not found, got %d", len(edges))
	}
}

func TestVerifyHealthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == "POST" && r.URL.Path == "/v1/usernames/users":
			// Official roblox account exists
			response := map[string]interface{}{
				"data": []map[string]interface{}{
					{
						"requestedUsername": "roblox",
						"hasVerifiedBadge":  true,
						"id":                1,
						"name":              "roblox",
						"displayName":       "Roblox",
					},
				},
			}
			json.NewEncoder(w).Encode(response)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	m := New()
	m.usersBaseURL = srv.URL

	client := httpclient.New()
	status, msg := m.Verify(context.Background(), client)
	if status != modules.Healthy {
		t.Errorf("expected Healthy, got %d: %s", status, msg)
	}
}

func TestVerifyOffline(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	m := New()
	m.usersBaseURL = srv.URL

	client := httpclient.New()
	status, msg := m.Verify(context.Background(), client)
	if status != modules.Offline {
		t.Errorf("expected Offline, got %d: %s", status, msg)
	}
}
