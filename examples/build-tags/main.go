// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

// Command build-tags demonstrates using RegistryLoader with build-tag-selected
// locales. Build with -tags to control which locales are compiled in:
//
//	go build -tags locale_en_us
//	go build -tags "locale_en_us,locale_es_es"
//	go build -tags locale_all
package main

import (
	"fmt"
	"log"

	"github.com/0verkilll/i18n"
)

func main() {
	// Show which locales are available in the build-tag registry.
	locales := i18n.RegisteredLocales()
	fmt.Println("Registered locales:", locales)

	if len(locales) == 0 {
		fmt.Println("No locales registered. Build with -tags locale_all to include all locales.")
		fmt.Println("Example: go run -tags locale_all main.go")
		return
	}

	// Create a translator using the registry loader.
	translator, err := i18n.New(
		i18n.WithRegistryLoader(),
		i18n.WithDefaultLocale(locales[0]),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Translate using each registered locale.
	for _, locale := range locales {
		translator.SetLocale(locale)
		fmt.Printf("[%s] %s\n", locale, translator.Translate("greeting"))
	}
}
