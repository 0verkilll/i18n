// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

package i18n_test

import (
	"embed"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/0verkilll/i18n"
)

//go:embed testdata/locales/*.json
var exampleEmbeddedLocales embed.FS

// Example demonstrates basic translation usage with filesystem loader
func Example() {
	// Create temporary directory for this example
	tmpDir := setupExampleLocales()
	defer func() { _ = os.RemoveAll(tmpDir) }() //nolint:errcheck // Cleanup in test, error is non-critical

	// Create translator with filesystem loader
	translator, err := i18n.New(
		i18n.WithFileSystemLoader(tmpDir),
		i18n.WithDefaultLocale("en-US"),
	)
	if err != nil {
		_ = os.RemoveAll(tmpDir) //nolint:errcheck // Best-effort cleanup before Fatal
		log.Fatal(err)           //nolint:gocritic // Cleanup already done before Fatal
	}

	// Simple translation
	greeting := translator.Translate("greeting")
	fmt.Println(greeting)

	// Nested key translation
	title := translator.Translate("user.profile.title")
	fmt.Println(title)

	// Output:
	// Hello
	// User Profile
}

// Example_withArguments demonstrates translation with format string arguments
func Example_withArguments() {
	tmpDir := setupExampleLocales()
	defer func() { _ = os.RemoveAll(tmpDir) }() //nolint:errcheck // Cleanup in test, error is non-critical

	translator, err := i18n.New(
		i18n.WithFileSystemLoader(tmpDir),
		i18n.WithDefaultLocale("en-US"),
	)
	if err != nil {
		_ = os.RemoveAll(tmpDir) //nolint:errcheck // Best-effort cleanup before Fatal
		log.Fatal(err)           //nolint:gocritic // Cleanup already done before Fatal
	}

	// Translation with string argument
	welcome := translator.TranslateWithArgs("welcome", "Alice")
	fmt.Println(welcome)

	// Translation with integer argument
	count := translator.TranslateWithArgs("items_count", 5)
	fmt.Println(count)

	// Translation with multiple arguments
	message := translator.TranslateWithArgs("multi", "Bob", 3)
	fmt.Println(message)

	// Output:
	// Welcome, Alice!
	// You have 5 items
	// Bob has 3 items
}

// Example_localeSwitching demonstrates changing locales at runtime
func Example_localeSwitching() {
	tmpDir := setupExampleLocales()
	defer func() { _ = os.RemoveAll(tmpDir) }() //nolint:errcheck // Cleanup in test, error is non-critical

	translator, err := i18n.New(
		i18n.WithFileSystemLoader(tmpDir),
		i18n.WithDefaultLocale("en-US"),
	)
	if err != nil {
		_ = os.RemoveAll(tmpDir) //nolint:errcheck // Best-effort cleanup before Fatal
		log.Fatal(err)           //nolint:gocritic // Cleanup already done before Fatal
	}

	// English greeting
	fmt.Println(translator.GetLocale()+":", translator.Translate("greeting"))

	// Switch to Spanish
	translator.SetLocale("es-ES")
	fmt.Println(translator.GetLocale()+":", translator.Translate("greeting"))

	// Switch back to English
	translator.SetLocale("en-US")
	fmt.Println(translator.GetLocale()+":", translator.Translate("greeting"))

	// Output:
	// en-US: Hello
	// es-ES: Hola
	// en-US: Hello
}

// Example_fallbackChain demonstrates locale fallback behavior
func Example_fallbackChain() {
	tmpDir := setupExampleLocalesWithFallback()
	defer func() { _ = os.RemoveAll(tmpDir) }() //nolint:errcheck // Cleanup in test, error is non-critical

	translator, err := i18n.New(
		i18n.WithFileSystemLoader(tmpDir),
		i18n.WithDefaultLocale("en-US"),
	)
	if err != nil {
		_ = os.RemoveAll(tmpDir) //nolint:errcheck // Best-effort cleanup before Fatal
		log.Fatal(err)           //nolint:gocritic // Cleanup already done before Fatal
	}

	// Switch to Mexican Spanish (has partial translations)
	translator.SetLocale("es-MX")

	// This exists in es-MX
	fmt.Println("greeting:", translator.Translate("greeting"))

	// This doesn't exist in es-MX, falls back to es-ES
	fmt.Println("farewell:", translator.Translate("farewell"))

	// This doesn't exist in es-MX or es-ES, falls back to en-US
	fmt.Println("welcome:", translator.Translate("welcome"))

	// Output:
	// greeting: Hola (Mexico)
	// farewell: Adios
	// welcome: Welcome
}

// Example_missingKey demonstrates behavior when a translation key is not found
func Example_missingKey() {
	tmpDir := setupExampleLocales()
	defer func() { _ = os.RemoveAll(tmpDir) }() //nolint:errcheck // Cleanup in test, error is non-critical

	translator, err := i18n.New(
		i18n.WithFileSystemLoader(tmpDir),
		i18n.WithDefaultLocale("en-US"),
	)
	if err != nil {
		_ = os.RemoveAll(tmpDir) //nolint:errcheck // Best-effort cleanup before Fatal
		log.Fatal(err)           //nolint:gocritic // Cleanup already done before Fatal
	}

	// Missing key returns the key itself
	result := translator.Translate("nonexistent.key")
	fmt.Println(result)

	// Check if key exists
	if !translator.HasKey("nonexistent.key") {
		fmt.Println("Key does not exist")
	}

	// Output:
	// nonexistent.key
	// Key does not exist
}

// Example_embeddedFS demonstrates using embedded filesystem for translations
func Example_embeddedFS() {
	// Create loader with embedded filesystem
	loader := i18n.NewEmbedFSLoader(exampleEmbeddedLocales, "testdata/locales")

	translator, err := i18n.New(
		i18n.WithLoader(loader),
		i18n.WithDefaultLocale("en-US"),
	)
	if err != nil {
		log.Fatal(err) //nolint:gocritic // Cleanup already done before Fatal
	}

	// Use translations from embedded files
	greeting := translator.Translate("greeting")
	fmt.Println(greeting)

	// Output:
	// Hello
}

// Example_customComponents demonstrates using custom loader and parser
func Example_customComponents() {
	tmpDir := setupExampleLocales()
	defer func() { _ = os.RemoveAll(tmpDir) }() //nolint:errcheck // Cleanup in test, error is non-critical

	// Create custom components
	loader := i18n.NewFileSystemLoader(tmpDir)
	parser := i18n.NewJSONParser()
	resolver := i18n.NewDefaultKeyResolver()

	translator, err := i18n.New(
		i18n.WithLoader(loader),
		i18n.WithParser(parser),
		i18n.WithResolver(resolver),
		i18n.WithDefaultLocale("en-US"),
	)
	if err != nil {
		_ = os.RemoveAll(tmpDir) //nolint:errcheck // Best-effort cleanup before Fatal
		log.Fatal(err)           //nolint:gocritic // Cleanup already done before Fatal
	}

	result := translator.Translate("greeting")
	fmt.Println(result)

	// Output:
	// Hello
}

// Example_localeNormalization demonstrates automatic locale normalization
func Example_localeNormalization() {
	tmpDir := setupExampleLocales()
	defer func() { _ = os.RemoveAll(tmpDir) }() //nolint:errcheck // Cleanup in test, error is non-critical

	translator, err := i18n.New(
		i18n.WithFileSystemLoader(tmpDir),
		i18n.WithDefaultLocale("en-US"),
	)
	if err != nil {
		_ = os.RemoveAll(tmpDir) //nolint:errcheck // Best-effort cleanup before Fatal
		log.Fatal(err)           //nolint:gocritic // Cleanup already done before Fatal
	}

	// Underscore is normalized to hyphen
	translator.SetLocale("en_US")
	fmt.Println(translator.GetLocale())

	// Encoding suffix is removed
	translator.SetLocale("es_ES.UTF-8")
	fmt.Println(translator.GetLocale())

	// POSIX locale normalized to en-US
	translator.SetLocale("POSIX")
	fmt.Println(translator.GetLocale())

	// Output:
	// en-US
	// es-ES
	// en-US
}

// Example_hasKey demonstrates checking for key existence
func Example_hasKey() {
	tmpDir := setupExampleLocales()
	defer func() { _ = os.RemoveAll(tmpDir) }() //nolint:errcheck // Cleanup in test, error is non-critical

	translator, err := i18n.New(
		i18n.WithFileSystemLoader(tmpDir),
		i18n.WithDefaultLocale("en-US"),
	)
	if err != nil {
		_ = os.RemoveAll(tmpDir) //nolint:errcheck // Best-effort cleanup before Fatal
		log.Fatal(err)           //nolint:gocritic // Cleanup already done before Fatal
	}

	// Check simple key
	if translator.HasKey("greeting") {
		fmt.Println("greeting exists")
	}

	// Check nested key
	if translator.HasKey("user.profile.title") {
		fmt.Println("user.profile.title exists")
	}

	// Check non-existent key
	if !translator.HasKey("nonexistent") {
		fmt.Println("nonexistent does not exist")
	}

	// Output:
	// greeting exists
	// user.profile.title exists
	// nonexistent does not exist
}

// Example_registerParser demonstrates registering a custom parser for a new format.
//
// External modules (e.g., i18n-toml, i18n-yaml) follow this pattern:
//  1. Implement TranslationParser
//  2. Call RegisterParser in init()
//  3. Application code activates via blank import
func Example_registerParser() {
	// Define a mock TOML parser implementing TranslationParser
	parser := &exampleTOMLParser{}

	// Register it for the .toml extension
	err := i18n.RegisterParser(".toml", parser)
	if err != nil {
		log.Fatal(err)
	}

	// Verify .toml is now registered
	formats := i18n.RegisteredFormats()
	for _, f := range formats {
		if f == ".toml" {
			fmt.Println(".toml is registered")
		}
	}

	// Retrieve it back from the registry
	got, err := i18n.GetParser(".toml")
	if err != nil {
		log.Fatal(err)
	}
	if got != nil {
		fmt.Println("parser retrieved successfully")
	}

	// Output:
	// .toml is registered
	// parser retrieved successfully
}

// Example_withRegisteredParser demonstrates using WithRegisteredParser to
// configure a Translator with a registry-resolved parser.
func Example_withRegisteredParser() {
	tmpDir := setupExampleLocales()
	defer func() { _ = os.RemoveAll(tmpDir) }() //nolint:errcheck // Cleanup in test, error is non-critical

	// Use WithRegisteredParser to pull the built-in JSON parser from the registry
	translator, err := i18n.New(
		i18n.WithFileSystemLoader(tmpDir),
		i18n.WithRegisteredParser(".json"),
		i18n.WithDefaultLocale("en-US"),
	)
	if err != nil {
		_ = os.RemoveAll(tmpDir) //nolint:errcheck // Best-effort cleanup before Fatal
		log.Fatal(err)           //nolint:gocritic // Cleanup already done before Fatal
	}

	fmt.Println(translator.Translate("greeting"))

	// Output:
	// Hello
}

// ExampleNamespace demonstrates Namespace key prefixing for package-scoped translations.
func ExampleNamespace() {
	tmpDir := setupExampleNamespaceLocales()
	defer func() { _ = os.RemoveAll(tmpDir) }() //nolint:errcheck // Cleanup in test, error is non-critical

	translator, err := i18n.New(
		i18n.WithFileSystemLoader(tmpDir),
		i18n.WithDefaultLocale("en-US"),
	)
	if err != nil {
		_ = os.RemoveAll(tmpDir) //nolint:errcheck // Best-effort cleanup before Fatal
		log.Fatal(err)           //nolint:gocritic // Cleanup already done before Fatal
	}

	// Create a namespace for "mypackage"
	ns, err := i18n.NewNamespace("mypackage", translator)
	if err != nil {
		_ = os.RemoveAll(tmpDir) //nolint:errcheck // Best-effort cleanup before Fatal
		log.Fatal(err)           //nolint:gocritic // Cleanup already done before Fatal
	}

	// T translates with the namespace prefix
	fmt.Println(ns.T("greeting"))

	// TF translates with format arguments
	fmt.Println(ns.TF("welcome", "Alice"))

	// TD returns default if key not found
	fmt.Println(ns.TD("missing", "Default Value"))

	// Has checks key existence
	fmt.Println(ns.Has("greeting"))

	// Key returns the full namespaced key
	fmt.Println(ns.Key("greeting"))

	// Output:
	// Hello from mypackage
	// Welcome, Alice!
	// Default Value
	// true
	// mypackage.greeting
}

// ExamplePackageTranslator demonstrates per-package translation with defaults.
func ExamplePackageTranslator() {
	tmpDir := setupExampleNamespaceLocales()
	defer func() { _ = os.RemoveAll(tmpDir) }() //nolint:errcheck // Cleanup in test, error is non-critical

	// Create a PackageTranslator with hardcoded defaults
	pt := i18n.NewPackageTranslator("mypackage", i18n.WithDefaults(map[string]string{
		"greeting": "Hi (default)",
		"welcome":  "Welcome, %s! (default)",
	}))

	// Before wiring up a translator, defaults are used
	fmt.Println(pt.T("greeting"))

	// Create and set the translator
	translator, err := i18n.New(
		i18n.WithFileSystemLoader(tmpDir),
		i18n.WithDefaultLocale("en-US"),
	)
	if err != nil {
		_ = os.RemoveAll(tmpDir) //nolint:errcheck // Best-effort cleanup before Fatal
		log.Fatal(err)           //nolint:gocritic // Cleanup already done before Fatal
	}
	pt.SetTranslator(translator)

	// Now translations come from locale files
	fmt.Println(pt.T("greeting"))
	fmt.Println(pt.TF("welcome", "Bob"))

	// Output:
	// Hi (default)
	// Hello from mypackage
	// Welcome, Bob!
}

// ExampleRegistryLoader demonstrates using RegistryLoader with build-tag-selected locales.
func ExampleRegistryLoader() {
	// RegistryLoader reads from the global locale registry, which is populated
	// by build-tag-selected locale files at init time. For this example, we
	// use the embedded testdata loader as a stand-in, since the registry
	// requires -tags locale_all to be populated.

	loader := i18n.NewEmbedFSLoader(exampleEmbeddedLocales, "testdata/locales")

	translator, err := i18n.New(
		i18n.WithLoader(loader),
		i18n.WithDefaultLocale("en-US"),
	)
	if err != nil {
		log.Fatal(err) //nolint:gocritic // Fatal terminates the program
	}

	// Translate using the loaded locale data
	fmt.Println(translator.Translate("greeting"))
	fmt.Println(translator.Translate("farewell"))

	// RegisteredLocales shows which locales are in the build-tag registry
	// (may be empty without -tags locale_all)
	fmt.Println("registry available:", i18n.NewRegistryLoader() != nil)

	// Output:
	// Hello
	// Goodbye
	// registry available: true
}

// exampleTOMLParser is a mock parser demonstrating the external module pattern.
// A real TOML parser would use github.com/BurntSushi/toml or similar.
// External parsers should enforce equivalent size and nesting depth limits
// (see MaxJSONSize and MaxJSONDepth) to maintain security parity.
type exampleTOMLParser struct{}

func (p *exampleTOMLParser) Parse(_ []byte) (map[string]interface{}, error) {
	return map[string]interface{}{"example": "toml-value"}, nil
}

// setupExampleLocales creates example locale files for basic translation examples.
func setupExampleLocales() string {
	tmpDir := os.TempDir() + "/i18n-example"
	_ = os.MkdirAll(tmpDir, 0o755) //nolint:errcheck // Error intentionally ignored in test helper

	// English translations
	enUS := []byte(`{
		"greeting": "Hello",
		"welcome": "Welcome, %s!",
		"farewell": "Goodbye",
		"user": {
			"profile": {
				"title": "User Profile"
			}
		},
		"items_count": "You have %d items",
		"multi": "%s has %d items"
	}`)
	_ = os.WriteFile(filepath.Join(tmpDir, "en-US.json"), enUS, 0o644) //nolint:errcheck // Error intentionally ignored in test helper

	// Spanish translations
	esES := []byte(`{
		"greeting": "Hola",
		"farewell": "Adios"
	}`)
	_ = os.WriteFile(filepath.Join(tmpDir, "es-ES.json"), esES, 0o644) //nolint:errcheck // Error intentionally ignored in test helper

	return tmpDir
}

// setupExampleLocalesWithFallback creates example locale files with partial
// translations to demonstrate fallback chain behavior.
func setupExampleLocalesWithFallback() string {
	tmpDir := os.TempDir() + "/i18n-example-fallback"
	_ = os.MkdirAll(tmpDir, 0o755) //nolint:errcheck // Error intentionally ignored in test helper

	// English (complete)
	enUS := []byte(`{
		"greeting": "Hello",
		"welcome": "Welcome",
		"farewell": "Goodbye"
	}`)
	_ = os.WriteFile(filepath.Join(tmpDir, "en-US.json"), enUS, 0o644) //nolint:errcheck // Error intentionally ignored in test helper

	// Spanish Spain (partial)
	esES := []byte(`{
		"greeting": "Hola",
		"farewell": "Adios"
	}`)
	_ = os.WriteFile(filepath.Join(tmpDir, "es-ES.json"), esES, 0o644) //nolint:errcheck // Error intentionally ignored in test helper

	// Mexican Spanish (minimal - for fallback testing)
	esMX := []byte(`{
		"greeting": "Hola (Mexico)"
	}`)
	_ = os.WriteFile(filepath.Join(tmpDir, "es-MX.json"), esMX, 0o644) //nolint:errcheck // Error intentionally ignored in test helper

	return tmpDir
}

// setupExampleNamespaceLocales creates locale files with namespace-prefixed keys
// for Namespace and PackageTranslator examples.
func setupExampleNamespaceLocales() string {
	tmpDir := os.TempDir() + "/i18n-example-ns"
	_ = os.MkdirAll(tmpDir, 0o755) //nolint:errcheck // Error intentionally ignored in test helper

	enUS := []byte(`{
		"mypackage": {
			"greeting": "Hello from mypackage",
			"welcome": "Welcome, %s!"
		}
	}`)
	_ = os.WriteFile(filepath.Join(tmpDir, "en-US.json"), enUS, 0o644) //nolint:errcheck // Error intentionally ignored in test helper

	return tmpDir
}
