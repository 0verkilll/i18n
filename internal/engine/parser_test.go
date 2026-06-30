// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

package engine

import (
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/0verkilll/i18n/internal/core"
)

// =============================================================================
// JSON parser tests
// =============================================================================

func TestJSONParser_Parse(t *testing.T) {
	tests := []struct {
		errType  error
		name     string
		input    string
		checkKey string
		wantVal  string
		wantErr  bool
	}{
		{
			name:     "simple flat JSON",
			input:    `{"greeting": "Hello", "farewell": "Goodbye"}`,
			wantErr:  false,
			checkKey: "greeting",
			wantVal:  "Hello",
		},
		{
			name:     "nested JSON",
			input:    `{"error": {"validation": {"required": "Field is required"}}}`,
			wantErr:  false,
			checkKey: "error",
		},
		{
			name:     "with numbers",
			input:    `{"count": 42, "message": "Items"}`,
			wantErr:  false,
			checkKey: "message",
			wantVal:  "Items",
		},
		{
			name:     "with booleans",
			input:    `{"enabled": true, "name": "Feature"}`,
			wantErr:  false,
			checkKey: "name",
			wantVal:  "Feature",
		},
		{
			name:     "with arrays",
			input:    `{"items": ["one", "two"], "title": "List"}`,
			wantErr:  false,
			checkKey: "title",
			wantVal:  "List",
		},
		{
			name:     "unicode characters",
			input:    `{"greeting": "Hola!", "emoji": "wave"}`,
			wantErr:  false,
			checkKey: "greeting",
			wantVal:  "Hola!",
		},
		{
			name:     "escaped characters",
			input:    `{"quote": "He said \"Hello\"", "newline": "Line1\nLine2"}`,
			wantErr:  false,
			checkKey: "quote",
		},
		{
			name:    "empty JSON object",
			input:   `{}`,
			wantErr: false,
		},
		{
			name:    "invalid JSON - missing brace",
			input:   `{"greeting": "Hello"`,
			wantErr: true,
			errType: &core.ErrInvalidFormat{},
		},
		{
			name:    "invalid JSON - trailing comma",
			input:   `{"greeting": "Hello",}`,
			wantErr: true,
			errType: &core.ErrInvalidFormat{},
		},
		{
			name:    "invalid JSON - single quotes",
			input:   `{'greeting': 'Hello'}`,
			wantErr: true,
			errType: &core.ErrInvalidFormat{},
		},
		{
			name:    "empty input",
			input:   ``,
			wantErr: true,
			errType: &core.ErrInvalidFormat{},
		},
		{
			name:    "not a JSON object - array",
			input:   `["item1", "item2"]`,
			wantErr: true,
			errType: &core.ErrInvalidFormat{},
		},
		{
			name:    "not a JSON object - string",
			input:   `"just a string"`,
			wantErr: true,
			errType: &core.ErrInvalidFormat{},
		},
		{
			name:    "not a JSON object - number",
			input:   `42`,
			wantErr: true,
			errType: &core.ErrInvalidFormat{},
		},
		{
			name:    "deeply nested - 10 levels",
			input:   `{"a":{"b":{"c":{"d":{"e":{"f":{"g":{"h":{"i":{"j":"value"}}}}}}}}}}`,
			wantErr: false,
		},
		{
			name:    "whitespace only",
			input:   `   `,
			wantErr: true,
			errType: &core.ErrInvalidFormat{},
		},
		// Merged from coverage_test.go
		{
			name:    "null at root",
			input:   `null`,
			wantErr: true,
			errType: &core.ErrInvalidFormat{},
		},
		{
			name:    "bool at root",
			input:   `true`,
			wantErr: true,
			errType: &core.ErrInvalidFormat{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewJSONParser()
			result, err := parser.Parse([]byte(tt.input))

			if tt.wantErr {
				if err == nil {
					t.Errorf("Parse() expected error, got nil")
				}
				if tt.errType != nil {
					var formatErr *core.ErrInvalidFormat
					if !errors.As(err, &formatErr) {
						t.Errorf("Parse() error should be core.ErrInvalidFormat, got: %v", err)
					}
				}
				return
			}

			if err != nil {
				t.Errorf("Parse() unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Error("Parse() returned nil result")
				return
			}

			if tt.checkKey != "" {
				val, ok := result[tt.checkKey]
				if !ok {
					t.Errorf("Parse() result missing key %q", tt.checkKey)
				}
				if tt.wantVal != "" {
					if strVal, ok := val.(string); ok {
						if strVal != tt.wantVal {
							t.Errorf("Parse() result[%q] = %q, want %q", tt.checkKey, strVal, tt.wantVal)
						}
					}
				}
			}
		})
	}
}

func TestJSONParser_ParseNestingDepth(t *testing.T) {
	parser := NewJSONParser()

	deep := strings.Repeat(`{"level":`, 100)
	deep += `"value"`
	deep += strings.Repeat(`}`, 100)

	_, err := parser.Parse([]byte(deep))
	if err == nil {
		t.Error("Parse() should reject excessively nested JSON")
	}

	var formatErr *core.ErrInvalidFormat
	if !errors.As(err, &formatErr) {
		t.Errorf("Parse() should return core.ErrInvalidFormat for deep nesting, got: %v", err)
	}
}

func TestJSONParser_ParseSizeLimit(t *testing.T) {
	parser := NewJSONParser()

	largeValue := strings.Repeat("a", 11*1024*1024)
	largeJSON := `{"data":"` + largeValue + `"}`

	_, err := parser.Parse([]byte(largeJSON))
	if err == nil {
		t.Error("Parse() should reject excessively large JSON")
	}

	var formatErr *core.ErrInvalidFormat
	if !errors.As(err, &formatErr) {
		t.Errorf("Parse() should return core.ErrInvalidFormat for oversized input, got: %v", err)
	}
}

func TestJSONParser_ParseBinaryData(t *testing.T) {
	parser := NewJSONParser()

	binaryData := []byte{0xFF, 0xFE, 0xFD, 0x00}

	_, err := parser.Parse(binaryData)
	if err == nil {
		t.Error("Parse() should reject binary data")
	}
}

func TestJSONParser_ParseSecurityChecks(t *testing.T) {
	parser := NewJSONParser()

	tests := []struct {
		name  string
		input string
	}{
		{name: "null bytes", input: "{\"key\": \"value\x00\"}"},
		{name: "control characters", input: "{\"key\": \"value\x01\x02\"}"},
		{name: "ANSI escape sequences", input: `{"key": "value\u001b[31m"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.Parse([]byte(tt.input))
			if err != nil {
				t.Logf("Parse() error: %v (this may be expected)", err)
			}
			if result != nil && len(result) == 0 {
				t.Error("Parse() returned empty result")
			}
		})
	}
}

func TestJSONParser_ParseNestedStructures(t *testing.T) {
	parser := NewJSONParser()

	complexJSON := `{
		"user": {
			"profile": {
				"name": "John",
				"email": "john@example.com",
				"settings": {"theme": "dark", "language": "en-US"}
			},
			"permissions": ["read", "write"]
		},
		"app": {
			"version": "1.0.0",
			"features": {"auth": true, "notifications": false}
		}
	}`

	result, err := parser.Parse([]byte(complexJSON))
	if err != nil {
		t.Fatalf("Parse() unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("Parse() returned nil")
	}

	user, ok := result["user"]
	if !ok {
		t.Error("Parse() missing 'user' key")
	}
	userMap, ok := user.(map[string]interface{})
	if !ok {
		t.Error("Parse() 'user' is not a map")
	}
	profile, ok := userMap["profile"]
	if !ok {
		t.Error("Parse() missing 'user.profile' key")
	}
	profileMap, ok := profile.(map[string]interface{})
	if !ok {
		t.Error("Parse() 'user.profile' is not a map")
	}
	name, ok := profileMap["name"]
	if !ok {
		t.Error("Parse() missing 'user.profile.name' key")
	}
	if nameStr, ok := name.(string); ok {
		if nameStr != "John" {
			t.Errorf("Parse() name = %q, want %q", nameStr, "John")
		}
	} else {
		t.Error("Parse() name is not a string")
	}
}

// Distributed from fuzz_test.go: FuzzJSONParser
func FuzzJSONParser(f *testing.F) {
	f.Add([]byte(`{"greeting": "Hello"}`))
	f.Add([]byte(`{"nested": {"key": "value"}}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`{"array": [1, 2, 3]}`))
	f.Add([]byte(`{"unicode": "text"}`))
	f.Add([]byte(`{"key": "value with\nnewline"}`))
	f.Add([]byte(`[1, 2, 3]`))
	f.Add([]byte(`"string"`))
	f.Add([]byte(`123`))
	f.Add([]byte(``))
	f.Add([]byte(`{`))
	f.Add([]byte(strings.Repeat(`{"a":`, 100) + `"value"` + strings.Repeat(`}`, 100)))

	parser := NewJSONParser()

	f.Fuzz(func(t *testing.T, input []byte) {
		result, err := parser.Parse(input)

		if err == nil {
			if result == nil {
				t.Error("Parse() returned nil result without error")
			}
		} else {
			var formatErr *core.ErrInvalidFormat
			if !errors.As(err, &formatErr) {
				t.Logf("Parse() returned non-core.ErrInvalidFormat error: %v", err)
			}
		}
	})
}

// Distributed from benchmark_test.go: BenchmarkJSONParser
func BenchmarkJSONParser(b *testing.B) {
	data := []byte(`{
		"greeting": "Hello",
		"user": {
			"profile": {
				"title": "User Profile"
			}
		}
	}`)

	parser := NewJSONParser()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = parser.Parse(data) //nolint:errcheck // benchmark
	}
}

// =============================================================================
// Parser registry tests
// =============================================================================

func TestNewRegistry_Empty(t *testing.T) {
	r := NewRegistry()
	formats := r.RegisteredFormats()
	if len(formats) != 0 {
		t.Errorf("NewRegistry().RegisteredFormats() returned %d formats, want 0", len(formats))
	}
}

func TestRegistry_RegisterParser_Valid(t *testing.T) {
	r := NewRegistry()
	p := NewJSONParser()

	err := r.RegisterParser(".json", p)
	if err != nil {
		t.Fatalf("RegisterParser() error = %v", err)
	}

	got, err := r.GetParser(".json")
	if err != nil {
		t.Fatalf("GetParser() error = %v", err)
	}
	if got == nil {
		t.Fatal("GetParser() returned nil parser")
	}
}

func TestRegistry_RegisterParser_InvalidInput(t *testing.T) {
	r := NewRegistry()
	p := NewJSONParser()

	tests := []struct {
		name string
		ext  string
		p    core.TranslationParser
	}{
		{"missing dot", "json", p},
		{"uppercase", ".JSON", p},
		{"empty", "", p},
		{"dot only", ".", p},
		{"special characters", ".to-ml", p},
		{"space in extension", ".to ml", p},
		{"nil parser", ".toml", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := r.RegisterParser(tt.ext, tt.p)
			if err == nil {
				t.Errorf("RegisterParser(%q, %v) should return an error", tt.ext, tt.p)
			}
		})
	}
}

func TestRegistry_GetParser_Known(t *testing.T) {
	r := NewRegistry()
	p := NewJSONParser()
	_ = r.RegisterParser(".json", p)

	got, err := r.GetParser(".json")
	if err != nil {
		t.Fatalf("GetParser() error = %v", err)
	}
	if got == nil {
		t.Fatal("GetParser() returned nil")
	}
}

func TestRegistry_GetParser_Unknown(t *testing.T) {
	r := NewRegistry()

	_, err := r.GetParser(".toml")
	if err == nil {
		t.Fatal("GetParser() should return error for unregistered extension")
	}
	if !errors.Is(err, core.ErrUnknownFormat{}) {
		t.Errorf("GetParser() error should be core.ErrUnknownFormat, got %T: %v", err, err)
	}
}

func TestRegistry_RegisteredFormats_Sorted(t *testing.T) {
	r := NewRegistry()
	_ = r.RegisterParser(".yaml", NewJSONParser())
	_ = r.RegisterParser(".json", NewJSONParser())
	_ = r.RegisterParser(".toml", NewJSONParser())

	formats := r.RegisteredFormats()
	if len(formats) != 3 {
		t.Fatalf("RegisteredFormats() returned %d formats, want 3", len(formats))
	}

	expected := []string{".json", ".toml", ".yaml"}
	for i, f := range formats {
		if f != expected[i] {
			t.Errorf("RegisteredFormats()[%d] = %q, want %q", i, f, expected[i])
		}
	}
}

// Gap-filling tests for TG6

func TestRegistry_ConcurrentAccess(t *testing.T) {
	r := NewRegistry()
	var wg sync.WaitGroup

	// Concurrent registrations
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			exts := []string{".json", ".toml", ".yaml", ".xml", ".ini"}
			_ = r.RegisterParser(exts[idx%len(exts)], NewJSONParser())
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = r.GetParser(".json")
			_ = r.RegisteredFormats()
		}()
	}

	wg.Wait()

	// Registry should be functional after concurrent access
	formats := r.RegisteredFormats()
	if len(formats) == 0 {
		t.Error("RegisteredFormats() should not be empty after concurrent registrations")
	}
}

func TestDefaultRegistry_InitState(t *testing.T) {
	// The default registry should have .json registered from init()
	formats := RegisteredFormats()
	found := false
	for _, f := range formats {
		if f == ".json" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("default registry should contain .json, got %v", formats)
	}

	// GetParser should return a valid JSON parser
	p, err := GetParser(".json")
	if err != nil {
		t.Fatalf("GetParser(.json) error = %v", err)
	}
	if p == nil {
		t.Fatal("GetParser(.json) returned nil")
	}
}

func TestRegistry_OverwriteBehavior(t *testing.T) {
	r := NewRegistry()
	first := NewJSONParser()
	second := NewJSONParser()

	_ = r.RegisterParser(".json", first)
	_ = r.RegisterParser(".json", second)

	got, err := r.GetParser(".json")
	if err != nil {
		t.Fatalf("GetParser() error = %v", err)
	}

	// The second parser should have overwritten the first
	if got != second {
		t.Error("RegisterParser should overwrite existing parser (last-write-wins)")
	}
}

func TestValidateExtension_BoundaryCases(t *testing.T) {
	tests := []struct {
		name    string
		ext     string
		wantErr bool
	}{
		{"single char", ".a", false},
		{"numeric only", ".123", false},
		{"mixed alphanum", ".mp3", false},
		{"no dot", "json", true},
		{"underscore", ".to_ml", true},
		{"dot in middle", ".to.ml", true},
		{"unicode", ".toml\u00e9", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateExtension(tt.ext)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateExtension(%q) error = %v, wantErr %v", tt.ext, err, tt.wantErr)
			}
		})
	}
}

func TestPackageLevelRegisterParser(t *testing.T) {
	// Package-level convenience functions should delegate to defaultRegistry
	err := RegisterParser(".testpkg", NewJSONParser())
	if err != nil {
		t.Fatalf("RegisterParser() error = %v", err)
	}

	p, err := GetParser(".testpkg")
	if err != nil {
		t.Fatalf("GetParser() error = %v", err)
	}
	if p == nil {
		t.Fatal("GetParser() returned nil")
	}

	formats := RegisteredFormats()
	found := false
	for _, f := range formats {
		if f == ".testpkg" {
			found = true
			break
		}
	}
	if !found {
		t.Error("RegisteredFormats() should include .testpkg")
	}
}

// =============================================================================
// Lazy-init default registry tests (spec 013)
// =============================================================================

func TestGetDefaultRegistry_NonNil(t *testing.T) {
	reg := getDefaultRegistry()
	if reg == nil {
		t.Fatal("getDefaultRegistry() returned nil")
	}
}

func TestGetDefaultRegistry_Singleton(t *testing.T) {
	first := getDefaultRegistry()
	second := getDefaultRegistry()
	if first != second {
		t.Error("getDefaultRegistry() should return the same instance on repeated calls")
	}
}

func TestGetDefaultRegistry_ConcurrentAccess(t *testing.T) {
	const goroutines = 100
	results := make([]*Registry, goroutines)
	var wg sync.WaitGroup

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			results[idx] = getDefaultRegistry()
		}(i)
	}
	wg.Wait()

	first := results[0]
	for i, reg := range results {
		if reg != first {
			t.Errorf("goroutine %d got a different registry instance", i)
		}
	}
}

func TestGetDefaultRegistry_ContainsJSON(t *testing.T) {
	reg := getDefaultRegistry()

	p, err := reg.GetParser(".json")
	if err != nil {
		t.Fatalf("getDefaultRegistry().GetParser(.json) error = %v", err)
	}
	if p == nil {
		t.Fatal("getDefaultRegistry() registry should have .json parser registered")
	}
}

// =============================================================================
// Fuzz targets for parser registration
// =============================================================================

// FuzzRegisterParser exercises RegisterParser extension validation with random inputs.
func FuzzRegisterParser(f *testing.F) {
	f.Add(".json")
	f.Add("")
	f.Add("\x00")
	f.Add(strings.Repeat("a", 110))
	f.Add("no-dot-prefix")
	f.Add(".YAML")

	stub := NewJSONParser()

	f.Fuzz(func(t *testing.T, ext string) {
		r := NewRegistry()
		err := r.RegisterParser(ext, stub)

		if err != nil {
			// Verify the error message is non-empty.
			if err.Error() == "" {
				t.Errorf("RegisterParser(%q) returned error with empty message", ext)
			}
		} else {
			// If no error, we should be able to retrieve the parser.
			p, getErr := r.GetParser(ext)
			if getErr != nil {
				t.Errorf("RegisterParser(%q) succeeded but GetParser failed: %v", ext, getErr)
			}
			if p == nil {
				t.Errorf("RegisterParser(%q) succeeded but GetParser returned nil", ext)
			}
		}
	})
}

// FuzzGetParser exercises GetParser extension lookup with random inputs.
func FuzzGetParser(f *testing.F) {
	f.Add(".json")
	f.Add("")
	f.Add("\x00")
	f.Add(strings.Repeat("a", 110))
	f.Add(".nonexistent")
	f.Add(".")

	f.Fuzz(func(t *testing.T, ext string) {
		r := NewRegistry()
		// Register .json so there is at least one known extension.
		_ = r.RegisterParser(".json", NewJSONParser()) //nolint:errcheck

		p, err := r.GetParser(ext)

		if err != nil {
			// If error, verify it is a typed error via errors.As.
			var unknownErr *core.ErrUnknownFormat
			if !errors.As(err, &unknownErr) {
				// GetParser returns core.ErrUnknownFormat for unknown extensions.
				// Any other error type is unexpected.
				t.Logf("GetParser(%q) returned non-core.ErrUnknownFormat error: %T", ext, err)
			}
		} else {
			// If no error, the returned parser must be non-nil.
			if p == nil {
				t.Errorf("GetParser(%q) returned nil parser without error", ext)
			}
		}
	})
}

// FuzzValidateExtension exercises validateExtension with random inputs.
func FuzzValidateExtension(f *testing.F) {
	f.Add(".json")
	f.Add("")
	f.Add("\x00\x01")
	f.Add(strings.Repeat("a", 110))
	f.Add(".JSON")
	f.Add(".")

	f.Fuzz(func(t *testing.T, ext string) {
		err := validateExtension(ext)

		if err != nil {
			// Verify error message is non-empty.
			if err.Error() == "" {
				t.Errorf("validateExtension(%q) returned error with empty message", ext)
			}
		} else {
			// If valid, the extension must start with a dot and contain only
			// lowercase alphanumeric characters after the dot.
			if len(ext) < 2 {
				t.Errorf("validateExtension(%q) returned nil error for short extension", ext)
			}
			if ext[0] != '.' {
				t.Errorf("validateExtension(%q) returned nil error but ext does not start with dot", ext)
			}
		}
	})
}

// =============================================================================
// Spec 018: Parser Benchmarks (Task Group 9)
// =============================================================================

// benchLargeJSON is a realistic 50+ key JSON document with 3 levels of nesting,
// flat keys, nested keys, plural sub-keys, and ICU MessageFormat values.
var benchLargeJSON = []byte(`{
	"app": {
		"title": "My Application",
		"description": "A wonderful application",
		"version": "2.1.0",
		"copyright": "Copyright 2024"
	},
	"nav": {
		"home": "Home",
		"about": "About",
		"contact": "Contact",
		"settings": "Settings",
		"profile": "Profile",
		"logout": "Logout",
		"help": "Help",
		"search": "Search"
	},
	"user": {
		"greeting": "Hello, %s!",
		"farewell": "Goodbye, %s!",
		"profile": {
			"title": "User Profile",
			"settings": "Settings",
			"privacy": "Privacy Settings",
			"avatar": "Profile Picture",
			"bio": "Biography",
			"email": "Email Address"
		},
		"messages": {
			"inbox": "Inbox",
			"sent": "Sent Messages",
			"count": "You have %d messages",
			"empty": "No messages"
		}
	},
	"errors": {
		"validation": {
			"required": "This field is required",
			"email": "Invalid email address",
			"min_length": "Minimum length is %d characters",
			"max_length": "Maximum length is %d characters",
			"pattern": "Invalid format",
			"unique": "This value already exists"
		},
		"network": {
			"timeout": "Request timed out",
			"offline": "You are offline",
			"server": "Server error",
			"forbidden": "Access denied"
		},
		"auth": {
			"invalid_credentials": "Invalid username or password",
			"session_expired": "Your session has expired",
			"locked": "Account locked"
		}
	},
	"items": {
		"one": "# item",
		"other": "# items"
	},
	"notifications": "{count, plural, one {# notification} other {# notifications}}",
	"common": {
		"yes": "Yes",
		"no": "No",
		"save": "Save",
		"cancel": "Cancel",
		"delete": "Delete",
		"edit": "Edit",
		"create": "Create",
		"update": "Update",
		"ok": "OK",
		"close": "Close",
		"back": "Back",
		"next": "Next",
		"loading": "Loading..."
	}
}`)

// Baseline: ~2000-8000 ns/op, 30-80 allocs/op
func BenchmarkJSONParser_LargePayload(b *testing.B) {
	parser := NewJSONParser()
	_, _ = parser.Parse(benchLargeJSON)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = parser.Parse(benchLargeJSON) //nolint:errcheck // benchmark
	}
}

// Baseline: ~200-600 ns/op, 5-10 allocs/op
func BenchmarkJSONParser_Parallel(b *testing.B) {
	parser := NewJSONParser()
	data := []byte(`{"greeting":"Hello","user":{"profile":{"title":"User Profile"}}}`)
	_, _ = parser.Parse(data)
	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = parser.Parse(data) //nolint:errcheck // benchmark
		}
	})
}
