// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

package engine

import (
	"strings"
	"testing"

	"github.com/0verkilll/i18n/internal/core"
)

// =============================================================================
// Locale validation tests
// =============================================================================

// TestValidateLocale tests locale string validation with BCP 47 format.
func TestValidateLocale(t *testing.T) {
	tests := []struct {
		name    string
		locale  string
		wantErr bool
	}{
		// Valid BCP 47 codes
		{name: "valid en-US", locale: "en-US", wantErr: false},
		{name: "valid es-ES", locale: "es-ES", wantErr: false},
		{name: "valid pt-BR", locale: "pt-BR", wantErr: false},
		{name: "valid zh-CN", locale: "zh-CN", wantErr: false},
		{name: "valid ja-JP", locale: "ja-JP", wantErr: false},
		{name: "valid fr-FR", locale: "fr-FR", wantErr: false},
		{name: "valid de-DE", locale: "de-DE", wantErr: false},
		{name: "valid 3-letter language", locale: "haw-US", wantErr: false},

		// Invalid formats
		{name: "empty string", locale: "", wantErr: true},
		{name: "only language uppercase", locale: "EN", wantErr: true},
		{name: "only language lowercase", locale: "en", wantErr: false},
		{name: "region lowercase", locale: "en-us", wantErr: true},
		{name: "language uppercase", locale: "EN-US", wantErr: true},
		{name: "with encoding", locale: "en-US.UTF-8", wantErr: true},
		{name: "underscore separator", locale: "en_US", wantErr: true},
		{name: "single char language", locale: "e-US", wantErr: true},
		{name: "single char region", locale: "en-U", wantErr: true},
		{name: "three char region", locale: "en-USA", wantErr: true},
		{name: "four letter language", locale: "engl-US", wantErr: true},
		{name: "valid language only es", locale: "es", wantErr: false},
		{name: "valid language only pt", locale: "pt", wantErr: false},
		{name: "numbers in locale", locale: "en-12", wantErr: true},

		// Path traversal attempts
		{name: "path traversal dots", locale: "../etc", wantErr: true},
		{name: "path traversal full", locale: "../../etc/passwd", wantErr: true},
		{name: "forward slash", locale: "en/US", wantErr: true},
		{name: "backslash", locale: "en\\US", wantErr: true},
		{name: "absolute path", locale: "/etc/passwd", wantErr: true},
		{name: "contains dot dot", locale: "en..US", wantErr: true},
		{name: "tilde path", locale: "~/config", wantErr: true},

		// Special characters
		{name: "null byte", locale: "en\x00US", wantErr: true},
		{name: "newline", locale: "en\nUS", wantErr: true},
		{name: "tab", locale: "en\tUS", wantErr: true},
		{name: "control char", locale: "en\x01US", wantErr: true},
		{name: "DEL character", locale: "en\x7FUS", wantErr: true},
		{name: "ANSI escape", locale: "en\x1b[31mUS", wantErr: true},

		// Length violations
		{name: "too long", locale: "en-US-extra", wantErr: true},
		{name: "exactly 10 chars", locale: "eng-US", wantErr: false},
		{name: "exceeds max", locale: strings.Repeat("a", 11), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := core.ValidateLocale(tt.locale)
			if (err != nil) != tt.wantErr {
				t.Errorf("core.ValidateLocale(%q) error = %v, wantErr %v", tt.locale, err, tt.wantErr)
			}
		})
	}
}

// TestMatchLocaleValid verifies matchLocale accepts valid BCP 47 locale strings.
func TestMatchLocaleValid(t *testing.T) {
	tests := []struct {
		name   string
		locale string
	}{
		{"2-letter language", "en"},
		{"2-letter with region", "en-US"},
		{"3-letter language", "haw"},
		{"3-letter with region", "haw-US"},
		{"language es", "es"},
		{"language es-MX", "es-MX"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !matchLocale(tt.locale) {
				t.Errorf("matchLocale(%q) = false, want true", tt.locale)
			}
		})
	}
}

// TestMatchLocaleInvalid verifies matchLocale rejects invalid patterns.
func TestMatchLocaleInvalid(t *testing.T) {
	tests := []struct {
		name   string
		locale string
	}{
		{"uppercase language", "EN-US"},
		{"lowercase region", "en-us"},
		{"single char language", "e"},
		{"4-letter language", "engl-US"},
		{"1-letter region", "en-U"},
		{"3-letter region", "en-USA"},
		{"empty string", ""},
		{"numbers in region", "en-12"},
		{"underscore separator", "en_US"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if matchLocale(tt.locale) {
				t.Errorf("matchLocale(%q) = true, want false", tt.locale)
			}
		})
	}
}

// Distributed from fuzz_test.go: FuzzValidateLocale
func FuzzValidateLocale(f *testing.F) {
	f.Add("en-US")
	f.Add("es-MX")
	f.Add("pt-BR")
	f.Add("../../etc/passwd")
	f.Add("\x00\x01\x02")
	f.Add("en\nUS")
	f.Add("en/US")
	f.Add("../etc")
	f.Add(strings.Repeat("a", 20))

	f.Fuzz(func(t *testing.T, locale string) {
		_ = core.ValidateLocale(locale) //nolint:errcheck // fuzz: must not panic
	})
}

// Distributed from benchmark_test.go: BenchmarkValidateLocale
func BenchmarkValidateLocale(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = core.ValidateLocale("en-US") //nolint:errcheck // benchmark
	}
}

// TestValidateLocaleSecurity_TildeOnly exercises the tilde path check
// independent of the path separator check.
func TestValidateLocaleSecurity_TildeOnly(t *testing.T) {
	err := core.ValidateLocale("~root")
	if err == nil {
		t.Error("core.ValidateLocale(~root) should return error for tilde prefix")
	}
}

// =============================================================================
// Key validation tests
// =============================================================================

// TestValidateKey tests translation key validation.
func TestValidateKey(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr bool
	}{
		// Valid keys
		{name: "simple key", key: "greeting", wantErr: false},
		{name: "nested key", key: "error.validation.required", wantErr: false},
		{name: "with underscores", key: "error_message", wantErr: false},
		{name: "with hyphens", key: "error-message", wantErr: false},
		{name: "mixed separators", key: "app.error_code", wantErr: false},
		{name: "numbers", key: "error.code404", wantErr: false},
		{name: "10 levels deep", key: "a.b.c.d.e.f.g.h.i.j", wantErr: false},

		// Invalid formats
		{name: "empty string", key: "", wantErr: true},
		{name: "starts with dot", key: ".error", wantErr: true},
		{name: "ends with dot", key: "error.", wantErr: true},
		{name: "consecutive dots", key: "error..validation", wantErr: true},
		{name: "space", key: "error message", wantErr: true},
		{name: "forward slash", key: "error/validation", wantErr: true},
		{name: "backslash", key: "error\\validation", wantErr: true},
		{name: "special char @", key: "error@validation", wantErr: true},
		{name: "special char #", key: "error#validation", wantErr: true},

		// Depth violations (> 10 levels)
		{name: "11 levels deep", key: "a.b.c.d.e.f.g.h.i.j.k", wantErr: true},
		{name: "15 levels deep", key: "a.b.c.d.e.f.g.h.i.j.k.l.m.n.o", wantErr: true},

		// Length violations (> 256 chars)
		{name: "exactly 256 chars", key: strings.Repeat("a", 256), wantErr: false},
		{name: "257 chars", key: strings.Repeat("a", 257), wantErr: true},
		{name: "500 chars", key: strings.Repeat("a", 500), wantErr: true},

		// Control characters
		{name: "null byte", key: "error\x00validation", wantErr: true},
		{name: "newline", key: "error\nvalidation", wantErr: true},
		{name: "tab", key: "error\tvalidation", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := core.ValidateKey(tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("core.ValidateKey(%q) error = %v, wantErr %v", tt.key, err, tt.wantErr)
			}
		})
	}
}

// TestMatchKeyValid verifies matchKey accepts valid key characters.
func TestMatchKeyValid(t *testing.T) {
	tests := []struct {
		name string
		key  string
	}{
		{"alphanumeric", "greeting123"},
		{"with dots", "error.validation.required"},
		{"with underscores", "error_message"},
		{"with hyphens", "error-message"},
		{"mixed", "app.error_code-v2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !matchKey(tt.key) {
				t.Errorf("matchKey(%q) = false, want true", tt.key)
			}
		})
	}
}

// TestMatchKeyInvalid verifies matchKey rejects keys with invalid characters.
func TestMatchKeyInvalid(t *testing.T) {
	tests := []struct {
		name string
		key  string
	}{
		{"space", "error message"},
		{"forward slash", "error/validation"},
		{"backslash", "error\\validation"},
		{"at sign", "error@validation"},
		{"hash", "error#validation"},
		{"empty", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if matchKey(tt.key) {
				t.Errorf("matchKey(%q) = true, want false", tt.key)
			}
		})
	}
}

// Distributed from fuzz_test.go: FuzzValidateKey
func FuzzValidateKey(f *testing.F) {
	f.Add("greeting")
	f.Add("error.validation.required")
	f.Add("a.b.c.d.e.f.g.h.i.j.k")
	f.Add(strings.Repeat("a", 300))
	f.Add(".error")
	f.Add("error.")
	f.Add("error..validation")
	f.Add("error/validation")
	f.Add("\x00test")

	f.Fuzz(func(t *testing.T, key string) {
		_ = core.ValidateKey(key) //nolint:errcheck // fuzz: must not panic
	})
}

// Distributed from benchmark_test.go: BenchmarkValidateKey
func BenchmarkValidateKey(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = core.ValidateKey("user.profile.title") //nolint:errcheck // benchmark
	}
}

// =============================================================================
// Format string validation tests
// =============================================================================

// TestValidateFormatString tests format string validation.
func TestValidateFormatString(t *testing.T) {
	tests := []struct {
		name     string
		format   string
		argCount int
		wantErr  bool
	}{
		// Valid format strings
		{name: "no format", format: "Hello, World!", argCount: 0, wantErr: false},
		{name: "one %s", format: "Hello, %s!", argCount: 1, wantErr: false},
		{name: "one %d", format: "Count: %d", argCount: 1, wantErr: false},
		{name: "one %v", format: "Value: %v", argCount: 1, wantErr: false},
		{name: "multiple", format: "%s %d %v", argCount: 3, wantErr: false},
		{name: "escaped %%", format: "100%% complete", argCount: 0, wantErr: false},
		{name: "mixed", format: "%s: %d%% complete", argCount: 2, wantErr: false},

		// Mismatched argument counts
		{name: "too few args", format: "%s %d", argCount: 1, wantErr: true},
		{name: "too many args", format: "%s", argCount: 2, wantErr: true},
		{name: "no format but args", format: "Hello", argCount: 1, wantErr: true},
		{name: "format but no args", format: "Hello %s", argCount: 0, wantErr: true},

		// Dangerous format specifiers
		{name: "%n specifier", format: "Test%n", argCount: 1, wantErr: true},
		{name: "multiple %n", format: "%n%n", argCount: 2, wantErr: true},

		// Edge cases
		{name: "empty format", format: "", argCount: 0, wantErr: false},
		{name: "only %%", format: "%%", argCount: 0, wantErr: false},
		{name: "many args", format: "%s%s%s%s%s", argCount: 5, wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := core.ValidateFormatString(tt.format, tt.argCount)
			if (err != nil) != tt.wantErr {
				t.Errorf("core.ValidateFormatString(%q, %d) error = %v, wantErr %v", tt.format, tt.argCount, err, tt.wantErr)
			}
		})
	}
}

// TestCountFormatSpecifiers verifies correct counting of format specifiers.
func TestCountFormatSpecifiers(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"no specifiers", "hello", 0},
		{"one %s", "hello %s", 1},
		{"two specifiers", "%s %d", 2},
		{"escaped %%", "100%%", 0},
		{"mixed", "%s: %d%% done", 2},
		{"percent at end", "hello %", 0},
		{"empty string", "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := countFormatSpecifiers(tt.input)
			if got != tt.want {
				t.Errorf("countFormatSpecifiers(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

// Distributed from fuzz_test.go: FuzzValidateFormatString
func FuzzValidateFormatString(f *testing.F) {
	f.Add("Hello %s", byte(1))
	f.Add("Value: %d %s", byte(2))
	f.Add("%s%s%s%s%s", byte(5))
	f.Add("No format", byte(0))
	f.Add("%n dangerous", byte(1))
	f.Add("%%escaped", byte(0))
	f.Add(strings.Repeat("%s", 10), byte(10))

	f.Fuzz(func(t *testing.T, format string, argCountByte byte) {
		argCount := int(argCountByte) % 20
		_ = core.ValidateFormatString(format, argCount) //nolint:errcheck // fuzz: must not panic
	})
}

// =============================================================================
// Merged from wasm_hardening_test.go: countFormatSpecifiers edge case
// =============================================================================

// TestCountFormatSpecifiersPercentAtEnd verifies countFormatSpecifiers handles % at end of string.
func TestCountFormatSpecifiersPercentAtEnd(t *testing.T) {
	got := countFormatSpecifiers("hello %")
	if got != 0 {
		t.Errorf("countFormatSpecifiers(\"hello %%\") = %d, want 0", got)
	}
}

// =============================================================================
// Output sanitization tests
// =============================================================================

// TestSanitizeOutput tests output sanitization.
func TestSanitizeOutput(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		// Valid strings (no change)
		{name: "simple text", input: "Hello, World!", want: "Hello, World!"},
		{name: "with newline", input: "Line 1\nLine 2", want: "Line 1\nLine 2"},
		{name: "with tab", input: "Col 1\tCol 2", want: "Col 1\tCol 2"},
		{name: "unicode", input: "Hllo Wrld 世界", want: "Hllo Wrld 世界"},

		// Control characters (should be removed, except \n and \t)
		{name: "null byte", input: "Hello\x00World", want: "HelloWorld"},
		{name: "bell", input: "Hello\x07World", want: "HelloWorld"},
		{name: "backspace", input: "Hello\x08World", want: "HelloWorld"},
		{name: "escape", input: "Hello\x1bWorld", want: "HelloWorld"},
		{name: "multiple control chars", input: "\x01\x02\x03Hello", want: "Hello"},

		// ANSI escape sequences
		{name: "ANSI color red", input: "\x1b[31mRed Text\x1b[0m", want: "Red Text"},
		{name: "ANSI bold", input: "\x1b[1mBold\x1b[0m", want: "Bold"},
		{name: "ANSI complex", input: "\x1b[1;31mBold Red\x1b[0m", want: "Bold Red"},
		{name: "ANSI cursor", input: "\x1b[2J\x1b[HHello", want: "Hello"},

		// BiDi override characters
		{name: "RLO U+202E", input: "Hello\u202EWorld", want: "HelloWorld"},
		{name: "LRO U+202D", input: "Hello\u202DWorld", want: "HelloWorld"},
		{name: "RLE U+202B", input: "Hello\u202BWorld", want: "HelloWorld"},
		{name: "LRE U+202A", input: "Hello\u202AWorld", want: "HelloWorld"},
		{name: "PDF U+202C", input: "Hello\u202CWorld", want: "HelloWorld"},
		{name: "multiple BiDi", input: "\u202A\u202B\u202CHello", want: "Hello"},

		// Mixed issues
		{name: "control + ANSI + BiDi", input: "\x01\x1b[31m\u202EHello", want: "Hello"},

		// Empty and whitespace
		{name: "empty string", input: "", want: ""},
		{name: "only spaces", input: "   ", want: "   "},
		{name: "only newlines", input: "\n\n\n", want: "\n\n\n"},

		// DEL character (0x7F)
		{name: "DEL character", input: "Text\x7FMore", want: "TextMore"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := core.SanitizeOutput(tt.input)
			if got != tt.want {
				t.Errorf("core.SanitizeOutput(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestSanitizeOutput_LengthLimit tests that oversized strings are truncated.
func TestSanitizeOutput_LengthLimit(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantLen   int
		truncated bool
	}{
		{name: "under limit", input: strings.Repeat("a", 100), wantLen: 100, truncated: false},
		{name: "at limit", input: strings.Repeat("a", 10240), wantLen: 10240, truncated: false},
		{name: "over limit by 1", input: strings.Repeat("a", 10241), wantLen: 10240, truncated: true},
		{name: "way over limit", input: strings.Repeat("a", 50000), wantLen: 10240, truncated: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := core.SanitizeOutput(tt.input)
			if len(got) != tt.wantLen {
				t.Errorf("core.SanitizeOutput() length = %d, want %d", len(got), tt.wantLen)
			}
		})
	}
}

// TestStripANSI verifies ANSI escape sequence removal.
func TestStripANSI(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"red color", "\x1b[31mRed\x1b[0m", "Red"},
		{"bold", "\x1b[1mBold\x1b[0m", "Bold"},
		{"complex params", "\x1b[1;31mText\x1b[0m", "Text"},
		{"no ANSI", "Hello World", "Hello World"},
		{"lone escape byte", "\x1bHello", "Hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripANSI(tt.input)
			if got != tt.want {
				t.Errorf("stripANSI(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestStripANSIPreservesUnicode verifies non-ANSI text including unicode is preserved.
func TestStripANSIPreservesUnicode(t *testing.T) {
	input := "Hello World 世界"
	got := stripANSI(input)
	if got != input {
		t.Errorf("stripANSI(%q) = %q, want %q", input, got, input)
	}
}

// Distributed from fuzz_test.go: FuzzSanitizeOutput
func FuzzSanitizeOutput(f *testing.F) {
	f.Add("Hello, World!")
	f.Add("\x1b[31mRed Text\x1b[0m")
	f.Add("Hello\u202EWorld")
	f.Add("\x00\x01\x02Hello")
	f.Add(strings.Repeat("a", 20000))
	f.Add("Line1\nLine2\tCol")
	f.Add("\u202A\u202B\u202CTest")

	f.Fuzz(func(t *testing.T, input string) {
		output := core.SanitizeOutput(input)

		if len(output) > core.MaxOutputLength {
			t.Errorf("core.SanitizeOutput produced output exceeding max length: %d > %d", len(output), core.MaxOutputLength)
		}

		for _, r := range output {
			if r != '\n' && r != '\t' && (r < 0x20 || r == 0x7F) {
				t.Errorf("core.SanitizeOutput produced output with control char: %U", r)
			}
		}

		if strings.ContainsAny(output, "\u202A\u202B\u202C\u202D\u202E") {
			t.Errorf("core.SanitizeOutput produced output with BiDi override characters")
		}
	})
}

// Distributed from benchmark_test.go: BenchmarkSanitizeOutput
func BenchmarkSanitizeOutput(b *testing.B) {
	input := "Hello, World! This is a test message."

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = core.SanitizeOutput(input)
	}
}

// Distributed from benchmark_test.go: BenchmarkSanitizeOutput_WithControlChars
func BenchmarkSanitizeOutput_WithControlChars(b *testing.B) {
	input := "Hello\x1b[31mRed\x1b[0m\u202EWorld"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = core.SanitizeOutput(input)
	}
}

// =============================================================================
// Merged from wasm_hardening_test.go: stripANSI malformed sequence tests
// =============================================================================

// TestStripANSIMalformedSequences verifies stripANSI handles nested/malformed ANSI sequences.
func TestStripANSIMalformedSequences(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"lone escape byte", "\x1bHello", "Hello"},
		{"escape not followed by bracket", "\x1bXHello", "XHello"},
		{"incomplete sequence at end", "\x1b[31", ""},
		{"empty after escape bracket", "\x1b[m", ""},
		{"double escape", "\x1b\x1b[31mHello", "Hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripANSI(tt.input)
			if got != tt.want {
				t.Errorf("stripANSI(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// =============================================================================
// Fuzz targets for validate internals
// =============================================================================

// FuzzStripANSI exercises stripANSI with random inputs.
func FuzzStripANSI(f *testing.F) {
	f.Add("\x1b[31mRed\x1b[0m")
	f.Add("")
	f.Add("\x00\x01")
	f.Add(strings.Repeat("a", 110))
	f.Add("\x1b[\x1b[\x1b[")
	f.Add("\x1b[1;31mBold Red\x1b[0m")

	f.Fuzz(func(t *testing.T, s string) {
		result := stripANSI(s)

		// Output must not contain ESC byte (0x1B).
		if strings.ContainsRune(result, 0x1B) {
			t.Errorf("stripANSI(%q) output contains ESC byte: %q", s, result)
		}

		// Output length must not exceed input length (stripping can only shrink or preserve).
		if len(result) > len(s) {
			t.Errorf("stripANSI(%q) output length %d exceeds input length %d", s, len(result), len(s))
		}
	})
}

// FuzzMatchLocale exercises matchLocale with random inputs.
func FuzzMatchLocale(f *testing.F) {
	f.Add("en-US")
	f.Add("")
	f.Add("\x00")
	f.Add(strings.Repeat("a", 110))
	f.Add("xx")
	f.Add("haw-US")

	f.Fuzz(func(t *testing.T, s string) {
		got := matchLocale(s)

		// If matchLocale returns true, the string must have length 2-6 and follow
		// the pattern: 2-3 lowercase letters optionally followed by '-' and 2 uppercase letters.
		if got {
			if len(s) < 2 || len(s) > 6 {
				t.Errorf("matchLocale(%q) = true but length %d is outside [2,6]", s, len(s))
			}
		}
	})
}

// FuzzCountFormatSpecifiers exercises countFormatSpecifiers with random inputs.
func FuzzCountFormatSpecifiers(f *testing.F) {
	f.Add("Hello %s, you have %d items")
	f.Add("")
	f.Add("\x00%")
	f.Add(strings.Repeat("a", 110))
	f.Add("%%%s%%")
	f.Add("%")

	f.Fuzz(func(t *testing.T, s string) {
		count := countFormatSpecifiers(s)

		// Returned int must be >= 0.
		if count < 0 {
			t.Errorf("countFormatSpecifiers(%q) = %d, want >= 0", s, count)
		}

		// Must not exceed the total number of '%' characters in the input.
		percentCount := strings.Count(s, "%")
		if count > percentCount {
			t.Errorf("countFormatSpecifiers(%q) = %d, exceeds %% count %d", s, count, percentCount)
		}
	})
}

// =============================================================================
// Spec 018: Validation and Sanitization Benchmarks (Task Group 6)
// =============================================================================

// Package-level benchmark strings for validation and sanitization.
var (
	benchLongCleanString = strings.Repeat("This is a clean translation string. ", 30) // ~1KB
	benchCJKString       = strings.Repeat("\u4f60\u597d\u4e16\u754c ", 50)            // CJK chars
	benchANSIString      = "Normal \x1b[31mred\x1b[0m text \x1b[1;32mbold green\x1b[0m end \x1b[4munderline\x1b[0m done"
	benchNestedKey       = "app.module.section.error.message"
	benchFormatString    = "%s has %d items (%v)"
)

// Baseline: ~10-30 ns/op, 0 allocs/op
func BenchmarkValidateLocale_Invalid(b *testing.B) {
	b.Run("empty", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = core.ValidateLocale("") //nolint:errcheck // benchmark
		}
	})

	b.Run("path_traversal", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = core.ValidateLocale("../etc") //nolint:errcheck // benchmark
		}
	})
}

// Baseline: ~30-80 ns/op, 0 allocs/op
func BenchmarkValidateKey_NestedKey(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = core.ValidateKey(benchNestedKey) //nolint:errcheck // benchmark
	}
}

// Baseline: ~30-80 ns/op, 0 allocs/op
func BenchmarkValidateFormatString(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = core.ValidateFormatString(benchFormatString, 3) //nolint:errcheck // benchmark
	}
}

// Baseline: ~200-800 ns/op, 1-3 allocs/op
func BenchmarkSanitizeOutput_LongClean(b *testing.B) {
	_ = core.SanitizeOutput(benchLongCleanString)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = core.SanitizeOutput(benchLongCleanString)
	}
}

// Baseline: ~200-800 ns/op, 1-3 allocs/op
func BenchmarkSanitizeOutput_Unicode(b *testing.B) {
	_ = core.SanitizeOutput(benchCJKString)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = core.SanitizeOutput(benchCJKString)
	}
}

// Baseline: ~100-400 ns/op, 1-2 allocs/op
func BenchmarkStripANSI(b *testing.B) {
	_ = stripANSI(benchANSIString)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = stripANSI(benchANSIString)
	}
}

// Baseline: ~30-80 ns/op, 0 allocs/op
func BenchmarkValidateKey_Parallel(b *testing.B) {
	_ = core.ValidateKey(benchNestedKey) //nolint:errcheck // warm
	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = core.ValidateKey(benchNestedKey) //nolint:errcheck // benchmark
		}
	})
}
