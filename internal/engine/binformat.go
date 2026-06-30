// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

package engine

import (
	"fmt"
	"strings"

	"github.com/0verkilll/i18n/internal/core"
)

// readUint16BE reads a big-endian uint16 from two bytes.
func readUint16BE(b []byte) int {
	return int(b[0])<<8 | int(b[1])
}

// appendUint16BE appends a big-endian uint16 to a byte slice.
func appendUint16BE(buf []byte, v int) []byte {
	return append(buf, byte(v>>8), byte(v)) //nolint:gosec // v validated <= 65535
}

// Binary format constants.
const (
	// binMagic0 is the first magic byte ('i').
	binMagic0 = 0x69
	// binMagic1 is the second magic byte ('1').
	binMagic1 = 0x31
	// binVersion is the current binary format version.
	binVersion = 1
	// binHeaderSize is the total header size in bytes: magic(2) + version(1) + count(2).
	binHeaderSize = 5
	// binMaxKeyLen is the maximum key length (fits in 1 byte).
	binMaxKeyLen = 255
	// binMaxValLen is the maximum value length (fits in 2 bytes big-endian).
	binMaxValLen = 65535
	// binMaxEntries is the maximum entry count (fits in 2 bytes big-endian).
	binMaxEntries = 65535
)

// Compile-time assertion that BinaryParser implements core.TranslationParser.
var _ core.TranslationParser = (*BinaryParser)(nil)

// BinaryParser parses translations from the compact binary format.
// The binary format eliminates JSON syntax overhead (quotes, braces, colons)
// saving approximately 40% on translation data size. The parser implementation
// is minimal compared to the full JSON parser, reducing compiled binary size.
//
// Binary format specification (version 1):
//
//	Header:  [0x69][0x31][version:1][entry_count:2 big-endian]
//	Entries: [key_len:1][key:N][val_len:2 big-endian][value:N]...
//
// Keys are stored in pre-flattened dot notation (e.g., "error.validation.required").
// Values are always UTF-8 strings. The parser reconstructs the nested map
// structure expected by the KeyResolver.
type BinaryParser struct{}

// NewBinaryParser creates a new BinaryParser for the compact binary translation format.
func NewBinaryParser() *BinaryParser {
	return &BinaryParser{}
}

// Parse decodes binary translation data into a nested map compatible with the
// KeyResolver interface. It validates magic bytes, version, and enforces
// core.MaxKeyCount. Keys in dot notation are unflattened into nested maps.
func (p *BinaryParser) Parse(data []byte) (map[string]interface{}, error) {
	flat, err := parseBinaryFlat(data)
	if err != nil {
		return nil, err
	}
	return unflattenKeys(flat), nil
}

// parseBinaryFlat decodes binary data into a flat string-to-string map.
func parseBinaryFlat(data []byte) (map[string]string, error) {
	if err := validateBinaryHeader(data); err != nil {
		return nil, err
	}

	count := readUint16BE(data[3:5])
	if count > core.MaxKeyCount {
		return nil, core.NewErrInvalidFormat("binary",
			fmt.Errorf("entry count %d exceeds maximum of %d", count, core.MaxKeyCount))
	}

	d := binDecoder{buf: data[binHeaderSize:]}
	return d.decodeAll(count)
}

// validateBinaryHeader checks magic bytes, minimum size, and version.
func validateBinaryHeader(data []byte) error {
	if len(data) < binHeaderSize {
		return core.NewErrInvalidFormat("binary",
			fmt.Errorf("input too short: need at least %d bytes, got %d", binHeaderSize, len(data)))
	}
	if data[0] != binMagic0 || data[1] != binMagic1 {
		return core.NewErrInvalidFormat("binary",
			fmt.Errorf("invalid magic bytes: expected 0x%02x 0x%02x, got 0x%02x 0x%02x",
				binMagic0, binMagic1, data[0], data[1]))
	}
	if data[2] != binVersion {
		return core.NewErrInvalidFormat("binary",
			fmt.Errorf("unsupported version %d, expected %d", data[2], binVersion))
	}
	return nil
}

// binDecoder reads key-value entries from a binary translation buffer.
type binDecoder struct {
	buf []byte
	pos int
}

// decodeAll reads count key-value pairs and returns them as a flat map.
func (d *binDecoder) decodeAll(count int) (map[string]string, error) {
	result := make(map[string]string, count)
	for i := 0; i < count; i++ {
		key, err := d.readKey(i)
		if err != nil {
			return nil, err
		}
		val, err := d.readValue(i)
		if err != nil {
			return nil, err
		}
		result[key] = val
	}
	return result, nil
}

// readKey reads the key length (1 byte) and key bytes from the buffer.
func (d *binDecoder) readKey(index int) (string, error) {
	if d.pos >= len(d.buf) {
		return "", core.NewErrInvalidFormat("binary",
			fmt.Errorf("truncated data: missing key length at entry %d", index))
	}
	keyLen := int(d.buf[d.pos])
	d.pos++

	if d.pos+keyLen > len(d.buf) {
		return "", core.NewErrInvalidFormat("binary",
			fmt.Errorf("truncated data: missing key bytes at entry %d", index))
	}
	key := string(d.buf[d.pos : d.pos+keyLen])
	d.pos += keyLen
	return key, nil
}

// readValue reads the value length (2 bytes big-endian) and value bytes.
func (d *binDecoder) readValue(index int) (string, error) {
	if d.pos+2 > len(d.buf) {
		return "", core.NewErrInvalidFormat("binary",
			fmt.Errorf("truncated data: missing value length at entry %d", index))
	}
	valLen := readUint16BE(d.buf[d.pos : d.pos+2])
	d.pos += 2

	if d.pos+valLen > len(d.buf) {
		return "", core.NewErrInvalidFormat("binary",
			fmt.Errorf("truncated data: missing value bytes at entry %d", index))
	}
	val := string(d.buf[d.pos : d.pos+valLen])
	d.pos += valLen
	return val, nil
}

// unflattenKeys converts a flat dot-notation map to a nested map structure
// compatible with the DefaultKeyResolver.
// For example, {"error.required": "..."} becomes {"error": {"required": "..."}}.
func unflattenKeys(flat map[string]string) map[string]interface{} {
	result := make(map[string]interface{}, len(flat))
	for key, val := range flat {
		setNested(result, key, val)
	}
	return result
}

// setNested inserts a value into a nested map at the path specified by a
// dot-separated key. Intermediate maps are created as needed.
func setNested(root map[string]interface{}, key, val string) {
	parts := strings.Split(key, ".")
	current := root

	for _, part := range parts[:len(parts)-1] {
		next, ok := current[part]
		if !ok {
			m := make(map[string]interface{})
			current[part] = m
			current = m
			continue
		}
		m, ok := next.(map[string]interface{})
		if !ok {
			// Key collision: a leaf value exists where we need a map.
			// Overwrite with a new map (last write wins).
			m = make(map[string]interface{})
			current[part] = m
		}
		current = m
	}

	current[parts[len(parts)-1]] = val
}

// =============================================================================
// Binary encoder (build-time / tooling)
// =============================================================================

// EncodeBinary converts a flat map of translations to the compact binary format.
// Keys must be in dot notation and no longer than 255 bytes. Values must be no
// longer than 65535 bytes. The entry count must not exceed 65535.
func EncodeBinary(translations map[string]string) ([]byte, error) {
	if len(translations) > binMaxEntries {
		return nil, fmt.Errorf("entry count %d exceeds maximum of %d",
			len(translations), binMaxEntries)
	}

	keys := sortedKeys(translations)
	size := estimateBinarySize(translations, keys)

	buf := make([]byte, 0, size)
	buf = appendBinaryHeader(buf, len(translations))

	var err error
	for _, key := range keys {
		buf, err = appendBinaryEntry(buf, key, translations[key])
		if err != nil {
			return nil, err
		}
	}

	return buf, nil
}

// estimateBinarySize calculates the approximate buffer size needed.
func estimateBinarySize(translations map[string]string, keys []string) int {
	size := binHeaderSize
	for _, k := range keys {
		// 1 (key_len) + len(key) + 2 (val_len) + len(val)
		size += 1 + len(k) + 2 + len(translations[k])
	}
	return size
}

// appendBinaryHeader writes the 5-byte header to buf.
func appendBinaryHeader(buf []byte, count int) []byte {
	buf = append(buf, binMagic0, binMagic1, binVersion)
	buf = appendUint16BE(buf, count)
	return buf
}

// appendBinaryEntry writes a single key-value entry to buf.
func appendBinaryEntry(buf []byte, key, val string) ([]byte, error) {
	if len(key) > binMaxKeyLen {
		return nil, fmt.Errorf("key %q exceeds maximum length of %d bytes",
			key, binMaxKeyLen)
	}
	if len(val) > binMaxValLen {
		return nil, fmt.Errorf("value for key %q exceeds maximum length of %d bytes",
			key, binMaxValLen)
	}

	buf = append(buf, byte(len(key))) //nolint:gosec // len(key) validated <= 255
	buf = append(buf, key...)
	buf = appendUint16BE(buf, len(val))
	buf = append(buf, val...)
	return buf, nil
}

// =============================================================================
// Key flattening helper
// =============================================================================

// FlattenKeys converts a nested map to a flat dot-notation map.
// For example, {"error": {"required": "..."}} becomes {"error.required": "..."}.
// Non-string leaf values are converted using fmt.Sprint.
func FlattenKeys(nested map[string]interface{}) map[string]string {
	result := make(map[string]string)
	flattenRecursive(nested, "", result)
	return result
}

// flattenRecursive walks the nested map and collects leaf values.
func flattenRecursive(m map[string]interface{}, prefix string, result map[string]string) {
	for key, val := range m {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}

		sub, ok := val.(map[string]interface{})
		if ok {
			flattenRecursive(sub, fullKey, result)
			continue
		}

		switch v := val.(type) {
		case string:
			result[fullKey] = v
		default:
			result[fullKey] = fmt.Sprint(v)
		}
	}
}

// sortedKeys returns the keys of a map in sorted order for deterministic output.
func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sortStrings(keys)
	return keys
}

// sortStrings sorts a string slice in ascending order using insertion sort.
// This avoids importing "sort" in binformat.go and keeps the dependency
// footprint minimal for WASM/TinyGo builds.
func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		key := s[i]
		j := i - 1
		for j >= 0 && s[j] > key {
			s[j+1] = s[j]
			j--
		}
		s[j+1] = key
	}
}
