// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

package engine

import (
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/0verkilll/i18n/internal/core"
	"github.com/0verkilll/i18n/internal/localedata"
)

// MaxRegisteredLocales caps the total number of locales that may be held in
// the global registry simultaneously. It exists to prevent unbounded growth
// of the in-memory locale table, which could otherwise be exploited by a
// component that calls registerLocale in a loop (or by a misconfigured
// dynamic registration path).
//
// The limit is intentionally generous — CLDR currently covers roughly 200
// locales — but finite.
const MaxRegisteredLocales = 256

// ErrRegistryFull is returned by registerLocale when the registry already
// holds MaxRegisteredLocales distinct entries and the caller is attempting
// to add a NEW locale (replacing an existing entry is always permitted).
var ErrRegistryFull = errors.New("locale registry is full (MaxRegisteredLocales reached)")

// localeRegistry holds locale translation data registered via build-tag-selected
// locale files. Each locale file calls registerLocale in its init() function to
// add its data to this map. The map is keyed by BCP 47 locale code (e.g., "en-US").
var localeRegistry = make(map[string][]byte)

// registryMu protects concurrent access to localeRegistry.
var registryMu sync.RWMutex

// registerLocale adds locale translation data to the global registry.
// It is called by each build-tag-selected locale file's init() function.
//
// The locale code is validated via core.ValidateLocale; invalid codes are
// silently skipped with a debug log message and a non-nil error is returned
// so callers that care can detect the rejection (init() callers typically
// ignore it).
//
// When the registry already holds MaxRegisteredLocales distinct codes AND
// the code being registered is NEW (not already present), the call is
// rejected with ErrRegistryFull. Replacing an existing entry always succeeds.
//
// This function never panics.
func registerLocale(code string, data []byte) error {
	if err := core.ValidateLocale(code); err != nil {
		GetLogger().Debug("skipping invalid locale registration",
			"code", code,
			"error", err)
		return fmt.Errorf("registerLocale: %w", err)
	}

	// Store a defensive copy so the caller cannot mutate registry data.
	copied := make([]byte, len(data))
	copy(copied, data)

	registryMu.Lock()
	defer registryMu.Unlock()

	if _, exists := localeRegistry[code]; !exists && len(localeRegistry) >= MaxRegisteredLocales {
		GetLogger().Debug("refusing locale registration: registry full",
			"code", code,
			"size", len(localeRegistry),
			"limit", MaxRegisteredLocales)
		return fmt.Errorf("registerLocale %q: %w (%d/%d)", code, ErrRegistryFull, len(localeRegistry), MaxRegisteredLocales)
	}

	localeRegistry[code] = copied

	GetLogger().Debug("locale registered", "code", code)
	return nil
}

// RegisteredLocales returns a sorted slice of all locale codes currently
// in the global registry. The returned slice is safe to modify; it does
// not share memory with the registry internals.
func RegisteredLocales() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()

	codes := make([]string, 0, len(localeRegistry))
	for code := range localeRegistry {
		codes = append(codes, code)
	}
	sort.Strings(codes)
	return codes
}

// bridgeLocaleData copies locale data from internal/localedata into the root
// registry. This function is called from init() to bridge build-tag-selected
// locale files that live in the internal sub-package.
func bridgeLocaleData() {
	for code, data := range localedata.All() {
		if err := registerLocale(code, data); err != nil {
			GetLogger().Debug("bridgeLocaleData: registerLocale failed",
				"code", code,
				"error", err)
		}
	}
}

func init() {
	bridgeLocaleData()
}

// resetRegistry clears all entries from the global locale registry.
// This is intended for use in tests to ensure a clean state between test cases.
func resetRegistry() {
	registryMu.Lock()
	localeRegistry = make(map[string][]byte)
	registryMu.Unlock()
}
