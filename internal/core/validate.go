// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

package core

import (
	"errors"
	"fmt"
	"strings"
)

// =============================================================================
// Locale validation
// =============================================================================

// matchLanguagePart parses the language part (2-3 lowercase letters) and returns
// the number of bytes consumed. Returns 0 if the language part is invalid.
func matchLanguagePart(s string) int {
	n := len(s)
	i := 0
	for i < n && s[i] >= 'a' && s[i] <= 'z' {
		i++
	}
	if i < 2 || i > 3 {
		return 0
	}
	return i
}

// matchRegionPart parses the region part (exactly 2 uppercase letters) starting
// after the dash separator. Returns the total bytes consumed including the dash,
// or 0 if the region part is invalid.
func matchRegionPart(s string, offset int) int {
	n := len(s)
	if offset >= n || s[offset] != '-' {
		return 0
	}
	i := offset + 1
	for i < n && s[i] >= 'A' && s[i] <= 'Z' {
		i++
	}
	regionLen := i - offset - 1
	if regionLen != 2 {
		return 0
	}
	return i
}

// matchLocale validates a locale string against BCP 47 format.
// Accepts: 2-3 lowercase a-z, optionally followed by '-' and exactly 2 uppercase A-Z.
func matchLocale(s string) bool {
	n := len(s)
	if n < 2 || n > 6 {
		return false
	}

	langLen := matchLanguagePart(s)
	if langLen == 0 {
		return false
	}

	// Language-only code is valid.
	if langLen == n {
		return true
	}

	// Must have a region part consuming the rest of the string.
	return matchRegionPart(s, langLen) == n
}

// hasControlChars reports whether the string contains control characters.
func hasControlChars(s string) bool {
	for _, r := range s {
		if r < 0x20 || r == 0x7F {
			return true
		}
	}
	return false
}

// validateLocaleSecurity checks the locale string for path traversal and injection attacks.
func validateLocaleSecurity(locale string) error {
	if strings.Contains(locale, "..") {
		return NewErrInvalidLocale(locale, errors.New("contains path traversal sequence"))
	}
	if strings.ContainsAny(locale, "/\\") {
		return NewErrInvalidLocale(locale, errors.New("contains path separator"))
	}
	if strings.HasPrefix(locale, "~") {
		return NewErrInvalidLocale(locale, errors.New("contains tilde path"))
	}
	if hasControlChars(locale) {
		return NewErrInvalidLocale(locale, errors.New("contains control characters"))
	}
	return nil
}

// ValidateLocale validates a locale string according to BCP 47 format
// and performs security checks to prevent path traversal and injection attacks.
func ValidateLocale(locale string) error {
	if locale == "" {
		return NewErrInvalidLocale(locale, errors.New("locale cannot be empty"))
	}

	if len(locale) > MaxLocaleLength {
		return NewErrInvalidLocale(locale, fmt.Errorf("exceeds maximum length of %d characters", MaxLocaleLength))
	}

	if err := validateLocaleSecurity(locale); err != nil {
		return err
	}

	if !matchLocale(locale) {
		return NewErrInvalidLocale(locale, errors.New("does not match BCP 47 format (e.g., en-US, es-MX)"))
	}

	return nil
}

// =============================================================================
// Key validation
// =============================================================================

// isKeyByte reports whether c is in the allowed set [a-zA-Z0-9._-].
func isKeyByte(c byte) bool {
	return (c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') ||
		c == '.' || c == '_' || c == '-'
}

// matchKey validates that every byte in the string is in the allowed set [a-zA-Z0-9._-].
func matchKey(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if !isKeyByte(s[i]) {
			return false
		}
	}
	return true
}

// validateKeyFormat checks character-level constraints on the key.
func validateKeyFormat(key string) error {
	for _, r := range key {
		if r < 0x20 || r == 0x7F {
			return NewErrInvalidKey(key, errors.New("contains control characters"))
		}
	}

	if !matchKey(key) {
		return NewErrInvalidKey(key, errors.New("contains invalid characters (only a-z, A-Z, 0-9, ., _, - allowed)"))
	}

	return nil
}

// validateKeyDots checks dot-related structural constraints.
func validateKeyDots(key string) error {
	if strings.HasPrefix(key, ".") || strings.HasSuffix(key, ".") {
		return NewErrInvalidKey(key, errors.New("cannot start or end with dot"))
	}

	if strings.Contains(key, "..") {
		return NewErrInvalidKey(key, errors.New("cannot contain consecutive dots"))
	}

	depth := strings.Count(key, ".") + 1
	if depth > MaxKeyDepth {
		return NewErrInvalidKey(key, fmt.Errorf("exceeds maximum depth of %d levels", MaxKeyDepth))
	}

	return nil
}

// ValidateKey validates a translation key for security and format compliance.
func ValidateKey(key string) error {
	if key == "" {
		return NewErrInvalidKey(key, errors.New("key cannot be empty"))
	}

	if len(key) > MaxKeyLength {
		return NewErrInvalidKey(key, fmt.Errorf("exceeds maximum length of %d characters", MaxKeyLength))
	}

	if err := validateKeyFormat(key); err != nil {
		return err
	}

	return validateKeyDots(key)
}

// =============================================================================
// Format string validation
// =============================================================================

// MaxFormatPrecision is the maximum allowed precision value in a printf-style
// format specifier (e.g., %.1000s). Larger values would let a malicious or
// corrupt translation string balloon fmt.Sprintf's output into a huge
// allocation — e.g., "%.99999999s" would cause fmt to pad the argument with
// spaces up to ~100M characters. 1024 is chosen as a generous cap: CJK width
// alignment, long log lines, and common formatting use well under 100; 1024
// accommodates a paragraph of padded text without inviting abuse.
const MaxFormatPrecision = 1024

// countFormatSpecifiers counts format specifiers in a string, skipping %% pairs.
// A specifier is any '%' followed by a non-'%' byte.
func countFormatSpecifiers(s string) int {
	count := 0
	i := 0
	n := len(s)
	for i < n {
		if s[i] == '%' {
			if i+1 < n {
				if s[i+1] == '%' {
					// Escaped %%, skip both
					i += 2
					continue
				}
				// Format specifier found
				count++
				i += 2
				continue
			}
			// '%' at end of string, not a valid specifier
			i++
			continue
		}
		i++
	}
	return count
}

// parseUintFromDigits reads a run of decimal digits from s starting at i.
// Returns the parsed value, the position after the digits, and ok=true if at
// least one digit was consumed. Uses a saturating accumulator so overflow
// produces a capped value larger than MaxFormatPrecision (which the caller
// will then reject) instead of wrapping.
func parseUintFromDigits(s string, i int) (value, next int, ok bool) {
	n := len(s)
	start := i
	v := 0
	for i < n && s[i] >= '0' && s[i] <= '9' {
		// Saturate to avoid int overflow on attacker-controlled input.
		if v > MaxFormatPrecision*10 {
			v = MaxFormatPrecision * 10
		} else {
			v = v*10 + int(s[i]-'0')
		}
		i++
	}
	return v, i, i > start
}

// validateFormatPrecision walks format specifiers in s and returns an error
// if any specifier declares a precision > MaxFormatPrecision. Unrecognized
// specifier shapes are permitted (fmt.Sprintf will reject them at format
// time) — this function's job is specifically to reject
// precision-amplification attacks.
//
// Grammar handled (printf-style, loose):
//
//	%[flags][width][.precision]verb
//
// We only parse the optional ".precision" tail and check its numeric value.
// A bare '*' in place of a digit run is also rejected as conservative defense
// (we have no arg-list visibility here; fmt.Sprintf will handle it correctly
// if legitimate, but indirect precision via * is not something translation
// strings should need).
func validateFormatPrecision(s string) error {
	n := len(s)
	i := 0
	for i < n {
		if s[i] != '%' {
			i++
			continue
		}
		if i+1 >= n {
			break
		}
		if s[i+1] == '%' {
			i += 2
			continue
		}
		j, err := scanFormatSpecifier(s, i+1)
		if err != nil {
			return err
		}
		i = j
	}
	return nil
}

// skipFormatFlags advances j past any printf flag characters ('-', '+', ' ', '#', '0').
func skipFormatFlags(s string, j int) int {
	for j < len(s) {
		c := s[j]
		if c != '-' && c != '+' && c != ' ' && c != '#' && c != '0' {
			break
		}
		j++
	}
	return j
}

// skipFormatWidth advances j past an optional printf width field (digits or '*').
func skipFormatWidth(s string, j int) int {
	if j < len(s) && s[j] == '*' {
		return j + 1
	}
	_, j, _ = parseUintFromDigits(s, j)
	return j
}

// scanFormatSpecifier advances past a single printf format specifier that begins
// at position j (the character after the leading '%'). It validates the
// precision field and returns the position after the verb byte, or an error if
// an illegal precision is found.
func scanFormatSpecifier(s string, j int) (int, error) {
	j = skipFormatFlags(s, j)
	j = skipFormatWidth(s, j)

	if j < len(s) && s[j] == '.' {
		var err error
		j, err = scanPrecision(s, j+1)
		if err != nil {
			return j, err
		}
	}

	if j < len(s) {
		return j + 1, nil
	}
	return j, nil
}

// scanPrecision parses the digits (or '*') after a '.' in a format specifier
// and returns the updated position. Returns an error if the precision is
// indirect (%.*) or exceeds MaxFormatPrecision.
func scanPrecision(s string, j int) (int, error) {
	if j < len(s) && s[j] == '*' {
		return j, NewErrInvalidFormat(s, errors.New("indirect precision (%.*) is not allowed in translation strings"))
	}
	prec, jAfter, hasDigits := parseUintFromDigits(s, j)
	if hasDigits {
		if prec > MaxFormatPrecision {
			return j, NewErrInvalidFormat(s, fmt.Errorf("precision %d exceeds maximum of %d", prec, MaxFormatPrecision))
		}
		return jAfter, nil
	}
	// A bare "%.verb" (no digits, no *) is precision 0 — fine.
	return j, nil
}

// ValidateFormatString validates that format is safe and that its format
// specifiers match argCount. The dangerous %n specifier is rejected, and
// precision values greater than MaxFormatPrecision are rejected to prevent
// allocation-amplification attacks (e.g., a malicious "%.99999999s"
// translation string coercing fmt.Sprintf into a ~100MB allocation).
// Returns an ErrInvalidFormat on any violation.
func ValidateFormatString(format string, argCount int) error {
	// Check for dangerous %n specifier (writes to memory in C printf)
	if strings.Contains(format, "%n") {
		return NewErrInvalidFormat(format, errors.New("%n specifier is not allowed"))
	}

	if err := validateFormatPrecision(format); err != nil {
		return err
	}

	// Count format specifiers (excluding escaped %%)
	specifierCount := countFormatSpecifiers(format)

	// Verify argument count matches
	if specifierCount != argCount {
		return NewErrInvalidFormat(format, fmt.Errorf("expected %d arguments but got %d", specifierCount, argCount))
	}

	return nil
}

// =============================================================================
// Output sanitization
// =============================================================================

// isCSITerminator reports whether b is a valid CSI sequence terminator [a-zA-Z].
func isCSITerminator(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

// isCSIParameter reports whether b is a valid CSI parameter byte [0-9;].
func isCSIParameter(b byte) bool {
	return (b >= '0' && b <= '9') || b == ';'
}

// skipCSISequence advances past a CSI (Control Sequence Introducer) sequence
// starting at position i (which points to the ESC byte). Returns the position
// after the sequence.
func skipCSISequence(s string, i int) int {
	n := len(s)
	// Skip ESC and '['
	i += 2
	// Skip parameter bytes [0-9;]
	for i < n && isCSIParameter(s[i]) {
		i++
	}
	// Skip the terminating letter [a-zA-Z]
	if i < n && isCSITerminator(s[i]) {
		i++
	}
	return i
}

// stripANSI removes ANSI escape sequences from a string.
// Scans for 0x1b followed by '[', then skips bytes in [0-9;] until
// a letter [a-zA-Z] terminates the sequence.
func stripANSI(s string) string {
	if !strings.Contains(s, "\x1b") {
		return s
	}

	var result strings.Builder
	result.Grow(len(s))

	i := 0
	n := len(s)
	for i < n {
		if s[i] != 0x1b {
			result.WriteByte(s[i])
			i++
			continue
		}

		// ESC followed by '[' starts a CSI sequence.
		if i+1 < n && s[i+1] == '[' {
			i = skipCSISequence(s, i)
			continue
		}

		// Lone escape byte (not followed by '['), skip it.
		i++
	}

	return result.String()
}

// SanitizeOutput removes dangerous characters and sequences from output strings.
// It strips control characters (except newline and tab), ANSI escape sequences,
// and BiDi override characters (U+202A through U+202E). The result is truncated
// to MaxOutputLength bytes if it exceeds that limit.
func SanitizeOutput(s string) string {
	s = stripANSI(s)

	var result strings.Builder
	result.Grow(len(s))

	for _, r := range s {
		if r == '\n' || r == '\t' {
			result.WriteRune(r)
			continue
		}

		// Remove control characters (0x00-0x1F and 0x7F)
		if r < 0x20 || r == 0x7F {
			continue
		}

		// Remove BiDi override characters (U+202A - U+202E)
		if r >= 0x202A && r <= 0x202E {
			continue
		}

		result.WriteRune(r)
	}

	sanitized := result.String()

	if len(sanitized) > MaxOutputLength {
		sanitized = sanitized[:MaxOutputLength]
	}

	return sanitized
}
