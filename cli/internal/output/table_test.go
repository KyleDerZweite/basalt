// SPDX-License-Identifier: AGPL-3.0-or-later

package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/KyleDerZweite/basalt/internal/graph"
)

func testGraph() *graph.Graph {
	g := graph.New()

	account := graph.NewAccountNode("github", "kyle", "https://github.com/kyle", "github")
	account.Confidence = 0.95
	account.Wave = 1
	g.AddNode(account)

	email := graph.NewNode(graph.NodeTypeEmail, "kyle@example.com", "github")
	email.Confidence = 0.90
	email.Wave = 1
	g.AddNode(email)

	domain := graph.NewNode(graph.NodeTypeDomain, "kylehub.dev", "github")
	domain.Confidence = 0.85
	domain.Wave = 2
	g.AddNode(domain)

	return g
}

func TestWriteTable(t *testing.T) {
	g := testGraph()
	g.Meta.DurationSecs = 1.5

	var buf bytes.Buffer
	if err := WriteTable(&buf, g); err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	if !strings.Contains(out, "github") {
		t.Error("table should contain 'github'")
	}
	if !strings.Contains(out, "kyle@example.com") {
		t.Error("table should contain email")
	}
	if !strings.Contains(out, "0.95") {
		t.Error("table should contain confidence score")
	}
}

func TestWriteTableEmpty(t *testing.T) {
	g := graph.New()
	var buf bytes.Buffer
	if err := WriteTable(&buf, g); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "No results") {
		t.Error("empty graph should print 'No results'")
	}
}
