// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

//go:build !js

package engine

import (
	"os"

	"github.com/0verkilll/i18n/internal/core"
)

// Compile-time assertion that OSEnvProvider implements core.EnvProvider.
var _ core.EnvProvider = (*OSEnvProvider)(nil)

// OSEnvProvider wraps os.Getenv for standard Go environments.
type OSEnvProvider struct{}

// Getenv returns the value of the environment variable named by the key.
func (p *OSEnvProvider) Getenv(key string) string {
	return os.Getenv(key)
}

// defaultEnvProvider returns the platform-appropriate default core.EnvProvider.
func defaultEnvProvider() core.EnvProvider {
	return &OSEnvProvider{}
}
