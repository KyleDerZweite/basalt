// SPDX-License-Identifier: AGPL-3.0-or-later

package app

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"

	"github.com/KyleDerZweite/basalt/internal/graph"
)

// Store persists scans, events, and local settings.
type Store struct {
	db      *sql.DB
	eventMu sync.Mutex
}

func openStore(dataDir string) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating data dir: %w", err)
	}

	dbPath := defaultDBPath(dataDir)
	db, err := sql.Open("sqlite", defaultDBDSN(dataDir))
	if err != nil {
		return nil, fmt.Errorf("opening sqlite database %s: %w", dbPath, err)
	}
	db.SetMaxOpenConns(1)

	store := &Store{db: db}
	if err := store.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) migrate() error {
	statements := []string{
		`PRAGMA journal_mode = WAL;`,
		`CREATE TABLE IF NOT EXISTS scans (
			id TEXT PRIMARY KEY,
			target_id TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL,
			started_at TEXT NOT NULL,
			completed_at TEXT,
			updated_at TEXT NOT NULL,
			seeds_json TEXT NOT NULL,
			options_json TEXT NOT NULL,
			health_json TEXT NOT NULL,
			insights_json TEXT,
			graph_json TEXT,
			node_count INTEGER NOT NULL DEFAULT 0,
			edge_count INTEGER NOT NULL DEFAULT 0,
			error_message TEXT NOT NULL DEFAULT ''
		);`,
		`CREATE TABLE IF NOT EXISTS scan_events (
			scan_id TEXT NOT NULL,
			seq INTEGER NOT NULL,
			time TEXT NOT NULL,
			type TEXT NOT NULL,
			module TEXT NOT NULL DEFAULT '',
			node_id TEXT NOT NULL DEFAULT '',
			edge_id TEXT NOT NULL DEFAULT '',
			message TEXT NOT NULL DEFAULT '',
			data_json TEXT NOT NULL DEFAULT '{}',
			PRIMARY KEY (scan_id, seq),
			FOREIGN KEY (scan_id) REFERENCES scans(id) ON DELETE CASCADE
		);`,
		`CREATE INDEX IF NOT EXISTS idx_scan_events_scan_seq ON scan_events(scan_id, seq);`,
		`CREATE TABLE IF NOT EXISTS settings (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			data_json TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS module_health_cache (
			module_name TEXT NOT NULL,
			basalt_version TEXT NOT NULL,
			config_hash TEXT NOT NULL,
			status TEXT NOT NULL,
			message TEXT NOT NULL DEFAULT '',
			checked_at TEXT NOT NULL,
			expires_at TEXT NOT NULL,
			PRIMARY KEY (module_name, basalt_version, config_hash)
		);`,
		`CREATE TABLE IF NOT EXISTS targets (
			id TEXT PRIMARY KEY,
			slug TEXT NOT NULL UNIQUE,
			display_name TEXT NOT NULL,
			notes TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS target_aliases (
			id TEXT PRIMARY KEY,
			target_id TEXT NOT NULL,
			seed_type TEXT NOT NULL,
			seed_value TEXT NOT NULL,
			label TEXT NOT NULL DEFAULT '',
			is_primary INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			FOREIGN KEY (target_id) REFERENCES targets(id) ON DELETE CASCADE,
			UNIQUE (target_id, seed_type, seed_value)
		);`,
		`CREATE INDEX IF NOT EXISTS idx_target_aliases_target ON target_aliases(target_id);`,
	}

	for _, statement := range statements {
		if _, err := s.db.Exec(statement); err != nil {
			return fmt.Errorf("migrating database: %w", err)
		}
	}
	for _, statement := range []string{
		`ALTER TABLE scans ADD COLUMN target_id TEXT NOT NULL DEFAULT '';`,
		`ALTER TABLE scans ADD COLUMN insights_json TEXT;`,
	} {
		if _, err := s.db.Exec(statement); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
			return fmt.Errorf("migrating database: %w", err)
		}
	}
	if _, err := s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_scans_target_id ON scans(target_id);`); err != nil {
		return fmt.Errorf("migrating database: %w", err)
	}
	return nil
}

type persistedGraph struct {
	Meta  graph.Meta    `json:"meta"`
	Nodes []*graph.Node `json:"nodes"`
	Edges []*graph.Edge `json:"edges"`
}

func decodeGraph(data []byte) (*graph.Graph, error) {
	var persisted persistedGraph
	if err := json.Unmarshal(data, &persisted); err != nil {
		return nil, fmt.Errorf("decoding graph: %w", err)
	}

	out := graph.New()
	out.Meta = persisted.Meta
	for _, node := range persisted.Nodes {
		out.AddNode(node)
	}
	for _, edge := range persisted.Edges {
		out.AddEdge(edge)
	}
	out.RestoreStats(persisted.Meta.Stats, len(persisted.Edges))
	return out, nil
}

func timeString(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func tempDBPath(dir string) string {
	return filepath.Join(dir, "basalt.db")
}
