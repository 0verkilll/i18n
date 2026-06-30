// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

package engine

import (
	"errors"
	"testing"

	"github.com/0verkilll/i18n/internal/core"
)

// =============================================================================
// BinaryParser.Parse tests
// =============================================================================

func TestBinaryParser_Parse(t *testing.T) {
	// Build valid binary data: two entries.
	flat := map[string]string{
		"greeting": "Hello",
		"farewell": "Goodbye",
	}
	data, err := EncodeBinary(flat)
	if err != nil {
		t.Fatalf("EncodeBinary: %v", err)
	}

	p := NewBinaryParser()
	result, err := p.Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	// Verify top-level keys exist.
	if v, ok := result["greeting"].(string); !ok || v != "Hello" {
		t.Errorf("greeting = %v, want Hello", result["greeting"])
	}
	if v, ok := result["farewell"].(string); !ok || v != "Goodbye" {
		t.Errorf("farewell = %v, want Goodbye", result["farewell"])
	}
}

func TestBinaryParser_Parse_NestedKeys(t *testing.T) {
	flat := map[string]string{
		"error.validation.required": "This field is required",
		"error.validation.email":    "Invalid email",
		"error.network.timeout":     "Timed out",
		"greeting":                  "Hello",
	}
	data, err := EncodeBinary(flat)
	if err != nil {
		t.Fatalf("EncodeBinary: %v", err)
	}

	p := NewBinaryParser()
	result, err := p.Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	// Verify nested structure was reconstructed.
	errMap, ok := result["error"].(map[string]interface{})
	if !ok {
		t.Fatal("error key is not a map")
	}
	valMap, ok := errMap["validation"].(map[string]interface{})
	if !ok {
		t.Fatal("error.validation key is not a map")
	}
	if v, ok := valMap["required"].(string); !ok || v != "This field is required" {
		t.Errorf("error.validation.required = %v, want 'This field is required'", valMap["required"])
	}

	// Verify the resolver can find keys.
	resolver := NewDefaultKeyResolver()
	val, err := resolver.Resolve(result, "error.validation.required")
	if err != nil {
		t.Errorf("Resolve error.validation.required: %v", err)
	}
	if val != "This field is required" {
		t.Errorf("Resolve = %q, want 'This field is required'", val)
	}

	val, err = resolver.Resolve(result, "greeting")
	if err != nil {
		t.Errorf("Resolve greeting: %v", err)
	}
	if val != "Hello" {
		t.Errorf("Resolve greeting = %q, want 'Hello'", val)
	}
}

func TestBinaryParser_InvalidMagic(t *testing.T) {
	data := []byte{0x00, 0x00, 0x01, 0x00, 0x00}

	p := NewBinaryParser()
	_, err := p.Parse(data)
	if err == nil {
		t.Fatal("expected error for invalid magic bytes")
	}

	var fmtErr *core.ErrInvalidFormat
	if !errors.As(err, &fmtErr) {
		t.Errorf("expected ErrInvalidFormat, got %T: %v", err, err)
	}
}

func TestBinaryParser_EmptyInput(t *testing.T) {
	p := NewBinaryParser()
	_, err := p.Parse(nil)
	if err == nil {
		t.Fatal("expected error for nil input")
	}

	_, err = p.Parse([]byte{})
	if err == nil {
		t.Fatal("expected error for empty input")
	}
}

func TestBinaryParser_TruncatedData(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "header only with entries expected",
			data: []byte{binMagic0, binMagic1, binVersion, 0x00, 0x01},
		},
		{
			name: "truncated after key length",
			data: []byte{binMagic0, binMagic1, binVersion, 0x00, 0x01, 0x05},
		},
		{
			name: "truncated in key",
			data: []byte{binMagic0, binMagic1, binVersion, 0x00, 0x01, 0x05, 'h', 'e'},
		},
		{
			name: "truncated before value length",
			data: []byte{binMagic0, binMagic1, binVersion, 0x00, 0x01, 0x02, 'h', 'i'},
		},
		{
			name: "truncated in value",
			data: []byte{binMagic0, binMagic1, binVersion, 0x00, 0x01, 0x02, 'h', 'i', 0x00, 0x05, 'a'},
		},
	}

	p := NewBinaryParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := p.Parse(tt.data)
			if err == nil {
				t.Error("expected error for truncated data")
			}
		})
	}
}

func TestBinaryParser_UnsupportedVersion(t *testing.T) {
	data := []byte{binMagic0, binMagic1, 0xFF, 0x00, 0x00}

	p := NewBinaryParser()
	_, err := p.Parse(data)
	if err == nil {
		t.Fatal("expected error for unsupported version")
	}
}

func TestBinaryParser_EmptyEntries(t *testing.T) {
	// Zero entries is valid.
	data := []byte{binMagic0, binMagic1, binVersion, 0x00, 0x00}

	p := NewBinaryParser()
	result, err := p.Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty map, got %d entries", len(result))
	}
}

// =============================================================================
// EncodeBinary and round-trip tests
// =============================================================================

func TestEncodeBinary_RoundTrip(t *testing.T) {
	original := map[string]string{
		"greeting":                  "Hello",
		"farewell":                  "Goodbye",
		"error.validation.required": "This field is required",
		"error.validation.email":    "Please enter a valid email",
		"items.one":                 "# item",
		"items.other":               "# items",
		"items.zero":                "No items",
	}

	encoded, err := EncodeBinary(original)
	if err != nil {
		t.Fatalf("EncodeBinary: %v", err)
	}

	// Verify header.
	if encoded[0] != binMagic0 || encoded[1] != binMagic1 {
		t.Error("invalid magic bytes")
	}
	if encoded[2] != binVersion {
		t.Errorf("version = %d, want %d", encoded[2], binVersion)
	}

	// Parse back and verify via resolver.
	p := NewBinaryParser()
	parsed, err := p.Parse(encoded)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	resolver := NewDefaultKeyResolver()
	for key, want := range original {
		got, err := resolver.Resolve(parsed, key)
		if err != nil {
			t.Errorf("Resolve(%q): %v", key, err)
			continue
		}
		if got != want {
			t.Errorf("Resolve(%q) = %q, want %q", key, got, want)
		}
	}
}

func TestEncodeBinary_KeyTooLong(t *testing.T) {
	longKey := ""
	for i := 0; i < 256; i++ {
		longKey += "a"
	}
	translations := map[string]string{longKey: "value"}

	_, err := EncodeBinary(translations)
	if err == nil {
		t.Fatal("expected error for key exceeding 255 bytes")
	}
}

func TestEncodeBinary_EmptyMap(t *testing.T) {
	encoded, err := EncodeBinary(map[string]string{})
	if err != nil {
		t.Fatalf("EncodeBinary: %v", err)
	}
	if len(encoded) != binHeaderSize {
		t.Errorf("expected %d bytes, got %d", binHeaderSize, len(encoded))
	}

	p := NewBinaryParser()
	result, err := p.Parse(encoded)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty map, got %d entries", len(result))
	}
}

// =============================================================================
// FlattenKeys tests
// =============================================================================

func TestFlattenKeys(t *testing.T) {
	nested := map[string]interface{}{
		"greeting": "Hello",
		"error": map[string]interface{}{
			"validation": map[string]interface{}{
				"required": "This field is required",
				"email":    "Invalid email",
			},
			"network": map[string]interface{}{
				"timeout": "Timed out",
			},
		},
		"button": map[string]interface{}{
			"submit": "Submit",
			"cancel": "Cancel",
		},
	}

	flat := FlattenKeys(nested)

	want := map[string]string{
		"greeting":                  "Hello",
		"error.validation.required": "This field is required",
		"error.validation.email":    "Invalid email",
		"error.network.timeout":     "Timed out",
		"button.submit":             "Submit",
		"button.cancel":             "Cancel",
	}

	if len(flat) != len(want) {
		t.Errorf("FlattenKeys returned %d keys, want %d", len(flat), len(want))
	}

	for k, wantVal := range want {
		if gotVal, ok := flat[k]; !ok {
			t.Errorf("missing key %q", k)
		} else if gotVal != wantVal {
			t.Errorf("flat[%q] = %q, want %q", k, gotVal, wantVal)
		}
	}
}

func TestFlattenKeys_NonStringLeaf(t *testing.T) {
	nested := map[string]interface{}{
		"count":   42,
		"enabled": true,
	}
	flat := FlattenKeys(nested)
	if flat["count"] != "42" {
		t.Errorf("count = %q, want '42'", flat["count"])
	}
	if flat["enabled"] != "true" {
		t.Errorf("enabled = %q, want 'true'", flat["enabled"])
	}
}

func TestFlattenKeys_Empty(t *testing.T) {
	flat := FlattenKeys(map[string]interface{}{})
	if len(flat) != 0 {
		t.Errorf("expected empty map, got %d entries", len(flat))
	}
}

// =============================================================================
// Full JSON-to-binary round-trip test
// =============================================================================

func TestBinaryParser_JSONRoundTrip(t *testing.T) {
	// Parse JSON, flatten, encode to binary, parse binary, verify.
	jsonData := []byte(`{
		"greeting": "Hello",
		"error": {
			"validation": {
				"required": "This field is required"
			}
		}
	}`)

	jp := NewJSONParser()
	nested, err := jp.Parse(jsonData)
	if err != nil {
		t.Fatalf("JSON Parse: %v", err)
	}

	flat := FlattenKeys(nested)
	binData, err := EncodeBinary(flat)
	if err != nil {
		t.Fatalf("EncodeBinary: %v", err)
	}

	bp := NewBinaryParser()
	result, err := bp.Parse(binData)
	if err != nil {
		t.Fatalf("Binary Parse: %v", err)
	}

	resolver := NewDefaultKeyResolver()

	val, err := resolver.Resolve(result, "greeting")
	if err != nil {
		t.Errorf("Resolve greeting: %v", err)
	}
	if val != "Hello" {
		t.Errorf("greeting = %q, want 'Hello'", val)
	}

	val, err = resolver.Resolve(result, "error.validation.required")
	if err != nil {
		t.Errorf("Resolve error.validation.required: %v", err)
	}
	if val != "This field is required" {
		t.Errorf("error.validation.required = %q, want 'This field is required'", val)
	}
}

// =============================================================================
// Benchmark
// =============================================================================

func BenchmarkBinaryParser_Parse(b *testing.B) {
	flat := map[string]string{
		"greeting":                  "Hello",
		"farewell":                  "Goodbye",
		"welcome":                   "Welcome to our application",
		"error.validation.required": "This field is required",
		"error.validation.email":    "Please enter a valid email address",
		"error.network.timeout":     "Request timed out. Please try again.",
		"button.submit":             "Submit",
		"button.cancel":             "Cancel",
		"messages.success":          "Operation completed successfully",
		"items.one":                 "# item",
		"items.other":               "# items",
		"items.zero":                "No items",
	}

	data, err := EncodeBinary(flat)
	if err != nil {
		b.Fatalf("EncodeBinary: %v", err)
	}

	p := NewBinaryParser()
	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		_, err := p.Parse(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkJSONParser_Parse(b *testing.B) {
	data := []byte(`{
		"greeting": "Hello",
		"farewell": "Goodbye",
		"welcome": "Welcome to our application",
		"error": {
			"validation": {
				"required": "This field is required",
				"email": "Please enter a valid email address"
			},
			"network": {
				"timeout": "Request timed out. Please try again."
			}
		},
		"button": {
			"submit": "Submit",
			"cancel": "Cancel"
		},
		"messages": {
			"success": "Operation completed successfully"
		},
		"items": {
			"one": "# item",
			"other": "# items",
			"zero": "No items"
		}
	}`)

	p := NewJSONParser()
	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		_, err := p.Parse(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// =============================================================================
// sortStrings test
// =============================================================================

func TestSortStrings(t *testing.T) {
	input := []string{"cherry", "apple", "banana", "date"}
	sortStrings(input)
	want := []string{"apple", "banana", "cherry", "date"}
	for i, v := range input {
		if v != want[i] {
			t.Errorf("index %d: got %q, want %q", i, v, want[i])
		}
	}
}

func TestSortStrings_Empty(t *testing.T) {
	var s []string
	sortStrings(s) // should not panic
}

func TestSortStrings_Single(t *testing.T) {
	s := []string{"only"}
	sortStrings(s)
	if s[0] != "only" {
		t.Errorf("got %q, want 'only'", s[0])
	}
}
