// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

package engine

import (
	"errors"
	"fmt"
	"testing"
)

// TestRegisterLocale_ReturnsErrorOnInvalid verifies registerLocale now
// returns an error for invalid codes (while still leaving the registry
// unchanged, matching prior silent-skip behaviour).
func TestRegisterLocale_ReturnsErrorOnInvalid(t *testing.T) {
	resetRegistry()
	defer resetRegistry()

	err := registerLocale("INVALID", []byte(`{}`))
	if err == nil {
		t.Fatal("registerLocale(\"INVALID\") returned nil error; expected validation error")
	}
	if len(RegisteredLocales()) != 0 {
		t.Errorf("registry should be empty after invalid registration, got %v", RegisteredLocales())
	}
}

// TestRegisterLocale_ReturnsNilOnValid asserts the happy-path return value.
func TestRegisterLocale_ReturnsNilOnValid(t *testing.T) {
	resetRegistry()
	defer resetRegistry()

	if err := registerLocale("en-US", []byte(`{}`)); err != nil {
		t.Fatalf("registerLocale(en-US) err=%v, want nil", err)
	}
}

// TestRegisterLocale_RejectsOverflow verifies that once the registry holds
// MaxRegisteredLocales distinct entries, a NEW registration returns
// ErrRegistryFull. Replacing an existing entry still succeeds.
func TestRegisterLocale_RejectsOverflow(t *testing.T) {
	resetRegistry()
	defer resetRegistry()

	// Synthesize MaxRegisteredLocales distinct valid BCP 47 codes. We use
	// two-letter-language + "-" + two-letter-region. There are 26*26 = 676
	// two-letter combinations per side, which is well above 256.
	codes := make([]string, 0, MaxRegisteredLocales)
	for i := 0; len(codes) < MaxRegisteredLocales; i++ {
		lang := string([]byte{byte('a' + (i/26)%26), byte('a' + i%26)})
		reg := string([]byte{byte('A' + (i/(26*26))%26), byte('A' + (i/26)%26)})
		codes = append(codes, lang+"-"+reg)
	}

	for _, c := range codes {
		if err := registerLocale(c, []byte(`{}`)); err != nil {
			t.Fatalf("registerLocale(%q) failed while filling registry: %v", c, err)
		}
	}

	if got := len(RegisteredLocales()); got != MaxRegisteredLocales {
		t.Fatalf("registry size = %d, want %d", got, MaxRegisteredLocales)
	}

	// Now adding a NEW code must fail.
	err := registerLocale("zz-ZZ", []byte(`{}`))
	if err == nil {
		t.Fatal("registerLocale(zz-ZZ) should have failed with ErrRegistryFull")
	}
	if !errors.Is(err, ErrRegistryFull) {
		t.Errorf("err=%v; errors.Is(err, ErrRegistryFull) should be true", err)
	}

	// And registry size is unchanged.
	if got := len(RegisteredLocales()); got != MaxRegisteredLocales {
		t.Errorf("registry size after rejected add = %d, want %d", got, MaxRegisteredLocales)
	}

	// Replacing an existing code still succeeds even when full.
	if err := registerLocale(codes[0], []byte(`{"new":"data"}`)); err != nil {
		t.Errorf("re-registering existing code at capacity should succeed, got err=%v", err)
	}
}

// TestRegisterLocale_OverflowErrorMentionsLimit asserts the error message
// carries the numeric limit so operators can diagnose the overflow.
func TestRegisterLocale_OverflowErrorMentionsLimit(t *testing.T) {
	resetRegistry()
	defer resetRegistry()

	for i := 0; i < MaxRegisteredLocales; i++ {
		lang := string([]byte{byte('a' + (i/26)%26), byte('a' + i%26)})
		reg := string([]byte{byte('A' + (i/(26*26))%26), byte('A' + (i/26)%26)})
		if err := registerLocale(lang+"-"+reg, []byte(`{}`)); err != nil {
			t.Fatalf("fill: %v", err)
		}
	}

	err := registerLocale("zz-ZZ", []byte(`{}`))
	if err == nil {
		t.Fatal("expected error")
	}
	wantSubstr := fmt.Sprintf("%d", MaxRegisteredLocales)
	if !errStringContains(err, wantSubstr) {
		t.Errorf("err=%q should contain limit %q", err.Error(), wantSubstr)
	}
}

// errStringContains is a small helper to keep the test free of a strings
// import when the file may not otherwise need one.
func errStringContains(err error, sub string) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
