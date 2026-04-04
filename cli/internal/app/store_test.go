// SPDX-License-Identifier: AGPL-3.0-or-later

package app

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/KyleDerZweite/basalt/internal/graph"
	_ "modernc.org/sqlite"
)

func TestStoreSettingsRoundTrip(t *testing.T) {
	store := testStore(t)
	defer store.Close()

	now := time.Now().UTC().Round(time.Second)
	settings := Settings{
		StrictMode:      true,
		DisabledModules: []string{"github", "github", "reddit"},
		LegalAcceptedAt: &now,
	}
	if err := store.SaveSettings(settings); err != nil {
		t.Fatal(err)
	}

	got, err := store.GetSettings()
	if err != nil {
		t.Fatal(err)
	}
	if !got.StrictMode {
		t.Fatal("expected strict mode to persist")
	}
	if len(got.DisabledModules) != 2 {
		t.Fatalf("expected deduped disabled modules, got %v", got.DisabledModules)
	}
	if got.LegalAcceptedAt == nil || !got.LegalAcceptedAt.Equal(now) {
		t.Fatalf("unexpected accepted timestamp: got %s want %s", got.LegalAcceptedAt, now)
	}
}

func TestStoreScanAndEventsRoundTrip(t *testing.T) {
	store := testStore(t)
	defer store.Close()

	startedAt := time.Now().UTC().Add(-time.Minute).Round(time.Second)
	record := &ScanRecord{
		ID:        "scan-1",
		Status:    ScanStatusRunning,
		StartedAt: startedAt,
		UpdatedAt: startedAt,
		Seeds:     []graph.Seed{{Type: graph.NodeTypeUsername, Value: "kyle"}},
		Options: ScanRequest{
			Depth:          2,
			Concurrency:    5,
			TimeoutSeconds: 10,
		},
	}
	if err := store.CreateScan(record); err != nil {
		t.Fatal(err)
	}

	g := graph.New()
	g.Meta.ScanID = record.ID
	g.IncrModulesRun()
	g.IncrModulesRun()
	g.IncrErrors()
	g.IncrNodesFound()
	node := graph.NewNode(graph.NodeTypeAccount, "github:kyle", "github")
	node.Confidence = 0.95
	if !g.AddNode(node) {
		t.Fatal("expected node to be added")
	}
	record.Graph = g
	record.Health = []ModuleStatus{{Name: "github", Status: "healthy", Message: "ok"}}
	record.Status = ScanStatusCompleted
	completedAt := time.Now().UTC().Round(time.Second)
	record.CompletedAt = &completedAt
	record.UpdatedAt = completedAt
	if err := store.UpdateScan(record); err != nil {
		t.Fatal(err)
	}

	event := &ScanEvent{
		ScanID:  record.ID,
		Time:    completedAt,
		Type:    "scan_finished",
		Message: "done",
	}
	if err := store.AppendEvent(event); err != nil {
		t.Fatal(err)
	}
	if event.Sequence != 1 {
		t.Fatalf("expected first event sequence 1, got %d", event.Sequence)
	}

	got, err := store.GetScan(record.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != ScanStatusCompleted {
		t.Fatalf("expected completed scan, got %s", got.Status)
	}
	if got.Graph == nil {
		t.Fatal("expected persisted graph")
	}
	nodes, _ := got.Graph.Collect()
	if len(nodes) != 1 {
		t.Fatalf("expected 1 graph node, got %d", len(nodes))
	}
	if got.Graph.Meta.Stats.ModulesRun != 2 || got.Graph.Meta.Stats.Errors != 1 || got.Graph.Meta.Stats.NodesFound != 1 {
		t.Fatalf("unexpected restored stats: %+v", got.Graph.Meta.Stats)
	}

	events, err := store.ListEvents(record.ID, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Type != "scan_finished" {
		t.Fatalf("unexpected events: %+v", events)
	}
}

func TestStoreTargetRoundTrip(t *testing.T) {
	store := testStore(t)
	defer store.Close()

	now := time.Now().UTC().Round(time.Second)
	target := &Target{
		ID:          "target-1",
		Slug:        "kyle",
		DisplayName: "Kyle",
		Notes:       "primary target",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := store.CreateTarget(target); err != nil {
		t.Fatal(err)
	}

	alias := &TargetAlias{
		ID:        "alias-1",
		TargetID:  target.ID,
		SeedType:  graph.NodeTypeUsername,
		SeedValue: "kylederzweite",
		Label:     "main handle",
		IsPrimary: true,
		CreatedAt: now,
	}
	if err := store.AddTargetAlias(alias); err != nil {
		t.Fatal(err)
	}

	got, err := store.GetTarget(target.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.DisplayName != "Kyle" || got.Slug != "kyle" {
		t.Fatalf("unexpected target: %+v", got)
	}
	if len(got.Aliases) != 1 {
		t.Fatalf("expected 1 alias, got %d", len(got.Aliases))
	}
	if got.Aliases[0].SeedValue != "kylederzweite" || !got.Aliases[0].IsPrimary {
		t.Fatalf("unexpected alias: %+v", got.Aliases[0])
	}
}

func TestStoreMigratesLegacyScansTable(t *testing.T) {
	dir := t.TempDir()
	db, err := sql.Open("sqlite", filepath.Join(dir, "basalt.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	for _, statement := range []string{
		`CREATE TABLE scans (
			id TEXT PRIMARY KEY,
			status TEXT NOT NULL,
			started_at TEXT NOT NULL,
			completed_at TEXT,
			updated_at TEXT NOT NULL,
			seeds_json TEXT NOT NULL,
			options_json TEXT NOT NULL,
			health_json TEXT NOT NULL,
			graph_json TEXT,
			node_count INTEGER NOT NULL DEFAULT 0,
			edge_count INTEGER NOT NULL DEFAULT 0,
			error_message TEXT NOT NULL DEFAULT ''
		);`,
		`CREATE TABLE scan_events (
			scan_id TEXT NOT NULL,
			seq INTEGER NOT NULL,
			time TEXT NOT NULL,
			type TEXT NOT NULL,
			module TEXT NOT NULL DEFAULT '',
			node_id TEXT NOT NULL DEFAULT '',
			edge_id TEXT NOT NULL DEFAULT '',
			message TEXT NOT NULL DEFAULT '',
			data_json TEXT NOT NULL DEFAULT '{}',
			PRIMARY KEY (scan_id, seq)
		);`,
		`CREATE TABLE settings (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			data_json TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	store, err := openStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	row := store.db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('scans') WHERE name = 'target_id'`)
	var targetIDCount int
	if err := row.Scan(&targetIDCount); err != nil {
		t.Fatal(err)
	}
	if targetIDCount != 1 {
		t.Fatalf("expected target_id column to be added, got %d", targetIDCount)
	}

	row = store.db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('scans') WHERE name = 'insights_json'`)
	var insightsCount int
	if err := row.Scan(&insightsCount); err != nil {
		t.Fatal(err)
	}
	if insightsCount != 1 {
		t.Fatalf("expected insights_json column to be added, got %d", insightsCount)
	}
}

func TestBuildWorkspaceAndInsights(t *testing.T) {
	store := testStore(t)
	defer store.Close()

	now := time.Now().UTC().Round(time.Second)
	target := &Target{
		ID:          "target-1",
		Slug:        "kyle",
		DisplayName: "Kyle",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := store.CreateTarget(target); err != nil {
		t.Fatal(err)
	}
	if err := store.AddTargetAlias(&TargetAlias{
		ID:        "alias-1",
		TargetID:  target.ID,
		SeedType:  graph.NodeTypeUsername,
		SeedValue: "kylederzweite",
		IsPrimary: true,
		CreatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	g := graph.New()
	g.Meta.ScanID = "scan-1"
	account := graph.NewAccountNode("github", "KyleDerZweite", "https://github.com/KyleDerZweite", "github")
	account.Confidence = 0.95
	account.Properties["site_name"] = "github"
	name := graph.NewNode(graph.NodeTypeFullName, "Kyle", "github")
	name.Confidence = 0.9
	domain := graph.NewNode(graph.NodeTypeDomain, "kylehub.dev", "github")
	domain.Confidence = 0.85
	if !g.AddNode(account) || !g.AddNode(name) || !g.AddNode(domain) {
		t.Fatal("expected nodes to be added")
	}
	g.AddEdge(graph.NewEdge(g.NextEdgeID(), graph.SeedNodeID(graph.NodeTypeUsername, "kylederzweite"), account.ID, graph.EdgeTypeHasAccount, "github"))

	record := &ScanRecord{
		ID:        "scan-1",
		TargetID:  target.ID,
		Status:    ScanStatusCompleted,
		StartedAt: now,
		UpdatedAt: now,
		Seeds:     []graph.Seed{{Type: graph.NodeTypeUsername, Value: "kylederzweite"}},
		Options: ScanRequest{
			TargetRef: target.Slug,
		},
		Health: []ModuleStatus{{Name: "github", Status: "healthy", Message: "ok"}},
		Graph:  g,
	}
	record.Insights = ptr(BuildScanInsights(g, record.Health, record.Status))
	if err := store.CreateScan(record); err != nil {
		t.Fatal(err)
	}
	if err := store.UpdateScan(record); err != nil {
		t.Fatal(err)
	}

	service := &Service{store: store}
	workspace, err := service.BuildWorkspace(record.ID)
	if err != nil {
		t.Fatal(err)
	}
	if workspace.Target == nil || workspace.Target.Slug != "kyle" {
		t.Fatalf("unexpected workspace target: %+v", workspace.Target)
	}
	if workspace.Insights == nil || workspace.Insights.Headline == "" {
		t.Fatal("expected workspace insights")
	}
	if len(workspace.Graph.Nodes) == 0 || len(workspace.Graph.Edges) == 0 {
		t.Fatalf("expected synthesized workspace graph, got %+v", workspace.Graph)
	}
}

func ptr[T any](value T) *T {
	return &value
}

func testStore(t *testing.T) *Store {
	t.Helper()

	dir := t.TempDir()
	store, err := openStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	return store
}
