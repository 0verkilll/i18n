// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

package engine

import (
	"embed"
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/0verkilll/i18n/internal/core"
)

// =============================================================================
// Namespace
// =============================================================================

// matchPrefix validates that s contains only alphanumeric characters,
// underscores, and hyphens. Returns false for empty strings.
func matchPrefix(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if !isPrefixChar(s[i]) {
			return false
		}
	}
	return true
}

// isPrefixChar returns true if c is an alphanumeric, underscore, or hyphen character.
func isPrefixChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-'
}

// Namespace automatically prefixes translation keys with a package name,
// eliminating manual key prefixing across ecosystem packages.
//
// Namespace is immutable after construction. It holds no mutable state and
// requires no mutex of its own. All translation operations delegate to the
// underlying core.TranslatorProvider, which is already thread-safe. Multiple
// goroutines may safely share a single *Namespace instance.
type Namespace struct {
	translator core.TranslatorProvider
	prefix     string
}

// NewNamespace creates a Namespace that prefixes all translation keys with the
// given prefix. The prefix must be non-empty, at most 64 characters, and
// contain only alphanumeric characters, underscores, or hyphens.
//
// A nil core.TranslatorProvider is accepted without error; all methods will degrade
// gracefully by returning the namespaced key or a default value.
func NewNamespace(prefix string, t core.TranslatorProvider) (*Namespace, error) {
	if prefix == "" {
		return nil, fmt.Errorf("namespace prefix cannot be empty")
	}

	if len(prefix) > core.MaxPrefixLength {
		return nil, fmt.Errorf("namespace prefix exceeds maximum length of %d characters", core.MaxPrefixLength)
	}

	if !matchPrefix(prefix) {
		return nil, fmt.Errorf("namespace prefix %q contains invalid characters (only a-z, A-Z, 0-9, _, - allowed)", prefix)
	}

	return &Namespace{
		translator: t,
		prefix:     prefix,
	}, nil
}

// T translates the namespaced key by joining the prefix and key with a dot
// separator and delegating to the underlying core.TranslatorProvider.
//
// When the translator is nil, it returns the full namespaced key as a fallback.
// Nil receiver safe: returns the key argument unchanged.
func (ns *Namespace) T(key string) string {
	if ns == nil {
		return key
	}

	fullKey := ns.prefix + "." + key

	if ns.translator == nil {
		return fullKey
	}

	return ns.translator.Translate(fullKey)
}

// TF translates the namespaced key with format arguments by joining the prefix
// and key with a dot separator and delegating to TranslateWithArgs.
//
// When the translator is nil, it returns the full namespaced key without formatting.
// Nil receiver safe: returns the key argument unchanged.
func (ns *Namespace) TF(key string, args ...interface{}) string {
	if ns == nil {
		return key
	}

	fullKey := ns.prefix + "." + key

	if ns.translator == nil {
		return fullKey
	}

	return ns.translator.TranslateWithArgs(fullKey, args...)
}

// TD translates the namespaced key, returning defaultValue when the key is not
// found or the translator is nil. This is the primary method for ecosystem
// packages that provide hardcoded English defaults.
//
// Nil receiver safe: returns defaultValue.
func (ns *Namespace) TD(key, defaultValue string) string {
	if ns == nil {
		return defaultValue
	}

	fullKey := ns.prefix + "." + key

	if ns.translator == nil {
		return defaultValue
	}

	if ns.translator.HasKey(fullKey) {
		return ns.translator.Translate(fullKey)
	}

	return defaultValue
}

// Has checks whether the namespaced key exists in the translator.
//
// When the translator is nil, it returns false.
// Nil receiver safe: returns false.
func (ns *Namespace) Has(key string) bool {
	if ns == nil {
		return false
	}

	fullKey := ns.prefix + "." + key

	if ns.translator == nil {
		return false
	}

	return ns.translator.HasKey(fullKey)
}

// Key returns the full namespaced key (prefix + "." + key) without performing
// any translation lookup. Useful for logging, error wrapping, or passing keys
// to other systems.
//
// Nil receiver safe: returns the key argument unchanged.
func (ns *Namespace) Key(key string) string {
	if ns == nil {
		return key
	}

	return ns.prefix + "." + key
}

// =============================================================================
// PackageTranslator
// =============================================================================

// PackageOption is a functional option for configuring a PackageTranslator.
// Unlike the core Option type, PackageOption does not return an error because
// option setters are trivial value assignments.
type PackageOption func(*PackageTranslator)

// translatorBox wraps core.TranslatorProvider so atomic.Pointer sees a
// consistent concrete type regardless of which implementation is stored.
// Standard Go idiom for atomic swap of an interface value.
type translatorBox struct{ t core.TranslatorProvider }

// PackageTranslator provides per-package translation with namespace scoping,
// thread-safe translator swapping, and hardcoded default fallbacks. Package
// authors use this type to eliminate ~100-200 lines of duplicated i18n
// boilerplate per package.
//
// All methods are nil-safe for the translator field: when no translator has
// been set, methods fall back to the defaults map, then to the raw key.
//
// PackageTranslator implements core.TranslatorProvider so it can be passed to
// other packages expecting that interface.
//
// The translator pointer is swapped atomically via atomic.Pointer for
// lock-free reads. Reads observe either the most recent Store or the zero
// value (which is treated as "no translator set").
type PackageTranslator struct {
	translator atomic.Pointer[translatorBox]
	defaults   map[string]string
	namespace  string
}

// Compile-time assertion that PackageTranslator implements core.TranslatorProvider.
var _ core.TranslatorProvider = (*PackageTranslator)(nil)

// loadTranslator returns the currently stored translator, or nil if none.
func (pt *PackageTranslator) loadTranslator() core.TranslatorProvider {
	if b := pt.translator.Load(); b != nil {
		return b.t
	}
	return nil
}

// WithDefaults sets the hardcoded English fallback strings on the PackageTranslator.
// Keys in the map are short keys (e.g., "error.empty_path"), not namespace-prefixed.
// A nil map is acceptable and is treated as empty.
func WithDefaults(defaults map[string]string) PackageOption {
	return func(pt *PackageTranslator) {
		pt.defaults = defaults
	}
}

// WithTranslator sets the initial translator on the PackageTranslator.
// This has the same effect as calling SetTranslator after construction.
// A nil value is acceptable and puts the PackageTranslator in defaults-only mode.
func WithTranslator(t core.TranslatorProvider) PackageOption {
	return func(pt *PackageTranslator) {
		pt.translator.Store(&translatorBox{t: t})
	}
}

// NewPackageTranslator creates a PackageTranslator with the given namespace
// and optional configuration. The namespace is validated with core.ValidateKey and
// becomes a dot-notation prefix for all translation keys.
//
// Returns nil if the namespace is invalid.
func NewPackageTranslator(namespace string, opts ...PackageOption) *PackageTranslator {
	if err := core.ValidateKey(namespace); err != nil {
		return nil
	}

	pt := &PackageTranslator{
		namespace: namespace,
	}

	for _, opt := range opts {
		opt(pt)
	}

	return pt
}

// T translates a key by prepending the namespace prefix, delegating to the
// underlying translator, and falling back to defaults or the raw key.
func (pt *PackageTranslator) T(key string) string {
	t := pt.loadTranslator()

	fullKey := pt.namespace + "." + key

	if t != nil {
		result := t.Translate(fullKey)
		if result != fullKey {
			return result
		}
	}

	if pt.defaults != nil {
		if def, ok := pt.defaults[key]; ok {
			return core.SanitizeOutput(def)
		}
	}

	return key
}

// TF translates a key with format arguments by prepending the namespace
// prefix, delegating to the underlying translator, and falling back to
// formatting the default string or returning the raw key.
func (pt *PackageTranslator) TF(key string, args ...interface{}) string {
	t := pt.loadTranslator()

	fullKey := pt.namespace + "." + key

	if t != nil {
		result := t.TranslateWithArgs(fullKey, args...)
		if result != fullKey {
			return result
		}
	}

	if pt.defaults != nil {
		if def, ok := pt.defaults[key]; ok {
			return core.SanitizeOutput(fmt.Sprintf(def, args...))
		}
	}

	return key
}

// Has checks whether the namespaced key exists in the underlying translator.
// Returns false when the translator is nil.
func (pt *PackageTranslator) Has(key string) bool {
	t := pt.loadTranslator()

	if t == nil {
		return false
	}

	fullKey := pt.namespace + "." + key
	return t.HasKey(fullKey)
}

// Translate implements core.TranslatorProvider. The key is expected to be fully
// qualified (already includes namespace when called through the interface).
// This method does NOT double-prepend the namespace.
func (pt *PackageTranslator) Translate(key string) string {
	t := pt.loadTranslator()

	if t != nil {
		result := t.Translate(key)
		if result != key {
			return result
		}
	}

	// Strip namespace prefix and look up in defaults
	if pt.defaults != nil {
		shortKey := stripNamespacePrefix(key, pt.namespace)
		if def, ok := pt.defaults[shortKey]; ok {
			return core.SanitizeOutput(def)
		}
	}

	return key
}

// TranslateWithArgs implements core.TranslatorProvider. The key is expected to be
// fully qualified. This method does NOT double-prepend the namespace.
func (pt *PackageTranslator) TranslateWithArgs(key string, args ...interface{}) string {
	t := pt.loadTranslator()

	if t != nil {
		result := t.TranslateWithArgs(key, args...)
		if result != key {
			return result
		}
	}

	// Strip namespace prefix and look up in defaults
	if pt.defaults != nil {
		shortKey := stripNamespacePrefix(key, pt.namespace)
		if def, ok := pt.defaults[shortKey]; ok {
			return core.SanitizeOutput(fmt.Sprintf(def, args...))
		}
	}

	return key
}

// TranslatePlural implements core.TranslatorProvider. Delegates to the underlying
// translator's TranslatePlural. Returns the key if the translator is nil.
func (pt *PackageTranslator) TranslatePlural(key string, count interface{}) string {
	t := pt.loadTranslator()

	if t != nil {
		return t.TranslatePlural(key, count)
	}

	return key
}

// TranslateGender implements core.TranslatorProvider. Delegates to the underlying
// translator's TranslateGender. Returns the key if the translator is nil.
func (pt *PackageTranslator) TranslateGender(key string, gender core.GenderCategory) string {
	t := pt.loadTranslator()

	if t != nil {
		return t.TranslateGender(key, gender)
	}

	return key
}

// HasKey implements core.TranslatorProvider. Delegates directly to the underlying
// translator's HasKey. Returns false if the translator is nil.
func (pt *PackageTranslator) HasKey(key string) bool {
	t := pt.loadTranslator()

	if t == nil {
		return false
	}

	return t.HasKey(key)
}

// SetLocale implements core.TranslatorProvider. Delegates to the underlying
// translator's SetLocale. No-op if the translator is nil.
func (pt *PackageTranslator) SetLocale(locale string) {
	t := pt.loadTranslator()

	if t == nil {
		return
	}

	t.SetLocale(locale)
}

// GetLocale implements core.TranslatorProvider. Delegates to the underlying
// translator's GetLocale. Returns "en-US" if the translator is nil.
func (pt *PackageTranslator) GetLocale() string {
	t := pt.loadTranslator()

	if t == nil {
		return "en-US"
	}

	return t.GetLocale()
}

// SetTranslator swaps the underlying translator. Nil is allowed and resets the
// PackageTranslator to defaults-only mode. This method is thread-safe.
func (pt *PackageTranslator) SetTranslator(t core.TranslatorProvider) {
	pt.translator.Store(&translatorBox{t: t})
}

// GetTranslator returns the current underlying translator, or nil if none is set.
func (pt *PackageTranslator) GetTranslator() core.TranslatorProvider {
	return pt.loadTranslator()
}

// stripNamespacePrefix removes "namespace." from the beginning of a key.
// If the key does not start with the namespace prefix, it is returned as-is.
func stripNamespacePrefix(key, namespace string) string {
	prefix := namespace + "."
	if strings.HasPrefix(key, prefix) {
		return key[len(prefix):]
	}
	return key
}

// DiscoverLocales reads an embed.FS directory and returns the locale codes
// found based on JSON filenames. This eliminates the per-package
// GetSupportedLocales reimplementation (~30 packages do this identically).
//
// Example:
//
//	//go:embed locales/*.json
//	var localeFS embed.FS
//	locales := i18n.DiscoverLocales(localeFS, "locales")
//	// returns ["en-US", "es-ES", "fr-FR"]
func DiscoverLocales(fs embed.FS, basePath string) []string {
	entries, err := fs.ReadDir(basePath)
	if err != nil {
		return []string{}
	}

	locales := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if len(name) > 5 && name[len(name)-5:] == ".json" {
			locales = append(locales, name[:len(name)-5])
		}
	}

	return locales
}

// NewPackageTranslatorWithFS creates a fully-configured PackageTranslator backed
// by an embedded filesystem. This is the primary integration point for ecosystem
// packages — it replaces ~300 lines of per-package i18n.go boilerplate with a
// single constructor call.
//
// It creates an i18n.Translator with EmbedFSLoader, detects the system locale,
// and wires everything into a PackageTranslator with namespace scoping.
//
// Example — replaces the entire per-package i18n.go:
//
//	//go:embed locales/*.json
//	var localeFS embed.FS
//
//	var I18n = i18n.NewPackageTranslatorWithFS("filesystem", localeFS, "locales",
//	    i18n.WithDefaults(map[string]string{
//	        "error.empty_path": "path cannot be empty",
//	        "error.not_found":  "file not found: %s",
//	    }),
//	)
//
// The returned PackageTranslator:
//   - Loads translations from the embedded filesystem
//   - Auto-detects system locale (LC_ALL > LANG > en-US)
//   - Prefixes all keys with the namespace (e.g., "filesystem.error.empty_path")
//   - Falls back to defaults when translator is nil or key is missing
//   - Is thread-safe for concurrent use
//   - Implements TranslatorProvider for passing to other packages
func NewPackageTranslatorWithFS(namespace string, fs embed.FS, basePath string, opts ...PackageOption) *PackageTranslator {
	pt := NewPackageTranslator(namespace, opts...)
	if pt == nil {
		return nil
	}

	loader := NewEmbedFSLoader(fs, basePath)
	locale := DetectLocale()

	translator, err := New(
		WithLoader(loader),
		WithDefaultLocale("en-US"),
		WithCache(NewMapCache()),
	)
	if err != nil {
		// Return PackageTranslator in defaults-only mode on error.
		return pt
	}

	// Check if detected locale is available, fall back to en-US if not.
	if _, loadErr := loader.Load(locale); loadErr != nil {
		locale = "en-US"
	}
	translator.SetLocale(locale)

	pt.SetTranslator(translator)
	return pt
}

// DetectLocale creates a temporary DefaultLocaleDetector and returns the
// detected system locale. This eliminates per-package DetectLocale duplication.
func DetectLocale() string {
	d := NewDefaultLocaleDetector(nil)
	return d.Detect()
}

// GetSupportedLocales probes a core.TranslationLoader with the given candidate
// locale codes and returns those for which Load succeeds without error.
// Returns an empty slice (never nil) if no candidates are supported.
func GetSupportedLocales(loader core.TranslationLoader, locales ...string) []string {
	supported := make([]string, 0, len(locales))
	for _, locale := range locales {
		if _, err := loader.Load(locale); err == nil {
			supported = append(supported, locale)
		}
	}
	return supported
}
