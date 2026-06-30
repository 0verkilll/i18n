// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

// Example: CLI tool that detects system locale and translates messages.
package main

import (
	"fmt"

	"github.com/0verkilll/i18n"
)

func main() {
	locale := i18n.DetectLocale()
	fmt.Printf("Detected system locale: %s\n", locale)

	t, err := i18n.NewWithFS("locales", locale)
	if err != nil {
		fmt.Printf("Warning: could not load translations: %v\n", err)
		fmt.Printf("Falling back to en-US\n")
		t, _ = i18n.NewWithFS("locales", "en-US")
	}

	fmt.Println(t.Translate("greeting"))
	fmt.Println(t.Translate("farewell"))
}
