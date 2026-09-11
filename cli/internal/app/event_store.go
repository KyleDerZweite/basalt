// SPDX-License-Identifier: AGPL-3.0-or-later

package app

import (
	"encoding/json"
	"fmt"
	"time"
)

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
