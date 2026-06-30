// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

//go:build js

package engine

import "github.com/0verkilll/i18n/internal/core"

// Compile-time assertion that WASMEnvProvider implements core.EnvProvider.
var _ core.EnvProvider = (*WASMEnvProvider)(nil)

// WASMEnvProvider provides environment variable access in WASM environments.
// Since browser WASM has no OS environment, it always returns empty strings.
type WASMEnvProvider struct{}

// Getenv always returns an empty string in WASM environments.
func (p *WASMEnvProvider) Getenv(_ string) string {
	return ""
}

// defaultEnvProvider returns the platform-appropriate default core.EnvProvider.
func defaultEnvProvider() core.EnvProvider {
	return &WASMEnvProvider{}
}
