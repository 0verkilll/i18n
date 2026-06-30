// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

package generator

import (
	"bytes"
	"encoding/json"
	"go/format"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0verkilll/i18n"
	"github.com/0verkilll/i18n/cmd/i18ngen/internal/extractor"
)

func TestGenerateJSON_SortedKeysWithDefaults(t *testing.T) {
	keys := []extractor.ExtractedKey{
		{Key: "greeting", Kind: extractor.KindStandard},
		{Key: "error.validation.required", Kind: extractor.KindWithDefault, DefaultValue: "Field is required"},
		{Key: "items.one", Kind: extractor.KindPlural},
		{Key: "items.other", Kind: extractor.KindPlural},
	}

	var buf bytes.Buffer
	err := GenerateJSON(keys, &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()

	// Verify it is valid JSON.
	var parsed map[string]string
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput:\n%s", err, output)
	}

	// Check TD default is populated.
	if parsed["error.validation.required"] != "Field is required" {
		t.Errorf("expected TD default, got %q", parsed["error.validation.required"])
	}

	// Check non-TD keys have empty values.
	if parsed["greeting"] != "" {
		t.Errorf("expected empty value for greeting, got %q", parsed["greeting"])
	}

	// Verify keys are sorted in the output.
	idx1 := strings.Index(output, "error.validation.required")
	idx2 := strings.Index(output, "greeting")
	idx3 := strings.Index(output, "items.one")
	idx4 := strings.Index(output, "items.other")
	if idx1 >= idx2 || idx2 >= idx3 || idx3 >= idx4 {
		t.Error("keys are not in alphabetical order in JSON output")
	}
}

func TestGenerateJSON_StructFieldKeys(t *testing.T) {
	keys := []extractor.ExtractedKey{
		{Key: "Greeting", Kind: extractor.KindStructField},
		{Key: "Farewell", Kind: extractor.KindStructField},
	}

	var buf bytes.Buffer
	err := GenerateJSON(keys, &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed map[string]string
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	if _, ok := parsed["Farewell"]; !ok {
		t.Error("expected struct field key 'Farewell' in JSON output")
	}
	if _, ok := parsed["Greeting"]; !ok {
		t.Error("expected struct field key 'Greeting' in JSON output")
	}
}

func TestGenerateStruct_PascalCaseFields(t *testing.T) {
	keys := []extractor.ExtractedKey{
		{Key: "error.validation.required", Kind: extractor.KindStandard},
		{Key: "greeting", Kind: extractor.KindStandard},
		{Key: "items.one", Kind: extractor.KindPlural},
		{Key: "items.other", Kind: extractor.KindPlural},
	}

	var buf bytes.Buffer
	err := GenerateStruct(keys, "mypackage", &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()

	// Verify it is valid Go source (passes go/format).
	if _, err := format.Source([]byte(output)); err != nil {
		t.Fatalf("output is not valid Go: %v\noutput:\n%s", err, output)
	}

	// Check expected field names exist (go/format may tab-align fields).
	expectedFields := []string{
		"ErrorValidationRequired",
		"Greeting",
		"ItemsOne",
		"ItemsOther",
	}
	for _, field := range expectedFields {
		if !containsStructField(output, field) {
			t.Errorf("expected field %q not found in output:\n%s", field, output)
		}
	}

	// Check package declaration.
	if !strings.Contains(output, "package mypackage") {
		t.Error("expected package declaration 'package mypackage'")
	}
}

func TestGenerateBinary_RoundTrip(t *testing.T) {
	keys := []extractor.ExtractedKey{
		{Key: "greeting", Kind: extractor.KindStandard},
		{Key: "farewell", Kind: extractor.KindWithDefault, DefaultValue: "Goodbye"},
	}

	var buf bytes.Buffer
	err := GenerateBinary(keys, &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Decode with BinaryParser.
	parser := i18n.NewBinaryParser()
	parsed, err := parser.Parse(buf.Bytes())
	if err != nil {
		t.Fatalf("failed to parse binary output: %v", err)
	}

	flat := i18n.FlattenKeys(parsed)
	if flat["greeting"] != "" {
		t.Errorf("expected empty value for greeting, got %q", flat["greeting"])
	}
	if flat["farewell"] != "Goodbye" {
		t.Errorf("expected 'Goodbye' for farewell, got %q", flat["farewell"])
	}
}

func TestGenerateInit_Scaffold(t *testing.T) {
	keys := []extractor.ExtractedKey{
		{Key: "error.required", Kind: extractor.KindWithDefault, DefaultValue: "This field is required"},
		{Key: "greeting", Kind: extractor.KindStandard},
	}

	var buf bytes.Buffer
	err := GenerateInit(keys, "mypackage", &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()

	// Verify it is valid Go source.
	if _, err := format.Source([]byte(output)); err != nil {
		t.Fatalf("output is not valid Go: %v\noutput:\n%s", err, output)
	}

	// Check package declaration.
	if !strings.Contains(output, "package mypackage") {
		t.Error("expected package declaration 'package mypackage'")
	}

	// Check import.
	if !strings.Contains(output, `"github.com/0verkilll/i18n"`) {
		t.Error("expected i18n import")
	}

	// Check PackageTranslator with namespace.
	if !strings.Contains(output, `i18n.NewPackageTranslator("mypackage"`) {
		t.Error("expected NewPackageTranslator call with namespace")
	}

	// Check TD default is populated.
	if !strings.Contains(output, `"error.required": "This field is required"`) {
		t.Error("expected TD default in WithDefaults map")
	}

	// Check non-TD key has empty placeholder.
	if !strings.Contains(output, `"greeting":`) {
		t.Error("expected greeting key in WithDefaults map")
	}
}

func TestWriteToFile_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "output.json")

	keys := []extractor.ExtractedKey{
		{Key: "hello", Kind: extractor.KindStandard},
	}

	err := WriteToFile(path, func(w io.Writer) error {
		return GenerateJSON(keys, w)
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading output file: %v", err)
	}

	var parsed map[string]string
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if _, ok := parsed["hello"]; !ok {
		t.Error("expected key 'hello' in output file")
	}
}

func TestWriteToFile_RefusesOverwrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "existing.json")

	// Create the file first.
	if err := os.WriteFile(path, []byte("{}"), 0o644); err != nil {
		t.Fatalf("creating file: %v", err)
	}

	err := WriteToFile(path, func(w io.Writer) error {
		return GenerateJSON(nil, w)
	})
	if err == nil {
		t.Fatal("expected error when file already exists")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("expected 'already exists' in error, got: %v", err)
	}
}

func TestGenerateStruct_GoFormatClean(t *testing.T) {
	keys := []extractor.ExtractedKey{
		{Key: "app.title", Kind: extractor.KindStandard},
		{Key: "nav.home", Kind: extractor.KindStandard},
	}

	var buf bytes.Buffer
	err := GenerateStruct(keys, "ui", &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the output is already formatted (idempotent go/format).
	original := buf.Bytes()
	reformatted, err := format.Source(original)
	if err != nil {
		t.Fatalf("go/format failed: %v", err)
	}
	if string(original) != string(reformatted) {
		t.Error("struct output is not go/format clean")
	}
}

func FuzzJSONGeneration(f *testing.F) {
	f.Add("hello")
	f.Add("error.required")
	f.Add("a.b.c.d")
	f.Add("")

	f.Fuzz(func(t *testing.T, key string) {
		keys := []extractor.ExtractedKey{
			{Key: key, Kind: extractor.KindStandard},
		}
		var buf bytes.Buffer
		err := GenerateJSON(keys, &buf)
		if err != nil {
			return
		}
		// Output must be valid JSON if no error.
		if !json.Valid(buf.Bytes()) {
			t.Errorf("invalid JSON output for key %q", key)
		}
	})
}

func FuzzStructGeneration(f *testing.F) {
	f.Add("hello")
	f.Add("error.required")
	f.Add("a_b_c")

	f.Fuzz(func(t *testing.T, key string) {
		keys := []extractor.ExtractedKey{
			{Key: key, Kind: extractor.KindStandard},
		}
		var buf bytes.Buffer
		err := GenerateStruct(keys, "main", &buf)
		if err != nil {
			return
		}
		// Output must pass go/format if no error.
		if _, fmtErr := format.Source(buf.Bytes()); fmtErr != nil {
			t.Errorf("go/format failed for key %q: %v", key, fmtErr)
		}
	})
}

// containsStructField checks whether the output contains a struct field with
// the given name followed by a string type on the same line. This handles
// go/format tab alignment where multiple whitespace characters may separate
// the field name from its type.
func containsStructField(output, fieldName string) bool {
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, fieldName) && strings.HasSuffix(trimmed, "string") {
			return true
		}
	}
	return false
}
