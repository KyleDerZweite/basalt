// SPDX-License-Identifier: AGPL-3.0-or-later

package graph

import (
	"encoding/json"
	"testing"
)

func TestNewNode(t *testing.T) {
	n := NewNode("username", "kylederzweite", "github")
	if n.ID != "username:kylederzweite" {
		t.Errorf("got ID %q, want %q", n.ID, "username:kylederzweite")
	}
	if n.Type != NodeTypeUsername {
		t.Errorf("got Type %q, want %q", n.Type, NodeTypeUsername)
	}
	if n.Label != "kylederzweite" {
		t.Errorf("got Label %q, want %q", n.Label, "kylederzweite")
	}
	if n.SourceModule != "github" {
		t.Errorf("got SourceModule %q, want %q", n.SourceModule, "github")
	}
}

func TestNewAccountNode(t *testing.T) {
	n := NewAccountNode("github", "kylederzweite", "https://github.com/kylederzweite", "github")
	if n.ID != "account:github:kylederzweite" {
		t.Errorf("got ID %q, want %q", n.ID, "account:github:kylederzweite")
	}
	if n.Type != NodeTypeAccount {
		t.Errorf("got Type %q, want %q", n.Type, NodeTypeAccount)
	}
	wantLabel := "github - kylederzweite"
	if n.Label != wantLabel {
		t.Errorf("got Label %q, want %q", n.Label, wantLabel)
	}
}

func TestNodePivotAndWave(t *testing.T) {
	n := NewNode("email", "kyle@example.com", "gravatar")
	n.Pivot = true
	n.Wave = 1
	n.Confidence = 0.95

	if !n.Pivot {
		t.Error("expected Pivot=true")
	}
	if n.Wave != 1 {
		t.Errorf("got Wave %d, want 1", n.Wave)
	}
	if n.Confidence != 0.95 {
		t.Errorf("got Confidence %f, want 0.95", n.Confidence)
	}
}

func TestGraphAddNodeDedup(t *testing.T) {
	g := New()
	n1 := NewNode("username", "kyle", "github")
	n2 := NewNode("username", "kyle", "reddit")

	if !g.AddNode(n1) {
		t.Error("first AddNode should return true")
	}
	if g.AddNode(n2) {
		t.Error("second AddNode with same ID should return false")
	}
}

func TestGraphEdgesNotDeduplicated(t *testing.T) {
	g := New()
	e1 := NewEdge(g.NextEdgeID(), "a", "b", EdgeTypeHasAccount, "github")
	e2 := NewEdge(g.NextEdgeID(), "a", "b", EdgeTypeHasAccount, "gravatar")
	g.AddEdge(e1)
	g.AddEdge(e2)

	nodes, edges := g.Collect()
	_ = nodes
	if len(edges) != 2 {
		t.Errorf("got %d edges, want 2 (edges should not be deduplicated)", len(edges))
	}
}

func TestSeedNodeID(t *testing.T) {
	id := SeedNodeID("username", "KyleDerZweite")
	if id != "seed:username:kylederzweite" {
		t.Errorf("got %q, want %q", id, "seed:username:kylederzweite")
	}
}

func TestParseSeed(t *testing.T) {
	s, err := ParseSeed("email:kyle@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if s.Type != "email" || s.Value != "kyle@example.com" {
		t.Errorf("got %+v", s)
	}

	_, err = ParseSeed("invalid")
	if err == nil {
		t.Error("expected error for invalid seed format")
	}
}

func TestGraphMarshalJSON(t *testing.T) {
	g := New()
	n := NewNode("username", "kyle", "test")
	g.AddNode(n)

	data, err := g.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	var out map[string]interface{}
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if _, ok := out["nodes"]; !ok {
		t.Error("JSON output missing 'nodes' key")
	}
	if _, ok := out["edges"]; !ok {
		t.Error("JSON output missing 'edges' key")
	}
	if _, ok := out["meta"]; !ok {
		t.Error("JSON output missing 'meta' key")
	}
}
