// SPDX-License-Identifier: AGPL-3.0-or-later

package dnsct

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/KyleDerZweite/basalt/internal/graph"
	"github.com/KyleDerZweite/basalt/internal/httpclient"
	"github.com/KyleDerZweite/basalt/internal/modules"
)

// mockResolver returns predictable DNS results for testing.
type mockResolver struct {
	hosts   []string
	hostErr error
	mx      []*net.MX
	mxErr   error
}

func (r *mockResolver) LookupHost(_ context.Context, _ string) ([]string, error) {
	return r.hosts, r.hostErr
}

func (r *mockResolver) LookupMX(_ context.Context, _ string) ([]*net.MX, error) {
	return r.mx, r.mxErr
}

func TestCanHandle(t *testing.T) {
	m := New()
	if !m.CanHandle("domain") {
		t.Error("should handle domain")
	}
	if m.CanHandle("username") {
		t.Error("should not handle username")
	}
	if m.CanHandle("email") {
		t.Error("should not handle email")
	}
}

func TestExtractFound(t *testing.T) {
	ctEntries := []map[string]interface{}{
		{"name_value": "www.example.com", "common_name": "example.com"},
		{"name_value": "mail.example.com", "common_name": "example.com"},
		{"name_value": "*.example.com", "common_name": "example.com"},
		{"name_value": "example.com", "common_name": "example.com"},
		// Duplicate to test dedup.
		{"name_value": "www.example.com", "common_name": "example.com"},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ctEntries)
	}))
	defer srv.Close()

	m := New()
	m.ctBaseURL = srv.URL
	m.resolver = &mockResolver{
		hosts: []string{"93.184.216.34", "2606:2800:220:1:248:1893:25c8:1946"},
		mx: []*net.MX{
			{Host: "mx1.example.com.", Pref: 10},
			{Host: "mx2.example.com.", Pref: 20},
		},
	}

	node := graph.NewNode("domain", "example.com", "seed")
	client := httpclient.New()

	nodes, edges, err := m.Extract(context.Background(), node, client)
	if err != nil {
		t.Fatal(err)
	}

	// Expect: 1 account + 2 subdomains (www + mail; wildcard and base domain filtered).
	if len(nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(nodes))
	}
	if len(edges) != 3 {
		t.Fatalf("expected 3 edges, got %d", len(edges))
	}

	// Verify account node.
	account := nodes[0]
	if account.Type != graph.NodeTypeAccount {
		t.Errorf("expected account node, got %s", account.Type)
	}
	if account.Confidence != 0.90 {
		t.Errorf("expected confidence 0.90, got %f", account.Confidence)
	}

	aRecords, ok := account.Properties["a_records"].([]string)
	if !ok || len(aRecords) != 2 {
		t.Errorf("expected 2 A records, got %v", account.Properties["a_records"])
	}

	mxRecords, ok := account.Properties["mx_records"].([]string)
	if !ok || len(mxRecords) != 2 {
		t.Errorf("expected 2 MX records, got %v", account.Properties["mx_records"])
	}
	// MX trailing dots should be trimmed.
	if mxRecords[0] != "mx1.example.com" {
		t.Errorf("expected mx1.example.com, got %s", mxRecords[0])
	}

	// Verify subdomain nodes are not pivotable.
	for _, n := range nodes[1:] {
		if n.Type != graph.NodeTypeDomain {
			t.Errorf("expected domain node, got %s", n.Type)
		}
		if n.Pivot {
			t.Error("subdomain nodes should not be pivotable")
		}
	}
}

func TestExtractNoDNS(t *testing.T) {
	// DNS fails, but CT still returns results.
	ctEntries := []map[string]interface{}{
		{"name_value": "api.example.com", "common_name": "example.com"},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ctEntries)
	}))
	defer srv.Close()

	m := New()
	m.ctBaseURL = srv.URL
	m.resolver = &mockResolver{
		hostErr: &net.DNSError{Err: "no such host", Name: "example.com"},
		mxErr:   &net.DNSError{Err: "no such host", Name: "example.com"},
	}

	node := graph.NewNode("domain", "example.com", "seed")
	client := httpclient.New()

	nodes, edges, err := m.Extract(context.Background(), node, client)
	if err != nil {
		t.Fatal(err)
	}

	// Should still return account + subdomain from CT.
	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes (account + 1 subdomain), got %d", len(nodes))
	}
	if len(edges) != 2 {
		t.Fatalf("expected 2 edges, got %d", len(edges))
	}

	// Account should not have A/MX records when DNS failed.
	account := nodes[0]
	if _, ok := account.Properties["a_records"]; ok {
		t.Error("should not have a_records when DNS failed")
	}
	if _, ok := account.Properties["mx_records"]; ok {
		t.Error("should not have mx_records when DNS failed")
	}
}

func TestExtractNothingFound(t *testing.T) {
	// Both DNS and CT return nothing.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
	}))
	defer srv.Close()

	m := New()
	m.ctBaseURL = srv.URL
	m.resolver = &mockResolver{
		hostErr: &net.DNSError{Err: "no such host", Name: "nonexistent.example"},
		mxErr:   &net.DNSError{Err: "no such host", Name: "nonexistent.example"},
	}

	node := graph.NewNode("domain", "nonexistent.example", "seed")
	client := httpclient.New()

	nodes, edges, err := m.Extract(context.Background(), node, client)
	if err != nil {
		t.Fatal(err)
	}
	if nodes != nil {
		t.Errorf("expected nil nodes, got %d", len(nodes))
	}
	if edges != nil {
		t.Errorf("expected nil edges, got %d", len(edges))
	}
}

func TestVerifyHealthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"name_value":"www.example.com"}]`))
	}))
	defer srv.Close()

	m := New()
	m.ctBaseURL = srv.URL
	m.resolver = &mockResolver{hosts: []string{"93.184.216.34"}}

	client := httpclient.New()
	status, msg := m.Verify(context.Background(), client)
	if status != modules.Healthy {
		t.Errorf("expected Healthy, got %d: %s", status, msg)
	}
}

func TestVerifyDegradedWhenCTUnavailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	m := New()
	m.ctBaseURL = srv.URL
	m.resolver = &mockResolver{hosts: []string{"93.184.216.34"}}

	client := httpclient.New()
	status, _ := m.Verify(context.Background(), client)
	if status != modules.Degraded {
		t.Errorf("expected Degraded, got %d", status)
	}
}
