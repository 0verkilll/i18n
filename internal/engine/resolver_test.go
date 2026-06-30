// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

package engine

import (
	"errors"
	"strings"
	"testing"

	"github.com/0verkilll/i18n/internal/core"
)

func TestDefaultKeyResolver_Resolve(t *testing.T) {
	translations := map[string]interface{}{
		"greeting": "Hello",
		"farewell": "Goodbye",
		"count":    42,
		"enabled":  true,
		"error": map[string]interface{}{
			"validation": map[string]interface{}{
				"required": "This field is required",
				"email":    "Invalid email address",
				"password": map[string]interface{}{
					"min_length": "Password too short",
					"strength":   "Password too weak",
				},
			},
			"network": map[string]interface{}{
				"timeout": "Request timed out",
				"offline": "You are offline",
			},
		},
		"button": map[string]interface{}{
			"submit": "Submit",
			"cancel": "Cancel",
		},
		"items": []interface{}{"one", "two", "three"},
	}

	tests := []struct {
		errType error
		name    string
		key     string
		wantVal string
		wantErr bool
	}{
		{name: "simple key - greeting", key: "greeting", wantVal: "Hello"},
		{name: "simple key - farewell", key: "farewell", wantVal: "Goodbye"},
		{name: "nested 2 levels", key: "error.validation.required", wantVal: "This field is required"},
		{name: "nested 2 levels - timeout", key: "error.network.timeout", wantVal: "Request timed out"},
		{name: "nested 3 levels - min_length", key: "error.validation.password.min_length", wantVal: "Password too short"},
		{name: "nested 3 levels - strength", key: "error.validation.password.strength", wantVal: "Password too weak"},
		{name: "nested 1 level - submit", key: "button.submit", wantVal: "Submit"},
		{name: "number value", key: "count", wantVal: "42"},
		{name: "boolean value", key: "enabled", wantVal: "true"},
		{name: "missing top-level key", key: "nonexistent", wantErr: true, errType: &core.ErrKeyNotFound{}},
		{name: "missing nested key", key: "error.validation.nonexistent", wantErr: true, errType: &core.ErrKeyNotFound{}},
		{name: "missing intermediate key", key: "error.nonexistent.timeout", wantErr: true, errType: &core.ErrKeyNotFound{}},
		{name: "empty key", key: "", wantErr: true, errType: &core.ErrInvalidKey{}},
		{name: "key starting with dot", key: ".error", wantErr: true, errType: &core.ErrInvalidKey{}},
		{name: "key ending with dot", key: "error.", wantErr: true, errType: &core.ErrInvalidKey{}},
		{name: "consecutive dots", key: "error..validation", wantErr: true, errType: &core.ErrInvalidKey{}},
		{name: "array value - items", key: "items", wantErr: true, errType: &core.ErrInvalidKey{}},
		{name: "object value - error.validation", key: "error.validation", wantErr: true, errType: &core.ErrInvalidKey{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := NewDefaultKeyResolver()
			result, err := resolver.Resolve(translations, tt.key)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Resolve() expected error, got nil")
				}
				if tt.errType != nil {
					var expectedKeyNotFound *core.ErrKeyNotFound
					var expectedInvalidKey *core.ErrInvalidKey
					if errors.As(tt.errType, &expectedKeyNotFound) {
						var keyNotFoundErr *core.ErrKeyNotFound
						if !errors.As(err, &keyNotFoundErr) {
							t.Errorf("Resolve() error should be core.ErrKeyNotFound, got: %v", err)
						}
					} else if errors.As(tt.errType, &expectedInvalidKey) {
						var invalidKeyErr *core.ErrInvalidKey
						if !errors.As(err, &invalidKeyErr) {
							t.Errorf("Resolve() error should be core.ErrInvalidKey, got: %v", err)
						}
					}
				}
				return
			}

			if err != nil {
				t.Errorf("Resolve() unexpected error: %v", err)
				return
			}
			if result != tt.wantVal {
				t.Errorf("Resolve() = %q, want %q", result, tt.wantVal)
			}
		})
	}
}

func TestDefaultKeyResolver_ResolveDepthLimit(t *testing.T) {
	deep := make(map[string]interface{})
	current := deep
	keys := []string{"level0"}

	for i := 1; i <= 11; i++ {
		levelKey := "level" + string(rune('0'+i))
		keys = append(keys, levelKey)
		if i == 11 {
			current[levelKey] = "deep value"
		} else {
			nextLevel := make(map[string]interface{})
			current[levelKey] = nextLevel
			current = nextLevel
		}
	}

	resolver := NewDefaultKeyResolver()
	deepKey := "level0.level1.level2.level3.level4.level5.level6.level7.level8.level9.level10"
	_, err := resolver.Resolve(deep, deepKey)

	if err == nil {
		t.Error("Resolve() should reject keys exceeding max depth")
	}

	var invalidKeyErr *core.ErrInvalidKey
	if !errors.As(err, &invalidKeyErr) {
		t.Errorf("Resolve() should return core.ErrInvalidKey for deep keys, got: %v", err)
	}
}

func TestDefaultKeyResolver_ResolveLengthLimit(t *testing.T) {
	translations := map[string]interface{}{"key": "value"}
	resolver := NewDefaultKeyResolver()

	longKey := strings.Repeat("a", 257)
	_, err := resolver.Resolve(translations, longKey)

	if err == nil {
		t.Error("Resolve() should reject keys exceeding max length")
	}

	var invalidKeyErr *core.ErrInvalidKey
	if !errors.As(err, &invalidKeyErr) {
		t.Errorf("Resolve() should return core.ErrInvalidKey for long keys, got: %v", err)
	}
}

func TestDefaultKeyResolver_ResolveEmptyTranslations(t *testing.T) {
	resolver := NewDefaultKeyResolver()

	_, err := resolver.Resolve(nil, "key")
	if err == nil {
		t.Error("Resolve() should fail with nil translations")
	}

	empty := make(map[string]interface{})
	_, err = resolver.Resolve(empty, "key")
	if err == nil {
		t.Error("Resolve() should fail with missing key in empty translations")
	}

	var keyNotFoundErr *core.ErrKeyNotFound
	if !errors.As(err, &keyNotFoundErr) {
		t.Errorf("Resolve() should return core.ErrKeyNotFound, got: %v", err)
	}
}

func TestDefaultKeyResolver_ResolveNullValue(t *testing.T) {
	translations := map[string]interface{}{"null_key": nil}
	resolver := NewDefaultKeyResolver()
	result, err := resolver.Resolve(translations, "null_key")
	if err != nil {
		t.Errorf("Resolve() unexpected error for null value: %v", err)
	}
	if result != "" {
		t.Errorf("Resolve() null value should return empty string, got: %q", result)
	}
}

func TestDefaultKeyResolver_ResolveWithWhitespace(t *testing.T) {
	translations := map[string]interface{}{
		"key with spaces": "value",
		"tab\tkey":        "value",
	}
	resolver := NewDefaultKeyResolver()

	tests := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{name: "key with spaces", key: "key with spaces", wantErr: true},
		{name: "key with tab", key: "tab\tkey", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := resolver.Resolve(translations, tt.key)
			if tt.wantErr && err == nil {
				t.Error("Resolve() expected error for invalid key")
			}
		})
	}
}

func TestDefaultKeyResolver_ResolveNumericTypes(t *testing.T) {
	translations := map[string]interface{}{
		"int_val":     42,
		"int64_val":   int64(1234567890),
		"float_val":   3.14,
		"float_int":   100.0,
		"large_float": 1.23e10,
	}

	resolver := NewDefaultKeyResolver()

	tests := []struct {
		name    string
		key     string
		wantVal string
	}{
		{name: "int value", key: "int_val", wantVal: "42"},
		{name: "int64 value", key: "int64_val", wantVal: "1234567890"},
		{name: "float value", key: "float_val", wantVal: "3.14"},
		{name: "float that equals integer", key: "float_int", wantVal: "100"},
		{name: "large float", key: "large_float", wantVal: "12300000000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := resolver.Resolve(translations, tt.key)
			if err != nil {
				t.Errorf("Resolve() unexpected error: %v", err)
				return
			}
			if result != tt.wantVal {
				t.Errorf("Resolve() = %q, want %q", result, tt.wantVal)
			}
		})
	}
}

func TestDefaultKeyResolver_ResolveNestedToNonMap(t *testing.T) {
	translations := map[string]interface{}{
		"error": map[string]interface{}{
			"message": "Error occurred",
		},
	}

	resolver := NewDefaultKeyResolver()
	_, err := resolver.Resolve(translations, "error.message.nested")
	if err == nil {
		t.Error("Resolve() should fail when trying to traverse through a non-map value")
	}

	var keyNotFoundErr *core.ErrKeyNotFound
	if !errors.As(err, &keyNotFoundErr) {
		t.Errorf("Resolve() should return core.ErrKeyNotFound, got: %v", err)
	}
}

// Merged from coverage_test.go: TestConvertToString_AllBranches
func TestConvertToString_AllBranches(t *testing.T) {
	resolver := NewDefaultKeyResolver()

	tests := []struct {
		name         string
		translations map[string]interface{}
		key          string
		wantResult   string
		wantErr      bool
	}{
		{
			name:         "string value",
			translations: map[string]interface{}{"msg": "hello"},
			key:          "msg",
			wantResult:   "hello",
		},
		{
			name:         "int value",
			translations: map[string]interface{}{"count": 42},
			key:          "count",
			wantResult:   "42",
		},
		{
			name:         "int64 value",
			translations: map[string]interface{}{"bignum": int64(9223372036854775807)},
			key:          "bignum",
			wantResult:   "9223372036854775807",
		},
		{
			name:         "float64 value integer",
			translations: map[string]interface{}{"num": float64(42)},
			key:          "num",
			wantResult:   "42",
		},
		{
			name:         "float64 value decimal",
			translations: map[string]interface{}{"pi": 3.14159},
			key:          "pi",
			wantResult:   "3.14159",
		},
		{
			name:         "bool value true",
			translations: map[string]interface{}{"active": true},
			key:          "active",
			wantResult:   "true",
		},
		{
			name:         "bool value false",
			translations: map[string]interface{}{"disabled": false},
			key:          "disabled",
			wantResult:   "false",
		},
		{
			name:         "map value - should error",
			translations: map[string]interface{}{"obj": map[string]interface{}{"key": "value"}},
			key:          "obj",
			wantErr:      true,
		},
		{
			name:         "slice value - should error",
			translations: map[string]interface{}{"arr": []string{"a", "b", "c"}},
			key:          "arr",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := resolver.Resolve(tt.translations, tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("Resolve() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && result != tt.wantResult {
				t.Errorf("Resolve() = %q, want %q", result, tt.wantResult)
			}
		})
	}
}

// TestFormatFloat64 tests the formatFloat64 helper.
func TestFormatFloat64(t *testing.T) {
	tests := []struct {
		name  string
		input float64
		want  string
	}{
		{"integer float", 42.0, "42"},
		{"decimal float", 3.14, "3.14"},
		{"zero", 0.0, "0"},
		{"negative integer", -5.0, "-5"},
		{"large number", 1e15, "1000000000000000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatFloat64(tt.input)
			if got != tt.want {
				t.Errorf("formatFloat64(%v) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// Distributed from fuzz_test.go: FuzzKeyResolver
func FuzzKeyResolver(f *testing.F) {
	f.Add("greeting")
	f.Add("error.validation.required")
	f.Add("button.submit")
	f.Add("a.b.c.d.e")
	f.Add("simple")
	f.Add("nested.key")
	f.Add("..invalid")
	f.Add(".start")
	f.Add("end.")
	f.Add("")
	f.Add(strings.Repeat("a", 300))
	f.Add(strings.Repeat("a.", 50))

	translations := map[string]interface{}{
		"greeting": "Hello",
		"error": map[string]interface{}{
			"validation": map[string]interface{}{
				"required": "Required field",
			},
		},
		"button": map[string]interface{}{
			"submit": "Submit",
		},
	}

	resolver := NewDefaultKeyResolver()

	f.Fuzz(func(t *testing.T, key string) {
		_, err := resolver.Resolve(translations, key)
		if err != nil {
			var invalidKeyErr *core.ErrInvalidKey
			var keyNotFoundErr *core.ErrKeyNotFound
			if !errors.As(err, &invalidKeyErr) && !errors.As(err, &keyNotFoundErr) {
				t.Logf("Resolve() returned unexpected error type: %v", err)
			}
		}
	})
}

// Distributed from benchmark_test.go: BenchmarkKeyResolver
func BenchmarkKeyResolver(b *testing.B) {
	translations := map[string]interface{}{
		"user": map[string]interface{}{
			"profile": map[string]interface{}{
				"title": "User Profile",
			},
		},
	}

	resolver := NewDefaultKeyResolver()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = resolver.Resolve(translations, "user.profile.title") //nolint:errcheck // benchmark
	}
}

// TestTraverseNested_NonMapIntermediate hits the !ok branch in traverseNested
// where an intermediate key resolves to a non-map value.
func TestTraverseNested_NonMapIntermediate(t *testing.T) {
	translations := map[string]interface{}{
		"greeting": "Hello",
	}

	resolver := NewDefaultKeyResolver()
	_, err := resolver.Resolve(translations, "greeting.sub.key")
	if err == nil {
		t.Error("Resolve() should fail when intermediate value is not a map")
	}

	var keyNotFoundErr *core.ErrKeyNotFound
	if !errors.As(err, &keyNotFoundErr) {
		t.Errorf("Resolve() should return core.ErrKeyNotFound, got: %v", err)
	}
}

// =============================================================================
// Spec 018: Resolver Benchmarks (Task Group 2)
// =============================================================================

// benchShallowTranslations is a package-level map for shallow key benchmarking.
var benchShallowTranslations = map[string]interface{}{
	"greeting": "Hello",
	"farewell": "Goodbye",
	"title":    "My Application",
}

// benchDeepTranslations is a package-level map for 5-level deep key benchmarking.
var benchDeepTranslations = map[string]interface{}{
	"app": map[string]interface{}{
		"module": map[string]interface{}{
			"section": map[string]interface{}{
				"error": map[string]interface{}{
					"message": "Something went wrong",
				},
			},
		},
	},
}

// Baseline: ~30-80 ns/op, 0 allocs/op
func BenchmarkKeyResolver_ShallowKey(b *testing.B) {
	resolver := NewDefaultKeyResolver()
	_, _ = resolver.Resolve(benchShallowTranslations, "greeting")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = resolver.Resolve(benchShallowTranslations, "greeting") //nolint:errcheck // benchmark
	}
}

// Baseline: ~100-300 ns/op, 1-2 allocs/op
func BenchmarkKeyResolver_DeepKey(b *testing.B) {
	resolver := NewDefaultKeyResolver()
	_, _ = resolver.Resolve(benchDeepTranslations, "app.module.section.error.message")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = resolver.Resolve(benchDeepTranslations, "app.module.section.error.message") //nolint:errcheck // benchmark
	}
}

// Baseline: ~30-100 ns/op, 0-1 allocs/op
func BenchmarkKeyResolver_Parallel(b *testing.B) {
	resolver := NewDefaultKeyResolver()
	translations := map[string]interface{}{
		"user": map[string]interface{}{
			"profile": map[string]interface{}{
				"title": "User Profile",
			},
		},
	}
	_, _ = resolver.Resolve(translations, "user.profile.title")
	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = resolver.Resolve(translations, "user.profile.title") //nolint:errcheck // benchmark
		}
	})
}

// =============================================================================
// Fallback chainer tests (merged from fallback_test.go)
// =============================================================================

func TestDefaultFallbackChainer_GetChain(t *testing.T) {
	tests := []struct {
		name     string
		locale   string
		expected []string
	}{
		// Regional variants fall back to base locale then English
		{
			name:     "es-MX falls back to es-ES then en-US",
			locale:   "es-MX",
			expected: []string{"es-MX", "es-ES", "en-US"},
		},
		{
			name:     "es-AR falls back to es-ES then en-US",
			locale:   "es-AR",
			expected: []string{"es-AR", "es-ES", "en-US"},
		},
		{
			name:     "pt-BR falls back to pt-PT then en-US",
			locale:   "pt-BR",
			expected: []string{"pt-BR", "pt-PT", "en-US"},
		},
		{
			name:     "en-GB falls back to en-US",
			locale:   "en-GB",
			expected: []string{"en-GB", "en-US"},
		},
		{
			name:     "en-AU falls back to en-US",
			locale:   "en-AU",
			expected: []string{"en-AU", "en-US"},
		},
		{
			name:     "fr-CA falls back to fr-FR then en-US",
			locale:   "fr-CA",
			expected: []string{"fr-CA", "fr-FR", "en-US"},
		},
		{
			name:     "zh-TW falls back to zh-CN then en-US",
			locale:   "zh-TW",
			expected: []string{"zh-TW", "zh-CN", "en-US"},
		},
		{
			name:     "de-AT falls back to de-DE then en-US",
			locale:   "de-AT",
			expected: []string{"de-AT", "de-DE", "en-US"},
		},

		// Base locales fall back directly to English
		{
			name:     "es-ES falls back to en-US",
			locale:   "es-ES",
			expected: []string{"es-ES", "en-US"},
		},
		{
			name:     "pt-PT falls back to en-US",
			locale:   "pt-PT",
			expected: []string{"pt-PT", "en-US"},
		},
		{
			name:     "fr-FR falls back to en-US",
			locale:   "fr-FR",
			expected: []string{"fr-FR", "en-US"},
		},
		{
			name:     "de-DE falls back to en-US",
			locale:   "de-DE",
			expected: []string{"de-DE", "en-US"},
		},
		{
			name:     "ja-JP falls back to en-US",
			locale:   "ja-JP",
			expected: []string{"ja-JP", "en-US"},
		},

		// English locales
		{
			name:     "en-US has no fallback (is the default)",
			locale:   "en-US",
			expected: []string{"en-US"},
		},

		// Unsupported languages still get English fallback
		{
			name:     "unsupported language xx-YY falls back to en-US",
			locale:   "xx-YY",
			expected: []string{"xx-YY", "en-US"},
		},
		{
			name:     "rare locale eo-001 falls back to en-US",
			locale:   "eo-001",
			expected: []string{"eo-001", "en-US"},
		},

		// Language-only codes (without region)
		{
			name:     "language-only es gets en-US fallback",
			locale:   "es",
			expected: []string{"es", "en-US"},
		},
		{
			name:     "language-only fr gets en-US fallback",
			locale:   "fr",
			expected: []string{"fr", "en-US"},
		},
		{
			name:     "language-only en returns just en",
			locale:   "en",
			expected: []string{"en"},
		},

		// Three-letter language codes
		{
			name:     "haw-US falls back to en-US",
			locale:   "haw-US",
			expected: []string{"haw-US", "en-US"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chainer := NewDefaultFallbackChainer()
			got := chainer.GetChain(tt.locale)
			if !slicesEqual(got, tt.expected) {
				t.Errorf("GetChain(%q) = %v, want %v", tt.locale, got, tt.expected)
			}
		})
	}
}

func TestDefaultFallbackChainer_NoDuplicates(t *testing.T) {
	chainer := NewDefaultFallbackChainer()

	testCases := []string{
		"en-US",
		"es-ES",
		"pt-BR",
		"en-GB",
		"zh-CN",
	}

	for _, locale := range testCases {
		chain := chainer.GetChain(locale)
		seen := make(map[string]bool)
		for _, item := range chain {
			if seen[item] {
				t.Errorf("GetChain(%q) contains duplicate: %q", locale, item)
			}
			seen[item] = true
		}
	}
}

func TestDefaultFallbackChainer_AlwaysEndsWithEnglish(t *testing.T) {
	chainer := NewDefaultFallbackChainer()

	testCases := []string{
		"es-MX", "pt-BR", "fr-CA", "de-AT", "zh-TW",
		"ja-JP", "ko-KR", "ar-SA", "ru-RU",
	}

	for _, locale := range testCases {
		chain := chainer.GetChain(locale)
		if len(chain) == 0 {
			t.Errorf("GetChain(%q) returned empty chain", locale)
			continue
		}
		last := chain[len(chain)-1]
		if last != "en-US" && last != "en" {
			t.Errorf("GetChain(%q) = %v, expected to end with en-US or en, got %q", locale, chain, last)
		}
	}
}

func TestDefaultFallbackChainer_ChainLength(t *testing.T) {
	chainer := NewDefaultFallbackChainer()

	tests := []struct {
		name      string
		locale    string
		maxLength int
	}{
		{name: "en-US has length 1", locale: "en-US", maxLength: 1},
		{name: "en has length 1", locale: "en", maxLength: 1},
		{name: "es-ES has length 2", locale: "es-ES", maxLength: 2},
		{name: "es-MX has length 3", locale: "es-MX", maxLength: 3},
		{name: "pt-BR has length 3", locale: "pt-BR", maxLength: 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chain := chainer.GetChain(tt.locale)
			if len(chain) > tt.maxLength {
				t.Errorf("GetChain(%q) length = %d, expected max %d, chain: %v", tt.locale, len(chain), tt.maxLength, chain)
			}
		})
	}
}

func TestFallback_ContainsString_AllBranches(t *testing.T) {
	tests := []struct {
		name     string
		str      string
		slice    []string
		expected bool
	}{
		{
			name:     "found in slice",
			slice:    []string{"en-US", "es-ES", "fr-FR"},
			str:      "es-ES",
			expected: true,
		},
		{
			name:     "not found in slice",
			slice:    []string{"en-US", "es-ES", "fr-FR"},
			str:      "de-DE",
			expected: false,
		},
		{
			name:     "empty slice",
			slice:    []string{},
			str:      "en-US",
			expected: false,
		},
		{
			name:     "found at end",
			slice:    []string{"en-US", "es-ES", "fr-FR"},
			str:      "fr-FR",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsString(tt.slice, tt.str)
			if result != tt.expected {
				t.Errorf("containsString() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestFallbackChain_EdgeCases(t *testing.T) {
	chainer := NewDefaultFallbackChainer()

	tests := []struct {
		name     string
		locale   string
		contains []string
		minLen   int
	}{
		{
			name:     "en-US no fallback",
			locale:   "en-US",
			minLen:   1,
			contains: []string{"en-US"},
		},
		{
			name:     "unknown locale falls back to en-US",
			locale:   "xyz-ABC",
			minLen:   2,
			contains: []string{"xyz-ABC", "en-US"},
		},
		{
			name:     "three letter language",
			locale:   "haw-US",
			minLen:   2,
			contains: []string{"haw-US", "en-US"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chain := chainer.GetChain(tt.locale)
			if len(chain) < tt.minLen {
				t.Errorf("GetChain(%q) length = %d, want >= %d", tt.locale, len(chain), tt.minLen)
			}
			for _, expected := range tt.contains {
				found := false
				for _, locale := range chain {
					if locale == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("GetChain(%q) = %v, should contain %q", tt.locale, chain, expected)
				}
			}
		})
	}
}

func TestGetBaseLocale(t *testing.T) {
	tests := []struct {
		lang     string
		expected string
	}{
		{"es", "es-ES"},
		{"pt", "pt-PT"},
		{"zh", "zh-CN"},
		{"fr", "fr-FR"},
		{"de", "de-DE"},
		{"ar", "ar-SA"},
		{"it", "it-IT"},
		{"nl", "nl-NL"},
		{"pl", "pl-PL"},
		{"tr", "tr-TR"},
		{"sv", "sv-SE"},
		{"da", "da-DK"},
		{"fi", "fi-FI"},
		{"no", "no-NO"},
		{"cs", "cs-CZ"},
		{"el", "el-GR"},
		{"he", "he-IL"},
		{"hi", "hi-IN"},
		{"th", "th-TH"},
		{"vi", "vi-VN"},
		{"id", "id-ID"},
		{"ms", "ms-MY"},
		{"bn", "bn-BD"},
		{"uk", "uk-UA"},
		{"ro", "ro-RO"},
		{"hu", "hu-HU"},
		{"ja", "ja-JP"},
		{"ko", "ko-KR"},
		{"ru", "ru-RU"},
		{"unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.lang, func(t *testing.T) {
			got := getBaseLocale(tt.lang)
			if got != tt.expected {
				t.Errorf("getBaseLocale(%q) = %q, want %q", tt.lang, got, tt.expected)
			}
		})
	}
}

func FuzzGetChain(f *testing.F) {
	f.Add("en-US")
	f.Add("es-MX")
	f.Add("pt-BR")
	f.Add("en-GB")
	f.Add("en")
	f.Add("xx-YY")
	f.Add("")
	f.Add("eo-001")
	f.Add("haw-US")
	f.Add("\x00\x01")
	f.Add(strings.Repeat("a", 50))

	chainer := NewDefaultFallbackChainer()

	f.Fuzz(func(t *testing.T, locale string) {
		chain := chainer.GetChain(locale)

		if len(chain) == 0 {
			t.Errorf("GetChain returned empty chain for locale %q", locale)
		}

		seen := make(map[string]bool)
		for _, item := range chain {
			if seen[item] {
				t.Errorf("GetChain produced chain with duplicates: %v", chain)
				break
			}
			seen[item] = true
		}

		if !strings.HasPrefix(strings.ToLower(locale), "en") && len(chain) > 1 {
			last := chain[len(chain)-1]
			if !strings.HasPrefix(strings.ToLower(last), "en") && last != locale {
				t.Logf("GetChain for non-English locale should typically end with English: %v", chain)
			}
		}
	})
}

// slicesEqual compares two string slices for equality without using reflect.
func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func BenchmarkGetChain(b *testing.B) {
	chainer := NewDefaultFallbackChainer()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = chainer.GetChain("es-MX")
	}
}

// Spec 018: Fallback Chain Benchmarks

func BenchmarkGetChain_EnUS(b *testing.B) {
	chainer := NewDefaultFallbackChainer()
	_ = chainer.GetChain("en-US")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = chainer.GetChain("en-US")
	}
}

func BenchmarkGetChain_DeepChain(b *testing.B) {
	chainer := NewDefaultFallbackChainer()
	_ = chainer.GetChain("es-MX")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = chainer.GetChain("es-MX")
	}
}

func BenchmarkGetChain_LanguageOnly(b *testing.B) {
	chainer := NewDefaultFallbackChainer()
	_ = chainer.GetChain("fr")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = chainer.GetChain("fr")
	}
}

func BenchmarkGetChain_Parallel(b *testing.B) {
	chainer := NewDefaultFallbackChainer()
	_ = chainer.GetChain("es-MX")
	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = chainer.GetChain("es-MX")
		}
	})
}
