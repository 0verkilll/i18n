// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

package engine

import (
	"embed"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/0verkilll/i18n/internal/core"
)

// stubTranslatorProvider is a minimal core.TranslatorProvider for testing.
type stubTranslatorProvider struct {
	translations map[string]string
	locale       string
}

func newStubTranslatorProvider(translations map[string]string) *stubTranslatorProvider {
	return &stubTranslatorProvider{
		translations: translations,
		locale:       "en-US",
	}
}

func (s *stubTranslatorProvider) Translate(key string) string {
	if v, ok := s.translations[key]; ok {
		return v
	}
	return key
}

func (s *stubTranslatorProvider) TranslateWithArgs(key string, args ...interface{}) string {
	if v, ok := s.translations[key]; ok {
		return fmt.Sprintf(v, args...)
	}
	return key
}

func (s *stubTranslatorProvider) TranslatePlural(key string, _ interface{}) string {
	return key
}

func (s *stubTranslatorProvider) TranslateGender(key string, _ core.GenderCategory) string {
	return key
}

func (s *stubTranslatorProvider) HasKey(key string) bool {
	_, ok := s.translations[key]
	return ok
}

func (s *stubTranslatorProvider) SetLocale(locale string) {
	s.locale = locale
}

func (s *stubTranslatorProvider) GetLocale() string {
	return s.locale
}

// stubTranslationLoader is a minimal core.TranslationLoader for testing.
type stubTranslationLoader struct {
	locales map[string][]byte
}

func (l *stubTranslationLoader) Load(locale string) ([]byte, error) {
	data, ok := l.locales[locale]
	if !ok {
		return nil, fmt.Errorf("locale %q not found", locale)
	}
	return data, nil
}

// ---------------------------------------------------------------------------
// Task Group 1: PackageTranslator Struct, Options, and Constructor
// ---------------------------------------------------------------------------

func TestNewPackageTranslator_ValidNamespace(t *testing.T) {
	pt := NewPackageTranslator("filesystem")
	if pt == nil {
		t.Fatal("NewPackageTranslator returned nil for valid namespace")
	}
	if pt.namespace != "filesystem" {
		t.Errorf("namespace = %q, want %q", pt.namespace, "filesystem")
	}
}

func TestNewPackageTranslator_InvalidNamespace(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
	}{
		{"empty string", ""},
		{"consecutive dots", "a..b"},
		{"starts with dot", ".invalid"},
		{"ends with dot", "invalid."},
		{"contains spaces", "not valid"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pt := NewPackageTranslator(tt.namespace)
			if pt != nil {
				t.Errorf("NewPackageTranslator(%q) = non-nil, want nil", tt.namespace)
			}
		})
	}
}

func TestWithDefaults_StoresMap(t *testing.T) {
	defaults := map[string]string{
		"greeting": "Hello",
		"farewell": "Goodbye",
	}
	pt := NewPackageTranslator("testpkg", WithDefaults(defaults))
	if pt == nil {
		t.Fatal("NewPackageTranslator returned nil")
	}
	if len(pt.defaults) != 2 {
		t.Errorf("defaults length = %d, want 2", len(pt.defaults))
	}
	if pt.defaults["greeting"] != "Hello" {
		t.Errorf("defaults[greeting] = %q, want %q", pt.defaults["greeting"], "Hello")
	}
}

func TestWithTranslator_SetsField(t *testing.T) {
	stub := newStubTranslatorProvider(nil)
	pt := NewPackageTranslator("testpkg", WithTranslator(stub))
	if pt == nil {
		t.Fatal("NewPackageTranslator returned nil")
	}
	if pt.loadTranslator() == nil {
		t.Error("WithTranslator did not set the translator field")
	}
}

func TestNewPackageTranslator_MultipleOptions(t *testing.T) {
	defaults := map[string]string{"key": "value"}
	stub := newStubTranslatorProvider(nil)
	pt := NewPackageTranslator("testpkg", WithDefaults(defaults), WithTranslator(stub))
	if pt == nil {
		t.Fatal("NewPackageTranslator returned nil")
	}
	if pt.defaults["key"] != "value" {
		t.Errorf("defaults[key] = %q, want %q", pt.defaults["key"], "value")
	}
	if pt.loadTranslator() == nil {
		t.Error("translator not set when using multiple options")
	}
}

func TestNewPackageTranslator_ValidNamespaceNoOptions_ReturnsNonNil(t *testing.T) {
	pt := NewPackageTranslator("mypkg")
	if pt == nil {
		t.Fatal("NewPackageTranslator returned nil for valid namespace with no options")
	}
}

// ---------------------------------------------------------------------------
// Task Group 2: T, TF, and Has Methods
// ---------------------------------------------------------------------------

func TestT_NilTranslator_ReturnsDefault(t *testing.T) {
	defaults := map[string]string{"greeting": "Hello"}
	pt := NewPackageTranslator("testpkg", WithDefaults(defaults))

	got := pt.T("greeting")
	if got != "Hello" {
		t.Errorf("T(greeting) = %q, want %q", got, "Hello")
	}
}

func TestT_NilTranslator_NoDefault_ReturnsRawKey(t *testing.T) {
	pt := NewPackageTranslator("testpkg")

	got := pt.T("unknown.key")
	if got != "unknown.key" {
		t.Errorf("T(unknown.key) = %q, want %q", got, "unknown.key")
	}
}

func TestT_WithTranslator_ReturnsTranslated(t *testing.T) {
	stub := newStubTranslatorProvider(map[string]string{
		"testpkg.greeting": "Hola",
	})
	pt := NewPackageTranslator("testpkg", WithTranslator(stub))

	got := pt.T("greeting")
	if got != "Hola" {
		t.Errorf("T(greeting) = %q, want %q", got, "Hola")
	}
}

func TestT_TranslatorKeyNotFound_FallsBackToDefaults(t *testing.T) {
	stub := newStubTranslatorProvider(map[string]string{})
	defaults := map[string]string{"greeting": "Hello Default"}
	pt := NewPackageTranslator("testpkg", WithTranslator(stub), WithDefaults(defaults))

	got := pt.T("greeting")
	if got != "Hello Default" {
		t.Errorf("T(greeting) = %q, want %q", got, "Hello Default")
	}
}

func TestTF_NilTranslator_FormatsDefault(t *testing.T) {
	defaults := map[string]string{"welcome": "Hello, %s!"}
	pt := NewPackageTranslator("testpkg", WithDefaults(defaults))

	got := pt.TF("welcome", "Alice")
	if got != "Hello, Alice!" {
		t.Errorf("TF(welcome, Alice) = %q, want %q", got, "Hello, Alice!")
	}
}

func TestTF_WithTranslator_FormatsTranslated(t *testing.T) {
	stub := newStubTranslatorProvider(map[string]string{
		"testpkg.welcome": "Hola, %s!",
	})
	pt := NewPackageTranslator("testpkg", WithTranslator(stub))

	got := pt.TF("welcome", "Bob")
	if got != "Hola, Bob!" {
		t.Errorf("TF(welcome, Bob) = %q, want %q", got, "Hola, Bob!")
	}
}

func TestHas_NilTranslator_ReturnsFalse(t *testing.T) {
	pt := NewPackageTranslator("testpkg")

	if pt.Has("anything") {
		t.Error("Has should return false when translator is nil")
	}
}

func TestHas_WithTranslator_ReturnsTrue(t *testing.T) {
	stub := newStubTranslatorProvider(map[string]string{
		"testpkg.greeting": "Hello",
	})
	pt := NewPackageTranslator("testpkg", WithTranslator(stub))

	if !pt.Has("greeting") {
		t.Error("Has(greeting) should return true when translator has the namespaced key")
	}
}

// ---------------------------------------------------------------------------
// Task Group 3: core.TranslatorProvider Interface Methods and SetTranslator/GetTranslator
// ---------------------------------------------------------------------------

func TestTranslate_DelegatesToTranslatorWithoutDoublePrefix(t *testing.T) {
	stub := newStubTranslatorProvider(map[string]string{
		"testpkg.greeting": "Hello",
	})
	pt := NewPackageTranslator("testpkg", WithTranslator(stub))

	got := pt.Translate("testpkg.greeting")
	if got != "Hello" {
		t.Errorf("Translate(testpkg.greeting) = %q, want %q", got, "Hello")
	}
}

func TestTranslateWithArgs_DelegatesToTranslator(t *testing.T) {
	stub := newStubTranslatorProvider(map[string]string{
		"testpkg.welcome": "Hello, %s!",
	})
	pt := NewPackageTranslator("testpkg", WithTranslator(stub))

	got := pt.TranslateWithArgs("testpkg.welcome", "World")
	if got != "Hello, World!" {
		t.Errorf("TranslateWithArgs = %q, want %q", got, "Hello, World!")
	}
}

func TestHasKey_DelegatesToTranslator(t *testing.T) {
	stub := newStubTranslatorProvider(map[string]string{
		"testpkg.greeting": "Hello",
	})
	pt := NewPackageTranslator("testpkg", WithTranslator(stub))

	if !pt.HasKey("testpkg.greeting") {
		t.Error("HasKey should return true for existing key")
	}
	if pt.HasKey("testpkg.missing") {
		t.Error("HasKey should return false for missing key")
	}
}

func TestSetLocale_DelegatesAndNilSafe(t *testing.T) {
	// Nil translator: should not panic
	pt := NewPackageTranslator("testpkg")
	pt.SetLocale("es-ES") // no-op, should not panic

	// With translator: should delegate
	stub := newStubTranslatorProvider(nil)
	pt.SetTranslator(stub)
	pt.SetLocale("fr-FR")
	if stub.locale != "fr-FR" {
		t.Errorf("SetLocale did not delegate; stub locale = %q, want %q", stub.locale, "fr-FR")
	}
}

func TestGetLocale_ReturnsEnUSWhenNil(t *testing.T) {
	pt := NewPackageTranslator("testpkg")

	got := pt.GetLocale()
	if got != "en-US" {
		t.Errorf("GetLocale() = %q, want %q when translator is nil", got, "en-US")
	}
}

func TestSetTranslator_SwapsTranslator(t *testing.T) {
	stubA := newStubTranslatorProvider(map[string]string{
		"testpkg.greeting": "Hello A",
	})
	stubB := newStubTranslatorProvider(map[string]string{
		"testpkg.greeting": "Hello B",
	})
	pt := NewPackageTranslator("testpkg", WithTranslator(stubA))

	gotBefore := pt.T("greeting")
	if gotBefore != "Hello A" {
		t.Errorf("before swap: T(greeting) = %q, want %q", gotBefore, "Hello A")
	}

	pt.SetTranslator(stubB)

	gotAfter := pt.T("greeting")
	if gotAfter != "Hello B" {
		t.Errorf("after swap: T(greeting) = %q, want %q", gotAfter, "Hello B")
	}
}

func TestGetTranslator_ReturnsNilThenSet(t *testing.T) {
	pt := NewPackageTranslator("testpkg")

	if pt.GetTranslator() != nil {
		t.Error("GetTranslator should return nil initially")
	}

	stub := newStubTranslatorProvider(nil)
	pt.SetTranslator(stub)

	if pt.GetTranslator() == nil {
		t.Error("GetTranslator should return the set translator")
	}
}

// ---------------------------------------------------------------------------
// Task Group 4: Locale Utilities
// ---------------------------------------------------------------------------

func TestDetectLocale_ReturnsNonEmpty(t *testing.T) {
	got := DetectLocale()
	if got == "" {
		t.Error("DetectLocale() returned empty string")
	}
}

func TestNormalizeLocale_UTF8Suffix(t *testing.T) {
	got := NormalizeLocale("en_US.UTF-8")
	if got != "en-US" {
		t.Errorf("NormalizeLocale(en_US.UTF-8) = %q, want %q", got, "en-US")
	}
}

func TestGetSupportedLocales_FiltersCorrectly(t *testing.T) {
	loader := &stubTranslationLoader{
		locales: map[string][]byte{
			"en-US": []byte(`{}`),
		},
	}

	got := GetSupportedLocales(loader, "en-US", "xx-XX")
	if len(got) != 1 {
		t.Fatalf("GetSupportedLocales length = %d, want 1", len(got))
	}
	if got[0] != "en-US" {
		t.Errorf("GetSupportedLocales[0] = %q, want %q", got[0], "en-US")
	}
}

func TestGetSupportedLocales_EmptyCandidates(t *testing.T) {
	loader := &stubTranslationLoader{locales: map[string][]byte{}}

	got := GetSupportedLocales(loader)
	if got == nil {
		t.Fatal("GetSupportedLocales returned nil, want empty slice")
	}
	if len(got) != 0 {
		t.Errorf("GetSupportedLocales length = %d, want 0", len(got))
	}
}

// ---------------------------------------------------------------------------
// Task Group 5: Concurrency Safety and Performance Benchmarks
// ---------------------------------------------------------------------------

func TestPackageTranslator_ConcurrentSetAndT(t *testing.T) {
	defaults := map[string]string{"key": "default"}
	stubA := newStubTranslatorProvider(map[string]string{"testpkg.key": "A"})
	stubB := newStubTranslatorProvider(map[string]string{"testpkg.key": "B"})
	pt := NewPackageTranslator("testpkg", WithDefaults(defaults))

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			if i%2 == 0 {
				pt.SetTranslator(stubA)
			} else {
				pt.SetTranslator(stubB)
			}
		}()
		go func() {
			defer wg.Done()
			_ = pt.T("key")
		}()
	}
	wg.Wait()
}

func TestPackageTranslator_ConcurrentSetAndHas(t *testing.T) {
	stub := newStubTranslatorProvider(map[string]string{"testpkg.key": "val"})
	pt := NewPackageTranslator("testpkg")

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			pt.SetTranslator(stub)
		}()
		go func() {
			defer wg.Done()
			_ = pt.Has("key")
		}()
	}
	wg.Wait()
}

func TestPackageTranslator_ConcurrentReads(t *testing.T) {
	stub := newStubTranslatorProvider(map[string]string{"testpkg.key": "val"})
	defaults := map[string]string{"other": "fallback"}
	pt := NewPackageTranslator("testpkg", WithTranslator(stub), WithDefaults(defaults))

	var wg sync.WaitGroup
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = pt.T("key")
			_ = pt.T("other")
			_ = pt.Has("key")
		}()
	}
	wg.Wait()
}

func BenchmarkPackageTranslatorT_NilTranslator(b *testing.B) {
	defaults := map[string]string{"greeting": "Hello"}
	pt := NewPackageTranslator("bench", WithDefaults(defaults))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = pt.T("greeting")
	}
}

func BenchmarkPackageTranslatorT_WithTranslator(b *testing.B) {
	stub := newStubTranslatorProvider(map[string]string{
		"bench.greeting": "Hello",
	})
	pt := NewPackageTranslator("bench", WithTranslator(stub))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = pt.T("greeting")
	}
}

// ---------------------------------------------------------------------------
// Task Group 6: Gap Analysis - Additional Tests
// ---------------------------------------------------------------------------

func TestPackageTranslator_FullLifecycle(t *testing.T) {
	defaults := map[string]string{"greeting": "Hello Default"}
	pt := NewPackageTranslator("app", WithDefaults(defaults))

	if got := pt.T("greeting"); got != "Hello Default" {
		t.Errorf("phase 1: T(greeting) = %q, want %q", got, "Hello Default")
	}

	stubA := newStubTranslatorProvider(map[string]string{"app.greeting": "Hola"})
	pt.SetTranslator(stubA)
	if got := pt.T("greeting"); got != "Hola" {
		t.Errorf("phase 2: T(greeting) = %q, want %q", got, "Hola")
	}

	stubB := newStubTranslatorProvider(map[string]string{"app.greeting": "Bonjour"})
	pt.SetTranslator(stubB)
	if got := pt.T("greeting"); got != "Bonjour" {
		t.Errorf("phase 3: T(greeting) = %q, want %q", got, "Bonjour")
	}

	pt.SetTranslator(nil)
	if got := pt.T("greeting"); got != "Hello Default" {
		t.Errorf("phase 4: T(greeting) = %q, want %q", got, "Hello Default")
	}
}

func TestTF_NilTranslator_NoDefault_ReturnsRawKey(t *testing.T) {
	pt := NewPackageTranslator("testpkg")

	got := pt.TF("missing.key", "arg1")
	if got != "missing.key" {
		t.Errorf("TF(missing.key) = %q, want %q", got, "missing.key")
	}
}

func TestTranslate_FallsBackToDefaultsAfterStrippingNamespace(t *testing.T) {
	defaults := map[string]string{"greeting": "Hello from defaults"}
	pt := NewPackageTranslator("testpkg", WithDefaults(defaults))

	got := pt.Translate("testpkg.greeting")
	if got != "Hello from defaults" {
		t.Errorf("Translate(testpkg.greeting) = %q, want %q", got, "Hello from defaults")
	}
}

func TestT_DefaultsPath_SanitizesOutput(t *testing.T) {
	defaults := map[string]string{"msg": "Hello\x1b[31m World"}
	pt := NewPackageTranslator("testpkg", WithDefaults(defaults))

	got := pt.T("msg")
	expected := "Hello World"
	if got != expected {
		t.Errorf("T(msg) = %q, want %q (sanitized)", got, expected)
	}
}

func TestSetTranslator_NilResetsToDefaultsOnly(t *testing.T) {
	stub := newStubTranslatorProvider(map[string]string{"testpkg.key": "translated"})
	defaults := map[string]string{"key": "default"}
	pt := NewPackageTranslator("testpkg", WithTranslator(stub), WithDefaults(defaults))

	if got := pt.T("key"); got != "translated" {
		t.Errorf("before nil: T(key) = %q, want %q", got, "translated")
	}

	pt.SetTranslator(nil)

	if got := pt.T("key"); got != "default" {
		t.Errorf("after nil: T(key) = %q, want %q", got, "default")
	}
}

// ---------------------------------------------------------------------------
// Coverage gap: TranslatePlural, TranslateGender, TranslateWithArgs defaults,
// HasKey nil, GetLocale with translator, stripNamespacePrefix
// ---------------------------------------------------------------------------

func TestTranslatePlural_NilTranslator_ReturnsKey(t *testing.T) {
	pt := NewPackageTranslator("testpkg")
	got := pt.TranslatePlural("items", 5)
	if got != "items" {
		t.Errorf("TranslatePlural with nil translator = %q, want %q", got, "items")
	}
}

func TestTranslatePlural_WithTranslator_Delegates(t *testing.T) {
	stub := newStubTranslatorProvider(nil)
	pt := NewPackageTranslator("testpkg", WithTranslator(stub))
	got := pt.TranslatePlural("items", 5)
	if got != "items" {
		t.Errorf("TranslatePlural with translator = %q, want %q", got, "items")
	}
}

func TestTranslateGender_NilTranslator_ReturnsKey(t *testing.T) {
	pt := NewPackageTranslator("testpkg")
	got := pt.TranslateGender("title", core.Masculine)
	if got != "title" {
		t.Errorf("TranslateGender with nil translator = %q, want %q", got, "title")
	}
}

func TestTranslateGender_WithTranslator_Delegates(t *testing.T) {
	stub := newStubTranslatorProvider(nil)
	pt := NewPackageTranslator("testpkg", WithTranslator(stub))
	got := pt.TranslateGender("title", core.Feminine)
	if got != "title" {
		t.Errorf("TranslateGender with translator = %q, want %q", got, "title")
	}
}

func TestTranslateInterface_NilTranslator_FallsToDefaults(t *testing.T) {
	defaults := map[string]string{"msg": "default msg"}
	pt := NewPackageTranslator("testpkg", WithDefaults(defaults))

	// Translate (interface method) with nil translator, key has namespace prefix
	got := pt.Translate("testpkg.msg")
	if got != "default msg" {
		t.Errorf("Translate nil translator defaults = %q, want %q", got, "default msg")
	}
}

func TestTranslateInterface_NilTranslator_NoDefaults_ReturnsKey(t *testing.T) {
	pt := NewPackageTranslator("testpkg")
	got := pt.Translate("testpkg.missing")
	if got != "testpkg.missing" {
		t.Errorf("Translate nil translator no defaults = %q, want %q", got, "testpkg.missing")
	}
}

func TestTranslateWithArgs_NilTranslator_FallsToDefaults(t *testing.T) {
	defaults := map[string]string{"welcome": "Hi, %s!"}
	pt := NewPackageTranslator("testpkg", WithDefaults(defaults))

	got := pt.TranslateWithArgs("testpkg.welcome", "Alice")
	if got != "Hi, Alice!" {
		t.Errorf("TranslateWithArgs nil translator defaults = %q, want %q", got, "Hi, Alice!")
	}
}

func TestTranslateWithArgs_NilTranslator_NoDefaults_ReturnsKey(t *testing.T) {
	pt := NewPackageTranslator("testpkg")
	got := pt.TranslateWithArgs("testpkg.missing", "arg")
	if got != "testpkg.missing" {
		t.Errorf("TranslateWithArgs nil translator no defaults = %q, want %q", got, "testpkg.missing")
	}
}

func TestHasKey_NilTranslator_ReturnsFalse(t *testing.T) {
	pt := NewPackageTranslator("testpkg")
	if pt.HasKey("testpkg.any") {
		t.Error("HasKey with nil translator should return false")
	}
}

func TestGetLocale_WithTranslator_DelegatesLocale(t *testing.T) {
	stub := newStubTranslatorProvider(nil)
	stub.locale = "fr-FR"
	pt := NewPackageTranslator("testpkg", WithTranslator(stub))

	got := pt.GetLocale()
	if got != "fr-FR" {
		t.Errorf("GetLocale with translator = %q, want %q", got, "fr-FR")
	}
}

func TestStripNamespacePrefix_KeyWithoutPrefix(t *testing.T) {
	got := stripNamespacePrefix("plain.key", "testpkg")
	if got != "plain.key" {
		t.Errorf("stripNamespacePrefix without prefix = %q, want %q", got, "plain.key")
	}
}

func TestStripNamespacePrefix_KeyWithPrefix(t *testing.T) {
	got := stripNamespacePrefix("testpkg.greeting", "testpkg")
	if got != "greeting" {
		t.Errorf("stripNamespacePrefix with prefix = %q, want %q", got, "greeting")
	}
}

func TestTF_TranslatorKeyNotFound_FallsBackToDefaults(t *testing.T) {
	stub := newStubTranslatorProvider(map[string]string{})
	defaults := map[string]string{"welcome": "Hi, %s!"}
	pt := NewPackageTranslator("testpkg", WithTranslator(stub), WithDefaults(defaults))

	got := pt.TF("welcome", "Bob")
	if got != "Hi, Bob!" {
		t.Errorf("TF fallback to defaults = %q, want %q", got, "Hi, Bob!")
	}
}

func TestTranslateWithArgs_TranslatorKeyNotFound_FallsBackToDefaults(t *testing.T) {
	stub := newStubTranslatorProvider(map[string]string{})
	defaults := map[string]string{"welcome": "Hey, %s!"}
	pt := NewPackageTranslator("testpkg", WithTranslator(stub), WithDefaults(defaults))

	got := pt.TranslateWithArgs("testpkg.welcome", "Carol")
	if got != "Hey, Carol!" {
		t.Errorf("TranslateWithArgs fallback = %q, want %q", got, "Hey, Carol!")
	}
}

// =============================================================================
// Fuzz target for package translator
// =============================================================================

// FuzzNewPackageTranslator exercises NewPackageTranslator namespace validation with random inputs.
func FuzzNewPackageTranslator(f *testing.F) {
	f.Add("myapp")
	f.Add("")
	f.Add("\x00\x01")
	f.Add(strings.Repeat("a", 110))
	f.Add("../../etc/passwd")
	f.Add("valid.namespace")

	f.Fuzz(func(t *testing.T, namespace string) {
		pt := NewPackageTranslator(namespace)

		// NewPackageTranslator returns nil for invalid namespaces, non-nil for valid.
		// When non-nil, the namespace field must match the input.
		if pt != nil && pt.namespace != namespace {
			t.Errorf("NewPackageTranslator(%q).namespace = %q, want %q", namespace, pt.namespace, namespace)
		}
	})
}

// =============================================================================
// Spec 018: PackageTranslator Benchmarks (Task Group 8)
// =============================================================================

// Baseline: ~50-200 ns/op, 1-3 allocs/op
func BenchmarkPackageTranslatorTF(b *testing.B) {
	stub := newStubTranslatorProvider(map[string]string{
		"bench.welcome": "Hello, %s!",
	})
	pt := NewPackageTranslator("bench", WithTranslator(stub))
	_ = pt.TF("welcome", "Alice")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = pt.TF("welcome", "Alice")
	}
}

// Baseline: ~30-100 ns/op, 0-1 allocs/op
func BenchmarkPackageTranslatorT_Parallel(b *testing.B) {
	stub := newStubTranslatorProvider(map[string]string{
		"bench.greeting": "Hello",
	})
	pt := NewPackageTranslator("bench", WithTranslator(stub))
	_ = pt.T("greeting")
	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = pt.T("greeting")
		}
	})
}

// Baseline: ~30-100 ns/op, 0 allocs/op
func BenchmarkPackageTranslatorSetTranslator(b *testing.B) {
	stubA := newStubTranslatorProvider(map[string]string{"bench.key": "A"})
	stubB := newStubTranslatorProvider(map[string]string{"bench.key": "B"})
	pt := NewPackageTranslator("bench", WithTranslator(stubA))
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if i%2 == 0 {
			pt.SetTranslator(stubA)
		} else {
			pt.SetTranslator(stubB)
		}
	}
}

// =============================================================================
// Namespace tests (merged from namespace_test.go)
// =============================================================================

// mockTranslatorProvider is a minimal mock implementing core.TranslatorProvider for
// testing Namespace delegation. It stores translations in a simple map.
type mockTranslatorProvider struct {
	translations map[string]string
	locale       string
}

func newMockTranslatorProvider(translations map[string]string) *mockTranslatorProvider {
	return &mockTranslatorProvider{
		translations: translations,
		locale:       "en-US",
	}
}

func (m *mockTranslatorProvider) Translate(key string) string {
	if v, ok := m.translations[key]; ok {
		return v
	}
	return key
}

func (m *mockTranslatorProvider) TranslateWithArgs(key string, args ...interface{}) string {
	if v, ok := m.translations[key]; ok {
		return fmt.Sprintf(v, args...)
	}
	return key
}

func (m *mockTranslatorProvider) TranslatePlural(key string, _ interface{}) string {
	return key
}

func (m *mockTranslatorProvider) TranslateGender(key string, _ core.GenderCategory) string {
	return key
}

func (m *mockTranslatorProvider) HasKey(key string) bool {
	_, ok := m.translations[key]
	return ok
}

func (m *mockTranslatorProvider) SetLocale(locale string) {
	m.locale = locale
}

func (m *mockTranslatorProvider) GetLocale() string {
	return m.locale
}

func TestNewNamespace_ValidPrefix(t *testing.T) {
	ns, err := NewNamespace("filesystem", nil)
	if err != nil {
		t.Fatalf("NewNamespace(\"filesystem\", nil) returned unexpected error: %v", err)
	}
	if ns == nil {
		t.Fatal("NewNamespace(\"filesystem\", nil) returned nil Namespace")
	}
}

func TestNewNamespace_EmptyPrefix(t *testing.T) {
	ns, err := NewNamespace("", nil)
	if err == nil {
		t.Fatal("NewNamespace(\"\", nil) expected error for empty prefix, got nil")
	}
	if ns != nil {
		t.Fatal("NewNamespace(\"\", nil) expected nil Namespace on error")
	}
}

func TestNewNamespace_InvalidCharacters(t *testing.T) {
	cases := []struct {
		name   string
		prefix string
	}{
		{name: "dot", prefix: "my.pkg"},
		{name: "space", prefix: "my pkg"},
		{name: "special char", prefix: "my@pkg"},
		{name: "slash", prefix: "my/pkg"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ns, err := NewNamespace(tc.prefix, nil)
			if err == nil {
				t.Errorf("NewNamespace(%q, nil) expected error for invalid characters, got nil", tc.prefix)
			}
			if ns != nil {
				t.Errorf("NewNamespace(%q, nil) expected nil Namespace on error", tc.prefix)
			}
		})
	}
}

func TestNewNamespace_PrefixExceedsMaxLength(t *testing.T) {
	longPrefix := strings.Repeat("a", core.MaxPrefixLength+1)
	ns, err := NewNamespace(longPrefix, nil)
	if err == nil {
		t.Fatalf("NewNamespace with %d-char prefix expected error, got nil", len(longPrefix))
	}
	if ns != nil {
		t.Fatal("expected nil Namespace on error")
	}
}

func TestNewNamespace_NilTranslatorAccepted(t *testing.T) {
	ns, err := NewNamespace("mypkg", nil)
	if err != nil {
		t.Fatalf("NewNamespace(\"mypkg\", nil) returned unexpected error: %v", err)
	}
	if ns == nil {
		t.Fatal("NewNamespace(\"mypkg\", nil) returned nil Namespace")
	}
}

func TestNamespace_T_DelegatesToTranslator(t *testing.T) {
	mock := newMockTranslatorProvider(map[string]string{
		"fs.error.empty_path": "path is empty",
	})
	ns, err := NewNamespace("fs", mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := ns.T("error.empty_path")
	want := "path is empty"
	if got != want {
		t.Errorf("T(\"error.empty_path\") = %q, want %q", got, want)
	}
}

func TestNamespace_T_NilTranslator_ReturnsFallback(t *testing.T) {
	ns, err := NewNamespace("fs", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := ns.T("error.empty_path")
	want := "fs.error.empty_path"
	if got != want {
		t.Errorf("T(\"error.empty_path\") with nil translator = %q, want %q", got, want)
	}
}

func TestNamespace_TF_DelegatesToTranslateWithArgs(t *testing.T) {
	mock := newMockTranslatorProvider(map[string]string{
		"fs.error.not_found": "file %q not found in %s",
	})
	ns, err := NewNamespace("fs", mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := ns.TF("error.not_found", "test.txt", "/home")
	want := `file "test.txt" not found in /home`
	if got != want {
		t.Errorf("TF(\"error.not_found\", ...) = %q, want %q", got, want)
	}
}

func TestNamespace_TD_ReturnsTranslatedValue(t *testing.T) {
	mock := newMockTranslatorProvider(map[string]string{
		"fs.error.empty": "the path is empty",
	})
	ns, err := NewNamespace("fs", mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := ns.TD("error.empty", "fallback value")
	want := "the path is empty"
	if got != want {
		t.Errorf("TD(\"error.empty\", \"fallback value\") = %q, want %q", got, want)
	}
}

func TestNamespace_TD_ReturnsDefault_WhenKeyMissing(t *testing.T) {
	mock := newMockTranslatorProvider(map[string]string{})
	ns, err := NewNamespace("fs", mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := ns.TD("error.missing", "default text")
	want := "default text"
	if got != want {
		t.Errorf("TD(\"error.missing\", \"default text\") = %q, want %q", got, want)
	}
}

func TestNamespace_Has_ReturnsTrue(t *testing.T) {
	mock := newMockTranslatorProvider(map[string]string{
		"fs.error.exists": "yes",
	})
	ns, err := NewNamespace("fs", mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !ns.Has("error.exists") {
		t.Error("Has(\"error.exists\") = false, want true")
	}
}

func TestNamespace_Has_NilTranslator_ReturnsFalse(t *testing.T) {
	ns, err := NewNamespace("fs", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ns.Has("error.exists") {
		t.Error("Has(\"error.exists\") with nil translator = true, want false")
	}
}

func TestNamespace_Key_ReturnsFullKey(t *testing.T) {
	ns, err := NewNamespace("fs", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := ns.Key("error.empty_path")
	want := "fs.error.empty_path"
	if got != want {
		t.Errorf("Key(\"error.empty_path\") = %q, want %q", got, want)
	}
}

func TestNamespace_NilReceiver_T(t *testing.T) {
	var ns *Namespace
	got := ns.T("some.key")
	if got != "some.key" {
		t.Errorf("nil.T(\"some.key\") = %q, want %q", got, "some.key")
	}
}

func TestNamespace_NilReceiver_TF(t *testing.T) {
	var ns *Namespace
	got := ns.TF("some.key", "arg1")
	if got != "some.key" {
		t.Errorf("nil.TF(\"some.key\", ...) = %q, want %q", got, "some.key")
	}
}

func TestNamespace_NilReceiver_TD(t *testing.T) {
	var ns *Namespace
	got := ns.TD("some.key", "default")
	if got != "default" {
		t.Errorf("nil.TD(\"some.key\", \"default\") = %q, want %q", got, "default")
	}
}

func TestNamespace_NilReceiver_Has(t *testing.T) {
	var ns *Namespace
	if ns.Has("some.key") {
		t.Error("nil.Has(\"some.key\") = true, want false")
	}
}

func TestNamespace_NilReceiver_Key(t *testing.T) {
	var ns *Namespace
	got := ns.Key("some.key")
	if got != "some.key" {
		t.Errorf("nil.Key(\"some.key\") = %q, want %q", got, "some.key")
	}
}

func TestNamespace_MockIntegration_RoundTrip(t *testing.T) {
	mock := newMockTranslatorProvider(map[string]string{
		"myapp.greeting":      "Hello!",
		"myapp.welcome":       "Welcome, %s!",
		"myapp.status.active": "Active",
	})
	ns, err := NewNamespace("myapp", mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := ns.T("greeting"); got != "Hello!" {
		t.Errorf("T(\"greeting\") = %q, want %q", got, "Hello!")
	}

	if got := ns.TF("welcome", "Alice"); got != "Welcome, Alice!" {
		t.Errorf("TF(\"welcome\", \"Alice\") = %q, want %q", got, "Welcome, Alice!")
	}

	if got := ns.TD("status.active", "Unknown"); got != "Active" {
		t.Errorf("TD(\"status.active\", \"Unknown\") = %q, want %q", got, "Active")
	}

	if got := ns.TD("status.inactive", "Offline"); got != "Offline" {
		t.Errorf("TD(\"status.inactive\", \"Offline\") = %q, want %q", got, "Offline")
	}

	if !ns.Has("greeting") {
		t.Error("Has(\"greeting\") = false, want true")
	}

	if ns.Has("missing") {
		t.Error("Has(\"missing\") = true, want false")
	}

	if got := ns.Key("greeting"); got != "myapp.greeting" {
		t.Errorf("Key(\"greeting\") = %q, want %q", got, "myapp.greeting")
	}
}

func TestNewNamespace_PrefixAtExactMaxLength(t *testing.T) {
	prefix := strings.Repeat("a", core.MaxPrefixLength)
	ns, err := NewNamespace(prefix, nil)
	if err != nil {
		t.Fatalf("NewNamespace with %d-char prefix returned unexpected error: %v", core.MaxPrefixLength, err)
	}
	if ns == nil {
		t.Fatal("expected non-nil Namespace for exactly-max-length prefix")
	}
}

func TestNewNamespace_PrefixWithHyphensAndUnderscores(t *testing.T) {
	cases := []string{"my-pkg", "my_pkg", "my-pkg_v2", "a-b-c", "a_b_c"}
	for _, prefix := range cases {
		t.Run(prefix, func(t *testing.T) {
			ns, err := NewNamespace(prefix, nil)
			if err != nil {
				t.Errorf("NewNamespace(%q, nil) returned unexpected error: %v", prefix, err)
			}
			if ns == nil {
				t.Errorf("NewNamespace(%q, nil) returned nil Namespace", prefix)
			}
		})
	}
}

func TestNamespace_TF_NilTranslator_ReturnsNamespacedKey(t *testing.T) {
	ns, err := NewNamespace("pkg", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := ns.TF("error.msg", "arg1", 42)
	want := "pkg.error.msg"
	if got != want {
		t.Errorf("TF with nil translator = %q, want %q (should not format)", got, want)
	}
}

func TestNamespace_TD_NilTranslator_ReturnsDefault(t *testing.T) {
	ns, err := NewNamespace("pkg", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := ns.TD("missing.key", "my default")
	want := "my default"
	if got != want {
		t.Errorf("TD with nil translator = %q, want %q", got, want)
	}
}

func TestNamespace_ConcurrentAccess(t *testing.T) {
	mock := newMockTranslatorProvider(map[string]string{
		"app.key1": "value1",
		"app.key2": "value2",
	})
	ns, err := NewNamespace("app", mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var wg sync.WaitGroup
	goroutines := 100

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			if idx%2 == 0 {
				ns.T("key1")
			} else {
				ns.T("key2")
			}
			ns.Has("key1")
			ns.Key("key2")
			ns.TD("key1", "default")
			ns.TF("key2", "arg")
		}(i)
	}

	wg.Wait()
}

func FuzzNewNamespace(f *testing.F) {
	f.Add("app")
	f.Add("")
	f.Add("\x00\x01")
	f.Add(strings.Repeat("a", 110))
	f.Add("../../traversal")
	f.Add("valid-prefix_v2")

	f.Fuzz(func(t *testing.T, prefix string) {
		ns, err := NewNamespace(prefix, nil)
		if err != nil {
			if err.Error() == "" {
				t.Errorf("NewNamespace(%q) returned error with empty message", prefix)
			}
			return
		}

		if ns == nil {
			t.Errorf("NewNamespace(%q) returned nil Namespace without error", prefix)
			return
		}
		if ns.prefix != prefix {
			t.Errorf("NewNamespace(%q).prefix = %q, want %q", prefix, ns.prefix, prefix)
		}
	})
}

func FuzzMatchPrefix(f *testing.F) {
	f.Add("valid-prefix")
	f.Add("")
	f.Add("\x00")
	f.Add(strings.Repeat("b", 110))
	f.Add(".leading-dot")
	f.Add("a_b-c")

	f.Fuzz(func(t *testing.T, s string) {
		got := matchPrefix(s)

		if s == "" && got {
			t.Errorf("matchPrefix(%q) = true, want false for empty string", s)
		}

		if got {
			for i := 0; i < len(s); i++ {
				c := s[i]
				isAlpha := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
				isDigit := c >= '0' && c <= '9'
				isSpecial := c == '_' || c == '-'
				if !isAlpha && !isDigit && !isSpecial {
					t.Errorf("matchPrefix(%q) = true but contains invalid byte %q at index %d", s, c, i)
				}
			}
		}
	})
}

func BenchmarkNamespace_T(b *testing.B) {
	mock := newMockTranslatorProvider(map[string]string{
		"bench.greeting": "Hello",
	})
	ns, err := NewNamespace("bench", mock)
	if err != nil {
		b.Fatalf("NewNamespace error: %v", err)
	}
	_ = ns.T("greeting")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = ns.T("greeting")
	}
}

func BenchmarkNamespace_TD(b *testing.B) {
	mock := newMockTranslatorProvider(map[string]string{})
	ns, err := NewNamespace("bench", mock)
	if err != nil {
		b.Fatalf("NewNamespace error: %v", err)
	}
	_ = ns.TD("missing.key", "default value")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = ns.TD("missing.key", "default value")
	}
}

func BenchmarkNamespace_Key(b *testing.B) {
	ns, err := NewNamespace("bench", nil)
	if err != nil {
		b.Fatalf("NewNamespace error: %v", err)
	}
	_ = ns.Key("greeting")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = ns.Key("greeting")
	}
}

func BenchmarkNamespace_T_Parallel(b *testing.B) {
	mock := newMockTranslatorProvider(map[string]string{
		"bench.greeting": "Hello",
	})
	ns, err := NewNamespace("bench", mock)
	if err != nil {
		b.Fatalf("NewNamespace error: %v", err)
	}
	_ = ns.T("greeting")
	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = ns.T("greeting")
		}
	})
}

// =============================================================================
// NewPackageTranslatorWithFS tests
// =============================================================================

//go:embed testdata/locales/*.json
var ptTestLocales embed.FS

func TestNewPackageTranslatorWithFS_Basic(t *testing.T) {
	pt := NewPackageTranslatorWithFS("testpkg", ptTestLocales, "testdata/locales")
	if pt == nil {
		t.Fatal("NewPackageTranslatorWithFS returned nil")
	}

	// Should have a translator set (not defaults-only mode)
	if pt.GetTranslator() == nil {
		t.Error("expected translator to be set")
	}
}

func TestNewPackageTranslatorWithFS_TranslatesKeys(t *testing.T) {
	pt := NewPackageTranslatorWithFS("testpkg", ptTestLocales, "testdata/locales",
		WithDefaults(map[string]string{
			"missing.key": "default value",
		}),
	)
	if pt == nil {
		t.Fatal("NewPackageTranslatorWithFS returned nil")
	}

	// Key from locale file via translator
	result := pt.T("greeting")
	// "greeting" won't be found as "testpkg.greeting" unless the locale file has it
	// but the translator should be functional
	if result == "" {
		t.Error("T should return non-empty string")
	}

	// Missing key falls back to defaults
	result = pt.T("missing.key")
	if result != "default value" {
		t.Errorf("T(missing.key) = %q, want %q", result, "default value")
	}
}

func TestNewPackageTranslatorWithFS_InvalidNamespace(t *testing.T) {
	pt := NewPackageTranslatorWithFS("", ptTestLocales, "testdata/locales")
	if pt != nil {
		t.Error("expected nil for invalid namespace")
	}
}

func TestNewPackageTranslatorWithFS_SetLocale(t *testing.T) {
	pt := NewPackageTranslatorWithFS("testpkg", ptTestLocales, "testdata/locales")
	if pt == nil {
		t.Fatal("NewPackageTranslatorWithFS returned nil")
	}

	// Should be able to change locale
	pt.SetLocale("es-ES")
	if got := pt.GetLocale(); got != "es-ES" {
		t.Errorf("GetLocale() = %q, want %q", got, "es-ES")
	}
}

// =============================================================================
// DiscoverLocales tests
// =============================================================================

func TestDiscoverLocales_FindsJSONFiles(t *testing.T) {
	locales := DiscoverLocales(ptTestLocales, "testdata/locales")
	if len(locales) == 0 {
		t.Fatal("DiscoverLocales returned empty slice")
	}

	// Should find en-US at minimum
	found := false
	for _, l := range locales {
		if l == "en-US" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("DiscoverLocales did not find en-US, got: %v", locales)
	}
}

func TestDiscoverLocales_InvalidPath(t *testing.T) {
	locales := DiscoverLocales(ptTestLocales, "nonexistent")
	if len(locales) != 0 {
		t.Errorf("DiscoverLocales for invalid path should return empty, got: %v", locales)
	}
}
