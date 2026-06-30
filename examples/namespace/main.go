// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

// Command namespace demonstrates Namespace and PackageTranslator usage with two
// simulated packages, each calling T() and TF() through their own namespace,
// sharing a single Translator instance.
package main

import (
	"fmt"
	"log"

	"github.com/0verkilll/i18n"
)

func main() {
	// Create a shared translator with filesystem loader.
	translator, err := i18n.New(
		i18n.WithFileSystemLoader("locales"),
		i18n.WithDefaultLocale("en-US"),
	)
	if err != nil {
		log.Fatal(err)
	}

	// --- Namespace approach ---
	authNS, err := i18n.NewNamespace("auth", translator)
	if err != nil {
		log.Fatal(err)
	}
	storageNS, err := i18n.NewNamespace("storage", translator)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("=== Namespace (en-US) ===")
	fmt.Println(authNS.T("greeting"))
	fmt.Println(authNS.TF("welcome", "Alice"))
	fmt.Println(storageNS.T("greeting"))
	fmt.Println(storageNS.TF("welcome", "Bob"))

	// --- PackageTranslator approach ---
	authPT := i18n.NewPackageTranslator("auth", i18n.WithDefaults(map[string]string{
		"greeting": "Hi (default)",
		"welcome":  "Welcome, %s! (default)",
	}))
	storagePT := i18n.NewPackageTranslator("storage", i18n.WithDefaults(map[string]string{
		"greeting": "Hi from storage (default)",
	}))

	// Wire up the shared translator.
	authPT.SetTranslator(translator)
	storagePT.SetTranslator(translator)

	fmt.Println("\n=== PackageTranslator (en-US) ===")
	fmt.Println(authPT.T("greeting"))
	fmt.Println(authPT.TF("welcome", "Alice"))
	fmt.Println(storagePT.T("greeting"))

	// Switch locale for the entire application.
	translator.SetLocale("es-ES")

	fmt.Println("\n=== PackageTranslator (es-ES) ===")
	fmt.Println(authPT.T("greeting"))
	fmt.Println(authPT.TF("welcome", "Carlos"))
	fmt.Println(storagePT.T("greeting"))
}
