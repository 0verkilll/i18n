// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

package engine

import (
	"errors"
	"fmt"
	"testing"

	"github.com/0verkilll/i18n/internal/core"
)

// TestTranslatorSatisfiesTranslationLookup verifies *Translator implements TranslationLookup.
func TestTranslatorSatisfiesTranslationLookup(t *testing.T) {
	var _ core.TranslationLookup = (*Translator)(nil)
}

// TestTranslatorSatisfiesFormattedTranslator verifies *Translator implements FormattedTranslator.
func TestTranslatorSatisfiesFormattedTranslator(t *testing.T) {
	var _ core.FormattedTranslator = (*Translator)(nil)
}

// TestTranslatorSatisfiesKeyChecker verifies *Translator implements KeyChecker.
func TestTranslatorSatisfiesKeyChecker(t *testing.T) {
	var _ core.KeyChecker = (*Translator)(nil)
}

// TestTranslatorSatisfiesTranslatorProvider verifies *Translator implements the composed TranslatorProvider.
func TestTranslatorSatisfiesTranslatorProvider(t *testing.T) {
	var _ core.TranslatorProvider = (*Translator)(nil)
}

// TestDefaultLocaleDetectorSatisfiesAllInterfaces verifies DefaultLocaleDetector implements
// LocaleDetector, Detector, and Normalizer.
func TestDefaultLocaleDetectorSatisfiesAllInterfaces(t *testing.T) {
	var _ core.LocaleDetector = (*DefaultLocaleDetector)(nil)
	var _ core.Detector = (*DefaultLocaleDetector)(nil)
	var _ core.Normalizer = (*DefaultLocaleDetector)(nil)
}

// TestTranslatorSatisfiesAllSubInterfaces verifies *Translator implements all seven sub-interfaces
// plus the composed TranslatorProvider.
func TestTranslatorSatisfiesAllSubInterfaces(t *testing.T) {
	var _ core.TranslatorProvider = (*Translator)(nil)
	var _ core.TranslationLookup = (*Translator)(nil)
	var _ core.FormattedTranslator = (*Translator)(nil)
	var _ core.KeyChecker = (*Translator)(nil)
	var _ core.LocaleSetter = (*Translator)(nil)
	var _ core.LocaleGetter = (*Translator)(nil)
	var _ core.PluralTranslator = (*Translator)(nil)
	var _ core.GenderTranslator = (*Translator)(nil)
}

// TestDuckTypedTranslatorProvider verifies that *Translator satisfies a locally-defined
// duck-typed interface matching the TranslatorProvider signature, confirming the
// integration pattern documented in the godoc works for downstream consumers.
func TestDuckTypedTranslatorProvider(t *testing.T) {
	// Simulate a downstream package defining its own interface
	type localProvider interface {
		Translate(key string) string
		TranslateWithArgs(key string, args ...interface{}) string
		HasKey(key string) bool
		SetLocale(locale string)
		GetLocale() string
	}

	var _ localProvider = (*Translator)(nil)
}

// TestErrorsIsWithWrappedError verifies errors.Is works through fmt.Errorf %w wrapping
// after the reflect removal from Is() methods.
func TestErrorsIsWithWrappedError(t *testing.T) {
	inner := core.NewErrInvalidLocale("bad", errors.New("cause"))
	wrapped := fmt.Errorf("outer context: %w", inner)

	if !errors.Is(wrapped, core.ErrInvalidLocale{}) {
		t.Error("errors.Is through %w wrapping should find ErrInvalidLocale")
	}

	if !errors.Is(wrapped, &core.ErrInvalidLocale{}) {
		t.Error("errors.Is through %w wrapping should find *ErrInvalidLocale")
	}
}

// TestErrorsIsWrappedKeyNotFound verifies errors.Is works through wrapping for ErrKeyNotFound.
func TestErrorsIsWrappedKeyNotFound(t *testing.T) {
	inner := core.NewErrKeyNotFound("missing.key")
	wrapped := fmt.Errorf("lookup failed: %w", inner)

	if !errors.Is(wrapped, core.ErrKeyNotFound{}) {
		t.Error("errors.Is through %w wrapping should find ErrKeyNotFound")
	}
}

// TestDetectorSubInterfaceIndependent verifies that Detector can be used independently
// of the composed LocaleDetector interface.
func TestDetectorSubInterfaceIndependent(t *testing.T) {
	detector := NewDefaultLocaleDetector(nil)

	// Use only the Detector sub-interface
	var d core.Detector = detector
	result := d.Detect()
	if result == "" {
		t.Error("Detector.Detect() should return a non-empty locale")
	}
}

// TestNormalizerSubInterfaceIndependent verifies that Normalizer can be used independently
// of the composed LocaleDetector interface.
func TestNormalizerSubInterfaceIndependent(t *testing.T) {
	detector := NewDefaultLocaleDetector(nil)

	// Use only the Normalizer sub-interface
	var n core.Normalizer = detector
	result := n.Normalize("en_US.UTF-8")
	if result != "en-US" {
		t.Errorf("Normalizer.Normalize(en_US.UTF-8) = %q, want en-US", result)
	}
}
