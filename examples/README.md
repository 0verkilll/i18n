# i18n Examples

Examples demonstrating how to use the i18n package.

## Prerequisites

```bash
go get github.com/0verkilll/i18n
```

## Examples

### basic

Basic usage with file system loader: simple key lookups, nested keys with dot notation, locale switching.

```bash
cd examples/basic
go run main.go
```

### embedded

Using embedded files for single-binary deployments with `//go:embed`.

```bash
cd examples/embedded
go run main.go
```

### formatting

String formatting with `TranslateWithArgs` for dynamic content.

```bash
cd examples/formatting
go run main.go
```

### web-app

HTTP handler integration with locale detection from query parameters and Accept-Language header.

```bash
cd examples/web-app
go run main.go
# Visit http://localhost:8080
```

### namespace

Demonstrates `Namespace` and `PackageTranslator` usage with two simulated packages, each calling `T()` and `TF()` through their own namespace, sharing a single `Translator` instance.

```bash
cd examples/namespace
go run main.go
```

### build-tags

Demonstrates build-tag locale selection with `RegistryLoader`. Control which locale data is compiled into the binary using Go build tags. Includes a `Makefile` with targets for standard Go, TinyGo, and WASM builds.

```bash
cd examples/build-tags
go run -tags locale_all main.go
```

## Translation File Structure

```
example-name/
├── main.go
└── locales/
    ├── en-US.json
    ├── es-ES.json
    └── fr-FR.json
```

## Common Patterns

**File System Loader:**
```go
translator, err := i18n.New(
    i18n.WithFileSystemLoader("./locales"),
    i18n.WithDefaultLocale("en-US"),
)
```

**Embedded Files:**
```go
//go:embed locales/*.json
var translationsFS embed.FS

loader := i18n.NewEmbedFSLoader(translationsFS, "locales")
translator, err := i18n.New(
    i18n.WithLoader(loader),
    i18n.WithDefaultLocale("en-US"),
)
```

**Usage:**
```go
greeting := translator.Translate("greeting")
message := translator.TranslateWithArgs("welcome", "World")
translator.SetLocale("es-ES")
```
