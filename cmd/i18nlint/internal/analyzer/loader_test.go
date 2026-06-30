// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

package analyzer

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestIssueFieldPopulationAndSorting(t *testing.T) {
	issues := []Issue{
		{File: "z.go", Line: 10, Col: 5, Severity: SeverityError, Code: CodeMissingKey, Message: "missing", Key: "a"},
		{File: "a.go", Line: 20, Col: 3, Severity: SeverityWarning, Code: CodeUnusedKey, Message: "unused", Key: "b"},
		{File: "a.go", Line: 10, Col: 8, Severity: SeverityError, Code: CodeMissingKey, Message: "missing2", Key: "c"},
		{File: "a.go", Line: 10, Col: 2, Severity: SeverityError, Code: CodeMissingKey, Message: "missing3", Key: "d"},
	}

	sortIssues(issues)

	// Sorted: a.go:10:2, a.go:10:8, a.go:20:3, z.go:10:5
	if issues[0].File != "a.go" || issues[0].Line != 10 || issues[0].Col != 2 {
		t.Errorf("expected first issue a.go:10:2, got %s:%d:%d", issues[0].File, issues[0].Line, issues[0].Col)
	}
	if issues[1].File != "a.go" || issues[1].Line != 10 || issues[1].Col != 8 {
		t.Errorf("expected second issue a.go:10:8, got %s:%d:%d", issues[1].File, issues[1].Line, issues[1].Col)
	}
	if issues[2].File != "a.go" || issues[2].Line != 20 || issues[2].Col != 3 {
		t.Errorf("expected third issue a.go:20:3, got %s:%d:%d", issues[2].File, issues[2].Line, issues[2].Col)
	}
	if issues[3].File != "z.go" || issues[3].Line != 10 || issues[3].Col != 5 {
		t.Errorf("expected fourth issue z.go:10:5, got %s:%d:%d", issues[3].File, issues[3].Line, issues[3].Col)
	}
}

func TestFlattenKeysNestedMap(t *testing.T) {
	m := map[string]interface{}{
		"error": map[string]interface{}{
			"timeout": "Request timed out",
			"network": map[string]interface{}{
				"offline": "You are offline",
			},
		},
		"greeting": "Hello",
	}

	data := newLocaleKeyData()
	flattenKeys(m, "", "test.json", data)

	expected := []string{"error.timeout", "error.network.offline", "greeting"}
	for _, key := range expected {
		if _, ok := data.keys[key]; !ok {
			t.Errorf("expected key %q to be present", key)
		}
	}

	if len(data.keys) != 3 {
		t.Errorf("expected 3 keys, got %d", len(data.keys))
	}
}

func TestFlattenKeysLeafTypes(t *testing.T) {
	m := map[string]interface{}{
		"str":     "hello",
		"num_int": float64(42),
		"num_flt": 3.14,
		"flag":    true,
		"empty":   nil,
	}

	data := newLocaleKeyData()
	flattenKeys(m, "", "test.json", data)

	// All 5 types should be treated as leaf nodes.
	if len(data.keys) != 5 {
		t.Errorf("expected 5 leaf keys, got %d", len(data.keys))
	}

	expectedValues := map[string]string{
		"str":     "hello",
		"num_int": "42",
		"num_flt": "3.14",
		"flag":    "true",
		"empty":   "",
	}

	for key, want := range expectedValues {
		got := data.values[key]
		if got != want {
			t.Errorf("key %q: want value %q, got %q", key, want, got)
		}
	}
}

func TestFlattenKeysSkipsArrays(t *testing.T) {
	m := map[string]interface{}{
		"valid":   "yes",
		"invalid": []interface{}{"a", "b"},
		"nested": map[string]interface{}{
			"arr": []interface{}{1, 2},
			"ok":  "fine",
		},
	}

	data := newLocaleKeyData()
	flattenKeys(m, "", "test.json", data)

	// Only "valid" and "nested.ok" should be present.
	if len(data.keys) != 2 {
		t.Errorf("expected 2 keys, got %d", len(data.keys))
	}

	if _, ok := data.keys["invalid"]; ok {
		t.Error("array key should not be flattened")
	}
	if _, ok := data.keys["nested.arr"]; ok {
		t.Error("nested array key should not be flattened")
	}
}

func TestFormatSpecifierCounting(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{name: "no specifiers", input: "Hello world", want: 0},
		{name: "one specifier", input: "Hello %s", want: 1},
		{name: "two specifiers", input: "%s has %d items", want: 2},
		{name: "escaped percent", input: "100%% complete", want: 0},
		{name: "mixed escaped and real", input: "100%% %s done", want: 1},
		{name: "double escaped with specifier", input: "100%% %s and %d%%", want: 2},
		{name: "only escaped percents", input: "100%% 50%%", want: 0},
		{name: "verb specifiers", input: "%v %t %f", want: 3},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := countFormatSpecifiers(tc.input)
			if got != tc.want {
				t.Errorf("countFormatSpecifiers(%q) = %d, want %d", tc.input, got, tc.want)
			}
		})
	}
}

func TestLoadMultipleLocaleFilesMerge(t *testing.T) {
	// Create temp directory with two locale files.
	tmpDir := t.TempDir()

	enUS := `{
		"greeting": "Hello",
		"farewell": "Goodbye",
		"error": {"timeout": "Timed out"}
	}`

	esMX := `{
		"greeting": "Hola",
		"error": {"timeout": "Se agoto"}
	}`

	writeFile(t, filepath.Join(tmpDir, "en-US.json"), enUS)
	writeFile(t, filepath.Join(tmpDir, "es-MX.json"), esMX)

	data, err := loadLocaleFiles([]string{tmpDir})
	if err != nil {
		t.Fatalf("loadLocaleFiles() error: %v", err)
	}

	// "greeting" should appear in both files.
	if len(data.keys["greeting"]) != 2 {
		t.Errorf("expected greeting in 2 files, got %d", len(data.keys["greeting"]))
	}

	// "farewell" should appear only in en-US.
	if len(data.keys["farewell"]) != 1 {
		t.Errorf("expected farewell in 1 file, got %d", len(data.keys["farewell"]))
	}

	// error.timeout should appear in both.
	if len(data.keys["error.timeout"]) != 2 {
		t.Errorf("expected error.timeout in 2 files, got %d", len(data.keys["error.timeout"]))
	}

	// 3 unique keys total.
	if len(data.keys) != 3 {
		keys := make([]string, 0, len(data.keys))
		for k := range data.keys {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		t.Errorf("expected 3 unique keys, got %d: %v", len(data.keys), keys)
	}
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing test file %s: %v", path, err)
	}
}
