# Formatting Example

Demonstrates how to use `TranslateWithArgs()` for dynamic content with format strings, including strings, integers, and floating-point numbers.

## What You'll Learn

- Using `TranslateWithArgs()` for parametrized translations
- Format string syntax (`%s`, `%d`, `%.2f`)
- Single and multiple argument formatting
- Numeric formatting for prices and quantities
- Maintaining consistent argument positions across locales

## Prerequisites

- Go 1.22 or later
- The i18n package installed

```bash
go get github.com/0verkilll/i18n
```

## File Structure

```
formatting/
├── main.go              # Example application
├── go.mod               # Go module definition
├── README.md            # This file
└── locales/
    ├── en-US.json       # English translations with format strings
    └── es-ES.json       # Spanish translations with format strings
```

## Running the Example

```bash
cd examples/formatting
go run main.go
```

## Expected Output

```
Welcome back, Alice!
Found 5 products in the catalog
Price: $99.99
¡Bienvenido de nuevo, Alice!
Se encontraron 5 productos en el catálogo
Precio: $99.99
```

## Code Walkthrough

### 1. Create the Translator

```go
translator, err := i18n.New(
    i18n.WithFileSystemLoader("./locales"),
    i18n.WithDefaultLocale("en-US"),
)
```

### 2. String Formatting

```go
username := "Alice"
welcome := translator.TranslateWithArgs("messages.welcome_user", username)
fmt.Println(welcome)  // Output: Welcome back, Alice!
```

The translation string uses `%s` as a placeholder:
```json
"welcome_user": "Welcome back, %s!"
```

### 3. Multiple Arguments

```go
itemCount := 5
message := translator.TranslateWithArgs("messages.items_found", itemCount, "products")
fmt.Println(message)  // Output: Found 5 products in the catalog
```

The translation uses multiple placeholders:
```json
"items_found": "Found %d %s in the catalog"
```

**Important:** Arguments are applied in order - first argument replaces first placeholder, second replaces second, etc.

### 4. Numeric Formatting

```go
price := 99.99
priceMsg := translator.TranslateWithArgs("messages.price", price)
fmt.Println(priceMsg)  // Output: Price: $99.99
```

The translation uses `%.2f` for two decimal places:
```json
"price": "Price: $%.2f"
```

### 5. Locale Switching with Arguments

```go
translator.SetLocale("es-ES")
welcome = translator.TranslateWithArgs("messages.welcome_user", username)
fmt.Println(welcome)  // Output: ¡Bienvenido de nuevo, Alice!
```

Same arguments work across locales - only the surrounding text changes.

## Translation File Structure

### English (`locales/en-US.json`)

```json
{
  "messages": {
    "welcome_user": "Welcome back, %s!",
    "items_found": "Found %d %s in the catalog",
    "price": "Price: $%.2f",
    "user_score": "%s scored %d points in %s",
    "time_remaining": "Time remaining: %d hours and %d minutes"
  }
}
```

### Spanish (`locales/es-ES.json`)

```json
{
  "messages": {
    "welcome_user": "¡Bienvenido de nuevo, %s!",
    "items_found": "Se encontraron %d %s en el catálogo",
    "price": "Precio: $%.2f",
    "user_score": "%s obtuvo %d puntos en %s",
    "time_remaining": "Tiempo restante: %d horas y %d minutos"
  }
}
```

## Format String Reference

The i18n package uses Go's `fmt.Sprintf` syntax:

| Specifier | Type | Example | Output |
|-----------|------|---------|--------|
| `%s` | String | `%s` with `"Alice"` | `Alice` |
| `%d` | Integer | `%d` with `42` | `42` |
| `%f` | Float | `%f` with `3.14159` | `3.141590` |
| `%.2f` | Float (2 decimals) | `%.2f` with `99.99` | `99.99` |
| `%.0f` | Float (no decimals) | `%.0f` with `99.99` | `100` |
| `%v` | Any value | `%v` with anything | Default format |
| `%%` | Literal % | `%%` | `%` |

### Width and Precision

| Specifier | Description | Example | Output |
|-----------|-------------|---------|--------|
| `%5d` | Width 5, right-aligned | `%5d` with `42` | `   42` |
| `%-5d` | Width 5, left-aligned | `%-5d` with `42` | `42   ` |
| `%05d` | Width 5, zero-padded | `%05d` with `42` | `00042` |
| `%8.2f` | Width 8, 2 decimals | `%8.2f` with `3.14` | `    3.14` |

## Common Patterns

### User Greetings

```go
// Translation: "Hello, %s! Welcome back."
greeting := translator.TranslateWithArgs("greeting.welcome", user.Name)
```

### Counts and Quantities

```go
// Translation: "You have %d unread messages"
message := translator.TranslateWithArgs("inbox.unread_count", count)
```

### Prices and Currency

```go
// Translation: "Total: $%.2f"
total := translator.TranslateWithArgs("cart.total", amount)
```

### Dates and Times

```go
// Translation: "Event on %s at %s"
dateTime := translator.TranslateWithArgs("event.datetime", dateStr, timeStr)
```

### Complex Messages

```go
// Translation: "%s scored %d points in %s"
score := translator.TranslateWithArgs(
    "game.score",
    playerName,    // %s - first string
    points,        // %d - integer
    gameName,      // %s - second string
)
```

## Argument Order Considerations

### Same Order Across Locales

When possible, keep argument order consistent:

**English:**
```json
"notification": "%s sent you %d files"
```

**Spanish:**
```json
"notification": "%s te envió %d archivos"
```

### When Order Must Change

Some languages require different word order. In these cases, you may need to design your translations carefully or use named placeholders in your own wrapper.

**English (Subject-Verb-Object):**
```
"Alice sent 5 files"
```

**Japanese (Subject-Object-Verb):**
```
"Alice が 5 ファイルを送りました"
```

## Security Note

The i18n package blocks the `%n` format specifier for security reasons. The `%n` specifier writes to memory and could be exploited in format string attacks.

```go
// This format string would be rejected:
// "Processed %n bytes"  // %n is blocked
```

## Error Handling

### Missing Arguments

If fewer arguments are provided than placeholders expect:

```go
// Translation: "Hello, %s! You have %d messages."
msg := translator.TranslateWithArgs("greeting", "Alice")
// Result: "Hello, Alice! You have %!d(MISSING) messages."
```

### Extra Arguments

Extra arguments are ignored:

```go
// Translation: "Hello, %s!"
msg := translator.TranslateWithArgs("greeting", "Alice", 42, "extra")
// Result: "Hello, Alice!"
```

### Type Mismatches

Go's fmt package handles type coercion gracefully:

```go
// Translation: "Count: %d"
msg := translator.TranslateWithArgs("count", "not a number")
// Result: "Count: %!d(string=not a number)"
```

## Best Practices

### 1. Use Meaningful Key Names

```json
{
  "messages": {
    "welcome_user": "Welcome, %s!",
    "items_in_cart": "You have %d items in your cart"
  }
}
```

### 2. Document Expected Arguments

Add comments in your code:

```go
// welcome_user expects: username (string)
welcome := translator.TranslateWithArgs("messages.welcome_user", user.Name)

// items_in_cart expects: count (int)
cartMsg := translator.TranslateWithArgs("messages.items_in_cart", cart.ItemCount)
```

### 3. Validate Translations

Ensure all locales have the same placeholders:

```json
// en-US.json
"items_found": "Found %d %s"

// es-ES.json
"items_found": "Se encontraron %d %s"  // Same placeholders
```

### 4. Handle Edge Cases

```go
// Handle zero, one, or many
count := len(items)
var key string
switch {
case count == 0:
    key = "messages.no_items"
case count == 1:
    key = "messages.one_item"
default:
    key = "messages.many_items"
}
msg := translator.TranslateWithArgs(key, count)
```

## Testing Format Strings

```go
func TestTranslationFormats(t *testing.T) {
    translator, _ := i18n.New(
        i18n.WithFileSystemLoader("./locales"),
        i18n.WithDefaultLocale("en-US"),
    )

    tests := []struct {
        key      string
        args     []interface{}
        expected string
    }{
        {"messages.welcome_user", []interface{}{"Alice"}, "Welcome back, Alice!"},
        {"messages.items_found", []interface{}{5, "products"}, "Found 5 products in the catalog"},
        {"messages.price", []interface{}{99.99}, "Price: $99.99"},
    }

    for _, tt := range tests {
        result := translator.TranslateWithArgs(tt.key, tt.args...)
        if result != tt.expected {
            t.Errorf("TranslateWithArgs(%q, %v) = %q; want %q",
                tt.key, tt.args, result, tt.expected)
        }
    }
}
```

## Next Steps

- Try the [basic example](../basic/) for simple key lookups
- Try the [embedded example](../embedded/) for single-binary deployments
- Try the [web-app example](../web-app/) for HTTP integration patterns
