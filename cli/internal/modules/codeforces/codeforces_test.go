// SPDX-License-Identifier: AGPL-3.0-or-later

package codeforces

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
	if m.CanHandle("domain") {
		t.Error("should not handle domain")
	}
}

func TestExtractFound(t *testing.T) {
	payload := map[string]interface{}{
		"status": "OK",
		"result": []map[string]interface{}{
			{
				"handle":                  "tourist",
				"firstName":               "Gennady",
				"lastName":                "Korotkevich",
				"country":                 "Belarus",
				"city":                    "Gomel",
				"avatar":                  "https://example.com/avatar.jpg",
				"titlePhoto":              "https://example.com/title.jpg",
				"organization":            "ITMO University",
				"rank":                    "legendary grandmaster",
				"maxRank":                 "tourist",
				"rating":                  3755,
				"maxRating":               4009,
				"contribution":            75,
				"friendOfCount":           88003,
				"lastOnlineTimeSeconds":   int64(1700000000),
				"registrationTimeSeconds": int64(1200000000),
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

	node := graph.NewNode("username", "tourist", "seed")
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
	if account.Properties["rating"] != 3755 {
		t.Errorf("expected rating 3755, got %v", account.Properties["rating"])
	}
	if account.Properties["rank"] != "legendary grandmaster" {
		t.Errorf("expected rank, got %v", account.Properties["rank"])
	}

	if nodes[1].Type != graph.NodeTypeFullName || nodes[1].Label != "Gennady Korotkevich" {
		t.Errorf("expected full name node, got %s %q", nodes[1].Type, nodes[1].Label)
	}
	if nodes[2].Type != graph.NodeTypeAvatarURL || nodes[2].Label != "https://example.com/title.jpg" {
		t.Errorf("expected avatar node, got %s %q", nodes[2].Type, nodes[2].Label)
	}
	if nodes[3].Type != graph.NodeTypeOrganization || nodes[3].Label != "ITMO University" {
		t.Errorf("expected organization node, got %s %q", nodes[3].Type, nodes[3].Label)
	}
}

func TestExtractNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "FAILED",
			"comment": "handles: User with handle missing not found",
		})
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
		t.Errorf("expected no results for missing user, got %d nodes and %d edges", len(nodes), len(edges))
	}
}

func TestVerifyHealthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "OK",
			"result": []map[string]string{
				{"handle": "tourist"},
			},
		})
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
