# Web App Example

A complete example demonstrating i18n integration in an HTTP web application, including locale detection from query parameters and `Accept-Language` headers.

## What You'll Learn

- Integrating i18n with Go's `net/http` package
- Detecting user locale from query parameters
- Parsing the `Accept-Language` HTTP header
- Creating per-request translator instances
- Building multilingual HTML pages
- Creating multilingual JSON API endpoints
- Combining embedded files with HTTP handlers

## Prerequisites

- Go 1.22 or later
- The i18n package installed
- A web browser for testing

```bash
go get github.com/0verkilll/i18n
```

## File Structure

```
web-app/
├── main.go              # HTTP server with i18n integration
├── go.mod               # Go module definition
├── README.md            # This file
└── locales/
    ├── en-US.json       # English translations
    ├── es-ES.json       # Spanish translations
    └── fr-FR.json       # French translations
```

## Running the Example

```bash
cd examples/web-app
go run main.go
```

You should see:

```
Server starting on http://localhost:8080
Try visiting:
  http://localhost:8080/
  http://localhost:8080/?lang=es-ES
  http://localhost:8080/?lang=fr-FR
```

## Testing the Application

### In Your Browser

1. **Default (English):** http://localhost:8080/
2. **Spanish:** http://localhost:8080/?lang=es-ES
3. **French:** http://localhost:8080/?lang=fr-FR

### Using curl

```bash
# Default locale
curl http://localhost:8080/

# Query parameter
curl "http://localhost:8080/?lang=es-ES"

# Accept-Language header
curl -H "Accept-Language: fr-FR" http://localhost:8080/

# API endpoint
curl http://localhost:8080/api/greeting
curl "http://localhost:8080/api/greeting?lang=es-ES"
```

## Expected Output

### HTML Page (English)

```
Welcome to i18n Demo
Internationalization Made Easy

This is a simple demonstration of the i18n package in a web application.
The content on this page is automatically translated based on your
language preference.

Current locale: en-US
Try different languages:
• English
• Español
• Français
```

### API Response (Spanish)

```json
{"locale": "es-ES", "message": "¡Hola desde la API!"}
```

## Code Walkthrough

### 1. Embed Translation Files

```go
//go:embed locales/*.json
var translationsFS embed.FS
```

Translations are embedded for single-binary deployment.

### 2. Initialize Global Translator

```go
var translator *i18n.Translator

func main() {
    loader := i18n.NewEmbedFSLoader(translationsFS, "locales")

    var err error
    translator, err = i18n.New(
        i18n.WithLoader(loader),
        i18n.WithDefaultLocale("en-US"),
    )
    if err != nil {
        log.Fatal(err)
    }
    // ...
}
```

### 3. Register HTTP Handlers

```go
http.HandleFunc("/", homeHandler)
http.HandleFunc("/api/greeting", greetingHandler)

log.Fatal(http.ListenAndServe(":8080", nil))
```

### 4. Locale Detection Function

```go
func getLocaleFromRequest(r *http.Request) string {
    // Priority 1: Query parameter
    if lang := r.URL.Query().Get("lang"); lang != "" {
        return lang
    }

    // Priority 2: Accept-Language header
    acceptLang := r.Header.Get("Accept-Language")
    if acceptLang != "" {
        parts := strings.Split(acceptLang, ",")
        if len(parts) > 0 {
            locale := strings.TrimSpace(strings.Split(parts[0], ";")[0])
            return locale
        }
    }

    // Priority 3: Default
    return "en-US"
}
```

### 5. HTML Handler

```go
func homeHandler(w http.ResponseWriter, r *http.Request) {
    locale := getLocaleFromRequest(r)

    // Create per-request translator
    reqTranslator, _ := i18n.New(
        i18n.WithLoader(i18n.NewEmbedFSLoader(translationsFS, "locales")),
        i18n.WithDefaultLocale(locale),
    )

    // Get translations
    title := reqTranslator.Translate("page.home.title")
    subtitle := reqTranslator.Translate("page.home.subtitle")
    description := reqTranslator.Translate("page.home.description")

    // Render HTML
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="%s">
<head>
    <title>%s</title>
</head>
<body>
    <h1>%s</h1>
    <h2>%s</h2>
    <p>%s</p>
    <p>Current locale: %s</p>
</body>
</html>`, locale, title, title, subtitle, description, locale)
}
```

### 6. API Handler

```go
func greetingHandler(w http.ResponseWriter, r *http.Request) {
    locale := getLocaleFromRequest(r)

    reqTranslator, _ := i18n.New(
        i18n.WithLoader(i18n.NewEmbedFSLoader(translationsFS, "locales")),
        i18n.WithDefaultLocale(locale),
    )

    greeting := reqTranslator.Translate("api.greeting")

    w.Header().Set("Content-Type", "application/json")
    fmt.Fprintf(w, `{"locale": "%s", "message": "%s"}`, locale, greeting)
}
```

## Translation File Structure

### English (`locales/en-US.json`)

```json
{
  "page": {
    "home": {
      "title": "Welcome to i18n Demo",
      "subtitle": "Internationalization Made Easy",
      "description": "This is a simple demonstration of the i18n package in a web application. The content on this page is automatically translated based on your language preference."
    }
  },
  "api": {
    "greeting": "Hello from the API!"
  }
}
```

### Spanish (`locales/es-ES.json`)

```json
{
  "page": {
    "home": {
      "title": "Bienvenido a la Demo de i18n",
      "subtitle": "Internacionalización Hecha Fácil",
      "description": "Esta es una demostración simple del paquete i18n en una aplicación web. El contenido de esta página se traduce automáticamente según su preferencia de idioma."
    }
  },
  "api": {
    "greeting": "¡Hola desde la API!"
  }
}
```

### French (`locales/fr-FR.json`)

```json
{
  "page": {
    "home": {
      "title": "Bienvenue à la Démo i18n",
      "subtitle": "L'Internationalisation Rendue Facile",
      "description": "Ceci est une démonstration simple du package i18n dans une application web. Le contenu de cette page est automatiquement traduit en fonction de votre préférence linguistique."
    }
  },
  "api": {
    "greeting": "Bonjour de l'API !"
  }
}
```

## Locale Detection Priority

The example implements a common priority order:

| Priority | Source | Example |
|----------|--------|---------|
| 1 | Query Parameter | `?lang=es-ES` |
| 2 | Accept-Language Header | `Accept-Language: fr-FR,fr;q=0.9` |
| 3 | Default | `en-US` |

### Accept-Language Header Format

Browsers send language preferences like:

```
Accept-Language: en-US,en;q=0.9,es;q=0.8,fr;q=0.7
```

This means:
- `en-US` - First preference (quality 1.0 implied)
- `en` - Second preference (quality 0.9)
- `es` - Third preference (quality 0.8)
- `fr` - Fourth preference (quality 0.7)

## Production Patterns

### Translator Pooling

For high-traffic applications, consider pooling translators:

```go
import "sync"

var translatorPool = sync.Pool{
    New: func() interface{} {
        loader := i18n.NewEmbedFSLoader(translationsFS, "locales")
        t, _ := i18n.New(
            i18n.WithLoader(loader),
            i18n.WithDefaultLocale("en-US"),
        )
        return t
    },
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
    locale := getLocaleFromRequest(r)

    t := translatorPool.Get().(*i18n.Translator)
    defer translatorPool.Put(t)

    t.SetLocale(locale)
    title := t.Translate("page.home.title")
    // ...
}
```

### Middleware Pattern

Create reusable locale middleware:

```go
type contextKey string

const localeKey contextKey = "locale"

func LocaleMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        locale := getLocaleFromRequest(r)
        ctx := context.WithValue(r.Context(), localeKey, locale)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

func getLocaleFromContext(ctx context.Context) string {
    if locale, ok := ctx.Value(localeKey).(string); ok {
        return locale
    }
    return "en-US"
}
```

### Cookie-Based Locale

Store user preference in cookies:

```go
func getLocaleFromRequest(r *http.Request) string {
    // Check query parameter first
    if lang := r.URL.Query().Get("lang"); lang != "" {
        return lang
    }

    // Check cookie
    if cookie, err := r.Cookie("locale"); err == nil {
        return cookie.Value
    }

    // Check Accept-Language header
    // ...

    return "en-US"
}

func setLocaleCookie(w http.ResponseWriter, locale string) {
    http.SetCookie(w, &http.Cookie{
        Name:     "locale",
        Value:    locale,
        Path:     "/",
        MaxAge:   365 * 24 * 60 * 60, // 1 year
        HttpOnly: true,
        Secure:   true,
        SameSite: http.SameSiteLaxMode,
    })
}
```

### Template Integration

Using Go templates with i18n:

```go
import "html/template"

var tmpl = template.Must(template.ParseFiles("templates/home.html"))

type PageData struct {
    Locale      string
    Title       string
    Subtitle    string
    Description string
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
    locale := getLocaleFromRequest(r)
    t := getTranslator(locale)

    data := PageData{
        Locale:      locale,
        Title:       t.Translate("page.home.title"),
        Subtitle:    t.Translate("page.home.subtitle"),
        Description: t.Translate("page.home.description"),
    }

    tmpl.Execute(w, data)
}
```

## Response Headers

Set appropriate headers for internationalized content:

```go
w.Header().Set("Content-Type", "text/html; charset=utf-8")
w.Header().Set("Content-Language", locale)
w.Header().Set("Vary", "Accept-Language")
```

The `Vary: Accept-Language` header tells caches that the response varies based on the `Accept-Language` request header.

## Testing

### Unit Test for Locale Detection

```go
func TestGetLocaleFromRequest(t *testing.T) {
    tests := []struct {
        name           string
        queryParam     string
        acceptLanguage string
        expected       string
    }{
        {"query param", "es-ES", "", "es-ES"},
        {"accept-language", "", "fr-FR,en;q=0.9", "fr-FR"},
        {"query takes priority", "de-DE", "fr-FR", "de-DE"},
        {"default fallback", "", "", "en-US"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            req := httptest.NewRequest("GET", "/", nil)
            if tt.queryParam != "" {
                q := req.URL.Query()
                q.Add("lang", tt.queryParam)
                req.URL.RawQuery = q.Encode()
            }
            if tt.acceptLanguage != "" {
                req.Header.Set("Accept-Language", tt.acceptLanguage)
            }

            got := getLocaleFromRequest(req)
            if got != tt.expected {
                t.Errorf("got %q, want %q", got, tt.expected)
            }
        })
    }
}
```

### Integration Test

```go
func TestHomeHandler(t *testing.T) {
    req := httptest.NewRequest("GET", "/?lang=es-ES", nil)
    rec := httptest.NewRecorder()

    homeHandler(rec, req)

    if rec.Code != http.StatusOK {
        t.Errorf("status = %d; want %d", rec.Code, http.StatusOK)
    }

    body := rec.Body.String()
    if !strings.Contains(body, "Bienvenido") {
        t.Error("response should contain Spanish greeting")
    }
}
```

## Common Issues

### Locale Not Detected

- Verify query parameter name matches (`lang` in this example)
- Check that Accept-Language header is being sent
- Ensure fallback locale exists

### Wrong Locale Selected

- Check locale detection priority
- Verify locale file names match expected format (`en-US.json`)
- Debug by logging the detected locale

### Translations Not Found

- Ensure translation keys match exactly
- Verify JSON files are valid
- Check that embedded files are included in build

## Next Steps

- Try the [basic example](../basic/) for simpler use cases
- Try the [embedded example](../embedded/) to learn about embedding
- Try the [formatting example](../formatting/) for dynamic content
- Check the main [i18n documentation](../../README.md) for advanced features
