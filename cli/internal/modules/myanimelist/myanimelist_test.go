// SPDX-License-Identifier: AGPL-3.0-or-later

package myanimelist

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
	if m.CanHandle("domain") {
		t.Error("should not handle domain")
	}
}

func TestExtractFound(t *testing.T) {
	// Mock response MUST have the data wrapper
	payload := map[string]interface{}{
		"data": map[string]interface{}{
			"username": "testuser",
			"url":      "https://myanimelist.net/profile/testuser",
			"images": map[string]interface{}{
				"jpg": map[string]interface{}{
					"image_url": "https://cdn.myanimelist.net/images/useravatar/test.jpg",
				},
			},
			"last_online": "2024-04-01T12:00:00Z",
			"gender":      "Male",
			"birthday":    "1990-05-15",
			"location":    "United States",
			"joined":      "2010-03-20T00:00:00Z",
			"mal_id":      12345,
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

	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}
	if nodes[0].Type != graph.NodeTypeAccount {
		t.Errorf("expected account node, got %s", nodes[0].Type)
	}
	if nodes[0].Confidence != 0.90 {
		t.Errorf("expected confidence 0.90, got %f", nodes[0].Confidence)
	}
	if nodes[0].Properties["profile_url"] != "https://myanimelist.net/profile/testuser" {
		t.Errorf("unexpected profile_url: %v", nodes[0].Properties["profile_url"])
	}

	// Verify properties are stored
	if gender, ok := nodes[0].Properties["gender"].(string); !ok || gender != "Male" {
		t.Errorf("expected gender 'Male', got %v", nodes[0].Properties["gender"])
	}
	if birthday, ok := nodes[0].Properties["birthday"].(string); !ok || birthday != "1990-05-15" {
		t.Errorf("expected birthday '1990-05-15', got %v", nodes[0].Properties["birthday"])
	}
	if location, ok := nodes[0].Properties["location"].(string); !ok || location != "United States" {
		t.Errorf("expected location 'United States', got %v", nodes[0].Properties["location"])
	}
	if joined, ok := nodes[0].Properties["joined"].(string); !ok || joined != "2010-03-20T00:00:00Z" {
		t.Errorf("expected joined '2010-03-20T00:00:00Z', got %v", nodes[0].Properties["joined"])
	}
	if lastOnline, ok := nodes[0].Properties["last_online"].(string); !ok || lastOnline != "2024-04-01T12:00:00Z" {
		t.Errorf("expected last_online '2024-04-01T12:00:00Z', got %v", nodes[0].Properties["last_online"])
	}
	if malID, ok := nodes[0].Properties["mal_id"].(int); !ok || malID != 12345 {
		t.Errorf("expected mal_id 12345, got %v", nodes[0].Properties["mal_id"])
	}
	if avatar, ok := nodes[0].Properties["avatar"].(string); !ok || avatar != "https://cdn.myanimelist.net/images/useravatar/test.jpg" {
		t.Errorf("expected avatar URL, got %v", nodes[0].Properties["avatar"])
	}

	if len(edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(edges))
	}
	if edges[0].Type != graph.EdgeTypeHasAccount {
		t.Errorf("expected has_account edge, got %s", edges[0].Type)
	}
}

func TestExtractNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"error": "not found"}`))
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
		"data": map[string]interface{}{
			"username": "Nekomata1037",
			"url":      "https://myanimelist.net/profile/Nekomata1037",
			"images": map[string]interface{}{
				"jpg": map[string]interface{}{
					"image_url": "https://cdn.myanimelist.net/images/useravatar/admin.jpg",
				},
			},
			"gender":   "Male",
			"location": "Japan",
			"mal_id":   1,
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

func TestVerifyUnexpectedStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"error": "internal error"}`))
	}))
	defer srv.Close()

	m := New()
	m.baseURL = srv.URL

	client := httpclient.New()
	status, msg := m.Verify(context.Background(), client)
	if status != modules.Offline {
		t.Errorf("expected Offline, got %d: %s", status, msg)
	}
}
