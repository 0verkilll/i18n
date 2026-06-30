// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/0verkilll/i18n/internal/core"
)

// TestNewMapCacheGetMissAndSetHit verifies that a new cache returns a miss for
// an unknown key and returns a hit after the key is set.
func TestNewMapCacheGetMissAndSetHit(t *testing.T) {
	c := NewMapCache()

	_, ok := c.Get("missing")
	if ok {
		t.Error("Get on empty cache should return false")
	}

	c.Set("greeting", "Hello")
	val, ok := c.Get("greeting")
	if !ok {
		t.Fatal("Get after Set should return true")
	}
	if val != "Hello" {
		t.Errorf("Get = %q, want %q", val, "Hello")
	}
}

// TestMapCacheInvalidate verifies that Invalidate clears all cached entries.
func TestMapCacheInvalidate(t *testing.T) {
	c := NewMapCache()
	c.Set("a", "1")
	c.Set("b", "2")

	c.Invalidate()

	if _, ok := c.Get("a"); ok {
		t.Error("Get(a) should miss after Invalidate")
	}
	if _, ok := c.Get("b"); ok {
		t.Error("Get(b) should miss after Invalidate")
	}
}

// TestMapCacheWithLimitEvictsLRU verifies that when the cache exceeds maxEntries,
// the least-recently-used entry is evicted.
func TestMapCacheWithLimitEvictsLRU(t *testing.T) {
	c := NewMapCacheWithLimit(2)

	c.Set("a", "1")
	c.Set("b", "2")
	// Cache is at capacity: [b, a] (head to tail)

	c.Set("c", "3")
	// "a" should be evicted as the LRU entry: [c, b]

	if _, ok := c.Get("a"); ok {
		t.Error("entry 'a' should have been evicted")
	}

	val, ok := c.Get("b")
	if !ok || val != "2" {
		t.Errorf("entry 'b' should still exist, got ok=%v val=%q", ok, val)
	}

	val, ok = c.Get("c")
	if !ok || val != "3" {
		t.Errorf("entry 'c' should still exist, got ok=%v val=%q", ok, val)
	}
}

// TestMapCacheLRUPromotion verifies that accessing an entry via Get promotes it,
// preventing it from being evicted.
func TestMapCacheLRUPromotion(t *testing.T) {
	c := NewMapCacheWithLimit(2)

	c.Set("a", "1")
	c.Set("b", "2")
	// Order: [b, a]

	// Access "a" to promote it to the head
	c.Get("a")
	// Order: [a, b]

	// Insert "c", which should evict "b" (now the tail)
	c.Set("c", "3")

	if _, ok := c.Get("b"); ok {
		t.Error("entry 'b' should have been evicted after 'a' was promoted")
	}

	if val, ok := c.Get("a"); !ok || val != "1" {
		t.Errorf("entry 'a' should still exist after promotion, got ok=%v val=%q", ok, val)
	}
}

// TestMapCacheWithLimitZeroBehavesUnlimited verifies that NewMapCacheWithLimit(0)
// disables eviction, behaving like an unlimited cache.
func TestMapCacheWithLimitZeroBehavesUnlimited(t *testing.T) {
	c := NewMapCacheWithLimit(0)

	for i := 0; i < 100; i++ {
		c.Set(fmt.Sprintf("key-%d", i), fmt.Sprintf("val-%d", i))
	}

	for i := 0; i < 100; i++ {
		val, ok := c.Get(fmt.Sprintf("key-%d", i))
		if !ok {
			t.Errorf("key-%d should exist in unlimited mode", i)
		}
		if val != fmt.Sprintf("val-%d", i) {
			t.Errorf("key-%d = %q, want %q", i, val, fmt.Sprintf("val-%d", i))
		}
	}
}

// TestMapCacheConcurrency exercises MapCache under concurrent reads and writes
// to verify thread safety with the -race detector.
func TestMapCacheConcurrency(t *testing.T) {
	c := NewMapCacheWithLimit(50)

	var wg sync.WaitGroup
	numWorkers := 20
	opsPerWorker := 200

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < opsPerWorker; i++ {
				key := fmt.Sprintf("key-%d-%d", id, i%10)
				c.Set(key, fmt.Sprintf("val-%d", i))
				c.Get(key)
				if i%50 == 0 {
					c.Invalidate()
				}
			}
		}(w)
	}

	wg.Wait()
}

// ---------------------------------------------------------------------------
// Gap analysis tests (Task Group 4)
// ---------------------------------------------------------------------------

// TestCacheKeyFormatDistinctness verifies that Translate, TranslatePlural, and
// TranslateGender produce distinct cache entries that do not collide.
func TestCacheKeyFormatDistinctness(t *testing.T) {
	tmpDir := t.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	enUS := []byte(`{
		"greeting": "Hello",
		"items": {"one": "# item", "other": "# items"},
		"welcome": {"masculine": "He joined", "feminine": "She joined", "other": "They joined"}
	}`)
	if err := os.WriteFile(filepath.Join(localeDir, "en-US.json"), enUS, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	cache := NewMapCache()
	tr, err := New(
		WithFileSystemLoader(localeDir),
		WithDefaultLocale("en-US"),
		WithCache(cache),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_ = tr.Translate("greeting")
	_ = tr.TranslatePlural("items", 1)
	_ = tr.TranslateGender("welcome", core.Masculine)

	// All three should exist under distinct cache key formats.
	if _, ok := cache.Get("en-US:greeting"); !ok {
		t.Error("expected cache entry for Translate key format (locale:key)")
	}
	if _, ok := cache.Get("en-US:items#1"); !ok {
		t.Error("expected cache entry for TranslatePlural key format (locale:key#count)")
	}
	if _, ok := cache.Get("en-US:welcome@masculine"); !ok {
		t.Error("expected cache entry for TranslateGender key format (locale:key@gender)")
	}
}

// TestCacheFallbackChainResultCachedUnderCurrentLocale verifies that a
// translation resolved through the fallback chain is cached under the
// requesting locale's key, not the fallback locale's key.
func TestCacheFallbackChainResultCachedUnderCurrentLocale(t *testing.T) {
	tmpDir := t.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	enUS := []byte(`{"greeting":"Hello"}`)
	esMX := []byte(`{"other":"Otro"}`)
	if err := os.WriteFile(filepath.Join(localeDir, "en-US.json"), enUS, 0o644); err != nil {
		t.Fatalf("write en-US: %v", err)
	}
	if err := os.WriteFile(filepath.Join(localeDir, "es-MX.json"), esMX, 0o644); err != nil {
		t.Fatalf("write es-MX: %v", err)
	}

	cache := NewMapCache()
	tr, err := New(
		WithFileSystemLoader(localeDir),
		WithDefaultLocale("es-MX"),
		WithCache(cache),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// "greeting" does not exist in es-MX, falls back to en-US.
	result := tr.Translate("greeting")
	if result != "Hello" {
		t.Fatalf("Translate = %q, want %q", result, "Hello")
	}

	// The result should be cached under es-MX (the requesting locale).
	val, ok := cache.Get("es-MX:greeting")
	if !ok {
		t.Error("expected fallback result cached under es-MX:greeting")
	}
	if val != "Hello" {
		t.Errorf("cached value = %q, want %q", val, "Hello")
	}
}

// TestTranslateWithArgsBenefitsFromTranslateCache verifies that
// TranslateWithArgs benefits indirectly from the Translate cache when
// the format string is already cached.
func TestTranslateWithArgsBenefitsFromTranslateCache(t *testing.T) {
	tmpDir := t.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	enUS := []byte(`{"welcome":"Hello, %s!"}`)
	if err := os.WriteFile(filepath.Join(localeDir, "en-US.json"), enUS, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	cache := NewMapCache()
	tr, err := New(
		WithFileSystemLoader(localeDir),
		WithDefaultLocale("en-US"),
		WithCache(cache),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// First call to TranslateWithArgs populates the cache for the format string.
	r1 := tr.TranslateWithArgs("welcome", "Alice")
	if r1 != "Hello, Alice!" {
		t.Fatalf("first call = %q, want %q", r1, "Hello, Alice!")
	}

	// The format string should now be cached via the inner Translate call.
	if _, ok := cache.Get("en-US:welcome"); !ok {
		t.Error("expected format string to be cached via Translate")
	}

	// Second call with different args should still work correctly.
	r2 := tr.TranslateWithArgs("welcome", "Bob")
	if r2 != "Hello, Bob!" {
		t.Errorf("second call = %q, want %q", r2, "Hello, Bob!")
	}
}

// TestMapCacheLRUInterleavedGetSet exercises LRU ordering when Get and Set
// operations are interleaved across more entries than the cache limit.
func TestMapCacheLRUInterleavedGetSet(t *testing.T) {
	c := NewMapCacheWithLimit(3)

	c.Set("a", "1")
	c.Set("b", "2")
	c.Set("c", "3")
	// Order: [c, b, a]

	// Access "a" to promote it; order becomes [a, c, b]
	c.Get("a")

	// Update "b" via Set; order becomes [b, a, c]
	c.Set("b", "2-updated")

	// Insert "d"; "c" is the tail and should be evicted.
	c.Set("d", "4")

	if _, ok := c.Get("c"); ok {
		t.Error("entry 'c' should have been evicted")
	}

	if val, ok := c.Get("a"); !ok || val != "1" {
		t.Errorf("entry 'a' should exist, got ok=%v val=%q", ok, val)
	}
	if val, ok := c.Get("b"); !ok || val != "2-updated" {
		t.Errorf("entry 'b' should exist with updated value, got ok=%v val=%q", ok, val)
	}
	if val, ok := c.Get("d"); !ok || val != "4" {
		t.Errorf("entry 'd' should exist, got ok=%v val=%q", ok, val)
	}
}

// TestMapCacheWithLimitNegativeBehavesUnlimited verifies that
// NewMapCacheWithLimit with a negative value behaves as unlimited.
func TestMapCacheWithLimitNegativeBehavesUnlimited(t *testing.T) {
	c := NewMapCacheWithLimit(-5)

	for i := 0; i < 50; i++ {
		c.Set(fmt.Sprintf("k%d", i), fmt.Sprintf("v%d", i))
	}

	for i := 0; i < 50; i++ {
		if _, ok := c.Get(fmt.Sprintf("k%d", i)); !ok {
			t.Errorf("k%d should exist with negative limit (unlimited)", i)
		}
	}
}

// TestMapCacheSetUpdatesExistingEntry verifies that calling Set with an
// existing key updates its value and promotes it to the head.
func TestMapCacheSetUpdatesExistingEntry(t *testing.T) {
	c := NewMapCacheWithLimit(2)

	c.Set("a", "original")
	c.Set("b", "second")
	// Order: [b, a]

	// Update "a" which should promote it to head: [a, b]
	c.Set("a", "updated")

	// Insert "c"; "b" should be evicted (tail)
	c.Set("c", "third")

	if _, ok := c.Get("b"); ok {
		t.Error("entry 'b' should have been evicted")
	}

	val, ok := c.Get("a")
	if !ok {
		t.Fatal("entry 'a' should exist")
	}
	if val != "updated" {
		t.Errorf("entry 'a' = %q, want %q", val, "updated")
	}
}

// ---------------------------------------------------------------------------
// Fuzz targets
// ---------------------------------------------------------------------------

// FuzzMapCacheGetSet exercises MapCache Get and Set with arbitrary string keys
// and values, verifying round-trip correctness and that the cache never panics.
func FuzzMapCacheGetSet(f *testing.F) {
	f.Add("greeting", "Hello")
	f.Add("", "")
	f.Add("\x00\x01\x02", "\x7f\x80\xff")
	f.Add(strings.Repeat("k", 300), strings.Repeat("v", 300))
	f.Add("../../etc/passwd", "attack-value")
	f.Add("key\u202Awith\u202Ebidi", "val\u202Cwith\u202Dbidi")
	f.Add("en-US:greeting", "cached-translation")

	f.Fuzz(func(t *testing.T, key, value string) {
		// Exercise unlimited-mode cache.
		unlimited := NewMapCache()
		unlimited.Set(key, value)

		got, ok := unlimited.Get(key)
		if !ok {
			t.Errorf("Get(%q) missed after Set on unlimited cache", key)
		}
		if got != value {
			t.Errorf("Get(%q) = %q, want %q", key, got, value)
		}

		// Exercise LRU-mode cache.
		lru := NewMapCacheWithLimit(5)
		lru.Set(key, value)

		got, ok = lru.Get(key)
		if !ok {
			t.Errorf("Get(%q) missed after Set on LRU cache", key)
		}
		if got != value {
			t.Errorf("LRU Get(%q) = %q, want %q", key, got, value)
		}

		// Invalidate must clear the entry.
		unlimited.Invalidate()
		if _, ok := unlimited.Get(key); ok {
			t.Errorf("Get(%q) should miss after Invalidate", key)
		}
	})
}

// ---------------------------------------------------------------------------
// Benchmarks
// ---------------------------------------------------------------------------

// benchTranslator creates a Translator backed by a temporary file system loader
// with a known key that resolves through the fallback chain. If cache is non-nil,
// it is wired via WithCache.
func benchTranslator(b *testing.B, cache core.Cacher) *Translator {
	b.Helper()

	tmpDir := b.TempDir()
	localeDir := filepath.Join(tmpDir, "locales")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		b.Fatalf("mkdir: %v", err)
	}

	// en-US has the target key; es-MX does not, so a lookup from es-MX
	// traverses the fallback chain before landing on en-US.
	enUS := []byte(`{"greeting":"Hello","nested":{"deep":{"key":"Resolved"}}}`)
	esMX := []byte(`{"other":"Otro"}`)
	if err := os.WriteFile(filepath.Join(localeDir, "en-US.json"), enUS, 0o644); err != nil {
		b.Fatalf("write en-US: %v", err)
	}
	if err := os.WriteFile(filepath.Join(localeDir, "es-MX.json"), esMX, 0o644); err != nil {
		b.Fatalf("write es-MX: %v", err)
	}

	opts := []Option{
		WithFileSystemLoader(localeDir),
		WithDefaultLocale("es-MX"),
	}
	if cache != nil {
		opts = append(opts, WithCache(cache))
	}

	tr, err := New(opts...)
	if err != nil {
		b.Fatalf("New: %v", err)
	}
	return tr
}

// BenchmarkTranslateUncached measures Translate throughput without caching,
// traversing the fallback chain on every call.
func BenchmarkTranslateUncached(b *testing.B) {
	tr := benchTranslator(b, nil)

	// Warm the file-level cache so we only measure key resolution overhead.
	tr.Translate("nested.deep.key")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tr.Translate("nested.deep.key")
	}
}

// BenchmarkTranslateCached measures Translate throughput with MapCache enabled,
// serving repeated lookups from the resolved-translation cache.
func BenchmarkTranslateCached(b *testing.B) {
	tr := benchTranslator(b, NewMapCache())

	// First call populates both the file-level cache and the translation cache.
	tr.Translate("nested.deep.key")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tr.Translate("nested.deep.key")
	}
}

// ---------------------------------------------------------------------------
// MapCache micro-benchmarks
// ---------------------------------------------------------------------------

// BenchmarkMapCache_Get_Hit measures Get throughput on an unlimited cache when
// the key is already present (cache hit path, RLock only).
func BenchmarkMapCache_Get_Hit(b *testing.B) {
	c := NewMapCache()
	c.Set("en-US:greeting", "Hello")

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = c.Get("en-US:greeting")
	}
}

// BenchmarkMapCache_Get_Miss measures Get throughput on an unlimited cache when
// the key does not exist (cache miss path, RLock only).
func BenchmarkMapCache_Get_Miss(b *testing.B) {
	c := NewMapCache()
	c.Set("en-US:greeting", "Hello")

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = c.Get("en-US:missing")
	}
}

// BenchmarkMapCache_Set measures Set throughput on an unlimited cache
// inserting new entries (no eviction, no key update).
func BenchmarkMapCache_Set(b *testing.B) {
	c := NewMapCache()

	// Pre-generate keys to avoid measuring fmt.Sprintf in the loop.
	keys := make([]string, b.N)
	for i := range keys {
		keys[i] = fmt.Sprintf("key-%d", i)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		c.Set(keys[i], "value")
	}
}

// BenchmarkMapCache_Set_Eviction measures Set throughput on an LRU-limited
// cache that is at capacity, triggering eviction on every insert.
func BenchmarkMapCache_Set_Eviction(b *testing.B) {
	const limit = 100
	c := NewMapCacheWithLimit(limit)

	// Fill the cache to capacity.
	for i := 0; i < limit; i++ {
		c.Set(fmt.Sprintf("seed-%d", i), "value")
	}

	// Pre-generate keys that are guaranteed to be new (trigger eviction).
	keys := make([]string, b.N)
	for i := range keys {
		keys[i] = fmt.Sprintf("evict-%d", i)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		c.Set(keys[i], "value")
	}
}
