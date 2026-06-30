// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

package main

import (
	"fmt"
	"log"

	"github.com/0verkilll/i18n"
)

func main() {
	translator, err := i18n.New(
		i18n.WithFileSystemLoader("./locales"),
		i18n.WithDefaultLocale("en-US"),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Simple string formatting with arguments
	username := "Alice"
	welcome := translator.TranslateWithArgs("messages.welcome_user", username)
	fmt.Println(welcome)

	// Multiple arguments
	itemCount := 5
	message := translator.TranslateWithArgs("messages.items_found", itemCount, "products")
	fmt.Println(message)

	// Numeric formatting
	price := 99.99
	priceMsg := translator.TranslateWithArgs("messages.price", price)
	fmt.Println(priceMsg)

	// Switch to Spanish
	translator.SetLocale("es-ES")
	welcome = translator.TranslateWithArgs("messages.welcome_user", username)
	fmt.Println(welcome)

	message = translator.TranslateWithArgs("messages.items_found", itemCount, "productos")
	fmt.Println(message)

	priceMsg = translator.TranslateWithArgs("messages.price", price)
	fmt.Println(priceMsg)
}
