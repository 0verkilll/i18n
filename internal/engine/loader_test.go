// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

//go:build !tinygo && !js

package engine

import (
	"embed"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/0verkilll/i18n/internal/core"
)

func TestFileSystemLoader_Load(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := t.TempDir()

	// Create a test locale file
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	testContent := []byte(`{"greeting": "Hello", "farewell": "Goodbye"}`)
	testFile := filepath.Join(localeDir, "en-US.json")
	if err := os.WriteFile(testFile, testContent, 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		errType   error
		name      string
		baseDir   string
		locale    string
		wantErr   bool
		checkData bool
	}{
		{
			name:      "valid locale file",
			baseDir:   localeDir,
			locale:    "en-US",
			wantErr:   false,
			checkData: true,
		},
		{
			name:    "non-existent locale",
			baseDir: localeDir,
			locale:  "xx-YY",
			wantErr: true,
		},
		{
			name:    "path traversal with ..",
			baseDir: localeDir,
			locale:  "../../../etc/passwd",
			wantErr: true,
			errType: &core.ErrPathTraversal{},
		},
		{
			name:    "path traversal with absolute path",
			baseDir: localeDir,
			locale:  "/etc/passwd",
			wantErr: true,
			errType: &core.ErrPathTraversal{},
		},
		{
			name:    "path traversal with ~",
			baseDir: localeDir,
			locale:  "~/secret",
			wantErr: true,
			errType: &core.ErrPathTraversal{},
		},
		{
			name:    "path with forward slash",
			baseDir: localeDir,
			locale:  "subdir/en-US",
			wantErr: true,
			errType: &core.ErrPathTraversal{},
		},
		{
			name:    "path with backslash",
			baseDir: localeDir,
			locale:  "subdir\\en-US",
			wantErr: true,
			errType: &core.ErrPathTraversal{},
		},
		{
			name:    "empty locale",
			baseDir: localeDir,
			locale:  "",
			wantErr: true,
		},
		{
			name:    "empty base directory",
			baseDir: "",
			locale:  "en-US",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader := NewFileSystemLoader(tt.baseDir)
			data, err := loader.Load(tt.locale)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Load() expected error, got nil")
				}
				if tt.errType != nil {
					// Check if the expected error type is core.ErrPathTraversal
					var expectedPathErr *core.ErrPathTraversal
					if errors.As(tt.errType, &expectedPathErr) {
						// Verify the actual error is also core.ErrPathTraversal
						var actualPathErr *core.ErrPathTraversal
						if !errors.As(err, &actualPathErr) {
							t.Errorf("Load() error should be core.ErrPathTraversal, got: %v", err)
						}
					}
				}
				return
			}

			if err != nil {
				t.Errorf("Load() unexpected error: %v", err)
				return
			}

			if tt.checkData {
				if len(data) == 0 {
					t.Error("Load() returned empty data")
				}
				if !strings.Contains(string(data), "greeting") {
					t.Error("Load() data does not contain expected content")
				}
			}
		})
	}
}

func TestFileSystemLoader_LoadWithSubdirectories(t *testing.T) {
	// Test that loader rejects any path with directory separators
	tmpDir := t.TempDir()
	loader := NewFileSystemLoader(tmpDir)

	testCases := []string{
		"en-US/messages",
		"../en-US",
		"./en-US",
		"en-US/../es-ES",
	}

	for _, locale := range testCases {
		t.Run(locale, func(t *testing.T) {
			_, err := loader.Load(locale)
			if err == nil {
				t.Error("Load() should reject paths with separators")
			}
			if !IsPathTraversalError(err) {
				t.Errorf("Load() should return path traversal error, got: %v", err)
			}
		})
	}
}

//go:embed testdata/locales/*.json
var testLocales embed.FS

func TestEmbedFSLoader_Load(t *testing.T) {
	tests := []struct {
		errType   error
		name      string
		basePath  string
		locale    string
		wantErr   bool
		checkData bool
	}{
		{
			name:      "valid embedded locale",
			basePath:  "testdata/locales",
			locale:    "en-US",
			wantErr:   false,
			checkData: true,
		},
		{
			name:     "non-existent locale",
			basePath: "testdata/locales",
			locale:   "xx-YY",
			wantErr:  true,
		},
		{
			name:     "path traversal attempt",
			basePath: "testdata/locales",
			locale:   "../../../secret",
			wantErr:  true,
			errType:  &core.ErrPathTraversal{},
		},
		{
			name:     "absolute path attempt",
			basePath: "testdata/locales",
			locale:   "/etc/passwd",
			wantErr:  true,
			errType:  &core.ErrPathTraversal{},
		},
		{
			name:     "empty locale",
			basePath: "testdata/locales",
			locale:   "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader := NewEmbedFSLoader(testLocales, tt.basePath)
			data, err := loader.Load(tt.locale)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Load() expected error, got nil")
				}
				if tt.errType != nil {
					//noinspection GoTypeAssertionOnErrors
					if _, ok := tt.errType.(*core.ErrPathTraversal); ok { //nolint:errorlint // Type check for test expectation, not error handling
						if !IsPathTraversalError(err) {
							t.Errorf("Load() error should be path traversal, got: %v", err)
						}
					}
				}
				return
			}

			if err != nil {
				t.Errorf("Load() unexpected error: %v", err)
				return
			}

			if tt.checkData {
				if len(data) == 0 {
					t.Error("Load() returned empty data")
				}
				// Verify it's valid JSON-like content
				if !strings.Contains(string(data), "{") {
					t.Error("Load() data does not appear to be JSON")
				}
			}
		})
	}
}

func TestEmbedFSLoader_InvalidFS(t *testing.T) {
	// Test with empty embed.FS
	var emptyFS embed.FS
	loader := NewEmbedFSLoader(emptyFS, "nonexistent")

	_, err := loader.Load("en-US")
	if err == nil {
		t.Error("Load() should fail with invalid embed.FS")
	}
}

// Helper function to check if error is path traversal
func IsPathTraversalError(err error) bool {
	if err == nil {
		return false
	}
	var pathErr *core.ErrPathTraversal
	return errors.As(err, &pathErr)
}

// TestValidatePathRejectsTraversal verifies validatePath rejects path traversal via ..
func TestValidatePathRejectsTraversal(t *testing.T) {
	err := validatePath("../etc/passwd")
	if err == nil {
		t.Error("validatePath should reject path traversal")
	}
	if !IsPathTraversalError(err) {
		t.Errorf("validatePath should return core.ErrPathTraversal, got: %v", err)
	}
}

// TestValidatePathRejectsAbsolutePaths verifies validatePath rejects absolute paths.
func TestValidatePathRejectsAbsolutePaths(t *testing.T) {
	err := validatePath("/etc/passwd")
	if err == nil {
		t.Error("validatePath should reject absolute paths")
	}
	if !IsPathTraversalError(err) {
		t.Errorf("validatePath should return core.ErrPathTraversal, got: %v", err)
	}
}

func TestFileSystemLoader_DefaultExtension(t *testing.T) {
	tmpDir := t.TempDir()
	testContent := []byte(`{"greeting": "Hello"}`)
	if err := os.WriteFile(filepath.Join(tmpDir, "en-US.json"), testContent, 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	loader := NewFileSystemLoader(tmpDir)
	data, err := loader.Load("en-US")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !strings.Contains(string(data), "greeting") {
		t.Error("Load() data does not contain expected content")
	}
}

func TestFileSystemLoader_WithExtension(t *testing.T) {
	tmpDir := t.TempDir()
	testContent := []byte(`key = "value"`)
	if err := os.WriteFile(filepath.Join(tmpDir, "en-US.toml"), testContent, 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	loader := NewFileSystemLoader(tmpDir, WithExtension(".toml"))
	data, err := loader.Load("en-US")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !strings.Contains(string(data), "key") {
		t.Error("Load() data does not contain expected content")
	}
}

func TestEmbedFSLoader_DefaultExtension(t *testing.T) {
	loader := NewEmbedFSLoader(testLocales, "testdata/locales")
	data, err := loader.Load("en-US")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(data) == 0 {
		t.Error("Load() returned empty data")
	}
}

func TestEmbedFSLoader_WithExtension(t *testing.T) {
	// Using an extension that does not match any embedded file triggers an error,
	// confirming the extension was applied.
	loader := NewEmbedFSLoader(testLocales, "testdata/locales", WithExtension(".yaml"))
	_, err := loader.Load("en-US")
	if err == nil {
		t.Error("Load() should fail when extension does not match embedded files")
	}
}

func TestWithExtension_OverridesDefault(t *testing.T) {
	tmpDir := t.TempDir()
	testContent := []byte(`greeting: Hello`)
	if err := os.WriteFile(filepath.Join(tmpDir, "en-US.yaml"), testContent, 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	loader := NewFileSystemLoader(tmpDir, WithExtension(".yaml"))
	data, err := loader.Load("en-US")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !strings.Contains(string(data), "greeting") {
		t.Error("Load() data does not contain expected content from .yaml file")
	}
}

// Coverage gap: validatePath tilde check independent of slash checks
func TestValidatePath_TildeOnly(t *testing.T) {
	err := validatePath("~test")
	if err == nil {
		t.Error("validatePath(~test) should return error for tilde prefix")
	}
	if !IsPathTraversalError(err) {
		t.Errorf("validatePath(~test) should return core.ErrPathTraversal, got: %v", err)
	}
}

// Coverage gap: FileSystemLoader.Load non-NotExist error (e.g., permission denied)
func TestFileSystemLoader_Load_ReadError(t *testing.T) {
	tmpDir := t.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Create a file that is a directory to provoke a read error (not "not exist")
	dirAsFile := filepath.Join(localeDir, "en-US.json")
	if err := os.MkdirAll(dirAsFile, 0o755); err != nil {
		t.Fatalf("Failed to create directory posing as file: %v", err)
	}

	loader := NewFileSystemLoader(localeDir)
	_, err := loader.Load("en-US")
	if err == nil {
		t.Error("Load() should fail when file is actually a directory")
	}
	// The error should NOT be path traversal (locale is valid)
	if IsPathTraversalError(err) {
		t.Error("Load() error should not be core.ErrPathTraversal for read errors")
	}
}

// Coverage gap: FuzzFileSystemLoader_Load
func FuzzFileSystemLoader_Load(f *testing.F) {
	tmpDir := f.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		f.Fatalf("Failed to create test directory: %v", err)
	}
	enUS := []byte(`{"greeting": "Hello"}`)
	if err := os.WriteFile(filepath.Join(localeDir, "en-US.json"), enUS, 0o644); err != nil {
		f.Fatalf("Failed to create en-US file: %v", err)
	}

	loader := NewFileSystemLoader(localeDir)

	f.Add("en-US")
	f.Add("")
	f.Add("../etc/passwd")
	f.Add("\x00\x01\x02")
	f.Add(strings.Repeat("a", 300))

	f.Fuzz(func(t *testing.T, locale string) {
		_, _ = loader.Load(locale) //nolint:errcheck // fuzz: must not panic
	})
}

// =============================================================================
// Merged from wasm_hardening_test.go: EmbedFSLoader test
// Uses the existing testLocales embed.FS variable declared above.
// =============================================================================

// TestEmbedFSLoaderStillWorksAfterSplit verifies that EmbedFSLoader works after the loader split.
func TestEmbedFSLoaderStillWorksAfterSplit(t *testing.T) {
	loader := NewEmbedFSLoader(testLocales, "testdata/locales")
	data, err := loader.Load("en-US")
	if err != nil {
		t.Fatalf("EmbedFSLoader.Load() error: %v", err)
	}
	if len(data) == 0 {
		t.Error("EmbedFSLoader.Load() returned empty data")
	}
	if !strings.Contains(string(data), "{") {
		t.Error("EmbedFSLoader.Load() data does not appear to be JSON")
	}
}

// =============================================================================
// Fuzz target for EmbedFSLoader
// =============================================================================

// FuzzEmbedFSLoader exercises EmbedFSLoader.Load with random locale inputs.
func FuzzEmbedFSLoader(f *testing.F) {
	f.Add("en-US")
	f.Add("")
	f.Add("\x00\x01")
	f.Add(strings.Repeat("a", 110))
	f.Add("../../etc/passwd")
	f.Add("xx-YY")

	loader := NewEmbedFSLoader(testLocales, "testdata/locales")

	f.Fuzz(func(t *testing.T, locale string) {
		data, err := loader.Load(locale)
		if err != nil {
			// If error is path traversal, verify typed error.
			var pathErr *core.ErrPathTraversal
			if errors.As(err, &pathErr) {
				// Valid typed error -- OK.
				return
			}
			// core.Other errors are expected for nonexistent locales.
			return
		}

		// If no error, returned data must be non-nil.
		if data == nil {
			t.Errorf("EmbedFSLoader.Load(%q) returned nil data without error", locale)
		}
	})
}

// =============================================================================
// Locale registry tests (merged from registry_test.go)
// =============================================================================

// requireLocaleAll is a test helper that skips the test when locale_all
// build-tag locales are not registered (i.e., init() did not run).
func requireLocaleAll(t *testing.T) {
	t.Helper()
	codes := RegisteredLocales()
	if len(codes) < 13 {
		t.Skip("skipping: locale_all build tag not active (expected at least 13 locales registered by init)")
	}
}

func TestLocaleAll_RegistersExampleLocales(t *testing.T) {
	requireLocaleAll(t)

	codes := RegisteredLocales()

	expected := map[string]bool{
		"en-US": false,
		"es-ES": false,
		"pt-BR": false,
	}

	for _, code := range codes {
		if _, ok := expected[code]; ok {
			expected[code] = true
		}
	}

	for code, found := range expected {
		if !found {
			t.Errorf("expected locale %q to be registered under locale_all tag", code)
		}
	}
}

func TestLocaleAll_TranslatorWithRegistryLoader(t *testing.T) {
	requireLocaleAll(t)

	translator, err := New(
		WithRegistryLoader(),
		WithDefaultLocale("en-US"),
	)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	result := translator.Translate("greeting")
	if result != "Hello" {
		t.Errorf("Translate(greeting) with en-US = %q, want %q", result, "Hello")
	}

	translator.SetLocale("es-ES")
	result = translator.Translate("greeting")
	if result != "Hola" {
		t.Errorf("Translate(greeting) with es-ES = %q, want %q", result, "Hola")
	}
}

func TestLocaleAll_RegisteredCount(t *testing.T) {
	requireLocaleAll(t)

	codes := RegisteredLocales()
	if len(codes) < 13 {
		t.Errorf("RegisteredLocales() returned %d codes, want at least 13", len(codes))
	}
}

func TestLocaleAll_NoTagsEmptyRegistry(t *testing.T) {
	t.Log("no-tag behavior verified via TestRegisteredLocales_Empty in this file")
}

func TestLocaleAll_RegistersNewLocales(t *testing.T) {
	requireLocaleAll(t)

	codes := RegisteredLocales()
	codeSet := make(map[string]bool, len(codes))
	for _, code := range codes {
		codeSet[code] = true
	}

	newLocales := []string{
		"fr-FR", "de-DE", "ja-JP", "zh-CN", "zh-TW",
		"ko-KR", "ar-SA", "it-IT", "ru-RU", "hi-IN",
	}

	for _, code := range newLocales {
		if !codeSet[code] {
			t.Errorf("expected locale %q to be registered under locale_all tag", code)
		}
	}
}

func TestLocaleAll_AllLocalesGreeting(t *testing.T) {
	requireLocaleAll(t)

	locales := []string{
		"en-US", "es-ES", "pt-BR",
		"fr-FR", "de-DE", "ja-JP", "zh-CN", "zh-TW",
		"ko-KR", "ar-SA", "it-IT", "ru-RU", "hi-IN",
	}

	for _, code := range locales {
		t.Run(code, func(t *testing.T) {
			translator, err := New(
				WithRegistryLoader(),
				WithDefaultLocale(code),
			)
			if err != nil {
				t.Fatalf("New() error: %v", err)
			}

			result := translator.Translate("greeting")
			if result == "" {
				t.Errorf("Translate(greeting) for locale %q returned empty string", code)
			}
			if result == "greeting" {
				t.Errorf("Translate(greeting) for locale %q returned the key itself (no translation found)", code)
			}
		})
	}
}

func TestLocaleAll_NewLocaleTranslation(t *testing.T) {
	requireLocaleAll(t)

	translator, err := New(
		WithRegistryLoader(),
		WithDefaultLocale("ja-JP"),
	)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	result := translator.Translate("greeting")
	if result == "" {
		t.Error("Translate(greeting) for ja-JP returned empty string")
	}
	if result == "greeting" {
		t.Error("Translate(greeting) for ja-JP returned the key itself (no translation found)")
	}
}

func TestRegisterLocale_ValidCode(t *testing.T) {
	resetRegistry()
	defer resetRegistry()

	if err := registerLocale("en-US", []byte(`{"greeting": "Hello"}`)); err != nil {
		t.Fatalf("registerLocale(en-US) returned error: %v", err)
	}

	registryMu.RLock()
	data, ok := localeRegistry["en-US"]
	registryMu.RUnlock()

	if !ok {
		t.Fatal("registerLocale did not add en-US to the registry")
	}
	if string(data) != `{"greeting": "Hello"}` {
		t.Errorf("registered data = %q, want %q", string(data), `{"greeting": "Hello"}`)
	}
}

func TestRegisterLocale_InvalidCode(t *testing.T) {
	resetRegistry()
	defer resetRegistry()

	// An invalid locale code is expected to be rejected with an error.
	_ = registerLocale("INVALID", []byte(`{"greeting": "Hello"}`))

	registryMu.RLock()
	_, ok := localeRegistry["INVALID"]
	registryMu.RUnlock()

	if ok {
		t.Error("registerLocale should silently skip an invalid locale code")
	}
}

func TestRegisteredLocales_Sorted(t *testing.T) {
	resetRegistry()
	defer resetRegistry()

	for _, code := range []string{"pt-BR", "en-US", "es-ES"} {
		if err := registerLocale(code, []byte(`{}`)); err != nil {
			t.Fatalf("registerLocale(%q) returned error: %v", code, err)
		}
	}

	codes := RegisteredLocales()
	if len(codes) != 3 {
		t.Fatalf("RegisteredLocales() returned %d codes, want 3", len(codes))
	}

	expected := []string{"en-US", "es-ES", "pt-BR"}
	for i, code := range codes {
		if code != expected[i] {
			t.Errorf("RegisteredLocales()[%d] = %q, want %q", i, code, expected[i])
		}
	}
}

func TestRegisteredLocales_Empty(t *testing.T) {
	resetRegistry()
	defer resetRegistry()

	codes := RegisteredLocales()
	if codes == nil {
		t.Error("RegisteredLocales() should return a non-nil empty slice, got nil")
	}
	if len(codes) != 0 {
		t.Errorf("RegisteredLocales() returned %d codes, want 0", len(codes))
	}
}

func TestRegisterLocale_ConcurrentAccess(t *testing.T) {
	resetRegistry()
	defer resetRegistry()

	var wg sync.WaitGroup
	locales := []struct {
		code string
		data []byte
	}{
		{"en-US", []byte(`{"greeting": "Hello"}`)},
		{"es-ES", []byte(`{"greeting": "Hola"}`)},
		{"pt-BR", []byte(`{"greeting": "Ola"}`)},
	}

	for _, loc := range locales {
		wg.Add(1)
		go func(code string, data []byte) {
			defer wg.Done()
			if err := registerLocale(code, data); err != nil {
				t.Errorf("registerLocale(%q) returned error: %v", code, err)
			}
		}(loc.code, loc.data)
	}

	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = RegisteredLocales()
		}()
	}

	wg.Wait()

	codes := RegisteredLocales()
	if len(codes) != 3 {
		t.Errorf("RegisteredLocales() returned %d codes after concurrent access, want 3", len(codes))
	}
}

func TestRegistry_EndToEnd_RegisterLoadTranslate(t *testing.T) {
	resetRegistry()
	defer resetRegistry()

	binData, err := EncodeBinary(map[string]string{"greeting": "Hello", "farewell": "Goodbye"})
	if err != nil {
		t.Fatalf("EncodeBinary: %v", err)
	}
	if err := registerLocale("en-US", binData); err != nil {
		t.Fatalf("registerLocale(en-US) returned error: %v", err)
	}

	translator, err := New(
		WithRegistryLoader(),
		WithDefaultLocale("en-US"),
	)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	got := translator.Translate("greeting")
	if got != "Hello" {
		t.Errorf("Translate(greeting) = %q, want %q", got, "Hello")
	}

	got = translator.Translate("farewell")
	if got != "Goodbye" {
		t.Errorf("Translate(farewell) = %q, want %q", got, "Goodbye")
	}
}

func TestRegistry_FallbackChain(t *testing.T) {
	resetRegistry()
	defer resetRegistry()

	binData, err := EncodeBinary(map[string]string{"greeting": "Hello", "farewell": "Goodbye"})
	if err != nil {
		t.Fatalf("EncodeBinary: %v", err)
	}
	if err := registerLocale("en-US", binData); err != nil {
		t.Fatalf("registerLocale(en-US) returned error: %v", err)
	}

	translator, err := New(
		WithRegistryLoader(),
		WithDefaultLocale("en-US"),
	)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	translator.SetLocale("es-MX")

	got := translator.Translate("greeting")
	if got != "Hello" {
		t.Errorf("Translate(greeting) after fallback = %q, want %q", got, "Hello")
	}
}

func TestRegisterLocale_DuplicateOverwrites(t *testing.T) {
	resetRegistry()
	defer resetRegistry()

	if err := registerLocale("en-US", []byte(`{"greeting": "First"}`)); err != nil {
		t.Fatalf("registerLocale(en-US) returned error: %v", err)
	}
	if err := registerLocale("en-US", []byte(`{"greeting": "Second"}`)); err != nil {
		t.Fatalf("registerLocale(en-US) returned error: %v", err)
	}

	loader := NewRegistryLoader()
	data, err := loader.Load("en-US")
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if !strings.Contains(string(data), "Second") {
		t.Errorf("duplicate registration should overwrite; got %q", string(data))
	}
}

func TestRegistryLoader_Load_EmptyLocale(t *testing.T) {
	resetRegistry()
	defer resetRegistry()

	loader := NewRegistryLoader()
	_, err := loader.Load("")
	if err == nil {
		t.Fatal("Load(\"\") expected error, got nil")
	}
	if !strings.Contains(err.Error(), "not registered") {
		t.Errorf("error should indicate locale is not registered, got: %v", err)
	}
}

func TestRegisteredLocales_Immutability(t *testing.T) {
	resetRegistry()
	defer resetRegistry()

	for _, code := range []string{"en-US", "es-ES"} {
		if err := registerLocale(code, []byte(`{}`)); err != nil {
			t.Fatalf("registerLocale(%q) returned error: %v", code, err)
		}
	}

	codes := RegisteredLocales()
	codes[0] = "MUTATED"

	fresh := RegisteredLocales()
	for _, code := range fresh {
		if code == "MUTATED" {
			t.Error("modifying the returned slice should not affect the registry")
		}
	}
}

func TestRegistryLoader_ErrorMentionsBuildTags(t *testing.T) {
	resetRegistry()
	defer resetRegistry()

	loader := NewRegistryLoader()
	_, err := loader.Load("fr-FR")
	if err == nil {
		t.Fatal("Load() expected error, got nil")
	}
	if !strings.Contains(err.Error(), "-tags") {
		t.Errorf("error should mention build tags, got: %v", err)
	}
}

func TestRegistryLoader_Load_ValidLocale(t *testing.T) {
	resetRegistry()
	defer resetRegistry()

	original := []byte(`{"greeting": "Hello"}`)
	if err := registerLocale("en-US", original); err != nil {
		t.Fatalf("registerLocale(en-US) returned error: %v", err)
	}

	loader := NewRegistryLoader()
	data, err := loader.Load("en-US")
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}
	if string(data) != `{"greeting": "Hello"}` {
		t.Errorf("Load() = %q, want %q", string(data), `{"greeting": "Hello"}`)
	}
}

func TestRegistryLoader_Load_UnregisteredLocale(t *testing.T) {
	resetRegistry()
	defer resetRegistry()

	loader := NewRegistryLoader()
	_, err := loader.Load("fr-FR")
	if err == nil {
		t.Fatal("Load() expected error for unregistered locale, got nil")
	}
	if !strings.Contains(err.Error(), "fr-FR") {
		t.Errorf("error should mention the locale code, got: %v", err)
	}
	if !strings.Contains(err.Error(), "not registered") {
		t.Errorf("error should mention that locale is not registered, got: %v", err)
	}
}

func TestRegistryLoader_Load_CopySafety(t *testing.T) {
	resetRegistry()
	defer resetRegistry()

	if err := registerLocale("en-US", []byte(`{"greeting": "Hello"}`)); err != nil {
		t.Fatalf("registerLocale(en-US) returned error: %v", err)
	}

	loader := NewRegistryLoader()
	data, err := loader.Load("en-US")
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}

	for i := range data {
		data[i] = 'X'
	}

	data2, err := loader.Load("en-US")
	if err != nil {
		t.Fatalf("Load() unexpected error on second call: %v", err)
	}
	if string(data2) != `{"greeting": "Hello"}` {
		t.Errorf("registry data was mutated: got %q, want %q", string(data2), `{"greeting": "Hello"}`)
	}
}

func TestNewRegistryLoader_NonNil(t *testing.T) {
	loader := NewRegistryLoader()
	if loader == nil {
		t.Fatal("NewRegistryLoader() returned nil")
	}
}

func TestWithRegistryLoader_SetsLoader(t *testing.T) {
	resetRegistry()
	defer resetRegistry()

	binData, err := EncodeBinary(map[string]string{"greeting": "Hello"})
	if err != nil {
		t.Fatalf("EncodeBinary: %v", err)
	}
	if err := registerLocale("en-US", binData); err != nil {
		t.Fatalf("registerLocale(en-US) returned error: %v", err)
	}

	translator, err := New(
		WithRegistryLoader(),
		WithDefaultLocale("en-US"),
	)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	if translator == nil {
		t.Fatal("New() returned nil translator")
	}

	result := translator.Translate("greeting")
	if result != "Hello" {
		t.Errorf("Translate(greeting) = %q, want %q", result, "Hello")
	}
}

func FuzzRegistryLoaderLoad(f *testing.F) {
	f.Add("en-US")
	f.Add("")
	f.Add("\x00\x01")
	f.Add(strings.Repeat("a", 110))
	f.Add("../../etc/passwd")
	f.Add("xx-YY")

	f.Fuzz(func(t *testing.T, locale string) {
		resetRegistry()
		if err := registerLocale("en-US", []byte(`{"greeting": "Hello"}`)); err != nil {
			t.Fatalf("registerLocale(en-US) returned error: %v", err)
		}

		loader := NewRegistryLoader()
		data, err := loader.Load(locale)
		if err != nil {
			if err.Error() == "" {
				t.Errorf("RegistryLoader.Load(%q) returned error with empty message", locale)
			}
			return
		}

		if data == nil {
			t.Errorf("RegistryLoader.Load(%q) returned nil data without error", locale)
		}
		if len(data) == 0 {
			t.Errorf("RegistryLoader.Load(%q) returned empty data without error", locale)
		}
	})
}
