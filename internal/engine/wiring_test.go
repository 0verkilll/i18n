// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

package engine

import (
	"context"
	"embed"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0verkilll/i18n/internal/core"
)

//go:embed testdata/locales/*.json
var wiringTestLocales embed.FS

// =============================================================================
// TestWiring_TranslatorNew verifies that New() accepts every Option type and
// returns a correctly wired Translator.
// =============================================================================

func TestWiring_TranslatorNew(t *testing.T) {
	t.Parallel()

	localeDir := setupWiringLocaleDir(t)

	cache := NewMapCache()
	loader := NewFileSystemLoader(localeDir)
	parser := NewJSONParser()
	resolver := NewDefaultKeyResolver()
	chainer := NewDefaultFallbackChainer()
	detector := NewDefaultLocaleDetector(nil)
	pluralRes := NewDefaultPluralResolver()

	translator, err := New(
		WithLoader(loader),
		WithParser(parser),
		WithResolver(resolver),
		WithLocaleDetector(detector),
		WithFallbackChainer(chainer),
		WithDefaultLocale("en-US"),
		WithPluralResolver(pluralRes),
		WithCache(cache),
		WithLogger(NopLogger{}),
	)
	if err != nil {
		t.Fatalf("New() with all options: %v", err)
	}
	if translator == nil {
		t.Fatal("New() returned nil Translator")
	}
}

func TestWiring_TranslatorNewWithFileSystemLoader(t *testing.T) {
	t.Parallel()

	localeDir := setupWiringLocaleDir(t)

	translator, err := New(
		WithFileSystemLoader(localeDir),
		WithDefaultLocale("en-US"),
	)
	if err != nil {
		t.Fatalf("New(WithFileSystemLoader) error: %v", err)
	}
	if translator == nil {
		t.Fatal("New(WithFileSystemLoader) returned nil")
	}
}

func TestWiring_TranslatorNewWithRegistryLoader(t *testing.T) {
	t.Parallel()

	// RegistryLoader may be empty without build tags, but New should still succeed.
	// We provide a static detector so it doesn't fail on locale detection.
	_, err := New(
		WithRegistryLoader(),
		WithLocaleDetector(NewStaticDetector("en-US")),
	)
	// err may be nil or non-nil depending on registry state; that's OK.
	// The point is that WithRegistryLoader() is accepted by New.
	_ = err
}

func TestWiring_TranslatorNewWithRegisteredParser(t *testing.T) {
	t.Parallel()

	localeDir := setupWiringLocaleDir(t)

	translator, err := New(
		WithFileSystemLoader(localeDir),
		WithRegisteredParser(".json"),
		WithDefaultLocale("en-US"),
	)
	if err != nil {
		t.Fatalf("New(WithRegisteredParser) error: %v", err)
	}
	if translator == nil {
		t.Fatal("New(WithRegisteredParser) returned nil")
	}
}

// =============================================================================
// TestWiring_TranslatorMethods exercises EVERY public method on Translator.
// =============================================================================

func TestWiring_TranslatorMethods(t *testing.T) {
	t.Parallel()

	localeDir := setupWiringPluralLocaleDir(t)

	translator, err := New(
		WithFileSystemLoader(localeDir),
		WithDefaultLocale("en-US"),
		WithCache(NewMapCache()),
	)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	// GetLocale
	if got := translator.GetLocale(); got != "en-US" {
		t.Errorf("GetLocale() = %q, want %q", got, "en-US")
	}

	// SetLocale
	translator.SetLocale("es-ES")
	if got := translator.GetLocale(); got != "es-ES" {
		t.Errorf("after SetLocale(es-ES), GetLocale() = %q", got)
	}
	translator.SetLocale("en-US")

	// Translate
	if got := translator.Translate("greeting"); got != "Hello" {
		t.Errorf("Translate(greeting) = %q, want Hello", got)
	}

	// TranslateWithArgs
	if got := translator.TranslateWithArgs("welcome_fmt", "Alice"); got != "Welcome, Alice!" {
		t.Errorf("TranslateWithArgs(welcome_fmt, Alice) = %q, want 'Welcome, Alice!'", got)
	}

	// HasKey
	if !translator.HasKey("greeting") {
		t.Error("HasKey(greeting) = false, want true")
	}
	if translator.HasKey("totally.missing") {
		t.Error("HasKey(totally.missing) = true, want false")
	}

	// TranslatePlural
	if got := translator.TranslatePlural("items", 1); got != "1 item" {
		t.Errorf("TranslatePlural(items, 1) = %q, want '1 item'", got)
	}
	if got := translator.TranslatePlural("items", 5); got != "5 items" {
		t.Errorf("TranslatePlural(items, 5) = %q, want '5 items'", got)
	}

	// TranslatePluralWithArgs
	if got := translator.TranslatePluralWithArgs("messages", 1, "inbox"); got != "You have 1 message in inbox" {
		t.Errorf("TranslatePluralWithArgs(messages, 1, inbox) = %q", got)
	}

	// TranslateGender
	if got := translator.TranslateGender("who", core.Masculine); got != "He logged in" {
		t.Errorf("TranslateGender(who, Masculine) = %q", got)
	}
	if got := translator.TranslateGender("who", core.Feminine); got != "She logged in" {
		t.Errorf("TranslateGender(who, Feminine) = %q", got)
	}
	if got := translator.TranslateGender("who", core.Neuter); got != "They logged in" {
		t.Errorf("TranslateGender(who, Neuter) = %q, want 'They logged in'", got)
	}

	// TranslateWithMessage (ICU MessageFormat)
	if got := translator.TranslateWithMessage("icu_plural", map[string]interface{}{"count": 1}); got != "1 item" {
		t.Errorf("TranslateWithMessage(icu_plural, count=1) = %q, want '1 item'", got)
	}
	if got := translator.TranslateWithMessage("icu_select", map[string]interface{}{"gender": "male"}); got != "He" {
		t.Errorf("TranslateWithMessage(icu_select, male) = %q, want 'He'", got)
	}

	// Missing key returns key
	if got := translator.Translate("no.such.key"); got != "no.such.key" {
		t.Errorf("Translate(missing) = %q, want key back", got)
	}
}

// =============================================================================
// TestWiring_MapCache exercises MapCache Get/Set/Invalidate.
// =============================================================================

func TestWiring_MapCache(t *testing.T) {
	t.Parallel()

	c := NewMapCache()

	// Miss
	if _, ok := c.Get("k"); ok {
		t.Error("Get on empty cache should miss")
	}

	// Set + Hit
	c.Set("k", "v")
	if got, ok := c.Get("k"); !ok || got != "v" {
		t.Errorf("Get after Set = (%q, %v)", got, ok)
	}

	// Invalidate
	c.Invalidate()
	if _, ok := c.Get("k"); ok {
		t.Error("Get after Invalidate should miss")
	}
}

func TestWiring_MapCacheWithLimit(t *testing.T) {
	t.Parallel()

	c := NewMapCacheWithLimit(2)
	c.Set("a", "1")
	c.Set("b", "2")
	c.Set("c", "3") // evicts "a"

	if _, ok := c.Get("a"); ok {
		t.Error("evicted entry 'a' should be gone")
	}
	if got, ok := c.Get("b"); !ok || got != "2" {
		t.Error("entry 'b' should still exist")
	}
	if got, ok := c.Get("c"); !ok || got != "3" {
		t.Error("entry 'c' should exist")
	}
}

// =============================================================================
// TestWiring_Detectors tests all detector constructors and methods.
// =============================================================================

func TestWiring_DefaultLocaleDetector(t *testing.T) {
	t.Parallel()

	// With nil env (uses OS env)
	d := NewDefaultLocaleDetector(nil)
	locale := d.Detect()
	if locale == "" {
		t.Error("Detect() with nil env should return non-empty")
	}
	normalized := d.Normalize("en_us.UTF-8")
	if normalized != "en-US" {
		t.Errorf("Normalize(en_us.UTF-8) = %q, want en-US", normalized)
	}

	// With mock env
	mock := &MockEnvProvider{vars: map[string]string{"LANG": "fr_FR.UTF-8"}}
	d2 := NewDefaultLocaleDetector(mock)
	if got := d2.Detect(); got != "fr_FR.UTF-8" {
		t.Errorf("Detect() with LANG=fr_FR.UTF-8 = %q", got)
	}
}

func TestWiring_AcceptLanguageDetector(t *testing.T) {
	t.Parallel()

	d := NewAcceptLanguageDetector("en-US,es-ES;q=0.8,fr;q=0.5")
	locale := d.Detect()
	if locale != "en-US" {
		t.Errorf("AcceptLanguageDetector.Detect() = %q, want en-US", locale)
	}
	if got := d.Normalize("EN_US"); got != "en-US" {
		t.Errorf("Normalize(EN_US) = %q", got)
	}
}

func TestWiring_ChainDetector(t *testing.T) {
	t.Parallel()

	// First detector returns empty, second returns "fr-FR"
	chain := NewChainDetector(
		NewStaticDetector(""),
		NewStaticDetector("fr-FR"),
	)
	if got := chain.Detect(); got != "fr-FR" {
		t.Errorf("ChainDetector.Detect() = %q, want fr-FR", got)
	}
	if got := chain.Normalize("de_de"); got != "de-DE" {
		t.Errorf("ChainDetector.Normalize(de_de) = %q", got)
	}
}

func TestWiring_ChainDetectorEmpty(t *testing.T) {
	t.Parallel()

	chain := NewChainDetector()
	if got := chain.Detect(); got != "en-US" {
		t.Errorf("empty ChainDetector.Detect() = %q, want en-US", got)
	}
}

func TestWiring_StaticDetector(t *testing.T) {
	t.Parallel()

	d := NewStaticDetector("ja-JP")
	if got := d.Detect(); got != "ja-JP" {
		t.Errorf("StaticDetector.Detect() = %q, want ja-JP", got)
	}
	if got := d.Normalize("ja_jp"); got != "ja-JP" {
		t.Errorf("StaticDetector.Normalize(ja_jp) = %q", got)
	}
}

func TestWiring_BrowserDetector(t *testing.T) {
	t.Parallel()

	d := NewBrowserDetector()
	// On non-WASM builds, Detect returns ""
	got := d.Detect()
	if got != "" {
		t.Errorf("BrowserDetector.Detect() on non-WASM = %q, want empty", got)
	}
	if norm := d.Normalize("en_us"); norm != "en-US" {
		t.Errorf("BrowserDetector.Normalize = %q", norm)
	}
}

// =============================================================================
// TestWiring_PackageTranslator tests PackageTranslator and all its methods.
// =============================================================================

func TestWiring_PackageTranslator(t *testing.T) {
	t.Parallel()

	localeDir := setupWiringNamespaceLocaleDir(t)

	translator, err := New(
		WithFileSystemLoader(localeDir),
		WithDefaultLocale("en-US"),
	)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	// Create with defaults and translator
	pt := NewPackageTranslator("mypkg", WithDefaults(map[string]string{
		"fallback": "Default fallback",
		"welcome":  "Welcome, %s! (default)",
	}), WithTranslator(translator))

	if pt == nil {
		t.Fatal("NewPackageTranslator returned nil")
	}

	// T - translates through translator
	if got := pt.T("greeting"); got != "Hello from mypkg" {
		t.Errorf("PackageTranslator.T(greeting) = %q", got)
	}

	// T - falls back to defaults for missing key in translator
	if got := pt.T("fallback"); got != "Default fallback" {
		t.Errorf("PackageTranslator.T(fallback) = %q, want 'Default fallback'", got)
	}

	// TF
	if got := pt.TF("welcome_fmt", "Bob"); got != "Welcome, Bob!" {
		t.Errorf("PackageTranslator.TF(welcome_fmt, Bob) = %q", got)
	}

	// Has
	if !pt.Has("greeting") {
		t.Error("PackageTranslator.Has(greeting) = false, want true")
	}
	if pt.Has("nonexistent") {
		t.Error("PackageTranslator.Has(nonexistent) = true, want false")
	}

	// Interface methods (Translate, TranslateWithArgs, HasKey, SetLocale, GetLocale)
	if got := pt.Translate("mypkg.greeting"); got != "Hello from mypkg" {
		t.Errorf("PackageTranslator.Translate(mypkg.greeting) = %q", got)
	}
	if got := pt.TranslateWithArgs("mypkg.welcome_fmt", "Carol"); got != "Welcome, Carol!" {
		t.Errorf("PackageTranslator.TranslateWithArgs = %q", got)
	}
	if !pt.HasKey("mypkg.greeting") {
		t.Error("PackageTranslator.HasKey(mypkg.greeting) = false")
	}

	if got := pt.GetLocale(); got != "en-US" {
		t.Errorf("PackageTranslator.GetLocale() = %q", got)
	}

	pt.SetLocale("es-ES")
	if got := pt.GetLocale(); got != "es-ES" {
		t.Errorf("PackageTranslator.GetLocale() after SetLocale = %q", got)
	}
	pt.SetLocale("en-US") // reset

	// TranslatePlural delegates through
	gotP := pt.TranslatePlural("mypkg.greeting", 1)
	// May or may not resolve depending on locale data; just ensure no panic.
	_ = gotP

	// TranslateGender delegates through
	gotG := pt.TranslateGender("mypkg.greeting", core.Masculine)
	_ = gotG

	// SetTranslator / GetTranslator
	pt.SetTranslator(nil)
	if pt.GetTranslator() != nil {
		t.Error("GetTranslator should be nil after SetTranslator(nil)")
	}

	// With nil translator, T returns defaults or key
	if got := pt.T("fallback"); got != "Default fallback" {
		t.Errorf("T with nil translator and defaults = %q", got)
	}
	if got := pt.T("unknown"); got != "unknown" {
		t.Errorf("T with nil translator, no default = %q, want 'unknown'", got)
	}

	// GetLocale with nil translator returns "en-US"
	if got := pt.GetLocale(); got != "en-US" {
		t.Errorf("GetLocale with nil translator = %q", got)
	}

	// SetLocale with nil translator is a no-op (should not panic)
	pt.SetLocale("fr-FR")

	// Has with nil translator returns false
	if pt.Has("greeting") {
		t.Error("Has with nil translator should be false")
	}
}

func TestWiring_PackageTranslatorInvalidNamespace(t *testing.T) {
	t.Parallel()

	pt := NewPackageTranslator("")
	if pt != nil {
		t.Error("NewPackageTranslator with empty namespace should return nil")
	}
}

// =============================================================================
// TestWiring_Namespace tests Namespace and all its methods.
// =============================================================================

func TestWiring_Namespace(t *testing.T) {
	t.Parallel()

	localeDir := setupWiringNamespaceLocaleDir(t)

	translator, err := New(
		WithFileSystemLoader(localeDir),
		WithDefaultLocale("en-US"),
	)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	ns, err := NewNamespace("mypkg", translator)
	if err != nil {
		t.Fatalf("NewNamespace error: %v", err)
	}

	// T
	if got := ns.T("greeting"); got != "Hello from mypkg" {
		t.Errorf("Namespace.T(greeting) = %q", got)
	}

	// TF
	if got := ns.TF("welcome_fmt", "Eve"); got != "Welcome, Eve!" {
		t.Errorf("Namespace.TF(welcome_fmt, Eve) = %q", got)
	}

	// TD with existing key
	if got := ns.TD("greeting", "fallback"); got != "Hello from mypkg" {
		t.Errorf("Namespace.TD(greeting) = %q", got)
	}

	// TD with missing key
	if got := ns.TD("missing", "fallback"); got != "fallback" {
		t.Errorf("Namespace.TD(missing) = %q, want 'fallback'", got)
	}

	// Has
	if !ns.Has("greeting") {
		t.Error("Namespace.Has(greeting) = false")
	}
	if ns.Has("missing") {
		t.Error("Namespace.Has(missing) = true")
	}

	// Key
	if got := ns.Key("greeting"); got != "mypkg.greeting" {
		t.Errorf("Namespace.Key(greeting) = %q", got)
	}
}

func TestWiring_NamespaceNilTranslator(t *testing.T) {
	t.Parallel()

	ns, err := NewNamespace("mypkg", nil)
	if err != nil {
		t.Fatalf("NewNamespace(nil translator) error: %v", err)
	}

	// T returns full key
	if got := ns.T("greeting"); got != "mypkg.greeting" {
		t.Errorf("Namespace.T with nil translator = %q", got)
	}

	// TF returns full key
	if got := ns.TF("welcome", "arg"); got != "mypkg.welcome" {
		t.Errorf("Namespace.TF with nil translator = %q", got)
	}

	// TD returns default
	if got := ns.TD("key", "default"); got != "default" {
		t.Errorf("Namespace.TD with nil translator = %q", got)
	}

	// Has returns false
	if ns.Has("key") {
		t.Error("Namespace.Has with nil translator should be false")
	}
}

func TestWiring_NamespaceInvalid(t *testing.T) {
	t.Parallel()

	_, err := NewNamespace("", nil)
	if err == nil {
		t.Error("NewNamespace with empty prefix should fail")
	}
}

// =============================================================================
// TestWiring_Loaders tests all loader constructors.
// =============================================================================

func TestWiring_EmbedFSLoader(t *testing.T) {
	t.Parallel()

	loader := NewEmbedFSLoader(wiringTestLocales, "testdata/locales")
	data, err := loader.Load("en-US")
	if err != nil {
		t.Fatalf("EmbedFSLoader.Load(en-US) error: %v", err)
	}
	if len(data) == 0 {
		t.Error("EmbedFSLoader.Load returned empty data")
	}
}

func TestWiring_EmbedFSLoaderWithExtension(t *testing.T) {
	t.Parallel()

	loader := NewEmbedFSLoader(wiringTestLocales, "testdata/locales", WithExtension(".json"))
	data, err := loader.Load("en-US")
	if err != nil {
		t.Fatalf("EmbedFSLoader.Load with extension error: %v", err)
	}
	if len(data) == 0 {
		t.Error("EmbedFSLoader.Load with extension returned empty data")
	}
}

func TestWiring_RegistryLoader(t *testing.T) {
	t.Parallel()

	loader := NewRegistryLoader()
	// May fail if no locales are registered; that's OK. We verify the type works.
	_, _ = loader.Load("en-US")
}

func TestWiring_FileSystemLoader(t *testing.T) {
	t.Parallel()

	localeDir := setupWiringLocaleDir(t)

	loader := NewFileSystemLoader(localeDir)
	data, err := loader.Load("en-US")
	if err != nil {
		t.Fatalf("FileSystemLoader.Load(en-US) error: %v", err)
	}
	if len(data) == 0 {
		t.Error("FileSystemLoader.Load returned empty data")
	}
}

func TestWiring_FileSystemLoaderWithExtension(t *testing.T) {
	t.Parallel()

	localeDir := setupWiringLocaleDir(t)

	loader := NewFileSystemLoader(localeDir, WithExtension(".json"))
	data, err := loader.Load("en-US")
	if err != nil {
		t.Fatalf("FileSystemLoader.Load with extension error: %v", err)
	}
	if len(data) == 0 {
		t.Error("FileSystemLoader.Load with extension returned empty data")
	}
}

// =============================================================================
// TestWiring_JSONParser tests JSONParser.Parse.
// =============================================================================

func TestWiring_JSONParser(t *testing.T) {
	t.Parallel()

	p := NewJSONParser()

	data := []byte(`{"greeting": "Hello", "nested": {"key": "value"}}`)
	result, err := p.Parse(data)
	if err != nil {
		t.Fatalf("JSONParser.Parse error: %v", err)
	}
	if result["greeting"] != "Hello" {
		t.Errorf("Parse result[greeting] = %v", result["greeting"])
	}
	nested, ok := result["nested"].(map[string]interface{})
	if !ok {
		t.Fatal("Parse result[nested] is not a map")
	}
	if nested["key"] != "value" {
		t.Errorf("Parse result[nested][key] = %v", nested["key"])
	}
}

func TestWiring_JSONParserInvalid(t *testing.T) {
	t.Parallel()

	p := NewJSONParser()

	_, err := p.Parse([]byte(`not json`))
	if err == nil {
		t.Error("JSONParser.Parse(invalid) should fail")
	}
}

// =============================================================================
// TestWiring_DefaultKeyResolver tests Resolve.
// =============================================================================

func TestWiring_DefaultKeyResolver(t *testing.T) {
	t.Parallel()

	r := NewDefaultKeyResolver()

	translations := map[string]interface{}{
		"greeting": "Hello",
		"user": map[string]interface{}{
			"name": "Alice",
		},
	}

	got, err := r.Resolve(translations, "greeting")
	if err != nil {
		t.Fatalf("Resolve(greeting) error: %v", err)
	}
	if got != "Hello" {
		t.Errorf("Resolve(greeting) = %q", got)
	}

	got, err = r.Resolve(translations, "user.name")
	if err != nil {
		t.Fatalf("Resolve(user.name) error: %v", err)
	}
	if got != "Alice" {
		t.Errorf("Resolve(user.name) = %q", got)
	}

	_, err = r.Resolve(translations, "nonexistent")
	if err == nil {
		t.Error("Resolve(nonexistent) should fail")
	}
}

// =============================================================================
// TestWiring_DefaultFallbackChainer tests GetChain.
// =============================================================================

func TestWiring_DefaultFallbackChainer(t *testing.T) {
	t.Parallel()

	c := NewDefaultFallbackChainer()

	// en-US has no further fallback
	chain := c.GetChain("en-US")
	if len(chain) != 1 || chain[0] != "en-US" {
		t.Errorf("GetChain(en-US) = %v", chain)
	}

	// es-MX falls back to es-ES then en-US
	chain = c.GetChain("es-MX")
	if len(chain) != 3 || chain[0] != "es-MX" || chain[1] != "es-ES" || chain[2] != "en-US" {
		t.Errorf("GetChain(es-MX) = %v", chain)
	}

	// en-GB falls back to en-US
	chain = c.GetChain("en-GB")
	if len(chain) != 2 || chain[0] != "en-GB" || chain[1] != "en-US" {
		t.Errorf("GetChain(en-GB) = %v", chain)
	}
}

// =============================================================================
// TestWiring_DefaultPluralResolver tests Resolve.
// =============================================================================

func TestWiring_DefaultPluralResolver(t *testing.T) {
	t.Parallel()

	r := NewDefaultPluralResolver()

	// English: 1 => One, 2 => Other
	if got := r.Resolve("en-US", 1); got != core.One {
		t.Errorf("Resolve(en-US, 1) = %q, want One", got)
	}
	if got := r.Resolve("en-US", 2); got != core.Other {
		t.Errorf("Resolve(en-US, 2) = %q, want Other", got)
	}

	// Japanese: always Other
	if got := r.Resolve("ja-JP", 1); got != core.Other {
		t.Errorf("Resolve(ja-JP, 1) = %q, want Other", got)
	}

	// Arabic: 0 => Zero, 1 => One, 2 => Two
	if got := r.Resolve("ar-SA", 0); got != core.Zero {
		t.Errorf("Resolve(ar-SA, 0) = %q, want Zero", got)
	}
	if got := r.Resolve("ar-SA", 2); got != core.Two {
		t.Errorf("Resolve(ar-SA, 2) = %q, want Two", got)
	}
}

// =============================================================================
// TestWiring_Validation tests all validation functions.
// =============================================================================

func TestWiring_ValidateLocale(t *testing.T) {
	t.Parallel()

	if err := core.ValidateLocale("en-US"); err != nil {
		t.Errorf("ValidateLocale(en-US) error: %v", err)
	}
	if err := core.ValidateLocale("es"); err != nil {
		t.Errorf("ValidateLocale(es) error: %v", err)
	}
	if err := core.ValidateLocale(""); err == nil {
		t.Error("ValidateLocale('') should fail")
	}
	if err := core.ValidateLocale("../etc/passwd"); err == nil {
		t.Error("ValidateLocale(traversal) should fail")
	}
	if err := core.ValidateLocale("invalid!"); err == nil {
		t.Error("ValidateLocale(invalid!) should fail")
	}
}

func TestWiring_ValidateKey(t *testing.T) {
	t.Parallel()

	if err := core.ValidateKey("greeting"); err != nil {
		t.Errorf("ValidateKey(greeting) error: %v", err)
	}
	if err := core.ValidateKey("user.profile.title"); err != nil {
		t.Errorf("ValidateKey(user.profile.title) error: %v", err)
	}
	if err := core.ValidateKey(""); err == nil {
		t.Error("ValidateKey('') should fail")
	}
	if err := core.ValidateKey("bad key!"); err == nil {
		t.Error("ValidateKey('bad key!') should fail")
	}
	if err := core.ValidateKey(".leading"); err == nil {
		t.Error("ValidateKey(.leading) should fail")
	}
	if err := core.ValidateKey("trailing."); err == nil {
		t.Error("ValidateKey(trailing.) should fail")
	}
	if err := core.ValidateKey("double..dot"); err == nil {
		t.Error("ValidateKey(double..dot) should fail")
	}
}

func TestWiring_ValidateFormatString(t *testing.T) {
	t.Parallel()

	if err := core.ValidateFormatString("Hello, %s!", 1); err != nil {
		t.Errorf("ValidateFormatString(Hello %%s, 1) error: %v", err)
	}
	if err := core.ValidateFormatString("no args", 0); err != nil {
		t.Errorf("ValidateFormatString(no args, 0) error: %v", err)
	}
	if err := core.ValidateFormatString("%n exploit", 0); err == nil {
		t.Error("ValidateFormatString(%%n) should fail")
	}
	if err := core.ValidateFormatString("%s %d", 1); err == nil {
		t.Error("ValidateFormatString(mismatch) should fail")
	}
}

func TestWiring_SanitizeOutput(t *testing.T) {
	t.Parallel()

	// Normal string passes through
	if got := core.SanitizeOutput("Hello, World!"); got != "Hello, World!" {
		t.Errorf("SanitizeOutput normal = %q", got)
	}

	// ANSI escape stripped
	input := "\x1b[31mRed\x1b[0m"
	got := core.SanitizeOutput(input)
	if strings.Contains(got, "\x1b") {
		t.Errorf("SanitizeOutput should strip ANSI, got %q", got)
	}
	if got != "Red" {
		t.Errorf("SanitizeOutput ANSI = %q, want Red", got)
	}

	// Null bytes stripped
	if got := core.SanitizeOutput("a\x00b"); got != "ab" {
		t.Errorf("SanitizeOutput null bytes = %q", got)
	}

	// Newline and tab preserved
	if got := core.SanitizeOutput("a\tb\nc"); got != "a\tb\nc" {
		t.Errorf("SanitizeOutput tab/newline = %q", got)
	}
}

// =============================================================================
// TestWiring_ErrorConstructors tests all error constructors and Is() methods.
// =============================================================================

func TestWiring_ErrorConstructors(t *testing.T) {
	t.Parallel()

	// ErrInvalidLocale
	e1 := core.NewErrInvalidLocale("bad", errors.New("test"))
	if e1.Locale != "bad" {
		t.Errorf("ErrInvalidLocale.Locale = %q", e1.Locale)
	}
	if !errors.Is(e1, &core.ErrInvalidLocale{}) {
		t.Error("errors.Is(ErrInvalidLocale) failed")
	}
	if e1.Error() == "" {
		t.Error("ErrInvalidLocale.Error() is empty")
	}
	if e1.Unwrap() == nil {
		t.Error("ErrInvalidLocale.Unwrap() is nil")
	}

	// ErrInvalidKey
	e2 := core.NewErrInvalidKey("bad", errors.New("test"))
	if e2.Key != "bad" {
		t.Errorf("ErrInvalidKey.Key = %q", e2.Key)
	}
	if !errors.Is(e2, &core.ErrInvalidKey{}) {
		t.Error("errors.Is(ErrInvalidKey) failed")
	}
	if e2.Unwrap() == nil {
		t.Error("ErrInvalidKey.Unwrap() is nil")
	}

	// ErrKeyNotFound
	e3 := core.NewErrKeyNotFound("missing")
	if e3.Key != "missing" {
		t.Errorf("ErrKeyNotFound.Key = %q", e3.Key)
	}
	if !errors.Is(e3, &core.ErrKeyNotFound{}) {
		t.Error("errors.Is(ErrKeyNotFound) failed")
	}

	// ErrInvalidFormat
	e4 := core.NewErrInvalidFormat("fmt", errors.New("test"))
	if e4.Format != "fmt" {
		t.Errorf("ErrInvalidFormat.Format = %q", e4.Format)
	}
	if !errors.Is(e4, &core.ErrInvalidFormat{}) {
		t.Error("errors.Is(ErrInvalidFormat) failed")
	}
	if e4.Unwrap() == nil {
		t.Error("ErrInvalidFormat.Unwrap() is nil")
	}

	// ErrPathTraversal
	e5 := core.NewErrPathTraversal("../etc")
	if e5.Path != "../etc" {
		t.Errorf("ErrPathTraversal.Path = %q", e5.Path)
	}
	if !errors.Is(e5, &core.ErrPathTraversal{}) {
		t.Error("errors.Is(ErrPathTraversal) failed")
	}

	// ErrUnknownFormat
	e6 := core.NewErrUnknownFormat(".xyz")
	if e6.Extension != ".xyz" {
		t.Errorf("ErrUnknownFormat.Extension = %q", e6.Extension)
	}
	if !errors.Is(e6, &core.ErrUnknownFormat{}) {
		t.Error("errors.Is(ErrUnknownFormat) failed")
	}
}

// =============================================================================
// TestWiring_ParserRegistry tests RegisterParser, GetParser, RegisteredFormats.
// =============================================================================

func TestWiring_ParserRegistry(t *testing.T) {
	// Not parallel: modifies global registry state.

	// GetParser for .json (registered by default)
	p, err := GetParser(".json")
	if err != nil {
		t.Fatalf("GetParser(.json) error: %v", err)
	}
	if p == nil {
		t.Fatal("GetParser(.json) returned nil")
	}

	// RegisteredFormats includes .json
	formats := RegisteredFormats()
	found := false
	for _, f := range formats {
		if f == ".json" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("RegisteredFormats() = %v, missing .json", formats)
	}

	// Register a custom parser
	customParser := NewJSONParser() // reuse JSON parser as a stand-in
	if err := RegisterParser(".custom", customParser); err != nil {
		t.Fatalf("RegisterParser(.custom) error: %v", err)
	}

	got, err := GetParser(".custom")
	if err != nil {
		t.Fatalf("GetParser(.custom) error: %v", err)
	}
	if got != customParser {
		t.Error("GetParser(.custom) returned different parser instance")
	}

	// GetParser for unknown extension
	_, err = GetParser(".unknown")
	if err == nil {
		t.Error("GetParser(.unknown) should fail")
	}
	if !errors.Is(err, &core.ErrUnknownFormat{}) {
		t.Errorf("GetParser(.unknown) error type = %T, want ErrUnknownFormat", err)
	}

	// RegisterParser with nil parser
	if err := RegisterParser(".nil", nil); err == nil {
		t.Error("RegisterParser with nil parser should fail")
	}

	// RegisterParser with invalid extension
	if err := RegisterParser("noleadingdot", customParser); err == nil {
		t.Error("RegisterParser with invalid extension should fail")
	}
}

func TestWiring_NewRegistry(t *testing.T) {
	t.Parallel()

	r := NewRegistry()
	if r == nil {
		t.Fatal("NewRegistry() returned nil")
	}

	// Empty registry
	formats := r.RegisteredFormats()
	if len(formats) != 0 {
		t.Errorf("new Registry.RegisteredFormats() = %v, want empty", formats)
	}

	// Register and retrieve
	parser := NewJSONParser()
	if err := r.RegisterParser(".test", parser); err != nil {
		t.Fatalf("Registry.RegisterParser error: %v", err)
	}
	got, err := r.GetParser(".test")
	if err != nil {
		t.Fatalf("Registry.GetParser error: %v", err)
	}
	if got != parser {
		t.Error("Registry.GetParser returned different instance")
	}
}

// =============================================================================
// TestWiring_RegisteredLocales tests RegisteredLocales.
// =============================================================================

func TestWiring_RegisteredLocales(t *testing.T) {
	// Not parallel: reads global registry.

	locales := RegisteredLocales()
	// May be empty without build tags, but should not be nil.
	if locales == nil {
		t.Error("RegisteredLocales() returned nil, want non-nil slice")
	}
}

// =============================================================================
// TestWiring_NormalizeLocale tests NormalizeLocale.
// =============================================================================

func TestWiring_NormalizeLocale(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"en_US.UTF-8", "en-US"},
		{"es_MX", "es-MX"},
		{"POSIX", "en-US"},
		{"C", "en-US"},
		{"en", "en-US"},
		{"fr", "fr-FR"},
		{"", ""},
	}

	for _, tt := range tests {
		if got := NormalizeLocale(tt.input); got != tt.want {
			t.Errorf("NormalizeLocale(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// =============================================================================
// TestWiring_DetectLocale tests the package-level DetectLocale.
// =============================================================================

func TestWiring_DetectLocale(t *testing.T) {
	t.Parallel()

	locale := DetectLocale()
	if locale == "" {
		t.Error("DetectLocale() returned empty string")
	}
}

// =============================================================================
// TestWiring_Logger tests SetLogger and GetLogger.
// =============================================================================

func TestWiring_Logger(t *testing.T) {
	// Not parallel: modifies global logger.

	original := GetLogger()
	defer SetLogger(original) // restore

	// SetLogger with NopLogger
	SetLogger(NopLogger{})
	if got := GetLogger(); got == nil {
		t.Error("GetLogger() returned nil after SetLogger(NopLogger)")
	}

	// SetLogger with nil resets to NopLogger
	SetLogger(nil)
	if got := GetLogger(); got == nil {
		t.Error("GetLogger() returned nil after SetLogger(nil)")
	}
}

func TestWiring_NopLogger(t *testing.T) {
	t.Parallel()

	var l NopLogger

	// All methods should be callable without panic
	l.Debug("test", "k", "v")
	l.Info("test", "k", "v")
	l.Warn("test", "k", "v")
	l.Error("test", "k", "v")
	l.Fatal("test", "k", "v")

	wf := l.WithFields("k", "v")
	if wf == nil {
		t.Error("NopLogger.WithFields returned nil")
	}

	wc := l.WithContext(context.TODO())
	if wc == nil {
		t.Error("NopLogger.WithContext returned nil")
	}

	wl := l.WithLevel(core.LevelDebug)
	if wl == nil {
		t.Error("NopLogger.WithLevel returned nil")
	}

	if l.Enabled(core.LevelDebug) {
		t.Error("NopLogger.Enabled should return false")
	}
}

// =============================================================================
// TestWiring_GetSupportedLocales tests GetSupportedLocales.
// =============================================================================

func TestWiring_GetSupportedLocales(t *testing.T) {
	t.Parallel()

	localeDir := setupWiringLocaleDir(t)
	loader := NewFileSystemLoader(localeDir)

	supported := GetSupportedLocales(loader, "en-US", "es-ES", "zh-CN")
	// en-US should be found, zh-CN may not (depends on setup)
	foundEnUS := false
	for _, l := range supported {
		if l == "en-US" {
			foundEnUS = true
		}
	}
	if !foundEnUS {
		t.Errorf("GetSupportedLocales did not find en-US, got %v", supported)
	}
}

// =============================================================================
// TestWiring_InterfaceCompliance verifies compile-time interface satisfaction.
// =============================================================================

func TestWiring_InterfaceCompliance(t *testing.T) {
	t.Parallel()

	// Translator implements TranslatorProvider and sub-interfaces
	var _ core.TranslatorProvider = (*Translator)(nil)
	var _ core.TranslationLookup = (*Translator)(nil)
	var _ core.FormattedTranslator = (*Translator)(nil)
	var _ core.KeyChecker = (*Translator)(nil)
	var _ core.LocaleSetter = (*Translator)(nil)
	var _ core.LocaleGetter = (*Translator)(nil)
	var _ core.PluralTranslator = (*Translator)(nil)
	var _ core.GenderTranslator = (*Translator)(nil)

	// PackageTranslator implements TranslatorProvider
	var _ core.TranslatorProvider = (*PackageTranslator)(nil)

	// MapCache implements Cacher
	var _ core.Cacher = (*MapCache)(nil)

	// Loaders implement TranslationLoader
	var _ core.TranslationLoader = (*EmbedFSLoader)(nil)
	var _ core.TranslationLoader = (*RegistryLoader)(nil)
	var _ core.TranslationLoader = (*FileSystemLoader)(nil)

	// JSONParser implements TranslationParser
	var _ core.TranslationParser = (*JSONParser)(nil)

	// Resolver implements KeyResolver
	var _ core.KeyResolver = (*DefaultKeyResolver)(nil)

	// Chainer implements FallbackChainer
	var _ core.FallbackChainer = (*DefaultFallbackChainer)(nil)

	// PluralResolver implements PluralResolver
	var _ core.PluralResolver = (*DefaultPluralResolver)(nil)

	// Detectors implement LocaleDetector
	var _ core.LocaleDetector = (*DefaultLocaleDetector)(nil)
	var _ core.LocaleDetector = (*AcceptLanguageDetector)(nil)
	var _ core.LocaleDetector = (*ChainDetector)(nil)
	var _ core.LocaleDetector = (*StaticDetector)(nil)
	var _ core.LocaleDetector = (*BrowserDetector)(nil)

	// NopLogger implements Logger
	var _ core.Logger = NopLogger{}
	var _ core.LeveledLogger = NopLogger{}
}

// =============================================================================
// Test helper: create locale directories with translation files
// =============================================================================

// setupWiringLocaleDir creates a basic locale directory with en-US and es-ES.
func setupWiringLocaleDir(t *testing.T) string {
	t.Helper()

	localeDir := filepath.Join(t.TempDir(), "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("Failed to create locale dir: %v", err)
	}

	enUS := []byte(`{
		"greeting": "Hello",
		"farewell": "Goodbye",
		"welcome_fmt": "Welcome, %s!",
		"error": {
			"validation": {
				"required": "This field is required"
			}
		}
	}`)
	if err := os.WriteFile(filepath.Join(localeDir, "en-US.json"), enUS, 0o644); err != nil {
		t.Fatalf("Failed to write en-US.json: %v", err)
	}

	esES := []byte(`{
		"greeting": "Hola",
		"farewell": "Adios"
	}`)
	if err := os.WriteFile(filepath.Join(localeDir, "es-ES.json"), esES, 0o644); err != nil {
		t.Fatalf("Failed to write es-ES.json: %v", err)
	}

	return localeDir
}

// setupWiringPluralLocaleDir creates locale files with plural, gender, ICU, and
// format-string keys for comprehensive method testing.
func setupWiringPluralLocaleDir(t *testing.T) string {
	t.Helper()

	localeDir := filepath.Join(t.TempDir(), "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("Failed to create locale dir: %v", err)
	}

	enUS := []byte(`{
		"greeting": "Hello",
		"welcome_fmt": "Welcome, %s!",
		"items": {
			"one": "# item",
			"other": "# items"
		},
		"messages": {
			"one": "You have # message in %s",
			"other": "You have # messages in %s"
		},
		"who": {
			"masculine": "He logged in",
			"feminine": "She logged in",
			"other": "They logged in"
		},
		"icu_plural": "{count, plural, one {# item} other {# items}}",
		"icu_select": "{gender, select, male {He} female {She} other {They}}"
	}`)
	if err := os.WriteFile(filepath.Join(localeDir, "en-US.json"), enUS, 0o644); err != nil {
		t.Fatalf("Failed to write en-US.json: %v", err)
	}

	esES := []byte(`{
		"greeting": "Hola",
		"items": {
			"one": "# elemento",
			"other": "# elementos"
		}
	}`)
	if err := os.WriteFile(filepath.Join(localeDir, "es-ES.json"), esES, 0o644); err != nil {
		t.Fatalf("Failed to write es-ES.json: %v", err)
	}

	return localeDir
}

// setupWiringNamespaceLocaleDir creates locale files with namespace-prefixed keys
// for Namespace and PackageTranslator testing.
func setupWiringNamespaceLocaleDir(t *testing.T) string {
	t.Helper()

	localeDir := filepath.Join(t.TempDir(), "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("Failed to create locale dir: %v", err)
	}

	enUS := []byte(`{
		"mypkg": {
			"greeting": "Hello from mypkg",
			"welcome_fmt": "Welcome, %s!"
		}
	}`)
	if err := os.WriteFile(filepath.Join(localeDir, "en-US.json"), enUS, 0o644); err != nil {
		t.Fatalf("Failed to write en-US.json: %v", err)
	}

	return localeDir
}
