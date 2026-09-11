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

func TestBuildScanInsightsLinksIdentities(t *testing.T) {
	g := graph.New()

	githubAccount := graph.NewAccountNode("github", "kyledev", "https://github.com/kyledev", "github")
	githubAccount.Confidence = 0.95
	gitlabAccount := graph.NewAccountNode("gitlab", "kyledev", "https://gitlab.com/kyledev", "gitlab")
	gitlabAccount.Confidence = 0.9
	twitchAccount := graph.NewAccountNode("twitch", "kyledevv", "https://twitch.tv/kyledevv", "twitch")
	twitchAccount.Confidence = 0.8
	for _, node := range []*graph.Node{githubAccount, gitlabAccount, twitchAccount} {
		g.AddNode(node)
	}

	email := graph.NewNode(graph.NodeTypeEmail, "kyle@example.com", "github")
	email.Pivot = true
	email.Confidence = 0.9
	g.AddNode(email)
	g.AddEdge(graph.NewEdge(0, githubAccount.ID, email.ID, graph.EdgeTypeHasEmail, "github"))
	g.AddEdge(graph.NewEdge(0, gitlabAccount.ID, email.ID, graph.EdgeTypeHasEmail, "gitlab"))

	insights := BuildScanInsights(g, nil, ScanStatusCompleted)

	if len(insights.LinkedIdentities) != 1 {
		t.Fatalf("expected 1 linked identity, got %+v", insights.LinkedIdentities)
	}
	link := insights.LinkedIdentities[0]
	if link.Identifier != "kyle@example.com" || len(link.AccountLabels) != 2 {
		t.Fatalf("unexpected linked identity: %+v", link)
	}

	foundSameHandle := false
	for _, match := range insights.PossibleMatches {
		if match.Reason == "Same handle on github, gitlab" {
			foundSameHandle = true
		}
	}
	if !foundSameHandle {
		t.Fatalf("expected same-handle match, got %+v", insights.PossibleMatches)
	}
}
