# i18n/middleware

HTTP middleware for Accept-Language locale detection.

This is a separate sub-package to avoid pulling `net/http` into WASM and TinyGo builds.

## Install

    go get github.com/0verkilll/i18n/middleware

## Usage

    import (
        i18n "github.com/0verkilll/i18n"
        "github.com/0verkilll/i18n/middleware"
    )

    t, _ := i18n.NewWithFS("locales", "en-US")

    handler := middleware.LocaleFromRequest(t)(yourMux)

    // In handlers:
    func handleRequest(w http.ResponseWriter, r *http.Request) {
        tr := middleware.TranslatorFromContext(r.Context())
        fmt.Fprintln(w, tr.Translate("greeting"))
    }

## API

- `LocaleFromRequest(t *i18n.Translator) func(http.Handler) http.Handler` — Wraps a handler to detect locale from Accept-Language header and store a ContextTranslator in the request context.
- `TranslatorFromContext(ctx context.Context) *i18n.ContextTranslator` — Retrieves the ContextTranslator from the request context. Returns nil if not present.
