// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

// Example: Game HUD translation with caching for per-frame performance.
package main

import (
	"fmt"
	"time"

	"github.com/0verkilll/i18n"
)

func main() {
	cache := i18n.NewMapCacheWithLimit(1000)
	t, err := i18n.NewWithFS("locales", "en-US", i18n.WithCache(cache))
	if err != nil {
		panic(err)
	}

	// Simulate 3 frames of a game loop.
	for frame := 0; frame < 3; frame++ {
		start := time.Now()

		// These are cached after the first frame -- sub-microsecond.
		health := t.Translate("hud.health")
		score := t.Translate("hud.score")
		ammo := t.Translate("hud.ammo")

		elapsed := time.Since(start)
		fmt.Printf("Frame %d: %s | %s | %s (took %v)\n", frame, health, score, ammo, elapsed)
	}
}
