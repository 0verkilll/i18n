// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

package engine

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/0verkilll/i18n/internal/core"
)

// Translator is the main type for translation operations.
// It coordinates between loaders, parsers, and resolvers to provide
// a complete translation service with fallback support and thread safety.
type Translator struct {
	loader           core.TranslationLoader
	parser           core.TranslationParser
	resolver         core.KeyResolver
	detector         core.LocaleDetector
	chainer          core.FallbackChainer
	logger           core.Logger
	pluralResolver   core.PluralResolver
	translationCache core.Cacher
	cache            map[string]map[string]interface{}
	currentLocale    atomic.Value // stores string; read without mutex on hot path
	defaultLocale    string
	mu               sync.Mutex   // protects locale writes only (SetLocale)
	cacheMu          sync.RWMutex // protects the translations cache map separately from locale state
}

// Compile-time assertions that Translator implements TranslatorProvider and all sub-interfaces.
var (
	_ core.TranslatorProvider  = (*Translator)(nil)
	_ core.TranslationLookup   = (*Translator)(nil)
	_ core.FormattedTranslator = (*Translator)(nil)
	_ core.KeyChecker          = (*Translator)(nil)
	_ core.LocaleSetter        = (*Translator)(nil)
	_ core.LocaleGetter        = (*Translator)(nil)
	_ core.PluralTranslator    = (*Translator)(nil)
	_ core.GenderTranslator    = (*Translator)(nil)
)

// Option is a functional option for configuring the Translator.
type Option func(*Translator) error

// WithLoader sets a custom TranslationLoader.
func WithLoader(loader core.TranslationLoader) Option {
	return func(t *Translator) error {
		if loader == nil {
			return fmt.Errorf("loader cannot be nil")
		}
		t.loader = loader
		return nil
	}
}

// WithParser sets a custom TranslationParser.
func WithParser(parser core.TranslationParser) Option {
	return func(t *Translator) error {
		if parser == nil {
			return fmt.Errorf("parser cannot be nil")
		}
		t.parser = parser
		return nil
	}
}

// WithRegisteredParser resolves a TranslationParser from the default package-level
// registry by file extension and sets it on the Translator. The extension must
// already be registered (e.g., ".json" is registered by default; external modules
// register additional formats in their init() functions).
//
// Returns an error wrapping ErrUnknownFormat if no parser is registered for ext.
func WithRegisteredParser(ext string) Option {
	return func(t *Translator) error {
		p, err := GetParser(ext)
		if err != nil {
			return fmt.Errorf("registered parser lookup failed: %w", err)
		}
		t.parser = p
		return nil
	}
}

// WithResolver sets a custom KeyResolver.
func WithResolver(resolver core.KeyResolver) Option {
	return func(t *Translator) error {
		if resolver == nil {
			return fmt.Errorf("resolver cannot be nil")
		}
		t.resolver = resolver
		return nil
	}
}

// WithLocaleDetector sets a custom LocaleDetector.
func WithLocaleDetector(detector core.LocaleDetector) Option {
	return func(t *Translator) error {
		if detector == nil {
			return fmt.Errorf("detector cannot be nil")
		}
		t.detector = detector
		return nil
	}
}

// WithFallbackChainer sets a custom FallbackChainer.
func WithFallbackChainer(chainer core.FallbackChainer) Option {
	return func(t *Translator) error {
		if chainer == nil {
			return fmt.Errorf("chainer cannot be nil")
		}
		t.chainer = chainer
		return nil
	}
}

// WithDefaultLocale sets the default locale for the translator.
func WithDefaultLocale(locale string) Option {
	return func(t *Translator) error {
		if err := core.ValidateLocale(locale); err != nil {
			return fmt.Errorf("invalid default locale: %w", err)
		}
		t.defaultLocale = locale
		t.currentLocale.Store(locale)
		return nil
	}
}

// WithFileSystemLoader creates a FileSystemLoader with the given base directory.
// This is a convenience function that combines loader creation with configuration.
func WithFileSystemLoader(baseDir string) Option {
	return func(t *Translator) error {
		loader := NewFileSystemLoader(baseDir)
		t.loader = loader
		return nil
	}
}

// WithLogger sets a custom Logger for the Translator instance.
// If l is nil, NopLogger is used (silent operation).
// This logger takes precedence over the package-level logger set via SetLogger.
func WithLogger(l core.Logger) Option {
	return func(t *Translator) error {
		if l == nil {
			t.logger = NopLogger{}
		} else {
			t.logger = l
		}
		return nil
	}
}

// WithPluralResolver sets a custom PluralResolver.
func WithPluralResolver(resolver core.PluralResolver) Option {
	return func(t *Translator) error {
		if resolver == nil {
			return fmt.Errorf("plural resolver cannot be nil")
		}
		t.pluralResolver = resolver
		return nil
	}
}

// WithCache sets a Cacher for resolved-translation caching.
// When enabled, Translate, TranslatePlural, and TranslateGender cache their
// results to avoid repeated fallback chain traversal and key resolution.
// Caching is opt-in; when not set, the Translator has zero caching overhead.
func WithCache(cache core.Cacher) Option {
	return func(t *Translator) error {
		if cache == nil {
			return fmt.Errorf("cache cannot be nil")
		}
		t.translationCache = cache
		return nil
	}
}

// setDefaults fills in default implementations for any unset optional components.
func (t *Translator) setDefaults() {
	if t.parser == nil {
		t.parser = NewJSONParser()
	}
	if t.resolver == nil {
		t.resolver = NewDefaultKeyResolver()
	}
	if t.detector == nil {
		t.detector = NewDefaultLocaleDetector(nil)
	}
	if t.chainer == nil {
		t.chainer = NewDefaultFallbackChainer()
	}
	if t.logger == nil {
		t.logger = GetLogger()
	}
	if t.pluralResolver == nil {
		t.pluralResolver = NewDefaultPluralResolver()
	}
}

// locale returns the current locale string stored atomically.
func (t *Translator) locale() string {
	v := t.currentLocale.Load()
	if v == nil {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

// New creates a new Translator with the given options.
// At minimum, a loader must be provided (either through WithLoader or WithFileSystemLoader).
// Other components will use default implementations if not specified.
func New(opts ...Option) (*Translator, error) {
	t := &Translator{
		cache: make(map[string]map[string]interface{}),
	}

	for _, opt := range opts {
		if err := opt(t); err != nil {
			if t.logger != nil {
				t.logger.Error("failed to apply option", "error", err)
			}
			return nil, err
		}
	}

	if t.loader == nil {
		return nil, fmt.Errorf("loader is required (use WithLoader or WithFileSystemLoader)")
	}

	t.setDefaults()

	if t.defaultLocale == "" {
		detected := t.detector.Detect()
		normalized := t.detector.Normalize(detected)
		t.defaultLocale = normalized
		t.currentLocale.Store(normalized)
	}

	t.logger.Debug("translator created",
		"locale", t.locale(),
		"loader_type", fmt.Sprintf("%T", t.loader))

	return t, nil
}

// NewWithFS creates a Translator that loads translations from the filesystem.
// This is a convenience wrapper around New with WithFileSystemLoader and
// WithDefaultLocale pre-applied. Additional options can be provided to
// customize other components.
func NewWithFS(baseDir, defaultLocale string, opts ...Option) (*Translator, error) {
	allOpts := make([]Option, 0, 2+len(opts))
	allOpts = append(allOpts, WithFileSystemLoader(baseDir), WithDefaultLocale(defaultLocale))
	allOpts = append(allOpts, opts...)
	return New(allOpts...)
}

// NewWithRegistry creates a Translator that loads translations from build-tag
// locale data registered at init time. This is a convenience wrapper around
// New with WithRegistryLoader, WithParser(BinaryParser), and WithDefaultLocale
// pre-applied. The BinaryParser is used because built-in locale data is stored
// in the compact binary format. Additional options can be provided to customize
// other components; a user-supplied WithParser overrides the default.
func NewWithRegistry(defaultLocale string, opts ...Option) (*Translator, error) {
	allOpts := make([]Option, 0, 3+len(opts))
	allOpts = append(allOpts, WithRegistryLoader(), WithParser(NewBinaryParser()), WithDefaultLocale(defaultLocale))
	allOpts = append(allOpts, opts...)
	return New(allOpts...)
}

// SetLocale changes the current locale for translation lookups.
// The locale is normalized then validated; invalid locales are silently rejected
// (the current locale is preserved and a warning is logged).
// If a translation cache is configured, it is invalidated on a successful change.
//
// NOTE: SetLocale mutates shared Translator state. In multi-tenant HTTP
// request scenarios where each request needs a distinct locale, prefer
// WithLocaleContext to obtain a per-request, locale-scoped view without
// mutating the shared Translator.
func (t *Translator) SetLocale(locale string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	oldLocale := t.locale()
	normalized := t.detector.Normalize(locale)

	if err := core.ValidateLocale(normalized); err != nil {
		t.logger.Warn("SetLocale: rejecting invalid locale",
			"locale", locale,
			"normalized", normalized,
			"error", err)
		return
	}

	t.currentLocale.Store(normalized)

	if t.translationCache != nil {
		t.translationCache.Invalidate()
	}

	t.logger.Info("locale changed",
		"old_locale", oldLocale,
		"new_locale", normalized)
}

// ReloadLocale forces a reload of translations for the given locale
// from the underlying loader. This clears both the internal translation
// cache for that locale and the resolved-translation cache.
func (t *Translator) ReloadLocale(locale string) error {
	t.cacheMu.Lock()
	delete(t.cache, locale)
	t.cacheMu.Unlock()

	if t.translationCache != nil {
		t.translationCache.Invalidate()
	}

	// Pre-load the locale to catch errors early.
	t.cacheMu.Lock()
	_, err := t.loadTranslations(locale)
	t.cacheMu.Unlock()

	return err
}

// GetLocale returns the current locale being used for translations.
func (t *Translator) GetLocale() string {
	return t.locale()
}

// loadTranslations loads and caches translations for the given locale.
// It must be called with cacheMu write lock held.
func (t *Translator) loadTranslations(locale string) (map[string]interface{}, error) {
	if translations, exists := t.cache[locale]; exists {
		t.logger.Debug("cache hit for translations", "locale", locale)
		return translations, nil
	}

	t.logger.Debug("loading translations from source", "locale", locale)

	data, err := t.loader.Load(locale)
	if err != nil {
		t.logger.Error("failed to load translations", "locale", locale, "error", err)
		return nil, err
	}

	translations, err := t.parser.Parse(data)
	if err != nil {
		t.logger.Error("failed to parse translations", "locale", locale, "error", err)
		return nil, err
	}

	if len(t.cache) >= MaxRegisteredLocales {
		t.logger.Warn("translation cache at capacity, rejecting locale",
			"locale", locale,
			"limit", MaxRegisteredLocales)
		return nil, fmt.Errorf("translation cache capacity exceeded (max %d locales)", MaxRegisteredLocales)
	}

	t.cache[locale] = translations

	t.logger.Debug("translations loaded", "locale", locale, "key_count", len(translations))

	return translations, nil
}

// ensureLoaded loads translations for the locale if not already cached.
// Returns the map reference with all locks released. The returned map is safe
// to read without holding a lock because per-locale maps are immutable after
// creation; ReloadLocale replaces the cache entry under a write lock without
// ever mutating a previously-returned map object.
func (t *Translator) ensureLoaded(locale string) (map[string]interface{}, error) {
	// Fast path: read under lock, release before returning.
	t.cacheMu.RLock()
	if translations, exists := t.cache[locale]; exists {
		t.cacheMu.RUnlock()
		return translations, nil
	}
	t.cacheMu.RUnlock()

	// Slow path: load under write lock, release before returning.
	t.cacheMu.Lock()
	translations, err := t.loadTranslations(locale)
	t.cacheMu.Unlock()

	return translations, err
}

// resolveKeyInLocale attempts to resolve a key from a specific locale's translations.
// Returns the resolved value and true if found, or empty string and false otherwise.
func (t *Translator) resolveKeyInLocale(fallbackLocale, key string) (string, bool) {
	translations, err := t.ensureLoaded(fallbackLocale)
	if err != nil {
		return "", false
	}

	value, err := t.resolver.Resolve(translations, key)
	if err != nil {
		return "", false
	}

	return value, true
}

// Translate looks up a translation key in the current locale.
// If the key is not found, it tries the fallback chain.
// If the key is still not found, it returns the key itself.
// When a translation cache is configured, resolved values are cached to
// skip fallback chain traversal on subsequent calls for the same key.
func (t *Translator) Translate(key string) string {
	return t.translateInLocale(t.locale(), key)
}

// translateInLocale is the locale-parameterized implementation of Translate.
// It uses the passed-in locale instead of reading the Translator's shared
// locale state, enabling per-request locale scoping without mutation.
func (t *Translator) translateInLocale(locale, key string) string {
	// Simple string concat is zero-alloc for short strings on modern Go
	// compilers. The empty string acts as a sentinel: skip cache ops when
	// translationCache is nil.
	cacheKey := ""
	if t.translationCache != nil {
		cacheKey = locale + ":" + key
		if cached, ok := t.getCached(cacheKey); ok {
			return cached
		}
	}

	if t.logger.Enabled(core.LevelDebug) {
		t.logger.Debug("translating key", "key", key, "locale", locale)
	}

	for i, fallbackLocale := range t.chainer.GetChain(locale) {
		value, found := t.resolveKeyInLocale(fallbackLocale, key)
		if !found {
			continue
		}
		t.logFallback(i, key, locale, fallbackLocale)
		result := core.SanitizeOutput(value)
		if cacheKey != "" {
			t.cacheResult(cacheKey, key, result)
		}
		return result
	}

	if t.logger.Enabled(core.LevelDebug) {
		t.logger.Debug("key not found in any locale", "key", key)
	}
	return key
}

// getCached returns a cached translation if the cache is enabled and the key exists.
func (t *Translator) getCached(cacheKey string) (string, bool) {
	if t.translationCache != nil {
		return t.translationCache.Get(cacheKey)
	}
	return "", false
}

// cacheResult stores a sanitized result in the translation cache when caching
// is enabled and the result differs from the raw key (indicating a successful lookup).
func (t *Translator) cacheResult(cacheKey, key, result string) {
	if t.translationCache != nil && result != key {
		t.translationCache.Set(cacheKey, result)
	}
}

// logFallback emits a warning when a fallback locale was used (index > 0).
func (t *Translator) logFallback(index int, key, primary, fallback string) {
	if index > 0 {
		t.logger.Warn("key not found, using fallback",
			"key", key,
			"primary_locale", primary,
			"fallback_locale", fallback)
	}
	if t.logger.Enabled(core.LevelDebug) {
		t.logger.Debug("translation found", "key", key, "locale", fallback)
	}
}

// TranslateWithArgs looks up a translation key and formats it with the given arguments.
// It uses fmt.Sprintf for formatting, and validates the format string before use.
// If the key is not found, it returns the key itself without formatting.
// This method is not individually cached because args are arbitrary interface{}
// values that are not safely serializable without reflect. It benefits indirectly
// from the Translate() cache on the format string lookup.
func (t *Translator) TranslateWithArgs(key string, args ...interface{}) string {
	return t.translateWithArgsInLocale(t.locale(), key, args...)
}

// translateWithArgsInLocale is the locale-parameterized implementation.
func (t *Translator) translateWithArgsInLocale(locale, key string, args ...interface{}) string {
	format := t.translateInLocale(locale, key)

	if format == key {
		return key
	}

	if err := core.ValidateFormatString(format, len(args)); err != nil {
		return format
	}

	result := fmt.Sprintf(format, args...)
	return core.SanitizeOutput(result)
}

// TranslatePlural resolves the plural category for the current locale and count,
// looks up key.<category> (falling back to key.other in the SAME locale, then
// walking the fallback chain), replaces # with the count, sanitizes the output,
// and traverses the fallback chain if needed.
// When a translation cache is configured, resolved values are cached with a
// count-differentiated key to skip repeated lookups.
func (t *Translator) TranslatePlural(key string, count interface{}) string {
	return t.translatePluralInLocale(t.locale(), key, count)
}

// translatePluralInLocale is the locale-parameterized implementation.
func (t *Translator) translatePluralInLocale(locale, key string, count interface{}) string {
	countStr := formatCount(count)

	// Skip cache-key allocation when caching is disabled.
	if t.translationCache == nil {
		category := t.pluralResolver.Resolve(locale, count)
		chain := t.chainer.GetChain(locale)
		for _, fallbackLocale := range chain {
			if result, found := t.resolvePluralInLocale(fallbackLocale, key, category, countStr); found {
				return result
			}
		}
		return key
	}

	cacheKey := locale + ":" + key + "#" + countStr

	if cached, ok := t.getCached(cacheKey); ok {
		return cached
	}

	category := t.pluralResolver.Resolve(locale, count)
	chain := t.chainer.GetChain(locale)

	for _, fallbackLocale := range chain {
		if result, found := t.resolvePluralInLocale(fallbackLocale, key, category, countStr); found {
			t.cacheResult(cacheKey, key, result)
			return result
		}
	}

	return key
}

// resolvePluralInLocale tries key.<category> then key.other in the given locale,
// replacing # with the formatted count and sanitizing the result.
//
// This ordering is deliberate: for a given locale, we exhaustively try the
// resolved category then the "other" fallback BEFORE the caller advances to
// the next locale in the fallback chain. This matches CLDR guidance: the
// current locale's "other" form is always preferable to walking into a
// different language's translations.
func (t *Translator) resolvePluralInLocale(locale, key string, category core.PluralCategory, countStr string) (string, bool) {
	categoryKey := key + "." + string(category)
	if value, found := t.resolveKeyInLocale(locale, categoryKey); found {
		return core.SanitizeOutput(strings.ReplaceAll(value, "#", countStr)), true
	}
	otherKey := key + "." + string(core.Other)
	if value, found := t.resolveKeyInLocale(locale, otherKey); found {
		return core.SanitizeOutput(strings.ReplaceAll(value, "#", countStr)), true
	}
	return "", false
}

// TranslatePluralWithArgs resolves the plural category, looks up the translation,
// replaces # with count, then applies fmt.Sprintf with the provided args.
// This method is not individually cached because args are arbitrary interface{}
// values that are not safely serializable without reflect.
//
// For a given locale, both key.<category> and key.other are tried before
// advancing to the next locale in the fallback chain.
func (t *Translator) TranslatePluralWithArgs(key string, count interface{}, args ...interface{}) string {
	return t.translatePluralWithArgsInLocale(t.locale(), key, count, args...)
}

// translatePluralWithArgsInLocale is the locale-parameterized implementation.
func (t *Translator) translatePluralWithArgsInLocale(locale, key string, count interface{}, args ...interface{}) string {
	category := t.pluralResolver.Resolve(locale, count)
	countStr := formatCount(count)

	chain := t.chainer.GetChain(locale)

	for _, fallbackLocale := range chain {
		// Try key.<category> first
		categoryKey := key + "." + string(category)
		if value, found := t.resolveKeyInLocale(fallbackLocale, categoryKey); found {
			template := strings.ReplaceAll(value, "#", countStr)
			if err := core.ValidateFormatString(template, len(args)); err != nil {
				return core.SanitizeOutput(template)
			}
			result := fmt.Sprintf(template, args...)
			return core.SanitizeOutput(result)
		}

		// Fall back to key.other in the SAME locale before moving to the next
		// fallback locale. This ensures the current locale's "other" form is
		// preferred over a different locale's translation.
		otherKey := key + "." + string(core.Other)
		if value, found := t.resolveKeyInLocale(fallbackLocale, otherKey); found {
			template := strings.ReplaceAll(value, "#", countStr)
			if err := core.ValidateFormatString(template, len(args)); err != nil {
				return core.SanitizeOutput(template)
			}
			result := fmt.Sprintf(template, args...)
			return core.SanitizeOutput(result)
		}
	}

	return key
}

// TranslateGender looks up key.<gender>, falls back to key.other,
// sanitizes the output, and traverses the fallback chain.
// When a translation cache is configured, resolved values are cached with a
// gender-differentiated key to skip repeated lookups.
func (t *Translator) TranslateGender(key string, gender core.GenderCategory) string {
	return t.translateGenderInLocale(t.locale(), key, gender)
}

// translateGenderInLocale is the locale-parameterized implementation.
func (t *Translator) translateGenderInLocale(locale, key string, gender core.GenderCategory) string {
	// Skip cache-key allocation when caching is disabled.
	if t.translationCache == nil {
		chain := t.chainer.GetChain(locale)
		for _, fallbackLocale := range chain {
			if result, found := t.resolveGenderInLocale(fallbackLocale, key, gender); found {
				return result
			}
		}
		return key
	}

	cacheKey := locale + ":" + key + "@" + string(gender)
	if cached, ok := t.getCached(cacheKey); ok {
		return cached
	}

	chain := t.chainer.GetChain(locale)

	for _, fallbackLocale := range chain {
		if result, found := t.resolveGenderInLocale(fallbackLocale, key, gender); found {
			t.cacheResult(cacheKey, key, result)
			return result
		}
	}

	return key
}

// resolveGenderInLocale tries key.<gender> then key.other in the given locale
// and sanitizes the result.
func (t *Translator) resolveGenderInLocale(locale, key string, gender core.GenderCategory) (string, bool) {
	genderKey := key + "." + string(gender)
	if value, found := t.resolveKeyInLocale(locale, genderKey); found {
		return core.SanitizeOutput(value), true
	}
	otherKey := key + "." + string(core.GenderOther)
	if value, found := t.resolveKeyInLocale(locale, otherKey); found {
		return core.SanitizeOutput(value), true
	}
	return "", false
}

// TranslateWithMessage resolves a key to an ICU MessageFormat template and evaluates
// it with the provided named arguments. For plural expressions, the Translator's
// PluralResolver determines the category. Falls back through the locale chain.
// This method is not individually cached because args are arbitrary interface{}
// values that are not safely serializable without reflect.
func (t *Translator) TranslateWithMessage(key string, args map[string]interface{}) string {
	return t.translateWithMessageInLocale(t.locale(), key, args)
}

// translateWithMessageInLocale is the locale-parameterized implementation.
func (t *Translator) translateWithMessageInLocale(locale, key string, args map[string]interface{}) string {
	chain := t.chainer.GetChain(locale)

	for _, fallbackLocale := range chain {
		value, found := t.resolveKeyInLocale(fallbackLocale, key)
		if !found {
			continue
		}

		if !isICUMessageFormat(value) {
			return core.SanitizeOutput(value)
		}

		result := evaluateICUMessage(value, args, locale, t.pluralResolver)
		return core.SanitizeOutput(result)
	}

	return key
}

// HasKey checks if a translation key exists in the current locale or its fallback chain.
func (t *Translator) HasKey(key string) bool {
	return t.hasKeyInLocale(t.locale(), key)
}

// hasKeyInLocale is the locale-parameterized implementation of HasKey.
func (t *Translator) hasKeyInLocale(locale, key string) bool {
	chain := t.chainer.GetChain(locale)

	for _, fallbackLocale := range chain {
		if _, found := t.resolveKeyInLocale(fallbackLocale, key); found {
			return true
		}
	}

	return false
}

// =============================================================================
// ContextTranslator
// =============================================================================

// ContextTranslator wraps a Translator and passes a context.Context to the
// logger for trace correlation and observability integration. Since translations
// are resolved from in-memory data (no I/O), context cancellation does not
// apply. The context is used solely for structured logging enrichment.
//
// A ContextTranslator may optionally carry a locale override. When set, all
// translation operations route through that locale instead of reading the
// underlying Translator's shared locale state. This enables per-request
// locale scoping (e.g., in HTTP middleware) without mutating the shared
// Translator, which would race between concurrent requests.
//
// ContextTranslator implements core.TranslatorProvider so it is substitutable
// anywhere a TranslatorProvider is accepted.
type ContextTranslator struct {
	translator *Translator
	ctx        context.Context
	// locale, when non-empty, overrides the underlying Translator's locale
	// for all operations on this ContextTranslator. It is normalized at
	// construction time.
	locale string
}

// Compile-time assertion that ContextTranslator implements core.TranslatorProvider.
var _ core.TranslatorProvider = (*ContextTranslator)(nil)

// WithContext returns a ContextTranslator that passes ctx to the Translator's
// logger for trace correlation. The returned ContextTranslator delegates all
// translation operations to the underlying Translator using the Translator's
// current shared locale.
func (t *Translator) WithContext(ctx context.Context) *ContextTranslator {
	return &ContextTranslator{
		translator: t,
		ctx:        ctx,
	}
}

// WithLocaleContext returns a ContextTranslator scoped to a specific locale
// AND context. Unlike SetLocale (which mutates shared Translator state),
// WithLocaleContext creates a lightweight per-call view whose translation
// operations use the supplied locale. This is the recommended approach for
// request-scoped locale selection (e.g., from an Accept-Language header) in
// concurrent HTTP handlers.
//
// The locale is normalized and validated. An invalid locale logs a warning and
// falls back to the Translator's current shared locale.
func (t *Translator) WithLocaleContext(ctx context.Context, locale string) *ContextTranslator {
	normalized := ""
	if locale != "" && t.detector != nil {
		normalized = t.detector.Normalize(locale)
		if err := core.ValidateLocale(normalized); err != nil {
			t.logger.Warn("WithLocaleContext: rejecting invalid locale, using shared locale",
				"locale", locale,
				"normalized", normalized,
				"error", err)
			normalized = ""
		}
	}
	return &ContextTranslator{
		translator: t,
		ctx:        ctx,
		locale:     normalized,
	}
}

// effectiveLocale returns the locale the ContextTranslator should use for
// lookups: the per-instance override if set, otherwise the underlying
// Translator's current locale.
func (ct *ContextTranslator) effectiveLocale() string {
	if ct.locale != "" {
		return ct.locale
	}
	return ct.translator.locale()
}

// contextLogger returns the Translator's logger enriched with the stored context.
func (ct *ContextTranslator) contextLogger() core.Logger {
	return ct.translator.logger.WithContext(ct.ctx)
}

// Translate delegates to the underlying Translator, scoped to the
// ContextTranslator's effective locale. The context is passed to the logger
// for trace correlation.
func (ct *ContextTranslator) Translate(key string) string {
	logger := ct.contextLogger()
	if logger.Enabled(core.LevelDebug) {
		logger.Debug("translating key with context", "key", key)
	}
	return ct.translator.translateInLocale(ct.effectiveLocale(), key)
}

// TranslateWithArgs delegates to the underlying Translator, scoped to the
// ContextTranslator's effective locale. The context is passed to the logger
// for trace correlation.
func (ct *ContextTranslator) TranslateWithArgs(key string, args ...interface{}) string {
	logger := ct.contextLogger()
	if logger.Enabled(core.LevelDebug) {
		logger.Debug("translating key with args and context", "key", key)
	}
	return ct.translator.translateWithArgsInLocale(ct.effectiveLocale(), key, args...)
}

// TranslatePlural delegates to the underlying Translator, scoped to the
// ContextTranslator's effective locale. The context is passed to the logger
// for trace correlation.
func (ct *ContextTranslator) TranslatePlural(key string, count interface{}) string {
	logger := ct.contextLogger()
	if logger.Enabled(core.LevelDebug) {
		logger.Debug("translating plural key with context", "key", key)
	}
	return ct.translator.translatePluralInLocale(ct.effectiveLocale(), key, count)
}

// TranslateGender delegates to the underlying Translator, scoped to the
// ContextTranslator's effective locale. The context is passed to the logger
// for trace correlation.
func (ct *ContextTranslator) TranslateGender(key string, gender core.GenderCategory) string {
	logger := ct.contextLogger()
	if logger.Enabled(core.LevelDebug) {
		logger.Debug("translating gender key with context", "key", key)
	}
	return ct.translator.translateGenderInLocale(ct.effectiveLocale(), key, gender)
}

// HasKey checks whether a key exists for the ContextTranslator's effective
// locale (or its fallback chain).
func (ct *ContextTranslator) HasKey(key string) bool {
	return ct.translator.hasKeyInLocale(ct.effectiveLocale(), key)
}

// SetLocale delegates to the underlying Translator's SetLocale method.
// The context is passed to the logger for trace correlation.
//
// NOTE: SetLocale mutates shared Translator state; it is NOT scoped to this
// ContextTranslator's locale override. For per-request locale scoping, use
// Translator.WithLocaleContext instead.
func (ct *ContextTranslator) SetLocale(locale string) {
	logger := ct.contextLogger()
	if logger.Enabled(core.LevelDebug) {
		logger.Debug("setting locale with context", "locale", locale)
	}
	ct.translator.SetLocale(locale)
}

// GetLocale returns the ContextTranslator's effective locale: the per-instance
// override if set (via WithLocaleContext), otherwise the underlying
// Translator's current locale.
func (ct *ContextTranslator) GetLocale() string {
	return ct.effectiveLocale()
}
