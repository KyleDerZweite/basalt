// SPDX-License-Identifier: AGPL-3.0-or-later

package app

import (
	"testing"

	"github.com/KyleDerZweite/basalt/internal/graph"
)

func TestWorkspaceNodeFromRawCarriesEvidence(t *testing.T) {
	node := graph.NewNode(graph.NodeTypeAccount, "octocat", "github")
	node.Confidence = 0.95
	node.Pivot = true
	node.Wave = 1
	node.Properties["company"] = "GitHub"
	node.Properties["verified"] = true
	node.Properties["score"] = 0.5
	node.Properties["nested"] = map[string]string{"a": "b"}

	out := workspaceNodeFromRaw(node, "accounts")

	if out.Label != "octocat" || out.Confidence != 0.95 {
		t.Fatalf("identity fields lost: %+v", out)
	}
	if !out.Pivot || out.Wave != 1 {
		t.Fatalf("pivot state lost: %+v", out)
	}
	if len(out.SourceModules) != 1 || out.SourceModules[0] != "github" {
		t.Fatalf("source modules lost: %+v", out.SourceModules)
	}
	for _, want := range []string{"company", "verified", "score"} {
		if _, ok := out.Properties[want]; !ok {
			t.Fatalf("property %q missing: %+v", want, out.Properties)
		}
	}
	if _, ok := out.Properties["nested"]; ok {
		t.Fatalf("compound property should be skipped: %+v", out.Properties)
	}
}
