// SPDX-License-Identifier: AGPL-3.0-or-later

package github

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
	m := New("")
	if !m.CanHandle("username") {
		t.Error("should handle username")
	}
	if !m.CanHandle("email") {
		t.Error("should handle email")
	}
	if m.CanHandle("domain") {
		t.Error("should not handle domain")
	}
}

func TestExtractUsername(t *testing.T) {
	user := map[string]interface{}{
		"login":            "kylederzweite",
		"name":             "Kyle",
		"email":            "kyle@kylehub.dev",
		"blog":             "https://kylehub.dev",
		"company":          "ACME",
		"location":         "Germany",
		"bio":              "Developer",
		"html_url":         "https://github.com/kylederzweite",
		"avatar_url":       "https://avatars.githubusercontent.com/u/123",
		"twitter_username": "kyletweets",
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(user)
	}))
	defer srv.Close()

	m := New("")
	m.baseURL = srv.URL

	node := graph.NewNode("username", "kylederzweite", "seed")
	client := httpclient.New()

	nodes, edges, err := m.Extract(context.Background(), node, client)
	if err != nil {
		t.Fatal(err)
	}

	if len(nodes) < 3 {
		t.Fatalf("expected at least 3 nodes (account + email + domain), got %d", len(nodes))
	}
	if len(edges) < 3 {
		t.Fatalf("expected at least 3 edges, got %d", len(edges))
	}

	var foundAccount bool
	for _, n := range nodes {
		if n.Type == graph.NodeTypeAccount {
			foundAccount = true
			if n.Confidence < 0.9 {
				t.Errorf("expected high confidence, got %f", n.Confidence)
			}
		}
	}
	if !foundAccount {
		t.Error("expected account node")
	}
}

func TestExtractNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	m := New("")
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

func TestExtractEmail(t *testing.T) {
	result := map[string]interface{}{
		"total_count": 1,
		"items": []interface{}{
			map[string]interface{}{
				"login":    "kylederzweite",
				"html_url": "https://github.com/kylederzweite",
			},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}))
	defer srv.Close()

	m := New("")
	m.baseURL = srv.URL

	node := graph.NewNode("email", "kyle@example.com", "seed")
	client := httpclient.New()

	nodes, _, err := m.Extract(context.Background(), node, client)
	if err != nil {
		t.Fatal(err)
	}

	var foundUsername bool
	for _, n := range nodes {
		if n.Type == graph.NodeTypeUsername && n.Label == "kylederzweite" {
			foundUsername = true
			if !n.Pivot {
				t.Error("discovered username should be pivotable")
			}
		}
	}
	if !foundUsername {
		t.Error("expected discovered username node")
	}
}

func TestVerifyHealthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"login": "octocat",
			"name":  "The Octocat",
		})
	}))
	defer srv.Close()

	m := New("")
	m.baseURL = srv.URL

	client := httpclient.New()
	status, msg := m.Verify(context.Background(), client)
	if status != modules.Healthy {
		t.Errorf("expected Healthy, got %d: %s", status, msg)
	}
}

func TestExtractFriends(t *testing.T) {
	user := map[string]interface{}{
		"login":    "kylederzweite",
		"html_url": "https://github.com/kylederzweite",
	}
	followers := []interface{}{
		map[string]interface{}{"login": "friendone", "html_url": "https://github.com/friendone"},
	}
	following := []interface{}{
		map[string]interface{}{"login": "friendtwo", "html_url": "https://github.com/friendtwo"},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/followers"):
			json.NewEncoder(w).Encode(followers)
		case strings.HasSuffix(r.URL.Path, "/following"):
			json.NewEncoder(w).Encode(following)
		default:
			json.NewEncoder(w).Encode(user)
		}
	}))
	defer srv.Close()

	m := New("")
	m.baseURL = srv.URL

	node := graph.NewNode("username", "kylederzweite", "seed")
	client := httpclient.New()

	nodes, edges, err := m.Extract(context.Background(), node, client)
	if err != nil {
		t.Fatal(err)
	}

	found := map[string]string{}
	for _, n := range nodes {
		if relationship, ok := n.Properties["relationship"]; ok {
			found[n.Label] = relationship.(string)
			if n.Pivot {
				t.Errorf("friend node %q should not pivot", n.Label)
			}
		}
	}
	if found["friendone"] != "follower" {
		t.Errorf("expected follower friendone, got %+v", found)
	}
	if found["friendtwo"] != "following" {
		t.Errorf("expected following friendtwo, got %+v", found)
	}

	var follows int
	for _, e := range edges {
		if e.Type == graph.EdgeTypeFollows {
			follows++
		}
	}
	if follows != 2 {
		t.Errorf("expected 2 follows edges, got %d", follows)
	}
}
