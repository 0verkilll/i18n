// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

package main

import (
	"bytes"
	"encoding/json"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0verkilll/i18n"
)

// testFixtureDir creates a temp directory with sample Go source files
// containing various translation call patterns.
func testFixtureDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	src := `package sample

func translate(t interface {
	Translate(string) string
	TranslateWithArgs(string, ...interface{}) string
	TD(string, string) string
}) {
	t.Translate("greeting")
	t.Translate("farewell")
	t.TranslateWithArgs("welcome", "Alice")
	t.TD("error.required", "This field is required")
}
`
	writeFile(t, filepath.Join(dir, "sample.go"), src)
	return dir
}

// writeFile creates a file with the given content.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing file %s: %v", path, err)
	}
}

func TestExtractSubcommand(t *testing.T) {
	dir := testFixtureDir(t)

	var stdout, stderr bytes.Buffer
	code := run([]string{"extract", dir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "greeting") {
		t.Error("expected 'greeting' in extract output")
	}
	if !strings.Contains(output, "farewell") {
		t.Error("expected 'farewell' in extract output")
	}
	if !strings.Contains(output, "error.required\tThis field is required") {
		t.Error("expected 'error.required\\tThis field is required' in extract output")
	}
}

func TestGenerateJSON(t *testing.T) {
	dir := testFixtureDir(t)

	var stdout, stderr bytes.Buffer
	code := run([]string{"generate", "-format", "json", dir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
	}

	var parsed map[string]string
	if err := json.Unmarshal(stdout.Bytes(), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput:\n%s", err, stdout.String())
	}

	if parsed["error.required"] != "This field is required" {
		t.Errorf("expected TD default, got %q", parsed["error.required"])
	}
	if _, ok := parsed["greeting"]; !ok {
		t.Error("expected 'greeting' key in JSON output")
	}
}

func TestGenerateStruct(t *testing.T) {
	dir := testFixtureDir(t)

	var stdout, stderr bytes.Buffer
	code := run([]string{"generate", "-format", "struct", "-package", "mypkg", dir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
	}

	output := stdout.String()
	if _, err := format.Source([]byte(output)); err != nil {
		t.Fatalf("output is not valid Go: %v\noutput:\n%s", err, output)
	}
	if !strings.Contains(output, "package mypkg") {
		t.Error("expected 'package mypkg' in struct output")
	}
}

func TestGenerateBinary(t *testing.T) {
	dir := testFixtureDir(t)

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
	if _, ok := flat["greeting"]; !ok {
		t.Error("expected 'greeting' key in binary output")
	}
}

func TestDiffSubcommand_WithIssues(t *testing.T) {
	dir := testFixtureDir(t)

	// Create a locale file that is missing "farewell" and has extra "unused_key".
	localeFile := filepath.Join(t.TempDir(), "en-US.json")
	localeJSON := `{
		"greeting": "Hello",
		"error.required": "Required",
		"welcome": "Welcome",
		"unused_key": "Not used"
	}`
	writeFile(t, localeFile, localeJSON)

	var stdout, stderr bytes.Buffer
	code := run([]string{"diff", "-locale", localeFile, dir}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected exit code 1 (issues found), got %d; stderr: %s; stdout: %s", code, stderr.String(), stdout.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "missing: farewell") {
		t.Errorf("expected 'missing: farewell' in diff output, got:\n%s", output)
	}
	if !strings.Contains(output, "unused: unused_key") {
		t.Errorf("expected 'unused: unused_key' in diff output, got:\n%s", output)
	}
}

func TestDiffSubcommand_Clean(t *testing.T) {
	dir := testFixtureDir(t)

	// Create a locale file with all required keys.
	localeFile := filepath.Join(t.TempDir(), "en-US.json")
	localeJSON := `{
		"greeting": "Hello",
		"farewell": "Goodbye",
		"error.required": "Required",
		"welcome": "Welcome"
	}`
	writeFile(t, localeFile, localeJSON)

	var stdout, stderr bytes.Buffer
	code := run([]string{"diff", "-locale", localeFile, dir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 (clean), got %d; stderr: %s; stdout: %s", code, stderr.String(), stdout.String())
	}
}

func TestInitSubcommand(t *testing.T) {
	dir := testFixtureDir(t)

	var stdout, stderr bytes.Buffer
	code := run([]string{"init", "-package", "mypkg", dir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
	}

	output := stdout.String()
	if _, err := format.Source([]byte(output)); err != nil {
		t.Fatalf("output is not valid Go: %v\noutput:\n%s", err, output)
	}
	if !strings.Contains(output, "package mypkg") {
		t.Error("expected 'package mypkg' in init output")
	}
	if !strings.Contains(output, `"github.com/0verkilll/i18n"`) {
		t.Error("expected i18n import in init output")
	}
}

func TestUnknownSubcommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"bogus"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "unknown command") {
		t.Error("expected 'unknown command' in stderr")
	}
}

func TestNoArgs(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run(nil, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "Usage:") {
		t.Error("expected usage text in stderr")
	}
}

func FuzzRunArgs(f *testing.F) {
	f.Add("extract")
	f.Add("generate")
	f.Add("diff")
	f.Add("init")
	f.Add("")
	f.Add("bogus")
	f.Add("extract -exclude vendor")

	f.Fuzz(func(t *testing.T, input string) {
		args := strings.Fields(input)
		var stdout, stderr bytes.Buffer
		// Must not panic on any input.
		_ = run(args, &stdout, &stderr)
	})
}
