// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

// Example: Web application with shared translator and middleware-based locale detection.
package main

import (
	"embed"
	"fmt"
	"log"
	"net/http"

	"github.com/0verkilll/i18n"
	"github.com/0verkilll/i18n/middleware"
)

//go:embed locales/*.json
var translationsFS embed.FS

func main() {
	// Create a shared translator once at startup using embedded translations.
	loader := i18n.NewEmbedFSLoader(translationsFS, "locales")
	t, err := i18n.New(
		i18n.WithLoader(loader),
		i18n.WithDefaultLocale("en-US"),
	)
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tr := middleware.TranslatorFromContext(r.Context())
		if tr == nil {
			http.Error(w, "translator not found", http.StatusInternalServerError)
			return
		}

		locale := tr.GetLocale()
		title := tr.Translate("page.home.title")
		subtitle := tr.Translate("page.home.subtitle")
		description := tr.Translate("page.home.description")

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="%s">
<head>
    <meta charset="UTF-8">
    <title>%s</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 800px; margin: 50px auto; padding: 20px; }
        h1 { color: #333; }
        .info { background: #f0f0f0; padding: 15px; border-radius: 5px; margin-top: 20px; }
    </style>
</head>
<body>
    <h1>%s</h1>
    <h2>%s</h2>
    <p>%s</p>
    <div class="info">
        <p><strong>Current locale:</strong> %s</p>
        <p><strong>Try different languages:</strong></p>
        <ul>
            <li><a href="?lang=en-US">English</a></li>
            <li><a href="?lang=es-ES">Espanol</a></li>
            <li><a href="?lang=fr-FR">Francais</a></li>
        </ul>
    </div>
</body>
</html>`, locale, title, title, subtitle, description, locale)
	})

	mux.HandleFunc("/api/greeting", func(w http.ResponseWriter, r *http.Request) {
		tr := middleware.TranslatorFromContext(r.Context())
		if tr == nil {
			http.Error(w, "translator not found", http.StatusInternalServerError)
			return
		}

		greeting := tr.Translate("api.greeting")
		locale := tr.GetLocale()

		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"locale": "%s", "message": "%s"}`, locale, greeting)
	})

	// Wrap all routes with the locale-detection middleware.
	// The middleware reads Accept-Language, sets the locale on the shared
	// translator, and injects a ContextTranslator into the request context.
	handler := middleware.LocaleFromRequest(t)(mux)

	fmt.Println("Server starting on http://localhost:8080")
	fmt.Println("Try visiting:")
	fmt.Println("  http://localhost:8080/")
	fmt.Println("  curl -H 'Accept-Language: es-ES' http://localhost:8080/api/greeting")
	fmt.Println("  curl -H 'Accept-Language: fr-FR' http://localhost:8080/api/greeting")
	log.Fatal(http.ListenAndServe(":8080", handler))
}
