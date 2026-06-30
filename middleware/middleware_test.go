// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/0verkilll/i18n"
)

func setupTranslator(t *testing.T) *i18n.Translator {
	t.Helper()

	tmpDir := t.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}

	enContent := []byte(`{"greeting": "Hello"}`)
	esContent := []byte(`{"greeting": "Hola"}`)

	if err := os.WriteFile(filepath.Join(localeDir, "en-US.json"), enContent, 0o644); err != nil {
		t.Fatalf("failed to create en-US file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(localeDir, "es-ES.json"), esContent, 0o644); err != nil {
		t.Fatalf("failed to create es-ES file: %v", err)
	}

	translator, err := i18n.New(
		i18n.WithFileSystemLoader(localeDir),
		i18n.WithDefaultLocale("en-US"),
	)
	if err != nil {
		t.Fatalf("failed to create translator: %v", err)
	}

	return translator
}

func TestLocaleFromRequest_SetsLocaleFromAcceptLanguage(t *testing.T) {
	translator := setupTranslator(t)

	var capturedTranslator *i18n.ContextTranslator
	handler := LocaleFromRequest(translator)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		capturedTranslator = TranslatorFromContext(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Language", "es-ES,en-US;q=0.9")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if capturedTranslator == nil {
		t.Fatal("expected ContextTranslator in request context, got nil")
	}

	locale := capturedTranslator.GetLocale()
	if locale != "es-ES" {
		t.Errorf("GetLocale() = %q, want %q", locale, "es-ES")
	}
}

func TestLocaleFromRequest_PreservesLocaleWhenNoHeader(t *testing.T) {
	translator := setupTranslator(t)

	var capturedTranslator *i18n.ContextTranslator
	handler := LocaleFromRequest(translator)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		capturedTranslator = TranslatorFromContext(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if capturedTranslator == nil {
		t.Fatal("expected ContextTranslator in request context, got nil")
	}

	locale := capturedTranslator.GetLocale()
	if locale != "en-US" {
		t.Errorf("GetLocale() = %q, want %q", locale, "en-US")
	}
}

func TestTranslatorFromContext_ReturnsNilWhenNotSet(t *testing.T) {
	ctx := context.Background()
	ct := TranslatorFromContext(ctx)
	if ct != nil {
		t.Errorf("expected nil, got %v", ct)
	}
}

func TestLocaleFromRequest_TranslationWorks(t *testing.T) {
	translator := setupTranslator(t)

	var greeting string
	handler := LocaleFromRequest(translator)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		ct := TranslatorFromContext(r.Context())
		if ct != nil {
			greeting = ct.Translate("greeting")
		}
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Language", "es-ES")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if greeting != "Hola" {
		t.Errorf("Translate(greeting) = %q, want %q", greeting, "Hola")
	}
}

// TestLocaleFromRequest_DoesNotMutateSharedTranslator is a regression test
// for a race where the middleware called t.SetLocale on a shared Translator,
// which would overwrite the shared locale state between concurrent requests
// and produce the wrong language for unrelated handlers.
//
// With the context-scoped fix the shared Translator's locale must remain
// whatever it was at construction time, regardless of how many requests with
// how many languages pass through the middleware.
func TestLocaleFromRequest_DoesNotMutateSharedTranslator(t *testing.T) {
	translator := setupTranslator(t)

	handler := LocaleFromRequest(translator)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		ct := TranslatorFromContext(r.Context())
		if ct == nil {
			t.Error("expected ContextTranslator in request context, got nil")
		}
	}))

	for _, header := range []string{"es-ES", "de-DE", "fr-FR", "ja-JP"} {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Accept-Language", header)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}

	// Shared translator locale must still be the one it was constructed with.
	if got := translator.GetLocale(); got != "en-US" {
		t.Errorf("shared Translator.GetLocale() mutated: got %q, want %q", got, "en-US")
	}
}

// TestLocaleFromRequest_ConcurrentRequestsDoNotRace is the core correctness
// test for the middleware race fix. Many concurrent requests, each with a
// different Accept-Language header, must each see their OWN translated
// greeting inside the handler. Before the fix, one request could clobber
// another's locale via t.SetLocale and receive the wrong language.
//
// Run with `go test -race` to also catch any residual data races.
func TestLocaleFromRequest_ConcurrentRequestsDoNotRace(t *testing.T) {
	translator := setupTranslator(t)

	cases := []struct {
		accept   string
		expected string
	}{
		{"en-US", "Hello"},
		{"es-ES", "Hola"},
	}

	handler := LocaleFromRequest(translator)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ct := TranslatorFromContext(r.Context())
		if ct == nil {
			http.Error(w, "no translator", http.StatusInternalServerError)
			return
		}
		if _, err := w.Write([]byte(ct.Translate("greeting"))); err != nil {
			t.Errorf("write failed: %v", err)
		}
	}))

	const iterationsPerCase = 200
	var wg sync.WaitGroup
	var mismatches atomic.Int64

	for _, tc := range cases {
		for i := 0; i < iterationsPerCase; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				req := httptest.NewRequest(http.MethodGet, "/", nil)
				req.Header.Set("Accept-Language", tc.accept)
				rec := httptest.NewRecorder()
				handler.ServeHTTP(rec, req)
				if rec.Body.String() != tc.expected {
					mismatches.Add(1)
				}
			}()
		}
	}

	wg.Wait()

	if mismatches.Load() != 0 {
		t.Errorf("concurrent requests produced %d wrong translations (expected 0)", mismatches.Load())
	}

	// Shared translator must not have been mutated.
	if got := translator.GetLocale(); got != "en-US" {
		t.Errorf("shared Translator.GetLocale() mutated after concurrent traffic: got %q, want %q", got, "en-US")
	}
}

// TestLocaleFromContext_ReturnsDetectedLocale verifies the LocaleFromContext
// helper returns the detected locale string when the middleware ran.
func TestLocaleFromContext_ReturnsDetectedLocale(t *testing.T) {
	translator := setupTranslator(t)

	var got string
	handler := LocaleFromRequest(translator)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got = LocaleFromContext(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Language", "es-ES")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got != "es-ES" {
		t.Errorf("LocaleFromContext = %q, want %q", got, "es-ES")
	}
}

// TestLocaleFromContext_EmptyWhenNoHeader verifies LocaleFromContext returns
// an empty string when Accept-Language was absent.
func TestLocaleFromContext_EmptyWhenNoHeader(t *testing.T) {
	translator := setupTranslator(t)

	var got string
	var gotTrans *i18n.ContextTranslator
	handler := LocaleFromRequest(translator)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got = LocaleFromContext(r.Context())
		gotTrans = TranslatorFromContext(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got != "" {
		t.Errorf("LocaleFromContext with no header = %q, want %q", got, "")
	}
	// The translator is still present; it just tracks the shared default.
	if gotTrans == nil {
		t.Fatal("expected non-nil ContextTranslator")
	}
	if gotTrans.GetLocale() != "en-US" {
		t.Errorf("ContextTranslator.GetLocale() = %q, want %q", gotTrans.GetLocale(), "en-US")
	}
}

// TestTranslatorFromRequest_Convenience verifies the request-accepting helper.
func TestTranslatorFromRequest_Convenience(t *testing.T) {
	translator := setupTranslator(t)

	var got *i18n.ContextTranslator
	handler := LocaleFromRequest(translator)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got = TranslatorFromRequest(r)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Language", "es-ES")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got == nil {
		t.Fatal("TranslatorFromRequest returned nil for a request with middleware applied")
	}
	if got.GetLocale() != "es-ES" {
		t.Errorf("TranslatorFromRequest.GetLocale() = %q, want %q", got.GetLocale(), "es-ES")
	}

	// nil request yields nil translator.
	if TranslatorFromRequest(nil) != nil {
		t.Error("TranslatorFromRequest(nil) should return nil")
	}
}
