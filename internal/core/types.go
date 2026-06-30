// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

package core

// =============================================================================
// Plural categories
// =============================================================================

// PluralCategory represents a CLDR plural category.
// Values are lowercase strings that double as key suffixes in translation files.
type PluralCategory string

// CLDR plural categories used by plural rules to select the correct translation form.
const (
	Zero  PluralCategory = "zero"
	One   PluralCategory = "one"
	Two   PluralCategory = "two"
	Few   PluralCategory = "few"
	Many  PluralCategory = "many"
	Other PluralCategory = "other"
)

// GenderCategory represents a grammatical gender for message selection.
// The caller specifies the gender value; no linguistic rule engine is needed.
type GenderCategory string

// Grammatical gender categories for gender-aware message selection.
const (
	Masculine   GenderCategory = "masculine"
	Feminine    GenderCategory = "feminine"
	Neuter      GenderCategory = "neuter"
	GenderOther GenderCategory = "other"
)

// =============================================================================
// Log levels
// =============================================================================

// LogLevel represents logging severity levels.
type LogLevel int

// Log levels matching github.com/0verkilll/logger.Level values.
const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
)

// =============================================================================
// Validation and parser constants
// =============================================================================

// MaxLocaleLength is the maximum allowed length for locale strings.
const MaxLocaleLength = 10

// MaxKeyLength is the maximum allowed length for translation keys.
const MaxKeyLength = 256

// MaxKeyDepth is the maximum nesting depth for translation keys.
const MaxKeyDepth = 10

// MaxOutputLength is the maximum allowed length for sanitized output (10 KB).
const MaxOutputLength = 10240

// MaxJSONSize is the maximum allowed size for JSON input in bytes (10 MB).
const MaxJSONSize = 10 * 1024 * 1024

// MaxJSONDepth is the maximum allowed nesting depth for JSON documents.
const MaxJSONDepth = 50

// MaxPrefixLength is the maximum allowed length for namespace prefixes.
const MaxPrefixLength = 64

// MaxKeyCount is the maximum number of total keys (including nested) allowed
// in a single parsed translation file. This prevents denial-of-service from
// translation files with an excessive number of entries.
const MaxKeyCount = 10000
