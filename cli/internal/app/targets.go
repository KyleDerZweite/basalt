// SPDX-License-Identifier: AGPL-3.0-or-later

package app

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/KyleDerZweite/basalt/internal/graph"
)

// CreateTarget persists a new target.
func (s *Service) CreateTarget(target Target) (*Target, error) {
	target.DisplayName = strings.TrimSpace(target.DisplayName)
	target.Notes = strings.TrimSpace(target.Notes)
	target.Slug = normalizeSlug(target.Slug)
	if target.DisplayName == "" {
		return nil, fmt.Errorf("display_name is required")
	}
	if target.Slug == "" {
		target.Slug = normalizeSlug(target.DisplayName)
	}
	if target.Slug == "" {
		return nil, fmt.Errorf("slug is required")
	}

	now := time.Now().UTC()
	target.ID = uuid.New().String()
	target.CreatedAt = now
	target.UpdatedAt = now
	target.Aliases = nil
	if err := s.store.CreateTarget(&target); err != nil {
		return nil, err
	}
	return s.store.GetTarget(target.ID)
}

// ListTargets returns all persisted targets.
func (s *Service) ListTargets() ([]*Target, error) {
	return s.store.ListTargets()
}

// GetTarget returns a persisted target by slug or ID.
func (s *Service) GetTarget(ref string) (*Target, error) {
	return s.store.GetTarget(strings.TrimSpace(ref))
}

// UpdateTarget updates a persisted target.
func (s *Service) UpdateTarget(ref string, updates Target) (*Target, error) {
	target, err := s.GetTarget(ref)
	if err != nil {
		return nil, err
	}
	if trimmed := strings.TrimSpace(updates.DisplayName); trimmed != "" {
		target.DisplayName = trimmed
	}
	if updates.Slug != "" {
		target.Slug = normalizeSlug(updates.Slug)
		if target.Slug == "" {
			return nil, fmt.Errorf("slug is required")
		}
	}
	target.Notes = strings.TrimSpace(updates.Notes)
	target.UpdatedAt = time.Now().UTC()
	if err := s.store.UpdateTarget(target); err != nil {
		return nil, err
	}
	return s.store.GetTarget(target.ID)
}

// DeleteTarget removes a persisted target.
func (s *Service) DeleteTarget(ref string) error {
	target, err := s.GetTarget(ref)
	if err != nil {
		return err
	}
	return s.store.DeleteTarget(target.ID)
}

// AddTargetAlias adds a normalized alias to a target.
func (s *Service) AddTargetAlias(ref string, seed graph.Seed, label string, primary bool) (*TargetAlias, error) {
	target, err := s.GetTarget(ref)
	if err != nil {
		return nil, err
	}
	seed, err = normalizeSeed(seed)
	if err != nil {
		return nil, err
	}

	alias := &TargetAlias{
		ID:        uuid.New().String(),
		TargetID:  target.ID,
		SeedType:  seed.Type,
		SeedValue: seed.Value,
		Label:     strings.TrimSpace(label),
		IsPrimary: primary,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.store.AddTargetAlias(alias); err != nil {
		return nil, err
	}
	target, err = s.store.GetTarget(target.ID)
	if err != nil {
		return nil, err
	}
	for _, candidate := range target.Aliases {
		if candidate.ID == alias.ID {
			return &candidate, nil
		}
	}
	return alias, nil
}

// RemoveTargetAlias removes an alias from a target.
func (s *Service) RemoveTargetAlias(ref, aliasID string) error {
	target, err := s.GetTarget(ref)
	if err != nil {
		return err
	}
	if strings.TrimSpace(aliasID) == "" {
		return fmt.Errorf("alias id is required")
	}
	return s.store.RemoveTargetAlias(target.ID, aliasID)
}

// ListTargetScans returns scans associated with a target.
func (s *Service) ListTargetScans(ref string, limit int) ([]*ScanRecord, error) {
	target, err := s.GetTarget(ref)
	if err != nil {
		return nil, err
	}
	return s.store.ListScansByTarget(target.ID, limit)
}

func (s *Service) resolveTargetSeeds(targetRef string) (string, []graph.Seed, error) {
	targetRef = strings.TrimSpace(targetRef)
	if targetRef == "" {
		return "", nil, nil
	}
	target, err := s.GetTarget(targetRef)
	if err != nil {
		return "", nil, err
	}

	seeds := make([]graph.Seed, 0, len(target.Aliases))
	for _, alias := range target.Aliases {
		seed, err := normalizeSeed(graph.Seed{Type: alias.SeedType, Value: alias.SeedValue})
		if err != nil {
			return "", nil, err
		}
		seeds = append(seeds, seed)
	}
	return target.ID, dedupeSeeds(seeds), nil
}

func dedupeSeeds(seeds []graph.Seed) []graph.Seed {
	if len(seeds) == 0 {
		return nil
	}
	seen := make(map[string]graph.Seed, len(seeds))
	for _, seed := range seeds {
		normalized, err := normalizeSeed(seed)
		if err != nil {
			continue
		}
		seen[normalized.Type+":"+normalized.Value] = normalized
	}
	out := make([]graph.Seed, 0, len(seen))
	for _, seed := range seen {
		out = append(out, seed)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Type == out[j].Type {
			return out[i].Value < out[j].Value
		}
		return out[i].Type < out[j].Type
	})
	return out
}

func normalizeSeed(seed graph.Seed) (graph.Seed, error) {
	seed.Type = strings.TrimSpace(seed.Type)
	seed.Value = strings.TrimSpace(seed.Value)
	switch seed.Type {
	case graph.NodeTypeUsername, graph.NodeTypeEmail, graph.NodeTypeDomain:
	default:
		return graph.Seed{}, fmt.Errorf("unsupported alias seed type %q", seed.Type)
	}
	if seed.Value == "" {
		return graph.Seed{}, fmt.Errorf("seed value is required")
	}
	seed.Value = strings.ToLower(seed.Value)
	return seed, nil
}

func normalizeSlug(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return ""
	}
	var builder strings.Builder
	lastDash := false
	for _, r := range value {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			builder.WriteRune(r)
			lastDash = false
		case r == '-' || r == '_' || r == ' ':
			if !lastDash && builder.Len() > 0 {
				builder.WriteByte('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(builder.String(), "-")
}
