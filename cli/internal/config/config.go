// SPDX-License-Identifier: AGPL-3.0-or-later

package config

import (
	"bufio"
	"os"
	"strings"
)

// Config holds key-value pairs loaded from a config file.
type Config struct {
	values map[string]string
}

// Load reads a config file in KEY=VALUE format.
// Blank lines and lines starting with # are ignored.
// If the file does not exist, returns an empty Config (not an error).
func Load(path string) (*Config, error) {
	cfg := &Config{values: make(map[string]string)}

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		// Strip surrounding quotes.
		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') ||
				(value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}

		cfg.values[key] = value
	}

	return cfg, scanner.Err()
}

// Get returns the value for a key, or empty string if not set.
func (c *Config) Get(key string) string {
	return c.values[key]
}
