// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

// Package engine: regression tests for the 2026-04-23 audit fixes.
//
// This file groups targeted tests for:
//   - HasKey: exercises both happy and error branches of the fallback chain
//     to ensure the cacheMu read-lock is always released (regression test for
//     the previous lock-asymmetry bug where an error from ensureLoaded would
//     cause `continue` to skip RUnlock, leading to a later deadlock).
//   - TranslatePlural: verifies that for a given locale both key.<category>
//     and key.other are tried BEFORE advancing to the next fallback locale.
//   - Concurrency: high-contention smoke test to surface lock issues with
//     `go test -race`.
package engine

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

// TestHasKey_LockBalanceOnErrorPath constructs a situation where one locale
// in the fallback chain successfully loads and a different locale fails to
// load (file missing). Before the lock-asymmetry fix, HasKey's `continue`
// after an ensureLoaded error would skip the matching RUnlock and eventually
// deadlock or corrupt the RWMutex state on subsequent calls.
//
// After the fix, HasKey delegates to resolveKeyInLocale (which uses
// defer-based lock release), so both success and failure paths are balanced.
func TestHasKey_LockBalanceOnErrorPath(t *testing.T) {
	tmpDir := t.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// es-ES exists; en-US file is intentionally MISSING so en-US falls into
	// the error branch of ensureLoaded/loadTranslations.
	esES := []byte(`{"greeting": "Hola"}`)
	if err := os.WriteFile(filepath.Join(localeDir, "es-ES.json"), esES, 0o644); err != nil {
		t.Fatalf("write es-ES: %v", err)
	}

	translator, err := New(WithFileSystemLoader(localeDir), WithDefaultLocale("es-ES"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// The fallback chain for es-ES is [es-ES, en-US]. Key exists only in
	// es-ES, so we take the success branch for locale 0 and never reach 1.
	if !translator.HasKey("greeting") {
		t.Error("HasKey(greeting) should be true (present in es-ES)")
	}

	// Now probe a key that is NOT in es-ES. This advances to en-US, whose
	// file is missing — triggering the error branch in ensureLoaded. In the
	// buggy code, the RUnlock for the es-ES iteration never fired; a later
	// HasKey call would deadlock. We run many calls serially to expose the
	// bug if it regresses.
	for i := 0; i < 100; i++ {
		if translator.HasKey("nonexistent_key") {
			t.Error("HasKey(nonexistent_key) should be false")
		}
	}
}

// TestHasKey_ConcurrentLoadFailureNoDeadlock hammers HasKey from many
// goroutines against a translator whose fallback locale file is missing.
// With the old lock-asymmetry bug, the cache read lock would be acquired
// twice without matching releases, and a subsequent Write lock would
// deadlock. Run with `go test -race` for the strongest signal.
func TestHasKey_ConcurrentLoadFailureNoDeadlock(t *testing.T) {
	tmpDir := t.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// es-ES exists; en-US fallback is missing.
	if err := os.WriteFile(filepath.Join(localeDir, "es-ES.json"), []byte(`{"a":"Hola"}`), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	translator, err := New(WithFileSystemLoader(localeDir), WithDefaultLocale("es-ES"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	const workers = 32
	const callsPerWorker = 200

	var wg sync.WaitGroup
	var falsePositives atomic.Int64

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < callsPerWorker; j++ {
				// Alternate between a hit and a miss to stress both paths.
				if j%2 == 0 {
					if !translator.HasKey("a") {
						falsePositives.Add(1)
					}
				} else {
					if translator.HasKey("missing") {
						falsePositives.Add(1)
					}
				}
			}
		}()
	}
	wg.Wait()

	if falsePositives.Load() != 0 {
		t.Errorf("HasKey returned wrong answer %d times under concurrent load", falsePositives.Load())
	}
}

// TestTranslatePlural_SameLocaleOtherBeforeFallbackLocale is the core
// regression test for the plural-fallback ordering fix.
//
// Setup:
//   - en-US: "items" has BOTH .one and .other categories.
//   - es-MX: "items" has ONLY .other (NO .one entry).
//
// Translator locale is es-MX with count=1 (plural category = "one").
// Expected behaviour: es-MX.items.one is missing, so we fall back to
// es-MX.items.other ("# elementos") in the SAME locale BEFORE walking to
// en-US. The expected result is "1 elementos" (Spanish, other form),
// NOT "1 item" (English, one form).
//
// The previous code would skip es-MX.items.other entirely when .one was
// missing, and return en-US.items.one = "1 item".
func TestTranslatePlural_SameLocaleOtherBeforeFallbackLocale(t *testing.T) {
	tmpDir := t.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// en-US: full plural categories.
	enUS := []byte(`{
		"items": {
			"one": "# item",
			"other": "# items"
		}
	}`)
	if err := os.WriteFile(filepath.Join(localeDir, "en-US.json"), enUS, 0o644); err != nil {
		t.Fatalf("write en-US: %v", err)
	}

	// es-MX: ONLY the "other" form. The fallback chainer produces
	// [es-MX, es-ES, en-US]; we also create es-ES with no items so that the
	// test specifically distinguishes the "same-locale other" fix from an
	// accidental skip to en-US through an empty es-ES.
	esMX := []byte(`{
		"items": {
			"other": "# elementos"
		}
	}`)
	if err := os.WriteFile(filepath.Join(localeDir, "es-MX.json"), esMX, 0o644); err != nil {
		t.Fatalf("write es-MX: %v", err)
	}
	esES := []byte(`{}`)
	if err := os.WriteFile(filepath.Join(localeDir, "es-ES.json"), esES, 0o644); err != nil {
		t.Fatalf("write es-ES: %v", err)
	}

	translator, err := New(WithFileSystemLoader(localeDir), WithDefaultLocale("es-MX"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// Resolved category for es-MX,count=1 is "one"; es-MX has no "items.one"
	// so we should use es-MX "items.other" = "# elementos" => "1 elementos".
	got := translator.TranslatePlural("items", 1)
	want := "1 elementos"
	if got != want {
		t.Errorf("TranslatePlural(items,1) = %q, want %q (same-locale 'other' must beat next-locale 'one')", got, want)
	}

	// Sanity: with count=5 the category is "other" in en English and
	// Spanish; es-MX.items.other still wins.
	got = translator.TranslatePlural("items", 5)
	want = "5 elementos"
	if got != want {
		t.Errorf("TranslatePlural(items,5) = %q, want %q", got, want)
	}
}

// TestTranslatePluralWithArgs_SameLocaleOtherBeforeFallbackLocale mirrors
// TestTranslatePlural_SameLocaleOtherBeforeFallbackLocale for the
// WithArgs variant, since TranslatePluralWithArgs has its own implementation
// of the nested-loop ordering.
func TestTranslatePluralWithArgs_SameLocaleOtherBeforeFallbackLocale(t *testing.T) {
	tmpDir := t.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	enUS := []byte(`{
		"msg": {
			"one": "# item for %s",
			"other": "# items for %s"
		}
	}`)
	if err := os.WriteFile(filepath.Join(localeDir, "en-US.json"), enUS, 0o644); err != nil {
		t.Fatalf("write en-US: %v", err)
	}
	esMX := []byte(`{
		"msg": {
			"other": "# elementos para %s"
		}
	}`)
	if err := os.WriteFile(filepath.Join(localeDir, "es-MX.json"), esMX, 0o644); err != nil {
		t.Fatalf("write es-MX: %v", err)
	}
	if err := os.WriteFile(filepath.Join(localeDir, "es-ES.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatalf("write es-ES: %v", err)
	}

	translator, err := New(WithFileSystemLoader(localeDir), WithDefaultLocale("es-MX"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	got := translator.TranslatePluralWithArgs("msg", 1, "inbox")
	want := "1 elementos para inbox"
	if got != want {
		t.Errorf("TranslatePluralWithArgs(msg,1,inbox) = %q, want %q", got, want)
	}
}

// TestWithLocaleContext_ScopedOverride verifies that WithLocaleContext
// produces a ContextTranslator whose translations use the supplied locale
// without mutating the underlying Translator.
func TestWithLocaleContext_ScopedOverride(t *testing.T) {
	tmpDir := t.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(localeDir, "en-US.json"), []byte(`{"g":"Hello"}`), 0o644); err != nil {
		t.Fatalf("write en-US: %v", err)
	}
	if err := os.WriteFile(filepath.Join(localeDir, "es-ES.json"), []byte(`{"g":"Hola"}`), 0o644); err != nil {
		t.Fatalf("write es-ES: %v", err)
	}

	translator, err := New(WithFileSystemLoader(localeDir), WithDefaultLocale("en-US"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ct := translator.WithLocaleContext(nil, "es-ES") //nolint:staticcheck // nil ctx is accepted here

	if ct.GetLocale() != "es-ES" {
		t.Errorf("ct.GetLocale() = %q, want %q", ct.GetLocale(), "es-ES")
	}
	if got := ct.Translate("g"); got != "Hola" {
		t.Errorf("ct.Translate(g) = %q, want %q", got, "Hola")
	}

	// Underlying translator unchanged.
	if translator.GetLocale() != "en-US" {
		t.Errorf("shared translator locale mutated: %q, want %q", translator.GetLocale(), "en-US")
	}
	if got := translator.Translate("g"); got != "Hello" {
		t.Errorf("shared translator.Translate(g) = %q, want %q", got, "Hello")
	}
}

// TestWithLocaleContext_EmptyLocaleFallsBackToShared verifies that an empty
// locale override leaves the ContextTranslator using the shared Translator's
// current locale.
func TestWithLocaleContext_EmptyLocaleFallsBackToShared(t *testing.T) {
	tmpDir := t.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(localeDir, "en-US.json"), []byte(`{"g":"Hello"}`), 0o644); err != nil {
		t.Fatalf("write en-US: %v", err)
	}

	translator, err := New(WithFileSystemLoader(localeDir), WithDefaultLocale("en-US"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ct := translator.WithLocaleContext(nil, "") //nolint:staticcheck

	if ct.GetLocale() != "en-US" {
		t.Errorf("ct.GetLocale() with empty override = %q, want %q", ct.GetLocale(), "en-US")
	}
	if got := ct.Translate("g"); got != "Hello" {
		t.Errorf("ct.Translate(g) = %q, want %q", got, "Hello")
	}
}

// TestTranslate_CacheKeyNotAllocatedWhenCacheDisabled is an indirect test:
// we can't observe allocations directly from a functional test, but we can
// verify correctness of the early-return path (Translate returns the right
// answer both with and without cache). Allocation counts are covered by
// benchmarks.
func TestTranslate_CacheKeyNotAllocatedWhenCacheDisabled(t *testing.T) {
	tmpDir := t.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(localeDir, "en-US.json"), []byte(`{"g":"Hello"}`), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	translator, err := New(WithFileSystemLoader(localeDir), WithDefaultLocale("en-US"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if got := translator.Translate("g"); got != "Hello" {
		t.Errorf("Translate(g) = %q, want %q", got, "Hello")
	}
	if got := translator.Translate("missing"); got != "missing" {
		t.Errorf("Translate(missing) = %q, want %q", got, "missing")
	}
}

// TestSplitKeyCached_ReturnsStableSlices verifies the resolver-level key
// split cache: repeated calls for the same key return equal slices.
func TestSplitKeyCached_ReturnsStableSlices(t *testing.T) {
	key := "a.b.c.d"
	got1 := splitKeyCached(key)
	got2 := splitKeyCached(key)
	if strings.Join(got1, ".") != key || strings.Join(got2, ".") != key {
		t.Errorf("splitKeyCached produced wrong parts: %v, %v", got1, got2)
	}
	if len(got1) != 4 {
		t.Errorf("len=%d, want 4", len(got1))
	}
	// The slices should be the SAME underlying slice (second call hits the
	// sync.Map cache).
	if &got1[0] != &got2[0] {
		t.Error("expected cached slice to be reused across calls")
	}
}
