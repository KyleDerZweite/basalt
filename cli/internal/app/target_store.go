// SPDX-License-Identifier: AGPL-3.0-or-later

package app

import (
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"time"
)

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
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
