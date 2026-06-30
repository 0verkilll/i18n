# Build-tag locale selection examples.
#
# Each target demonstrates a different build configuration.

# Standard Go: single locale
build-single:
	go build -tags locale_en_us -o build-tags .

# Standard Go: multiple locales
build-multi:
	go build -tags "locale_en_us,locale_es_es" -o build-tags .

# Standard Go: all locales (development convenience)
build-all:
	go build -tags locale_all -o build-tags .

# WASM target with locale tags
build-wasm:
	GOOS=js GOARCH=wasm go build -tags "locale_en_us" -o app.wasm .

# TinyGo WASM target with locale tags
build-tinygo-wasm:
	tinygo build -tags "locale_en_us" -target wasm -o app.wasm .

# Run with all locales
run:
	go run -tags locale_all main.go

clean:
	rm -f build-tags app.wasm

.PHONY: build-single build-multi build-all build-wasm build-tinygo-wasm run clean
