// SPDX-License-Identifier: AGPL-3.0-or-later

package app

import (
	"fmt"
	"testing"

	"github.com/KyleDerZweite/basalt/internal/graph"
)

func TestSocialEdgesDoNotLinkPersonIdentities(t *testing.T) {
	for _, edgeType := range []string{graph.EdgeTypeFollows, graph.EdgeTypeFriendOf, graph.EdgeTypeMentions} {
		t.Run(edgeType, func(t *testing.T) {
			g := graph.New()
			alice := graph.NewAccountNode("github", "alice123", "https://github.com/alice123", "github")
			bob := graph.NewAccountNode("github", "bobby456", "https://github.com/bobby456", "github")
			other := graph.NewNode(graph.NodeTypeUsername, "popular789", "github")
			for _, node := range []*graph.Node{alice, bob, other} {
				node.Confidence = 0.9
				g.AddNode(node)
			}
			g.AddEdge(graph.NewEdge(g.NextEdgeID(), alice.ID, other.ID, edgeType, "github"))
			g.AddEdge(graph.NewEdge(g.NextEdgeID(), bob.ID, other.ID, edgeType, "github"))
			insights := BuildScanInsights(g, nil, ScanStatusCompleted)
			if len(insights.Profiles) != 2 || len(insights.LinkedIdentities) != 0 {
				t.Fatalf("social neighbors became identity evidence: profiles=%+v links=%+v", insights.Profiles, insights.LinkedIdentities)
			}
			for _, profile := range insights.Profiles {
				if len(profile.Usernames) != 1 || profile.Usernames[0] == other.Label || len(profile.NodeIDs) != 1 {
					t.Fatalf("social neighbor leaked into person profile: %+v", profile)
				}
			}
		})
	}
}

func TestWorkspaceKeepsFriendsVisibleWithIdentitySignals(t *testing.T) {
	for _, count := range []int{0, 25} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			g := graph.New()
			friend := graph.NewNode(graph.NodeTypeUsername, "popular789", "github")
			friend.Properties["relationship"] = "following"
			friend.Confidence = 0.9
			g.AddNode(friend)
			for i := 0; i < count; i++ {
				node := graph.NewNode(graph.NodeTypeEmail, fmt.Sprintf("identity%d@example.com", i), "github")
				node.Confidence = 1
				g.AddNode(node)
			}
			view := BuildWorkspaceGraph(&ScanRecord{Graph: g}, nil)
			found := 0
			for _, node := range view.Nodes {
				if node.ID == "raw:"+friend.ID {
					found++
					if node.Category != "people" {
						t.Fatalf("friend assigned to %q instead of people", node.Category)
					}
				}
			}
			if found != 1 {
				t.Fatalf("expected exactly one visible friend, got %d", found)
			}
		})
	}
}
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
