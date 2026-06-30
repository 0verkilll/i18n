// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

// Package localedata holds build-tag-selected example locale data.
// These translations are example entries for testing and demonstrating
// the build-tag locale selection mechanism.
//
// Production applications should provide their own translation files
// via FileSystemLoader, EmbedFSLoader, or a custom TranslationLoader
// implementation. The built-in locale data contains a set of common keys
// (greeting, farewell, welcome, error messages, plural forms) to verify
// the library works correctly.
package localedata

import "sync"

// entries holds locale data registered by build-tag-selected files.
var entries = make(map[string][]byte)

// mu protects concurrent access to entries during init().
var mu sync.Mutex

// Register adds locale translation data to the internal registry.
// Called by each build-tag-selected locale file's init() function.
func Register(code string, data []byte) {
	mu.Lock()
	entries[code] = data
	mu.Unlock()
}

// All returns a snapshot of all registered locale data.
// The returned map is a shallow copy; callers may iterate it safely.
func All() map[string][]byte {
	mu.Lock()
	defer mu.Unlock()

	result := make(map[string][]byte, len(entries))
	for k, v := range entries {
		result[k] = v
	}
	return result
}
