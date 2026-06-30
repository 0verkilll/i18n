// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

package i18n

import (
	"embed"

	"github.com/0verkilll/i18n/internal/core"
	"github.com/0verkilll/i18n/internal/engine"
)

// =============================================================================
// Translator constructors and options
// =============================================================================

// New creates a new Translator with the given options.
func New(opts ...Option) (*Translator, error) {
	return engine.New(opts...)
}

// NewWithFS creates a Translator that loads translations from the filesystem.
// This is a convenience wrapper around New with WithFileSystemLoader and
// WithDefaultLocale pre-applied. Additional options can be provided to
// customize other components.
func NewWithFS(baseDir, defaultLocale string, opts ...Option) (*Translator, error) {
	return engine.NewWithFS(baseDir, defaultLocale, opts...)
}

// NewWithRegistry creates a Translator that loads translations from build-tag
// locale data registered at init time. This is a convenience wrapper around
// New with WithRegistryLoader and WithDefaultLocale pre-applied. Additional
// options can be provided to customize other components.
func NewWithRegistry(defaultLocale string, opts ...Option) (*Translator, error) {
	return engine.NewWithRegistry(defaultLocale, opts...)
}

// WithLoader sets a custom TranslationLoader.
func WithLoader(loader TranslationLoader) Option { return engine.WithLoader(loader) }

// WithParser sets a custom TranslationParser.
func WithParser(parser TranslationParser) Option { return engine.WithParser(parser) }

// WithRegisteredParser selects a parser from the registry by file extension.
func WithRegisteredParser(ext string) Option { return engine.WithRegisteredParser(ext) }

// WithResolver sets a custom KeyResolver.
func WithResolver(resolver KeyResolver) Option { return engine.WithResolver(resolver) }

// WithLocaleDetector sets a custom LocaleDetector.
func WithLocaleDetector(detector LocaleDetector) Option { return engine.WithLocaleDetector(detector) }

// WithFallbackChainer sets a custom FallbackChainer.
func WithFallbackChainer(chainer FallbackChainer) Option {
	return engine.WithFallbackChainer(chainer)
}

// WithDefaultLocale sets the default locale.
func WithDefaultLocale(locale string) Option { return engine.WithDefaultLocale(locale) }

// WithFileSystemLoader creates a FileSystemLoader for the given base directory.
func WithFileSystemLoader(baseDir string) Option { return engine.WithFileSystemLoader(baseDir) }

// WithLogger sets a custom Logger.
func WithLogger(l Logger) Option { return engine.WithLogger(l) }

// WithPluralResolver sets a custom PluralResolver.
func WithPluralResolver(resolver PluralResolver) Option {
	return engine.WithPluralResolver(resolver)
}

// WithCache sets a Cacher for resolved translations.
func WithCache(cache Cacher) Option { return engine.WithCache(cache) }

// WithRegistryLoader creates a RegistryLoader for build-tag locale data.
func WithRegistryLoader() Option { return engine.WithRegistryLoader() }

// =============================================================================
// Cache constructors
// =============================================================================

// NewMapCache creates a MapCache with no size limit.
func NewMapCache() *MapCache { return engine.NewMapCache() }

// NewMapCacheWithLimit creates a MapCache with LRU eviction.
func NewMapCacheWithLimit(maxEntries int) *MapCache {
	return engine.NewMapCacheWithLimit(maxEntries)
}

// =============================================================================
// Resolver constructors
// =============================================================================

// NewDefaultKeyResolver creates a new dot-notation key resolver.
func NewDefaultKeyResolver() *DefaultKeyResolver { return engine.NewDefaultKeyResolver() }

// NewDefaultFallbackChainer creates a new locale fallback chainer.
func NewDefaultFallbackChainer() *DefaultFallbackChainer { return engine.NewDefaultFallbackChainer() }

// =============================================================================
// Plural constructors
// =============================================================================

// NewDefaultPluralResolver creates a CLDR plural resolver for 36 languages.
func NewDefaultPluralResolver() *DefaultPluralResolver { return engine.NewDefaultPluralResolver() }

// =============================================================================
// Parser constructors and registry
// =============================================================================

// NewJSONParser creates a new JSON translation file parser.
func NewJSONParser() *JSONParser { return engine.NewJSONParser() }

// NewBinaryParser creates a new compact binary translation file parser.
// The binary format reduces both data size (~40% smaller than JSON) and
// parser code size. Use EncodeBinary and FlattenKeys to produce binary
// translation data at build time.
//
// To register for automatic use with file-based loaders:
//
//	i18n.RegisterParser(".bin", i18n.NewBinaryParser())
func NewBinaryParser() *BinaryParser { return engine.NewBinaryParser() }

// EncodeBinary converts a flat map of translations to the compact binary format.
// Keys must be in dot notation (use FlattenKeys to convert nested maps).
// This function is intended for build-time tooling to convert JSON locale files.
func EncodeBinary(translations map[string]string) ([]byte, error) {
	return engine.EncodeBinary(translations)
}

// FlattenKeys converts a nested map to a flat dot-notation map.
// For example, {"error": {"required": "..."}} becomes {"error.required": "..."}.
// Use this together with EncodeBinary to convert JSON translation data.
func FlattenKeys(nested map[string]interface{}) map[string]string {
	return engine.FlattenKeys(nested)
}

// NewRegistry creates a new parser registry.
func NewRegistry() *Registry { return engine.NewRegistry() }

// RegisterParser registers a parser for a file extension in the default registry.
func RegisterParser(ext string, p TranslationParser) error {
	return engine.RegisterParser(ext, p)
}

// GetParser retrieves a parser for a file extension from the default registry.
func GetParser(ext string) (TranslationParser, error) { return engine.GetParser(ext) }

// RegisteredFormats returns all registered file extensions.
func RegisteredFormats() []string { return engine.RegisteredFormats() }

// =============================================================================
// Loader constructors
// =============================================================================

// NewEmbedFSLoader creates a loader for embedded filesystem translations.
func NewEmbedFSLoader(fs embed.FS, basePath string, opts ...LoaderOption) *EmbedFSLoader {
	return engine.NewEmbedFSLoader(fs, basePath, opts...)
}

// NewRegistryLoader creates a loader for build-tag locale data.
func NewRegistryLoader() *RegistryLoader { return engine.NewRegistryLoader() }

// NewFileSystemLoader creates a loader for filesystem-based translations.
func NewFileSystemLoader(baseDir string, opts ...LoaderOption) *FileSystemLoader {
	return engine.NewFileSystemLoader(baseDir, opts...)
}

// WithExtension sets the file extension used by a loader.
func WithExtension(ext string) LoaderOption { return engine.WithExtension(ext) }

// RegisteredLocales returns all locale codes in the registry.
func RegisteredLocales() []string { return engine.RegisteredLocales() }

// =============================================================================
// Locale detection
// =============================================================================

// NewDefaultLocaleDetector creates a locale detector using environment variables.
func NewDefaultLocaleDetector(env EnvProvider) *DefaultLocaleDetector {
	return engine.NewDefaultLocaleDetector(env)
}

// NewAcceptLanguageDetector creates a detector from an Accept-Language header.
func NewAcceptLanguageDetector(header string) *AcceptLanguageDetector {
	return engine.NewAcceptLanguageDetector(header)
}

// NewChainDetector creates a detector that chains multiple detectors.
func NewChainDetector(detectors ...LocaleDetector) *ChainDetector {
	return engine.NewChainDetector(detectors...)
}

// NewStaticDetector creates a detector that returns a fixed locale.
func NewStaticDetector(locale string) *StaticDetector { return engine.NewStaticDetector(locale) }

// NewBrowserDetector creates a detector that reads navigator.language in js/wasm
// builds. On non-WASM platforms, Detect returns "". Use in a ChainDetector so
// a fallback detector provides the locale outside browser environments.
func NewBrowserDetector() *BrowserDetector { return engine.NewBrowserDetector() }

// NormalizeLocale converts a locale string to BCP 47 format.
func NormalizeLocale(locale string) string { return engine.NormalizeLocale(locale) }

// DetectLocale detects the system locale using default detection.
func DetectLocale() string { return engine.DetectLocale() }

// =============================================================================
// Logger
// =============================================================================

// SetLogger sets the global logger.
func SetLogger(l Logger) { engine.SetLogger(l) }

// GetLogger returns the global logger.
func GetLogger() Logger { return engine.GetLogger() }

// =============================================================================
// Package translator and namespace
// =============================================================================

// NewNamespace creates a Namespace that prefixes all keys with the given prefix.
func NewNamespace(prefix string, t TranslatorProvider) (*Namespace, error) {
	return engine.NewNamespace(prefix, t)
}

// NewPackageTranslator creates a PackageTranslator with namespace scoping.
func NewPackageTranslator(namespace string, opts ...PackageOption) *PackageTranslator {
	return engine.NewPackageTranslator(namespace, opts...)
}

// WithDefaults sets hardcoded fallback strings on a PackageTranslator.
func WithDefaults(defaults map[string]string) PackageOption { return engine.WithDefaults(defaults) }

// WithTranslator sets the initial translator on a PackageTranslator.
func WithTranslator(t TranslatorProvider) PackageOption { return engine.WithTranslator(t) }

// NewPackageTranslatorWithFS creates a fully-configured PackageTranslator backed
// by an embedded filesystem. This replaces ~300 lines of per-package i18n.go
// boilerplate with a single constructor call.
//
// Example — replaces the entire per-package i18n.go:
//
//	//go:embed locales/*.json
//	var localeFS embed.FS
//
//	var I18n = i18n.NewPackageTranslatorWithFS("filesystem", localeFS, "locales",
//	    i18n.WithDefaults(map[string]string{
//	        "error.empty_path": "path cannot be empty",
//	    }),
//	)
func NewPackageTranslatorWithFS(namespace string, fs embed.FS, basePath string, opts ...PackageOption) *PackageTranslator {
	return engine.NewPackageTranslatorWithFS(namespace, fs, basePath, opts...)
}

// DiscoverLocales reads an embed.FS directory and returns locale codes from
// JSON filenames. Replaces per-package GetSupportedLocales reimplementations.
func DiscoverLocales(fs embed.FS, basePath string) []string {
	return engine.DiscoverLocales(fs, basePath)
}

// GetSupportedLocales probes a loader for supported locale codes.
func GetSupportedLocales(loader TranslationLoader, locales ...string) []string {
	return engine.GetSupportedLocales(loader, locales...)
}

// =============================================================================
// Struct-based translations (zero-cost lookups for games and real-time apps)
// =============================================================================

// StructTranslator provides zero-cost translation lookups using Go struct field
// access (0.25 ns) instead of map lookups (25 ns). Locale switching is an
// atomic pointer swap — safe from any goroutine with zero locking overhead.
//
// Define a struct with string fields for each translation key, create one
// instance per locale with build tags, and use StructTranslator to switch:
//
//	type Messages struct {
//	    Greeting string
//	    Farewell string
//	}
//
//	//go:build locale_en_us || locale_all
//	var enUS = Messages{Greeting: "Hello", Farewell: "Goodbye"}
//
//	var Msg = i18n.NewStructTranslator(&enUS)
//
//	// Read (0.25 ns per lookup):
//	fmt.Println(Msg.Get().Greeting)
//
//	// Switch locale atomically:
//	Msg.Set(&esES)
type StructTranslator[T any] = engine.StructTranslator[T]

// LocaleSet holds named locale structs for lookup by locale code.
type LocaleSet[T any] = engine.LocaleSet[T]

// NewStructTranslator creates a StructTranslator with the given initial locale.
func NewStructTranslator[T any](initial *T) *StructTranslator[T] {
	return engine.NewStructTranslator(initial)
}

// NewLocaleSet creates a LocaleSet with a fallback locale for struct-based translations.
func NewLocaleSet[T any](fallbackCode string, fallback *T) *LocaleSet[T] {
	return engine.NewLocaleSet(fallbackCode, fallback)
}

// =============================================================================
// Validation and sanitization
// =============================================================================

// ValidateLocale validates a locale code against BCP 47 format.
func ValidateLocale(locale string) error { return core.ValidateLocale(locale) }

// ValidateKey validates a translation key.
func ValidateKey(key string) error { return core.ValidateKey(key) }

// ValidateFormatString validates a format string for safety.
func ValidateFormatString(format string, argCount int) error {
	return core.ValidateFormatString(format, argCount)
}

// SanitizeOutput removes potentially dangerous characters from translation output.
func SanitizeOutput(s string) string { return core.SanitizeOutput(s) }
