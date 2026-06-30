// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

package differ

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/0verkilll/i18n/cmd/i18ngen/internal/extractor"
)

// DiffResult holds the comparison between source keys and locale file keys.
type DiffResult struct {
	Missing []string `json:"missing"` // keys in source but not in locale file
	Unused  []string `json:"unused"`  // keys in locale file but not in source
}

// HasIssues reports whether the diff found any missing or unused keys.
func (r *DiffResult) HasIssues() bool {
	return len(r.Missing) > 0 || len(r.Unused) > 0
}

// Diff compares extracted keys against a locale JSON file.
func Diff(sourceKeys []extractor.ExtractedKey, localeFile string) (*DiffResult, error) {
	localeData, err := loadLocaleFile(localeFile)
	if err != nil {
		return nil, err
	}
	return DiffFromData(sourceKeys, localeData), nil
}

// DiffFromData compares extracted keys against parsed JSON data.
func DiffFromData(sourceKeys []extractor.ExtractedKey, localeData map[string]interface{}) *DiffResult {
	flat := flattenKeys(localeData, "")
	return diffKeys(sourceKeys, flat)
}

// FormatText formats the diff result as human-readable text.
func FormatText(result *DiffResult, w io.Writer) error {
	for _, key := range result.Missing {
		if _, err := fmt.Fprintf(w, "missing: %s\n", key); err != nil {
			return fmt.Errorf("writing text diff: %w", err)
		}
	}
	for _, key := range result.Unused {
		if _, err := fmt.Fprintf(w, "unused: %s\n", key); err != nil {
			return fmt.Errorf("writing text diff: %w", err)
		}
	}
	return nil
}

// FormatJSON formats the diff result as JSON.
func FormatJSON(result *DiffResult, w io.Writer) error {
	out := DiffResult{
		Missing: result.Missing,
		Unused:  result.Unused,
	}
	if out.Missing == nil {
		out.Missing = []string{}
	}
	if out.Unused == nil {
		out.Unused = []string{}
	}

	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		return fmt.Errorf("writing JSON diff: %w", err)
	}
	return nil
}

// diffKeys computes the missing and unused key sets.
func diffKeys(sourceKeys []extractor.ExtractedKey, localeKeys map[string]bool) *DiffResult {
	sourceSet := buildSourceKeySet(sourceKeys)

	var missing []string
	for key := range sourceSet {
		if !localeKeys[key] {
			missing = append(missing, key)
		}
	}
	sort.Strings(missing)

	var unused []string
	for key := range localeKeys {
		if !sourceSet[key] {
			unused = append(unused, key)
		}
	}
	sort.Strings(unused)

	return &DiffResult{
		Missing: missing,
		Unused:  unused,
	}
}

// buildSourceKeySet creates a set of canonical key names from extracted keys.
// Struct field keys (KindStructField) are included by their Key field directly,
// which the extractor stores as the plain field name without any prefix.
func buildSourceKeySet(keys []extractor.ExtractedKey) map[string]bool {
	set := make(map[string]bool, len(keys))
	for _, k := range keys {
		set[k.Key] = true
	}
	return set
}

// loadLocaleFile reads a JSON locale file and returns the raw parsed data.
func loadLocaleFile(filePath string) (map[string]interface{}, error) {
	cleanPath := filepath.Clean(filePath)
	raw, err := os.ReadFile(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("loading locale file %q: %w", filePath, err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, fmt.Errorf("parsing locale file %q: %w", filePath, err)
	}

	return data, nil
}

// flattenKeys recursively flattens a nested map into dot-notation leaf keys.
func flattenKeys(m map[string]interface{}, prefix string) map[string]bool {
	result := make(map[string]bool)
	flattenKeysInto(m, prefix, result)
	return result
}

// flattenKeysInto is the recursive helper for flattenKeys.
func flattenKeysInto(m map[string]interface{}, prefix string, result map[string]bool) {
	for key, value := range m {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}

		nested, ok := toStringMap(value)
		if ok {
			flattenKeysInto(nested, fullKey, result)
			continue
		}

		result[fullKey] = true
	}
}

// toStringMap attempts to cast a value to map[string]interface{}.
func toStringMap(v interface{}) (map[string]interface{}, bool) {
	m, ok := v.(map[string]interface{})
	return m, ok
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

// LoadLocaleFlat reads a JSON locale file and returns a flat map of key to value.
// This is useful for CLI consumers that need the actual translation values.
func LoadLocaleFlat(filePath string) (map[string]string, error) {
	data, err := loadLocaleFile(filePath)
	if err != nil {
		return nil, err
	}

	flat := make(map[string]string)
	flattenValuesInto(data, "", flat)
	return flat, nil
}

// flattenValuesInto recursively flattens nested keys and collects leaf string values.
func flattenValuesInto(m map[string]interface{}, prefix string, result map[string]string) {
	for key, value := range m {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}

		nested, ok := toStringMap(value)
		if ok {
			flattenValuesInto(nested, fullKey, result)
			continue
		}

		result[fullKey] = leafToString(value)
	}
}
