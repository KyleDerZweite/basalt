// SPDX-License-Identifier: AGPL-3.0-or-later

package modules

import (
	"testing"

	"github.com/KyleDerZweite/basalt/internal/graph"
)

func TestInferUsernameFromDisplayName(t *testing.T) {
	node := InferUsernameFromDisplayName("VispLP", "kylederzweite", "dockerhub")
	if node == nil {
		t.Fatal("expected username inference for single-token handle-like display name")
	}
	if node.Type != graph.NodeTypeUsername {
		t.Fatalf("expected username node, got %s", node.Type)
	}
	if node.Label != "VispLP" {
		t.Fatalf("expected inferred label VispLP, got %q", node.Label)
	}
	if !node.Pivot {
		t.Fatal("expected inferred username to be pivotable")
	}
	if node.Confidence != 0.65 {
		t.Fatalf("expected inferred confidence 0.65, got %.2f", node.Confidence)
	}
	if got := node.Properties["inferred_from"]; got != graph.NodeTypeFullName {
		t.Fatalf("expected inferred_from full_name, got %#v", got)
	}
}

func TestInferUsernameFromDisplayNameRejectsUnsafeCandidates(t *testing.T) {
	cases := []struct {
		name          string
		displayName   string
		knownUsername string
	}{
		{name: "multi-word", displayName: "Test User", knownUsername: "testuser"},
		{name: "same as username", displayName: "kylederzweite", knownUsername: "KyleDerZweite"},
		{name: "email", displayName: "test@example.com", knownUsername: "testuser"},
		{name: "url", displayName: "https://example.com", knownUsername: "testuser"},
		{name: "numeric only", displayName: "123456", knownUsername: "testuser"},
		{name: "invalid chars", displayName: "VispLP!", knownUsername: "testuser"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if node := InferUsernameFromDisplayName(tc.displayName, tc.knownUsername, "dockerhub"); node != nil {
				t.Fatalf("expected no inferred username, got %+v", node)
			}
		})
	}
}
