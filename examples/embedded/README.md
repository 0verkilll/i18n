# Embedded Example

Demonstrates how to embed translation files directly into your Go binary using `//go:embed`, enabling single-binary deployments without external translation files.

## What You'll Learn

- Using Go's `embed` package with i18n
- Creating an embedded filesystem loader
- Building self-contained binaries with bundled translations
- Best practices for production deployments

## Prerequisites

- Go 1.22 or later (embed package requires Go 1.16+)
- The i18n package installed

```bash
go get github.com/0verkilll/i18n
```

## File Structure

```
embedded/
├── main.go              # Example application with embed directive
├── go.mod               # Go module definition
├── README.md            # This file
└── locales/
    ├── en-US.json       # English translations (embedded)
    └── fr-FR.json       # French translations (embedded)
```

## Running the Example

```bash
cd examples/embedded
go run main.go
```

## Expected Output

```
My Awesome App
A simple example of using embedded translations
Mon Application Géniale
Un exemple simple d'utilisation des traductions intégrées
```

## Code Walkthrough

### 1. Import the embed Package

```go
import (
    "embed"
    "fmt"
    "log"

    "github.com/0verkilll/i18n"
)
```

The `embed` package is part of Go's standard library and enables embedding files at compile time.

### 2. Declare the Embedded Filesystem

```go
//go:embed locales/*.json
var translationsFS embed.FS
```

This directive tells the Go compiler to:
- Embed all `.json` files from the `locales/` directory
- Store them in the `translationsFS` variable as an `embed.FS`
- Include them in the compiled binary

**Important:** The `//go:embed` comment must:
- Be directly above the variable declaration
- Have no space between `//` and `go:embed`
- Use a valid glob pattern

### 3. Create the Embedded Loader

```go
loader := i18n.NewEmbedFSLoader(translationsFS, "locales")
```

`NewEmbedFSLoader` creates a translation loader that reads from the embedded filesystem:
- First argument: the embedded filesystem (`embed.FS`)
- Second argument: the directory prefix within the embedded FS

### 4. Initialize the Translator

```go
translator, err := i18n.New(
    i18n.WithLoader(loader),
    i18n.WithDefaultLocale("en-US"),
)
```

Pass the embedded loader using `WithLoader()` instead of `WithFileSystemLoader()`.

### 5. Use Translations

```go
fmt.Println(translator.Translate("app.name"))
fmt.Println(translator.Translate("app.description"))

translator.SetLocale("fr-FR")
fmt.Println(translator.Translate("app.name"))
```

Usage is identical to the file system loader - the embedding is transparent to the rest of your code.

## Translation File Structure

### English (`locales/en-US.json`)

```json
{
  "app": {
    "name": "My Awesome App",
    "description": "A simple example of using embedded translations",
    "version": "1.0.0"
  },
  "menu": {
    "home": "Home",
    "about": "About",
    "contact": "Contact"
  }
}
```

### French (`locales/fr-FR.json`)

```json
{
  "app": {
    "name": "Mon Application Géniale",
    "description": "Un exemple simple d'utilisation des traductions intégrées",
    "version": "1.0.0"
  },
  "menu": {
    "home": "Accueil",
    "about": "À propos",
    "contact": "Contact"
  }
}
```

## Building a Single Binary

### Build the Binary

```bash
cd examples/embedded
go build -o myapp main.go
```

### Verify Embedded Files

The resulting `myapp` binary contains all translations:

```bash
./myapp
# Works without any external files!
```

### Move and Test

```bash
mv myapp /tmp/
cd /tmp
./myapp
# Still works - translations are embedded in the binary
```

## Benefits of Embedded Translations

| Benefit | Description |
|---------|-------------|
| **Single Binary** | Distribute one file instead of binary + translation files |
| **No Missing Files** | Translations can't be accidentally deleted or misconfigured |
| **Atomic Updates** | Update translations by deploying new binary |
| **Security** | Translation files can't be tampered with at runtime |
| **Simpler Deployment** | No need to manage file paths in production |

## Embed Patterns

### Embed All JSON Files

```go
//go:embed locales/*.json
var translationsFS embed.FS
```

### Embed Specific Files

```go
//go:embed locales/en-US.json locales/fr-FR.json
var translationsFS embed.FS
```

### Embed Entire Directory (Including Subdirectories)

```go
//go:embed locales
var translationsFS embed.FS
```

### Multiple Embed Directives

```go
//go:embed locales/*.json
//go:embed templates/*.html
var assetsFS embed.FS
```

## Production Patterns

### Conditional Loading (Development vs Production)

```go
func createTranslator(isDevelopment bool) (*i18n.Translator, error) {
    if isDevelopment {
        // Load from filesystem for hot-reload during development
        return i18n.New(
            i18n.WithFileSystemLoader("./locales"),
            i18n.WithDefaultLocale("en-US"),
        )
    }

    // Use embedded files in production
    loader := i18n.NewEmbedFSLoader(translationsFS, "locales")
    return i18n.New(
        i18n.WithLoader(loader),
        i18n.WithDefaultLocale("en-US"),
    )
}
```

### Docker Deployments

With embedded translations, your Dockerfile becomes simpler:

```dockerfile
FROM golang:1.22 AS builder
WORKDIR /app
COPY . .
RUN go build -o /myapp ./cmd/server

FROM scratch
COPY --from=builder /myapp /myapp
ENTRYPOINT ["/myapp"]
```

No need to copy translation files - they're in the binary!

## Troubleshooting

### "pattern locales/*.json: no matching files found"

- Ensure the `locales/` directory exists relative to the Go file with the embed directive
- Check that JSON files exist in the directory
- Verify the glob pattern matches your file structure

### Translations Not Found at Runtime

- Verify the directory prefix in `NewEmbedFSLoader` matches your embed pattern
- If you used `//go:embed locales/*.json`, the prefix should be `"locales"`
- If you used `//go:embed *.json` from within locales/, the prefix should be `""`

### Build Errors with Embed

- Ensure Go 1.16+ (embed package was introduced in Go 1.16)
- The embed directive must be in the same package as the variable
- The variable must be at package level (not inside a function)

## Key Concepts

### Compile-Time Embedding

Files are read and embedded during `go build`, not at runtime. This means:
- Changes to JSON files require recompilation
- Binary size increases with translation file sizes
- No filesystem access needed at runtime

### embed.FS is Read-Only

The embedded filesystem implements `fs.FS` interface but is read-only:
- You can read files
- You cannot write or modify files
- Perfect for translations which should be immutable

## Binary Size Considerations

Embedded files increase binary size:

```bash
# Without embedls -la myapp-no-embed
# -rwxr-xr-x 1 user user 2.1M myapp-no-embed

# With embed
ls -la myapp-embed
# -rwxr-xr-x 1 user user 2.2M myapp-embed
```

For most translation files (a few KB each), the size impact is negligible.

## Next Steps

- Try the [basic example](../basic/) if you need file system loading
- Try the [formatting example](../formatting/) to learn about dynamic content
- Try the [web-app example](../web-app/) for HTTP integration with embedded files
