// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

//go:build !(js && wasm)

package engine

import "testing"

func TestBrowserDetector_StubDetect(t *testing.T) {
	d := NewBrowserDetector()
	got := d.Detect()
	if got != "" {
		t.Errorf("Detect() = %q, want \"\" on non-WASM build", got)
	}
}

func TestBrowserDetector_StubNormalize(t *testing.T) {
	d := NewBrowserDetector()
	got := d.Normalize("en_US.UTF-8")
	want := "en-US"
	if got != want {
		t.Errorf("Normalize(\"en_US.UTF-8\") = %q, want %q", got, want)
	}
}
