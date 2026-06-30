// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

package engine

import (
	"embed"
	"fmt"

	"github.com/0verkilll/i18n/internal/core"
)

// =============================================================================
// Loader configuration
// =============================================================================

// loaderConfig holds shared configuration for loader constructors.
type loaderConfig struct {
	ext string
}

// LoaderOption configures a loader via the functional options pattern.
type LoaderOption func(*loaderConfig)

// WithExtension sets the file extension used by a loader to construct filenames.
// The extension must include the leading dot (e.g., ".toml", ".yaml").
func WithExtension(ext string) LoaderOption {
	return func(cfg *loaderConfig) {
		cfg.ext = ext
	}
}

// defaultLoaderConfig returns a loaderConfig with default values.
func defaultLoaderConfig() loaderConfig {
	return loaderConfig{ext: ".json"}
}

// =============================================================================
// EmbedFS loader
// =============================================================================

// Compile-time assertion that EmbedFSLoader implements core.TranslationLoader.
var _ core.TranslationLoader = (*EmbedFSLoader)(nil)

// EmbedFSLoader loads translation files from an embedded filesystem.
type EmbedFSLoader struct {
	fs       embed.FS
	basePath string
	ext      string
}

// NewEmbedFSLoader creates a new EmbedFSLoader. The default file extension is
// ".json"; use WithExtension to override it.
func NewEmbedFSLoader(fs embed.FS, basePath string, opts ...LoaderOption) *EmbedFSLoader {
	cfg := defaultLoaderConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	return &EmbedFSLoader{
		fs:       fs,
		basePath: basePath,
		ext:      cfg.ext,
	}
}

// Load reads a translation file for the given locale from the embedded filesystem.
func (l *EmbedFSLoader) Load(locale string) ([]byte, error) {
	// Validate locale for path traversal attempts
	if err := validatePath(locale); err != nil {
		return nil, err
	}

	// Construct file path - only allow simple filenames
	// Note: embed.FS always uses forward slashes, regardless of OS
	filename := locale + l.ext
	fullPath := l.basePath + "/" + filename

	// Read from embedded filesystem
	data, err := l.fs.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded translation file for locale %q: %w", locale, err)
	}

	return data, nil
}

// =============================================================================
// Registry loader
// =============================================================================

// Compile-time assertion that RegistryLoader implements core.TranslationLoader.
var _ core.TranslationLoader = (*RegistryLoader)(nil)

// RegistryLoader loads translation data from the global locale registry.
// Locale data is registered at init time by build-tag-selected locale files.
// This loader is a peer to FileSystemLoader and EmbedFSLoader; all three
// loaders remain available and interchangeable via the core.TranslationLoader interface.
type RegistryLoader struct{}

// NewRegistryLoader creates a new RegistryLoader that reads from the global
// locale registry populated by build-tag-selected locale files.
func NewRegistryLoader() *RegistryLoader {
	return &RegistryLoader{}
}

// Load retrieves translation data for the specified locale from the global registry.
// If the locale is registered, Load returns a defensive copy of the data to prevent
// caller mutation of registry contents. If the locale is not registered, Load returns
// a descriptive error that mentions the build-tag mechanism.
func (l *RegistryLoader) Load(locale string) ([]byte, error) {
	registryMu.RLock()
	data, ok := localeRegistry[locale]
	registryMu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("locale %q not registered; build with -tags locale_<code> to include it", locale)
	}

	// Return a defensive copy to prevent callers from mutating registry data.
	copied := append([]byte(nil), data...)
	return copied, nil
}

// WithRegistryLoader creates a RegistryLoader and configures it as the translation
// loader for a Translator. Because built-in locale data is stored in the compact
// binary format, this option also sets the BinaryParser as the translation parser.
// A subsequent WithParser call in the options chain can override the parser if needed.
//
// Usage:
//
//	translator, err := i18n.New(
//	    i18n.WithRegistryLoader(),
//	    i18n.WithDefaultLocale("en-US"),
//	)
func WithRegistryLoader() Option {
	return func(t *Translator) error {
		t.loader = NewRegistryLoader()
		t.parser = NewBinaryParser()
		return nil
	}
}
