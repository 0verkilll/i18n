// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

package core

import "context"

// TranslationLoader abstracts loading translation data from various sources.
// Implementations must ensure secure file access and prevent path traversal attacks.
type TranslationLoader interface {
	// Load retrieves translation data for the specified locale.
	// The locale parameter must be a valid BCP 47 code.
	// Returns the raw translation data or an error if loading fails.
	Load(locale string) ([]byte, error)
}

// TranslationParser abstracts parsing translation data formats.
// Implementations must validate input and prevent malicious data from causing issues.
type TranslationParser interface {
	// Parse converts raw translation data into a structured format.
	// The data should be validated for security issues (nesting depth, size, etc.).
	// Returns a map of translation keys to values (which may be nested) or an error.
	Parse(data []byte) (map[string]interface{}, error)
}

// KeyResolver handles nested key resolution using dot notation.
// Implementations must enforce maximum key depth and length limits.
type KeyResolver interface {
	// Resolve looks up a translation key in the provided translations map.
	// Keys use dot notation for nested values (e.g., "error.validation.required").
	// Returns the translated string or an error if the key is invalid or not found.
	Resolve(translations map[string]interface{}, key string) (string, error)
}

// Detector retrieves the system locale from environment variables.
type Detector interface {
	// Detect retrieves the system locale from environment variables.
	// Returns a locale string (which may need normalization).
	Detect() string
}

// Normalizer converts locale strings to BCP 47 format.
type Normalizer interface {
	// Normalize converts locale strings to BCP 47 format.
	// Handles various input formats (en_US, en_US.UTF-8, etc.) and
	// returns a standardized BCP 47 code (e.g., "en-US").
	Normalize(locale string) string
}

// LocaleDetector detects and normalizes system locale information.
// It composes the Detector and Normalizer interfaces.
type LocaleDetector interface {
	Detector
	Normalizer
}

// FallbackChainer generates fallback locale chains for graceful degradation.
type FallbackChainer interface {
	// GetChain returns the fallback chain for a given locale.
	// For example, "es-MX" might return ["es-MX", "es-ES", "en-US"].
	// The chain should always include a default fallback locale.
	GetChain(locale string) []string
}

// Cacher abstracts resolved-translation caching for the Translator.
// Implementations must be safe for concurrent use by multiple goroutines.
// This interface is independent of TranslatorProvider and all other interfaces.
type Cacher interface {
	// Get retrieves a cached translation by its cache key.
	// Returns the cached value and true on a hit, or an empty string and false on a miss.
	Get(key string) (string, bool)

	// Set stores a resolved translation under the given cache key.
	Set(key string, value string)

	// Invalidate discards all cached entries.
	Invalidate()
}

// TranslationLookup provides single-key translation lookup.
type TranslationLookup interface {
	// Translate looks up a translation key in the current locale.
	// If the key is not found, it tries the fallback chain.
	// Returns the key itself if not found in any locale.
	Translate(key string) string
}

// FormattedTranslator provides formatted translation lookup with arguments.
type FormattedTranslator interface {
	// TranslateWithArgs looks up a translation key and formats it with arguments.
	// Uses fmt.Sprintf formatting. If the key is not found, returns the key itself.
	TranslateWithArgs(key string, args ...interface{}) string
}

// KeyChecker checks whether a translation key exists.
type KeyChecker interface {
	// HasKey checks if a translation key exists in the current locale or fallback chain.
	HasKey(key string) bool
}

// LocaleSetter changes the active locale for translation lookups.
type LocaleSetter interface {
	// SetLocale changes the current locale for translation lookups.
	// The locale will be normalized before being set.
	SetLocale(locale string)
}

// LocaleGetter retrieves the active locale.
type LocaleGetter interface {
	// GetLocale returns the current locale being used for translations.
	GetLocale() string
}

// PluralTranslator provides count-dependent translation lookup.
type PluralTranslator interface {
	// TranslatePlural resolves the plural category for the current locale and count,
	// looks up key.<category>, and falls back to key.other.
	TranslatePlural(key string, count interface{}) string
}

// GenderTranslator provides gender-aware translation lookup.
type GenderTranslator interface {
	// TranslateGender looks up key.<gender>, falling back to key.other.
	TranslateGender(key string, gender GenderCategory) string
}

// TranslatorProvider defines the interface for translation services.
// This interface allows other packages to accept translation capabilities
// without creating a hard dependency on the full i18n package.
//
// TranslatorProvider composes seven single-responsibility sub-interfaces:
// TranslationLookup, FormattedTranslator, KeyChecker, LocaleSetter, LocaleGetter,
// PluralTranslator, and GenderTranslator.
// Consumers that need only a subset of these capabilities should accept the
// narrower sub-interface instead.
//
// # Integration Pattern
//
// Other Go packages can support optional i18n by defining a local interface
// matching this signature. Application developers then pass their i18n.Translator
// instance to the package's SetTranslator() function.
//
// Example usage in a package:
//
//	// In mypackage/i18n.go
//	type TranslatorProvider interface {
//	    Translate(key string) string
//	    TranslateWithArgs(key string, args ...interface{}) string
//	    TranslatePlural(key string, count interface{}) string
//	    TranslateGender(key string, gender i18n.GenderCategory) string
//	    HasKey(key string) bool
//	    SetLocale(locale string)
//	    GetLocale() string
//	}
//
//	var globalTranslator TranslatorProvider
//
//	func SetTranslator(t TranslatorProvider) {
//	    globalTranslator = t
//	}
//
// Application developers can then use it:
//
//	translator, _ := i18n.New(
//	    i18n.WithFileSystemLoader("locales"),
//	    i18n.WithDefaultLocale("en-US"),
//	)
//	mypackage.SetTranslator(translator)
//
// # Benefits
//
//   - No forced dependencies: Packages work with or without i18n
//   - Shared translator: One translator instance serves all packages
//   - Centralized control: Application manages all translations
//   - Language switching: Changes affect all integrated packages
//
// The Translator type in this package implements TranslatorProvider automatically.
type TranslatorProvider interface {
	TranslationLookup
	FormattedTranslator
	PluralTranslator
	GenderTranslator
	KeyChecker
	LocaleSetter
	LocaleGetter
}

// PluralResolver determines the plural category for a given locale and count.
type PluralResolver interface {
	// Resolve returns the plural category for the given locale and count.
	// The count parameter accepts int, int64, float64, and string (numeric string).
	// Returns Other for unrecognized types or unknown locales.
	Resolve(locale string, count interface{}) PluralCategory
}

// Logger defines the logging interface accepted by the i18n package.
// This interface is compatible with github.com/0verkilll/logger.Logger
// and allows any implementation that satisfies these methods.
//
// The interface follows SOLID principles:
//   - Single Responsibility: focused on logging operations
//   - Open/Closed: extensible through implementation
//   - Liskov Substitution: any implementation is substitutable
//   - Interface Segregation: LeveledLogger provides minimal subset
//   - Dependency Inversion: depend on this abstraction, not concrete loggers
type Logger interface {
	// LeveledLogger provides the core logging methods.
	LeveledLogger

	// WithFields returns a new Logger with additional structured fields.
	// Fields are provided as key-value pairs: ("user_id", 123, "action", "login").
	WithFields(fields ...any) Logger

	// WithContext returns a new Logger with the given context.
	WithContext(ctx context.Context) Logger

	// WithLevel returns a new Logger that only logs at or above the given level.
	WithLevel(level LogLevel) Logger

	// Enabled returns true if logging at the given level would produce output.
	Enabled(level LogLevel) bool
}

// LeveledLogger defines minimal leveled logging without fields or context.
// Use this interface when you only need basic logging capabilities.
type LeveledLogger interface {
	// Debug logs a debug-level message with optional format arguments.
	Debug(msg string, args ...any)

	// Info logs an info-level message with optional format arguments.
	Info(msg string, args ...any)

	// Warn logs a warn-level message with optional format arguments.
	Warn(msg string, args ...any)

	// Error logs an error-level message with optional format arguments.
	Error(msg string, args ...any)

	// Fatal logs a fatal-level message with optional format arguments.
	Fatal(msg string, args ...any)
}

// EnvProvider abstracts environment variable access for testing.
type EnvProvider interface {
	// Getenv returns the value of the environment variable named by the key.
	Getenv(key string) string
}
