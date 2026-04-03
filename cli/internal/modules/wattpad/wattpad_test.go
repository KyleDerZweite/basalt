// SPDX-License-Identifier: AGPL-3.0-or-later

package wattpad

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
	user := map[string]interface{}{
		"username":            "bookworm123",
		"name":                "Sarah Johnson",
		"avatar":              "https://a.wattpad.com/useravatar/bookworm123.jpg",
		"description":         "Fantasy and romance lover",
		"location":            "New York, USA",
		"gender":              "female",
		"genderCode":          "F",
		"createDate":          1609459200,
		"verified":            true,
		"website":             "https://sarahjohnson.com",
		"facebook":            "sarah.johnson.123",
		"numFollowers":        1500,
		"numFollowing":        300,
		"numStoriesPublished": 5,
		"votesReceived":       10000,
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(user)
	}))
	defer srv.Close()

	m := New()
	m.baseURL = srv.URL

	node := graph.NewNode("username", "bookworm123", "seed")
	client := httpclient.New()

	nodes, edges, err := m.Extract(context.Background(), node, client)
	if err != nil {
		t.Fatal(err)
	}

	// Should have: account + website + facebook = 3 nodes
	if len(nodes) != 3 {
		t.Fatalf("expected 3 nodes (account + website + facebook), got %d", len(nodes))
	}

	// Should have: seed->account + account->website + account->facebook = 3 edges
	if len(edges) != 3 {
		t.Fatalf("expected 3 edges, got %d", len(edges))
	}

	// Verify account node
	var foundAccount bool
	var foundWebsite bool
	var foundFacebook bool

	for _, n := range nodes {
		if n.Type == graph.NodeTypeAccount {
			foundAccount = true
			if n.Label != "wattpad - bookworm123" {
				t.Errorf("account label should be 'wattpad - bookworm123', got '%s'", n.Label)
			}
			if n.Confidence != 0.90 {
				t.Errorf("account confidence should be 0.90, got %f", n.Confidence)
			}
			if n.Properties["name"] != "Sarah Johnson" {
				t.Errorf("expected name 'Sarah Johnson', got '%s'", n.Properties["name"])
			}
			if n.Properties["location"] != "New York, USA" {
				t.Errorf("expected location 'New York, USA', got '%s'", n.Properties["location"])
			}
			if n.Properties["verified"] != "true" {
				t.Errorf("expected verified 'true', got '%s'", n.Properties["verified"])
			}
			if n.Properties["numFollowers"] != "1500" {
				t.Errorf("expected numFollowers '1500', got '%s'", n.Properties["numFollowers"])
			}
		} else if n.Type == graph.NodeTypeWebsite {
			foundWebsite = true
			if n.Label != "https://sarahjohnson.com" {
				t.Errorf("website should be 'https://sarahjohnson.com', got '%s'", n.Label)
			}
			if !n.Pivot {
				t.Error("website node should be pivotable")
			}
			if n.Confidence != 0.85 {
				t.Errorf("website confidence should be 0.85, got %f", n.Confidence)
			}
		} else if n.Type == graph.NodeTypeUsername {
			foundFacebook = true
			if n.Label != "sarah.johnson.123" {
				t.Errorf("facebook username should be 'sarah.johnson.123', got '%s'", n.Label)
			}
			if !n.Pivot {
				t.Error("facebook username node should be pivotable")
			}
			if n.Confidence != 0.80 {
				t.Errorf("facebook username confidence should be 0.80, got %f", n.Confidence)
			}
			if n.Properties["platform_hint"] != "facebook" {
				t.Errorf("expected platform_hint 'facebook', got '%s'", n.Properties["platform_hint"])
			}
		}
	}

	if !foundAccount {
		t.Error("expected account node")
	}
	if !foundWebsite {
		t.Error("expected website node")
	}
	if !foundFacebook {
		t.Error("expected facebook username node")
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

	nodes, _, err := m.Extract(context.Background(), node, client)
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 0 {
		t.Errorf("expected no nodes for 404, got %d", len(nodes))
	}
}

func TestExtractNotFoundBadRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	m := New()
	m.baseURL = srv.URL

	node := graph.NewNode("username", "invalid", "seed")
	client := httpclient.New()

	nodes, _, err := m.Extract(context.Background(), node, client)
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 0 {
		t.Errorf("expected no nodes for 400, got %d", len(nodes))
	}
}

func TestExtractMinimalData(t *testing.T) {
	user := map[string]interface{}{
		"username":            "minimaluser",
		"name":                "",
		"avatar":              "",
		"description":         "",
		"location":            "",
		"gender":              "",
		"genderCode":          "",
		"createDate":          0,
		"verified":            false,
		"website":             "",
		"facebook":            "",
		"numFollowers":        0,
		"numFollowing":        0,
		"numStoriesPublished": 0,
		"votesReceived":       0,
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(user)
	}))
	defer srv.Close()

	m := New()
	m.baseURL = srv.URL

	node := graph.NewNode("username", "minimaluser", "seed")
	client := httpclient.New()

	nodes, edges, err := m.Extract(context.Background(), node, client)
	if err != nil {
		t.Fatal(err)
	}

	// Should only have account node (no website, no facebook)
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node (account only), got %d", len(nodes))
	}

	// Should only have seed->account edge
	if len(edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(edges))
	}

	if nodes[0].Type != graph.NodeTypeAccount {
		t.Errorf("expected account node, got %s", nodes[0].Type)
	}
}

func TestVerifyHealthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"username": "wattpad",
			"name":     "Wattpad",
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

func TestVerifyOffline(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
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

func TestVerifyDegraded(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Return wrong username
		json.NewEncoder(w).Encode(map[string]interface{}{
			"username": "wrong_user",
		})
	}))
	defer srv.Close()

	m := New()
	m.baseURL = srv.URL

	client := httpclient.New()
	status, msg := m.Verify(context.Background(), client)
	if status != modules.Degraded {
		t.Errorf("expected Degraded, got %d: %s", status, msg)
	}
}

func TestExtractFacebookUsername(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"sarah.johnson.123", "sarah.johnson.123"},
		{"https://facebook.com/sarah.johnson.123", "sarah.johnson.123"},
		{"https://facebook.com/sarah.johnson.123/", "sarah.johnson.123"},
		{"facebook.com/john.doe", "john.doe"},
		{"  jane.doe  ", "jane.doe"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := extractFacebookUsername(tt.input)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}
