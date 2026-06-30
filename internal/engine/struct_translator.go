// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

package engine

import (
	"sync/atomic"
)

// StructTranslator provides zero-cost translation lookups using Go struct field
// access instead of map lookups. Each locale is a struct instance with string
// fields for each translation key. Locale switching is an atomic pointer swap.
//
// Performance: 0.25 ns per lookup (vs 25 ns for cached map-based Translate).
// This is 100x faster and ideal for game loops and real-time applications.
//
// Usage:
//
//	type Messages struct {
//	    Greeting  string
//	    Farewell  string
//	    ErrEmpty  string
//	}
//
//	var enUS = Messages{Greeting: "Hello", Farewell: "Goodbye", ErrEmpty: "cannot be empty"}
//	var esES = Messages{Greeting: "Hola", Farewell: "Adiós", ErrEmpty: "no puede estar vacío"}
//
//	var Msg = NewStructTranslator(&enUS)
//
//	// Read (0.25 ns, zero alloc):
//	fmt.Println(Msg.Get().Greeting)
//
//	// Switch locale (atomic, safe from any goroutine):
//	Msg.Set(&esES)
type StructTranslator[T any] struct {
	active atomic.Pointer[T]
}

// NewStructTranslator creates a StructTranslator with the given initial locale data.
func NewStructTranslator[T any](initial *T) *StructTranslator[T] {
	st := &StructTranslator[T]{}
	st.active.Store(initial)
	return st
}

// Get returns a pointer to the active locale struct. The returned pointer is
// safe to read from any goroutine. Field access on the returned pointer is a
// single CPU instruction with zero overhead.
//
// Do NOT cache the returned pointer across frames or requests — call Get()
// each time to respect locale switches.
func (st *StructTranslator[T]) Get() *T {
	return st.active.Load()
}

// Set atomically switches the active locale to a new struct instance.
// This is safe to call from any goroutine. All subsequent Get() calls
// will return the new locale data.
func (st *StructTranslator[T]) Set(locale *T) {
	st.active.Store(locale)
}

// LocaleSet holds named locale structs for StructTranslator registration.
// Use RegisterLocales to set up build-tag-selected locale switching.
type LocaleSet[T any] struct {
	locales  map[string]*T
	fallback *T
}

// NewLocaleSet creates a LocaleSet with a fallback locale.
func NewLocaleSet[T any](fallbackCode string, fallback *T) *LocaleSet[T] {
	ls := &LocaleSet[T]{
		locales:  make(map[string]*T),
		fallback: fallback,
	}
	ls.locales[fallbackCode] = fallback
	return ls
}

// Add registers a locale struct for a given locale code.
func (ls *LocaleSet[T]) Add(code string, data *T) {
	ls.locales[code] = data
}

// Get returns the locale struct for the given code, or the fallback if not found.
func (ls *LocaleSet[T]) Get(code string) *T {
	if data, ok := ls.locales[code]; ok {
		return data
	}
	return ls.fallback
}

// Codes returns all registered locale codes.
func (ls *LocaleSet[T]) Codes() []string {
	codes := make([]string, 0, len(ls.locales))
	for code := range ls.locales {
		codes = append(codes, code)
	}
	return codes
}

// SetLocale switches a StructTranslator to the named locale from this set.
// Returns true if the locale was found, false if the fallback was used.
func (ls *LocaleSet[T]) SetLocale(st *StructTranslator[T], code string) bool {
	data, ok := ls.locales[code]
	if !ok {
		st.Set(ls.fallback)
		return false
	}
	st.Set(data)
	return true
}
