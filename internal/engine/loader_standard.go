// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

//go:build !tinygo && !js

package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/0verkilll/i18n/internal/core"
)

// Compile-time assertion that FileSystemLoader implements core.TranslationLoader.
var _ core.TranslationLoader = (*FileSystemLoader)(nil)

// FileSystemLoader loads translation files from the filesystem.
type FileSystemLoader struct {
	baseDir string
	ext     string
}

// NewFileSystemLoader creates a new FileSystemLoader. The default file extension
// is ".json"; use WithExtension to override it.
func NewFileSystemLoader(baseDir string, opts ...LoaderOption) *FileSystemLoader {
	cfg := defaultLoaderConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	return &FileSystemLoader{
		baseDir: baseDir,
		ext:     cfg.ext,
	}
}

// Load reads a translation file for the given locale from the filesystem.
func (l *FileSystemLoader) Load(locale string) ([]byte, error) {
	// Validate base directory
	if l.baseDir == "" {
		return nil, fmt.Errorf("base directory cannot be empty")
	}

	// Validate locale for path traversal attempts
	if err := validatePath(locale); err != nil {
		return nil, err
	}

	// Construct file path - only allow simple filenames, no directories
	filename := locale + l.ext
	fullPath := filepath.Join(l.baseDir, filename)

	// Read the file
	data, err := os.ReadFile(fullPath) // #nosec G304 -- Path is validated by validatePath which prevents path traversal
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("translation file not found for locale %q: %w", locale, err)
		}
		return nil, fmt.Errorf("failed to read translation file for locale %q: %w", locale, err)
	}

	return data, nil
}

// validatePath checks for path traversal attempts.
func validatePath(locale string) error {
	if locale == "" {
		return fmt.Errorf("locale cannot be empty")
	}

	// Check for absolute paths first (most general check)
	if filepath.IsAbs(locale) {
		return core.NewErrPathTraversal(locale)
	}

	// Check for path traversal patterns
	if strings.Contains(locale, "..") {
		return core.NewErrPathTraversal(locale)
	}

	// Check for slashes (prevents any path separators)
	if strings.Contains(locale, "/") {
		return core.NewErrPathTraversal(locale)
	}

	if strings.Contains(locale, "\\") {
		return core.NewErrPathTraversal(locale)
	}

	// Check for home directory expansion
	if strings.HasPrefix(locale, "~") {
		return core.NewErrPathTraversal(locale)
	}

	return nil
}
