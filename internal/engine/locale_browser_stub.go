// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

//go:build !(js && wasm)

package engine

import "github.com/0verkilll/i18n/internal/core"

// Compile-time interface assertion.
var _ core.LocaleDetector = (*BrowserDetector)(nil)

// BrowserDetector is a core.LocaleDetector that reads the user's preferred
// language from the browser via navigator.language / navigator.languages.
//
// On non-WASM builds Detect always returns "". The real implementation is in
// locale_browser_js.go and is compiled only when targeting js/wasm.
type BrowserDetector struct{}

// NewBrowserDetector creates a new BrowserDetector. On non-WASM builds,
// Detect always returns "". Use this in a ChainDetector so that a fallback
// detector provides the locale when not running in a browser.
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

// Detect returns "" on non-WASM builds.
func (d *BrowserDetector) Detect() string {
	return ""
}

// Normalize converts a locale string to BCP 47 format using the shared
// normalization logic.
func (d *BrowserDetector) Normalize(locale string) string {
	return NormalizeLocale(locale)
}
