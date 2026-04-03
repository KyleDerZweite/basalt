// SPDX-License-Identifier: AGPL-3.0-or-later

package opgg

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/KyleDerZweite/basalt/internal/graph"
	"github.com/KyleDerZweite/basalt/internal/httpclient"
	"github.com/KyleDerZweite/basalt/internal/modules"
)

const foundHTML = `<html><head>
<meta property="og:title" content="TestPlayer#EUW - Summoner Stats - League of Legends" />
<meta property="og:description" content="TestPlayer#EUW / Lv. 120" />
<meta property="og:site_name" content="OP.GG" />
</head><body></body></html>`

const genericSearchHTML = `<html><head>
<meta property="og:title" content="OP.GG - The Best LoL Builds and Tier List. Search Riot ID and Tagline for Stats" />
<meta property="og:description" content="The Best LoL Champion Builds and Player Stats by OP.GG" />
<meta property="og:url" content="https://op.gg/lol/summoners/search" />
<meta property="og:site_name" content="OP.GG" />
</head><body></body></html>`

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
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(foundHTML))
	}))
	defer srv.Close()

	m := New()
	m.baseURL = srv.URL

	node := graph.NewNode("username", "TestPlayer", "seed")
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
	if nodes[0].Properties["summoner_name"] != "TestPlayer#EUW" {
		t.Errorf("expected summoner_name, got %v", nodes[0].Properties["summoner_name"])
	}
	if nodes[0].Properties["level"] != "120" {
		t.Errorf("expected level 120, got %v", nodes[0].Properties["level"])
	}
	if len(edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(edges))
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
	if len(nodes) != 0 || len(edges) != 0 {
		t.Error("expected no results for 404")
	}
}

func TestVerifyHealthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(foundHTML))
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

func TestVerifyDegradedForGenericSearchPage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(genericSearchHTML))
	}))
	defer srv.Close()

	m := New()
	m.baseURL = srv.URL

	client := httpclient.New()
	status, _ := m.Verify(context.Background(), client)
	if status != modules.Degraded {
		t.Errorf("expected Degraded, got %d", status)
	}
}
