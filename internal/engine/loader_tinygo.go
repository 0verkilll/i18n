// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

//go:build tinygo || js

package engine

import (
	"fmt"
	"strings"

	"github.com/0verkilll/i18n/internal/core"
)

// FileSystemLoader is a no-op implementation for TinyGo/WASM environments where filesystem access is unavailable.
type FileSystemLoader struct {
	baseDir string
	ext     string
}

// NewFileSystemLoader creates a no-op FileSystemLoader for TinyGo/WASM environments.
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

// Load returns an error because filesystem access is unavailable in TinyGo/WASM environments.
func (l *FileSystemLoader) Load(_ string) ([]byte, error) {
	return nil, fmt.Errorf("filesystem access is unavailable in TinyGo/WASM environment")
}

// validatePath checks for path traversal attempts without using filepath.IsAbs.
func validatePath(locale string) error {
	if locale == "" {
		return fmt.Errorf("locale cannot be empty")
	}

	// Check for absolute paths (starts with / or \)
	if strings.HasPrefix(locale, "/") || strings.HasPrefix(locale, "\\") {
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
