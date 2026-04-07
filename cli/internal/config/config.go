// SPDX-License-Identifier: AGPL-3.0-or-later

package config

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"sort"
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

// Fingerprint returns a stable hash of the loaded config values.
func (c *Config) Fingerprint() string {
	if c == nil || len(c.values) == 0 {
		sum := sha256.Sum256(nil)
		return hex.EncodeToString(sum[:])
	}

	keys := make([]string, 0, len(c.values))
	for key := range c.values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var builder strings.Builder
	for _, key := range keys {
		builder.WriteString(key)
		builder.WriteByte('=')
		builder.WriteString(c.values[key])
		builder.WriteByte('\n')
	}

	sum := sha256.Sum256([]byte(builder.String()))
	return hex.EncodeToString(sum[:])
}
