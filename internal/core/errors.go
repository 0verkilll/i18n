// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

package core

import (
	"fmt"
)

// ErrInvalidLocale indicates that a locale string is invalid or malformed.
type ErrInvalidLocale struct {
	Cause  error
	Locale string
}

// NewErrInvalidLocale creates a new ErrInvalidLocale error for the given locale
// string and underlying cause.
func NewErrInvalidLocale(locale string, cause error) *ErrInvalidLocale {
	return &ErrInvalidLocale{
		Locale: locale,
		Cause:  cause,
	}
}

// Error returns a formatted message including the invalid locale and its cause.
func (e ErrInvalidLocale) Error() string {
	return fmt.Sprintf("invalid locale %q (expected BCP 47 format like 'en-US'): %v", e.Locale, e.Cause)
}

// Unwrap returns the underlying cause of the error.
func (e ErrInvalidLocale) Unwrap() error {
	return e.Cause
}

// Is implements error comparison for ErrInvalidLocale.
// Called by errors.Is() which handles unwrapping; this method only matches the immediate target.
func (e ErrInvalidLocale) Is(target error) bool { //nolint:erroris // called by errors.Is which unwraps
	switch target.(type) {
	case ErrInvalidLocale, *ErrInvalidLocale:
		return true
	default:
		return false
	}
}

// ErrInvalidKey indicates that a translation key is invalid.
type ErrInvalidKey struct {
	Cause error
	Key   string
}

// NewErrInvalidKey creates a new ErrInvalidKey error for the given key and
// underlying cause.
func NewErrInvalidKey(key string, cause error) *ErrInvalidKey {
	return &ErrInvalidKey{
		Key:   key,
		Cause: cause,
	}
}

// Error returns a formatted message including the invalid key and its cause.
func (e ErrInvalidKey) Error() string {
	return fmt.Sprintf("invalid translation key %q (allowed: a-z A-Z 0-9 . _ -): %v", e.Key, e.Cause)
}

// Unwrap returns the underlying cause of the error.
func (e ErrInvalidKey) Unwrap() error {
	return e.Cause
}

// Is implements error comparison for ErrInvalidKey.
func (e ErrInvalidKey) Is(target error) bool { //nolint:erroris // called by errors.Is which unwraps
	switch target.(type) {
	case ErrInvalidKey, *ErrInvalidKey:
		return true
	default:
		return false
	}
}

// ErrKeyNotFound indicates that a translation key was not found.
type ErrKeyNotFound struct {
	Key string
}

// NewErrKeyNotFound creates a new ErrKeyNotFound error for the given key.
func NewErrKeyNotFound(key string) *ErrKeyNotFound {
	return &ErrKeyNotFound{
		Key: key,
	}
}

// Error returns a formatted message including the missing key.
func (e ErrKeyNotFound) Error() string {
	return fmt.Sprintf("translation key %q not found in loaded translations", e.Key)
}

// Is implements error comparison for ErrKeyNotFound.
func (e ErrKeyNotFound) Is(target error) bool { //nolint:erroris // called by errors.Is which unwraps
	switch target.(type) {
	case ErrKeyNotFound, *ErrKeyNotFound:
		return true
	default:
		return false
	}
}

// ErrInvalidFormat indicates that a format string is invalid.
type ErrInvalidFormat struct {
	Cause  error
	Format string
}

// NewErrInvalidFormat creates a new ErrInvalidFormat error for the given format
// identifier and underlying cause.
func NewErrInvalidFormat(format string, cause error) *ErrInvalidFormat {
	return &ErrInvalidFormat{
		Format: format,
		Cause:  cause,
	}
}

// Error returns a formatted message including the format identifier and its cause.
func (e ErrInvalidFormat) Error() string {
	return fmt.Sprintf("invalid format string %q: %v", e.Format, e.Cause)
}

// Unwrap returns the underlying cause of the error.
func (e ErrInvalidFormat) Unwrap() error {
	return e.Cause
}

// Is implements error comparison for ErrInvalidFormat.
func (e ErrInvalidFormat) Is(target error) bool { //nolint:erroris // called by errors.Is which unwraps
	switch target.(type) {
	case ErrInvalidFormat, *ErrInvalidFormat:
		return true
	default:
		return false
	}
}

// ErrPathTraversal indicates that a path traversal attempt was detected.
type ErrPathTraversal struct {
	Path string
}

// NewErrPathTraversal creates a new ErrPathTraversal error for the given path.
func NewErrPathTraversal(path string) *ErrPathTraversal {
	return &ErrPathTraversal{
		Path: path,
	}
}

// Error returns a formatted message including the offending path.
func (e ErrPathTraversal) Error() string {
	return fmt.Sprintf("path traversal detected in %q (use simple locale codes like 'en-US')", e.Path)
}

// Is implements error comparison for ErrPathTraversal.
func (e ErrPathTraversal) Is(target error) bool { //nolint:erroris // called by errors.Is which unwraps
	switch target.(type) {
	case ErrPathTraversal, *ErrPathTraversal:
		return true
	default:
		return false
	}
}

// ErrUnknownFormat indicates that no parser is registered for a given file extension.
type ErrUnknownFormat struct {
	Extension string
}

// NewErrUnknownFormat creates a new ErrUnknownFormat error for the given
// extension string.
func NewErrUnknownFormat(ext string) *ErrUnknownFormat {
	return &ErrUnknownFormat{
		Extension: ext,
	}
}

// Error returns a formatted message including the unregistered extension.
func (e ErrUnknownFormat) Error() string {
	return fmt.Sprintf("unknown file format %q (registered formats: use RegisterParser to add support)", e.Extension)
}

// Is implements error comparison for ErrUnknownFormat.
func (e ErrUnknownFormat) Is(target error) bool { //nolint:erroris // called by errors.Is which unwraps
	switch target.(type) {
	case ErrUnknownFormat, *ErrUnknownFormat:
		return true
	default:
		return false
	}
}
