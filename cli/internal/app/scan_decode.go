// SPDX-License-Identifier: AGPL-3.0-or-later

package app

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

type scanRowFields struct {
	record    ScanRecord
	started   string
	completed sql.NullString
	updated   string
	seeds     string
	options   string
	health    string
	insights  sql.NullString
	graph     sql.NullString
}

func scanFromRow(scan func(...any) error) (*ScanRecord, error) {
	return scanRecordFromRow(scan, true)
}

func scanSummaryFromRow(scan func(...any) error) (*ScanRecord, error) {
	return scanRecordFromRow(scan, false)
}

func scanRecordFromRow(scan func(...any) error, includeGraph bool) (*ScanRecord, error) {
	f := scanRowFields{}
	dest := []any{&f.record.ID, &f.record.TargetID, &f.record.Status, &f.started, &f.completed,
		&f.updated, &f.seeds, &f.options, &f.health, &f.insights}
	if includeGraph {
		dest = append(dest, &f.graph)
	}
	dest = append(dest, &f.record.NodeCount, &f.record.EdgeCount, &f.record.ErrorMessage)
	if err := scan(dest...); err != nil {
		return nil, err
	}
	if err := decodeScanRow(&f, includeGraph); err != nil {
		return nil, err
	}
	return &f.record, nil
}

func decodeScanRow(f *scanRowFields, includeGraph bool) error {
	var err error
	f.record.StartedAt, err = time.Parse(time.RFC3339Nano, f.started)
	if err != nil {
		return fmt.Errorf("parsing started_at: %w", err)
	}
	f.record.UpdatedAt, err = time.Parse(time.RFC3339Nano, f.updated)
	if err != nil {
		return fmt.Errorf("parsing updated_at: %w", err)
	}
	if f.completed.Valid && f.completed.String != "" {
		completed, err := time.Parse(time.RFC3339Nano, f.completed.String)
		if err != nil {
			return fmt.Errorf("parsing completed_at: %w", err)
		}
		f.record.CompletedAt = &completed
	}
	for _, item := range []struct {
		name string
		raw  string
		out  any
	}{{"seeds", f.seeds, &f.record.Seeds}, {"scan options", f.options, &f.record.Options}, {"module health", f.health, &f.record.Health}} {
		if err := json.Unmarshal([]byte(item.raw), item.out); err != nil {
			return fmt.Errorf("decoding %s: %w", item.name, err)
		}
	}
	if f.insights.Valid && f.insights.String != "" {
		f.record.Insights = &ScanInsights{}
		if err := json.Unmarshal([]byte(f.insights.String), f.record.Insights); err != nil {
			return fmt.Errorf("decoding scan insights: %w", err)
		}
	}
	if includeGraph && f.graph.Valid && f.graph.String != "" {
		f.record.Graph, err = decodeGraph([]byte(f.graph.String))
	}
	return err
}
