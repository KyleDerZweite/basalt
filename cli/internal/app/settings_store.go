// SPDX-License-Identifier: AGPL-3.0-or-later

package app

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

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
