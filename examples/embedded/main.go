// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

package main

import (
	"embed"
	"fmt"
	"log"

	"github.com/0verkilll/i18n"
)

// Embed translation files into the binary
//
//go:embed locales/*.json
var translationsFS embed.FS

func main() {
	// Create a translator using embedded files
	// This is perfect for distributing applications as a single binary
	loader := i18n.NewEmbedFSLoader(translationsFS, "locales")

	translator, err := i18n.New(
		i18n.WithLoader(loader),
		i18n.WithDefaultLocale("en-US"),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Use the translator
	fmt.Println(translator.Translate("app.name"))
	fmt.Println(translator.Translate("app.description"))

	// Switch to French
	translator.SetLocale("fr-FR")
	fmt.Println(translator.Translate("app.name"))
	fmt.Println(translator.Translate("app.description"))
}
