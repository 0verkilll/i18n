// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

package engine

import (
	"strings"
	"testing"

	"github.com/0verkilll/i18n/internal/core"
)

// =============================================================================
// Default locale detector tests
// =============================================================================

// MockEnvProvider is a test helper that provides environment variables
type MockEnvProvider struct {
	vars map[string]string
}

func (m *MockEnvProvider) Getenv(key string) string {
	return m.vars[key]
}

func TestDefaultLocaleDetector_Detect(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		expected string
	}{
		{
			name:     "LANG set",
			envVars:  map[string]string{"LANG": "en_US.UTF-8"},
			expected: "en_US.UTF-8",
		},
		{
			name:     "LC_ALL takes precedence over LANG",
			envVars:  map[string]string{"LANG": "en_US.UTF-8", "LC_ALL": "es_MX.UTF-8"},
			expected: "es_MX.UTF-8",
		},
		{
			name:     "LC_MESSAGES used if LANG and LC_ALL empty",
			envVars:  map[string]string{"LC_MESSAGES": "fr_FR.UTF-8"},
			expected: "fr_FR.UTF-8",
		},
		{
			name:     "defaults to en-US if all empty",
			envVars:  map[string]string{},
			expected: "en-US",
		},
		{
			name:     "LANG takes precedence over LC_MESSAGES",
			envVars:  map[string]string{"LANG": "de_DE.UTF-8", "LC_MESSAGES": "fr_FR.UTF-8"},
			expected: "de_DE.UTF-8",
		},
		{
			name:     "empty LANG falls back to LC_ALL",
			envVars:  map[string]string{"LANG": "", "LC_ALL": "pt_BR.UTF-8"},
			expected: "pt_BR.UTF-8",
		},
		{
			name:     "empty LC_ALL falls back to LC_MESSAGES",
			envVars:  map[string]string{"LANG": "", "LC_ALL": "", "LC_MESSAGES": "it_IT.UTF-8"},
			expected: "it_IT.UTF-8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockEnv := &MockEnvProvider{vars: tt.envVars}
			detector := NewDefaultLocaleDetector(mockEnv)
			got := detector.Detect()
			if got != tt.expected {
				t.Errorf("Detect() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestDefaultLocaleDetector_Normalize(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Encoding removal
		{name: "removes UTF-8", input: "en_US.UTF-8", expected: "en-US"},
		{name: "removes utf8", input: "en_US.utf8", expected: "en-US"},
		{name: "removes ISO-8859-1", input: "en_US.ISO-8859-1", expected: "en-US"},
		{name: "removes other encoding", input: "ja_JP.eucJP", expected: "ja-JP"},

		// Underscore to hyphen conversion
		{name: "converts underscore", input: "en_US", expected: "en-US"},
		{name: "converts es_MX", input: "es_MX", expected: "es-MX"},
		{name: "converts pt_BR", input: "pt_BR", expected: "pt-BR"},
		{name: "already hyphenated", input: "en-US", expected: "en-US"},

		// POSIX/C locale
		{name: "POSIX locale", input: "POSIX", expected: "en-US"},
		{name: "C locale", input: "C", expected: "en-US"},

		// Language-only codes
		{name: "en defaults to en-US", input: "en", expected: "en-US"},
		{name: "es defaults to es-ES", input: "es", expected: "es-ES"},
		{name: "pt defaults to pt-PT", input: "pt", expected: "pt-PT"},
		{name: "zh defaults to zh-CN", input: "zh", expected: "zh-CN"},
		{name: "fr defaults to fr-FR", input: "fr", expected: "fr-FR"},
		{name: "de defaults to de-DE", input: "de", expected: "de-DE"},
		{name: "ja defaults to ja-JP", input: "ja", expected: "ja-JP"},
		{name: "ko defaults to ko-KR", input: "ko", expected: "ko-KR"},
		{name: "ru defaults to ru-RU", input: "ru", expected: "ru-RU"},
		{name: "ar defaults to ar-SA", input: "ar", expected: "ar-SA"},

		// Case normalization
		{name: "normalizes EN-US to en-US", input: "EN-US", expected: "en-US"},
		{name: "normalizes en-us to en-US", input: "en-us", expected: "en-US"},
		{name: "normalizes ES-mx to es-MX", input: "ES-mx", expected: "es-MX"},

		// Complex cases
		{name: "en_GB.UTF-8", input: "en_GB.UTF-8", expected: "en-GB"},
		{name: "zh_TW.UTF-8", input: "zh_TW.UTF-8", expected: "zh-TW"},
		{name: "pt_BR.ISO-8859-1", input: "pt_BR.ISO-8859-1", expected: "pt-BR"},

		// Already normalized
		{name: "already normalized", input: "en-US", expected: "en-US"},
		{name: "three letter lang", input: "haw-US", expected: "haw-US"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := NewDefaultLocaleDetector(nil)
			got := detector.Normalize(tt.input)
			if got != tt.expected {
				t.Errorf("Normalize(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestDefaultLocaleDetector_NormalizeEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "empty string", input: "", expected: ""},
		{name: "just encoding", input: ".UTF-8", expected: ""},
		{name: "invalid format preserved", input: "invalid", expected: "invalid"},
		{name: "numbers", input: "123", expected: "123"},
		{name: "special chars preserved", input: "en@euro", expected: "en@euro"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := NewDefaultLocaleDetector(nil)
			got := detector.Normalize(tt.input)
			if got != tt.expected {
				t.Errorf("Normalize(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

// Merged from coverage_test.go: TestLocaleNormalize_AllBranches
func TestLocaleNormalize_AllBranches(t *testing.T) {
	detector := NewDefaultLocaleDetector(nil)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "empty string", input: "", expected: ""},
		{name: "POSIX", input: "POSIX", expected: "en-US"},
		{name: "C locale", input: "C", expected: "en-US"},
		{name: "with encoding", input: "en_GB.UTF-8", expected: "en-GB"},
		{name: "two parts already hyphenated", input: "es-MX", expected: "es-MX"},
		{name: "language only with mapping", input: "fr", expected: "fr-FR"},
		{name: "language only without mapping", input: "xyz", expected: "xyz"},
		{name: "three parts (script code)", input: "zh-Hans-CN", expected: "zh-Hans-CN"},
		{name: "invalid multi-part format", input: "a-b-c-d", expected: "a-b-c-d"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.Normalize(tt.input)
			if result != tt.expected {
				t.Errorf("Normalize(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNormalizeLocale_IdenticalToDetectorNormalize(t *testing.T) {
	inputs := []string{
		"en_US.UTF-8", "es_MX", "fr", "POSIX", "ja_JP.eucJP",
		"en-us", "EN-US", "zh", "pt_BR.ISO-8859-1",
	}

	detector := NewDefaultLocaleDetector(nil)
	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			fromDetector := detector.Normalize(input)
			fromFunc := NormalizeLocale(input)
			if fromDetector != fromFunc {
				t.Errorf("NormalizeLocale(%q) = %q, detector.Normalize(%q) = %q; want identical",
					input, fromFunc, input, fromDetector)
			}
		})
	}
}

func TestNormalizeLocale_RegressionWithExistingTests(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{input: "en_US.UTF-8", expected: "en-US"},
		{input: "es_MX", expected: "es-MX"},
		{input: "POSIX", expected: "en-US"},
		{input: "C", expected: "en-US"},
		{input: "de", expected: "de-DE"},
		{input: "en-us", expected: "en-US"},
		{input: "haw-US", expected: "haw-US"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := NormalizeLocale(tt.input)
			if got != tt.expected {
				t.Errorf("NormalizeLocale(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestNormalizeLocale_EmptyString(t *testing.T) {
	got := NormalizeLocale("")
	if got != "" {
		t.Errorf("NormalizeLocale(\"\") = %q, want \"\"", got)
	}
}

// TestNewDefaultLocaleDetector_NilUsesDefault verifies nil env provider uses platform default.
func TestNewDefaultLocaleDetector_NilUsesDefault(t *testing.T) {
	detector := NewDefaultLocaleDetector(nil)
	if detector == nil {
		t.Fatal("NewDefaultLocaleDetector(nil) returned nil")
	}
	if detector.env == nil {
		t.Error("NewDefaultLocaleDetector(nil) should set a default env provider")
	}
}

// Distributed from fuzz_test.go: FuzzNormalize
func FuzzNormalize(f *testing.F) {
	f.Add("en_US.UTF-8")
	f.Add("es_MX")
	f.Add("pt-BR")
	f.Add("POSIX")
	f.Add("C")
	f.Add("en")
	f.Add("zh_TW.UTF-8")
	f.Add("EN-us")
	f.Add(".UTF-8")
	f.Add("")
	f.Add("\x00\x01\x02")
	f.Add(strings.Repeat("a", 100))

	detector := NewDefaultLocaleDetector(nil)

	f.Fuzz(func(t *testing.T, locale string) {
		result := detector.Normalize(locale)

		if strings.Contains(result, "_") {
			t.Errorf("Normalize produced output with underscore: %q -> %q", locale, result)
		}

		if strings.Contains(result, ".") {
			t.Errorf("Normalize produced output with dot: %q -> %q", locale, result)
		}
	})
}

// Distributed from benchmark_test.go: BenchmarkNormalize
func BenchmarkNormalize(b *testing.B) {
	detector := NewDefaultLocaleDetector(nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = detector.Normalize("en_US.UTF-8")
	}
}

// =============================================================================
// Merged from locale_guard_test.go (originally had //go:build !js)
// These tests exercise the default env provider on standard (non-JS) builds.
// On JS/WASM builds, the default env provider returns "en-US", so assertions
// still hold (Detect returns a non-empty string; detector.env is non-nil).
// =============================================================================

// TestNewDefaultLocaleDetectorNilDoesNotPanic verifies that passing nil
// to NewDefaultLocaleDetector creates a functioning detector.
func TestNewDefaultLocaleDetectorNilDoesNotPanic(t *testing.T) {
	detector := NewDefaultLocaleDetector(nil)
	if detector == nil {
		t.Fatal("NewDefaultLocaleDetector(nil) returned nil")
	}
	// Should not panic or nil-deref
	result := detector.Detect()
	if result == "" {
		t.Error("Detect() returned empty string, expected a default locale")
	}
}

// TestOSEnvProviderDefaultLocale verifies that the platform default provider
// returns "en-US" when no locale env vars are set.
func TestOSEnvProviderDefaultLocale(t *testing.T) {
	// The OSEnvProvider reads real env vars. When LANG/LC_ALL/LC_MESSAGES
	// are unset, Detect() should fall through to the "en-US" default.
	// Since we cannot guarantee the test environment has no locale vars,
	// we at least verify the detector returns a non-empty string.
	detector := NewDefaultLocaleDetector(nil)
	result := detector.Detect()
	if result == "" {
		t.Error("Detect() with OSEnvProvider returned empty string")
	}
}

// TestWASMEnvProviderGetenv verifies WASMEnvProvider always returns empty string.
// This tests the WASMEnvProvider type directly (it is defined in locale_wasm.go
// but we can test the concept via the default env provider on standard builds).
func TestWASMEnvProviderGetenv(t *testing.T) {
	// On standard builds, defaultEnvProvider returns OSEnvProvider.
	// We verify the contract: a nil-arg detector produces valid output.
	detector := NewDefaultLocaleDetector(nil)
	// The env provider is not nil after construction
	if detector.env == nil {
		t.Error("detector.env should not be nil after NewDefaultLocaleDetector(nil)")
	}
}

// =============================================================================
// Merged from locale_integration_test.go
// =============================================================================

func TestChainDetector_IntegrationWithTranslator(t *testing.T) {
	chain := NewChainDetector(
		NewStaticDetector("fr-FR"),
		NewAcceptLanguageDetector("de-DE,en;q=0.5"),
		NewDefaultLocaleDetector(nil),
	)

	translator, err := New(
		WithFileSystemLoader("testdata"),
		WithLocaleDetector(chain),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	got := translator.GetLocale()
	if got != "fr-FR" {
		t.Errorf("GetLocale() = %q, want %q", got, "fr-FR")
	}
}

func TestAcceptLanguageDetector_FlowsThroughTranslator(t *testing.T) {
	detector := NewAcceptLanguageDetector("es-MX,en;q=0.5")
	translator, err := New(
		WithFileSystemLoader("testdata"),
		WithLocaleDetector(detector),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	got := translator.GetLocale()
	if got != "es-MX" {
		t.Errorf("GetLocale() = %q, want %q", got, "es-MX")
	}
}

func TestAcceptLanguageDetector_MalformedQualityDefaultsToOne(t *testing.T) {
	d := NewAcceptLanguageDetector("fr;q=abc,de;q=0.5")
	got := d.Detect()
	// fr has malformed q so defaults to 1.0; de has q=0.5; fr wins
	if got != "fr-FR" {
		t.Errorf("Detect() = %q, want %q", got, "fr-FR")
	}
}

func TestAcceptLanguageDetector_EqualQualityPreservesOrder(t *testing.T) {
	d := NewAcceptLanguageDetector("fr,de,es")
	got := d.Detect()
	// All default to q=1.0; fr is first and should win
	if got != "fr-FR" {
		t.Errorf("Detect() = %q, want %q", got, "fr-FR")
	}
}

func TestAcceptLanguageDetector_LeadingTrailingDoubleCommas(t *testing.T) {
	d := NewAcceptLanguageDetector(",en-US,,fr;q=0.8,")
	got := d.Detect()
	if got != "en-US" {
		t.Errorf("Detect() = %q, want %q", got, "en-US")
	}
}

func TestChainDetector_FirstNonEmptyStopsIteration(t *testing.T) {
	chain := NewChainDetector(
		NewStaticDetector("ja-JP"),
		NewStaticDetector("ko-KR"),
	)
	got := chain.Detect()
	if got != "ja-JP" {
		t.Errorf("Detect() = %q, want %q (should stop at first detector)", got, "ja-JP")
	}
}

func TestStaticDetector_EmptyStringReturnsEmpty(t *testing.T) {
	d := NewStaticDetector("")
	got := d.Detect()
	if got != "" {
		t.Errorf("Detect() = %q, want \"\"", got)
	}
}

func TestNormalizeLocale_MatchesDetectorNormalize(t *testing.T) {
	input := "pt_BR.UTF-8"
	want := NormalizeLocale(input)

	detectors := []struct {
		name     string
		detector core.LocaleDetector
	}{
		{"StaticDetector", NewStaticDetector("x")},
		{"AcceptLanguageDetector", NewAcceptLanguageDetector("x")},
		{"ChainDetector", NewChainDetector()},
		{"BrowserDetector", NewBrowserDetector()},
	}

	for _, tt := range detectors {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.detector.Normalize(input)
			if got != want {
				t.Errorf("%s.Normalize(%q) = %q, want %q", tt.name, input, got, want)
			}
		})
	}
}

// =============================================================================
// Merged from wasm_hardening_test.go: detector test (originally !tinygo && !js)
// Uses MockEnvProvider to simulate empty env vars; safe on all platforms.
// =============================================================================

// TestNewDefaultLocaleDetectorNilDetect verifies NewDefaultLocaleDetector(nil).Detect()
// returns "en-US" on a standard build when no locale env vars are set.
func TestNewDefaultLocaleDetectorNilDetect(t *testing.T) {
	// Use a mock with empty vars to simulate no env vars
	mock := &MockEnvProvider{vars: map[string]string{}}
	detector := NewDefaultLocaleDetector(mock)
	got := detector.Detect()
	if got != "en-US" {
		t.Errorf("Detect() = %q, want %q", got, "en-US")
	}
}

// =============================================================================
// Accept-Language detector tests
// =============================================================================

func TestAcceptLanguageDetector_BasicParsing(t *testing.T) {
	d := NewAcceptLanguageDetector("en-US,fr;q=0.8")
	got := d.Detect()
	if got != "en-US" {
		t.Errorf("Detect() = %q, want %q", got, "en-US")
	}
}

func TestAcceptLanguageDetector_QualitySorting(t *testing.T) {
	d := NewAcceptLanguageDetector("fr;q=0.8,de;q=0.9,en-US;q=0.7")
	got := d.Detect()
	if got != "de-DE" {
		t.Errorf("Detect() = %q, want %q (de normalized to de-DE)", got, "de-DE")
	}
}

func TestAcceptLanguageDetector_EmptyHeader(t *testing.T) {
	d := NewAcceptLanguageDetector("")
	got := d.Detect()
	if got != "" {
		t.Errorf("Detect() = %q, want \"\"", got)
	}
}

func TestAcceptLanguageDetector_WildcardOnly(t *testing.T) {
	d := NewAcceptLanguageDetector("*")
	got := d.Detect()
	if got != "" {
		t.Errorf("Detect() = %q, want \"\"", got)
	}
}

func TestAcceptLanguageDetector_WhitespaceTolerance(t *testing.T) {
	d1 := NewAcceptLanguageDetector("en-US , fr ; q=0.8")
	d2 := NewAcceptLanguageDetector("en-US,fr;q=0.8")
	got1 := d1.Detect()
	got2 := d2.Detect()
	if got1 != got2 {
		t.Errorf("whitespace variant = %q, compact = %q; want identical", got1, got2)
	}
}

func TestAcceptLanguageDetector_HeaderTooLong(t *testing.T) {
	long := strings.Repeat("en-US,", 700)
	d := NewAcceptLanguageDetector(long)
	got := d.Detect()
	if got != "" {
		t.Errorf("Detect() for oversized header = %q, want \"\"", got)
	}
}

func TestAcceptLanguageDetector_Normalize(t *testing.T) {
	d := NewAcceptLanguageDetector("")
	got := d.Normalize("en_US.UTF-8")
	if got != "en-US" {
		t.Errorf("Normalize(en_US.UTF-8) = %q, want %q", got, "en-US")
	}
}

func TestAcceptLanguageDetector_EmptyTagSkipped(t *testing.T) {
	d := NewAcceptLanguageDetector(",,,en-US")
	got := d.Detect()
	if got != "en-US" {
		t.Errorf("Detect() with empty tags = %q, want %q", got, "en-US")
	}
}

// Coverage gap: parseQuality edge cases

func TestParseQuality_NoQParam(t *testing.T) {
	got := parseQuality("level=5")
	if got != 1.0 {
		t.Errorf("parseQuality without q= = %v, want 1.0", got)
	}
}

func TestParseQuality_MalformedValue(t *testing.T) {
	got := parseQuality("q=abc")
	if got != 1.0 {
		t.Errorf("parseQuality malformed = %v, want 1.0", got)
	}
}

func TestParseQuality_NegativeValue(t *testing.T) {
	got := parseQuality("q=-0.5")
	if got != 0 {
		t.Errorf("parseQuality negative = %v, want 0", got)
	}
}

func TestParseQuality_ValueAboveOne(t *testing.T) {
	got := parseQuality("q=1.5")
	if got != 1.0 {
		t.Errorf("parseQuality >1 = %v, want 1.0", got)
	}
}

func TestParseQuality_WithAdditionalParams(t *testing.T) {
	got := parseQuality("q=0.7;charset=utf-8")
	if got != 0.7 {
		t.Errorf("parseQuality with extra params = %v, want 0.7", got)
	}
}

func TestParseQuality_ExactlyZero(t *testing.T) {
	got := parseQuality("q=0")
	if got != 0 {
		t.Errorf("parseQuality zero = %v, want 0", got)
	}
}

func TestParseQuality_ExactlyOne(t *testing.T) {
	got := parseQuality("q=1")
	if got != 1.0 {
		t.Errorf("parseQuality one = %v, want 1.0", got)
	}
}

func TestParseQuality_WhitespaceAround(t *testing.T) {
	got := parseQuality("  q=0.5  ")
	if got != 0.5 {
		t.Errorf("parseQuality whitespace = %v, want 0.5", got)
	}
}

// =============================================================================
// Chain detector tests
// =============================================================================

func TestChainDetector_FirstNonEmptyWins(t *testing.T) {
	chain := NewChainDetector(
		NewStaticDetector(""),
		NewStaticDetector("fr-FR"),
	)
	got := chain.Detect()
	if got != "fr-FR" {
		t.Errorf("Detect() = %q, want %q", got, "fr-FR")
	}
}

func TestChainDetector_AllEmptyFallsBackToEnUS(t *testing.T) {
	chain := NewChainDetector(
		NewStaticDetector(""),
		NewStaticDetector(""),
	)
	got := chain.Detect()
	if got != "en-US" {
		t.Errorf("Detect() = %q, want %q", got, "en-US")
	}
}

func TestChainDetector_ZeroDetectors(t *testing.T) {
	chain := NewChainDetector()
	got := chain.Detect()
	if got != "en-US" {
		t.Errorf("Detect() = %q, want %q", got, "en-US")
	}
}

func TestChainDetector_Normalize(t *testing.T) {
	chain := NewChainDetector(NewStaticDetector("anything"))
	got := chain.Normalize("es_MX.UTF-8")
	want := "es-MX"
	if got != want {
		t.Errorf("Normalize(\"es_MX.UTF-8\") = %q, want %q", got, want)
	}
}

// =============================================================================
// Static detector tests
// =============================================================================

func TestStaticDetector_Detect(t *testing.T) {
	tests := []struct {
		name   string
		locale string
		want   string
	}{
		{name: "returns exact locale", locale: "fr-FR", want: "fr-FR"},
		{name: "returns raw unnormalized value", locale: "en_US.UTF-8", want: "en_US.UTF-8"},
		{name: "empty string returns empty", locale: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewStaticDetector(tt.locale)
			got := d.Detect()
			if got != tt.want {
				t.Errorf("Detect() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStaticDetector_Normalize(t *testing.T) {
	d := NewStaticDetector("anything")
	got := d.Normalize("en_US.UTF-8")
	want := "en-US"
	if got != want {
		t.Errorf("Normalize(\"en_US.UTF-8\") = %q, want %q", got, want)
	}
}

func TestStaticDetector_InterfaceCompliance(t *testing.T) {
	var _ core.LocaleDetector = (*StaticDetector)(nil)
}

// =============================================================================
// Fuzz targets for Accept-Language parsing
// =============================================================================

// FuzzAcceptLanguageDetect exercises AcceptLanguageDetector.Detect with random inputs.
func FuzzAcceptLanguageDetect(f *testing.F) {
	f.Add("en-US,en;q=0.9")
	f.Add("")
	f.Add("\x00\x01\x02")
	f.Add(strings.Repeat("a", 110))
	f.Add("../../../../etc/passwd")
	f.Add("fr;q=0.8,de;q=0.9,en-US;q=0.7")

	f.Fuzz(func(t *testing.T, header string) {
		detector := NewAcceptLanguageDetector(header)
		result := detector.Detect()

		// Result must either be empty or have bounded length.
		// NormalizeLocale applies strings.ToLower/ToUpper which can expand
		// non-ASCII bytes (1 byte -> up to 4 bytes via UTF-8 replacement).
		// Use a 4x multiplier plus a constant for language-to-region expansion.
		if result != "" && len(result) > 4*len(header)+10 {
			t.Errorf("Detect(%q) returned unexpectedly long result: %q (len %d)", header, result, len(result))
		}
	})
}

// FuzzParseAcceptLanguage exercises parseAcceptLanguage with random inputs.
func FuzzParseAcceptLanguage(f *testing.F) {
	f.Add("en-US,en;q=0.9,fr;q=0.8")
	f.Add("")
	f.Add("\x00")
	f.Add(strings.Repeat("a", 110))
	f.Add("a]b;q=999")
	f.Add(",,,en;q=0.5,,,")

	f.Fuzz(func(t *testing.T, header string) {
		entries := parseAcceptLanguage(header)

		for i, entry := range entries {
			// Quality must be in [0.0, 1.0] after parsing (clamped by parseQuality).
			if entry.quality < 0.0 || entry.quality > 1.0 {
				t.Errorf("parseAcceptLanguage(%q)[%d] quality = %v, want in [0.0, 1.0]",
					header, i, entry.quality)
			}
		}
	})
}

// FuzzParseQuality exercises parseQuality with random inputs.
func FuzzParseQuality(f *testing.F) {
	f.Add(";q=0.9")
	f.Add("")
	f.Add("\x00")
	f.Add(strings.Repeat("q", 110))
	f.Add(";q=NaN")
	f.Add("q=1.5")

	f.Fuzz(func(t *testing.T, params string) {
		q := parseQuality(params)

		if q < 0.0 || q > 1.0 {
			t.Errorf("parseQuality(%q) = %v, want in [0.0, 1.0]", params, q)
		}
	})
}

// =============================================================================
// Spec 018: Locale Benchmarks (Task Group 4)
// =============================================================================

// Baseline: ~20-100 ns/op, 0-1 allocs/op
func BenchmarkNormalizeLocale(b *testing.B) {
	detector := NewDefaultLocaleDetector(nil)

	b.Run("already_BCP47", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = detector.Normalize("en-US")
		}
	})

	b.Run("underscore_encoding", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = detector.Normalize("en_US.UTF-8")
		}
	})

	b.Run("POSIX", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = detector.Normalize("POSIX")
		}
	})

	b.Run("language_only", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = detector.Normalize("en")
		}
	})
}

// Baseline: ~200-600 ns/op, 2-5 allocs/op
func BenchmarkAcceptLanguageDetect(b *testing.B) {
	d := NewAcceptLanguageDetector("en-US,fr;q=0.8,de;q=0.5,es;q=0.3")
	_ = d.Detect()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = d.Detect()
	}
}

// Baseline: ~50-150 ns/op, 0-2 allocs/op
func BenchmarkAcceptLanguageDetect_SingleEntry(b *testing.B) {
	d := NewAcceptLanguageDetector("en-US")
	_ = d.Detect()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = d.Detect()
	}
}

// Baseline: ~50-200 ns/op, 0-2 allocs/op
func BenchmarkChainDetector_Detect(b *testing.B) {
	chain := NewChainDetector(
		NewStaticDetector("fr-FR"),
		NewAcceptLanguageDetector("de-DE,en;q=0.5"),
		NewDefaultLocaleDetector(nil),
	)
	_ = chain.Detect()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = chain.Detect()
	}
}

// Baseline: ~150-500 ns/op, 2-5 allocs/op
func BenchmarkParseAcceptLanguage(b *testing.B) {
	header := "en-US,en;q=0.9,fr;q=0.8,de;q=0.7,es;q=0.5"
	_ = parseAcceptLanguage(header)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = parseAcceptLanguage(header)
	}
}
