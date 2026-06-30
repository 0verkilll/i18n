// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

// Package middleware provides net/http middleware for locale detection and
// translator injection. It reads the Accept-Language header from incoming
// requests, derives a per-request locale-scoped translator, and stores it in
// the request context for downstream handlers.
//
// This package is separated from the core i18n library so that users who
// do not need HTTP middleware do not pay the cost of importing net/http.
// Import it as:
//
//	import "github.com/0verkilll/i18n/middleware"
package middleware

import (
	"context"
	"net/http"

	"github.com/0verkilll/i18n"
)

// translatorKey is an unexported type used as the key for storing a
// ContextTranslator in request context. Using a private type prevents
// collisions with keys defined in other packages.
type translatorKey struct{}

// localeKey is an unexported type used as the key for storing the
// per-request detected locale string in request context. It is available to
// downstream code that wants to read the raw detected locale directly via
// LocaleFromContext, bypassing the ContextTranslator wrapper.
type localeKey struct{}

// LocaleFromRequest returns an http.Handler middleware that detects the
// locale from the Accept-Language header and stores a locale-scoped
// ContextTranslator in the request context.
//
// CONCURRENCY: this middleware does NOT mutate the shared Translator. Each
// request gets its own lightweight ContextTranslator carrying its detected
// locale, via i18n.Translator.WithLocaleContext. This makes it safe to use
// a single shared Translator across many concurrent HTTP requests with
// different Accept-Language headers — the previous behaviour of calling
// t.SetLocale raced between requests and could serve the wrong language to
// an unrelated request.
//
// Downstream handlers retrieve the translator with TranslatorFromContext
// (or read the raw locale with LocaleFromContext).
//
// If the Accept-Language header is empty or unparseable, the translator
// falls back to the shared Translator's current locale.
//
// Usage:
//
//	translator, _ := i18n.New(
//	    i18n.WithFileSystemLoader("locales"),
//	    i18n.WithDefaultLocale("en-US"),
//	)
//	mux := http.NewServeMux()
//	mux.Handle("/", middleware.LocaleFromRequest(translator)(myHandler))
func LocaleFromRequest(t *i18n.Translator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			detected := ""
			if acceptLang := r.Header.Get("Accept-Language"); acceptLang != "" {
				detector := i18n.NewAcceptLanguageDetector(acceptLang)
				detected = detector.Detect()
			}

			// Build a per-request locale-scoped ContextTranslator. When
			// detected is empty, WithLocaleContext leaves the locale
			// override unset and the ContextTranslator transparently uses
			// the shared Translator's current locale.
			ct := t.WithLocaleContext(r.Context(), detected)

			ctx := context.WithValue(r.Context(), translatorKey{}, ct)
			if detected != "" {
				ctx = context.WithValue(ctx, localeKey{}, ct.GetLocale())
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// TranslatorFromContext retrieves the ContextTranslator stored in the request
// context by LocaleFromRequest. Returns nil if no translator is present.
//
// The returned ContextTranslator is scoped to the request's detected locale;
// it is safe to use concurrently with other requests that may have detected
// different locales.
func TranslatorFromContext(ctx context.Context) *i18n.ContextTranslator {
	ct, ok := ctx.Value(translatorKey{}).(*i18n.ContextTranslator)
	if !ok {
		return nil
	}
	return ct
}

// TranslatorFromRequest is a convenience wrapper around TranslatorFromContext
// that accepts an *http.Request directly. Returns nil if no translator is
// present in the request context.
func TranslatorFromRequest(r *http.Request) *i18n.ContextTranslator {
	if r == nil {
		return nil
	}
	return TranslatorFromContext(r.Context())
}

// LocaleFromContext returns the raw detected locale string stored in the
// request context by LocaleFromRequest. Returns an empty string if no locale
// was detected (e.g., the request had no Accept-Language header) or if the
// middleware did not run on this context.
func LocaleFromContext(ctx context.Context) string {
	s, ok := ctx.Value(localeKey{}).(string)
	if !ok {
		return ""
	}
	return s
}
