// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

package i18n

import (
	"github.com/0verkilll/i18n/internal/core"
	"github.com/0verkilll/i18n/internal/engine"
)

// =============================================================================
// Type aliases — re-export core types so consumers use i18n.X directly
// =============================================================================

// PluralCategory represents CLDR plural categories.
type PluralCategory = core.PluralCategory

// Plural category constants.
const (
	Zero  = core.Zero
	One   = core.One
	Two   = core.Two
	Few   = core.Few
	Many  = core.Many
	Other = core.Other
)

// GenderCategory represents grammatical gender categories.
type GenderCategory = core.GenderCategory

// Gender category constants.
const (
	Masculine   = core.Masculine
	Feminine    = core.Feminine
	Neuter      = core.Neuter
	GenderOther = core.GenderOther
)

// LogLevel represents logging severity levels.
type LogLevel = core.LogLevel

// Log level constants.
const (
	LevelDebug = core.LevelDebug
	LevelInfo  = core.LevelInfo
	LevelWarn  = core.LevelWarn
	LevelError = core.LevelError
	LevelFatal = core.LevelFatal
)

// Validation and security limit constants.
const (
	MaxLocaleLength      = core.MaxLocaleLength
	MaxKeyLength         = core.MaxKeyLength
	MaxKeyDepth          = core.MaxKeyDepth
	MaxOutputLength      = core.MaxOutputLength
	MaxJSONSize          = core.MaxJSONSize
	MaxJSONDepth         = core.MaxJSONDepth
	MaxPrefixLength      = core.MaxPrefixLength
	MaxFormatPrecision   = core.MaxFormatPrecision
	MaxRegisteredLocales = engine.MaxRegisteredLocales
)

// =============================================================================
// Interface aliases — re-export core interfaces
// =============================================================================

// TranslatorProvider defines the complete interface for translation services.
type TranslatorProvider = core.TranslatorProvider

// TranslationLookup provides single-key translation lookup.
type TranslationLookup = core.TranslationLookup

// FormattedTranslator provides formatted translation lookup with arguments.
type FormattedTranslator = core.FormattedTranslator

// KeyChecker checks whether a translation key exists.
type KeyChecker = core.KeyChecker

// LocaleSetter changes the active locale for translation lookups.
type LocaleSetter = core.LocaleSetter

// LocaleGetter retrieves the active locale.
type LocaleGetter = core.LocaleGetter

// PluralTranslator provides count-dependent translation lookup.
type PluralTranslator = core.PluralTranslator

// GenderTranslator provides gender-aware translation lookup.
type GenderTranslator = core.GenderTranslator

// TranslationLoader abstracts loading translation data from various sources.
type TranslationLoader = core.TranslationLoader

// TranslationParser abstracts parsing translation data formats.
type TranslationParser = core.TranslationParser

// KeyResolver handles nested key resolution using dot notation.
type KeyResolver = core.KeyResolver

// FallbackChainer generates fallback locale chains for graceful degradation.
type FallbackChainer = core.FallbackChainer

// Cacher abstracts resolved-translation caching for the Translator.
type Cacher = core.Cacher

// Detector retrieves the system locale from environment variables.
type Detector = core.Detector

// Normalizer converts locale strings to BCP 47 format.
type Normalizer = core.Normalizer

// LocaleDetector detects and normalizes system locale information.
type LocaleDetector = core.LocaleDetector

// PluralResolver resolves plural categories for a given locale and count.
type PluralResolver = core.PluralResolver

// Logger provides structured logging.
type Logger = core.Logger

// LeveledLogger extends Logger with level-checking capabilities.
type LeveledLogger = core.LeveledLogger

// EnvProvider abstracts environment variable access for locale detection.
type EnvProvider = core.EnvProvider

// =============================================================================
// Error type aliases — re-export core error types
// =============================================================================

// ErrInvalidLocale indicates a locale code failed validation.
type ErrInvalidLocale = core.ErrInvalidLocale

// ErrInvalidKey indicates a translation key failed validation.
type ErrInvalidKey = core.ErrInvalidKey

// ErrKeyNotFound indicates a translation key was not found.
type ErrKeyNotFound = core.ErrKeyNotFound

// ErrInvalidFormat indicates a data format error.
type ErrInvalidFormat = core.ErrInvalidFormat

// ErrPathTraversal indicates a path traversal attack was detected.
type ErrPathTraversal = core.ErrPathTraversal

// ErrUnknownFormat indicates an unregistered file format.
type ErrUnknownFormat = core.ErrUnknownFormat

// Error constructors.
var (
	NewErrInvalidLocale = core.NewErrInvalidLocale
	NewErrInvalidKey    = core.NewErrInvalidKey
	NewErrKeyNotFound   = core.NewErrKeyNotFound
	NewErrInvalidFormat = core.NewErrInvalidFormat
	NewErrPathTraversal = core.NewErrPathTraversal
	NewErrUnknownFormat = core.NewErrUnknownFormat
)

// Sentinel errors re-exported from the engine package.
var (
	// ErrRegistryFull is returned when the in-process locale registry already
	// holds MaxRegisteredLocales distinct entries and a new locale is being added.
	ErrRegistryFull = engine.ErrRegistryFull
)

// =============================================================================
// Implementation type aliases — re-export engine types
// =============================================================================

// Translator is the main type for translation operations.
type Translator = engine.Translator

// ContextTranslator wraps a Translator and passes a context.Context to the
// logger for trace correlation and observability integration.
type ContextTranslator = engine.ContextTranslator

// Option is a functional option for configuring the Translator.
type Option = engine.Option

// MapCache is a thread-safe, in-memory translation cache with optional LRU eviction.
type MapCache = engine.MapCache

// DefaultKeyResolver resolves translation keys using dot notation.
type DefaultKeyResolver = engine.DefaultKeyResolver

// DefaultFallbackChainer generates locale fallback chains.
type DefaultFallbackChainer = engine.DefaultFallbackChainer

// DefaultPluralResolver resolves CLDR plural categories.
type DefaultPluralResolver = engine.DefaultPluralResolver

// JSONParser parses JSON translation files.
type JSONParser = engine.JSONParser

// BinaryParser parses compact binary translation files.
type BinaryParser = engine.BinaryParser

// Registry stores registered translation parsers.
type Registry = engine.Registry

// NopLogger is a no-op logger implementation.
type NopLogger = engine.NopLogger

// EmbedFSLoader loads translations from an embedded filesystem.
type EmbedFSLoader = engine.EmbedFSLoader

// RegistryLoader loads translations from the locale data registry.
type RegistryLoader = engine.RegistryLoader

// DefaultLocaleDetector detects system locale from environment variables.
type DefaultLocaleDetector = engine.DefaultLocaleDetector

// AcceptLanguageDetector parses Accept-Language headers.
type AcceptLanguageDetector = engine.AcceptLanguageDetector

// ChainDetector chains multiple locale detectors.
type ChainDetector = engine.ChainDetector

// StaticDetector returns a fixed locale.
type StaticDetector = engine.StaticDetector

// BrowserDetector reads the browser's preferred language via navigator.language
// in js/wasm builds. On non-WASM platforms, Detect returns "".
type BrowserDetector = engine.BrowserDetector

// Namespace automatically prefixes translation keys with a package name.
type Namespace = engine.Namespace

// PackageTranslator provides per-package translation with namespace scoping.
type PackageTranslator = engine.PackageTranslator

// PackageOption is a functional option for configuring a PackageTranslator.
type PackageOption = engine.PackageOption

// LoaderOption configures a loader via the functional options pattern.
type LoaderOption = engine.LoaderOption

// FileSystemLoader loads translations from the local filesystem.
type FileSystemLoader = engine.FileSystemLoader
