// SPDX-License-Identifier: AGPL-3.0-or-later

package app

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"

	"github.com/KyleDerZweite/basalt/internal/graph"
)

// Store persists scans, events, and local settings.
type Store struct {
	db *sql.DB
}

func openStore(dataDir string) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating data dir: %w", err)
	}

	dbPath := defaultDBPath(dataDir)
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening sqlite database %s: %w", dbPath, err)
	}

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
	}

	for _, statement := range statements {
		if _, err := s.db.Exec(statement); err != nil {
			return fmt.Errorf("migrating database: %w", err)
		}
	}
	return nil
}

func (s *Store) GetSettings() (Settings, error) {
	var data string
	err := s.db.QueryRow(`SELECT data_json FROM settings WHERE id = 1`).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return Settings{}, nil
	}
	if err != nil {
		return Settings{}, fmt.Errorf("querying settings: %w", err)
	}

	var settings Settings
	if err := json.Unmarshal([]byte(data), &settings); err != nil {
		return Settings{}, fmt.Errorf("decoding settings: %w", err)
	}
	return normalizeSettings(settings), nil
}

func (s *Store) SaveSettings(settings Settings) error {
	settings = normalizeSettings(settings)
	data, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("encoding settings: %w", err)
	}

	_, err = s.db.Exec(`
		INSERT INTO settings (id, data_json, updated_at)
		VALUES (1, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			data_json = excluded.data_json,
			updated_at = excluded.updated_at
	`, string(data), time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("saving settings: %w", err)
	}
	return nil
}

func (s *Store) CreateScan(record *ScanRecord) error {
	seedsJSON, err := json.Marshal(record.Seeds)
	if err != nil {
		return fmt.Errorf("encoding seeds: %w", err)
	}
	optionsJSON, err := json.Marshal(record.Options)
	if err != nil {
		return fmt.Errorf("encoding scan options: %w", err)
	}
	healthJSON, err := json.Marshal(record.Health)
	if err != nil {
		return fmt.Errorf("encoding module health: %w", err)
	}

	_, err = s.db.Exec(`
		INSERT INTO scans (
			id, status, started_at, completed_at, updated_at, seeds_json, options_json,
			health_json, graph_json, node_count, edge_count, error_message
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		record.ID,
		string(record.Status),
		record.StartedAt.UTC().Format(time.RFC3339Nano),
		timeString(record.CompletedAt),
		record.UpdatedAt.UTC().Format(time.RFC3339Nano),
		string(seedsJSON),
		string(optionsJSON),
		string(healthJSON),
		nil,
		record.NodeCount,
		record.EdgeCount,
		record.ErrorMessage,
	)
	if err != nil {
		return fmt.Errorf("creating scan: %w", err)
	}
	return nil
}

func (s *Store) UpdateScan(record *ScanRecord) error {
	seedsJSON, err := json.Marshal(record.Seeds)
	if err != nil {
		return fmt.Errorf("encoding seeds: %w", err)
	}
	optionsJSON, err := json.Marshal(record.Options)
	if err != nil {
		return fmt.Errorf("encoding scan options: %w", err)
	}
	healthJSON, err := json.Marshal(record.Health)
	if err != nil {
		return fmt.Errorf("encoding module health: %w", err)
	}

	var graphJSON any
	if record.Graph != nil {
		payload, err := record.Graph.MarshalJSON()
		if err != nil {
			return fmt.Errorf("encoding graph: %w", err)
		}
		graphJSON = string(payload)
		nodes, edges := record.Graph.Collect()
		record.NodeCount = len(nodes)
		record.EdgeCount = len(edges)
	}

	_, err = s.db.Exec(`
		UPDATE scans
		SET status = ?, completed_at = ?, updated_at = ?, seeds_json = ?, options_json = ?,
		    health_json = ?, graph_json = ?, node_count = ?, edge_count = ?, error_message = ?
		WHERE id = ?
	`,
		string(record.Status),
		timeString(record.CompletedAt),
		record.UpdatedAt.UTC().Format(time.RFC3339Nano),
		string(seedsJSON),
		string(optionsJSON),
		string(healthJSON),
		graphJSON,
		record.NodeCount,
		record.EdgeCount,
		record.ErrorMessage,
		record.ID,
	)
	if err != nil {
		return fmt.Errorf("updating scan %s: %w", record.ID, err)
	}
	return nil
}

func (s *Store) GetScan(id string) (*ScanRecord, error) {
	row := s.db.QueryRow(`
		SELECT id, status, started_at, completed_at, updated_at, seeds_json, options_json,
		       health_json, graph_json, node_count, edge_count, error_message
		FROM scans
		WHERE id = ?
	`, id)
	record, err := scanFromRow(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("scan %s not found", id)
	}
	if err != nil {
		return nil, err
	}
	return record, nil
}

func (s *Store) ListScans(limit int) ([]*ScanRecord, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.Query(`
		SELECT id, status, started_at, completed_at, updated_at, seeds_json, options_json,
		       health_json, graph_json, node_count, edge_count, error_message
		FROM scans
		ORDER BY started_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("listing scans: %w", err)
	}
	defer rows.Close()

	var out []*ScanRecord
	for rows.Next() {
		record, err := scanFromRow(rows.Scan)
		if err != nil {
			return nil, err
		}
		record.Graph = nil
		out = append(out, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating scans: %w", err)
	}
	if out == nil {
		out = []*ScanRecord{}
	}
	return out, nil
}

func (s *Store) AppendEvent(event *ScanEvent) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("starting event transaction: %w", err)
	}
	defer tx.Rollback()

	var nextSeq int64
	if err := tx.QueryRow(`SELECT COALESCE(MAX(seq), 0) + 1 FROM scan_events WHERE scan_id = ?`, event.ScanID).Scan(&nextSeq); err != nil {
		return fmt.Errorf("allocating event sequence: %w", err)
	}

	event.Sequence = nextSeq
	if event.Time.IsZero() {
		event.Time = time.Now().UTC()
	}

	payload, err := json.Marshal(event.Data)
	if err != nil {
		return fmt.Errorf("encoding event payload: %w", err)
	}

	if _, err := tx.Exec(`
		INSERT INTO scan_events (scan_id, seq, time, type, module, node_id, edge_id, message, data_json)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		event.ScanID,
		event.Sequence,
		event.Time.UTC().Format(time.RFC3339Nano),
		event.Type,
		event.Module,
		event.NodeID,
		event.EdgeID,
		event.Message,
		string(payload),
	); err != nil {
		return fmt.Errorf("saving event: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing event: %w", err)
	}
	return nil
}

func (s *Store) ListEvents(scanID string, afterSeq int64) ([]ScanEvent, error) {
	rows, err := s.db.Query(`
		SELECT seq, time, type, module, node_id, edge_id, message, data_json
		FROM scan_events
		WHERE scan_id = ? AND seq > ?
		ORDER BY seq ASC
	`, scanID, afterSeq)
	if err != nil {
		return nil, fmt.Errorf("listing events: %w", err)
	}
	defer rows.Close()

	var out []ScanEvent
	for rows.Next() {
		var event ScanEvent
		var rawTime string
		var rawData string
		if err := rows.Scan(&event.Sequence, &rawTime, &event.Type, &event.Module, &event.NodeID, &event.EdgeID, &event.Message, &rawData); err != nil {
			return nil, fmt.Errorf("scanning event: %w", err)
		}
		event.ScanID = scanID
		event.Time, err = time.Parse(time.RFC3339Nano, rawTime)
		if err != nil {
			return nil, fmt.Errorf("parsing event time: %w", err)
		}
		if rawData != "" {
			if err := json.Unmarshal([]byte(rawData), &event.Data); err != nil {
				return nil, fmt.Errorf("decoding event payload: %w", err)
			}
		}
		out = append(out, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating events: %w", err)
	}
	if out == nil {
		out = []ScanEvent{}
	}
	return out, nil
}

func scanFromRow(scan func(dest ...any) error) (*ScanRecord, error) {
	var record ScanRecord
	var rawStarted string
	var rawCompleted sql.NullString
	var rawUpdated string
	var seedsJSON string
	var optionsJSON string
	var healthJSON string
	var graphJSON sql.NullString

	err := scan(
		&record.ID,
		&record.Status,
		&rawStarted,
		&rawCompleted,
		&rawUpdated,
		&seedsJSON,
		&optionsJSON,
		&healthJSON,
		&graphJSON,
		&record.NodeCount,
		&record.EdgeCount,
		&record.ErrorMessage,
	)
	if err != nil {
		return nil, err
	}

	record.StartedAt, err = time.Parse(time.RFC3339Nano, rawStarted)
	if err != nil {
		return nil, fmt.Errorf("parsing started_at: %w", err)
	}
	record.UpdatedAt, err = time.Parse(time.RFC3339Nano, rawUpdated)
	if err != nil {
		return nil, fmt.Errorf("parsing updated_at: %w", err)
	}
	if rawCompleted.Valid && rawCompleted.String != "" {
		completedAt, err := time.Parse(time.RFC3339Nano, rawCompleted.String)
		if err != nil {
			return nil, fmt.Errorf("parsing completed_at: %w", err)
		}
		record.CompletedAt = &completedAt
	}
	if err := json.Unmarshal([]byte(seedsJSON), &record.Seeds); err != nil {
		return nil, fmt.Errorf("decoding seeds: %w", err)
	}
	if err := json.Unmarshal([]byte(optionsJSON), &record.Options); err != nil {
		return nil, fmt.Errorf("decoding scan options: %w", err)
	}
	if err := json.Unmarshal([]byte(healthJSON), &record.Health); err != nil {
		return nil, fmt.Errorf("decoding module health: %w", err)
	}
	if graphJSON.Valid && graphJSON.String != "" {
		record.Graph, err = decodeGraph([]byte(graphJSON.String))
		if err != nil {
			return nil, err
		}
	}
	return &record, nil
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
