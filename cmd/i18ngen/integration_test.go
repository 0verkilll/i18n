// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

package main

import (
	"bytes"
	"encoding/json"
	"go/format"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/0verkilll/i18n"
)

// testdataDir returns the absolute path to the testdata directory.
func testdataDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("unable to determine caller path")
	}
	return filepath.Join(filepath.Dir(file), "testdata")
}

// TestIntegration_ExtractToJSON verifies the extract -> generate JSON round-trip:
// all extracted keys appear as top-level keys in the generated JSON template.
func TestIntegration_ExtractToJSON(t *testing.T) {
	dir := testdataDir(t)

	var stdout, stderr bytes.Buffer
	code := run([]string{"generate", "-format", "json", dir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
	}

	var parsed map[string]string
	if err := json.Unmarshal(stdout.Bytes(), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput:\n%s", err, stdout.String())
	}

	// All 9 call patterns plus struct field keys should appear.
	expectedKeys := []string{
		"greeting",
		"welcome_user",
		"item_count",
		"salutation",
		"farewell",
		"items_format",
		"error.required",
		"optional_feature",
		"nav.home",
		"Greeting",
		"Farewell",
	}
	for _, key := range expectedKeys {
		if _, ok := parsed[key]; !ok {
			t.Errorf("expected key %q in JSON output, not found", key)
		}
	}

	// TD default should be populated.
	if parsed["error.required"] != "This field is required" {
		t.Errorf("expected TD default for error.required, got %q", parsed["error.required"])
	}
}

// TestIntegration_ExtractToBinary verifies the extract -> generate binary round-trip:
// binary output can be decoded back and contains the expected keys.
func TestIntegration_ExtractToBinary(t *testing.T) {
	dir := testdataDir(t)

	var stdout, stderr bytes.Buffer
	code := run([]string{"generate", "-format", "binary", dir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
	}

	parser := i18n.NewBinaryParser()
	parsed, err := parser.Parse(stdout.Bytes())
	if err != nil {
		t.Fatalf("failed to parse binary output: %v", err)
	}

	flat := i18n.FlattenKeys(parsed)

	// Spot-check a few keys.
	checkKeys := []string{"greeting", "farewell", "error.required", "nav.home"}
	for _, key := range checkKeys {
		if _, ok := flat[key]; !ok {
			t.Errorf("expected key %q in binary output, not found", key)
		}
	}

	// TD default should survive the round-trip.
	if flat["error.required"] != "This field is required" {
		t.Errorf("expected TD default for error.required in binary, got %q", flat["error.required"])
	}
}

// TestIntegration_DiffClean verifies that diffing against a locale file that
// contains all source keys produces exit code 0 and no output.
func TestIntegration_DiffClean(t *testing.T) {
	dir := testdataDir(t)
	localeFile := filepath.Join(dir, "locale_complete.json")

	var stdout, stderr bytes.Buffer
	code := run([]string{"diff", "-locale", localeFile, dir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 (clean diff), got %d; stderr: %s; stdout: %s", code, stderr.String(), stdout.String())
	}

	if stdout.Len() != 0 {
		t.Errorf("expected no output for clean diff, got:\n%s", stdout.String())
	}
}

// TestIntegration_DiffWithIssues verifies that diffing against an incomplete
// locale file produces exit code 1 and reports the correct missing/unused keys.
func TestIntegration_DiffWithIssues(t *testing.T) {
	dir := testdataDir(t)
	localeFile := filepath.Join(dir, "locale_incomplete.json")

	var stdout, stderr bytes.Buffer
	code := run([]string{"diff", "-locale", localeFile, dir}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected exit code 1 (issues found), got %d; stderr: %s; stdout: %s", code, stderr.String(), stdout.String())
	}

	output := stdout.String()

	// Keys in source but not in locale_incomplete.json should be missing.
	missingExpected := []string{
		"Farewell",
		"Greeting",
		"item_count",
		"items_format",
		"nav.home",
		"optional_feature",
		"salutation",
		"welcome_user",
	}
	for _, key := range missingExpected {
		if !strings.Contains(output, "missing: "+key) {
			t.Errorf("expected 'missing: %s' in diff output", key)
		}
	}

	// Keys in locale_incomplete.json but not in source should be unused.
	unusedExpected := []string{
		"deprecated_key",
		"legacy_feature",
	}
	for _, key := range unusedExpected {
		if !strings.Contains(output, "unused: "+key) {
			t.Errorf("expected 'unused: %s' in diff output", key)
		}
	}
}

// TestIntegration_InitScaffold verifies the extract -> init scaffold produces
// a compilable Go file.
func TestIntegration_InitScaffold(t *testing.T) {
	dir := testdataDir(t)

	var stdout, stderr bytes.Buffer
	code := run([]string{"init", "-package", "sample", dir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
	}

	output := stdout.Bytes()

	// Must be valid Go source.
	if _, err := format.Source(output); err != nil {
		t.Fatalf("init output is not valid Go: %v\noutput:\n%s", err, string(output))
	}

	src := string(output)
	if !strings.Contains(src, "package sample") {
		t.Error("expected 'package sample' in init output")
	}
	if !strings.Contains(src, `"github.com/0verkilll/i18n"`) {
		t.Error("expected i18n import in init output")
	}

	// go/format may tab-align map entries, so check that both the key and
	// value appear on the same line rather than requiring exact spacing.
	if !containsKeyValue(src, `"error.required"`, `"This field is required"`) {
		t.Error("expected TD default for error.required in WithDefaults map")
	}
}

// TestIntegration_ExtractWithExclude verifies that the -exclude flag causes
// directories to be skipped during extraction.
func TestIntegration_ExtractWithExclude(t *testing.T) {
	// Use the testdata fixture but exclude it, so no keys should be found.
	dir := testdataDir(t)
	parent := filepath.Dir(dir)

	var stdout, stderr bytes.Buffer
	code := run([]string{"extract", "-exclude", "testdata", parent}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
	}

	output := stdout.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if lines[0] == "" {
		lines = nil
	}

	// Extract the keys from the non-testdata files (main.go, etc.).
	// The testdata keys (greeting, farewell, etc.) must NOT appear.
	testdataOnlyKeys := []string{
		"item_count",
		"salutation",
		"welcome_user",
		"items_format",
		"optional_feature",
		"nav.home",
	}
	for _, key := range testdataOnlyKeys {
		for _, line := range lines {
			cleanLine := strings.Split(line, "\t")[0]
			if cleanLine == key {
				t.Errorf("key %q should have been excluded with -exclude testdata", key)
			}
		}
	}
}

// TestIntegration_ExtractAllPatterns verifies all 9 call patterns are detected
// from the testdata fixture, along with struct field and WithDefaults patterns.
func TestIntegration_ExtractAllPatterns(t *testing.T) {
	dir := testdataDir(t)

	var stdout, stderr bytes.Buffer
	code := run([]string{"extract", dir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
	}

	output := stdout.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Collect all keys from the output lines.
	extractedKeys := make(map[string]bool)
	for _, line := range lines {
		key := strings.Split(line, "\t")[0]
		// Strip struct: prefix for the key set.
		key = strings.TrimPrefix(key, "struct:")
		extractedKeys[key] = true
	}

	expectedKeys := []string{
		"greeting",         // Translate
		"welcome_user",     // TranslateWithArgs
		"item_count",       // TranslatePlural
		"salutation",       // TranslateGender
		"farewell",         // T
		"items_format",     // TF
		"error.required",   // TD
		"optional_feature", // Has
		"nav.home",         // Key
		"Greeting",         // Msg.Get().Greeting
		"Farewell",         // Msg.Get().Farewell
	}
	for _, key := range expectedKeys {
		if !extractedKeys[key] {
			t.Errorf("expected key %q in extract output", key)
		}
	}

	// Verify the output is sorted.
	sortedLines := make([]string, len(lines))
	copy(sortedLines, lines)
	sort.Strings(sortedLines)
	for i, line := range lines {
		if line != sortedLines[i] {
			t.Error("extract output is not sorted")
			break
		}
	}
}

// containsKeyValue checks whether a key and value appear together on the same
// line in the source string. This tolerates go/format tab alignment between
// the key and value.
func containsKeyValue(src, key, value string) bool {
	for _, line := range strings.Split(src, "\n") {
		if strings.Contains(line, key) && strings.Contains(line, value) {
			return true
		}
	}
	return false
}
