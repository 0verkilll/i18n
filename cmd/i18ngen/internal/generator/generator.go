// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

package generator

import (
	"encoding/json"
	"fmt"
	"go/format"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/0verkilll/i18n"
	"github.com/0verkilll/i18n/cmd/i18ngen/internal/extractor"
)

// GenerateJSON writes a flat JSON template with sorted keys and empty string
// values. If an ExtractedKey has a non-empty DefaultValue, that value is used
// instead. Struct field keys are included without any prefix.
func GenerateJSON(keys []extractor.ExtractedKey, w io.Writer) error {
	m := buildKeyMap(keys)
	sorted := sortedMapKeys(m)

	output, err := buildJSONString(sorted, m)
	if err != nil {
		return fmt.Errorf("encoding JSON: %w", err)
	}
	_, err = io.WriteString(w, output)
	if err != nil {
		return fmt.Errorf("writing JSON output: %w", err)
	}
	return nil
}

// buildJSONString encodes the key-value pairs as indented JSON with
// SetEscapeHTML(false) for consistency with the i18nlint reporter pattern.
func buildJSONString(sorted []string, m map[string]string) (string, error) {
	var buf strings.Builder
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")

	ordered := make(orderedMap, 0, len(sorted))
	for _, k := range sorted {
		ordered = append(ordered, mapEntry{Key: k, Value: m[k]})
	}
	if err := enc.Encode(ordered); err != nil {
		return "", fmt.Errorf("encoding ordered map: %w", err)
	}
	return buf.String(), nil
}

// GenerateStruct writes a Go source file containing a Messages struct with one
// exported field per key. Dot-separated keys are converted to PascalCase field
// names. The output is formatted with go/format.Source.
func GenerateStruct(keys []extractor.ExtractedKey, packageName string, w io.Writer) error {
	m := buildKeyMap(keys)
	sorted := sortedMapKeys(m)
	fields := buildStructFields(sorted)

	src := formatStructSource(packageName, fields)
	formatted, err := format.Source([]byte(src))
	if err != nil {
		return fmt.Errorf("formatting struct source: %w", err)
	}
	_, err = w.Write(formatted)
	if err != nil {
		return fmt.Errorf("writing struct output: %w", err)
	}
	return nil
}

// GenerateBinary encodes the extracted keys into the compact binary format
// using i18n.EncodeBinary.
func GenerateBinary(keys []extractor.ExtractedKey, w io.Writer) error {
	m := buildKeyMap(keys)
	data, err := i18n.EncodeBinary(m)
	if err != nil {
		return fmt.Errorf("encoding binary: %w", err)
	}
	_, err = w.Write(data)
	if err != nil {
		return fmt.Errorf("writing binary output: %w", err)
	}
	return nil
}

// GenerateInit writes a complete i18n.go scaffold file with a
// PackageTranslator declaration. TD-extracted keys use their default values;
// all other keys get empty string placeholders.
func GenerateInit(keys []extractor.ExtractedKey, packageName string, w io.Writer) error {
	m := buildKeyMap(keys)
	sorted := sortedMapKeys(m)
	entries := buildInitEntries(sorted, m)

	src := formatInitSource(packageName, entries)
	formatted, err := format.Source([]byte(src))
	if err != nil {
		return fmt.Errorf("formatting init source: %w", err)
	}
	_, err = w.Write(formatted)
	if err != nil {
		return fmt.Errorf("writing init output: %w", err)
	}
	return nil
}

// WriteToFile writes content from a generate function to a file path. It
// returns an error if the file already exists (no silent overwrite).
func WriteToFile(path string, fn func(io.Writer) error) error {
	_, err := os.Stat(path)
	if err == nil {
		return fmt.Errorf("file %q already exists: refusing to overwrite", path)
	}
	if !os.IsNotExist(err) {
		return fmt.Errorf("checking file %q: %w", path, err)
	}

	return writeNewFile(path, fn)
}

// writeNewFile creates a new file and writes content to it. The caller must
// ensure the file does not already exist.
func writeNewFile(path string, fn func(io.Writer) error) (retErr error) {
	f, err := os.Create(path) //nolint:gosec // path validated by caller via os.Stat
	if err != nil {
		return fmt.Errorf("creating file %q: %w", path, err)
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil && retErr == nil {
			retErr = fmt.Errorf("closing file %q: %w", path, closeErr)
		}
	}()

	return fn(f)
}

// buildKeyMap builds a map of key to value from extracted keys. Struct field
// keys are included with their original name (no "struct:" prefix).
func buildKeyMap(keys []extractor.ExtractedKey) map[string]string {
	m := make(map[string]string, len(keys))
	for _, k := range keys {
		val := ""
		if k.DefaultValue != "" {
			val = k.DefaultValue
		}
		m[k.Key] = val
	}
	return m
}

// sortedMapKeys returns the keys of a map in sorted order.
func sortedMapKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// toPascalCase converts a dot-separated or underscore-separated key to
// PascalCase. For example, "error.not_found" becomes "ErrorNotFound".
func toPascalCase(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	upper := true
	for _, c := range s {
		if c == '.' || c == '_' || c == '-' {
			upper = true
			continue
		}
		if upper {
			b.WriteRune(toUpper(c))
			upper = false
		} else {
			b.WriteRune(c)
		}
	}
	return b.String()
}

// toUpper converts a lowercase ASCII letter to uppercase. Non-lowercase
// characters are returned unchanged.
func toUpper(c rune) rune {
	if c >= 'a' && c <= 'z' {
		return c - 'a' + 'A'
	}
	return c
}

// buildStructFields produces the struct field lines for a Messages struct.
func buildStructFields(sorted []string) []string {
	fields := make([]string, 0, len(sorted))
	for _, key := range sorted {
		fieldName := toPascalCase(key)
		fields = append(fields, fmt.Sprintf("\t%s string", fieldName))
	}
	return fields
}

// formatStructSource produces the raw Go source for a Messages struct.
func formatStructSource(packageName string, fields []string) string {
	var b strings.Builder
	b.WriteString("package " + packageName + "\n\n")
	b.WriteString("// Messages holds all translated strings for this package.\n")
	b.WriteString("type Messages struct {\n")
	for _, f := range fields {
		b.WriteString(f + "\n")
	}
	b.WriteString("}\n")
	return b.String()
}

// buildInitEntries builds the map literal entries for the init scaffold.
func buildInitEntries(sorted []string, m map[string]string) []string {
	entries := make([]string, 0, len(sorted))
	for _, key := range sorted {
		val := m[key]
		entries = append(entries, fmt.Sprintf("\t\t%q: %q,", key, val))
	}
	return entries
}

// formatInitSource produces the raw Go source for an i18n.go init scaffold.
func formatInitSource(packageName string, entries []string) string {
	var b strings.Builder
	b.WriteString("package " + packageName + "\n\n")
	b.WriteString("import i18n \"github.com/0verkilll/i18n\"\n\n")
	b.WriteString("var I18n = i18n.NewPackageTranslator(\"" + packageName + "\",\n")
	b.WriteString("\ti18n.WithDefaults(map[string]string{\n")
	for _, e := range entries {
		b.WriteString(e + "\n")
	}
	b.WriteString("\t}),\n")
	b.WriteString(")\n")
	return b.String()
}

// orderedMap is a JSON-encodable type that preserves key insertion order.
type orderedMap []mapEntry

// mapEntry is a single key-value pair in an orderedMap.
type mapEntry struct {
	Key   string
	Value string
}

// MarshalJSON produces a JSON object with keys in insertion order.
func (om orderedMap) MarshalJSON() ([]byte, error) {
	var b strings.Builder
	b.WriteByte('{')
	for i, e := range om {
		if i > 0 {
			b.WriteByte(',')
		}
		keyBytes, err := json.Marshal(e.Key)
		if err != nil {
			return nil, fmt.Errorf("marshaling key %q: %w", e.Key, err)
		}
		valBytes, err := json.Marshal(e.Value)
		if err != nil {
			return nil, fmt.Errorf("marshaling value for key %q: %w", e.Key, err)
		}
		b.Write(keyBytes)
		b.WriteByte(':')
		b.Write(valBytes)
	}
	b.WriteByte('}')
	return []byte(b.String()), nil
}
