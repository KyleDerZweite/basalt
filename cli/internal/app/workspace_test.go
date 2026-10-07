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

func TestBuildPersonProfiles(t *testing.T) {
	g := graph.New()

	githubAccount := graph.NewAccountNode("github", "kyledev", "https://github.com/kyledev", "github")
	githubAccount.Confidence = 0.95
	gitlabAccount := graph.NewAccountNode("gitlab", "kyledev", "https://gitlab.com/kyledev", "gitlab")
	gitlabAccount.Confidence = 0.9
	strayAccount := graph.NewAccountNode("twitch", "someoneelse", "https://twitch.tv/someoneelse", "twitch")
	strayAccount.Confidence = 0.8
	for _, node := range []*graph.Node{githubAccount, gitlabAccount, strayAccount} {
		g.AddNode(node)
	}

	email := graph.NewNode(graph.NodeTypeEmail, "kyle@example.com", "github")
	email.Pivot = true
	g.AddNode(email)
	g.AddEdge(graph.NewEdge(0, githubAccount.ID, email.ID, graph.EdgeTypeHasEmail, "github"))
	g.AddEdge(graph.NewEdge(0, gitlabAccount.ID, email.ID, graph.EdgeTypeHasEmail, "gitlab"))

	avatar := graph.NewNode(graph.NodeTypeAvatarURL, "https://avatars.example.com/kyle.png", "github")
	avatar.Confidence = 0.95
	g.AddNode(avatar)
	g.AddEdge(graph.NewEdge(0, githubAccount.ID, avatar.ID, graph.EdgeTypeLinkedTo, "github"))

	insights := BuildScanInsights(g, nil, ScanStatusCompleted)

	if len(insights.Profiles) != 2 {
		t.Fatalf("expected 2 profiles, got %+v", insights.Profiles)
	}
	main := insights.Profiles[0]
	if main.PrimaryHandle != "kyledev" {
		t.Fatalf("expected primary handle kyledev, got %+v", main)
	}
	if len(main.Modules) != 2 || len(main.Emails) != 1 {
		t.Fatalf("profile missing modules or emails: %+v", main)
	}
	if len(main.AvatarURLs) != 1 || main.AvatarURLs[0] != "https://avatars.example.com/kyle.png" {
		t.Fatalf("profile missing avatar: %+v", main)
	}
	if len(main.Reasons) == 0 {
		t.Fatalf("profile should explain its merge: %+v", main)
	}
}
