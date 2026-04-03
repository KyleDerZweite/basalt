// SPDX-License-Identifier: AGPL-3.0-or-later

package whois

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
	rdap := map[string]interface{}{
		"ldhName": "example.com",
		"entities": []interface{}{
			map[string]interface{}{
				"roles": []interface{}{"registrant"},
				"vcardArray": []interface{}{
					"vcard",
					[]interface{}{
						[]interface{}{"version", map[string]interface{}{}, "text", "4.0"},
						[]interface{}{"fn", map[string]interface{}{}, "text", "Test Registrant"},
						[]interface{}{"email", map[string]interface{}{}, "text", "admin@example.com"},
						[]interface{}{"org", map[string]interface{}{}, "text", "Example Inc"},
					},
				},
			},
		},
		"events": []interface{}{
			map[string]interface{}{"eventAction": "registration", "eventDate": "2020-01-01T00:00:00Z"},
			map[string]interface{}{"eventAction": "expiration", "eventDate": "2025-01-01T00:00:00Z"},
		},
		"links": []interface{}{
			map[string]interface{}{"rel": "self", "href": "https://rdap.org/domain/example.com"},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rdap+json")
		json.NewEncoder(w).Encode(rdap)
	}))
	defer srv.Close()

	m := New()
	m.baseURL = srv.URL

	node := graph.NewNode("domain", "example.com", "seed")
	client := httpclient.New()

	nodes, edges, err := m.Extract(context.Background(), node, client)
	if err != nil {
		t.Fatal(err)
	}

	// Expect: account + full_name + email + organization = 4 nodes.
	if len(nodes) != 4 {
		t.Fatalf("expected 4 nodes, got %d", len(nodes))
	}
	// Expect: registered_to + linked_to(name) + has_email + linked_to(org) = 4 edges.
	if len(edges) != 4 {
		t.Fatalf("expected 4 edges, got %d", len(edges))
	}

	// Verify account node properties.
	account := nodes[0]
	if account.Type != graph.NodeTypeAccount {
		t.Errorf("expected account node, got %s", account.Type)
	}
	if account.Confidence != 0.85 {
		t.Errorf("expected confidence 0.85, got %f", account.Confidence)
	}
	if account.Properties["registration_date"] != "2020-01-01T00:00:00Z" {
		t.Errorf("missing registration_date")
	}
	if account.Properties["expiration_date"] != "2025-01-01T00:00:00Z" {
		t.Errorf("missing expiration_date")
	}

	// Verify extracted contact nodes.
	var foundName, foundEmail, foundOrg bool
	for _, n := range nodes {
		switch n.Type {
		case graph.NodeTypeFullName:
			foundName = true
			if n.Label != "Test Registrant" {
				t.Errorf("expected name 'Test Registrant', got %q", n.Label)
			}
		case graph.NodeTypeEmail:
			foundEmail = true
			if n.Label != "admin@example.com" {
				t.Errorf("expected email 'admin@example.com', got %q", n.Label)
			}
			if !n.Pivot {
				t.Error("email node should be pivotable")
			}
		case graph.NodeTypeOrganization:
			foundOrg = true
			if n.Label != "Example Inc" {
				t.Errorf("expected org 'Example Inc', got %q", n.Label)
			}
		}
	}
	if !foundName {
		t.Error("expected full_name node")
	}
	if !foundEmail {
		t.Error("expected email node")
	}
	if !foundOrg {
		t.Error("expected organization node")
	}
}

func TestExtractNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	m := New()
	m.baseURL = srv.URL

	node := graph.NewNode("domain", "nonexistent.example", "seed")
	client := httpclient.New()

	nodes, edges, err := m.Extract(context.Background(), node, client)
	if err != nil {
		t.Fatal(err)
	}
	if nodes != nil {
		t.Errorf("expected nil nodes for 404, got %d", len(nodes))
	}
	if edges != nil {
		t.Errorf("expected nil edges for 404, got %d", len(edges))
	}
}

func TestExtractRedacted(t *testing.T) {
	// Simulates GDPR-redacted WHOIS: entities exist but no vcard data.
	rdap := map[string]interface{}{
		"ldhName": "redacted.com",
		"entities": []interface{}{
			map[string]interface{}{
				"roles": []interface{}{"registrant"},
				// No vcardArray at all.
			},
		},
		"events": []interface{}{
			map[string]interface{}{"eventAction": "registration", "eventDate": "2021-06-15T00:00:00Z"},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rdap+json")
		json.NewEncoder(w).Encode(rdap)
	}))
	defer srv.Close()

	m := New()
	m.baseURL = srv.URL

	node := graph.NewNode("domain", "redacted.com", "seed")
	client := httpclient.New()

	nodes, edges, err := m.Extract(context.Background(), node, client)
	if err != nil {
		t.Fatal(err)
	}

	// Should still return the account node even without contact data.
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node (account only), got %d", len(nodes))
	}
	if nodes[0].Type != graph.NodeTypeAccount {
		t.Errorf("expected account node, got %s", nodes[0].Type)
	}
	if len(edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(edges))
	}
	if nodes[0].Properties["registration_date"] != "2021-06-15T00:00:00Z" {
		t.Error("expected registration_date on redacted domain")
	}
}

func TestVerifyHealthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rdap+json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ldhName": "example.com",
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
