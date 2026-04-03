// SPDX-License-Identifier: AGPL-3.0-or-later

package ipinfo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/KyleDerZweite/basalt/internal/graph"
	"github.com/KyleDerZweite/basalt/internal/httpclient"
	"github.com/KyleDerZweite/basalt/internal/modules"
)

type mockResolver struct{ ip string }

func (r *mockResolver) LookupHost(_ context.Context, _ string) ([]string, error) {
	if r.ip == "" {
		return nil, fmt.Errorf("dns failed")
	}
	return []string{r.ip}, nil
}

func TestCanHandle(t *testing.T) {
	m := New()
	if !m.CanHandle("domain") {
		t.Error("should handle domain")
	}
	if m.CanHandle("username") {
		t.Error("should not handle username")
	}
}

func TestExtractFound(t *testing.T) {
	info := map[string]string{
		"ip":       "8.8.8.8",
		"hostname": "dns.google",
		"city":     "Mountain View",
		"region":   "California",
		"country":  "US",
		"loc":      "37.4056,-122.0775",
		"org":      "AS15169 Google LLC",
		"postal":   "94043",
		"timezone": "America/Los_Angeles",
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(info)
	}))
	defer srv.Close()

	m := &Module{
		baseURL:  srv.URL,
		resolver: &mockResolver{ip: "8.8.8.8"},
	}

	node := graph.NewNode("domain", "google.com", "seed")
	client := httpclient.New()

	nodes, edges, err := m.Extract(context.Background(), node, client)
	if err != nil {
		t.Fatal(err)
	}

	// account + IP + org = 3 nodes
	if len(nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(nodes))
	}

	account := nodes[0]
	if account.Type != graph.NodeTypeAccount {
		t.Errorf("expected account node, got %s", account.Type)
	}
	if account.Confidence != 0.90 {
		t.Errorf("expected confidence 0.90, got %f", account.Confidence)
	}
	if account.Properties["city"] != "Mountain View" {
		t.Errorf("expected city Mountain View, got %v", account.Properties["city"])
	}
	if account.Properties["org"] != "AS15169 Google LLC" {
		t.Errorf("expected org AS15169 Google LLC, got %v", account.Properties["org"])
	}

	ipNode := nodes[1]
	if ipNode.Type != graph.NodeTypeIP {
		t.Errorf("expected IP node, got %s", ipNode.Type)
	}
	if ipNode.Label != "8.8.8.8" {
		t.Errorf("expected label 8.8.8.8, got %s", ipNode.Label)
	}

	orgNode := nodes[2]
	if orgNode.Type != graph.NodeTypeOrganization {
		t.Errorf("expected organization node, got %s", orgNode.Type)
	}

	// has_account + resolves_to + linked_to = 3 edges
	if len(edges) != 3 {
		t.Fatalf("expected 3 edges, got %d", len(edges))
	}
}

func TestExtractNotResolvable(t *testing.T) {
	m := &Module{
		baseURL:  "http://unused",
		resolver: &mockResolver{ip: ""},
	}

	node := graph.NewNode("domain", "nonexistent.invalid", "seed")
	client := httpclient.New()

	_, _, err := m.Extract(context.Background(), node, client)
	if err == nil {
		t.Error("expected error for unresolvable domain")
	}
}

func TestVerifyHealthy(t *testing.T) {
	info := map[string]string{
		"ip":  "8.8.8.8",
		"org": "AS15169 Google LLC",
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(info)
	}))
	defer srv.Close()

	m := &Module{
		baseURL:  srv.URL,
		resolver: &mockResolver{ip: "8.8.8.8"},
	}

	client := httpclient.New()
	status, msg := m.Verify(context.Background(), client)
	if status != modules.Healthy {
		t.Errorf("expected Healthy, got %d: %s", status, msg)
	}
}
