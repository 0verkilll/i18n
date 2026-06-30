// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

//go:build js && wasm

package engine

import (
	"syscall/js"

	"github.com/0verkilll/i18n/internal/core"
)

// Compile-time interface assertion.
var _ core.LocaleDetector = (*BrowserDetector)(nil)

// BrowserDetector is a core.LocaleDetector that reads the user's preferred
// language from the browser via navigator.language / navigator.languages.
type BrowserDetector struct{}

// NewBrowserDetector creates a new BrowserDetector that reads locale
// information from the browser's navigator object via syscall/js.
//
// Example usage:
//
//	chain := engine.NewChainDetector(
//	    engine.NewBrowserDetector(),
//	    engine.NewDefaultLocaleDetector(nil),
//	)
func NewBrowserDetector() *BrowserDetector {
	return &BrowserDetector{}
}

// Detect reads navigator.language from the browser. If empty or undefined,
// it iterates navigator.languages and returns the first non-empty value.
// Returns "" if navigator is unavailable.
func (d *BrowserDetector) Detect() string {
	navigator := js.Global().Get("navigator")
	if navigator.IsUndefined() || navigator.IsNull() {
		return ""
	}

	// Try navigator.language first
	lang := navigator.Get("language")
	if !lang.IsUndefined() && !lang.IsNull() {
		if s := lang.String(); s != "" {
			return s
		}
	}

	// Fall back to navigator.languages array
	languages := navigator.Get("languages")
	if languages.IsUndefined() || languages.IsNull() {
		return ""
	}

	length := languages.Length()
	for i := 0; i < length; i++ {
		entry := languages.Index(i)
		if !entry.IsUndefined() && !entry.IsNull() {
			if s := entry.String(); s != "" {
				return s
			}
		}
	}

	return ""
}

// Normalize converts a locale string to BCP 47 format using the shared
// normalization logic.
func (d *BrowserDetector) Normalize(locale string) string {
	return NormalizeLocale(locale)
}
