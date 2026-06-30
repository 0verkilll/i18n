// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

package analyzer

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/0verkilll/i18n"
)

// localeKeyData holds the merged results of loading and flattening all locale files.
type localeKeyData struct {
	// keys maps each dot-notation key to the set of locale files that contain it.
	keys map[string]map[string]bool

	// values maps each dot-notation key to its leaf string value (from the first locale found).
	values map[string]string

	// specifierCounts maps each dot-notation key to the number of format specifiers in its value.
	specifierCounts map[string]int

	// topLevelKeys maps each top-level JSON key to the set of locale files that contain it.
	topLevelKeys map[string]map[string]bool
}

// newLocaleKeyData creates an empty localeKeyData.
func newLocaleKeyData() *localeKeyData {
	return &localeKeyData{
		keys:            make(map[string]map[string]bool),
		values:          make(map[string]string),
		specifierCounts: make(map[string]int),
		topLevelKeys:    make(map[string]map[string]bool),
	}
}

// loadLocaleFiles reads and flattens all JSON translation files from the given paths.
// Each path may be a directory (all .json files inside are loaded) or a single file.
func loadLocaleFiles(paths []string) (*localeKeyData, error) {
	data := newLocaleKeyData()

	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			return nil, fmt.Errorf("accessing locale path %q: %w", p, err)
		}

		if info.IsDir() {
			if err := loadLocaleDir(p, data); err != nil {
				return nil, err
			}
		} else {
			if err := loadLocaleFile(p, data); err != nil {
				return nil, err
			}
		}
	}

	return data, nil
}

// loadLocaleDir loads all .json files from a directory.
func loadLocaleDir(dir string, data *localeKeyData) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("reading locale directory %q: %w", dir, err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		filePath := filepath.Join(dir, entry.Name())
		if err := loadLocaleFile(filePath, data); err != nil {
			return err
		}
	}

	return nil
}

// loadLocaleFile parses a single JSON locale file and merges its keys into data.
func loadLocaleFile(filePath string, data *localeKeyData) error {
	cleanPath := filepath.Clean(filePath)
	raw, err := os.ReadFile(cleanPath)
	if err != nil {
		return fmt.Errorf("reading locale file %q: %w", filePath, err)
	}

	p := i18n.NewJSONParser()
	parsed, err := p.Parse(raw)
	if err != nil {
		return fmt.Errorf("parsing locale file %q: %w", filePath, err)
	}

	for key := range parsed {
		if data.topLevelKeys[key] == nil {
			data.topLevelKeys[key] = make(map[string]bool)
		}
		data.topLevelKeys[key][filePath] = true
	}

	flattenKeys(parsed, "", filePath, data)

	return nil
}

// flattenKeys recursively flattens a nested map into dot-notation leaf keys.
// Leaf node types: string, int, float64, bool, nil (matching convertToString in resolver.go).
func flattenKeys(m map[string]interface{}, prefix, filePath string, data *localeKeyData) {
	for key, value := range m {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}

		switch v := value.(type) {
		case map[string]interface{}:
			flattenKeys(v, fullKey, filePath, data)
		case []interface{}:
			continue
		default:
			if data.keys[fullKey] == nil {
				data.keys[fullKey] = make(map[string]bool)
			}
			data.keys[fullKey][filePath] = true

			if _, exists := data.values[fullKey]; !exists {
				strVal := leafToString(v)
				data.values[fullKey] = strVal
				data.specifierCounts[fullKey] = countFormatSpecifiers(strVal)
			}
		}
	}
}

// leafToString converts a leaf JSON value to its string representation.
func leafToString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case float64:
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val))
		}
		return fmt.Sprintf("%g", val)
	case bool:
		return fmt.Sprintf("%t", val)
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", val)
	}
}
