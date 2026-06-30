// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

package analyzer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMissingKeyDetection(t *testing.T) {
	srcDir := writeGoSource(t, "missing", `package main
func main() { t.Translate("nonexistent.key") }
`)
	localeDir := t.TempDir()
	writeFile(t, filepath.Join(localeDir, "en.json"), `{"greeting": "Hello"}`)

	cfg := Config{
		SourceDirs:  []string{srcDir},
		LocalePaths: []string{localeDir},
	}
	issues, err := Run(cfg)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	found := false
	for _, issue := range issues {
		if issue.Code == CodeMissingKey && issue.Key == "nonexistent.key" {
			found = true
			if issue.Severity != SeverityError {
				t.Errorf("missing key should be severity error, got %q", issue.Severity)
			}
		}
	}
	if !found {
		t.Error("expected missing-key issue for nonexistent.key")
	}
}

func TestUnusedKeyDetection(t *testing.T) {
	srcDir := writeGoSource(t, "unused", `package main
func main() { t.Translate("greeting") }
`)
	localeDir := t.TempDir()
	writeFile(t, filepath.Join(localeDir, "en.json"), `{"greeting": "Hello", "farewell": "Goodbye"}`)

	cfg := Config{
		SourceDirs:  []string{srcDir},
		LocalePaths: []string{localeDir},
	}
	issues, err := Run(cfg)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	found := false
	for _, issue := range issues {
		if issue.Code == CodeUnusedKey && issue.Key == "farewell" {
			found = true
			if issue.Severity != SeverityWarning {
				t.Errorf("unused key should be severity warning, got %q", issue.Severity)
			}
		}
	}
	if !found {
		t.Error("expected unused-key issue for farewell")
	}
}

func TestFormatMismatchDetection(t *testing.T) {
	srcDir := writeGoSource(t, "fmtmatch", `package main
func main() { t.TF("msg", a, b) }
`)
	localeDir := t.TempDir()
	// Translation has 1 specifier but source passes 2 arguments.
	writeFile(t, filepath.Join(localeDir, "en.json"), `{"msg": "Hello %s"}`)

	cfg := Config{
		SourceDirs:  []string{srcDir},
		LocalePaths: []string{localeDir},
	}
	issues, err := Run(cfg)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	found := false
	for _, issue := range issues {
		if issue.Code == CodeFormatMismatch && issue.Key == "msg" {
			found = true
			if issue.Severity != SeverityError {
				t.Errorf("format mismatch should be severity error, got %q", issue.Severity)
			}
		}
	}
	if !found {
		t.Error("expected format-mismatch issue for msg")
	}
}

func TestFormatMismatchEscapedPercent(t *testing.T) {
	srcDir := writeGoSource(t, "fmtesc", `package main
func main() { t.TF("pct", a) }
`)
	localeDir := t.TempDir()
	// "100%% %s" has 1 real specifier (%% is escaped), matching 1 argument.
	writeFile(t, filepath.Join(localeDir, "en.json"), `{"pct": "100%% %s"}`)

	cfg := Config{
		SourceDirs:  []string{srcDir},
		LocalePaths: []string{localeDir},
	}
	issues, err := Run(cfg)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	// Should NOT have a format mismatch since 1 specifier matches 1 argument.
	for _, issue := range issues {
		if issue.Code == CodeFormatMismatch && issue.Key == "pct" {
			t.Error("should not report format mismatch for escaped %% with correct arg count")
		}
	}
}

func TestNamespaceFilterRestrictsMissingKeyReports(t *testing.T) {
	srcDir := writeGoSource(t, "nsfilter", `package main
func main() {
	t.Translate("error.timeout")
	t.Translate("button.submit")
}
`)
	localeDir := t.TempDir()
	writeFile(t, filepath.Join(localeDir, "en.json"), `{"greeting": "Hello"}`)

	cfg := Config{
		SourceDirs:  []string{srcDir},
		LocalePaths: []string{localeDir},
		Namespace:   "error",
	}
	issues, err := Run(cfg)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	// Only error.timeout should be reported as missing; button.submit should be filtered out.
	for _, issue := range issues {
		if issue.Code == CodeMissingKey && issue.Key == "button.submit" {
			t.Error("button.submit should be filtered out by namespace=error")
		}
	}

	found := false
	for _, issue := range issues {
		if issue.Code == CodeMissingKey && issue.Key == "error.timeout" {
			found = true
		}
	}
	if !found {
		t.Error("expected missing-key issue for error.timeout under namespace=error")
	}
}

func TestOrphanedNamespaceDetection(t *testing.T) {
	srcDir := writeGoSource(t, "orphnns", `package main
func main() { t.Translate("greeting") }
`)
	localeDir := t.TempDir()
	// "unused_ns" is a top-level key with no source reference.
	writeFile(t, filepath.Join(localeDir, "en.json"), `{"greeting": "Hello", "unused_ns": {"a": "b"}}`)

	cfg := Config{
		SourceDirs:  []string{srcDir},
		LocalePaths: []string{localeDir},
	}
	issues, err := Run(cfg)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	found := false
	for _, issue := range issues {
		if issue.Code == CodeOrphanedNamespace && issue.Key == "unused_ns" {
			found = true
			if issue.Severity != SeverityWarning {
				t.Errorf("orphaned namespace should be warning, got %q", issue.Severity)
			}
		}
	}
	if !found {
		t.Error("expected orphaned-namespace issue for unused_ns")
	}
}

func TestUnknownNamespaceDetection(t *testing.T) {
	srcDir := writeGoSource(t, "unknns", `package main
func main() { t.Translate("mystery.key") }
`)
	localeDir := t.TempDir()
	writeFile(t, filepath.Join(localeDir, "en.json"), `{"greeting": "Hello"}`)

	cfg := Config{
		SourceDirs:  []string{srcDir},
		LocalePaths: []string{localeDir},
	}
	issues, err := Run(cfg)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	found := false
	for _, issue := range issues {
		if issue.Code == CodeUnknownNamespace && issue.Key == "mystery" {
			found = true
			if issue.Severity != SeverityWarning {
				t.Errorf("unknown namespace should be warning, got %q", issue.Severity)
			}
		}
	}
	if !found {
		t.Error("expected unknown-namespace issue for mystery")
	}
}

func TestRunReturnsSortedIssues(t *testing.T) {
	srcDir := t.TempDir()
	srcSubDir := filepath.Join(srcDir, "pkg")
	if err := os.MkdirAll(srcSubDir, 0o755); err != nil {
		t.Fatal(err)
	}

	writeFile(t, filepath.Join(srcDir, "b.go"), `package main
func main() { t.Translate("z.key") }
`)
	writeFile(t, filepath.Join(srcDir, "a.go"), `package main
func init() { t.Translate("a.key") }
`)

	localeDir := t.TempDir()
	writeFile(t, filepath.Join(localeDir, "en.json"), `{"other": "val"}`)

	cfg := Config{
		SourceDirs:  []string{srcDir},
		LocalePaths: []string{localeDir},
	}
	issues, err := Run(cfg)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	// Verify sorting: issues should be ordered by file, then line, then col.
	for i := 1; i < len(issues); i++ {
		prev, curr := issues[i-1], issues[i]
		if prev.File > curr.File {
			t.Errorf("issues not sorted by file: %q > %q", prev.File, curr.File)
		}
		if prev.File == curr.File && prev.Line > curr.Line {
			t.Errorf("issues not sorted by line within same file")
		}
		if prev.File == curr.File && prev.Line == curr.Line && prev.Col > curr.Col {
			t.Errorf("issues not sorted by col within same file and line")
		}
	}

	if len(issues) == 0 {
		t.Error("expected at least one issue")
	}
}

// Task Group 5: Strategic gap-filling tests

func TestEndToEndIntegration(t *testing.T) {
	// Multiple Go files referencing keys, with a mix of found, missing, and mismatched keys.
	srcDir := t.TempDir()
	writeFile(t, filepath.Join(srcDir, "app.go"), `package main

func main() {
	t.Translate("greeting")
	t.Translate("missing.key")
	t.TF("welcome", name)
}
`)
	writeFile(t, filepath.Join(srcDir, "util.go"), `package main

func helper() {
	t.HasKey("farewell")
}
`)

	localeDir := t.TempDir()
	writeFile(t, filepath.Join(localeDir, "en.json"), `{
		"greeting": "Hello",
		"farewell": "Goodbye",
		"welcome": "Welcome %s %s",
		"orphaned": "Never used"
	}`)

	cfg := Config{
		SourceDirs:  []string{srcDir},
		LocalePaths: []string{localeDir},
	}
	issues, err := Run(cfg)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	codes := make(map[string]int)
	for _, issue := range issues {
		codes[issue.Code]++
	}

	// Expect: missing.key (missing-key), welcome format mismatch (1 arg vs 2 specifiers),
	// orphaned (unused-key), and namespace issues.
	if codes[CodeMissingKey] < 1 {
		t.Error("expected at least one missing-key issue")
	}
	if codes[CodeFormatMismatch] < 1 {
		t.Error("expected at least one format-mismatch issue")
	}
	if codes[CodeUnusedKey] < 1 {
		t.Error("expected at least one unused-key issue")
	}
}

func TestEmptySourceDirectoryAllKeysUnused(t *testing.T) {
	// Source directory with no Go files means all locale keys are unused.
	srcDir := t.TempDir()
	localeDir := t.TempDir()
	writeFile(t, filepath.Join(localeDir, "en.json"), `{"key1": "val1", "key2": "val2"}`)

	cfg := Config{
		SourceDirs:  []string{srcDir},
		LocalePaths: []string{localeDir},
	}
	issues, err := Run(cfg)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	unusedCount := 0
	for _, issue := range issues {
		if issue.Code == CodeUnusedKey {
			unusedCount++
		}
	}

	if unusedCount != 2 {
		t.Errorf("expected 2 unused-key issues, got %d", unusedCount)
	}
}

func TestSourceWithNoTranslationCalls(t *testing.T) {
	srcDir := writeGoSource(t, "notrans", `package main

import "fmt"

func main() {
	fmt.Println("hello")
}
`)
	localeDir := t.TempDir()
	writeFile(t, filepath.Join(localeDir, "en.json"), `{"greeting": "Hello"}`)

	cfg := Config{
		SourceDirs:  []string{srcDir},
		LocalePaths: []string{localeDir},
	}
	issues, err := Run(cfg)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	// No missing keys since there are no translation calls, but greeting is unused.
	for _, issue := range issues {
		if issue.Code == CodeMissingKey {
			t.Errorf("should have no missing-key issues; got: %s", issue.Message)
		}
	}
}

func TestMultiLocaleKeyPresentInOneNotMissing(t *testing.T) {
	srcDir := writeGoSource(t, "multiloc", `package main
func main() { t.Translate("greeting") }
`)
	localeDir := t.TempDir()
	// greeting exists in en but not es -- should NOT be reported as missing.
	writeFile(t, filepath.Join(localeDir, "en.json"), `{"greeting": "Hello"}`)
	writeFile(t, filepath.Join(localeDir, "es.json"), `{"farewell": "Adios"}`)

	cfg := Config{
		SourceDirs:  []string{srcDir},
		LocalePaths: []string{localeDir},
	}
	issues, err := Run(cfg)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	for _, issue := range issues {
		if issue.Code == CodeMissingKey && issue.Key == "greeting" {
			t.Error("greeting exists in en.json, so it should NOT be reported as missing")
		}
	}
}

func TestExcludePatternWithNamespaceFilter(t *testing.T) {
	rootDir := t.TempDir()
	vendorDir := filepath.Join(rootDir, "vendor")
	if err := os.MkdirAll(vendorDir, 0o755); err != nil {
		t.Fatal(err)
	}

	writeFile(t, filepath.Join(rootDir, "main.go"), `package main
func main() {
	t.Translate("app.key")
	t.Translate("other.key")
}
`)
	writeFile(t, filepath.Join(vendorDir, "v.go"), `package vendor
func init() { t.Translate("vendor.key") }
`)

	localeDir := t.TempDir()
	writeFile(t, filepath.Join(localeDir, "en.json"), `{"app": {"key": "val"}, "other": {"key": "val"}}`)

	cfg := Config{
		SourceDirs:      []string{rootDir},
		LocalePaths:     []string{localeDir},
		ExcludePatterns: []string{"vendor"},
		Namespace:       "app",
	}
	issues, err := Run(cfg)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	// vendor.key should not appear (excluded), other.key should not be reported as missing (filtered).
	for _, issue := range issues {
		if issue.Code == CodeMissingKey && issue.Key == "vendor.key" {
			t.Error("vendor.key should be excluded")
		}
		if issue.Code == CodeMissingKey && issue.Key == "other.key" {
			t.Error("other.key should be filtered out by namespace=app")
		}
	}
}

func TestBacktickRawStringLiterals(t *testing.T) {
	src := "package main\n\nfunc main() {\n\tt.Translate(`backtick.key`)\n}\n"
	srcDir := writeGoSource(t, "backtick", src)

	localeDir := t.TempDir()
	writeFile(t, filepath.Join(localeDir, "en.json"), `{"backtick": {"key": "val"}}`)

	cfg := Config{
		SourceDirs:  []string{srcDir},
		LocalePaths: []string{localeDir},
	}
	issues, err := Run(cfg)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	// backtick.key exists, so no missing key should be reported.
	for _, issue := range issues {
		if issue.Code == CodeMissingKey && issue.Key == "backtick.key" {
			t.Error("backtick.key exists in locale, should not be missing")
		}
	}
}

func TestOnlyEscapedPercentsZeroSpecifiers(t *testing.T) {
	srcDir := writeGoSource(t, "onlyesc", `package main
func main() { t.TF("pctonly", a) }
`)
	localeDir := t.TempDir()
	// Only %% means 0 real specifiers, but call passes 1 arg.
	writeFile(t, filepath.Join(localeDir, "en.json"), `{"pctonly": "100%%"}`)

	cfg := Config{
		SourceDirs:  []string{srcDir},
		LocalePaths: []string{localeDir},
	}
	issues, err := Run(cfg)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	found := false
	for _, issue := range issues {
		if issue.Code == CodeFormatMismatch && issue.Key == "pctonly" {
			found = true
		}
	}
	if !found {
		t.Error("expected format-mismatch: 0 specifiers but 1 argument passed")
	}
}
