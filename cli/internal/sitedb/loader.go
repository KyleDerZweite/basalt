// SPDX-License-Identifier: AGPL-3.0-or-later

package sitedb

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// LoadSites loads all YAML site definition files from one or more directories.
// Files must have a .yaml or .yml extension. Definitions are merged across files.
// Duplicates (by name or URL template) are resolved by keeping the first encountered.
func LoadSites(dirs ...string) ([]SiteDefinition, error) {
	seenName := make(map[string]struct{})
	seenURL := make(map[string]struct{})
	var sites []SiteDefinition

	for _, dir := range dirs {
		info, err := os.Stat(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue // skip missing directories silently
			}
			return nil, fmt.Errorf("stat %s: %w", dir, err)
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("%s is not a directory", dir)
		}

		err = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}

			ext := strings.ToLower(filepath.Ext(path))
			if ext != ".yaml" && ext != ".yml" {
				return nil
			}

			loaded, err := loadFile(path)
			if err != nil {
				return fmt.Errorf("loading %s: %w", path, err)
			}

			for _, site := range loaded {
				if site.Disabled {
					continue
				}
				nameKey := strings.ToLower(site.Name)
				urlKey := strings.ToLower(site.URLTemplate)
				if _, dup := seenName[nameKey]; dup {
					continue
				}
				if _, dup := seenURL[urlKey]; dup {
					continue
				}
				seenName[nameKey] = struct{}{}
				seenURL[urlKey] = struct{}{}
				sites = append(sites, site)
			}

			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("walking %s: %w", dir, err)
		}
	}

	return sites, nil
}

// loadFile parses a single YAML file into site definitions.
func loadFile(path string) ([]SiteDefinition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var sf SiteFile
	if err := yaml.Unmarshal(data, &sf); err != nil {
		return nil, fmt.Errorf("parsing YAML: %w", err)
	}

	return sf.Sites, nil
}

// WriteSites writes site definitions to a YAML file.
func WriteSites(path string, sites []SiteDefinition) error {
	sf := SiteFile{Sites: sites}

	data, err := yaml.Marshal(&sf)
	if err != nil {
		return fmt.Errorf("marshaling YAML: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	return os.WriteFile(path, data, 0o644)
}
