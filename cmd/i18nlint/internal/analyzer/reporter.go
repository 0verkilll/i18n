// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

package analyzer

import (
	"encoding/json"
	"fmt"
	"io"
)

// FormatText is the text output format identifier.
const FormatText = "text"

// FormatJSON is the JSON output format identifier.
const FormatJSON = "json"

// formatIssueLine formats a single issue as a diagnostic line.
func formatIssueLine(issue Issue) string {
	switch {
	case issue.File != "" && issue.Line > 0:
		return fmt.Sprintf("%s:%d:%d: %s: %s", issue.File, issue.Line, issue.Col, issue.Severity, issue.Message)
	case issue.File != "":
		return fmt.Sprintf("%s: %s: %s", issue.File, issue.Severity, issue.Message)
	default:
		return fmt.Sprintf("%s: %s", issue.Severity, issue.Message)
	}
}

// ReportText writes issues in the Go compiler diagnostic format: file:line:col: severity: message
func ReportText(w io.Writer, issues []Issue) error {
	for _, issue := range issues {
		line := formatIssueLine(issue)
		if _, err := fmt.Fprintln(w, line); err != nil {
			return fmt.Errorf("writing text report: %w", err)
		}
	}
	return nil
}

// ReportJSON writes issues as NDJSON (one JSON object per line).
func ReportJSON(w io.Writer, issues []Issue) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	for _, issue := range issues {
		if err := enc.Encode(issue); err != nil {
			return fmt.Errorf("writing JSON report: %w", err)
		}
	}
	return nil
}
