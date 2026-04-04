// SPDX-License-Identifier: AGPL-3.0-or-later

package app

import (
	"testing"
	"time"

	"github.com/KyleDerZweite/basalt/internal/graph"
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

func testStore(t *testing.T) *Store {
	t.Helper()

	dir := t.TempDir()
	store, err := openStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	return store
}
