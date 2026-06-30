// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

package analyzer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectTranslateCall(t *testing.T) {
	src := `package main

func main() {
	t.Translate("greeting")
}
`
	dir := writeGoSource(t, "translate", src)
	keys, err := scanDirectories([]string{dir}, nil)
	if err != nil {
		t.Fatalf("scanDirectories() error: %v", err)
	}

	if len(keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(keys))
	}

	if keys[0].key != "greeting" {
		t.Errorf("expected key %q, got %q", "greeting", keys[0].key)
	}
	if keys[0].line != 4 {
		t.Errorf("expected line 4, got %d", keys[0].line)
	}
	if keys[0].argCount != -1 {
		t.Errorf("Translate should have argCount -1, got %d", keys[0].argCount)
	}
}

func TestDetectTranslateWithArgsCountsArguments(t *testing.T) {
	src := `package main

func main() {
	t.TranslateWithArgs("welcome.user", name, count)
}
`
	dir := writeGoSource(t, "twargs", src)
	keys, err := scanDirectories([]string{dir}, nil)
	if err != nil {
		t.Fatalf("scanDirectories() error: %v", err)
	}

	if len(keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(keys))
	}

	if keys[0].key != "welcome.user" {
		t.Errorf("expected key %q, got %q", "welcome.user", keys[0].key)
	}
	if keys[0].argCount != 2 {
		t.Errorf("expected argCount 2, got %d", keys[0].argCount)
	}
}

func TestDetectNamespaceHelpers(t *testing.T) {
	src := `package main

func main() {
	ns.T("hello")
	ns.TF("formatted", a)
	ns.Has("check")
	ns.TD("default")
	ns.Key("raw")
}
`
	dir := writeGoSource(t, "nshelp", src)
	keys, err := scanDirectories([]string{dir}, nil)
	if err != nil {
		t.Fatalf("scanDirectories() error: %v", err)
	}

	if len(keys) != 5 {
		t.Fatalf("expected 5 keys, got %d", len(keys))
	}

	expected := map[string]bool{
		"hello":     true,
		"formatted": true,
		"check":     true,
		"default":   true,
		"raw":       true,
	}

	for _, k := range keys {
		if !expected[k.key] {
			t.Errorf("unexpected key %q", k.key)
		}
	}
}

func TestNonLiteralArgumentsSkipped(t *testing.T) {
	src := `package main

var key = "dynamic"

func main() {
	t.Translate(key)
	t.Translate("literal")
	t.Translate(getKey())
	t.Translate("a" + "b")
}
`
	dir := writeGoSource(t, "nonlit", src)
	keys, err := scanDirectories([]string{dir}, nil)
	if err != nil {
		t.Fatalf("scanDirectories() error: %v", err)
	}

	// Only "literal" should be extracted.
	if len(keys) != 1 {
		t.Fatalf("expected 1 key (literal only), got %d", len(keys))
	}

	if keys[0].key != "literal" {
		t.Errorf("expected key %q, got %q", "literal", keys[0].key)
	}
}

func TestDetectHasKeyCall(t *testing.T) {
	src := `package main

func main() {
	t.HasKey("error.timeout")
}
`
	dir := writeGoSource(t, "haskey", src)
	keys, err := scanDirectories([]string{dir}, nil)
	if err != nil {
		t.Fatalf("scanDirectories() error: %v", err)
	}

	if len(keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(keys))
	}
	if keys[0].key != "error.timeout" {
		t.Errorf("expected key %q, got %q", "error.timeout", keys[0].key)
	}
	if keys[0].argCount != -1 {
		t.Errorf("HasKey should have argCount -1, got %d", keys[0].argCount)
	}
}

func TestExcludePatterns(t *testing.T) {
	// Create a root directory with a subdirectory named "vendor".
	rootDir := t.TempDir()
	vendorDir := filepath.Join(rootDir, "vendor")
	if err := os.MkdirAll(vendorDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write Go source in both root and vendor.
	rootSrc := `package main
func main() { t.Translate("root.key") }
`
	vendorSrc := `package vendor
func init() { t.Translate("vendor.key") }
`
	writeFile(t, filepath.Join(rootDir, "main.go"), rootSrc)
	writeFile(t, filepath.Join(vendorDir, "lib.go"), vendorSrc)

	keys, err := scanDirectories([]string{rootDir}, []string{"vendor"})
	if err != nil {
		t.Fatalf("scanDirectories() error: %v", err)
	}

	// Only root.key should be found; vendor should be excluded.
	if len(keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(keys))
	}
	if keys[0].key != "root.key" {
		t.Errorf("expected key %q, got %q", "root.key", keys[0].key)
	}
}

func TestSourceLocationAccuracy(t *testing.T) {
	// The string literal starts at column 15 (1-indexed) on line 4.
	src := `package main

func main() {
	t.Translate("location.check")
}
`
	dir := writeGoSource(t, "location", src)
	keys, err := scanDirectories([]string{dir}, nil)
	if err != nil {
		t.Fatalf("scanDirectories() error: %v", err)
	}

	if len(keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(keys))
	}

	if keys[0].line != 4 {
		t.Errorf("expected line 4, got %d", keys[0].line)
	}
	// The column should be at the opening quote of the string literal.
	if keys[0].col < 1 {
		t.Errorf("expected col > 0, got %d", keys[0].col)
	}
	// Verify the file path is absolute and ends with the expected file name.
	if filepath.Base(keys[0].file) != "main.go" {
		t.Errorf("expected file to end with main.go, got %q", keys[0].file)
	}
}

// writeGoSource creates a temp directory with a single Go source file.
func writeGoSource(t *testing.T, name string, src string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "main.go"), src)
	return dir
}
