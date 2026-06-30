// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

package engine

import (
	"errors"
	"strings"
	"testing"

	"github.com/0verkilll/i18n/internal/core"
)

// TestValidateFormatString_PrecisionCap verifies that precision specifiers
// greater than core.MaxFormatPrecision are rejected. This guards against a
// malicious or corrupted translation string using "%.99999999s" to coerce
// fmt.Sprintf into a multi-megabyte allocation.
func TestValidateFormatString_PrecisionCap(t *testing.T) {
	tests := []struct {
		name     string
		format   string
		argCount int
		wantErr  bool
	}{
		// Allowed — precision within cap.
		{"no precision", "%s", 1, false},
		{"precision zero", "%.0s", 1, false},
		{"precision one", "%.1s", 1, false},
		{"precision ten", "%.10s", 1, false},
		{"precision equals cap", "%.1024s", 1, false},
		{"precision under cap", "%.500d", 1, false},
		{"width and precision ok", "%10.100s", 1, false},
		{"flags width precision", "%-0+ #10.100d", 1, false},

		// Rejected — precision above cap.
		{"precision above cap by 1", "%.1025s", 1, true},
		{"precision 10000", "%.10000s", 1, true},
		{"precision 99999", "%.99999s", 1, true},
		{"precision 1000000", "%.1000000s", 1, true},
		{"precision with width", "%10.10000s", 1, true},

		// Indirect precision ('*') is rejected as a conservative guard.
		{"indirect precision", "%.*s", 2, true},

		// Mid-string specifiers are scanned too.
		{"embedded bad precision", "hello %.9999s world", 1, true},

		// Ensure the pre-existing %n guard still wins.
		{"percent n still rejected", "%n", 1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := core.ValidateFormatString(tt.format, tt.argCount)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFormatString(%q,%d) err=%v, wantErr=%v", tt.format, tt.argCount, err, tt.wantErr)
			}
			if tt.wantErr && err != nil {
				var invFmt *core.ErrInvalidFormat
				if !errors.As(err, &invFmt) {
					t.Errorf("err should wrap *core.ErrInvalidFormat, got %T", err)
				}
			}
		})
	}
}

// TestValidateFormatString_AuditSeeds hits the specific payloads called out
// in the audit plan.
func TestValidateFormatString_AuditSeeds(t *testing.T) {
	type want struct {
		format  string
		args    int
		wantErr bool
	}
	cases := []want{
		{"%.1000s", 1, false},      // within cap
		{"%.0s", 1, false},         // zero precision is fine
		{"%.99999s", 1, true},      // must reject
		{"%.1000000s", 1, true},    // must reject
		{"%.2147483648s", 1, true}, // int overflow territory — still rejected (saturated)
	}
	for _, c := range cases {
		err := core.ValidateFormatString(c.format, c.args)
		if (err != nil) != c.wantErr {
			t.Errorf("ValidateFormatString(%q) err=%v, wantErr=%v", c.format, err, c.wantErr)
		}
	}
}

// FuzzValidateFormatStringPrecision seeds the fuzzer with precision-laden
// format strings. The fuzzer's contract is simple: ValidateFormatString must
// never panic and must always reject precision > MaxFormatPrecision.
func FuzzValidateFormatStringPrecision(f *testing.F) {
	f.Add("%.1000s", byte(1))
	f.Add("%.0s", byte(1))
	f.Add("%.99999s", byte(1))
	f.Add("%.*s", byte(2))
	f.Add("hello %.10d %.2000s world", byte(2))
	f.Add(strings.Repeat("%.10s ", 100), byte(100))

	f.Fuzz(func(t *testing.T, format string, argCountByte byte) {
		argCount := int(argCountByte) % 32
		err := core.ValidateFormatString(format, argCount) //nolint:errcheck // fuzz: must not panic
		// If any precision exceeds the cap, err must be non-nil.
		if containsBadPrecision(format) && err == nil {
			t.Errorf("ValidateFormatString(%q,%d) should reject precision > %d but returned nil", format, argCount, core.MaxFormatPrecision)
		}
	})
}

// containsBadPrecision is a reference implementation used only by the fuzz
// test to decide whether ValidateFormatString MUST return an error. It uses
// a different code path than the production validator so we're
// cross-checking rather than tautologically re-asserting the same logic.
func containsBadPrecision(s string) bool {
	n := len(s)
	i := 0
	for i < n {
		if s[i] != '%' {
			i++
			continue
		}
		if i+1 >= n {
			return false
		}
		if s[i+1] == '%' {
			i += 2
			continue
		}
		// Scan past flags/width.
		j := i + 1
		for j < n && (s[j] == '-' || s[j] == '+' || s[j] == ' ' || s[j] == '#' || s[j] == '0') {
			j++
		}
		if j < n && s[j] == '*' {
			j++
		} else {
			for j < n && s[j] >= '0' && s[j] <= '9' {
				j++
			}
		}
		if j < n && s[j] == '.' {
			j++
			if j < n && s[j] == '*' {
				// production validator also rejects indirect precision
				return true
			}
			// Read digit run.
			v := 0
			start := j
			for j < n && s[j] >= '0' && s[j] <= '9' {
				if v > core.MaxFormatPrecision*100 {
					v = core.MaxFormatPrecision * 100
				} else {
					v = v*10 + int(s[j]-'0')
				}
				j++
			}
			if j > start && v > core.MaxFormatPrecision {
				return true
			}
		}
		i = j + 1
	}
	return false
}
