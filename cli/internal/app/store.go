// SPDX-License-Identifier: AGPL-3.0-or-later

package app

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
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

func (s *Store) LoadModuleHealthCache(version, configHash string, moduleNames []string, now time.Time) (map[string]ModuleHealthCacheEntry, error) {
	if len(moduleNames) == 0 {
		return map[string]ModuleHealthCacheEntry{}, nil
	}

	placeholders := strings.TrimRight(strings.Repeat("?,", len(moduleNames)), ",")
	args := make([]any, 0, len(moduleNames)+3)
	args = append(args, version, configHash, now.UTC().Format(time.RFC3339Nano))
	for _, name := range moduleNames {
		args = append(args, name)
	}

	rows, err := s.db.Query(fmt.Sprintf(`
		SELECT module_name, basalt_version, config_hash, status, message, checked_at, expires_at
		FROM module_health_cache
		WHERE basalt_version = ? AND config_hash = ? AND expires_at > ? AND module_name IN (%s)
	`, placeholders), args...)
	if err != nil {
		return nil, fmt.Errorf("listing module health cache: %w", err)
	}
	defer rows.Close()

	out := make(map[string]ModuleHealthCacheEntry, len(moduleNames))
	for rows.Next() {
		var entry ModuleHealthCacheEntry
		var rawChecked string
		var rawExpires string
		if err := rows.Scan(&entry.ModuleName, &entry.Version, &entry.ConfigHash, &entry.Status, &entry.Message, &rawChecked, &rawExpires); err != nil {
			return nil, fmt.Errorf("scanning module health cache: %w", err)
		}
		entry.CheckedAt, err = time.Parse(time.RFC3339Nano, rawChecked)
		if err != nil {
			return nil, fmt.Errorf("parsing cached module checked_at: %w", err)
		}
		entry.ExpiresAt, err = time.Parse(time.RFC3339Nano, rawExpires)
		if err != nil {
			return nil, fmt.Errorf("parsing cached module expires_at: %w", err)
		}
		out[entry.ModuleName] = entry
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating module health cache: %w", err)
	}
	return out, nil
}

func (s *Store) SaveModuleHealthCache(entries []ModuleHealthCacheEntry) error {
	if len(entries) == 0 {
		return nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("starting module health cache transaction: %w", err)
	}
	defer tx.Rollback()

	for _, entry := range entries {
		if _, err := tx.Exec(`
			INSERT INTO module_health_cache (
				module_name, basalt_version, config_hash, status, message, checked_at, expires_at
			) VALUES (?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(module_name, basalt_version, config_hash) DO UPDATE SET
				status = excluded.status,
				message = excluded.message,
				checked_at = excluded.checked_at,
				expires_at = excluded.expires_at
		`,
			entry.ModuleName,
			entry.Version,
			entry.ConfigHash,
			entry.Status,
			entry.Message,
			entry.CheckedAt.UTC().Format(time.RFC3339Nano),
			entry.ExpiresAt.UTC().Format(time.RFC3339Nano),
		); err != nil {
			return fmt.Errorf("saving module health cache: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing module health cache transaction: %w", err)
	}
	return nil
}

func (s *Store) ClearModuleHealthCache() error {
	if _, err := s.db.Exec(`DELETE FROM module_health_cache`); err != nil {
		return fmt.Errorf("clearing module health cache: %w", err)
	}
	return nil
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
	var insightsJSON any
	if record.Insights != nil {
		payload, err := json.Marshal(record.Insights)
		if err != nil {
			return fmt.Errorf("encoding scan insights: %w", err)
		}
		insightsJSON = string(payload)
	}

	_, err = s.db.Exec(`
		INSERT INTO scans (
			id, target_id, status, started_at, completed_at, updated_at, seeds_json, options_json,
			health_json, insights_json, graph_json, node_count, edge_count, error_message
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		record.ID,
		record.TargetID,
		string(record.Status),
		record.StartedAt.UTC().Format(time.RFC3339Nano),
		timeString(record.CompletedAt),
		record.UpdatedAt.UTC().Format(time.RFC3339Nano),
		string(seedsJSON),
		string(optionsJSON),
		string(healthJSON),
		insightsJSON,
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
	var insightsJSON any
	if record.Insights != nil {
		payload, err := json.Marshal(record.Insights)
		if err != nil {
			return fmt.Errorf("encoding scan insights: %w", err)
		}
		insightsJSON = string(payload)
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
		SET target_id = ?, status = ?, completed_at = ?, updated_at = ?, seeds_json = ?, options_json = ?,
		    health_json = ?, insights_json = ?, graph_json = ?, node_count = ?, edge_count = ?, error_message = ?
		WHERE id = ?
	`,
		record.TargetID,
		string(record.Status),
		timeString(record.CompletedAt),
		record.UpdatedAt.UTC().Format(time.RFC3339Nano),
		string(seedsJSON),
		string(optionsJSON),
		string(healthJSON),
		insightsJSON,
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
		SELECT id, target_id, status, started_at, completed_at, updated_at, seeds_json, options_json,
		       health_json, insights_json, graph_json, node_count, edge_count, error_message
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
		SELECT id, target_id, status, started_at, completed_at, updated_at, seeds_json, options_json,
		       health_json, insights_json, node_count, edge_count, error_message
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
		record, err := scanSummaryFromRow(rows.Scan)
		if err != nil {
			return nil, err
		}
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

// ListScansByTarget returns scans associated with a target.
func (s *Store) ListScansByTarget(targetID string, limit int) ([]*ScanRecord, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.Query(`
		SELECT id, target_id, status, started_at, completed_at, updated_at, seeds_json, options_json,
		       health_json, insights_json, node_count, edge_count, error_message
		FROM scans
		WHERE target_id = ?
		ORDER BY started_at DESC
		LIMIT ?
	`, targetID, limit)
	if err != nil {
		return nil, fmt.Errorf("listing target scans: %w", err)
	}
	defer rows.Close()

	var out []*ScanRecord
	for rows.Next() {
		record, err := scanSummaryFromRow(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating target scans: %w", err)
	}
	if out == nil {
		out = []*ScanRecord{}
	}
	return out, nil
}

// CreateTarget persists a target.
func (s *Store) CreateTarget(target *Target) error {
	_, err := s.db.Exec(`
		INSERT INTO targets (id, slug, display_name, notes, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, target.ID, target.Slug, target.DisplayName, target.Notes, target.CreatedAt.UTC().Format(time.RFC3339Nano), target.UpdatedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("creating target: %w", err)
	}
	return nil
}

// UpdateTarget persists target changes.
func (s *Store) UpdateTarget(target *Target) error {
	_, err := s.db.Exec(`
		UPDATE targets
		SET slug = ?, display_name = ?, notes = ?, updated_at = ?
		WHERE id = ?
	`, target.Slug, target.DisplayName, target.Notes, target.UpdatedAt.UTC().Format(time.RFC3339Nano), target.ID)
	if err != nil {
		return fmt.Errorf("updating target: %w", err)
	}
	return nil
}

// DeleteTarget removes a target and its aliases.
func (s *Store) DeleteTarget(targetID string) error {
	_, err := s.db.Exec(`DELETE FROM targets WHERE id = ?`, targetID)
	if err != nil {
		return fmt.Errorf("deleting target: %w", err)
	}
	return nil
}

// GetTarget loads a target by id or slug.
func (s *Store) GetTarget(ref string) (*Target, error) {
	row := s.db.QueryRow(`
		SELECT id, slug, display_name, notes, created_at, updated_at
		FROM targets
		WHERE id = ? OR slug = ?
		LIMIT 1
	`, ref, ref)
	target, err := targetFromRow(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("target %s not found", ref)
	}
	if err != nil {
		return nil, err
	}
	target.Aliases, err = s.listTargetAliases(target.ID)
	if err != nil {
		return nil, err
	}
	return target, nil
}

// ListTargets returns all persisted targets with aliases.
func (s *Store) ListTargets() ([]*Target, error) {
	rows, err := s.db.Query(`
		SELECT id, slug, display_name, notes, created_at, updated_at
		FROM targets
		ORDER BY display_name ASC, slug ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("listing targets: %w", err)
	}
	defer rows.Close()

	var out []*Target
	for rows.Next() {
		target, err := targetFromRow(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, target)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating targets: %w", err)
	}
	aliasesByTarget, err := s.listAllTargetAliases()
	if err != nil {
		return nil, err
	}
	for _, target := range out {
		target.Aliases = aliasesByTarget[target.ID]
		if target.Aliases == nil {
			target.Aliases = []TargetAlias{}
		}
	}
	if out == nil {
		out = []*Target{}
	}
	return out, nil
}

// AddTargetAlias adds a seed alias to a target.
func (s *Store) AddTargetAlias(alias *TargetAlias) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("starting alias transaction: %w", err)
	}
	defer tx.Rollback()

	if alias.IsPrimary {
		if _, err := tx.Exec(`UPDATE target_aliases SET is_primary = 0 WHERE target_id = ?`, alias.TargetID); err != nil {
			return fmt.Errorf("clearing primary alias: %w", err)
		}
	}

	if _, err := tx.Exec(`
		INSERT INTO target_aliases (id, target_id, seed_type, seed_value, label, is_primary, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, alias.ID, alias.TargetID, alias.SeedType, alias.SeedValue, alias.Label, boolToInt(alias.IsPrimary), alias.CreatedAt.UTC().Format(time.RFC3339Nano)); err != nil {
		return fmt.Errorf("creating target alias: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing alias transaction: %w", err)
	}
	return nil
}

// RemoveTargetAlias removes a persisted alias from a target.
func (s *Store) RemoveTargetAlias(targetID, aliasID string) error {
	_, err := s.db.Exec(`DELETE FROM target_aliases WHERE id = ? AND target_id = ?`, aliasID, targetID)
	if err != nil {
		return fmt.Errorf("removing target alias: %w", err)
	}
	return nil
}

func (s *Store) listTargetAliases(targetID string) ([]TargetAlias, error) {
	rows, err := s.db.Query(`
		SELECT id, target_id, seed_type, seed_value, label, is_primary, created_at
		FROM target_aliases
		WHERE target_id = ?
		ORDER BY is_primary DESC, created_at ASC, seed_type ASC, seed_value ASC
	`, targetID)
	if err != nil {
		return nil, fmt.Errorf("listing target aliases: %w", err)
	}
	defer rows.Close()

	var out []TargetAlias
	for rows.Next() {
		var alias TargetAlias
		var rawCreated string
		var isPrimary int
		if err := rows.Scan(&alias.ID, &alias.TargetID, &alias.SeedType, &alias.SeedValue, &alias.Label, &isPrimary, &rawCreated); err != nil {
			return nil, fmt.Errorf("scanning target alias: %w", err)
		}
		alias.IsPrimary = isPrimary == 1
		alias.CreatedAt, err = time.Parse(time.RFC3339Nano, rawCreated)
		if err != nil {
			return nil, fmt.Errorf("parsing alias created_at: %w", err)
		}
		out = append(out, alias)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating target aliases: %w", err)
	}
	if out == nil {
		out = []TargetAlias{}
	}
	return out, nil
}

func (s *Store) AppendEvent(event *ScanEvent) error {
	s.eventMu.Lock()
	defer s.eventMu.Unlock()

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
	var targetID string
	var rawStarted string
	var rawCompleted sql.NullString
	var rawUpdated string
	var seedsJSON string
	var optionsJSON string
	var healthJSON string
	var insightsJSON sql.NullString
	var graphJSON sql.NullString

	err := scan(
		&record.ID,
		&targetID,
		&record.Status,
		&rawStarted,
		&rawCompleted,
		&rawUpdated,
		&seedsJSON,
		&optionsJSON,
		&healthJSON,
		&insightsJSON,
		&graphJSON,
		&record.NodeCount,
		&record.EdgeCount,
		&record.ErrorMessage,
	)
	if err != nil {
		return nil, err
	}
	record.TargetID = targetID

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
	if insightsJSON.Valid && insightsJSON.String != "" {
		var insights ScanInsights
		if err := json.Unmarshal([]byte(insightsJSON.String), &insights); err != nil {
			return nil, fmt.Errorf("decoding scan insights: %w", err)
		}
		record.Insights = &insights
	}
	if graphJSON.Valid && graphJSON.String != "" {
		record.Graph, err = decodeGraph([]byte(graphJSON.String))
		if err != nil {
			return nil, err
		}
	}
	return &record, nil
}

func scanSummaryFromRow(scan func(dest ...any) error) (*ScanRecord, error) {
	var record ScanRecord
	var targetID string
	var rawStarted string
	var rawCompleted sql.NullString
	var rawUpdated string
	var seedsJSON string
	var optionsJSON string
	var healthJSON string
	var insightsJSON sql.NullString

	err := scan(
		&record.ID,
		&targetID,
		&record.Status,
		&rawStarted,
		&rawCompleted,
		&rawUpdated,
		&seedsJSON,
		&optionsJSON,
		&healthJSON,
		&insightsJSON,
		&record.NodeCount,
		&record.EdgeCount,
		&record.ErrorMessage,
	)
	if err != nil {
		return nil, err
	}
	record.TargetID = targetID

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
	if insightsJSON.Valid && insightsJSON.String != "" {
		var insights ScanInsights
		if err := json.Unmarshal([]byte(insightsJSON.String), &insights); err != nil {
			return nil, fmt.Errorf("decoding scan insights: %w", err)
		}
		record.Insights = &insights
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

func (s *Store) listAllTargetAliases() (map[string][]TargetAlias, error) {
	rows, err := s.db.Query(`
		SELECT id, target_id, seed_type, seed_value, label, is_primary, created_at
		FROM target_aliases
		ORDER BY target_id ASC, is_primary DESC, created_at ASC, seed_type ASC, seed_value ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("listing target aliases: %w", err)
	}
	defer rows.Close()

	out := make(map[string][]TargetAlias)
	for rows.Next() {
		var alias TargetAlias
		var rawCreated string
		var isPrimary int
		if err := rows.Scan(&alias.ID, &alias.TargetID, &alias.SeedType, &alias.SeedValue, &alias.Label, &isPrimary, &rawCreated); err != nil {
			return nil, fmt.Errorf("scanning target alias: %w", err)
		}
		alias.IsPrimary = isPrimary == 1
		alias.CreatedAt, err = time.Parse(time.RFC3339Nano, rawCreated)
		if err != nil {
			return nil, fmt.Errorf("parsing alias created_at: %w", err)
		}
		out[alias.TargetID] = append(out[alias.TargetID], alias)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating target aliases: %w", err)
	}
	for targetID, aliases := range out {
		slices.SortStableFunc(aliases, func(a, b TargetAlias) int {
			switch {
			case a.IsPrimary && !b.IsPrimary:
				return -1
			case !a.IsPrimary && b.IsPrimary:
				return 1
			case a.CreatedAt.Before(b.CreatedAt):
				return -1
			case a.CreatedAt.After(b.CreatedAt):
				return 1
			case a.SeedType < b.SeedType:
				return -1
			case a.SeedType > b.SeedType:
				return 1
			case a.SeedValue < b.SeedValue:
				return -1
			case a.SeedValue > b.SeedValue:
				return 1
			default:
				return 0
			}
		})
		out[targetID] = aliases
	}
	return out, nil
}

func targetFromRow(scan func(dest ...any) error) (*Target, error) {
	var target Target
	var rawCreated string
	var rawUpdated string
	if err := scan(&target.ID, &target.Slug, &target.DisplayName, &target.Notes, &rawCreated, &rawUpdated); err != nil {
		return nil, err
	}
	var err error
	target.CreatedAt, err = time.Parse(time.RFC3339Nano, rawCreated)
	if err != nil {
		return nil, fmt.Errorf("parsing target created_at: %w", err)
	}
	target.UpdatedAt, err = time.Parse(time.RFC3339Nano, rawUpdated)
	if err != nil {
		return nil, fmt.Errorf("parsing target updated_at: %w", err)
	}
	return &target, nil
}

func timeString(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func tempDBPath(dir string) string {
	return filepath.Join(dir, "basalt.db")
}
