// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

package differ

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/0verkilll/i18n/cmd/i18ngen/internal/extractor"
)

func TestDiff_MissingKeys(t *testing.T) {
	sourceKeys := []extractor.ExtractedKey{
		{Key: "a", Kind: extractor.KindStandard},
		{Key: "b", Kind: extractor.KindStandard},
		{Key: "c", Kind: extractor.KindStandard},
	}
	localeData := map[string]interface{}{"a": "val", "b": "val"}

	result := DiffFromData(sourceKeys, localeData)

	if len(result.Missing) != 1 || result.Missing[0] != "c" {
		t.Errorf("expected missing=[c], got %v", result.Missing)
	}
	if len(result.Unused) != 0 {
		t.Errorf("expected no unused keys, got %v", result.Unused)
	}
}

func TestDiff_UnusedKeys(t *testing.T) {
	sourceKeys := []extractor.ExtractedKey{
		{Key: "a", Kind: extractor.KindStandard},
	}
	localeData := map[string]interface{}{
		"a": "val",
		"b": "val",
		"c": "val",
	}

	result := DiffFromData(sourceKeys, localeData)

	if len(result.Missing) != 0 {
		t.Errorf("expected no missing keys, got %v", result.Missing)
	}
	if len(result.Unused) != 2 {
		t.Fatalf("expected 2 unused keys, got %d: %v", len(result.Unused), result.Unused)
	}
	if result.Unused[0] != "b" || result.Unused[1] != "c" {
		t.Errorf("expected unused=[b, c], got %v", result.Unused)
	}
}

func TestDiff_PerfectlyAligned(t *testing.T) {
	sourceKeys := []extractor.ExtractedKey{
		{Key: "greeting", Kind: extractor.KindStandard},
		{Key: "farewell", Kind: extractor.KindStandard},
	}
	localeData := map[string]interface{}{
		"greeting": "Hello",
		"farewell": "Goodbye",
	}

	result := DiffFromData(sourceKeys, localeData)

	if result.HasIssues() {
		t.Errorf("expected no issues, got missing=%v, unused=%v", result.Missing, result.Unused)
	}
}

func TestDiff_StructFieldKeys(t *testing.T) {
	sourceKeys := []extractor.ExtractedKey{
		{Key: "Greeting", Kind: extractor.KindStructField},
		{Key: "Farewell", Kind: extractor.KindStructField},
	}
	localeData := map[string]interface{}{
		"Greeting": "Hello",
		"Farewell": "Goodbye",
	}

	result := DiffFromData(sourceKeys, localeData)

	if result.HasIssues() {
		t.Errorf("expected no issues for struct fields, got missing=%v, unused=%v", result.Missing, result.Unused)
	}
}

func TestDiff_NestedLocaleKeysFlattened(t *testing.T) {
	sourceKeys := []extractor.ExtractedKey{
		{Key: "error.required", Kind: extractor.KindStandard},
		{Key: "error.too_long", Kind: extractor.KindStandard},
		{Key: "greeting", Kind: extractor.KindStandard},
	}
	localeData := map[string]interface{}{
		"error": map[string]interface{}{
			"required": "This field is required",
			"too_long": "Value is too long",
		},
		"greeting": "Hello",
	}

	result := DiffFromData(sourceKeys, localeData)

	if result.HasIssues() {
		t.Errorf("expected no issues with nested keys, got missing=%v, unused=%v", result.Missing, result.Unused)
	}
}

func TestFormatText(t *testing.T) {
	result := &DiffResult{
		Missing: []string{"alpha", "beta"},
		Unused:  []string{"gamma"},
	}

	var buf bytes.Buffer
	if err := FormatText(result, &buf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "missing: alpha\nmissing: beta\nunused: gamma\n"
	if buf.String() != want {
		t.Errorf("text output mismatch\ngot:  %q\nwant: %q", buf.String(), want)
	}
}

func TestFormatJSON(t *testing.T) {
	result := &DiffResult{
		Missing: []string{"alpha"},
		Unused:  []string{"beta", "gamma"},
	}

	var buf bytes.Buffer
	if err := FormatJSON(result, &buf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed DiffResult
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	if len(parsed.Missing) != 1 || parsed.Missing[0] != "alpha" {
		t.Errorf("expected missing=[alpha], got %v", parsed.Missing)
	}
	if len(parsed.Unused) != 2 || parsed.Unused[0] != "beta" || parsed.Unused[1] != "gamma" {
		t.Errorf("expected unused=[beta, gamma], got %v", parsed.Unused)
	}
}

func TestFormatJSON_EmptyResult(t *testing.T) {
	result := &DiffResult{}

	var buf bytes.Buffer
	if err := FormatJSON(result, &buf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed DiffResult
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	if parsed.Missing == nil || len(parsed.Missing) != 0 {
		t.Errorf("expected empty missing array, got %v", parsed.Missing)
	}
	if parsed.Unused == nil || len(parsed.Unused) != 0 {
		t.Errorf("expected empty unused array, got %v", parsed.Unused)
	}
}

func TestDiff_FromFile(t *testing.T) {
	tmpDir := t.TempDir()
	localeFile := filepath.Join(tmpDir, "en-US.json")

	localeJSON := `{
		"greeting": "Hello",
		"error": {
			"required": "Required"
		}
	}`
	if err := os.WriteFile(localeFile, []byte(localeJSON), 0o644); err != nil {
		t.Fatalf("writing locale file: %v", err)
	}

	sourceKeys := []extractor.ExtractedKey{
		{Key: "greeting", Kind: extractor.KindStandard},
		{Key: "error.required", Kind: extractor.KindStandard},
		{Key: "error.not_found", Kind: extractor.KindStandard},
	}

	result, err := Diff(sourceKeys, localeFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Missing) != 1 || result.Missing[0] != "error.not_found" {
		t.Errorf("expected missing=[error.not_found], got %v", result.Missing)
	}
	if len(result.Unused) != 0 {
		t.Errorf("expected no unused keys, got %v", result.Unused)
	}
}

func TestDiff_FileNotFound(t *testing.T) {
	sourceKeys := []extractor.ExtractedKey{{Key: "a"}}
	_, err := Diff(sourceKeys, "/nonexistent/file.json")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestLoadLocaleFlat(t *testing.T) {
	tmpDir := t.TempDir()
	localeFile := filepath.Join(tmpDir, "locale.json")

	localeJSON := `{
		"greeting": "Hello",
		"error": {
			"required": "Required",
			"count": 42
		},
		"active": true
	}`
	if err := os.WriteFile(localeFile, []byte(localeJSON), 0o644); err != nil {
		t.Fatalf("writing locale file: %v", err)
	}

	flat, err := LoadLocaleFlat(localeFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := map[string]string{
		"greeting":       "Hello",
		"error.required": "Required",
		"error.count":    "42",
		"active":         "true",
	}

	for k, v := range want {
		if flat[k] != v {
			t.Errorf("key %q: got %q, want %q", k, flat[k], v)
		}
	}
}

func FuzzDiffKeys(f *testing.F) {
	f.Add("a,b,c", "b,c,d")
	f.Add("", "x")
	f.Add("x", "")
	f.Add("", "")
	f.Add("alpha,beta", "alpha,beta")

	f.Fuzz(func(t *testing.T, sourceCSV, localeCSV string) {
		sourceKeys := csvToExtractedKeys(sourceCSV)
		localeData := csvToLocaleData(localeCSV)
		result := DiffFromData(sourceKeys, localeData)

		// Verify consistency: every key must appear in exactly one of:
		// both sets (aligned), missing only, or unused only.
		if result == nil {
			t.Fatal("result must not be nil")
		}
		for _, m := range result.Missing {
			if m == "" {
				continue
			}
			if _, ok := localeData[m]; ok {
				t.Errorf("key %q is in missing but also in locale data", m)
			}
		}
		for _, u := range result.Unused {
			if u == "" {
				continue
			}
			found := false
			for _, sk := range sourceKeys {
				if sk.Key == u {
					found = true
					break
				}
			}
			if found {
				t.Errorf("key %q is in unused but also in source keys", u)
			}
		}
	})
}

// csvToExtractedKeys splits a comma-separated string into extracted keys.
func csvToExtractedKeys(csv string) []extractor.ExtractedKey {
	if csv == "" {
		return nil
	}
	var keys []extractor.ExtractedKey
	start := 0
	for i := 0; i <= len(csv); i++ {
		if i == len(csv) || csv[i] == ',' {
			part := csv[start:i]
			if part != "" {
				keys = append(keys, extractor.ExtractedKey{Key: part})
			}
			start = i + 1
		}
	}
	return keys
}

// csvToLocaleData splits a comma-separated string into a locale data map.
func csvToLocaleData(csv string) map[string]interface{} {
	if csv == "" {
		return map[string]interface{}{}
	}
	data := make(map[string]interface{})
	start := 0
	for i := 0; i <= len(csv); i++ {
		if i == len(csv) || csv[i] == ',' {
			part := csv[start:i]
			if part != "" {
				data[part] = ""
			}
			start = i + 1
		}
	}
	return data
}
