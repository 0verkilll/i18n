# Build-Tag Locale Selection Example

Demonstrates building with Go build tags to control which locale data is
compiled into the binary. This is especially useful for WASM and TinyGo builds
where binary size matters.

## What This Shows

- Using `RegistryLoader` to load translations from the build-tag registry
- Querying registered locales at runtime with `RegisteredLocales()`
- Build commands for single locale, multiple locales, and all locales
- WASM and TinyGo build targets

## Run

```bash
cd examples/build-tags

# All locales (development)
go run -tags locale_all main.go

# Single locale
go run -tags locale_en_us main.go

# Multiple locales
go run -tags "locale_en_us,locale_es_es" main.go
```

## Build Targets (Makefile)

```bash
make build-single        # go build -tags locale_en_us
make build-multi         # go build -tags "locale_en_us,locale_es_es"
make build-all           # go build -tags locale_all
make build-wasm          # GOOS=js GOARCH=wasm go build -tags "locale_en_us"
make build-tinygo-wasm   # tinygo build -tags "locale_en_us" -target wasm
make run                 # go run -tags locale_all main.go
```

## Expected Output (with -tags locale_all)

```
Registered locales: [en-US es-ES pt-BR]
[en-US] Hello
[es-ES] Hola
[pt-BR] Ola
```
