// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

package engine

import (
	"fmt"
	"sort"
	"strconv"
	"sync"
	"unicode/utf8"

	"github.com/0verkilll/i18n/internal/core"
)

// =============================================================================
// JSON parser
// =============================================================================

// Compile-time assertion that JSONParser implements core.TranslationParser.
var _ core.TranslationParser = (*JSONParser)(nil)

// JSONParser parses JSON translation files into key-value maps. It enforces
// size limits (core.MaxJSONSize), nesting depth limits (core.MaxJSONDepth),
// and key count limits (core.MaxKeyCount) to prevent denial-of-service attacks
// from malicious input.
type JSONParser struct{}

// NewJSONParser creates a new JSONParser with built-in security limits for
// input size, nesting depth, and key count.
func NewJSONParser() *JSONParser {
	return &JSONParser{}
}

// Parse decodes data from JSON into a map of translation keys to values. The
// data must be a JSON object (not array, string, or null). Parse enforces
// core.MaxJSONSize, core.MaxJSONDepth, and core.MaxKeyCount, returning an
// ErrInvalidFormat on violations.
func (p *JSONParser) Parse(data []byte) (map[string]interface{}, error) {
	if err := validateJSONInput(data); err != nil {
		return nil, err
	}

	jp := jsonParser{data: data}
	jp.skipWhitespace()

	if jp.pos >= len(jp.data) || jp.data[jp.pos] != '{' {
		return nil, core.NewErrInvalidFormat("JSON", fmt.Errorf("expected JSON object"))
	}

	val, err := jp.parseValue()
	if err != nil {
		return nil, core.NewErrInvalidFormat("JSON", err)
	}

	jp.skipWhitespace()
	if jp.pos < len(jp.data) {
		return nil, core.NewErrInvalidFormat("JSON", fmt.Errorf("unexpected data after top-level value at position %d", jp.pos))
	}

	return validateJSONResult(val)
}

// validateJSONInput checks size and emptiness before parsing.
func validateJSONInput(data []byte) error {
	if len(data) > core.MaxJSONSize {
		return core.NewErrInvalidFormat("JSON", fmt.Errorf("input exceeds maximum size of %d bytes", core.MaxJSONSize))
	}
	if len(data) == 0 {
		return core.NewErrInvalidFormat("JSON", fmt.Errorf("empty input"))
	}
	return nil
}

// validateJSONResult ensures the parsed value is a non-nil object.
// Depth and key-count limits are enforced during streaming parse by the
// jsonParser (p.depth and p.keys), so no post-parse re-traversal is needed.
func validateJSONResult(val interface{}) (map[string]interface{}, error) {
	result, ok := val.(map[string]interface{})
	if !ok || result == nil {
		return nil, core.NewErrInvalidFormat("JSON", fmt.Errorf("expected JSON object"))
	}
	return result, nil
}

// =============================================================================
// Hand-written recursive descent JSON parser
// =============================================================================

// jsonParser holds the state for a single parse operation. It walks through the
// byte slice using a position cursor and tracks nesting depth and key count to
// enforce the security limits defined in core.
type jsonParser struct {
	data  []byte
	pos   int
	depth int
	keys  int
}

// atEnd reports whether the parser has consumed all input.
func (p *jsonParser) atEnd() bool {
	return p.pos >= len(p.data)
}

// peekIsDigit reports whether the byte at the current position is an ASCII digit.
// Returns false if at end of input.
func (p *jsonParser) peekIsDigit() bool {
	return !p.atEnd() && isDigit(p.data[p.pos])
}

// skipDigits advances pos past consecutive ASCII digits.
func (p *jsonParser) skipDigits() {
	for p.peekIsDigit() {
		p.pos++
	}
}

// isDigit reports whether b is an ASCII digit ('0' through '9').
func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

// skipWhitespace advances pos past any JSON whitespace characters (space, tab,
// newline, carriage return).
func (p *jsonParser) skipWhitespace() {
	for p.pos < len(p.data) {
		c := p.data[p.pos]
		if c != ' ' && c != '\t' && c != '\n' && c != '\r' {
			return
		}
		p.pos++
	}
}

// parseValue dispatches to the correct type parser based on the first non-
// whitespace character. It enforces MaxJSONDepth before descending into
// objects or arrays.
func (p *jsonParser) parseValue() (interface{}, error) {
	p.skipWhitespace()
	if p.atEnd() {
		return nil, fmt.Errorf("unexpected end of input")
	}

	switch p.data[p.pos] {
	case '{':
		return p.parseObject()
	case '[':
		return p.parseArray()
	case '"':
		return p.parseString()
	case 't', 'f', 'n':
		return p.parseLiteral()
	case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return p.parseNumber()
	default:
		return nil, fmt.Errorf("unexpected character %q at position %d", p.data[p.pos], p.pos)
	}
}

// parseObject parses a JSON object: { key : value [, key : value ...] }.
// It enforces MaxJSONDepth for nesting and counts keys toward MaxKeyCount.
func (p *jsonParser) parseObject() (map[string]interface{}, error) {
	if p.depth > core.MaxJSONDepth {
		return nil, fmt.Errorf("JSON nesting exceeds maximum depth of %d", core.MaxJSONDepth)
	}
	p.depth++
	defer func() { p.depth-- }()

	// Consume opening '{'
	p.pos++

	result := make(map[string]interface{})

	p.skipWhitespace()
	if p.atEnd() {
		return nil, fmt.Errorf("unexpected end of input in object")
	}

	// Empty object
	if p.data[p.pos] == '}' {
		p.pos++
		return result, nil
	}

	for {
		if err := p.parseObjectEntry(result); err != nil {
			return nil, err
		}

		// Expect ',' or '}'
		p.skipWhitespace()
		if p.atEnd() {
			return nil, fmt.Errorf("unexpected end of input in object")
		}

		if p.data[p.pos] == '}' {
			p.pos++
			return result, nil
		}

		if err := p.consumeObjectComma(); err != nil {
			return nil, err
		}
	}
}

// parseObjectEntry parses a single key-value pair and stores it in result.
func (p *jsonParser) parseObjectEntry(result map[string]interface{}) error {
	// Parse key
	p.skipWhitespace()
	if p.atEnd() {
		return fmt.Errorf("unexpected end of input in object key")
	}
	if p.data[p.pos] != '"' {
		return fmt.Errorf("expected string key at position %d, got %q", p.pos, p.data[p.pos])
	}

	key, err := p.parseString()
	if err != nil {
		return fmt.Errorf("invalid object key: %w", err)
	}

	p.keys++
	if p.keys > core.MaxKeyCount {
		return fmt.Errorf("key count exceeds maximum of %d", core.MaxKeyCount)
	}

	// Expect ':'
	p.skipWhitespace()
	if p.atEnd() {
		return fmt.Errorf("unexpected end of input after object key")
	}
	if p.data[p.pos] != ':' {
		return fmt.Errorf("expected ':' after object key at position %d, got %q", p.pos, p.data[p.pos])
	}
	p.pos++

	// Parse value
	val, err := p.parseValue()
	if err != nil {
		return err
	}

	result[key] = val
	return nil
}

// consumeObjectComma consumes a comma separator in an object and checks for
// trailing commas.
func (p *jsonParser) consumeObjectComma() error {
	if p.data[p.pos] != ',' {
		return fmt.Errorf("expected ',' or '}' in object at position %d, got %q", p.pos, p.data[p.pos])
	}
	p.pos++

	// Check for trailing comma (comma followed by '}')
	p.skipWhitespace()
	if p.atEnd() {
		return fmt.Errorf("unexpected end of input after ',' in object")
	}
	if p.data[p.pos] == '}' {
		return fmt.Errorf("trailing comma in object at position %d", p.pos)
	}
	return nil
}

// parseArray parses a JSON array: [ value [, value ...] ].
func (p *jsonParser) parseArray() ([]interface{}, error) {
	if p.depth > core.MaxJSONDepth {
		return nil, fmt.Errorf("JSON nesting exceeds maximum depth of %d", core.MaxJSONDepth)
	}
	p.depth++
	defer func() { p.depth-- }()

	// Consume opening '['
	p.pos++

	var result []interface{}

	p.skipWhitespace()
	if p.atEnd() {
		return nil, fmt.Errorf("unexpected end of input in array")
	}

	// Empty array
	if p.data[p.pos] == ']' {
		p.pos++
		return result, nil
	}

	for {
		val, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		result = append(result, val)

		p.skipWhitespace()
		if p.atEnd() {
			return nil, fmt.Errorf("unexpected end of input in array")
		}

		if p.data[p.pos] == ']' {
			p.pos++
			return result, nil
		}

		if p.data[p.pos] != ',' {
			return nil, fmt.Errorf("expected ',' or ']' in array at position %d, got %q", p.pos, p.data[p.pos])
		}
		p.pos++
	}
}

// parseString parses a JSON string with full escape sequence support including
// \uXXXX unicode escapes and surrogate pairs.
func (p *jsonParser) parseString() (string, error) {
	if p.atEnd() || p.data[p.pos] != '"' {
		return "", fmt.Errorf("expected '\"' at position %d", p.pos)
	}
	p.pos++ // consume opening quote

	// Fast path: scan for end quote without escapes
	start := p.pos
	for !p.atEnd() {
		c := p.data[p.pos]
		if c == '\\' {
			break
		}
		if c == '"' {
			// No escapes, return the substring directly
			s := string(p.data[start:p.pos])
			p.pos++ // consume closing quote
			return s, nil
		}
		// Reject unescaped control characters (U+0000 through U+001F)
		if c < 0x20 {
			return "", fmt.Errorf("invalid control character at position %d", p.pos)
		}
		p.pos++
	}

	if p.atEnd() {
		return "", fmt.Errorf("unterminated string starting at position %d", start-1)
	}

	// Slow path: build the string with escape processing
	buf := make([]byte, 0, p.pos-start+32)
	buf = append(buf, p.data[start:p.pos]...)

	return p.parseStringEscaped(buf)
}

// parseStringEscaped processes the slow path of string parsing, handling escape
// sequences. It continues from where the fast path left off, appending to buf.
func (p *jsonParser) parseStringEscaped(buf []byte) (string, error) {
	for !p.atEnd() {
		c := p.data[p.pos]

		if c == '"' {
			p.pos++
			return string(buf), nil
		}

		// Reject unescaped control characters (U+0000 through U+001F)
		if c < 0x20 {
			return "", fmt.Errorf("invalid control character at position %d", p.pos)
		}

		if c != '\\' {
			buf = append(buf, c)
			p.pos++
			continue
		}

		escaped, err := p.processEscape()
		if err != nil {
			return "", err
		}
		buf = append(buf, escaped...)
	}

	return "", fmt.Errorf("unterminated string")
}

// simpleEscapes maps single-character escape codes to their byte values.
var simpleEscapes = [256]byte{
	'"':  '"',
	'\\': '\\',
	'/':  '/',
	'b':  '\b',
	'f':  '\f',
	'n':  '\n',
	'r':  '\r',
	't':  '\t',
}

// simpleEscapeValid tracks which escape byte codes are valid simple escapes.
var simpleEscapeValid = [256]bool{
	'"':  true,
	'\\': true,
	'/':  true,
	'b':  true,
	'f':  true,
	'n':  true,
	'r':  true,
	't':  true,
}

// processEscape reads a single escape sequence starting at the current backslash
// position. It returns the decoded bytes as a slice and any error.
func (p *jsonParser) processEscape() ([]byte, error) {
	p.pos++ // consume backslash
	if p.atEnd() {
		return nil, fmt.Errorf("unterminated escape sequence at end of input")
	}

	esc := p.data[p.pos]
	p.pos++

	if simpleEscapeValid[esc] {
		return []byte{simpleEscapes[esc]}, nil
	}

	if esc == 'u' {
		r, err := p.parseUnicodeEscape()
		if err != nil {
			return nil, err
		}
		var ubuf [4]byte
		n := utf8.EncodeRune(ubuf[:], r)
		return ubuf[:n], nil
	}

	return nil, fmt.Errorf("invalid escape character %q at position %d", esc, p.pos-1)
}

// parseUnicodeEscape reads a unicode escape value, handling surrogate pairs.
// It returns the final rune.
func (p *jsonParser) parseUnicodeEscape() (rune, error) {
	r1, err := p.parseHex4()
	if err != nil {
		return 0, err
	}

	// Handle UTF-16 surrogate pairs
	switch {
	case r1 >= 0xD800 && r1 <= 0xDBFF:
		return p.parseSurrogatePair(r1)
	case r1 >= 0xDC00 && r1 <= 0xDFFF:
		return 0, fmt.Errorf("unexpected low surrogate U+%04X at position %d", r1, p.pos)
	default:
		// parseHex4 returns values in 0..0xFFFF, safe for rune conversion.
		return rune(r1), nil //nolint:gosec // r1 is 0..0xFFFF from parseHex4
	}
}

// parseSurrogatePair completes a UTF-16 surrogate pair given the high surrogate.
func (p *jsonParser) parseSurrogatePair(high int) (rune, error) {
	if p.pos+1 >= len(p.data) || p.data[p.pos] != '\\' || p.data[p.pos+1] != 'u' {
		return 0, fmt.Errorf("expected low surrogate after high surrogate at position %d", p.pos)
	}
	p.pos += 2 // skip \u

	low, err := p.parseHex4()
	if err != nil {
		return 0, err
	}
	if low < 0xDC00 || low > 0xDFFF {
		return 0, fmt.Errorf("invalid low surrogate U+%04X at position %d", low, p.pos)
	}

	// Combined value is always <= 0x10FFFF, safe for rune conversion.
	combined := 0x10000 + (high-0xD800)*0x400 + (low - 0xDC00)
	return rune(combined), nil //nolint:gosec // combined is <= 0x10FFFF
}

// parseHex4 reads exactly 4 hexadecimal digits from the current position and
// returns the decoded 16-bit value.
func (p *jsonParser) parseHex4() (int, error) {
	if p.pos+4 > len(p.data) {
		return 0, fmt.Errorf("incomplete unicode escape at position %d", p.pos)
	}

	var val int
	for i := 0; i < 4; i++ {
		c := p.data[p.pos+i]
		switch {
		case c >= '0' && c <= '9':
			val = val*16 + int(c-'0')
		case c >= 'a' && c <= 'f':
			val = val*16 + int(c-'a') + 10
		case c >= 'A' && c <= 'F':
			val = val*16 + int(c-'A') + 10
		default:
			return 0, fmt.Errorf("invalid hex digit %q in unicode escape at position %d", c, p.pos+i)
		}
	}
	p.pos += 4
	return val, nil
}

// parseNumber parses a JSON number. It returns int64 for integer values and
// float64 for values containing a decimal point or exponent.
func (p *jsonParser) parseNumber() (interface{}, error) {
	start := p.pos

	// Optional leading minus
	if !p.atEnd() && p.data[p.pos] == '-' {
		p.pos++
	}

	// Integer part
	if p.atEnd() {
		return nil, fmt.Errorf("unexpected end of input in number")
	}

	if err := p.scanIntegerPart(start); err != nil {
		return nil, err
	}

	isFloat := p.scanFractionalPart()

	floatExp, err := p.scanExponentPart()
	if err != nil {
		return nil, err
	}

	return p.convertNumber(start, isFloat || floatExp)
}

// scanIntegerPart scans the integer digits of a JSON number.
func (p *jsonParser) scanIntegerPart(start int) error {
	switch {
	case p.data[p.pos] == '0':
		p.pos++
		// After a leading zero, next char must not be a digit (no leading zeros)
		if p.peekIsDigit() {
			return fmt.Errorf("leading zeros not allowed in number at position %d", start)
		}
	case p.data[p.pos] >= '1' && p.data[p.pos] <= '9':
		p.pos++
		p.skipDigits()
	default:
		return fmt.Errorf("invalid number at position %d", start)
	}
	return nil
}

// scanFractionalPart scans the optional fractional part of a JSON number.
// Returns true if a fractional part was found.
func (p *jsonParser) scanFractionalPart() bool {
	if p.atEnd() || p.data[p.pos] != '.' {
		return false
	}

	p.pos++
	p.skipDigits()
	return true
}

// scanExponentPart scans the optional exponent part of a JSON number.
// Returns true if an exponent was found.
func (p *jsonParser) scanExponentPart() (bool, error) {
	if p.atEnd() {
		return false, nil
	}

	if p.data[p.pos] != 'e' && p.data[p.pos] != 'E' {
		return false, nil
	}

	p.pos++
	p.skipExponentSign()

	if !p.peekIsDigit() {
		return false, fmt.Errorf("expected digit in exponent at position %d", p.pos)
	}
	p.skipDigits()
	return true, nil
}

// skipExponentSign skips an optional '+' or '-' sign in an exponent.
func (p *jsonParser) skipExponentSign() {
	if !p.atEnd() && (p.data[p.pos] == '+' || p.data[p.pos] == '-') {
		p.pos++
	}
}

// convertNumber converts the scanned number substring to int64 or float64.
func (p *jsonParser) convertNumber(start int, isFloat bool) (interface{}, error) {
	numStr := string(p.data[start:p.pos])

	if isFloat {
		val, err := strconv.ParseFloat(numStr, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid number %q: %w", numStr, err)
		}
		return val, nil
	}

	return p.convertInteger(numStr)
}

// convertInteger attempts to parse numStr as int64, falling back to float64
// for values that overflow int64.
func (p *jsonParser) convertInteger(numStr string) (interface{}, error) {
	val, err := strconv.ParseInt(numStr, 10, 64)
	if err == nil {
		return val, nil
	}

	// Fall back to float64 for integers too large for int64
	fval, ferr := strconv.ParseFloat(numStr, 64)
	if ferr != nil {
		return nil, fmt.Errorf("invalid number %q: %w", numStr, err)
	}
	return fval, nil
}

// parseLiteral parses a JSON literal: true, false, or null.
func (p *jsonParser) parseLiteral() (interface{}, error) {
	if p.matchLiteral("true") {
		return true, nil
	}
	if p.matchLiteral("false") {
		return false, nil
	}
	if p.matchLiteral("null") {
		return nil, nil
	}
	return nil, fmt.Errorf("unexpected literal at position %d", p.pos)
}

// matchLiteral checks if the data at the current position matches the given
// literal string exactly and advances the position if it does.
func (p *jsonParser) matchLiteral(lit string) bool {
	end := p.pos + len(lit)
	if end > len(p.data) {
		return false
	}
	if string(p.data[p.pos:end]) == lit {
		p.pos = end
		return true
	}
	return false
}

// =============================================================================
// Parser registry
// =============================================================================

// Registry holds a map of file extensions to core.TranslationParser implementations.
// It provides thread-safe registration and lookup of parsers by file extension.
//
// Registration typically happens in init() functions, which run in a single
// goroutine before main(). However, the Registry is safe for concurrent access
// at any time via its internal sync.RWMutex. Callers registering parsers
// concurrently outside init() should be aware of potential races between
// registration and first use if done in separate goroutines.
type Registry struct {
	parsers map[string]core.TranslationParser
	mu      sync.RWMutex
}

// NewRegistry returns an empty parser registry.
func NewRegistry() *Registry {
	return &Registry{
		parsers: make(map[string]core.TranslationParser),
	}
}

// RegisterParser registers a core.TranslationParser for the given file extension.
// The extension must start with a dot followed by one or more lowercase
// alphanumeric characters (e.g., ".json", ".toml", ".yaml"). If a parser
// is already registered for the extension, it is overwritten (last-write-wins).
//
// Returns an error if ext is invalid or p is nil.
//
// # External Module Pattern
//
// A separate Go module can provide parsers for additional formats without
// adding any dependency to the core i18n module. The external module:
//
//  1. Implements the core.TranslationParser interface
//  2. Calls RegisterParser in its init() function
//
// For example, a TOML parser module would contain:
//
//	func init() {
//	    i18n.RegisterParser(".toml", &TOMLParser{})
//	}
//
// Application code activates the parser via a blank import:
//
//	import _ "github.com/0verkilll/i18n-toml"
//
// The same pattern applies to YAML or any other format. External parsers
// should enforce equivalent size and nesting depth limits (see core.MaxJSONSize
// and core.MaxJSONDepth) to maintain security parity with the built-in JSON parser.
func (r *Registry) RegisterParser(ext string, p core.TranslationParser) error {
	if err := validateExtension(ext); err != nil {
		return err
	}
	if p == nil {
		return fmt.Errorf("parser cannot be nil")
	}

	r.mu.Lock()
	r.parsers[ext] = p
	r.mu.Unlock()

	return nil
}

// GetParser returns the core.TranslationParser registered for the given extension.
// Returns ErrUnknownFormat if no parser is registered for the extension.
func (r *Registry) GetParser(ext string) (core.TranslationParser, error) {
	r.mu.RLock()
	p, ok := r.parsers[ext]
	r.mu.RUnlock()

	if !ok {
		return nil, core.NewErrUnknownFormat(ext)
	}
	return p, nil
}

// RegisteredFormats returns a sorted slice of all registered file extensions.
// The returned slice is safe to modify; it does not share memory with the
// registry internals.
func (r *Registry) RegisteredFormats() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	formats := make([]string, 0, len(r.parsers))
	for ext := range r.parsers {
		formats = append(formats, ext)
	}
	sort.Strings(formats)
	return formats
}

// defaultRegistry is the package-level parser registry, lazily initialized on
// first use via getDefaultRegistry(). The built-in JSON parser is registered
// during initialization; if registration fails, an error is logged and the
// registry continues without it.
var defaultRegistry *Registry

// defaultRegistryOnce ensures the default registry is initialized exactly once.
var defaultRegistryOnce sync.Once

// getDefaultRegistry returns the package-level parser registry, initializing it
// on first call. The built-in JSON parser is registered during initialization.
// If registration fails, the error is logged and the registry remains usable
// but without the JSON parser.
func getDefaultRegistry() *Registry {
	defaultRegistryOnce.Do(func() {
		defaultRegistry = NewRegistry()
		if err := defaultRegistry.RegisterParser(".json", NewJSONParser()); err != nil {
			GetLogger().Error("failed to register built-in JSON parser",
				"ext", ".json",
				"error", err,
			)
		}
	})
	return defaultRegistry
}

// RegisterParser registers a core.TranslationParser for the given file extension
// in the default package-level registry. See Registry.RegisterParser for details.
func RegisterParser(ext string, p core.TranslationParser) error {
	return getDefaultRegistry().RegisterParser(ext, p)
}

// GetParser returns the core.TranslationParser registered for the given extension
// in the default package-level registry. See Registry.GetParser for details.
func GetParser(ext string) (core.TranslationParser, error) {
	return getDefaultRegistry().GetParser(ext)
}

// RegisteredFormats returns a sorted slice of all registered file extensions
// from the default package-level registry. See Registry.RegisteredFormats for details.
func RegisteredFormats() []string {
	return getDefaultRegistry().RegisteredFormats()
}

// isExtensionChar reports whether r is valid after the leading dot (lowercase letter or digit).
func isExtensionChar(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
}

// validateExtension checks that ext matches the pattern ^\.[a-z0-9]+$
// using a rune loop. Returns an error describing the issue if invalid.
func validateExtension(ext string) error {
	if len(ext) < 2 {
		return fmt.Errorf("extension %q must be a dot followed by one or more lowercase alphanumeric characters", ext)
	}

	for i, r := range ext {
		if i == 0 {
			if r != '.' {
				return fmt.Errorf("extension %q must start with a dot", ext)
			}
			continue
		}
		if !isExtensionChar(r) {
			return fmt.Errorf("extension %q contains invalid character %q; only lowercase alphanumeric characters are allowed after the dot", ext, r)
		}
	}
	return nil
}
