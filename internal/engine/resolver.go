// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

package engine

import (
	"fmt"
	"strings"
	"sync"

	"github.com/0verkilll/i18n/internal/core"
)

// =============================================================================
// Fallback chainer
// =============================================================================

// Compile-time assertion that DefaultFallbackChainer implements FallbackChainer.
var _ core.FallbackChainer = (*DefaultFallbackChainer)(nil)

// baseLocales maps language codes to their primary regional variant.
// Hoisted to package level to avoid allocating a new map on every GetChain call.
var baseLocales = map[string]string{
	"es": "es-ES", // Spanish -> Spain
	"pt": "pt-PT", // Portuguese -> Portugal
	"zh": "zh-CN", // Chinese -> China (Simplified)
	"fr": "fr-FR", // French -> France
	"de": "de-DE", // German -> Germany
	"ar": "ar-SA", // Arabic -> Saudi Arabia
	"it": "it-IT", // Italian -> Italy
	"nl": "nl-NL", // Dutch -> Netherlands
	"pl": "pl-PL", // Polish -> Poland
	"tr": "tr-TR", // Turkish -> Turkey
	"sv": "sv-SE", // Swedish -> Sweden
	"da": "da-DK", // Danish -> Denmark
	"fi": "fi-FI", // Finnish -> Finland
	"no": "no-NO", // Norwegian -> Norway
	"cs": "cs-CZ", // Czech -> Czech Republic
	"el": "el-GR", // Greek -> Greece
	"he": "he-IL", // Hebrew -> Israel
	"hi": "hi-IN", // Hindi -> India
	"th": "th-TH", // Thai -> Thailand
	"vi": "vi-VN", // Vietnamese -> Vietnam
	"id": "id-ID", // Indonesian -> Indonesia
	"ms": "ms-MY", // Malay -> Malaysia
	"bn": "bn-BD", // Bengali -> Bangladesh
	"uk": "uk-UA", // Ukrainian -> Ukraine
	"ro": "ro-RO", // Romanian -> Romania
	"hu": "hu-HU", // Hungarian -> Hungary
	"ja": "ja-JP", // Japanese -> Japan
	"ko": "ko-KR", // Korean -> South Korea
	"ru": "ru-RU", // Russian -> Russia
}

// DefaultFallbackChainer generates locale fallback chains using a built-in
// mapping of language codes to their primary regional variant. It always
// includes en-US as the final fallback when the input is not already en-US.
// Computed chains are cached internally so repeated calls for the same locale
// return a pre-built slice without allocation.
type DefaultFallbackChainer struct {
	chainCache map[string][]string
	mu         sync.RWMutex
}

// NewDefaultFallbackChainer creates a new DefaultFallbackChainer with built-in
// language-to-region mappings for 28 languages.
func NewDefaultFallbackChainer() *DefaultFallbackChainer {
	return &DefaultFallbackChainer{
		chainCache: make(map[string][]string),
	}
}

// GetChain returns the fallback chain for a given locale.
// The returned slice must not be modified by the caller.
// Examples:
//   - "es-MX" returns ["es-MX", "es-ES", "en-US"]
//   - "pt-BR" returns ["pt-BR", "pt-PT", "en-US"]
//   - "en-GB" returns ["en-GB", "en-US"]
//   - "en-US" returns ["en-US"]
func (c *DefaultFallbackChainer) GetChain(locale string) []string {
	// Fast path: check cached chain under read lock.
	c.mu.RLock()
	if chain, ok := c.chainCache[locale]; ok {
		c.mu.RUnlock()
		return chain
	}
	c.mu.RUnlock()

	// Slow path: compute the chain and cache it.
	chain := c.buildChain(locale)

	c.mu.Lock()
	c.chainCache[locale] = chain
	c.mu.Unlock()

	return chain
}

// buildChain computes the fallback chain for a locale without caching.
func (c *DefaultFallbackChainer) buildChain(locale string) []string {
	chain := []string{locale}

	// Extract language and region
	parts := strings.Split(locale, "-")
	if len(parts) != 2 {
		// Language-only code or invalid format
		// Just add English fallback if not already English
		if !strings.HasPrefix(strings.ToLower(locale), "en") {
			chain = append(chain, "en-US")
		}
		return chain
	}

	lang := strings.ToLower(parts[0])
	region := strings.ToUpper(parts[1])
	currentLocale := lang + "-" + region

	// If already en-US, no fallback needed
	if currentLocale == "en-US" {
		return []string{"en-US"}
	}

	// If English variant (en-GB, en-AU, etc.), fall back directly to en-US
	if lang == "en" {
		return []string{currentLocale, "en-US"}
	}

	// Get base locale for this language
	baseLocale := getBaseLocale(lang)

	// If current locale is not the base locale, add base locale to chain
	if baseLocale != "" && currentLocale != baseLocale {
		chain = append(chain, baseLocale)
	}

	// Always add en-US as final fallback (if not already in chain)
	if !containsString(chain, "en-US") {
		chain = append(chain, "en-US")
	}

	return chain
}

// getBaseLocale returns the primary/base locale for a given language code.
func getBaseLocale(lang string) string {
	return baseLocales[lang]
}

// containsString checks if a string slice contains a specific string.
func containsString(slice []string, str string) bool {
	for _, item := range slice {
		if item == str {
			return true
		}
	}
	return false
}

// =============================================================================
// Key resolver
// =============================================================================

// Compile-time assertion that DefaultKeyResolver implements KeyResolver.
var _ core.KeyResolver = (*DefaultKeyResolver)(nil)

// keyPartsCache caches the result of strings.Split(key, ".") for translation
// keys that have already been resolved at least once. Since every translation
// call does a split on a (typically small) set of repeating keys, caching
// the part slice eliminates an allocation + scan on the hot path.
//
// The cache is package-scoped and shared across all DefaultKeyResolver
// instances: (a) there's no registry invalidation event that would make a
// past split incorrect (key "a.b.c" always splits to ["a","b","c"]);
// (b) keys are validated to MaxKeyLength (256) with depth <= MaxKeyDepth
// (10), so cache entries are small and bounded in realistic workloads.
//
// The cache uses sync.Map because the common access pattern is "many reads,
// rare first-time writes" — sync.Map's read-mostly optimization fits
// perfectly.
var keyPartsCache sync.Map // map[string][]string

// splitKeyCached returns strings.Split(key, ".") with a fast-path sync.Map
// cache. The returned slice is shared across callers and MUST NOT be
// mutated by callers.
func splitKeyCached(key string) []string {
	if v, ok := keyPartsCache.Load(key); ok {
		if parts, ok := v.([]string); ok {
			return parts
		}
	}
	parts := strings.Split(key, ".")
	// LoadOrStore to collapse concurrent cache misses for the same key
	// onto a single stored slice.
	actual, _ := keyPartsCache.LoadOrStore(key, parts)
	if stored, ok := actual.([]string); ok {
		return stored
	}
	return parts
}

// DefaultKeyResolver resolves translation keys using dot notation to navigate
// nested maps. It enforces MaxKeyLength and MaxKeyDepth validation via
// ValidateKey before performing the lookup.
type DefaultKeyResolver struct{}

// NewDefaultKeyResolver creates a new DefaultKeyResolver that resolves keys
// using dot-separated notation (e.g., "user.profile.title").
func NewDefaultKeyResolver() *DefaultKeyResolver {
	return &DefaultKeyResolver{}
}

// Resolve retrieves a translation string for the given key from the translations map.
// Supports dot notation for nested keys (e.g., "error.validation.required").
// Returns the translation string, or an error if the key is not found or invalid.
func (r *DefaultKeyResolver) Resolve(translations map[string]interface{}, key string) (string, error) {
	if err := core.ValidateKey(key); err != nil {
		return "", err
	}

	if translations == nil {
		return "", core.NewErrKeyNotFound(key)
	}

	parts := splitKeyCached(key)

	current, err := traverseNested(translations, parts)
	if err != nil {
		return "", core.NewErrKeyNotFound(key)
	}

	finalMap, ok := current.(map[string]interface{})
	if !ok {
		return "", core.NewErrKeyNotFound(key)
	}

	finalValue, exists := finalMap[parts[len(parts)-1]]
	if !exists {
		return "", core.NewErrKeyNotFound(key)
	}

	return convertToString(finalValue, key)
}

// traverseNested walks through nested maps following the key parts,
// returning the parent container of the final key segment.
func traverseNested(translations map[string]interface{}, parts []string) (interface{}, error) {
	var current interface{} = translations
	for i := 0; i < len(parts)-1; i++ {
		currentMap, ok := current.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("not a map at part %d", i)
		}

		value, exists := currentMap[parts[i]]
		if !exists {
			return nil, fmt.Errorf("key not found at part %d", i)
		}

		current = value
	}
	return current, nil
}

// convertToString converts various types to strings.
func convertToString(value interface{}, key string) (string, error) {
	switch v := value.(type) {
	case string:
		return v, nil
	case int:
		return fmt.Sprintf("%d", v), nil
	case int64:
		return fmt.Sprintf("%d", v), nil
	case float64:
		return formatFloat64(v), nil
	case bool:
		return fmt.Sprintf("%t", v), nil
	case nil:
		return "", nil
	default:
		return "", core.NewErrInvalidKey(key, fmt.Errorf("unsupported value type: %T", v))
	}
}
