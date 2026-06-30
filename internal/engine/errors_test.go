// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

package engine

import (
	"errors"
	"fmt"
	"testing"

	"github.com/0verkilll/i18n/internal/core"
)

func TestErrorTypes(t *testing.T) {
	tests := []struct {
		err         error
		wantType    error
		name        string
		wantMessage string
	}{
		{
			name:        "ErrInvalidLocale",
			err:         core.NewErrInvalidLocale("en_US.UTF-8", errors.New("contains invalid characters")),
			wantMessage: "invalid locale \"en_US.UTF-8\" (expected BCP 47 format like 'en-US'): contains invalid characters",
			wantType:    core.ErrInvalidLocale{},
		},
		{
			name:        "ErrInvalidKey",
			err:         core.NewErrInvalidKey("some..key", errors.New("consecutive dots")),
			wantMessage: "invalid translation key \"some..key\" (allowed: a-z A-Z 0-9 . _ -): consecutive dots",
			wantType:    core.ErrInvalidKey{},
		},
		{
			name:        "ErrKeyNotFound",
			err:         core.NewErrKeyNotFound("missing.key"),
			wantMessage: "translation key \"missing.key\" not found in loaded translations",
			wantType:    core.ErrKeyNotFound{},
		},
		{
			name:        "ErrInvalidFormat",
			err:         core.NewErrInvalidFormat("%s %d", errors.New("mismatched arguments")),
			wantMessage: "invalid format string \"%s %d\": mismatched arguments",
			wantType:    core.ErrInvalidFormat{},
		},
		{
			name:        "ErrPathTraversal",
			err:         core.NewErrPathTraversal("../../etc/passwd"),
			wantMessage: "path traversal detected in \"../../etc/passwd\" (use simple locale codes like 'en-US')",
			wantType:    core.ErrPathTraversal{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test error message
			if got := tt.err.Error(); got != tt.wantMessage {
				t.Errorf("Error() = %q, want %q", got, tt.wantMessage)
			}

			// Test errors.Is() compatibility
			if !errors.Is(tt.err, tt.wantType) {
				t.Errorf("errors.Is() failed for %T", tt.wantType)
			}
		})
	}
}

func TestErrorUnwrapping(t *testing.T) {
	originalErr := errors.New("original error")

	tests := []struct {
		err  error
		want error
		name string
	}{
		{
			name: "ErrInvalidLocale unwraps",
			err:  core.NewErrInvalidLocale("bad", originalErr),
			want: originalErr,
		},
		{
			name: "ErrInvalidKey unwraps",
			err:  core.NewErrInvalidKey("bad", originalErr),
			want: originalErr,
		},
		{
			name: "ErrInvalidFormat unwraps",
			err:  core.NewErrInvalidFormat("bad", originalErr),
			want: originalErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unwrapped := errors.Unwrap(tt.err)
			// Verify Unwrap returns the expected error
			// We check both nil-ness and pointer equality for exact instance match
			if (unwrapped == nil && tt.want != nil) || (unwrapped != nil && tt.want == nil) {
				t.Errorf("Unwrap() = %v, want %v", unwrapped, tt.want)
			} else if unwrapped != nil && tt.want != nil {
				// Compare instances using formatted pointers to avoid direct error comparison warning
				gotPtr := fmt.Sprintf("%p", unwrapped)
				wantPtr := fmt.Sprintf("%p", tt.want)
				if gotPtr != wantPtr {
					t.Errorf("Unwrap() = %v (ptr: %s), want %v (ptr: %s)", unwrapped, gotPtr, tt.want, wantPtr)
				}
			}
		})
	}
}

func TestErrorsWithoutCause(t *testing.T) {
	tests := []struct {
		err  error
		name string
	}{
		{
			name: "ErrKeyNotFound without cause",
			err:  core.NewErrKeyNotFound("key"),
		},
		{
			name: "ErrPathTraversal without cause",
			err:  core.NewErrPathTraversal("path"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// These errors don't wrap another error
			unwrapped := errors.Unwrap(tt.err)
			if unwrapped != nil {
				t.Errorf("Unwrap() = %v, want nil", unwrapped)
			}
		})
	}
}

func TestErrUnknownFormat_Error(t *testing.T) {
	err := core.NewErrUnknownFormat(".toml")
	want := `unknown file format ".toml" (registered formats: use RegisterParser to add support)`
	if got := err.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestErrUnknownFormat_Is(t *testing.T) {
	err := core.NewErrUnknownFormat(".toml")

	// Match via value type
	if !errors.Is(err, core.ErrUnknownFormat{}) {
		t.Error("errors.Is(err, ErrUnknownFormat{}) should return true")
	}

	// Match via pointer type
	if !errors.Is(err, &core.ErrUnknownFormat{}) {
		t.Error("errors.Is(err, &ErrUnknownFormat{}) should return true")
	}

	// Does not match unrelated error type
	if errors.Is(err, core.ErrKeyNotFound{}) {
		t.Error("errors.Is(err, ErrKeyNotFound{}) should return false")
	}
}

func TestNewErrUnknownFormat(t *testing.T) {
	err := core.NewErrUnknownFormat(".yaml")
	if err.Extension != ".yaml" {
		t.Errorf("Extension = %q, want %q", err.Extension, ".yaml")
	}
}

// =============================================================================
// Merged from errors_is_test.go
// =============================================================================

// TestIsMatchesValueReceiver verifies errors.Is matches value receivers for all error types.
func TestIsMatchesValueReceiver(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		target error
	}{
		{"ErrInvalidLocale", core.NewErrInvalidLocale("en", errors.New("test")), core.ErrInvalidLocale{}},
		{"ErrInvalidKey", core.NewErrInvalidKey("k", errors.New("test")), core.ErrInvalidKey{}},
		{"ErrKeyNotFound", core.NewErrKeyNotFound("k"), core.ErrKeyNotFound{}},
		{"ErrInvalidFormat", core.NewErrInvalidFormat("%s", errors.New("test")), core.ErrInvalidFormat{}},
		{"ErrPathTraversal", core.NewErrPathTraversal(".."), core.ErrPathTraversal{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !errors.Is(tt.err, tt.target) {
				t.Errorf("errors.Is(%T, %T) = false, want true", tt.err, tt.target)
			}
		})
	}
}

// TestIsMatchesPointerReceiver verifies errors.Is matches pointer receivers for all error types.
func TestIsMatchesPointerReceiver(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		target error
	}{
		{"ErrInvalidLocale", core.NewErrInvalidLocale("en", errors.New("test")), &core.ErrInvalidLocale{}},
		{"ErrInvalidKey", core.NewErrInvalidKey("k", errors.New("test")), &core.ErrInvalidKey{}},
		{"ErrKeyNotFound", core.NewErrKeyNotFound("k"), &core.ErrKeyNotFound{}},
		{"ErrInvalidFormat", core.NewErrInvalidFormat("%s", errors.New("test")), &core.ErrInvalidFormat{}},
		{"ErrPathTraversal", core.NewErrPathTraversal(".."), &core.ErrPathTraversal{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !errors.Is(tt.err, tt.target) {
				t.Errorf("errors.Is(%T, %T) = false, want true", tt.err, tt.target)
			}
		})
	}
}

// TestIsReturnsFalseForUnrelatedTypes verifies errors.Is returns false for unrelated errors.
// This exercises the "return false" path in every Is() method.
func TestIsReturnsFalseForUnrelatedTypes(t *testing.T) {
	unrelated := fmt.Errorf("unrelated error")

	// ErrInvalidLocale.Is returns false for unrelated types
	localeErr := core.NewErrInvalidLocale("en", errors.New("test"))
	if errors.Is(localeErr, unrelated) {
		t.Error("errors.Is(ErrInvalidLocale, unrelated) should return false")
	}
	if errors.Is(localeErr, core.ErrKeyNotFound{}) {
		t.Error("errors.Is(ErrInvalidLocale, ErrKeyNotFound) should return false")
	}

	// ErrInvalidKey.Is returns false for unrelated types
	keyErr := core.NewErrInvalidKey("k", errors.New("test"))
	if errors.Is(keyErr, unrelated) {
		t.Error("errors.Is(ErrInvalidKey, unrelated) should return false")
	}
	if errors.Is(keyErr, core.ErrInvalidLocale{}) {
		t.Error("errors.Is(ErrInvalidKey, ErrInvalidLocale) should return false")
	}

	// ErrKeyNotFound.Is returns false for unrelated types
	notFoundErr := core.NewErrKeyNotFound("k")
	if errors.Is(notFoundErr, unrelated) {
		t.Error("errors.Is(ErrKeyNotFound, unrelated) should return false")
	}
	if errors.Is(notFoundErr, core.ErrInvalidKey{}) {
		t.Error("errors.Is(ErrKeyNotFound, ErrInvalidKey) should return false")
	}

	// ErrInvalidFormat.Is returns false for unrelated types
	formatErr := core.NewErrInvalidFormat("%s", errors.New("test"))
	if errors.Is(formatErr, unrelated) {
		t.Error("errors.Is(ErrInvalidFormat, unrelated) should return false")
	}
	if errors.Is(formatErr, core.ErrPathTraversal{}) {
		t.Error("errors.Is(ErrInvalidFormat, ErrPathTraversal) should return false")
	}

	// ErrPathTraversal.Is returns false for unrelated types
	pathErr := core.NewErrPathTraversal("..")
	if errors.Is(pathErr, unrelated) {
		t.Error("errors.Is(ErrPathTraversal, unrelated) should return false")
	}
	if errors.Is(pathErr, core.ErrInvalidFormat{}) {
		t.Error("errors.Is(ErrPathTraversal, ErrInvalidFormat) should return false")
	}

	// ErrUnknownFormat.Is returns false for unrelated types
	unknownErr := core.NewErrUnknownFormat(".toml")
	if errors.Is(unknownErr, unrelated) {
		t.Error("errors.Is(ErrUnknownFormat, unrelated) should return false")
	}
	if errors.Is(unknownErr, core.ErrKeyNotFound{}) {
		t.Error("errors.Is(ErrUnknownFormat, ErrKeyNotFound) should return false")
	}
}

// TestIsReturnsFalseForNilTarget verifies errors.Is handles nil target correctly.
func TestIsReturnsFalseForNilTarget(t *testing.T) {
	err := core.NewErrInvalidLocale("en", errors.New("test"))
	if errors.Is(err, nil) {
		t.Error("errors.Is(err, nil) should return false")
	}
}

// TestUnwrapStillWorksAfterIsChanges confirms Unwrap is not regressed by Is() changes.
func TestUnwrapStillWorksAfterIsChanges(t *testing.T) {
	cause := errors.New("root cause")
	tests := []struct {
		name string
		err  error
		want error
	}{
		{"ErrInvalidLocale", core.NewErrInvalidLocale("en", cause), cause},
		{"ErrInvalidKey", core.NewErrInvalidKey("k", cause), cause},
		{"ErrInvalidFormat", core.NewErrInvalidFormat("%s", cause), cause},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unwrapped := errors.Unwrap(tt.err)
			if !errors.Is(unwrapped, tt.want) {
				t.Errorf("Unwrap() = %v, want %v", unwrapped, tt.want)
			}
		})
	}
}
