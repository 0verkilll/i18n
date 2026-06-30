# Basic Example

A minimal example demonstrating core i18n functionality using the file system loader.

## What You'll Learn

- Creating a translator with file system loader
- Simple key lookups with `Translate()`
- Nested key access using dot notation
- Checking if translation keys exist with `HasKey()`
- Switching locales at runtime
- Getting the current locale

## Prerequisites

- Go 1.22 or later
- The i18n package installed

```bash
go get github.com/0verkilll/i18n
```

## File Structure

```
basic/
├── main.go              # Example application
├── go.mod               # Go module definition
├── README.md            # This file
└── locales/
    ├── en-US.json       # English translations
    └── es-ES.json       # Spanish translations
```

## Running the Example

```bash
cd examples/basic
go run main.go
```

## Expected Output

```
Hello!
This field is required
Goodbye!
¡Hola!
Current locale: es-ES
```

## Code Walkthrough

### 1. Create the Translator

```go
translator, err := i18n.New(
    i18n.WithFileSystemLoader("./locales"),
    i18n.WithDefaultLocale("en-US"),
)
```

This creates a new translator instance that:
- Loads translation files from the `./locales` directory
- Uses `en-US` as the default locale
- Automatically detects available locales from JSON files

### 2. Simple Key Lookup

```go
greeting := translator.Translate("greeting")
fmt.Println(greeting)  // Output: Hello!
```

The `Translate()` method looks up a key in the current locale's translations and returns the corresponding string.

### 3. Nested Key Access

```go
errorMsg := translator.Translate("errors.validation.required")
fmt.Println(errorMsg)  // Output: This field is required
```

Use dot notation to access nested keys in your translation files. This maps to:

```json
{
  "errors": {
    "validation": {
      "required": "This field is required"
    }
  }
}
```

### 4. Check Key Existence

```go
if translator.HasKey("farewell") {
    fmt.Println(translator.Translate("farewell"))
}
```

Use `HasKey()` to check if a translation exists before attempting to translate. This is useful for:
- Conditional rendering
- Graceful fallback handling
- Debugging missing translations

### 5. Switch Locale at Runtime

```go
translator.SetLocale("es-ES")
greeting = translator.Translate("greeting")
fmt.Println(greeting)  // Output: ¡Hola!
```

Change the active locale dynamically. All subsequent `Translate()` calls will use the new locale.

### 6. Get Current Locale

```go
fmt.Printf("Current locale: %s\n", translator.GetLocale())
// Output: Current locale: es-ES
```

Retrieve the currently active locale.

## Translation File Structure

### English (`locales/en-US.json`)

```json
{
  "greeting": "Hello!",
  "farewell": "Goodbye!",
  "welcome": "Welcome to our application",
  "errors": {
    "validation": {
      "required": "This field is required",
      "email": "Please enter a valid email address",
      "password": "Password must be at least 8 characters"
    },
    "network": {
      "timeout": "Request timed out",
      "offline": "You are currently offline"
    }
  },
  "actions": {
    "submit": "Submit",
    "cancel": "Cancel",
    "save": "Save",
    "delete": "Delete"
  }
}
```

### Spanish (`locales/es-ES.json`)

```json
{
  "greeting": "¡Hola!",
  "farewell": "¡Adiós!",
  "welcome": "Bienvenido a nuestra aplicación",
  "errors": {
    "validation": {
      "required": "Este campo es obligatorio",
      "email": "Por favor ingrese un correo electrónico válido",
      "password": "La contraseña debe tener al menos 8 caracteres"
    },
    "network": {
      "timeout": "Se agotó el tiempo de espera",
      "offline": "Actualmente estás desconectado"
    }
  },
  "actions": {
    "submit": "Enviar",
    "cancel": "Cancelar",
    "save": "Guardar",
    "delete": "Eliminar"
  }
}
```

## Key Concepts

### Locale Naming Convention

Locales follow the BCP 47 format: `language-REGION`
- `en-US` - English (United States)
- `es-ES` - Spanish (Spain)
- `fr-FR` - French (France)

### File Naming

Translation files must be named `{locale}.json` (e.g., `en-US.json`).

### Nested Keys

Organize translations hierarchically for better maintainability:

| Dot Notation Path | JSON Structure |
|-------------------|----------------|
| `greeting` | `{"greeting": "..."}` |
| `errors.network.timeout` | `{"errors": {"network": {"timeout": "..."}}}` |
| `actions.submit` | `{"actions": {"submit": "..."}}` |

## Common Use Cases

### User Interface Labels

```go
submitLabel := translator.Translate("actions.submit")
cancelLabel := translator.Translate("actions.cancel")
```

### Error Messages

```go
if err != nil {
    errorMsg := translator.Translate("errors.validation.required")
    // Display to user
}
```

### Feature Flags Based on Translation Availability

```go
if translator.HasKey("features.beta.title") {
    // Show beta feature
}
```

## Troubleshooting

### "translation not found" Returned

- Verify the key exists in your JSON file
- Check for typos in the key path
- Ensure the JSON file is valid (no syntax errors)
- Confirm the locale file exists in the `locales` directory

### Locale Not Loading

- Verify the file is named correctly (e.g., `en-US.json`, not `en_US.json`)
- Check file permissions
- Ensure the `locales` directory path is correct relative to where you run the program

## Next Steps

- Try the [embedded example](../embedded/) to learn about single-binary deployments
- Try the [formatting example](../formatting/) to learn about dynamic content with arguments
- Try the [web-app example](../web-app/) for HTTP integration patterns
