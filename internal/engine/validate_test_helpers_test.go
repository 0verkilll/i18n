// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

package engine

import "strings"

// matchLocale validates a locale string against BCP 47 format.
// This is a test-only replica of core.matchLocale (unexported).
// Accepts: 2-3 lowercase a-z, optionally followed by '-' and exactly 2 uppercase A-Z.
func matchLocale(s string) bool {
	n := len(s)
	if n < 2 || n > 6 {
		return false
	}

	i := 0
	for i < n && s[i] >= 'a' && s[i] <= 'z' {
		i++
	}
	langLen := i
	if langLen < 2 || langLen > 3 {
		return false
	}
	if langLen == n {
		return true
	}
	// Region part
	if i >= n || s[i] != '-' {
		return false
	}
	i++
	regionStart := i
	for i < n && s[i] >= 'A' && s[i] <= 'Z' {
		i++
	}
	if i-regionStart != 2 || i != n {
		return false
	}
	return true
}

// matchKey validates that every byte in the string is in the allowed set [a-zA-Z0-9._-].
// This is a test-only replica of core.matchKey (unexported).
func matchKey(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !isKeyCharHelper(c) {
			return false
		}
	}
	return true
}

func isKeyCharHelper(c byte) bool {
	return (c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') ||
		c == '.' || c == '_' || c == '-'
}

// countFormatSpecifiers counts format specifiers in a string, skipping %% pairs.
// This is a test-only replica of core.countFormatSpecifiers (unexported).
func countFormatSpecifiers(s string) int {
	count := 0
	i := 0
	n := len(s)
	for i < n {
		if s[i] == '%' {
			if i+1 < n {
				if s[i+1] == '%' {
					i += 2
					continue
				}
				count++
				i += 2
				continue
			}
			i++
			continue
		}
		i++
	}
	return count
}

// stripANSI removes ANSI escape sequences from a string.
// This is a test-only replica of core.stripANSI (unexported).
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
		if i+1 < n && s[i+1] == '[' {
			i += 2
			for i < n && ((s[i] >= '0' && s[i] <= '9') || s[i] == ';') {
				i++
			}
			if i < n && ((s[i] >= 'a' && s[i] <= 'z') || (s[i] >= 'A' && s[i] <= 'Z')) {
				i++
			}
			continue
		}
		i++
	}

	return result.String()
}
