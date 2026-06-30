// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

package extractor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtract_TranslateAndT(t *testing.T) {
	src := `package main

func f(t interface{ Translate(string) string }) {
	t.Translate("greeting")
	t.T("farewell")
}
`
	keys, err := ExtractFromSource("test.go", src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := map[string]bool{"greeting": false, "farewell": false}
	for _, k := range keys {
		if _, ok := want[k.Key]; ok {
			want[k.Key] = true
		}
	}
	for key, found := range want {
		if !found {
			t.Errorf("expected key %q not found", key)
		}
	}
}

func TestExtract_TranslateWithArgsAndTF(t *testing.T) {
	src := `package main

func f(t interface{ TranslateWithArgs(string, ...interface{}) string }) {
	t.TranslateWithArgs("welcome", "Alice")
	t.TF("items_count", 5, "books")
}
`
	keys, err := ExtractFromSource("test.go", src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	foundWelcome := false
	foundItems := false
	for _, k := range keys {
		if k.Key == "welcome" && k.Kind == KindFormat && k.ArgCount == 1 {
			foundWelcome = true
		}
		if k.Key == "items_count" && k.Kind == KindFormat && k.ArgCount == 2 {
			foundItems = true
		}
	}
	if !foundWelcome {
		t.Error("expected key 'welcome' with KindFormat and ArgCount=1")
	}
	if !foundItems {
		t.Error("expected key 'items_count' with KindFormat and ArgCount=2")
	}
}

func TestExtract_PluralAndGender(t *testing.T) {
	src := `package main

func f(t interface{}) {
	t.TranslatePlural("item_count", 5)
	t.TranslateGender("greeting_gendered", "female")
}
`
	keys, err := ExtractFromSource("test.go", src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	foundPlural := false
	foundGender := false
	for _, k := range keys {
		if k.Key == "item_count" && k.Kind == KindPlural {
			foundPlural = true
		}
		if k.Key == "greeting_gendered" && k.Kind == KindGender {
			foundGender = true
		}
	}
	if !foundPlural {
		t.Error("expected key 'item_count' with KindPlural")
	}
	if !foundGender {
		t.Error("expected key 'greeting_gendered' with KindGender")
	}
}

func TestExtract_TDWithDefault(t *testing.T) {
	src := `package main

func f(ns interface{}) {
	ns.TD("error.required", "This field is required")
}
`
	keys, err := ExtractFromSource("test.go", src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, k := range keys {
		if k.Key == "error.required" && k.DefaultValue == "This field is required" && k.Kind == KindWithDefault {
			found = true
		}
	}
	if !found {
		t.Error("expected key 'error.required' with default 'This field is required'")
	}
}

func TestExtract_WithDefaultsMap(t *testing.T) {
	src := `package main

import "github.com/0verkilll/i18n"

func f() {
	i18n.WithDefaults(map[string]string{
		"app.name":    "MyApp",
		"app.version": "1.0",
	})
}
`
	keys, err := ExtractFromSource("test.go", src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := map[string]string{
		"app.name":    "MyApp",
		"app.version": "1.0",
	}
	found := make(map[string]bool)
	for _, k := range keys {
		if k.Kind == KindWithDefault && k.Pattern == "WithDefaults" {
			if expected, ok := want[k.Key]; ok && k.DefaultValue == expected {
				found[k.Key] = true
			}
		}
	}
	for key := range want {
		if !found[key] {
			t.Errorf("expected WithDefaults key %q not found", key)
		}
	}
}

func TestExtract_StructFieldPattern(t *testing.T) {
	src := `package main

type Messages struct {
	Greeting string
	Farewell string
}

type holder struct{}

func (h holder) Get() *Messages { return nil }

func f() {
	var Msg holder
	_ = Msg.Get().Greeting
	_ = Msg.Get().Farewell
}
`
	keys, err := ExtractFromSource("test.go", src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	foundGreeting := false
	foundFarewell := false
	for _, k := range keys {
		if k.Key == "Greeting" && k.Kind == KindStructField {
			foundGreeting = true
		}
		if k.Key == "Farewell" && k.Kind == KindStructField {
			foundFarewell = true
		}
	}
	if !foundGreeting {
		t.Error("expected struct field key 'Greeting'")
	}
	if !foundFarewell {
		t.Error("expected struct field key 'Farewell'")
	}
}

func TestExtract_HasAndHasKey(t *testing.T) {
	src := `package main

func f(t interface{}) {
	t.Has("optional_key")
	t.HasKey("another_key")
}
`
	keys, err := ExtractFromSource("test.go", src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := map[string]bool{"optional_key": false, "another_key": false}
	for _, k := range keys {
		if _, ok := want[k.Key]; ok {
			want[k.Key] = true
		}
	}
	for key, found := range want {
		if !found {
			t.Errorf("expected key %q not found", key)
		}
	}
}

func TestExtract_NonLiteralArgsSkipped(t *testing.T) {
	src := `package main

var key = "dynamic"

func f(t interface{}) {
	t.Translate(key)
	t.Translate("literal_key")
	t.T(getString())
}

func getString() string { return "x" }
`
	keys, err := ExtractFromSource("test.go", src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(keys))
	}
	if keys[0].Key != "literal_key" {
		t.Errorf("expected 'literal_key', got %q", keys[0].Key)
	}
}

func TestExtract_TestFilesSkipped(t *testing.T) {
	if !isGoSource("main.go") {
		t.Error("main.go should be a valid Go source file")
	}
	if isGoSource("main_test.go") {
		t.Error("main_test.go should be skipped")
	}
	if isGoSource("readme.md") {
		t.Error("readme.md should be skipped")
	}
}

func TestExtract_DirectoryWalkingWithExclude(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a Go file in the root.
	writeGoFile(t, tmpDir, "main.go", `package main
func f(t interface{}) { t.Translate("root_key") }
`)

	// Create a subdirectory with a Go file.
	subDir := filepath.Join(tmpDir, "sub")
	mustMkdir(t, subDir)
	writeGoFile(t, subDir, "sub.go", `package sub
func f(t interface{}) { t.Translate("sub_key") }
`)

	// Create an excluded directory with a Go file.
	vendorDir := filepath.Join(tmpDir, "vendor")
	mustMkdir(t, vendorDir)
	writeGoFile(t, vendorDir, "vendor.go", `package vendor
func f(t interface{}) { t.Translate("vendor_key") }
`)

	keys, err := Extract([]string{tmpDir}, []string{"vendor"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	foundRoot := false
	foundSub := false
	for _, k := range keys {
		if k.Key == "root_key" {
			foundRoot = true
		}
		if k.Key == "sub_key" {
			foundSub = true
		}
		if k.Key == "vendor_key" {
			t.Error("vendor_key should have been excluded")
		}
	}
	if !foundRoot {
		t.Error("expected root_key to be found")
	}
	if !foundSub {
		t.Error("expected sub_key to be found")
	}
}

func FuzzExtractKeysFromSource(f *testing.F) {
	f.Add(`package main
func f(t interface{}) { t.Translate("hello") }
`)
	f.Add(`package main
func f() { }
`)
	f.Add(`package main
func f(t interface{}) { t.TD("k", "v") }
`)

	f.Fuzz(func(t *testing.T, src string) {
		// Must not panic on any input.
		_, _ = ExtractFromSource("fuzz.go", src)
	})
}

// writeGoFile creates a Go file in the given directory.
func writeGoFile(t *testing.T, dir, name, content string) {
	t.Helper()
	err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644)
	if err != nil {
		t.Fatalf("writing file: %v", err)
	}
}

// mustMkdir creates a directory.
func mustMkdir(t *testing.T, path string) {
	t.Helper()
	err := os.MkdirAll(path, 0o755)
	if err != nil {
		t.Fatalf("creating directory: %v", err)
	}
}
