// SPDX-License-Identifier: AGPL-3.0-or-later

package app

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

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
