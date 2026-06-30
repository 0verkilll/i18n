// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

package analyzer

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestReportTextFormat(t *testing.T) {
	issues := []Issue{
		{
			File:     "main.go",
			Line:     10,
			Col:      5,
			Severity: SeverityError,
			Code:     CodeMissingKey,
			Message:  `translation key "greeting" not found in any locale file`,
			Key:      "greeting",
		},
	}

	var buf bytes.Buffer
	if err := ReportText(&buf, issues); err != nil {
		t.Fatalf("ReportText() error: %v", err)
	}

	output := buf.String()
	expected := `main.go:10:5: error: translation key "greeting" not found in any locale file`
	if !strings.Contains(output, expected) {
		t.Errorf("text output does not match expected format\ngot:  %s\nwant: %s", strings.TrimSpace(output), expected)
	}
}

func TestReportJSONFormat(t *testing.T) {
	issues := []Issue{
		{
			File:     "main.go",
			Line:     10,
			Col:      5,
			Severity: SeverityError,
			Code:     CodeMissingKey,
			Message:  `translation key "greeting" not found`,
			Key:      "greeting",
		},
		{
			File:     "en.json",
			Line:     0,
			Col:      0,
			Severity: SeverityWarning,
			Code:     CodeUnusedKey,
			Message:  `key "farewell" is unused`,
			Key:      "farewell",
		},
	}

	var buf bytes.Buffer
	if err := ReportJSON(&buf, issues); err != nil {
		t.Fatalf("ReportJSON() error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 NDJSON lines, got %d", len(lines))
	}

	// Verify first line is valid JSON with all required fields.
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(lines[0]), &parsed); err != nil {
		t.Fatalf("first JSON line is invalid: %v", err)
	}

	requiredFields := []string{"file", "line", "col", "severity", "code", "message", "key"}
	for _, field := range requiredFields {
		if _, ok := parsed[field]; !ok {
			t.Errorf("JSON output missing field %q", field)
		}
	}

	if parsed["file"] != "main.go" {
		t.Errorf("expected file=main.go, got %v", parsed["file"])
	}
	if parsed["severity"] != "error" {
		t.Errorf("expected severity=error, got %v", parsed["severity"])
	}
}

func TestExitCodeZeroWhenNoIssues(t *testing.T) {
	srcDir := writeGoSource(t, "clean", `package main
func main() { t.Translate("greeting") }
`)
	localeDir := t.TempDir()
	writeFile(t, localeDir+"/en.json", `{"greeting": "Hello"}`)

	cfg := Config{
		SourceDirs:  []string{srcDir},
		LocalePaths: []string{localeDir},
	}
	issues, err := Run(cfg)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	// Exit code logic: 0 = no issues.
	exitCode := 0
	if len(issues) > 0 {
		exitCode = 1
	}

	if exitCode != 0 {
		t.Errorf("expected exit code 0 (no issues), got %d; issues: %v", exitCode, issues)
	}
}

func TestExitCodeOneWhenIssuesFound(t *testing.T) {
	srcDir := writeGoSource(t, "issues", `package main
func main() { t.Translate("missing.key") }
`)
	localeDir := t.TempDir()
	writeFile(t, localeDir+"/en.json", `{"greeting": "Hello"}`)

	cfg := Config{
		SourceDirs:  []string{srcDir},
		LocalePaths: []string{localeDir},
	}
	issues, err := Run(cfg)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	exitCode := 0
	if len(issues) > 0 {
		exitCode = 1
	}

	if exitCode != 1 {
		t.Errorf("expected exit code 1 (issues found), got %d", exitCode)
	}
}

func TestExitCodeTwoOnFatalError(t *testing.T) {
	cfg := Config{
		SourceDirs:  []string{"/nonexistent/path/that/does/not/exist"},
		LocalePaths: []string{"/also/nonexistent"},
	}
	_, err := Run(cfg)

	// Fatal I/O error should return non-nil error (which maps to exit code 2).
	if err == nil {
		t.Error("expected error for nonexistent paths, got nil")
	}
}
