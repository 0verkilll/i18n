// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

package main

import (
	"fmt"
	"log"

	"github.com/0verkilll/i18n"
)

func main() {
	// Create a translator with file system loader
	// Translation files are expected in ./locales/ directory
	translator, err := i18n.New(
		i18n.WithFileSystemLoader("./locales"),
		i18n.WithDefaultLocale("en-US"),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Translate a simple key
	greeting := translator.Translate("greeting")
	fmt.Println(greeting)

	// Translate a nested key using dot notation
	errorMsg := translator.Translate("errors.validation.required")
	fmt.Println(errorMsg)

	// Check if a key exists
	if translator.HasKey("farewell") {
		fmt.Println(translator.Translate("farewell"))
	}

	// Switch to a different locale
	translator.SetLocale("es-ES")
	greeting = translator.Translate("greeting")
	fmt.Println(greeting)

	// Get current locale
	fmt.Printf("Current locale: %s\n", translator.GetLocale())
}
