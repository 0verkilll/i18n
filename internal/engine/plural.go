// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

package engine

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/0verkilll/i18n/internal/core"
)

// =============================================================================
// Plural operands (CLDR)
// =============================================================================

// pluralOperands holds the CLDR plural rule operands extracted from a numeric value.
// See https://unicode.org/reports/tr35/tr35-numbers.html#Operands
//
//   - n: absolute value of the source number (with fraction)
//   - i: integer digits of n
//   - v: number of visible fraction digits (with trailing zeros)
//   - w: number of visible fraction digits (without trailing zeros)
//   - f: visible fraction digits (with trailing zeros) as integer
//   - t: visible fraction digits (without trailing zeros) as integer
type pluralOperands struct {
	n float64
	i uint64
	v uint64
	w uint64
	f uint64
	t uint64
}

// absInt returns the absolute value of an int as uint64.
func absInt(v int) uint64 {
	if v < 0 {
		return uint64(-v)
	}
	return uint64(v)
}

// absInt64 returns the absolute value of an int64 as uint64.
func absInt64(v int64) uint64 {
	if v < 0 {
		return uint64(-v)
	}
	return uint64(v)
}

// extractOperands extracts CLDR plural operands from a count value.
// Accepts int, int64, float64, and string (numeric string).
// Returns zero-value operands for unrecognized types.
func extractOperands(count interface{}) pluralOperands {
	switch c := count.(type) {
	case int:
		abs := absInt(c)
		return pluralOperands{
			n: float64(abs),
			i: abs,
		}
	case int64:
		abs := absInt64(c)
		return pluralOperands{
			n: float64(abs),
			i: abs,
		}
	case float64:
		return extractFromFloat64(c)
	case string:
		return extractFromString(c)
	default:
		return pluralOperands{}
	}
}

// extractFromFloat64 extracts operands from a float64 value.
// Since float64 doesn't preserve trailing zeros, we format it to determine v/w/f/t.
func extractFromFloat64(val float64) pluralOperands {
	abs := math.Abs(val)
	intPart := uint64(abs)

	s := strconv.FormatFloat(abs, 'f', -1, 64)
	return extractFromNumericString(s, abs, intPart)
}

// extractFromString parses a numeric string and extracts operands.
// The string representation is used directly, preserving trailing zeros.
func extractFromString(s string) pluralOperands {
	s = strings.TrimSpace(s)
	if s == "" {
		return pluralOperands{}
	}

	if s[0] == '-' {
		s = s[1:]
	}

	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return pluralOperands{}
	}

	abs := math.Abs(val)
	intPart := uint64(abs)

	return extractFromNumericString(s, abs, intPart)
}

// extractFromNumericString extracts operands from a string representation of a number.
// This preserves trailing zeros which are significant for CLDR plural rules.
func extractFromNumericString(s string, abs float64, intPart uint64) pluralOperands {
	ops := pluralOperands{
		n: abs,
		i: intPart,
	}

	dotIdx := strings.IndexByte(s, '.')
	if dotIdx < 0 {
		return ops
	}

	fracStr := s[dotIdx+1:]
	if fracStr == "" {
		return ops
	}

	ops.v = uint64(len(fracStr))

	f, err := strconv.ParseUint(fracStr, 10, 64)
	if err != nil {
		return ops
	}
	ops.f = f

	trimmed := strings.TrimRight(fracStr, "0")
	if trimmed == "" {
		ops.t = 0
		ops.w = 0
	} else {
		ops.w = uint64(len(trimmed))
		t, err := strconv.ParseUint(trimmed, 10, 64)
		if err != nil {
			return ops
		}
		ops.t = t
	}

	return ops
}

// =============================================================================
// Plural rules (CLDR)
// =============================================================================

// Compile-time assertion that DefaultPluralResolver implements core.PluralResolver.
var _ core.PluralResolver = (*DefaultPluralResolver)(nil)

// DefaultPluralResolver resolves plural categories using built-in CLDR rules.
type DefaultPluralResolver struct{}

// NewDefaultPluralResolver creates a new DefaultPluralResolver.
func NewDefaultPluralResolver() *DefaultPluralResolver {
	return &DefaultPluralResolver{}
}

// Resolve determines the plural category for the given locale and count.
// It extracts the language subtag from the locale and looks up the CLDR rule.
// Returns Other for unknown locales or unrecognized count types.
func (r *DefaultPluralResolver) Resolve(locale string, count interface{}) core.PluralCategory {
	lang := extractLanguage(locale)
	ops := extractOperands(count)

	ruleFn, ok := pluralRules[lang]
	if !ok {
		return core.Other
	}

	return ruleFn(ops)
}

// extractLanguage returns the language subtag from a locale string.
// For "en-US" it returns "en", for "zh" it returns "zh".
func extractLanguage(locale string) string {
	idx := strings.IndexByte(locale, '-')
	if idx < 0 {
		return strings.ToLower(locale)
	}
	return strings.ToLower(locale[:idx])
}

// pluralRules maps language codes to their CLDR plural rule functions.
// Covers 57 languages including all major world languages and CLDR cardinal forms.
var pluralRules = map[string]func(pluralOperands) core.PluralCategory{
	// Other-only languages (no plural distinctions)
	"ja": pluralOtherOnly,
	"zh": pluralOtherOnly,
	"ko": pluralOtherOnly,
	"tr": pluralOtherOnly,
	"th": pluralOtherOnly,
	"vi": pluralOtherOnly,
	"id": pluralOtherOnly,
	"ms": pluralOtherOnly,
	"lo": pluralOtherOnly, // Lao: other only
	"my": pluralOtherOnly, // Burmese: other only

	// Simple one/other rules (one: i = 1 and v = 0)
	"en": pluralEnglish,
	"de": pluralEnglish, // same as English: one when i=1, v=0
	"it": pluralEnglish, // same as English: one when i=1, v=0
	"es": pluralEnglish, // same: one when i=1, v=0 (CLDR: n=1)
	"nl": pluralEnglish, // same as English
	"sv": pluralEnglish, // same as English
	"no": pluralEnglish, // same as English
	"da": pluralEnglish, // same as English
	"fi": pluralEnglish, // same as English
	"el": pluralEnglish, // Greek: one when n=1
	"he": pluralEnglish, // Hebrew: one when i=1 and v=0
	"sw": pluralEnglish, // Swahili: one when i=1 and v=0
	"ca": pluralEnglish, // Catalan: one when i=1 and v=0

	// Simple one/other rules (one: n = 1)
	"hu": pluralHungarian, // one when n=1
	"bg": pluralHungarian, // Bulgarian: one when n=1
	"sq": pluralHungarian, // Albanian: one when n=1
	"az": pluralHungarian, // Azerbaijani: one when n=1
	"kk": pluralHungarian, // Kazakh: one when n=1
	"ta": pluralHungarian, // Tamil: one when n=1
	"te": pluralHungarian, // Telugu: one when n=1
	"ne": pluralHungarian, // Nepali: one when n=1
	"ka": pluralHungarian, // Georgian: one when n=1

	// Portuguese-like (one: i = 0..1)
	"pt": pluralPortuguese, // pt: one when i=0..1
	"pa": pluralPortuguese, // Punjabi: one when n=0..1 (i=0 or i=1 with no fraction)

	// French-like (one: i = 0 or i = 1)
	"fr": pluralFrench,
	"hi": pluralFrench, // Hindi: one when i=0 or n=1
	"bn": pluralFrench, // Bengali: one when i=0 or n=1
	"am": pluralFrench, // Amharic: one when i=0 or n=1
	"kn": pluralFrench, // Kannada: one when i=0 or n=1
	"gu": pluralFrench, // Gujarati: one when i=0 or n=1
	"mr": pluralFrench, // Marathi: one when i=0 or n=1

	// Armenian (one: i = 0 or i = 1)
	"hy": pluralArmenian, // Armenian: one when i=0,1

	// Macedonian (one: v=0 and i%10=1 and i%100!=11, or f%10=1 and f%100!=11)
	"mk": pluralMacedonian,

	// Complex rules
	"ru": pluralRussian,
	"uk": pluralUkrainian,
	"pl": pluralPolish,
	"cs": pluralCzech,
	"ro": pluralRomanian,
	"ar": pluralArabic,
	"hr": pluralCroatian,
	"sl": pluralSlovenian,
	"lt": pluralLithuanian,
	"lv": pluralLatvian,
	"ga": pluralIrish,
	"cy": pluralWelsh,
}

// pluralOtherOnly returns Other for all inputs.
// Used by: ja, zh, ko, tr, th, vi, id, ms, lo, my.
func pluralOtherOnly(_ pluralOperands) core.PluralCategory {
	return core.Other
}

// pluralEnglish implements CLDR rules for English and similar languages.
// one: i = 1 and v = 0
// other: everything else
// Used by: en, de, it, nl, sv, no, da, fi, el, he, sw, ca.
func pluralEnglish(ops pluralOperands) core.PluralCategory {
	if ops.i == 1 && ops.v == 0 {
		return core.One
	}
	return core.Other
}

// pluralHungarian implements CLDR rules for Hungarian and similar languages.
// one: n = 1
// other: everything else
// Used by: hu, bg, sq, az, kk, ta, te, ne, ka.
func pluralHungarian(ops pluralOperands) core.PluralCategory {
	if ops.n == 1 {
		return core.One
	}
	return core.Other
}

// pluralFrench implements CLDR rules for French and similar languages.
// one: i = 0 or i = 1
// other: everything else
// Used by: fr, hi, bn, am, kn, gu, mr.
func pluralFrench(ops pluralOperands) core.PluralCategory {
	if ops.i == 0 || ops.i == 1 {
		return core.One
	}
	return core.Other
}

// pluralPortuguese implements CLDR rules for Portuguese.
// one: i = 0..1
// other: everything else
// Used by: pt, pa.
func pluralPortuguese(ops pluralOperands) core.PluralCategory {
	if ops.i == 0 || ops.i == 1 {
		return core.One
	}
	return core.Other
}

// pluralArmenian implements CLDR rules for Armenian.
// one: i = 0 or i = 1
// other: everything else
func pluralArmenian(ops pluralOperands) core.PluralCategory {
	if ops.i == 0 || ops.i == 1 {
		return core.One
	}
	return core.Other
}

// pluralMacedonian implements CLDR rules for Macedonian.
// one: v = 0 and i % 10 = 1 and i % 100 != 11, or f % 10 = 1 and f % 100 != 11
// other: everything else
func pluralMacedonian(ops pluralOperands) core.PluralCategory {
	iMod10 := ops.i % 10
	iMod100 := ops.i % 100
	fMod10 := ops.f % 10
	fMod100 := ops.f % 100

	if (ops.v == 0 && iMod10 == 1 && iMod100 != 11) || (fMod10 == 1 && fMod100 != 11) {
		return core.One
	}
	return core.Other
}

// isEastSlavicOne checks the "one" condition for East Slavic languages (Russian, Ukrainian).
// one: v = 0 and i % 10 = 1 and i % 100 != 11
func isEastSlavicOne(ops pluralOperands, mod10, mod100 uint64) bool {
	return ops.v == 0 && mod10 == 1 && mod100 != 11
}

// isEastSlavicFew checks the "few" condition for East Slavic languages.
// few: v = 0 and i % 10 = 2..4 and i % 100 != 12..14
func isEastSlavicFew(ops pluralOperands, mod10, mod100 uint64) bool {
	return ops.v == 0 && mod10 >= 2 && mod10 <= 4 && (mod100 < 12 || mod100 > 14)
}

// pluralRussian implements CLDR rules for Russian.
// one:  v = 0 and i % 10 = 1 and i % 100 != 11
// few:  v = 0 and i % 10 = 2..4 and i % 100 != 12..14
// many: v = 0 and (i % 10 = 0 or i % 10 = 5..9 or i % 100 = 11..14)
// other: everything else
func pluralRussian(ops pluralOperands) core.PluralCategory {
	mod10 := ops.i % 10
	mod100 := ops.i % 100

	if isEastSlavicOne(ops, mod10, mod100) {
		return core.One
	}
	if isEastSlavicFew(ops, mod10, mod100) {
		return core.Few
	}
	if ops.v == 0 && (mod10 == 0 || (mod10 >= 5 && mod10 <= 9) || (mod100 >= 11 && mod100 <= 14)) {
		return core.Many
	}
	return core.Other
}

// pluralUkrainian implements CLDR rules for Ukrainian.
// Same as Russian.
func pluralUkrainian(ops pluralOperands) core.PluralCategory {
	return pluralRussian(ops)
}

// isPolishMany checks the "many" condition for Polish.
// many: v = 0 and i != 1 and (i % 10 = 0..1 or i % 10 = 5..9 or i % 100 = 12..14)
func isPolishMany(ops pluralOperands, mod10, mod100 uint64) bool {
	return ops.v == 0 && ops.i != 1 && (mod10 <= 1 || (mod10 >= 5 && mod10 <= 9) || (mod100 >= 12 && mod100 <= 14))
}

// pluralPolish implements CLDR rules for Polish.
// one:  i = 1 and v = 0
// few:  v = 0 and i % 10 = 2..4 and i % 100 != 12..14
// many: v = 0 and (i != 1) and (i % 10 = 0..1 or i % 10 = 5..9 or i % 100 = 12..14)
// other: everything else
func pluralPolish(ops pluralOperands) core.PluralCategory {
	mod10 := ops.i % 10
	mod100 := ops.i % 100

	if ops.i == 1 && ops.v == 0 {
		return core.One
	}
	if isEastSlavicFew(ops, mod10, mod100) {
		return core.Few
	}
	if isPolishMany(ops, mod10, mod100) {
		return core.Many
	}
	return core.Other
}

// pluralCzech implements CLDR rules for Czech.
// one:  i = 1 and v = 0
// few:  i = 2..4 and v = 0
// many: v != 0
// other: everything else
func pluralCzech(ops pluralOperands) core.PluralCategory {
	if ops.i == 1 && ops.v == 0 {
		return core.One
	}
	if ops.i >= 2 && ops.i <= 4 && ops.v == 0 {
		return core.Few
	}
	if ops.v != 0 {
		return core.Many
	}
	return core.Other
}

// pluralRomanian implements CLDR rules for Romanian.
// one:  i = 1 and v = 0
// few:  v != 0 or n = 0 or (n != 1 and n % 100 = 1..19)
// other: everything else
func pluralRomanian(ops pluralOperands) core.PluralCategory {
	nMod100 := uint64(ops.n) % 100

	if ops.i == 1 && ops.v == 0 {
		return core.One
	}
	if ops.v != 0 || ops.n == 0 || (ops.n != 1 && nMod100 >= 1 && nMod100 <= 19) {
		return core.Few
	}
	return core.Other
}

// pluralArabic implements CLDR rules for Arabic (all six categories).
// zero:  n = 0
// one:   n = 1
// two:   n = 2
// few:   n % 100 = 3..10
// many:  n % 100 = 11..99
// other: everything else
func pluralArabic(ops pluralOperands) core.PluralCategory {
	nMod100 := uint64(ops.n) % 100

	switch {
	case ops.n == 0:
		return core.Zero
	case ops.n == 1:
		return core.One
	case ops.n == 2:
		return core.Two
	case nMod100 >= 3 && nMod100 <= 10:
		return core.Few
	case nMod100 >= 11 && nMod100 <= 99:
		return core.Many
	default:
		return core.Other
	}
}

// isCroatianOne checks the "one" condition for Croatian/Serbian/Bosnian.
func isCroatianOne(ops pluralOperands, iMod10, iMod100, fMod10, fMod100 uint64) bool {
	return (ops.v == 0 && iMod10 == 1 && iMod100 != 11) ||
		(fMod10 == 1 && fMod100 != 11)
}

// isCroatianFew checks the "few" condition for Croatian/Serbian/Bosnian.
func isCroatianFew(ops pluralOperands, iMod10, iMod100, fMod10, fMod100 uint64) bool {
	return (ops.v == 0 && iMod10 >= 2 && iMod10 <= 4 && (iMod100 < 12 || iMod100 > 14)) ||
		(fMod10 >= 2 && fMod10 <= 4 && (fMod100 < 12 || fMod100 > 14))
}

// pluralCroatian implements CLDR rules for Croatian/Serbian/Bosnian.
// one:  v = 0 and i % 10 = 1 and i % 100 != 11, or f % 10 = 1 and f % 100 != 11
// few:  v = 0 and i % 10 = 2..4 and i % 100 != 12..14, or f % 10 = 2..4 and f % 100 != 12..14
// other: everything else
func pluralCroatian(ops pluralOperands) core.PluralCategory {
	iMod10 := ops.i % 10
	iMod100 := ops.i % 100
	fMod10 := ops.f % 10
	fMod100 := ops.f % 100

	if isCroatianOne(ops, iMod10, iMod100, fMod10, fMod100) {
		return core.One
	}
	if isCroatianFew(ops, iMod10, iMod100, fMod10, fMod100) {
		return core.Few
	}
	return core.Other
}

// pluralSlovenian implements CLDR rules for Slovenian.
// one:  v = 0 and i % 100 = 1
// two:  v = 0 and i % 100 = 2
// few:  v = 0 and i % 100 = 3..4, or v != 0
// other: everything else
func pluralSlovenian(ops pluralOperands) core.PluralCategory {
	iMod100 := ops.i % 100

	if ops.v == 0 && iMod100 == 1 {
		return core.One
	}
	if ops.v == 0 && iMod100 == 2 {
		return core.Two
	}
	if (ops.v == 0 && (iMod100 == 3 || iMod100 == 4)) || ops.v != 0 {
		return core.Few
	}
	return core.Other
}

// pluralLithuanian implements CLDR rules for Lithuanian.
// one:  n % 10 = 1 and n % 100 != 11..19
// few:  n % 10 = 2..9 and n % 100 != 11..19
// many: f != 0
// other: everything else
func pluralLithuanian(ops pluralOperands) core.PluralCategory {
	nMod10 := uint64(ops.n) % 10
	nMod100 := uint64(ops.n) % 100

	if nMod10 == 1 && (nMod100 < 11 || nMod100 > 19) {
		return core.One
	}
	if nMod10 >= 2 && nMod10 <= 9 && (nMod100 < 11 || nMod100 > 19) {
		return core.Few
	}
	if ops.f != 0 {
		return core.Many
	}
	return core.Other
}

// isLatvianZero checks the "zero" condition for Latvian.
func isLatvianZero(ops pluralOperands, nMod10, nMod100, fMod100 uint64) bool {
	return nMod10 == 0 || (nMod100 >= 11 && nMod100 <= 19) ||
		(ops.v == 2 && fMod100 >= 11 && fMod100 <= 19)
}

// isLatvianOne checks the "one" condition for Latvian.
func isLatvianOne(ops pluralOperands, nMod10, nMod100, fMod10, fMod100 uint64) bool {
	return (nMod10 == 1 && nMod100 != 11) ||
		(ops.v == 2 && fMod10 == 1 && fMod100 != 11) ||
		(ops.v != 2 && fMod10 == 1)
}

// pluralLatvian implements CLDR rules for Latvian.
// zero:  n % 10 = 0 or n % 100 = 11..19 or (v = 2 and f % 100 = 11..19)
// one:   n % 10 = 1 and n % 100 != 11 or (v = 2 and f % 10 = 1 and f % 100 != 11) or (v != 2 and f % 10 = 1)
// other: everything else
func pluralLatvian(ops pluralOperands) core.PluralCategory {
	nMod10 := uint64(ops.n) % 10
	nMod100 := uint64(ops.n) % 100
	fMod10 := ops.f % 10
	fMod100 := ops.f % 100

	if isLatvianZero(ops, nMod10, nMod100, fMod100) {
		return core.Zero
	}
	if isLatvianOne(ops, nMod10, nMod100, fMod10, fMod100) {
		return core.One
	}
	return core.Other
}

// pluralIrish implements CLDR rules for Irish.
// one:   n = 1
// two:   n = 2
// few:   n = 3..6
// many:  n = 7..10
// other: everything else
func pluralIrish(ops pluralOperands) core.PluralCategory {
	n := uint64(ops.n)
	if ops.n != float64(n) {
		return core.Other
	}

	switch {
	case n == 1:
		return core.One
	case n == 2:
		return core.Two
	case n >= 3 && n <= 6:
		return core.Few
	case n >= 7 && n <= 10:
		return core.Many
	default:
		return core.Other
	}
}

// pluralWelsh implements CLDR rules for Welsh (all six categories).
// zero:  n = 0
// one:   n = 1
// two:   n = 2
// few:   n = 3
// many:  n = 6
// other: everything else
func pluralWelsh(ops pluralOperands) core.PluralCategory {
	switch ops.n {
	case 0:
		return core.Zero
	case 1:
		return core.One
	case 2:
		return core.Two
	case 3:
		return core.Few
	case 6:
		return core.Many
	default:
		return core.Other
	}
}

// =============================================================================
// ICU MessageFormat
// =============================================================================

// isICUMessageFormat checks if a string contains ICU MessageFormat syntax.
// It looks for the pattern of { followed by , which indicates an ICU expression.
func isICUMessageFormat(s string) bool {
	braceIdx := strings.IndexByte(s, '{')
	if braceIdx < 0 {
		return false
	}
	// Look for a comma after the opening brace (indicating variable, type pattern)
	rest := s[braceIdx+1:]
	return strings.ContainsRune(rest, ',')
}

// evaluateICUMessage parses and evaluates an ICU MessageFormat string.
// Supports plural and select expressions. Returns the raw template on parse failure.
func evaluateICUMessage(template string, args map[string]interface{}, locale string, resolver core.PluralResolver) string {
	result, ok := parseAndEvaluate(template, args, locale, resolver)
	if !ok {
		return template
	}
	return result
}

// parseAndEvaluate processes the template, evaluating ICU expressions and
// concatenating literal text.
func parseAndEvaluate(template string, args map[string]interface{}, locale string, resolver core.PluralResolver) (string, bool) {
	var sb strings.Builder
	sb.Grow(len(template))

	i := 0
	n := len(template)

	for i < n {
		if template[i] != '{' {
			sb.WriteByte(template[i])
			i++
			continue
		}

		// Found opening brace; try to parse an ICU expression
		expr, end, ok := parseICUExpression(template, i)
		if !ok {
			// Not a valid ICU expression, treat as literal
			sb.WriteByte(template[i])
			i++
			continue
		}

		evaluated := evaluateExpression(expr, args, locale, resolver)
		sb.WriteString(evaluated)
		i = end
	}

	return sb.String(), true
}

// icuExpression holds a parsed ICU expression.
type icuExpression struct {
	branches map[string]string
	variable string
	exprType string // "plural" or "select"
}

// parseICUExpression parses an ICU expression starting at position start.
// Returns the parsed expression, the position after the closing brace, and success.
func parseICUExpression(s string, start int) (icuExpression, int, bool) {
	if start >= len(s) || s[start] != '{' {
		return icuExpression{}, start, false
	}

	// Find the matching closing brace, tracking depth
	end := findMatchingBrace(s, start)
	if end < 0 {
		return icuExpression{}, start, false
	}

	inner := s[start+1 : end]
	expr, ok := parseInner(inner)
	if !ok {
		return icuExpression{}, start, false
	}

	return expr, end + 1, true
}

// findMatchingBrace finds the position of the closing brace matching the opening
// brace at position start. Returns -1 if not found.
func findMatchingBrace(s string, start int) int {
	depth := 0
	for i := start; i < len(s); i++ {
		switch s[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// parseInner parses the content between the outer braces of an ICU expression.
// Expected format: "variable, type, branch1 {text1} branch2 {text2} ..."
func parseInner(inner string) (icuExpression, bool) {
	firstComma := strings.IndexByte(inner, ',')
	if firstComma < 0 {
		return icuExpression{}, false
	}

	variable := strings.TrimSpace(inner[:firstComma])
	rest := inner[firstComma+1:]

	secondComma := strings.IndexByte(rest, ',')
	if secondComma < 0 {
		return icuExpression{}, false
	}

	exprType := strings.TrimSpace(rest[:secondComma])
	branchStr := rest[secondComma+1:]

	if exprType != "plural" && exprType != "select" {
		return icuExpression{}, false
	}

	branches := parseBranches(strings.TrimSpace(branchStr))
	if len(branches) == 0 {
		return icuExpression{}, false
	}

	return icuExpression{
		variable: variable,
		exprType: exprType,
		branches: branches,
	}, true
}

// isBranchWhitespace reports whether the byte is whitespace used in branch parsing.
func isBranchWhitespace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n'
}

// skipWhitespace advances past whitespace characters from position i.
func skipWhitespace(s string, i int) int {
	for i < len(s) && isBranchWhitespace(s[i]) {
		i++
	}
	return i
}

// readBranchKey reads a branch key (characters until whitespace or '{') from position i.
func readBranchKey(s string, i int) (key string, end int) {
	keyStart := i
	for i < len(s) && s[i] != ' ' && s[i] != '\t' && s[i] != '{' {
		i++
	}
	return s[keyStart:i], i
}

// parseBranches parses "key1 {text1} key2 {text2} ..." into a map.
func parseBranches(s string) map[string]string {
	branches := make(map[string]string)
	i := 0
	n := len(s)

	for i < n {
		i = skipWhitespace(s, i)
		if i >= n {
			break
		}

		key, keyEnd := readBranchKey(s, i)
		i = keyEnd
		if key == "" {
			break
		}

		// Skip whitespace before '{'
		i = skipWhitespace(s, i)
		if i >= n || s[i] != '{' {
			break
		}

		braceEnd := findMatchingBrace(s, i)
		if braceEnd < 0 {
			break
		}

		branches[key] = s[i+1 : braceEnd]
		i = braceEnd + 1
	}

	return branches
}

// evaluateExpression evaluates a parsed ICU expression with the given arguments.
func evaluateExpression(expr icuExpression, args map[string]interface{}, locale string, resolver core.PluralResolver) string {
	argVal, ok := args[expr.variable]
	if !ok {
		if other, exists := expr.branches["other"]; exists {
			return other
		}
		return ""
	}

	switch expr.exprType {
	case "plural":
		return evaluatePlural(expr, argVal, locale, resolver)
	case "select":
		return evaluateSelect(expr, argVal)
	default:
		if other, exists := expr.branches["other"]; exists {
			return other
		}
		return ""
	}
}

// evaluatePlural evaluates a plural ICU expression.
func evaluatePlural(expr icuExpression, count interface{}, locale string, resolver core.PluralResolver) string {
	category := resolver.Resolve(locale, count)
	countStr := formatCount(count)

	if text, ok := expr.branches[string(category)]; ok {
		return strings.ReplaceAll(text, "#", countStr)
	}

	if text, ok := expr.branches["other"]; ok {
		return strings.ReplaceAll(text, "#", countStr)
	}

	return countStr
}

// evaluateSelect evaluates a select ICU expression.
func evaluateSelect(expr icuExpression, value interface{}) string {
	var key string
	switch v := value.(type) {
	case string:
		key = v
	case core.GenderCategory:
		key = string(v)
	default:
		key = formatCount(value)
	}

	if text, ok := expr.branches[key]; ok {
		return text
	}

	if text, ok := expr.branches["other"]; ok {
		return text
	}

	return key
}

// =============================================================================
// Count formatting helpers
// =============================================================================

// formatCount converts a count value to its string representation for # replacement.
func formatCount(count interface{}) string {
	switch c := count.(type) {
	case int:
		return fmt.Sprintf("%d", c)
	case int64:
		return fmt.Sprintf("%d", c)
	case float64:
		return formatFloat64(c)
	case string:
		return c
	default:
		return fmt.Sprintf("%v", c)
	}
}

// formatFloat64 converts a float64 to its string representation.
// Integers are formatted without decimal places.
func formatFloat64(v float64) string {
	if v == float64(int64(v)) {
		return fmt.Sprintf("%d", int64(v))
	}
	return fmt.Sprintf("%g", v)
}
