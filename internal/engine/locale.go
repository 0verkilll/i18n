// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

package engine

import (
	"sort"
	"strconv"
	"strings"

	"github.com/0verkilll/i18n/internal/core"
)

// =============================================================================
// Default locale detector
// =============================================================================

// Compile-time interface assertions that DefaultLocaleDetector implements
// core.LocaleDetector and its composed sub-interfaces core.Detector and core.Normalizer.
var (
	_ core.LocaleDetector = (*DefaultLocaleDetector)(nil)
	_ core.Detector       = (*DefaultLocaleDetector)(nil)
	_ core.Normalizer     = (*DefaultLocaleDetector)(nil)
)

// DefaultLocaleDetector is the default implementation of core.LocaleDetector. It
// reads locale information from environment variables using the provided
// core.EnvProvider, falling back to "en-US" when no locale is configured.
type DefaultLocaleDetector struct {
	env core.EnvProvider
}

// NewDefaultLocaleDetector creates a new DefaultLocaleDetector. If env is nil,
// the platform default core.EnvProvider is used (OSEnvProvider on standard Go,
// WASMEnvProvider on js/wasm).
func NewDefaultLocaleDetector(env core.EnvProvider) *DefaultLocaleDetector {
	if env == nil {
		env = defaultEnvProvider()
	}
	return &DefaultLocaleDetector{
		env: env,
	}
}

// Detect retrieves the system locale from environment variables.
// Priority order: LC_ALL > LANG > LC_MESSAGES > default (en-US).
func (d *DefaultLocaleDetector) Detect() string {
	// Try LC_ALL first (highest priority - overrides everything)
	if lcAll := d.env.Getenv("LC_ALL"); lcAll != "" {
		return lcAll
	}

	// Try LANG next (most common)
	if lang := d.env.Getenv("LANG"); lang != "" {
		return lang
	}

	// Try LC_MESSAGES as final fallback
	if lcMessages := d.env.Getenv("LC_MESSAGES"); lcMessages != "" {
		return lcMessages
	}

	// Default to US English
	return "en-US"
}

// Normalize converts locale strings to BCP 47 format.
// Handles: en_US.UTF-8 -> en-US, POSIX -> en-US, en -> en-US.
func (d *DefaultLocaleDetector) Normalize(locale string) string {
	return NormalizeLocale(locale)
}

// languageDefaultRegions maps language-only codes to their primary regional variant.
// Hoisted to package level to avoid allocating a new map on every NormalizeLocale call.
var languageDefaultRegions = map[string]string{
	"en": "en-US", // English -> United States
	"es": "es-ES", // Spanish -> Spain
	"pt": "pt-PT", // Portuguese -> Portugal
	"zh": "zh-CN", // Chinese -> China (Simplified)
	"fr": "fr-FR", // French -> France
	"de": "de-DE", // German -> Germany
	"ja": "ja-JP", // Japanese -> Japan
	"ko": "ko-KR", // Korean -> South Korea
	"ru": "ru-RU", // Russian -> Russia
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
}

// NormalizeLocale converts a locale string to BCP 47 format.
// It handles encoding suffix removal (e.g., .UTF-8), underscore-to-hyphen
// conversion, case normalization, POSIX/C locale mapping, and language-only
// codes mapped to their primary regional variant.
//
// All LocaleDetector implementations delegate their Normalize method to this
// shared function to avoid duplicating normalization logic.
func NormalizeLocale(locale string) string {
	if locale == "" {
		return ""
	}

	// Handle POSIX/C locale
	localeLower := strings.ToLower(locale)
	if localeLower == "posix" || localeLower == "c" {
		return "en-US"
	}

	// Remove encoding suffix (e.g., .UTF-8, .ISO-8859-1)
	if idx := strings.IndexByte(locale, '.'); idx != -1 {
		locale = locale[:idx]
	}

	// Convert underscores to hyphens for BCP 47 format
	locale = strings.ReplaceAll(locale, "_", "-")

	// Normalize to proper BCP 47 casing: language lowercase, region uppercase
	parts := strings.Split(locale, "-")
	if len(parts) == 2 {
		return strings.ToLower(parts[0]) + "-" + strings.ToUpper(parts[1])
	}

	// Handle language-only codes (map to primary region)
	if len(parts) == 1 {
		langOnly := strings.ToLower(parts[0])
		if mappedLocale, ok := getLanguageDefaultRegion(langOnly); ok {
			return mappedLocale
		}
		return locale
	}

	return locale
}

// getLanguageDefaultRegion maps language-only codes to their primary regional variant.
func getLanguageDefaultRegion(lang string) (string, bool) {
	mapped, ok := languageDefaultRegions[lang]
	return mapped, ok
}

// =============================================================================
// Accept-Language detector
// =============================================================================

// Compile-time interface assertion.
var _ core.LocaleDetector = (*AcceptLanguageDetector)(nil)

// maxAcceptLanguageLength is the maximum allowed length for an Accept-Language
// header value. Headers exceeding this limit are rejected to prevent abuse.
const maxAcceptLanguageLength = 4096

// AcceptLanguageDetector is a core.LocaleDetector that parses an HTTP
// Accept-Language header per RFC 7231 Section 5.3.5 and returns the
// highest-priority language tag.
type AcceptLanguageDetector struct {
	header string
}

// NewAcceptLanguageDetector creates a new AcceptLanguageDetector that parses
// the given raw Accept-Language header value. The header is parsed lazily
// when Detect is called.
//
// Example usage in an HTTP handler:
//
//	func handler(w http.ResponseWriter, r *http.Request) {
//	    detector := engine.NewAcceptLanguageDetector(r.Header.Get("Accept-Language"))
//	    translator, _ := engine.New(
//	        engine.WithFileSystemLoader("locales"),
//	        engine.WithLocaleDetector(detector),
//	    )
//	    // translator will use the browser's preferred language
//	}
func NewAcceptLanguageDetector(header string) *AcceptLanguageDetector {
	return &AcceptLanguageDetector{header: header}
}

// langEntry holds a parsed language tag and its associated quality value.
type langEntry struct {
	tag     string
	quality float64
	order   int
}

// Detect parses the Accept-Language header and returns the highest-priority
// language tag after normalization. Returns "" if the header is empty,
// exceeds the length limit, contains only wildcards, or is unparseable.
func (d *AcceptLanguageDetector) Detect() string {
	if d.header == "" || len(d.header) > maxAcceptLanguageLength {
		return ""
	}

	entries := parseAcceptLanguage(d.header)
	if len(entries) == 0 {
		return ""
	}

	// Stable sort by quality descending; equal quality preserves original order
	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].quality > entries[j].quality
	})

	return NormalizeLocale(entries[0].tag)
}

// Normalize converts a locale string to BCP 47 format using the shared
// normalization logic.
func (d *AcceptLanguageDetector) Normalize(locale string) string {
	return NormalizeLocale(locale)
}

// parseAcceptLanguage splits the header on commas and extracts language
// tags with their quality values. Wildcards and empty tags are skipped.
func parseAcceptLanguage(header string) []langEntry {
	parts := strings.Split(header, ",")
	entries := make([]langEntry, 0, len(parts))

	for i, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		tag, params, hasParams := strings.Cut(part, ";")
		tag = strings.TrimSpace(tag)

		// Skip empty tags and wildcards
		if tag == "" || tag == "*" {
			continue
		}

		quality := 1.0
		if hasParams {
			quality = parseQuality(params)
		}

		entries = append(entries, langEntry{
			tag:     tag,
			quality: quality,
			order:   i,
		})
	}

	return entries
}

// parseQuality extracts the quality value from the parameters portion of an
// Accept-Language entry. Returns 1.0 if the q parameter is absent or malformed.
func parseQuality(params string) float64 {
	params = strings.TrimSpace(params)

	_, qVal, found := strings.Cut(params, "q=")
	if !found {
		return 1.0
	}

	qVal = strings.TrimSpace(qVal)

	// Handle additional parameters after the quality value
	if idx := strings.IndexByte(qVal, ';'); idx != -1 {
		qVal = strings.TrimSpace(qVal[:idx])
	}

	q, err := strconv.ParseFloat(qVal, 64)
	if err != nil {
		return 1.0
	}

	// Clamp to valid range
	if q < 0 {
		q = 0
	}
	if q > 1 {
		q = 1
	}

	return q
}

// =============================================================================
// Chain detector
// =============================================================================

// Compile-time interface assertion.
var _ core.LocaleDetector = (*ChainDetector)(nil)

// ChainDetector is a core.LocaleDetector that composes multiple detectors in
// priority order. It iterates through its detectors and returns the first
// non-empty result. If all detectors return "", it falls back to "en-US".
type ChainDetector struct {
	detectors []core.LocaleDetector
}

// NewChainDetector creates a new ChainDetector that iterates the given
// detectors in order, returning the first non-empty locale. If no detectors
// are provided or all return "", Detect falls back to "en-US".
//
// Example usage composing multiple detection strategies:
//
//	chain := engine.NewChainDetector(
//	    engine.NewStaticDetector(overrideFromURL),       // highest priority
//	    engine.NewAcceptLanguageDetector(acceptHeader),   // browser preference
//	    engine.NewDefaultLocaleDetector(nil),             // env var fallback
//	)
//	translator, _ := engine.New(
//	    engine.WithFileSystemLoader("locales"),
//	    engine.WithLocaleDetector(chain),
//	)
func NewChainDetector(detectors ...core.LocaleDetector) *ChainDetector {
	return &ChainDetector{detectors: detectors}
}

// Detect iterates detectors in order and returns the first non-empty result.
// If all detectors return "" or the chain is empty, returns "en-US".
func (c *ChainDetector) Detect() string {
	for _, d := range c.detectors {
		if result := d.Detect(); result != "" {
			return result
		}
	}
	return "en-US"
}

// Normalize converts a locale string to BCP 47 format. If the chain has at
// least one detector, it delegates to the first detector's Normalize method.
// Otherwise it calls NormalizeLocale directly.
func (c *ChainDetector) Normalize(locale string) string {
	if len(c.detectors) > 0 {
		return c.detectors[0].Normalize(locale)
	}
	return NormalizeLocale(locale)
}

// =============================================================================
// Static detector
// =============================================================================

// Compile-time interface assertion.
var _ core.LocaleDetector = (*StaticDetector)(nil)

// StaticDetector is a core.LocaleDetector that always returns a fixed locale.
// It is intended for testing and explicit locale overrides. Placing it first
// in a ChainDetector forces a specific locale regardless of other detectors.
type StaticDetector struct {
	locale string
}

// NewStaticDetector creates a new StaticDetector that always returns the
// given locale from Detect. The locale is returned exactly as provided,
// without normalization.
//
// Example usage:
//
//	translator, _ := engine.New(
//	    engine.WithFileSystemLoader("locales"),
//	    engine.WithLocaleDetector(engine.NewStaticDetector("fr-FR")),
//	)
func NewStaticDetector(locale string) *StaticDetector {
	return &StaticDetector{locale: locale}
}

// Detect returns the configured locale string exactly as provided to the
// constructor.
func (s *StaticDetector) Detect() string {
	return s.locale
}

// Normalize converts a locale string to BCP 47 format using the shared
// normalization logic.
func (s *StaticDetector) Normalize(locale string) string {
	return NormalizeLocale(locale)
}
