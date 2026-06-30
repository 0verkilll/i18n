// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

package engine

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/0verkilll/i18n/internal/core"
)

func TestNew_DefaultOptions(t *testing.T) {
	tmpDir := t.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	testContent := []byte(`{"greeting": "Hello", "farewell": "Goodbye"}`)
	if err := os.WriteFile(filepath.Join(localeDir, "en-US.json"), testContent, 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	translator, err := New(
		WithFileSystemLoader(localeDir),
		WithDefaultLocale("en-US"),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if translator == nil {
		t.Fatal("New() returned nil translator")
	}
	if translator.GetLocale() != "en-US" {
		t.Errorf("GetLocale() = %q, want %q", translator.GetLocale(), "en-US")
	}
}

func TestNew_WithCustomComponents(t *testing.T) {
	tmpDir := t.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	testContent := []byte(`{"test": "value"}`)
	if err := os.WriteFile(filepath.Join(localeDir, "en-US.json"), testContent, 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	loader := NewFileSystemLoader(localeDir)
	parser := NewJSONParser()
	resolver := NewDefaultKeyResolver()
	detector := NewDefaultLocaleDetector(&MockEnvProvider{vars: map[string]string{"LANG": "en-US"}})
	chainer := NewDefaultFallbackChainer()

	translator, err := New(
		WithLoader(loader),
		WithParser(parser),
		WithResolver(resolver),
		WithLocaleDetector(detector),
		WithFallbackChainer(chainer),
		WithDefaultLocale("en-US"),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if translator == nil {
		t.Fatal("New() returned nil translator")
	}
}

func TestNew_MissingRequiredOptions(t *testing.T) {
	_, err := New(WithDefaultLocale("en-US"))
	if err == nil {
		t.Error("New() should fail without loader")
	}
}

func TestTranslator_Translate(t *testing.T) {
	tmpDir := t.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	enUS := []byte(`{"greeting":"Hello","farewell":"Goodbye","error":{"validation":{"required":"This field is required"}}}`)
	if err := os.WriteFile(filepath.Join(localeDir, "en-US.json"), enUS, 0o644); err != nil {
		t.Fatalf("Failed to create en-US file: %v", err)
	}
	esES := []byte(`{"greeting":"Hola","farewell":"Adios","error":{"validation":{"required":"Este campo es obligatorio"}}}`)
	if err := os.WriteFile(filepath.Join(localeDir, "es-ES.json"), esES, 0o644); err != nil {
		t.Fatalf("Failed to create es-ES file: %v", err)
	}
	esMX := []byte(`{"greeting":"Hola"}`)
	if err := os.WriteFile(filepath.Join(localeDir, "es-MX.json"), esMX, 0o644); err != nil {
		t.Fatalf("Failed to create es-MX file: %v", err)
	}

	translator, err := New(WithFileSystemLoader(localeDir), WithDefaultLocale("en-US"))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	tests := []struct {
		name   string
		locale string
		key    string
		want   string
	}{
		{name: "simple key - en-US", locale: "en-US", key: "greeting", want: "Hello"},
		{name: "nested key - en-US", locale: "en-US", key: "error.validation.required", want: "This field is required"},
		{name: "simple key - es-ES", locale: "es-ES", key: "greeting", want: "Hola"},
		{name: "fallback to es-ES from es-MX", locale: "es-MX", key: "farewell", want: "Adios"},
		{name: "missing key returns key itself", locale: "en-US", key: "nonexistent.key", want: "nonexistent.key"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			translator.SetLocale(tt.locale)
			result := translator.Translate(tt.key)
			if result != tt.want {
				t.Errorf("Translate(%q) = %q, want %q", tt.key, result, tt.want)
			}
		})
	}
}

func TestTranslator_TranslateWithArgs(t *testing.T) {
	tmpDir := t.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	enUS := []byte(`{"welcome":"Welcome, %s!","items_count":"You have %d items","price":"Price: $%.2f","multi":"%s has %d items"}`)
	if err := os.WriteFile(filepath.Join(localeDir, "en-US.json"), enUS, 0o644); err != nil {
		t.Fatalf("Failed to create en-US file: %v", err)
	}

	translator, err := New(WithFileSystemLoader(localeDir), WithDefaultLocale("en-US"))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	tests := []struct {
		name string
		key  string
		want string
		args []interface{}
	}{
		{name: "string argument", key: "welcome", args: []interface{}{"Alice"}, want: "Welcome, Alice!"},
		{name: "integer argument", key: "items_count", args: []interface{}{5}, want: "You have 5 items"},
		{name: "float argument", key: "price", args: []interface{}{19.99}, want: "Price: $19.99"},
		{name: "multiple arguments", key: "multi", args: []interface{}{"Cart", 3}, want: "Cart has 3 items"},
		{name: "missing key returns key", key: "missing", args: []interface{}{"test"}, want: "missing"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := translator.TranslateWithArgs(tt.key, tt.args...)
			if result != tt.want {
				t.Errorf("TranslateWithArgs(%q, %v) = %q, want %q", tt.key, tt.args, result, tt.want)
			}
		})
	}
}

func TestTranslator_SetLocale_GetLocale(t *testing.T) {
	tmpDir := t.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	for _, locale := range []string{"en-US", "es-ES", "fr-FR"} {
		content := []byte(`{"test": "value"}`)
		if err := os.WriteFile(filepath.Join(localeDir, locale+".json"), content, 0o644); err != nil {
			t.Fatalf("Failed to create %s file: %v", locale, err)
		}
	}

	translator, err := New(WithFileSystemLoader(localeDir), WithDefaultLocale("en-US"))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if got := translator.GetLocale(); got != "en-US" {
		t.Errorf("GetLocale() = %q, want %q", got, "en-US")
	}
	translator.SetLocale("es-ES")
	if got := translator.GetLocale(); got != "es-ES" {
		t.Errorf("After SetLocale(es-ES), GetLocale() = %q, want %q", got, "es-ES")
	}
	translator.SetLocale("fr_FR")
	if got := translator.GetLocale(); got != "fr-FR" {
		t.Errorf("After SetLocale(fr_FR), GetLocale() = %q, want %q (normalized)", got, "fr-FR")
	}
}

func TestTranslator_HasKey(t *testing.T) {
	tmpDir := t.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	enUS := []byte(`{"greeting":"Hello","error":{"validation":{"required":"Required"}}}`)
	if err := os.WriteFile(filepath.Join(localeDir, "en-US.json"), enUS, 0o644); err != nil {
		t.Fatalf("Failed to create en-US file: %v", err)
	}
	esES := []byte(`{"greeting":"Hola"}`)
	if err := os.WriteFile(filepath.Join(localeDir, "es-ES.json"), esES, 0o644); err != nil {
		t.Fatalf("Failed to create es-ES file: %v", err)
	}

	translator, err := New(WithFileSystemLoader(localeDir), WithDefaultLocale("en-US"))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	tests := []struct {
		name   string
		locale string
		key    string
		want   bool
	}{
		{name: "existing simple key", locale: "en-US", key: "greeting", want: true},
		{name: "existing nested key", locale: "en-US", key: "error.validation.required", want: true},
		{name: "missing key", locale: "en-US", key: "nonexistent", want: false},
		{name: "key exists in fallback", locale: "es-ES", key: "error.validation.required", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			translator.SetLocale(tt.locale)
			got := translator.HasKey(tt.key)
			if got != tt.want {
				t.Errorf("HasKey(%q) = %v, want %v", tt.key, got, tt.want)
			}
		})
	}
}

func TestTranslator_ConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	for _, locale := range []string{"en-US", "es-ES", "fr-FR"} {
		content := []byte(`{"greeting":"Hello","farewell":"Goodbye"}`)
		if err := os.WriteFile(filepath.Join(localeDir, locale+".json"), content, 0o644); err != nil {
			t.Fatalf("Failed to create %s file: %v", locale, err)
		}
	}

	translator, err := New(WithFileSystemLoader(localeDir), WithDefaultLocale("en-US"))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	var wg sync.WaitGroup
	numGoroutines := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = translator.Translate("greeting")
			_ = translator.GetLocale()
			_ = translator.HasKey("greeting")
		}()
	}
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			locales := []string{"en-US", "es-ES", "fr-FR"}
			translator.SetLocale(locales[idx%len(locales)])
		}(i)
	}

	wg.Wait()

	result := translator.Translate("greeting")
	if result == "" {
		t.Error("Translator not functional after concurrent access")
	}
}

func TestTranslator_OptionErrors(t *testing.T) {
	tests := []struct {
		opt  Option
		name string
	}{
		{name: "WithLoader nil", opt: WithLoader(nil)},
		{name: "WithParser nil", opt: WithParser(nil)},
		{name: "WithResolver nil", opt: WithResolver(nil)},
		{name: "WithLocaleDetector nil", opt: WithLocaleDetector(nil)},
		{name: "WithFallbackChainer nil", opt: WithFallbackChainer(nil)},
		{name: "WithPluralResolver nil", opt: WithPluralResolver(nil)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			localeDir := filepath.Join(tmpDir, "locales")
			if err := os.MkdirAll(localeDir, 0o755); err != nil {
				t.Fatalf("Failed to create test directory: %v", err)
			}
			testContent := []byte(`{"test": "value"}`)
			if err := os.WriteFile(filepath.Join(localeDir, "en-US.json"), testContent, 0o644); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			_, err := New(
				WithFileSystemLoader(localeDir),
				WithDefaultLocale("en-US"),
				tt.opt,
			)
			if err == nil {
				t.Errorf("Expected error for %s, got nil", tt.name)
			}
		})
	}
}

func TestTranslator_WithDefaultLocaleInvalid(t *testing.T) {
	tmpDir := t.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	testContent := []byte(`{"test": "value"}`)
	if err := os.WriteFile(filepath.Join(localeDir, "en-US.json"), testContent, 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	_, err := New(WithFileSystemLoader(localeDir), WithDefaultLocale("../../etc/passwd"))
	if err == nil {
		t.Error("Expected error for invalid default locale, got nil")
	}
}

func TestTranslator_CacheHit(t *testing.T) {
	tmpDir := t.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	enUS := []byte(`{"greeting":"Hello","farewell":"Goodbye"}`)
	if err := os.WriteFile(filepath.Join(localeDir, "en-US.json"), enUS, 0o644); err != nil {
		t.Fatalf("Failed to create en-US file: %v", err)
	}

	translator, err := New(WithFileSystemLoader(localeDir), WithDefaultLocale("en-US"))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	result1 := translator.Translate("greeting")
	if result1 != "Hello" {
		t.Errorf("First Translate() = %q, want %q", result1, "Hello")
	}
	result2 := translator.Translate("farewell")
	if result2 != "Goodbye" {
		t.Errorf("Second Translate() = %q, want %q", result2, "Goodbye")
	}
	if !translator.HasKey("greeting") {
		t.Error("HasKey() should return true for cached key")
	}
}

func TestTranslator_TranslateWithArgsValidationError(t *testing.T) {
	tmpDir := t.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	enUS := []byte(`{"message":"Hello %s %s"}`)
	if err := os.WriteFile(filepath.Join(localeDir, "en-US.json"), enUS, 0o644); err != nil {
		t.Fatalf("Failed to create en-US file: %v", err)
	}

	translator, err := New(WithFileSystemLoader(localeDir), WithDefaultLocale("en-US"))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	result := translator.TranslateWithArgs("message", "Alice")
	if result != "Hello %s %s" {
		t.Errorf("TranslateWithArgs() = %q, want %q", result, "Hello %s %s")
	}
}

func TestTranslator_NoDefaultLocale(t *testing.T) {
	tmpDir := t.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	enUS := []byte(`{"greeting":"Hello"}`)
	if err := os.WriteFile(filepath.Join(localeDir, "en-US.json"), enUS, 0o644); err != nil {
		t.Fatalf("Failed to create en-US file: %v", err)
	}

	translator, err := New(WithFileSystemLoader(localeDir))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	locale := translator.GetLocale()
	if locale == "" {
		t.Error("GetLocale() should return non-empty locale")
	}
}

func TestTranslator_LoadError(t *testing.T) {
	tmpDir := t.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	translator, err := New(WithFileSystemLoader(localeDir), WithDefaultLocale("en-US"))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	result := translator.Translate("greeting")
	if result != "greeting" {
		t.Errorf("Translate() with load error = %q, want %q", result, "greeting")
	}
	if translator.HasKey("greeting") {
		t.Error("HasKey() should return false when file can't be loaded")
	}
}

func TestTranslator_ParseError(t *testing.T) {
	tmpDir := t.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	invalidJSON := []byte(`{invalid json`)
	if err := os.WriteFile(filepath.Join(localeDir, "en-US.json"), invalidJSON, 0o644); err != nil {
		t.Fatalf("Failed to create en-US file: %v", err)
	}

	translator, err := New(WithFileSystemLoader(localeDir), WithDefaultLocale("en-US"))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	result := translator.Translate("greeting")
	if result != "greeting" {
		t.Errorf("Translate() with parse error = %q, want %q", result, "greeting")
	}
	if translator.HasKey("greeting") {
		t.Error("HasKey() should return false when file can't be parsed")
	}
}

func TestWithRegisteredParser_JSON(t *testing.T) {
	tmpDir := t.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	enUS := []byte(`{"greeting":"Hello"}`)
	if err := os.WriteFile(filepath.Join(localeDir, "en-US.json"), enUS, 0o644); err != nil {
		t.Fatalf("Failed to create en-US file: %v", err)
	}

	translator, err := New(WithFileSystemLoader(localeDir), WithRegisteredParser(".json"), WithDefaultLocale("en-US"))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	result := translator.Translate("greeting")
	if result != "Hello" {
		t.Errorf("Translate() = %q, want %q", result, "Hello")
	}
}

func TestWithRegisteredParser_Unknown(t *testing.T) {
	tmpDir := t.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	testContent := []byte(`{"test": "value"}`)
	if err := os.WriteFile(filepath.Join(localeDir, "en-US.json"), testContent, 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	_, err := New(WithFileSystemLoader(localeDir), WithRegisteredParser(".unknown"), WithDefaultLocale("en-US"))
	if err == nil {
		t.Fatal("New() should fail for unregistered extension")
	}
	if !errors.Is(err, core.ErrUnknownFormat{}) {
		t.Errorf("error should wrap core.ErrUnknownFormat, got: %v", err)
	}
}

func TestWithRegisteredParser_CustomParser(t *testing.T) {
	tmpDir := t.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	customContent := []byte(`greeting=Bonjour`)
	if err := os.WriteFile(filepath.Join(localeDir, "fr-FR.custom"), customContent, 0o644); err != nil {
		t.Fatalf("Failed to create custom file: %v", err)
	}

	mockParser := &mockCustomParser{}
	if err := RegisterParser(".custom", mockParser); err != nil {
		t.Fatalf("RegisterParser() error = %v", err)
	}

	translator, err := New(
		WithLoader(NewFileSystemLoader(localeDir, WithExtension(".custom"))),
		WithRegisteredParser(".custom"),
		WithDefaultLocale("fr-FR"),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	result := translator.Translate("greeting")
	if result != "Bonjour" {
		t.Errorf("Translate() = %q, want %q", result, "Bonjour")
	}
}

type mockCustomParser struct{}

func (p *mockCustomParser) Parse(data []byte) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	line := string(data)
	for _, part := range splitLines(line) {
		if idx := indexByte(part, '='); idx >= 0 {
			result[part[:idx]] = part[idx+1:]
		}
	}
	return result, nil
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func indexByte(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}

// =============================================================================
// Merged from integration_test.go
// =============================================================================

func TestIntegration_FullWorkflowFilesystem(t *testing.T) {
	tmpDir := t.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	locales := map[string]string{
		"en-US.json": `{"app":{"title":"My Application","description":"A wonderful application"},"user":{"greeting":"Hello, %s!","farewell":"Goodbye, %s!","profile":{"title":"User Profile","settings":"Settings"}},"errors":{"validation":{"required":"This field is required","email":"Invalid email address","min_length":"Minimum length is %d characters"},"network":{"timeout":"Request timed out","offline":"You are offline"}}}`,
		"es-ES.json": `{"app":{"title":"Mi Aplicacion","description":"Una aplicacion maravillosa"},"user":{"greeting":"Hola, %s!","farewell":"Adios, %s!","profile":{"title":"Perfil de Usuario","settings":"Configuración"}},"errors":{"validation":{"required":"Este campo es obligatorio","email":"Direccion de correo invalida","min_length":"La longitud minima es %d caracteres"},"network":{"timeout":"Tiempo de espera agotado","offline":"Estas desconectado"}}}`,
		"es-MX.json": `{"user":{"greeting":"Que onda, %s?","farewell":"Nos vemos, %s!"}}`,
	}

	for filename, content := range locales {
		if err := os.WriteFile(filepath.Join(localeDir, filename), []byte(content), 0o644); err != nil {
			t.Fatalf("Failed to create %s: %v", filename, err)
		}
	}

	translator, err := New(WithFileSystemLoader(localeDir), WithDefaultLocale("en-US"))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if result := translator.Translate("app.title"); result != "My Application" {
		t.Errorf("Translate(app.title) = %q, want %q", result, "My Application")
	}

	translator.SetLocale("es-ES")
	if result := translator.Translate("app.title"); result != "Mi Aplicacion" {
		t.Errorf("After SetLocale(es-ES), Translate(app.title) = %q, want %q", result, "Mi Aplicacion")
	}

	translator.SetLocale("es-MX")
	if result := translator.Translate("app.title"); result != "Mi Aplicacion" {
		t.Errorf("Translate(app.title) in es-MX with fallback = %q, want %q", result, "Mi Aplicacion")
	}

	result := translator.Translate("nonexistent.key")
	if result != "nonexistent.key" {
		t.Errorf("Translate(nonexistent.key) = %q, want %q", result, "nonexistent.key")
	}
}

func TestIntegration_LocaleSwitchingAtRuntime(t *testing.T) {
	tmpDir := t.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	locales := map[string]string{
		"en-US.json": `{"message":"Hello"}`,
		"es-ES.json": `{"message":"Hola"}`,
		"fr-FR.json": `{"message":"Bonjour"}`,
		"de-DE.json": `{"message":"Guten Tag"}`,
	}
	for filename, content := range locales {
		if err := os.WriteFile(filepath.Join(localeDir, filename), []byte(content), 0o644); err != nil {
			t.Fatalf("Failed to create %s: %v", filename, err)
		}
	}

	translator, err := New(WithFileSystemLoader(localeDir), WithDefaultLocale("en-US"))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	expected := map[string]string{"en-US": "Hello", "es-ES": "Hola", "fr-FR": "Bonjour", "de-DE": "Guten Tag"}
	for locale, expectedMsg := range expected {
		translator.SetLocale(locale)
		result := translator.Translate("message")
		if result != expectedMsg {
			t.Errorf("After SetLocale(%s), Translate(message) = %q, want %q", locale, result, expectedMsg)
		}
	}

	translator.SetLocale("en_US")
	if translator.GetLocale() != "en-US" {
		t.Errorf("SetLocale(en_US) should normalize to en-US, got %q", translator.GetLocale())
	}
}

func TestIntegration_FallbackChainResolution(t *testing.T) {
	tmpDir := t.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	locales := map[string]string{
		"en-US.json": `{"common":{"yes":"Yes","no":"No","save":"Save","cancel":"Cancel"},"validation":{"required":"Required","email":"Invalid email"}}`,
		"es-ES.json": `{"common":{"yes":"Si","no":"No","save":"Guardar","cancel":"Cancelar"},"validation":{"required":"Requerido"}}`,
		"es-MX.json": `{"common":{"yes":"Si","no":"No"}}`,
	}
	for filename, content := range locales {
		if err := os.WriteFile(filepath.Join(localeDir, filename), []byte(content), 0o644); err != nil {
			t.Fatalf("Failed to create %s: %v", filename, err)
		}
	}

	translator, err := New(WithFileSystemLoader(localeDir), WithDefaultLocale("en-US"))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	translator.SetLocale("es-MX")

	tests := []struct {
		key      string
		expected string
	}{
		{"common.yes", "Si"},
		{"common.save", "Guardar"},
		{"validation.required", "Requerido"},
		{"validation.email", "Invalid email"},
	}

	for _, tt := range tests {
		result := translator.Translate(tt.key)
		if result != tt.expected {
			t.Errorf("Translate(%s) = %q, want %q", tt.key, result, tt.expected)
		}
	}
}

// =============================================================================
// Merged from security_test.go
// =============================================================================

func TestSecurity_PathTraversalPrevention(t *testing.T) {
	tmpDir := t.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	validLocale := filepath.Join(localeDir, "en-US.json")
	if err := os.WriteFile(validLocale, []byte(`{"test":"value"}`), 0o644); err != nil {
		t.Fatalf("Failed to create locale file: %v", err)
	}

	translator, err := New(WithFileSystemLoader(localeDir), WithDefaultLocale("en-US"))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	pathTraversalAttempts := []string{
		"../sensitive/secrets", "../../sensitive/secrets",
		"../../../etc/passwd", "..\\sensitive\\secrets",
		"/etc/passwd", "/absolute/path/to/secrets",
		"~/secrets", "~root/.ssh/id_rsa",
	}

	for _, maliciousLocale := range pathTraversalAttempts {
		translator.SetLocale(maliciousLocale)
		result := translator.Translate("secret")
		if result != "secret" {
			t.Errorf("Path traversal with locale %q should not succeed, got: %q", maliciousLocale, result)
		}
	}

	translator.SetLocale("en-US")
	result := translator.Translate("test")
	if result != "value" {
		t.Errorf("Valid locale should still work after traversal attempts, got: %q", result)
	}
}

func TestSecurity_ControlCharacterInjection(t *testing.T) {
	tmpDir := t.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	maliciousContent := []byte(`{"null_byte":"Hello\u0000World","ansi_escape":"\u001b[31mRed Text\u001b[0m","bidi_override":"Hello\u202EWorld","del_char":"Text\u007FMore","newline":"Hello\nWorld","tab":"Hello\tWorld"}`)
	if err := os.WriteFile(filepath.Join(localeDir, "en-US.json"), maliciousContent, 0o644); err != nil {
		t.Fatalf("Failed to create locale file: %v", err)
	}

	translator, err := New(WithFileSystemLoader(localeDir), WithDefaultLocale("en-US"))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	tests := []struct {
		key              string
		shouldContain    string
		shouldNotContain []string
	}{
		{key: "null_byte", shouldContain: "HelloWorld", shouldNotContain: []string{"\x00"}},
		{key: "ansi_escape", shouldContain: "Red Text", shouldNotContain: []string{"\x1b[31m", "\x1b[0m"}},
		{key: "bidi_override", shouldContain: "HelloWorld", shouldNotContain: []string{"\u202E"}},
		{key: "del_char", shouldContain: "TextMore", shouldNotContain: []string{"\x7F"}},
		{key: "newline", shouldContain: "Hello\nWorld"},
		{key: "tab", shouldContain: "Hello\tWorld"},
	}

	for _, tt := range tests {
		result := translator.Translate(tt.key)
		if !strings.Contains(result, tt.shouldContain) {
			t.Errorf("Translate(%s) should contain %q, got: %q", tt.key, tt.shouldContain, result)
		}
		for _, badStr := range tt.shouldNotContain {
			if strings.Contains(result, badStr) {
				t.Errorf("Translate(%s) should not contain control char %q, got: %q", tt.key, badStr, result)
			}
		}
	}
}

func TestSecurity_FormatStringInjection(t *testing.T) {
	tmpDir := t.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	content := []byte(`{"dangerous_n":"User %n logged in","valid_format":"Hello %s!","no_format":"Plain text message"}`)
	if err := os.WriteFile(filepath.Join(localeDir, "en-US.json"), content, 0o644); err != nil {
		t.Fatalf("Failed to create locale file: %v", err)
	}

	translator, err := New(WithFileSystemLoader(localeDir), WithDefaultLocale("en-US"))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	result := translator.TranslateWithArgs("dangerous_n", "Alice")
	if result != "User %n logged in" {
		t.Errorf("Format string with %%n should not be processed, got: %q", result)
	}

	result = translator.TranslateWithArgs("valid_format", "Bob")
	if result != "Hello Bob!" {
		t.Errorf("Valid format string should work, got: %q", result)
	}

	result = translator.TranslateWithArgs("no_format", "ignored")
	if result != "Plain text message" {
		t.Errorf("Plain text should work, got: %q", result)
	}
}

func TestSecurity_DoSPrevention(t *testing.T) {
	tmpDir := t.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(localeDir, "en-US.json"), []byte(`{"short":"value"}`), 0o644); err != nil {
		t.Fatalf("Failed to create locale file: %v", err)
	}

	translator, err := New(WithFileSystemLoader(localeDir), WithDefaultLocale("en-US"))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	longKey := strings.Repeat("a", 300)
	result := translator.Translate(longKey)
	if result != longKey {
		t.Errorf("Long key should be returned as-is, got length: %d", len(result))
	}

	longOutput := strings.Repeat("x", 15000)
	sanitized := core.SanitizeOutput(longOutput)
	if len(sanitized) > core.MaxOutputLength {
		t.Errorf("core.SanitizeOutput should truncate to %d chars, got: %d", core.MaxOutputLength, len(sanitized))
	}
}

func TestSecurity_CombinedAttackVectors(t *testing.T) {
	tmpDir := t.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	content := []byte(`{"combined":"\u202E\u001b[31m%n\u0000\u007Fattack\u202C","nested":{"deep":{"path":"\u202EInjection"}}}`)
	if err := os.WriteFile(filepath.Join(localeDir, "en-US.json"), content, 0o644); err != nil {
		t.Fatalf("Failed to create locale file: %v", err)
	}

	translator, err := New(WithFileSystemLoader(localeDir), WithDefaultLocale("en-US"))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	result := translator.Translate("combined")
	dangerousPatterns := []string{"\u202E", "\u202C", "\x1b", "\x00", "\x7F"}
	for _, pattern := range dangerousPatterns {
		if strings.Contains(result, pattern) {
			t.Errorf("Translate(combined) contains dangerous pattern %q, got: %q", pattern, result)
		}
	}
	if !strings.Contains(result, "attack") {
		t.Errorf("Translate(combined) lost content, got: %q", result)
	}

	result = translator.Translate("nested.deep.path")
	if !strings.Contains(result, "Injection") {
		t.Errorf("Nested translation failed, got: %q", result)
	}
	if strings.Contains(result, "\u202E") {
		t.Errorf("Nested translation contains BiDi, got: %q", result)
	}
}

// TestFormatCount tests the formatCount helper for various types.
func TestFormatCount(t *testing.T) {
	tests := []struct {
		name  string
		count interface{}
		want  string
	}{
		{"int", 42, "42"},
		{"int64", int64(100), "100"},
		{"float64 integer", float64(5), "5"},
		{"float64 decimal", 3.14, "3.14"},
		{"string", "7", "7"},
		{"bool", true, "true"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatCount(tt.count)
			if got != tt.want {
				t.Errorf("formatCount(%v) = %q, want %q", tt.count, got, tt.want)
			}
		})
	}
}

// =============================================================================
// New fuzz targets (TG4)
// =============================================================================

func FuzzTranslate(f *testing.F) {
	tmpDir := f.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		f.Fatalf("Failed to create test directory: %v", err)
	}
	enUS := []byte(`{"greeting":"Hello","user":{"profile":{"title":"Profile"}},"welcome":"Welcome, %s!"}`)
	if err := os.WriteFile(filepath.Join(localeDir, "en-US.json"), enUS, 0o644); err != nil {
		f.Fatalf("Failed to create en-US file: %v", err)
	}

	translator, err := New(WithFileSystemLoader(localeDir), WithDefaultLocale("en-US"))
	if err != nil {
		f.Fatalf("New() error = %v", err)
	}

	f.Add("greeting")
	f.Add("")
	f.Add("user.profile.title")
	f.Add("nonexistent.key")
	f.Add(strings.Repeat("a", 300))

	f.Fuzz(func(t *testing.T, key string) {
		_ = translator.Translate(key)
	})
}

func FuzzTranslateWithArgs(f *testing.F) {
	tmpDir := f.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		f.Fatalf("Failed to create test directory: %v", err)
	}
	enUS := []byte(`{"welcome":"Welcome, %s!","count":"Count: %d"}`)
	if err := os.WriteFile(filepath.Join(localeDir, "en-US.json"), enUS, 0o644); err != nil {
		f.Fatalf("Failed to create en-US file: %v", err)
	}

	translator, err := New(WithFileSystemLoader(localeDir), WithDefaultLocale("en-US"))
	if err != nil {
		f.Fatalf("New() error = %v", err)
	}

	f.Add("welcome", "Alice")
	f.Add("count", "5")
	f.Add("", "test")
	f.Add("nonexistent", "arg")
	f.Add(strings.Repeat("a", 300), "value")

	f.Fuzz(func(t *testing.T, key, arg string) {
		_ = translator.TranslateWithArgs(key, arg)
	})
}

// =============================================================================
// Distributed from benchmark_test.go
// =============================================================================

func setupBenchmarkLocales(b *testing.B) string {
	b.Helper()
	tmpDir := b.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		b.Fatalf("Failed to create test directory: %v", err)
	}

	locales := map[string]string{
		"en-US.json": `{"greeting":"Hello","farewell":"Goodbye","welcome":"Welcome, %s!","user":{"profile":{"title":"User Profile","settings":"Settings","privacy":"Privacy Settings"},"messages":{"inbox":"Inbox","sent":"Sent Messages","count":"You have %d messages"}},"errors":{"validation":{"required":"This field is required","email":"Invalid email address","min_length":"Minimum length is %d characters"},"network":{"timeout":"Request timed out","offline":"You are offline"}}}`,
		"es-ES.json": `{"greeting":"Hola","farewell":"Adios","welcome":"Bienvenido, %s!","user":{"profile":{"title":"Perfil de Usuario","settings":"Configuración"},"messages":{"inbox":"Bandeja de entrada","count":"Tienes %d mensajes"}},"errors":{"validation":{"required":"Este campo es obligatorio","email":"Direccion de correo invalida"}}}`,
		"es-MX.json": `{"greeting":"Hola","welcome":"Bienvenido, %s!"}`,
	}

	for filename, content := range locales {
		if err := os.WriteFile(filepath.Join(localeDir, filename), []byte(content), 0o644); err != nil {
			b.Fatalf("Failed to create %s: %v", filename, err)
		}
	}

	return localeDir
}

func BenchmarkTranslate_Cached(b *testing.B) {
	localeDir := setupBenchmarkLocales(b)
	translator, err := New(WithFileSystemLoader(localeDir), WithDefaultLocale("en-US"))
	if err != nil {
		b.Fatalf("New() error = %v", err)
	}
	_ = translator.Translate("greeting")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = translator.Translate("greeting")
	}
}

func BenchmarkTranslate_NestedKey(b *testing.B) {
	localeDir := setupBenchmarkLocales(b)
	translator, err := New(WithFileSystemLoader(localeDir), WithDefaultLocale("en-US"))
	if err != nil {
		b.Fatalf("New() error = %v", err)
	}
	_ = translator.Translate("user.profile.title")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = translator.Translate("user.profile.title")
	}
}

func BenchmarkTranslate_FallbackChain(b *testing.B) {
	localeDir := setupBenchmarkLocales(b)
	translator, err := New(WithFileSystemLoader(localeDir), WithDefaultLocale("en-US"))
	if err != nil {
		b.Fatalf("New() error = %v", err)
	}
	translator.SetLocale("es-MX")
	_ = translator.Translate("user.profile.settings")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = translator.Translate("user.profile.settings")
	}
}

func BenchmarkTranslate_MissingKey(b *testing.B) {
	localeDir := setupBenchmarkLocales(b)
	translator, err := New(WithFileSystemLoader(localeDir), WithDefaultLocale("en-US"))
	if err != nil {
		b.Fatalf("New() error = %v", err)
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = translator.Translate("nonexistent.key")
	}
}

func BenchmarkTranslateWithArgs_OneArg(b *testing.B) {
	localeDir := setupBenchmarkLocales(b)
	translator, err := New(WithFileSystemLoader(localeDir), WithDefaultLocale("en-US"))
	if err != nil {
		b.Fatalf("New() error = %v", err)
	}
	_ = translator.TranslateWithArgs("welcome", "Alice")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = translator.TranslateWithArgs("welcome", "Alice")
	}
}

func BenchmarkHasKey(b *testing.B) {
	localeDir := setupBenchmarkLocales(b)
	translator, err := New(WithFileSystemLoader(localeDir), WithDefaultLocale("en-US"))
	if err != nil {
		b.Fatalf("New() error = %v", err)
	}
	_ = translator.HasKey("greeting")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = translator.HasKey("greeting")
	}
}

func BenchmarkSetLocale(b *testing.B) {
	localeDir := setupBenchmarkLocales(b)
	translator, err := New(WithFileSystemLoader(localeDir), WithDefaultLocale("en-US"))
	if err != nil {
		b.Fatalf("New() error = %v", err)
	}
	locales := []string{"en-US", "es-ES", "es-MX"}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		translator.SetLocale(locales[i%len(locales)])
	}
}

func BenchmarkGetLocale(b *testing.B) {
	localeDir := setupBenchmarkLocales(b)
	translator, err := New(WithFileSystemLoader(localeDir), WithDefaultLocale("en-US"))
	if err != nil {
		b.Fatalf("New() error = %v", err)
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = translator.GetLocale()
	}
}

func BenchmarkConcurrentTranslate(b *testing.B) {
	localeDir := setupBenchmarkLocales(b)
	translator, err := New(WithFileSystemLoader(localeDir), WithDefaultLocale("en-US"))
	if err != nil {
		b.Fatalf("New() error = %v", err)
	}
	_ = translator.Translate("greeting")
	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = translator.Translate("greeting")
		}
	})
}

func BenchmarkLoadAndParse(b *testing.B) {
	localeDir := setupBenchmarkLocales(b)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		translator, err := New(WithFileSystemLoader(localeDir), WithDefaultLocale("en-US"))
		if err != nil {
			b.Fatalf("New() error = %v", err)
		}
		_ = translator.Translate("greeting")
	}
}

// =============================================================================
// Merged from translator_plural_test.go
// =============================================================================

// createPluralTestLocales sets up test locale files with plural/gender translations.
func createPluralTestLocales(t *testing.T) (string, *Translator) {
	t.Helper()

	tmpDir := t.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	enUS := []byte(`{
		"items": {
			"one": "# item",
			"other": "# items"
		},
		"messages": {
			"one": "You have # message in %s",
			"other": "You have # messages in %s"
		},
		"welcome": {
			"masculine": "He logged in",
			"feminine": "She logged in",
			"other": "They logged in"
		},
		"icu_plural": "{count, plural, one {# item} other {# items}}",
		"icu_select": "{gender, select, male {He} female {She} other {They}}"
	}`)
	if err := os.WriteFile(filepath.Join(localeDir, "en-US.json"), enUS, 0o644); err != nil {
		t.Fatalf("Failed to create en-US file: %v", err)
	}

	esES := []byte(`{
		"items": {
			"one": "# elemento",
			"other": "# elementos"
		}
	}`)
	if err := os.WriteFile(filepath.Join(localeDir, "es-ES.json"), esES, 0o644); err != nil {
		t.Fatalf("Failed to create es-ES file: %v", err)
	}

	// es-MX partial -- missing items, falls back to es-ES then en-US
	esMX := []byte(`{
		"greeting": "Hola"
	}`)
	if err := os.WriteFile(filepath.Join(localeDir, "es-MX.json"), esMX, 0o644); err != nil {
		t.Fatalf("Failed to create es-MX file: %v", err)
	}

	translator, err := New(
		WithFileSystemLoader(localeDir),
		WithDefaultLocale("en-US"),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	return localeDir, translator
}

func TestTranslatePlural_ResolvesOneCategory(t *testing.T) {
	_, translator := createPluralTestLocales(t)

	got := translator.TranslatePlural("items", 1)
	want := "1 item"
	if got != want {
		t.Errorf("TranslatePlural(items, 1) = %q, want %q", got, want)
	}
}

func TestTranslatePlural_FallsBackToOther(t *testing.T) {
	_, translator := createPluralTestLocales(t)

	got := translator.TranslatePlural("items", 5)
	want := "5 items"
	if got != want {
		t.Errorf("TranslatePlural(items, 5) = %q, want %q", got, want)
	}
}

func TestTranslatePlural_ReplacesHashPlaceholder(t *testing.T) {
	_, translator := createPluralTestLocales(t)

	got := translator.TranslatePlural("items", 42)
	want := "42 items"
	if got != want {
		t.Errorf("TranslatePlural(items, 42) = %q, want %q", got, want)
	}
}

func TestTranslatePlural_FallbackThroughLocaleChain(t *testing.T) {
	_, translator := createPluralTestLocales(t)

	translator.SetLocale("es-MX")
	got := translator.TranslatePlural("items", 1)
	// es-MX has no "items", falls back to es-ES
	want := "1 elemento"
	if got != want {
		t.Errorf("TranslatePlural(items, 1) with es-MX = %q, want %q", got, want)
	}
}

func TestTranslatePluralWithArgs_AppliesSprintf(t *testing.T) {
	_, translator := createPluralTestLocales(t)

	got := translator.TranslatePluralWithArgs("messages", 1, "inbox")
	want := "You have 1 message in inbox"
	if got != want {
		t.Errorf("TranslatePluralWithArgs(messages, 1, inbox) = %q, want %q", got, want)
	}
}

func TestTranslateGender_ResolvesGenderKey(t *testing.T) {
	_, translator := createPluralTestLocales(t)

	got := translator.TranslateGender("welcome", core.Masculine)
	want := "He logged in"
	if got != want {
		t.Errorf("TranslateGender(welcome, core.Masculine) = %q, want %q", got, want)
	}
}

func TestTranslateGender_FallsBackToOther(t *testing.T) {
	_, translator := createPluralTestLocales(t)

	got := translator.TranslateGender("welcome", core.Neuter)
	want := "They logged in"
	if got != want {
		t.Errorf("TranslateGender(welcome, core.Neuter) = %q, want %q", got, want)
	}
}

func TestWithPluralResolver_NilReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	testContent := []byte(`{"test": "value"}`)
	if err := os.WriteFile(filepath.Join(localeDir, "en-US.json"), testContent, 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	_, err := New(
		WithFileSystemLoader(localeDir),
		WithDefaultLocale("en-US"),
		WithPluralResolver(nil),
	)
	if err == nil {
		t.Error("WithPluralResolver(nil) should return error, got nil")
	}
}

func TestNew_DefaultsToDefaultPluralResolver(t *testing.T) {
	tmpDir := t.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	testContent := []byte(`{"test": "value"}`)
	if err := os.WriteFile(filepath.Join(localeDir, "en-US.json"), testContent, 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	translator, err := New(
		WithFileSystemLoader(localeDir),
		WithDefaultLocale("en-US"),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if translator.pluralResolver == nil {
		t.Error("New() should default to DefaultPluralResolver, got nil")
	}
}

// Coverage gap: TranslatePlural missing key returns key
func TestTranslatePlural_MissingKeyReturnsKey(t *testing.T) {
	_, translator := createPluralTestLocales(t)

	got := translator.TranslatePlural("nonexistent", 1)
	if got != "nonexistent" {
		t.Errorf("TranslatePlural missing key = %q, want %q", got, "nonexistent")
	}
}

// Coverage gap: TranslatePluralWithArgs missing key returns key
func TestTranslatePluralWithArgs_MissingKeyReturnsKey(t *testing.T) {
	_, translator := createPluralTestLocales(t)

	got := translator.TranslatePluralWithArgs("nonexistent", 1, "arg")
	if got != "nonexistent" {
		t.Errorf("TranslatePluralWithArgs missing key = %q, want %q", got, "nonexistent")
	}
}

// Coverage gap: TranslatePluralWithArgs uses the other branch when category not found
func TestTranslatePluralWithArgs_FallsBackToOther(t *testing.T) {
	_, translator := createPluralTestLocales(t)

	got := translator.TranslatePluralWithArgs("messages", 5, "inbox")
	want := "You have 5 messages in inbox"
	if got != want {
		t.Errorf("TranslatePluralWithArgs other = %q, want %q", got, want)
	}
}

// Coverage gap: TranslatePluralWithArgs with format string validation error
func TestTranslatePluralWithArgs_FormatValidationError(t *testing.T) {
	tmpDir := t.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	// Template has %s but we give no format args
	enUS := []byte(`{
		"msg": {
			"one": "# item for %s",
			"other": "# items for %s"
		}
	}`)
	if err := os.WriteFile(filepath.Join(localeDir, "en-US.json"), enUS, 0o644); err != nil {
		t.Fatalf("Failed to create en-US file: %v", err)
	}

	translator, err := New(WithFileSystemLoader(localeDir), WithDefaultLocale("en-US"))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Pass no format args: core.ValidateFormatString fails, returns sanitized template
	got := translator.TranslatePluralWithArgs("msg", 1)
	want := "1 item for %s"
	if got != want {
		t.Errorf("TranslatePluralWithArgs format error = %q, want %q", got, want)
	}

	// Also test the other-key path with format validation error
	got = translator.TranslatePluralWithArgs("msg", 5)
	want = "5 items for %s"
	if got != want {
		t.Errorf("TranslatePluralWithArgs format error other = %q, want %q", got, want)
	}
}

// Coverage gap: TranslateGender missing key returns key
func TestTranslateGender_MissingKeyReturnsKey(t *testing.T) {
	_, translator := createPluralTestLocales(t)

	got := translator.TranslateGender("nonexistent", core.Masculine)
	if got != "nonexistent" {
		t.Errorf("TranslateGender missing key = %q, want %q", got, "nonexistent")
	}
}

// Coverage gap: TranslatePlural through locale chain fallback to other-key
func TestTranslatePlural_FallbackToOtherKey(t *testing.T) {
	tmpDir := t.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	// Only "other" key exists, no "one"
	enUS := []byte(`{
		"counter": {
			"other": "# things"
		}
	}`)
	if err := os.WriteFile(filepath.Join(localeDir, "en-US.json"), enUS, 0o644); err != nil {
		t.Fatalf("Failed to create en-US file: %v", err)
	}

	translator, err := New(WithFileSystemLoader(localeDir), WithDefaultLocale("en-US"))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// count=1 resolves to "one" category, which doesn't exist, so falls back to "other"
	got := translator.TranslatePlural("counter", 1)
	want := "1 things"
	if got != want {
		t.Errorf("TranslatePlural fallback to other = %q, want %q", got, want)
	}
}

// Coverage gap: New with logger already set via option error
func TestNew_OptionErrorWithLoggerSet(t *testing.T) {
	mockLogger := newTestMockLogger()
	_, err := New(
		WithLogger(mockLogger),
		WithLoader(nil), // This will error
	)
	if err == nil {
		t.Error("New() should fail with nil loader option")
	}
}

// Coverage gap: HasKey with failing load
func TestTranslator_HasKey_LoadFails(t *testing.T) {
	mockLogger := newTestMockLogger()
	translator, err := New(
		WithLoader(&failingTestLoader{}),
		WithDefaultLocale("en-US"),
		WithLogger(mockLogger),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if translator.HasKey("any.key") {
		t.Error("HasKey should return false when load fails")
	}
}

// Coverage gap: TranslatePluralWithArgs falls back to other-key when category-key not found
func TestTranslatePluralWithArgs_FallsBackToOtherKey(t *testing.T) {
	tmpDir := t.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	// Only "other" key exists, category "one" does NOT
	enUS := []byte(`{
		"msg": {
			"other": "# items for %s"
		}
	}`)
	if err := os.WriteFile(filepath.Join(localeDir, "en-US.json"), enUS, 0o644); err != nil {
		t.Fatalf("Failed to create en-US file: %v", err)
	}

	translator, err := New(WithFileSystemLoader(localeDir), WithDefaultLocale("en-US"))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// count=1 resolves to "one" category, but only "other" exists
	got := translator.TranslatePluralWithArgs("msg", 1, "inbox")
	want := "1 items for inbox"
	if got != want {
		t.Errorf("TranslatePluralWithArgs other fallback = %q, want %q", got, want)
	}
}

// Coverage gap: TranslatePluralWithArgs other-key format validation error
func TestTranslatePluralWithArgs_OtherKeyFormatError(t *testing.T) {
	tmpDir := t.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	// "other" has %s but we pass no format args
	enUS := []byte(`{
		"msg": {
			"other": "# items for %s"
		}
	}`)
	if err := os.WriteFile(filepath.Join(localeDir, "en-US.json"), enUS, 0o644); err != nil {
		t.Fatalf("Failed to create en-US file: %v", err)
	}

	translator, err := New(WithFileSystemLoader(localeDir), WithDefaultLocale("en-US"))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// count=1, category="one" not found, falls to "other", format validation fails (expects 1 arg, got 0)
	got := translator.TranslatePluralWithArgs("msg", 1)
	want := "1 items for %s"
	if got != want {
		t.Errorf("TranslatePluralWithArgs other format error = %q, want %q", got, want)
	}
}

// =============================================================================
// Merged from plural_gap_test.go: translator-level plural/gender tests
// =============================================================================

// TestEndToEnd_RussianPluralThroughTranslator verifies the full workflow:
// New() with default resolver -> SetLocale("ru-RU") -> TranslatePlural with Russian rules.
func TestEndToEnd_RussianPluralThroughTranslator(t *testing.T) {
	tmpDir := t.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	ruRU := []byte(`{
		"items": {
			"one": "# \u044d\u043b\u0435\u043c\u0435\u043d\u0442",
			"few": "# \u044d\u043b\u0435\u043c\u0435\u043d\u0442\u0430",
			"many": "# \u044d\u043b\u0435\u043c\u0435\u043d\u0442\u043e\u0432",
			"other": "# \u044d\u043b\u0435\u043c\u0435\u043d\u0442\u043e\u0432"
		}
	}`)
	if err := os.WriteFile(filepath.Join(localeDir, "ru-RU.json"), ruRU, 0o644); err != nil {
		t.Fatalf("Failed to create ru-RU file: %v", err)
	}

	enUS := []byte(`{"items": {"one": "# item", "other": "# items"}}`)
	if err := os.WriteFile(filepath.Join(localeDir, "en-US.json"), enUS, 0o644); err != nil {
		t.Fatalf("Failed to create en-US file: %v", err)
	}

	translator, err := New(
		WithFileSystemLoader(localeDir),
		WithDefaultLocale("en-US"),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	translator.SetLocale("ru-RU")

	tests := []struct {
		name  string
		count int
		want  string
	}{
		{"1 -> one", 1, "1 \u044d\u043b\u0435\u043c\u0435\u043d\u0442"},
		{"2 -> few", 2, "2 \u044d\u043b\u0435\u043c\u0435\u043d\u0442\u0430"},
		{"5 -> many", 5, "5 \u044d\u043b\u0435\u043c\u0435\u043d\u0442\u043e\u0432"},
		{"21 -> one", 21, "21 \u044d\u043b\u0435\u043c\u0435\u043d\u0442"},
		{"11 -> many", 11, "11 \u044d\u043b\u0435\u043c\u0435\u043d\u0442\u043e\u0432"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := translator.TranslatePlural("items", tt.count)
			if got != tt.want {
				t.Errorf("TranslatePlural(items, %d) = %q, want %q", tt.count, got, tt.want)
			}
		})
	}
}

// TestTranslatePluralWithArgs_CombinedHashAndSprintf verifies # replacement
// and fmt.Sprintf args work together in a single call.
func TestTranslatePluralWithArgs_CombinedHashAndSprintf(t *testing.T) {
	tmpDir := t.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	enUS := []byte(`{
		"cart": {
			"one": "# item in %s",
			"other": "# items in %s"
		}
	}`)
	if err := os.WriteFile(filepath.Join(localeDir, "en-US.json"), enUS, 0o644); err != nil {
		t.Fatalf("Failed to create en-US file: %v", err)
	}

	translator, err := New(
		WithFileSystemLoader(localeDir),
		WithDefaultLocale("en-US"),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	got := translator.TranslatePluralWithArgs("cart", 3, "your cart")
	want := "3 items in your cart"
	if got != want {
		t.Errorf("TranslatePluralWithArgs = %q, want %q", got, want)
	}
}

// TestTranslateWithMessage_PluralICU_FullTranslator verifies TranslateWithMessage
// with a plural ICU expression works through the full Translator lifecycle.
func TestTranslateWithMessage_PluralICU_FullTranslator(t *testing.T) {
	tmpDir := t.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	enUS := []byte(`{
		"msg": "{count, plural, one {You have # notification} other {You have # notifications}}"
	}`)
	if err := os.WriteFile(filepath.Join(localeDir, "en-US.json"), enUS, 0o644); err != nil {
		t.Fatalf("Failed to create en-US file: %v", err)
	}

	translator, err := New(
		WithFileSystemLoader(localeDir),
		WithDefaultLocale("en-US"),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	got := translator.TranslateWithMessage("msg", map[string]interface{}{"count": 1})
	want := "You have 1 notification"
	if got != want {
		t.Errorf("TranslateWithMessage(count=1) = %q, want %q", got, want)
	}

	got = translator.TranslateWithMessage("msg", map[string]interface{}{"count": 10})
	want = "You have 10 notifications"
	if got != want {
		t.Errorf("TranslateWithMessage(count=10) = %q, want %q", got, want)
	}
}

// TestTranslatePlural_ConcurrentAccess verifies thread safety of TranslatePlural.
func TestTranslatePlural_ConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	enUS := []byte(`{"items": {"one": "# item", "other": "# items"}}`)
	if err := os.WriteFile(filepath.Join(localeDir, "en-US.json"), enUS, 0o644); err != nil {
		t.Fatalf("Failed to create en-US file: %v", err)
	}

	translator, err := New(
		WithFileSystemLoader(localeDir),
		WithDefaultLocale("en-US"),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			_ = translator.TranslatePlural("items", n)
		}(i)
	}
	wg.Wait()
}

// TestWithPluralResolver_CustomOverridesDefault verifies a custom core.PluralResolver
// overrides the default.
func TestWithPluralResolver_CustomOverridesDefault(t *testing.T) {
	tmpDir := t.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	enUS := []byte(`{"items": {"one": "# item", "other": "# items"}}`)
	if err := os.WriteFile(filepath.Join(localeDir, "en-US.json"), enUS, 0o644); err != nil {
		t.Fatalf("Failed to create en-US file: %v", err)
	}

	// Custom resolver that always returns "one"
	custom := &alwaysOnePluralResolver{}
	translator, err := New(
		WithFileSystemLoader(localeDir),
		WithDefaultLocale("en-US"),
		WithPluralResolver(custom),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Even with count=5, the custom resolver returns core.One
	got := translator.TranslatePlural("items", 5)
	want := "5 item"
	if got != want {
		t.Errorf("TranslatePlural with custom resolver = %q, want %q", got, want)
	}
}

// alwaysOnePluralResolver is a mock resolver that always returns core.One.
type alwaysOnePluralResolver struct{}

func (r *alwaysOnePluralResolver) Resolve(_ string, _ interface{}) core.PluralCategory {
	return core.One
}

// TestTranslateGender_FallbackThroughLocaleChain verifies gender translation
// falls back through the locale chain.
func TestTranslateGender_FallbackThroughLocaleChain(t *testing.T) {
	tmpDir := t.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	enUS := []byte(`{
		"welcome": {
			"masculine": "He joined",
			"feminine": "She joined",
			"other": "They joined"
		}
	}`)
	if err := os.WriteFile(filepath.Join(localeDir, "en-US.json"), enUS, 0o644); err != nil {
		t.Fatalf("Failed to create en-US file: %v", err)
	}

	// es-ES doesn't have the welcome key
	esES := []byte(`{"greeting": "Hola"}`)
	if err := os.WriteFile(filepath.Join(localeDir, "es-ES.json"), esES, 0o644); err != nil {
		t.Fatalf("Failed to create es-ES file: %v", err)
	}

	translator, err := New(
		WithFileSystemLoader(localeDir),
		WithDefaultLocale("en-US"),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	translator.SetLocale("es-ES")
	got := translator.TranslateGender("welcome", core.Feminine)
	want := "She joined"
	if got != want {
		t.Errorf("TranslateGender with fallback = %q, want %q", got, want)
	}
}

// =============================================================================
// Spec 018: Translator Hot-Path Benchmarks (Task Group 1)
// =============================================================================

// setupBenchmarkLocalesExtended writes en-US, es-ES, es-MX, and ru-RU locale
// files containing plural keys, gender keys, ICU MessageFormat values, and
// multi-arg format strings for comprehensive translator benchmarks.
func setupBenchmarkLocalesExtended(b *testing.B) string {
	b.Helper()
	tmpDir := b.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		b.Fatalf("Failed to create test directory: %v", err)
	}

	enUS := []byte(`{
		"greeting":"Hello","farewell":"Goodbye","welcome":"Welcome, %s!",
		"multi_args":"%s scored %d points in %s",
		"items":{"one":"# item","other":"# items"},
		"messages":{"one":"You have # message in %s","other":"You have # messages in %s"},
		"welcome_gender":{"masculine":"He logged in","feminine":"She logged in","other":"They logged in"},
		"icu_plural":"{count, plural, one {# item} other {# items}}",
		"icu_select":"{gender, select, male {He} female {She} other {They}}",
		"user":{"profile":{"title":"User Profile","settings":"Settings"}}
	}`)
	if err := os.WriteFile(filepath.Join(localeDir, "en-US.json"), enUS, 0o644); err != nil {
		b.Fatalf("Failed to create en-US file: %v", err)
	}

	esES := []byte(`{"greeting":"Hola","items":{"one":"# elemento","other":"# elementos"}}`)
	if err := os.WriteFile(filepath.Join(localeDir, "es-ES.json"), esES, 0o644); err != nil {
		b.Fatalf("Failed to create es-ES file: %v", err)
	}

	esMX := []byte(`{"greeting":"Hola"}`)
	if err := os.WriteFile(filepath.Join(localeDir, "es-MX.json"), esMX, 0o644); err != nil {
		b.Fatalf("Failed to create es-MX file: %v", err)
	}

	ruRU := []byte(`{
		"items":{
			"one":"# \u044d\u043b\u0435\u043c\u0435\u043d\u0442",
			"few":"# \u044d\u043b\u0435\u043c\u0435\u043d\u0442\u0430",
			"many":"# \u044d\u043b\u0435\u043c\u0435\u043d\u0442\u043e\u0432",
			"other":"# \u044d\u043b\u0435\u043c\u0435\u043d\u0442\u043e\u0432"
		}
	}`)
	if err := os.WriteFile(filepath.Join(localeDir, "ru-RU.json"), ruRU, 0o644); err != nil {
		b.Fatalf("Failed to create ru-RU file: %v", err)
	}

	return localeDir
}

// Baseline: ~200-500 ns/op, 0-2 allocs/op
func BenchmarkTranslatePlural(b *testing.B) {
	localeDir := setupBenchmarkLocalesExtended(b)
	translator, err := New(WithFileSystemLoader(localeDir), WithDefaultLocale("en-US"))
	if err != nil {
		b.Fatalf("New() error = %v", err)
	}
	// Warm cache
	_ = translator.TranslatePlural("items", 1)
	_ = translator.TranslatePlural("items", 5)

	b.Run("one", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = translator.TranslatePlural("items", 1)
		}
	})

	b.Run("other", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = translator.TranslatePlural("items", 5)
		}
	})

	b.Run("few_russian", func(b *testing.B) {
		translator.SetLocale("ru-RU")
		_ = translator.TranslatePlural("items", 2)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = translator.TranslatePlural("items", 2)
		}
		translator.SetLocale("en-US")
	})
}

// Baseline: ~300-700 ns/op, 2-4 allocs/op
func BenchmarkTranslatePluralWithArgs(b *testing.B) {
	localeDir := setupBenchmarkLocalesExtended(b)
	translator, err := New(WithFileSystemLoader(localeDir), WithDefaultLocale("en-US"))
	if err != nil {
		b.Fatalf("New() error = %v", err)
	}
	_ = translator.TranslatePluralWithArgs("messages", 5, "inbox")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = translator.TranslatePluralWithArgs("messages", 5, "inbox")
	}
}

// Baseline: ~200-400 ns/op, 0-2 allocs/op
func BenchmarkTranslateGender(b *testing.B) {
	localeDir := setupBenchmarkLocalesExtended(b)
	translator, err := New(WithFileSystemLoader(localeDir), WithDefaultLocale("en-US"))
	if err != nil {
		b.Fatalf("New() error = %v", err)
	}
	_ = translator.TranslateGender("welcome_gender", core.Masculine)

	b.Run("masculine", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = translator.TranslateGender("welcome_gender", core.Masculine)
		}
	})

	b.Run("feminine", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = translator.TranslateGender("welcome_gender", core.Feminine)
		}
	})

	b.Run("other", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = translator.TranslateGender("welcome_gender", core.GenderOther)
		}
	})
}

// Baseline: ~400-1000 ns/op, 3-8 allocs/op
func BenchmarkTranslateWithMessage(b *testing.B) {
	localeDir := setupBenchmarkLocalesExtended(b)
	translator, err := New(WithFileSystemLoader(localeDir), WithDefaultLocale("en-US"))
	if err != nil {
		b.Fatalf("New() error = %v", err)
	}
	pluralArgs := map[string]interface{}{"count": 5}
	selectArgs := map[string]interface{}{"gender": "male"}
	_ = translator.TranslateWithMessage("icu_plural", pluralArgs)
	_ = translator.TranslateWithMessage("icu_select", selectArgs)

	b.Run("plural", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = translator.TranslateWithMessage("icu_plural", pluralArgs)
		}
	})

	b.Run("select", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = translator.TranslateWithMessage("icu_select", selectArgs)
		}
	})
}

// Baseline: ~300-600 ns/op, 3-5 allocs/op
func BenchmarkTranslateWithArgs_MultipleArgs(b *testing.B) {
	localeDir := setupBenchmarkLocalesExtended(b)
	translator, err := New(WithFileSystemLoader(localeDir), WithDefaultLocale("en-US"))
	if err != nil {
		b.Fatalf("New() error = %v", err)
	}
	_ = translator.TranslateWithArgs("multi_args", "Alice", 42, "level5")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = translator.TranslateWithArgs("multi_args", "Alice", 42, "level5")
	}
}

// Baseline: ~200-500 ns/op, 0-2 allocs/op
func BenchmarkTranslatePlural_Parallel(b *testing.B) {
	localeDir := setupBenchmarkLocalesExtended(b)
	translator, err := New(WithFileSystemLoader(localeDir), WithDefaultLocale("en-US"))
	if err != nil {
		b.Fatalf("New() error = %v", err)
	}
	_ = translator.TranslatePlural("items", 5)
	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = translator.TranslatePlural("items", 5)
		}
	})
}

// Baseline: ~400-1000 ns/op, 3-8 allocs/op
func BenchmarkTranslateWithMessage_Parallel(b *testing.B) {
	localeDir := setupBenchmarkLocalesExtended(b)
	translator, err := New(WithFileSystemLoader(localeDir), WithDefaultLocale("en-US"))
	if err != nil {
		b.Fatalf("New() error = %v", err)
	}
	args := map[string]interface{}{"count": 5}
	_ = translator.TranslateWithMessage("icu_plural", args)
	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = translator.TranslateWithMessage("icu_plural", args)
		}
	})
}

// Baseline: ~200-500 ns/op, 2-4 allocs/op
func BenchmarkTranslateWithArgs_Parallel(b *testing.B) {
	localeDir := setupBenchmarkLocalesExtended(b)
	translator, err := New(WithFileSystemLoader(localeDir), WithDefaultLocale("en-US"))
	if err != nil {
		b.Fatalf("New() error = %v", err)
	}
	_ = translator.TranslateWithArgs("welcome", "Alice")
	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = translator.TranslateWithArgs("welcome", "Alice")
		}
	})
}

// ---------------------------------------------------------------------------
// Translation cache integration tests
// ---------------------------------------------------------------------------

func TestWithCacheNilReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	enUS := []byte(`{"greeting":"Hello"}`)
	if err := os.WriteFile(filepath.Join(localeDir, "en-US.json"), enUS, 0o644); err != nil {
		t.Fatalf("Failed to create en-US file: %v", err)
	}

	_, err := New(
		WithFileSystemLoader(localeDir),
		WithDefaultLocale("en-US"),
		WithCache(nil),
	)
	if err == nil {
		t.Fatal("WithCache(nil) should return an error")
	}
	if !strings.Contains(err.Error(), "cache cannot be nil") {
		t.Errorf("error = %q, want it to contain 'cache cannot be nil'", err.Error())
	}
}

func TestWithCacheValidCreatesTranslator(t *testing.T) {
	tmpDir := t.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	enUS := []byte(`{"greeting":"Hello"}`)
	if err := os.WriteFile(filepath.Join(localeDir, "en-US.json"), enUS, 0o644); err != nil {
		t.Fatalf("Failed to create en-US file: %v", err)
	}

	tr, err := New(
		WithFileSystemLoader(localeDir),
		WithDefaultLocale("en-US"),
		WithCache(NewMapCache()),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if tr == nil {
		t.Fatal("New() returned nil translator")
	}
}

func TestTranslateCacheHitOnSecondCall(t *testing.T) {
	tmpDir := t.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	enUS := []byte(`{"greeting":"Hello"}`)
	if err := os.WriteFile(filepath.Join(localeDir, "en-US.json"), enUS, 0o644); err != nil {
		t.Fatalf("Failed to create en-US file: %v", err)
	}

	cache := NewMapCache()
	tr, err := New(
		WithFileSystemLoader(localeDir),
		WithDefaultLocale("en-US"),
		WithCache(cache),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// First call: cache miss, populates cache.
	result1 := tr.Translate("greeting")
	if result1 != "Hello" {
		t.Fatalf("first Translate = %q, want %q", result1, "Hello")
	}

	// Second call: should hit cache.
	result2 := tr.Translate("greeting")
	if result2 != "Hello" {
		t.Errorf("second Translate = %q, want %q", result2, "Hello")
	}

	// Verify the cache contains the entry.
	val, ok := cache.Get("en-US:greeting")
	if !ok {
		t.Error("expected cache hit for 'en-US:greeting'")
	}
	if val != "Hello" {
		t.Errorf("cached value = %q, want %q", val, "Hello")
	}
}

func TestTranslateCacheDoesNotCacheMisses(t *testing.T) {
	tmpDir := t.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	enUS := []byte(`{"greeting":"Hello"}`)
	if err := os.WriteFile(filepath.Join(localeDir, "en-US.json"), enUS, 0o644); err != nil {
		t.Fatalf("Failed to create en-US file: %v", err)
	}

	cache := NewMapCache()
	tr, err := New(
		WithFileSystemLoader(localeDir),
		WithDefaultLocale("en-US"),
		WithCache(cache),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Missing key returns the key itself; should not be cached.
	result := tr.Translate("nonexistent.key")
	if result != "nonexistent.key" {
		t.Fatalf("Translate = %q, want %q", result, "nonexistent.key")
	}

	if _, ok := cache.Get("en-US:nonexistent.key"); ok {
		t.Error("cache should not store entries for missing keys")
	}
}

func TestTranslatePluralCacheWithDifferentCounts(t *testing.T) {
	tmpDir := t.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	enUS := []byte(`{"items":{"one":"# item","other":"# items"}}`)
	if err := os.WriteFile(filepath.Join(localeDir, "en-US.json"), enUS, 0o644); err != nil {
		t.Fatalf("Failed to create en-US file: %v", err)
	}

	cache := NewMapCache()
	tr, err := New(
		WithFileSystemLoader(localeDir),
		WithDefaultLocale("en-US"),
		WithCache(cache),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	r1 := tr.TranslatePlural("items", 1)
	if r1 != "1 item" {
		t.Errorf("TranslatePlural(1) = %q, want %q", r1, "1 item")
	}

	r5 := tr.TranslatePlural("items", 5)
	if r5 != "5 items" {
		t.Errorf("TranslatePlural(5) = %q, want %q", r5, "5 items")
	}

	// Verify distinct cache entries exist.
	if _, ok := cache.Get("en-US:items#1"); !ok {
		t.Error("expected cache hit for count=1")
	}
	if _, ok := cache.Get("en-US:items#5"); !ok {
		t.Error("expected cache hit for count=5")
	}
}

func TestTranslateGenderCacheWithDifferentGenders(t *testing.T) {
	tmpDir := t.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	enUS := []byte(`{"welcome":{"masculine":"He joined","feminine":"She joined","other":"They joined"}}`)
	if err := os.WriteFile(filepath.Join(localeDir, "en-US.json"), enUS, 0o644); err != nil {
		t.Fatalf("Failed to create en-US file: %v", err)
	}

	cache := NewMapCache()
	tr, err := New(
		WithFileSystemLoader(localeDir),
		WithDefaultLocale("en-US"),
		WithCache(cache),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	rm := tr.TranslateGender("welcome", core.Masculine)
	if rm != "He joined" {
		t.Errorf("TranslateGender(masculine) = %q, want %q", rm, "He joined")
	}

	rf := tr.TranslateGender("welcome", core.Feminine)
	if rf != "She joined" {
		t.Errorf("TranslateGender(feminine) = %q, want %q", rf, "She joined")
	}

	// Verify distinct cache entries.
	if _, ok := cache.Get("en-US:welcome@masculine"); !ok {
		t.Error("expected cache hit for masculine")
	}
	if _, ok := cache.Get("en-US:welcome@feminine"); !ok {
		t.Error("expected cache hit for feminine")
	}
}

func TestSetLocaleInvalidatesCache(t *testing.T) {
	tmpDir := t.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	enUS := []byte(`{"greeting":"Hello"}`)
	esES := []byte(`{"greeting":"Hola"}`)
	if err := os.WriteFile(filepath.Join(localeDir, "en-US.json"), enUS, 0o644); err != nil {
		t.Fatalf("Failed to create en-US file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(localeDir, "es-ES.json"), esES, 0o644); err != nil {
		t.Fatalf("Failed to create es-ES file: %v", err)
	}

	cache := NewMapCache()
	tr, err := New(
		WithFileSystemLoader(localeDir),
		WithDefaultLocale("en-US"),
		WithCache(cache),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Populate cache with en-US translation.
	r1 := tr.Translate("greeting")
	if r1 != "Hello" {
		t.Fatalf("Translate en-US = %q, want %q", r1, "Hello")
	}

	// Switch locale; cache should be invalidated.
	tr.SetLocale("es-ES")

	// The old en-US entry should be gone.
	if _, ok := cache.Get("en-US:greeting"); ok {
		t.Error("cache should be invalidated after SetLocale")
	}

	// New locale should produce the correct translation.
	r2 := tr.Translate("greeting")
	if r2 != "Hola" {
		t.Errorf("Translate es-ES = %q, want %q", r2, "Hola")
	}
}

func TestTranslatorWithoutCacheBehavesIdentically(t *testing.T) {
	tmpDir := t.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	enUS := []byte(`{"greeting":"Hello","items":{"one":"# item","other":"# items"}}`)
	if err := os.WriteFile(filepath.Join(localeDir, "en-US.json"), enUS, 0o644); err != nil {
		t.Fatalf("Failed to create en-US file: %v", err)
	}

	tr, err := New(
		WithFileSystemLoader(localeDir),
		WithDefaultLocale("en-US"),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if got := tr.Translate("greeting"); got != "Hello" {
		t.Errorf("Translate = %q, want %q", got, "Hello")
	}
	if got := tr.TranslatePlural("items", 1); got != "1 item" {
		t.Errorf("TranslatePlural(1) = %q, want %q", got, "1 item")
	}
	if got := tr.Translate("missing"); got != "missing" {
		t.Errorf("Translate(missing) = %q, want %q", got, "missing")
	}
}

// =============================================================================
// Finding 5: ContextTranslator tests
// =============================================================================

func TestContextTranslator_Translate(t *testing.T) {
	tmpDir := t.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	enUS := []byte(`{"greeting":"Hello","welcome":"Welcome, %s!"}`)
	if err := os.WriteFile(filepath.Join(localeDir, "en-US.json"), enUS, 0o644); err != nil {
		t.Fatalf("Failed to create en-US file: %v", err)
	}

	tr, err := New(WithFileSystemLoader(localeDir), WithDefaultLocale("en-US"))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	ct := tr.WithContext(context.Background())
	if ct == nil {
		t.Fatal("WithContext returned nil")
	}

	if got := ct.Translate("greeting"); got != "Hello" {
		t.Errorf("ContextTranslator.Translate = %q, want %q", got, "Hello")
	}
}

func TestContextTranslator_TranslateWithArgs(t *testing.T) {
	tmpDir := t.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	enUS := []byte(`{"welcome":"Welcome, %s!"}`)
	if err := os.WriteFile(filepath.Join(localeDir, "en-US.json"), enUS, 0o644); err != nil {
		t.Fatalf("Failed to create en-US file: %v", err)
	}

	tr, err := New(WithFileSystemLoader(localeDir), WithDefaultLocale("en-US"))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	ct := tr.WithContext(context.Background())
	if got := ct.TranslateWithArgs("welcome", "Alice"); got != "Welcome, Alice!" {
		t.Errorf("ContextTranslator.TranslateWithArgs = %q, want %q", got, "Welcome, Alice!")
	}
}

func TestContextTranslator_ImplementsTranslatorProvider(t *testing.T) {
	tmpDir := t.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	enUS := []byte(`{"greeting":"Hello"}`)
	if err := os.WriteFile(filepath.Join(localeDir, "en-US.json"), enUS, 0o644); err != nil {
		t.Fatalf("Failed to create en-US file: %v", err)
	}

	tr, err := New(WithFileSystemLoader(localeDir), WithDefaultLocale("en-US"))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	ct := tr.WithContext(context.Background())

	// Verify it satisfies TranslatorProvider by assigning to the interface.
	var provider core.TranslatorProvider = ct
	if got := provider.Translate("greeting"); got != "Hello" {
		t.Errorf("TranslatorProvider.Translate = %q, want %q", got, "Hello")
	}
	if got := provider.GetLocale(); got != "en-US" {
		t.Errorf("TranslatorProvider.GetLocale = %q, want %q", got, "en-US")
	}
}

func TestContextTranslator_SetLocaleAndGetLocale(t *testing.T) {
	tmpDir := t.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	enUS := []byte(`{"greeting":"Hello"}`)
	esES := []byte(`{"greeting":"Hola"}`)
	if err := os.WriteFile(filepath.Join(localeDir, "en-US.json"), enUS, 0o644); err != nil {
		t.Fatalf("Failed to create en-US file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(localeDir, "es-ES.json"), esES, 0o644); err != nil {
		t.Fatalf("Failed to create es-ES file: %v", err)
	}

	tr, err := New(WithFileSystemLoader(localeDir), WithDefaultLocale("en-US"))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	ct := tr.WithContext(context.Background())
	ct.SetLocale("es-ES")
	if got := ct.GetLocale(); got != "es-ES" {
		t.Errorf("GetLocale after SetLocale = %q, want %q", got, "es-ES")
	}
	if got := ct.Translate("greeting"); got != "Hola" {
		t.Errorf("Translate after SetLocale = %q, want %q", got, "Hola")
	}
}

func TestContextTranslator_HasKey(t *testing.T) {
	tmpDir := t.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	enUS := []byte(`{"greeting":"Hello"}`)
	if err := os.WriteFile(filepath.Join(localeDir, "en-US.json"), enUS, 0o644); err != nil {
		t.Fatalf("Failed to create en-US file: %v", err)
	}

	tr, err := New(WithFileSystemLoader(localeDir), WithDefaultLocale("en-US"))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	ct := tr.WithContext(context.Background())
	if !ct.HasKey("greeting") {
		t.Error("HasKey(greeting) = false, want true")
	}
	if ct.HasKey("nonexistent") {
		t.Error("HasKey(nonexistent) = true, want false")
	}
}

// =============================================================================
// Finding 10: Convenience constructor tests
// =============================================================================

func TestNewWithFS_CreatesTranslator(t *testing.T) {
	tmpDir := t.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	enUS := []byte(`{"greeting":"Hello"}`)
	if err := os.WriteFile(filepath.Join(localeDir, "en-US.json"), enUS, 0o644); err != nil {
		t.Fatalf("Failed to create en-US file: %v", err)
	}

	tr, err := NewWithFS(localeDir, "en-US")
	if err != nil {
		t.Fatalf("NewWithFS() error = %v", err)
	}
	if tr == nil {
		t.Fatal("NewWithFS() returned nil")
	}
	if got := tr.GetLocale(); got != "en-US" {
		t.Errorf("GetLocale() = %q, want %q", got, "en-US")
	}
	if got := tr.Translate("greeting"); got != "Hello" {
		t.Errorf("Translate(greeting) = %q, want %q", got, "Hello")
	}
}

func TestNewWithFS_WithAdditionalOptions(t *testing.T) {
	tmpDir := t.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	enUS := []byte(`{"greeting":"Hello"}`)
	if err := os.WriteFile(filepath.Join(localeDir, "en-US.json"), enUS, 0o644); err != nil {
		t.Fatalf("Failed to create en-US file: %v", err)
	}

	cache := NewMapCache()
	tr, err := NewWithFS(localeDir, "en-US", WithCache(cache))
	if err != nil {
		t.Fatalf("NewWithFS() error = %v", err)
	}

	// Translate to populate cache
	if got := tr.Translate("greeting"); got != "Hello" {
		t.Errorf("Translate(greeting) = %q, want %q", got, "Hello")
	}

	// Verify cache was used
	if val, ok := cache.Get("en-US:greeting"); !ok || val != "Hello" {
		t.Errorf("cache miss after Translate; got (%q, %v)", val, ok)
	}
}

func TestNewWithFS_InvalidLocale(t *testing.T) {
	tmpDir := t.TempDir()
	_, err := NewWithFS(tmpDir, "invalid!!locale")
	if err == nil {
		t.Error("NewWithFS() should fail with invalid locale")
	}
}

func TestNewWithRegistry_InvalidLocale(t *testing.T) {
	_, err := NewWithRegistry("invalid!!locale")
	if err == nil {
		t.Error("NewWithRegistry() should fail with invalid locale")
	}
}

// ---------------------------------------------------------------------------
// Game-oriented benchmarks
// ---------------------------------------------------------------------------

// gameHUDKeys holds the translation keys a game HUD renders each frame.
var gameHUDKeys = [5]string{
	"hud.health",
	"hud.ammo",
	"hud.score",
	"hud.level",
	"hud.timer",
}

// setupGameBenchmarkLocales creates locale files with game HUD keys.
func setupGameBenchmarkLocales(b *testing.B) string {
	b.Helper()
	tmpDir := b.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		b.Fatalf("mkdir: %v", err)
	}

	enUS := []byte(`{
		"hud":{"health":"Health","ammo":"Ammo","score":"Score","level":"Level","timer":"Time"},
		"menu":{"play":"Play","settings":"Settings","quit":"Quit"},
		"status":{"connecting":"Connecting...","ready":"Ready"}
	}`)
	if err := os.WriteFile(filepath.Join(localeDir, "en-US.json"), enUS, 0o644); err != nil {
		b.Fatalf("write en-US: %v", err)
	}

	return localeDir
}

// BenchmarkTranslate_GameLoop simulates a game render loop translating UI
// strings each frame. Five HUD keys are translated per iteration with
// caching enabled, matching the access pattern of a 60fps game.
// Baseline: ~130 ns/op, 5 allocs/op (1 per key for cache key concat)
func BenchmarkTranslate_GameLoop(b *testing.B) {
	localeDir := setupGameBenchmarkLocales(b)
	cache := NewMapCache()
	tr, err := New(
		WithFileSystemLoader(localeDir),
		WithDefaultLocale("en-US"),
		WithCache(cache),
	)
	if err != nil {
		b.Fatalf("New: %v", err)
	}

	// Warm the cache for all HUD keys.
	for _, key := range gameHUDKeys {
		tr.Translate(key)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for _, key := range gameHUDKeys {
			_ = tr.Translate(key)
		}
	}
}

// BenchmarkTranslate_Parallel_GameServer simulates a game server where many
// player goroutines translate UI strings concurrently, stressing the
// translation cache and locale read path under contention.
// Baseline: ~130 ns/op per op, 1 alloc/op (cache key concat)
func BenchmarkTranslate_Parallel_GameServer(b *testing.B) {
	localeDir := setupGameBenchmarkLocales(b)
	cache := NewMapCache()
	tr, err := New(
		WithFileSystemLoader(localeDir),
		WithDefaultLocale("en-US"),
		WithCache(cache),
	)
	if err != nil {
		b.Fatalf("New: %v", err)
	}

	// Warm cache.
	for _, key := range gameHUDKeys {
		tr.Translate(key)
	}

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		idx := 0
		for pb.Next() {
			_ = tr.Translate(gameHUDKeys[idx%len(gameHUDKeys)])
			idx++
		}
	})
}

// =============================================================================
// ReloadLocale tests
// =============================================================================

func TestReloadLocale_ClearsAndReloads(t *testing.T) {
	tmpDir := t.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Write initial translation file.
	enUS := []byte(`{"greeting":"Hello"}`)
	filePath := filepath.Join(localeDir, "en-US.json")
	if err := os.WriteFile(filePath, enUS, 0o644); err != nil {
		t.Fatalf("Failed to create en-US file: %v", err)
	}

	translator, err := New(WithFileSystemLoader(localeDir), WithDefaultLocale("en-US"))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Translate to populate the internal cache.
	got := translator.Translate("greeting")
	if got != "Hello" {
		t.Fatalf("initial Translate = %q, want %q", got, "Hello")
	}

	// Modify the translation file on disk.
	updated := []byte(`{"greeting":"Hi there"}`)
	if err := os.WriteFile(filePath, updated, 0o644); err != nil {
		t.Fatalf("Failed to update en-US file: %v", err)
	}

	// ReloadLocale should clear the internal cache and re-read the file.
	if err := translator.ReloadLocale("en-US"); err != nil {
		t.Fatalf("ReloadLocale() error = %v", err)
	}

	got = translator.Translate("greeting")
	if got != "Hi there" {
		t.Errorf("after ReloadLocale, Translate = %q, want %q", got, "Hi there")
	}
}

func TestReloadLocale_InvalidatesResultCache(t *testing.T) {
	tmpDir := t.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	enUS := []byte(`{"greeting":"Hello"}`)
	filePath := filepath.Join(localeDir, "en-US.json")
	if err := os.WriteFile(filePath, enUS, 0o644); err != nil {
		t.Fatalf("Failed to create en-US file: %v", err)
	}

	cache := NewMapCache()
	translator, err := New(
		WithFileSystemLoader(localeDir),
		WithDefaultLocale("en-US"),
		WithCache(cache),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Translate to populate the result cache.
	got := translator.Translate("greeting")
	if got != "Hello" {
		t.Fatalf("initial Translate = %q, want %q", got, "Hello")
	}

	// Verify the result cache has the entry.
	if _, ok := cache.Get("en-US:greeting"); !ok {
		t.Fatal("expected result cache to contain en-US:greeting")
	}

	// Modify the file and reload.
	updated := []byte(`{"greeting":"Hey"}`)
	if err := os.WriteFile(filePath, updated, 0o644); err != nil {
		t.Fatalf("Failed to update en-US file: %v", err)
	}

	if err := translator.ReloadLocale("en-US"); err != nil {
		t.Fatalf("ReloadLocale() error = %v", err)
	}

	// The result cache should have been invalidated.
	if _, ok := cache.Get("en-US:greeting"); ok {
		t.Error("result cache should be invalidated after ReloadLocale")
	}

	// New translation should reflect the updated file.
	got = translator.Translate("greeting")
	if got != "Hey" {
		t.Errorf("after ReloadLocale, Translate = %q, want %q", got, "Hey")
	}
}

func TestReloadLocale_NonexistentLocale(t *testing.T) {
	tmpDir := t.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	enUS := []byte(`{"greeting":"Hello"}`)
	if err := os.WriteFile(filepath.Join(localeDir, "en-US.json"), enUS, 0o644); err != nil {
		t.Fatalf("Failed to create en-US file: %v", err)
	}

	translator, err := New(WithFileSystemLoader(localeDir), WithDefaultLocale("en-US"))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// ReloadLocale for a locale that has no file should return an error.
	err = translator.ReloadLocale("xx-XX")
	if err == nil {
		t.Error("ReloadLocale(xx-XX) should return an error for nonexistent locale")
	}
}

// =============================================================================
// ContextTranslator: TranslatePlural and TranslateGender tests
// =============================================================================

func TestContextTranslator_TranslatePlural(t *testing.T) {
	tmpDir := t.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	enUS := []byte(`{"items":{"one":"# item","other":"# items"}}`)
	if err := os.WriteFile(filepath.Join(localeDir, "en-US.json"), enUS, 0o644); err != nil {
		t.Fatalf("Failed to create en-US file: %v", err)
	}

	tr, err := New(WithFileSystemLoader(localeDir), WithDefaultLocale("en-US"))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	ct := tr.WithContext(context.Background())

	tests := []struct {
		name  string
		count interface{}
		want  string
	}{
		{"singular", 1, "1 item"},
		{"plural", 5, "5 items"},
		{"zero", 0, "0 items"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ct.TranslatePlural("items", tt.count)
			if got != tt.want {
				t.Errorf("ContextTranslator.TranslatePlural(items, %v) = %q, want %q", tt.count, got, tt.want)
			}
		})
	}
}

func TestContextTranslator_TranslateGender(t *testing.T) {
	tmpDir := t.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	enUS := []byte(`{"welcome":{"masculine":"He joined","feminine":"She joined","other":"They joined"}}`)
	if err := os.WriteFile(filepath.Join(localeDir, "en-US.json"), enUS, 0o644); err != nil {
		t.Fatalf("Failed to create en-US file: %v", err)
	}

	tr, err := New(WithFileSystemLoader(localeDir), WithDefaultLocale("en-US"))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	ct := tr.WithContext(context.Background())

	tests := []struct {
		name   string
		gender core.GenderCategory
		want   string
	}{
		{"masculine", core.Masculine, "He joined"},
		{"feminine", core.Feminine, "She joined"},
		{"neuter fallback to other", core.Neuter, "They joined"},
		{"other", core.GenderOther, "They joined"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ct.TranslateGender("welcome", tt.gender)
			if got != tt.want {
				t.Errorf("ContextTranslator.TranslateGender(welcome, %s) = %q, want %q", tt.gender, got, tt.want)
			}
		})
	}
}
