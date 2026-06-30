// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

// Package i18n provides a security-hardened internationalization library for Go
// with locale fallback chains, embedded filesystem support, build-tag locale
// selection for WASM/TinyGo, and zero external dependencies.
//
// # Quick Start
//
//	translator, err := i18n.New(
//	    i18n.WithFileSystemLoader("locales"),
//	    i18n.WithDefaultLocale("en-US"),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	message := translator.Translate("greeting")
//	welcome := translator.TranslateWithArgs("welcome", "Alice")
//	translator.SetLocale("es-ES")
//
// # Key Types
//
//   - [Translator]: Main translation engine with fallback chain, plural, gender,
//     and ICU MessageFormat support. Thread-safe for concurrent use.
//   - [TranslatorProvider]: Interface for translation services; implemented by
//     [Translator] and [PackageTranslator].
//   - [MapCache]: LRU-aware translation cache implementing [Cacher].
//   - [Namespace]: Immutable key prefix helper for scoping translations to a
//     package name. Methods: T, TF, TD, Has, Key.
//   - [PackageTranslator]: Per-package translator with hardcoded defaults and
//     thread-safe translator swapping. Replaces ~100-200 lines of boilerplate.
//
// # Interfaces
//
//   - [TranslationLookup]: Simple key-to-string translation.
//   - [FormattedTranslator]: Translation with format arguments.
//   - [KeyChecker]: Checks whether a key exists.
//   - [LocaleSetter] / [LocaleGetter]: Locale mutation and query.
//   - [PluralTranslator]: CLDR plural-aware translation (zero/one/two/few/many/other).
//   - [GenderTranslator]: Gender-aware translation (masculine/feminine/other).
//   - [TranslationLoader]: Loads translation data ([FileSystemLoader], [EmbedFSLoader], [RegistryLoader]).
//   - [TranslationParser]: Parses translation data ([JSONParser]; extensible via [Registry]).
//   - [KeyResolver]: Resolves nested dot-notation keys.
//   - [LocaleDetector]: Detects locale ([DefaultLocaleDetector], [AcceptLanguageDetector],
//     [BrowserDetector], [StaticDetector], [ChainDetector]).
//   - [FallbackChainer]: Generates locale fallback chains.
//   - [PluralResolver]: Determines CLDR plural categories for 30+ languages.
//   - [Cacher]: Caches resolved translations.
//   - [Logger] / [LeveledLogger]: Structured logging with levels and context.
//
// # Build-Tag Locale Selection
//
// For WASM and TinyGo builds, compile only the locales you need:
//
//	go build -tags locale_en_us
//	go build -tags "locale_en_us,locale_es_es"
//	go build -tags locale_all
//
// Use [RegistryLoader] to serve translations from the build-tag registry:
//
//	translator, err := i18n.New(
//	    i18n.WithRegistryLoader(),
//	    i18n.WithDefaultLocale("en-US"),
//	)
//
// When no locale tags are specified, the registry is empty and [RegistryLoader]
// returns an error for any locale. The locale data files in internal/localedata/
// are example translations for testing; production applications should supply their own
// translation files.
//
// # Plural and Gender
//
//	translator.TranslatePlural("items", 1)                      // "1 item"
//	translator.TranslatePlural("items", 5)                      // "5 items"
//	translator.TranslateGender("greeting", i18n.Feminine)       // "greeting.feminine"
//	translator.TranslateWithMessage("msg", map[string]interface{}{"count": 3})
//
// # Security
//
// All input is validated and all output is sanitized automatically:
//
//   - Path traversal prevention in locale names and file paths
//   - Control character and ANSI escape sequence filtering
//   - BiDi override attack prevention (U+202A-U+202E)
//   - Format string validation (blocks %n, validates argument counts)
//   - Input size limits: [MaxLocaleLength] (10), [MaxKeyLength] (256),
//     [MaxKeyDepth] (10), [MaxOutputLength] (10240)
//   - JSON size limit (10 MB) and nesting depth limit (50 levels)
//
// # Error Types
//
// Custom error types supporting errors.Is and errors.As:
//
//   - [ErrInvalidLocale]: Invalid locale format or path traversal attempt
//   - [ErrInvalidKey]: Invalid translation key format
//   - [ErrKeyNotFound]: Translation key does not exist
//   - [ErrInvalidFormat]: Invalid file format or parsing error
//   - [ErrPathTraversal]: Path traversal attack detected
//   - [ErrUnknownFormat]: No parser registered for a file extension
//
// # Examples
//
// See example_test.go for complete working examples covering basic translation,
// format strings, locale switching, fallback chains, embedded filesystem,
// build-tag locale selection, [Namespace], [PackageTranslator], plural and
// gender translation, and parser registry extensibility.
package i18n
