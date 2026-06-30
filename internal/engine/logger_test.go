// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

package engine

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/0verkilll/i18n/internal/core"
)

// TestSetLogger_ValidLogger tests that SetLogger with a valid logger sets the global logger.
func TestSetLogger_ValidLogger(t *testing.T) {
	defer SetLogger(nil)

	customLogger := NopLogger{}
	SetLogger(customLogger)

	got := GetLogger()
	if got == nil {
		t.Fatal("GetLogger() returned nil after SetLogger with valid logger")
	}
	if _, ok := got.(NopLogger); !ok {
		t.Errorf("GetLogger() returned unexpected type %T, want NopLogger", got)
	}
}

// TestSetLogger_NilResetsToNopLogger tests that SetLogger with nil resets to NopLogger.
func TestSetLogger_NilResetsToNopLogger(t *testing.T) {
	defer SetLogger(nil)

	SetLogger(NopLogger{})
	SetLogger(nil)

	got := GetLogger()
	if got == nil {
		t.Fatal("GetLogger() returned nil after SetLogger(nil)")
	}
	if _, ok := got.(NopLogger); !ok {
		t.Errorf("GetLogger() returned %T after SetLogger(nil), want NopLogger", got)
	}
}

// TestGetLogger_DefaultIsNopLogger tests that GetLogger returns NopLogger by default.
func TestGetLogger_DefaultIsNopLogger(t *testing.T) {
	SetLogger(nil)

	got := GetLogger()
	if got == nil {
		t.Fatal("GetLogger() returned nil by default")
	}
	if _, ok := got.(NopLogger); !ok {
		t.Errorf("GetLogger() returned %T by default, want NopLogger", got)
	}
}

// TestSetLogger_GetLogger_ConcurrentAccess tests thread-safe SetLogger/GetLogger.
func TestSetLogger_GetLogger_ConcurrentAccess(t *testing.T) {
	defer SetLogger(nil)

	const numGoroutines = 100
	const numIterations = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 2)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < numIterations; j++ {
				if j%2 == 0 {
					SetLogger(nil)
				} else {
					SetLogger(NopLogger{})
				}
			}
		}()
	}

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < numIterations; j++ {
				got := GetLogger()
				if got == nil {
					t.Error("GetLogger() returned nil during concurrent access")
					return
				}
			}
		}()
	}

	wg.Wait()

	finalLogger := GetLogger()
	if finalLogger == nil {
		t.Error("GetLogger() returned nil after concurrent operations")
	}
}

// =============================================================================
// Unified mock logger (merged from logger_integration_test.go and logger_option_test.go)
// =============================================================================

// testMockLogger is a thread-safe mock logger that captures all log calls with fields.
type testMockLogger struct {
	debugCalls []testLogCall
	infoCalls  []testLogCall
	warnCalls  []testLogCall
	errorCalls []testLogCall
	fields     []any
	mu         sync.Mutex
}

type testLogCall struct {
	msg    string
	args   []any
	fields []any
}

func newTestMockLogger() *testMockLogger {
	return &testMockLogger{}
}

func (m *testMockLogger) Debug(msg string, args ...any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.debugCalls = append(m.debugCalls, testLogCall{msg: msg, args: args, fields: m.fields})
}

func (m *testMockLogger) Info(msg string, args ...any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.infoCalls = append(m.infoCalls, testLogCall{msg: msg, args: args, fields: m.fields})
}

func (m *testMockLogger) Warn(msg string, args ...any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.warnCalls = append(m.warnCalls, testLogCall{msg: msg, args: args, fields: m.fields})
}

func (m *testMockLogger) Error(msg string, args ...any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errorCalls = append(m.errorCalls, testLogCall{msg: msg, args: args, fields: m.fields})
}

func (m *testMockLogger) Fatal(string, ...any) {}

func (m *testMockLogger) WithFields(fields ...any) core.Logger {
	m.mu.Lock()
	defer m.mu.Unlock()
	newLogger := &testMockLogger{
		debugCalls: m.debugCalls,
		infoCalls:  m.infoCalls,
		warnCalls:  m.warnCalls,
		errorCalls: m.errorCalls,
		fields:     append(append([]any{}, m.fields...), fields...),
	}
	return newLogger
}

func (m *testMockLogger) WithContext(context.Context) core.Logger { return m }
func (m *testMockLogger) WithLevel(core.LogLevel) core.Logger     { return m }
func (m *testMockLogger) Enabled(core.LogLevel) bool              { return true }

func (m *testMockLogger) getDebugCalls() []testLogCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]testLogCall{}, m.debugCalls...)
}

func (m *testMockLogger) getInfoCalls() []testLogCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]testLogCall{}, m.infoCalls...)
}

func (m *testMockLogger) getWarnCalls() []testLogCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]testLogCall{}, m.warnCalls...)
}

func (m *testMockLogger) getErrorCalls() []testLogCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]testLogCall{}, m.errorCalls...)
}

func (m *testMockLogger) reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.debugCalls = nil
	m.infoCalls = nil
	m.warnCalls = nil
	m.errorCalls = nil
}

func hasLogMessage(calls []testLogCall, msgSubstring string) bool {
	for _, call := range calls {
		if strings.Contains(call.msg, msgSubstring) {
			return true
		}
	}
	return false
}

func hasFieldKey(calls []testLogCall, key string) bool {
	for _, call := range calls {
		for i := 0; i < len(call.args)-1; i += 2 {
			if keyStr, ok := call.args[i].(string); ok && keyStr == key {
				return true
			}
		}
		for i := 0; i < len(call.fields)-1; i += 2 {
			if keyStr, ok := call.fields[i].(string); ok && keyStr == key {
				return true
			}
		}
	}
	return false
}

func getFieldValue(calls []testLogCall, key string) (any, bool) {
	for _, call := range calls {
		for i := 0; i < len(call.args)-1; i += 2 {
			if keyStr, ok := call.args[i].(string); ok && keyStr == key {
				return call.args[i+1], true
			}
		}
	}
	return nil, false
}

// failingTestLoader is a core.TranslationLoader that always fails.
type failingTestLoader struct{}

func (f *failingTestLoader) Load(string) ([]byte, error) {
	return nil, os.ErrNotExist
}

// createTestLocaleDir creates a temporary directory with locale files for testing.
func createTestLocaleDir(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	enUSContent := []byte(`{"greeting": "Hello", "farewell": "Goodbye", "fallback_only": "From US", "welcome": "Welcome, %s!"}`)
	if err := os.WriteFile(filepath.Join(localeDir, "en-US.json"), enUSContent, 0o644); err != nil {
		t.Fatalf("Failed to create en-US test file: %v", err)
	}

	enGBContent := []byte(`{"greeting": "Hello from GB", "farewell": "Cheerio"}`)
	if err := os.WriteFile(filepath.Join(localeDir, "en-GB.json"), enGBContent, 0o644); err != nil {
		t.Fatalf("Failed to create en-GB test file: %v", err)
	}

	return localeDir
}

// =============================================================================
// Merged from logger_integration_test.go
// =============================================================================

func TestLogging_DebugOnLoadingTranslations(t *testing.T) {
	defer SetLogger(nil)
	localeDir := createTestLocaleDir(t)

	mockLogger := newTestMockLogger()

	translator, err := New(
		WithFileSystemLoader(localeDir),
		WithDefaultLocale("en-US"),
		WithLogger(mockLogger),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	_ = translator.Translate("greeting")

	debugCalls := mockLogger.getDebugCalls()
	if len(debugCalls) == 0 {
		t.Error("Expected debug logs when loading translations, got none")
	}

	foundLoadingLog := hasLogMessage(debugCalls, "loading") || hasLogMessage(debugCalls, "translations")
	if !foundLoadingLog {
		t.Errorf("Expected debug log about loading translations, got: %+v", debugCalls)
	}
}

func TestLogging_DebugOnCacheHit(t *testing.T) {
	defer SetLogger(nil)
	localeDir := createTestLocaleDir(t)

	mockLogger := newTestMockLogger()

	translator, err := New(
		WithFileSystemLoader(localeDir),
		WithDefaultLocale("en-US"),
		WithLogger(mockLogger),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// First call loads translations from disk.
	result1 := translator.Translate("greeting")
	if result1 == "greeting" {
		t.Error("first Translate should resolve the key")
	}

	mockLogger.reset()

	// Second call for a different key in the same locale should
	// reuse the already-loaded translation map (no reload from disk).
	result2 := translator.Translate("farewell")
	if result2 == "farewell" {
		t.Error("second Translate should resolve from cached translations")
	}

	// Verify that no "loading translations from source" debug log was emitted,
	// confirming the translations map was reused.
	debugCalls := mockLogger.getDebugCalls()
	for _, call := range debugCalls {
		if call.msg == "loading translations from source" {
			t.Error("expected no reload from source on second key lookup in same locale")
		}
	}
}

func TestLogging_WarnOnFallbackChain(t *testing.T) {
	defer SetLogger(nil)
	localeDir := createTestLocaleDir(t)

	mockLogger := newTestMockLogger()

	translator, err := New(
		WithFileSystemLoader(localeDir),
		WithDefaultLocale("en-GB"),
		WithLogger(mockLogger),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	mockLogger.reset()

	result := translator.Translate("fallback_only")
	if result != "From US" {
		t.Fatalf("Expected 'From US' from fallback, got: %s", result)
	}

	warnCalls := mockLogger.getWarnCalls()
	foundFallbackWarn := hasLogMessage(warnCalls, "fallback") || hasLogMessage(warnCalls, "not found")
	if !foundFallbackWarn {
		t.Errorf("Expected warn log about fallback chain usage, got warn calls: %+v", warnCalls)
	}
}

func TestLogging_ErrorOnLoadFailure(t *testing.T) {
	defer SetLogger(nil)

	mockLogger := newTestMockLogger()

	translator, err := New(
		WithLoader(&failingTestLoader{}),
		WithDefaultLocale("en-US"),
		WithLogger(mockLogger),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	_ = translator.Translate("greeting")

	errorCalls := mockLogger.getErrorCalls()
	foundLoadError := hasLogMessage(errorCalls, "failed") || hasLogMessage(errorCalls, "error") || hasLogMessage(errorCalls, "load")
	if len(errorCalls) == 0 || !foundLoadError {
		t.Errorf("Expected error log about load failure, got: %+v", errorCalls)
	}
}

func TestLogging_InfoOnLocaleChange(t *testing.T) {
	defer SetLogger(nil)
	localeDir := createTestLocaleDir(t)

	mockLogger := newTestMockLogger()

	translator, err := New(
		WithFileSystemLoader(localeDir),
		WithDefaultLocale("en-US"),
		WithLogger(mockLogger),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	mockLogger.reset()
	translator.SetLocale("en-GB")

	infoCalls := mockLogger.getInfoCalls()
	if len(infoCalls) == 0 {
		t.Error("Expected info log when changing locale, got none")
	}

	foundLocaleChange := hasLogMessage(infoCalls, "locale") || hasLogMessage(infoCalls, "changed")
	if !foundLocaleChange {
		t.Errorf("Expected info log about locale change, got: %+v", infoCalls)
	}
}

func TestLogging_SilentWithNopLogger(t *testing.T) {
	defer SetLogger(nil)
	localeDir := createTestLocaleDir(t)

	translator, err := New(
		WithFileSystemLoader(localeDir),
		WithDefaultLocale("en-US"),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if _, ok := translator.logger.(NopLogger); !ok {
		t.Errorf("Expected NopLogger by default, got %T", translator.logger)
	}

	_ = translator.Translate("greeting")
	_ = translator.Translate("nonexistent_key")
	translator.SetLocale("en-GB")
	_ = translator.HasKey("greeting")

	if translator.logger.Enabled(core.LevelDebug) {
		t.Error("NopLogger.Enabled() should return false")
	}
}

func TestLogging_StructuredFieldsInLogMessages(t *testing.T) {
	defer SetLogger(nil)
	localeDir := createTestLocaleDir(t)

	mockLogger := newTestMockLogger()

	translator, err := New(
		WithFileSystemLoader(localeDir),
		WithDefaultLocale("en-US"),
		WithLogger(mockLogger),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	_ = translator.Translate("greeting")

	debugCalls := mockLogger.getDebugCalls()
	if !hasFieldKey(debugCalls, "locale") {
		t.Error("Expected 'locale' field in debug log calls")
	}

	if localeVal, found := getFieldValue(debugCalls, "locale"); found {
		if localeStr, ok := localeVal.(string); ok {
			if localeStr != "en-US" {
				t.Errorf("Expected locale field value 'en-US', got '%s'", localeStr)
			}
		}
	}
}

func TestLogging_TranslateWithArgsGeneratesLogs(t *testing.T) {
	defer SetLogger(nil)
	localeDir := createTestLocaleDir(t)

	mockLogger := newTestMockLogger()

	translator, err := New(
		WithFileSystemLoader(localeDir),
		WithDefaultLocale("en-US"),
		WithLogger(mockLogger),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	mockLogger.reset()

	result := translator.TranslateWithArgs("welcome", "World")
	if result != "Welcome, World!" {
		t.Errorf("TranslateWithArgs result = %s, want 'Welcome, World!'", result)
	}

	debugCalls := mockLogger.getDebugCalls()
	if len(debugCalls) == 0 {
		t.Error("Expected debug logs from TranslateWithArgs, got none")
	}

	foundTranslationLog := hasLogMessage(debugCalls, "translat") || hasLogMessage(debugCalls, "cache") || hasLogMessage(debugCalls, "found")
	if !foundTranslationLog {
		t.Errorf("Expected translation-related debug log, got: %+v", debugCalls)
	}
}

func TestNopLogger_AllMethodsReturnExpectedValues(t *testing.T) {
	logger := NopLogger{}

	levels := []core.LogLevel{core.LevelDebug, core.LevelInfo, core.LevelWarn, core.LevelError, core.LevelFatal}
	for _, level := range levels {
		if logger.Enabled(level) {
			t.Errorf("NopLogger.Enabled(%v) = true, want false", level)
		}
	}

	withFields := logger.WithFields("key", "value")
	if _, ok := withFields.(NopLogger); !ok {
		t.Errorf("NopLogger.WithFields() returned %T, want NopLogger", withFields)
	}

	withCtx := logger.WithContext(context.Background())
	if _, ok := withCtx.(NopLogger); !ok {
		t.Errorf("NopLogger.WithContext() returned %T, want NopLogger", withCtx)
	}

	withLevel := logger.WithLevel(core.LevelDebug)
	if _, ok := withLevel.(NopLogger); !ok {
		t.Errorf("NopLogger.WithLevel() returned %T, want NopLogger", withLevel)
	}

	// Verify logging methods don't panic
	logger.Debug("test debug")
	logger.Info("test info")
	logger.Warn("test warn")
	logger.Error("test error")
	logger.Fatal("test fatal")
}

func TestLogging_ConcurrentLoggingIsSafe(t *testing.T) {
	defer SetLogger(nil)
	localeDir := createTestLocaleDir(t)

	mockLogger := newTestMockLogger()

	translator, err := New(
		WithFileSystemLoader(localeDir),
		WithDefaultLocale("en-US"),
		WithLogger(mockLogger),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	const numGoroutines = 20
	const numIterations = 50

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < numIterations; j++ {
				switch j % 4 {
				case 0:
					_ = translator.Translate("greeting")
				case 1:
					_ = translator.Translate("farewell")
				case 2:
					_ = translator.HasKey("greeting")
				case 3:
					if j%2 == 0 {
						translator.SetLocale("en-US")
					} else {
						translator.SetLocale("en-GB")
					}
				}
			}
		}()
	}

	wg.Wait()

	debugCalls := mockLogger.getDebugCalls()
	infoCalls := mockLogger.getInfoCalls()
	if len(debugCalls)+len(infoCalls) == 0 {
		t.Error("Expected some log calls after concurrent operations")
	}
}

func TestLogging_LocaleChangeContainsOldAndNewLocale(t *testing.T) {
	defer SetLogger(nil)
	localeDir := createTestLocaleDir(t)

	mockLogger := newTestMockLogger()

	translator, err := New(
		WithFileSystemLoader(localeDir),
		WithDefaultLocale("en-US"),
		WithLogger(mockLogger),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	mockLogger.reset()
	translator.SetLocale("en-GB")

	infoCalls := mockLogger.getInfoCalls()
	if len(infoCalls) == 0 {
		t.Fatal("Expected info log for locale change")
	}

	if !hasFieldKey(infoCalls, "old_locale") {
		t.Error("Expected 'old_locale' field in locale change log")
	}
	if !hasFieldKey(infoCalls, "new_locale") {
		t.Error("Expected 'new_locale' field in locale change log")
	}

	if oldLocale, found := getFieldValue(infoCalls, "old_locale"); found {
		if oldStr, ok := oldLocale.(string); ok && oldStr != "en-US" {
			t.Errorf("Expected old_locale 'en-US', got '%s'", oldStr)
		}
	}
	if newLocale, found := getFieldValue(infoCalls, "new_locale"); found {
		if newStr, ok := newLocale.(string); ok && newStr != "en-GB" {
			t.Errorf("Expected new_locale 'en-GB', got '%s'", newStr)
		}
	}
}

// =============================================================================
// Merged from logger_option_test.go
// =============================================================================

func TestWithLogger_SetsLoggerOnTranslator(t *testing.T) {
	defer SetLogger(nil)
	localeDir := createTestLocaleDir(t)

	customLogger := newTestMockLogger()

	translator, err := New(
		WithFileSystemLoader(localeDir),
		WithDefaultLocale("en-US"),
		WithLogger(customLogger),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if translator.logger == nil {
		t.Fatal("Translator logger is nil after WithLogger")
	}

	if _, ok := translator.logger.(*testMockLogger); !ok {
		t.Errorf("Translator logger type = %T, want *testMockLogger", translator.logger)
	}
}

func TestWithLogger_NilUsesNopLogger(t *testing.T) {
	defer SetLogger(nil)
	localeDir := createTestLocaleDir(t)

	translator, err := New(
		WithFileSystemLoader(localeDir),
		WithDefaultLocale("en-US"),
		WithLogger(nil),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if translator.logger == nil {
		t.Fatal("Translator logger is nil after WithLogger(nil)")
	}

	if _, ok := translator.logger.(NopLogger); !ok {
		t.Errorf("Translator logger type = %T after WithLogger(nil), want NopLogger", translator.logger)
	}
}

func TestTranslator_WithoutWithLogger_UsesPackageLevelLogger(t *testing.T) {
	defer SetLogger(nil)
	localeDir := createTestLocaleDir(t)

	packageLogger := newTestMockLogger()
	SetLogger(packageLogger)

	translator, err := New(
		WithFileSystemLoader(localeDir),
		WithDefaultLocale("en-US"),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if translator.logger == nil {
		t.Fatal("Translator logger is nil without WithLogger")
	}

	if _, ok := translator.logger.(*testMockLogger); !ok {
		t.Errorf("Translator logger type = %T, want *testMockLogger (package-level)", translator.logger)
	}
}

func TestWithLogger_OverridesPackageLevelLogger(t *testing.T) {
	defer SetLogger(nil)
	localeDir := createTestLocaleDir(t)

	packageLogger := newTestMockLogger()
	SetLogger(packageLogger)

	instanceLogger := newTestMockLogger()

	translator, err := New(
		WithFileSystemLoader(localeDir),
		WithDefaultLocale("en-US"),
		WithLogger(instanceLogger),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if translator.logger == nil {
		t.Fatal("Translator logger is nil with WithLogger")
	}

	if translator.logger == packageLogger {
		t.Error("Translator should use instance logger, not package-level logger")
	}

	if translator.logger != instanceLogger {
		t.Error("Translator should use the exact instance logger provided to WithLogger")
	}
}

// =============================================================================
// Distributed from benchmark_test.go: logger benchmarks
// =============================================================================

// benchmarkMockLogger is a lightweight thread-safe mock for benchmarking.
type benchmarkMockLogger struct {
	mu         sync.Mutex
	debugCalls int
	infoCalls  int
	warnCalls  int
	errorCalls int
}

func newBenchmarkMockLogger() *benchmarkMockLogger {
	return &benchmarkMockLogger{}
}

func (m *benchmarkMockLogger) Debug(string, ...any) {
	m.mu.Lock()
	m.debugCalls++
	m.mu.Unlock()
}

func (m *benchmarkMockLogger) Info(string, ...any) {
	m.mu.Lock()
	m.infoCalls++
	m.mu.Unlock()
}

func (m *benchmarkMockLogger) Warn(string, ...any) {
	m.mu.Lock()
	m.warnCalls++
	m.mu.Unlock()
}

func (m *benchmarkMockLogger) Error(string, ...any) {
	m.mu.Lock()
	m.errorCalls++
	m.mu.Unlock()
}

func (m *benchmarkMockLogger) Fatal(string, ...any)                    {}
func (m *benchmarkMockLogger) WithFields(...any) core.Logger           { return m }
func (m *benchmarkMockLogger) WithContext(context.Context) core.Logger { return m }
func (m *benchmarkMockLogger) WithLevel(core.LogLevel) core.Logger     { return m }
func (m *benchmarkMockLogger) Enabled(core.LogLevel) bool              { return true }

func setupBenchmarkLoggerLocales(b *testing.B) string {
	b.Helper()
	tmpDir := b.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		b.Fatalf("Failed to create test directory: %v", err)
	}

	enUS := []byte(`{"greeting": "Hello", "farewell": "Goodbye", "welcome": "Welcome, %s!"}`)
	if err := os.WriteFile(filepath.Join(localeDir, "en-US.json"), enUS, 0o644); err != nil {
		b.Fatalf("Failed to create locale file: %v", err)
	}

	return localeDir
}

func BenchmarkTranslateWithNopLogger(b *testing.B) {
	localeDir := setupBenchmarkLoggerLocales(b)

	translator, err := New(
		WithFileSystemLoader(localeDir),
		WithDefaultLocale("en-US"),
		WithLogger(NopLogger{}),
	)
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

func BenchmarkTranslateWithActiveLogger(b *testing.B) {
	localeDir := setupBenchmarkLoggerLocales(b)

	activeLogger := newBenchmarkMockLogger()

	translator, err := New(
		WithFileSystemLoader(localeDir),
		WithDefaultLocale("en-US"),
		WithLogger(activeLogger),
	)
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

func BenchmarkLoggerOverhead(b *testing.B) {
	localeDir := setupBenchmarkLoggerLocales(b)

	b.Run("NopLogger", func(b *testing.B) {
		translator, err := New(
			WithFileSystemLoader(localeDir),
			WithDefaultLocale("en-US"),
			WithLogger(NopLogger{}),
		)
		if err != nil {
			b.Fatalf("New() error = %v", err)
		}
		_ = translator.Translate("greeting")

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_ = translator.Translate("greeting")
		}
	})

	b.Run("ActiveLogger", func(b *testing.B) {
		activeLogger := newBenchmarkMockLogger()
		translator, err := New(
			WithFileSystemLoader(localeDir),
			WithDefaultLocale("en-US"),
			WithLogger(activeLogger),
		)
		if err != nil {
			b.Fatalf("New() error = %v", err)
		}
		_ = translator.Translate("greeting")

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_ = translator.Translate("greeting")
		}
	})
}
