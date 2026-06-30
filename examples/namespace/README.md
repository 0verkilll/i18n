# Namespace Example

Demonstrates `Namespace` and `PackageTranslator` usage with two simulated
packages (`auth` and `storage`), each calling `T()` and `TF()` through their
own namespace while sharing a single `Translator` instance.

## What This Shows

- Creating namespaces with `NewNamespace` for key prefixing
- Creating per-package translators with `NewPackageTranslator` and `WithDefaults`
- Wiring multiple packages to one shared translator via `SetTranslator`
- Switching locale at runtime affects all packages simultaneously

## Run

```bash
cd examples/namespace
go run main.go
```

## Expected Output

```
=== Namespace (en-US) ===
Hello from auth
Welcome, Alice!
Hello from storage
Welcome to storage, Bob!

=== PackageTranslator (en-US) ===
Hello from auth
Welcome, Alice!
Hello from storage

=== PackageTranslator (es-ES) ===
Hola desde auth
Bienvenido, Carlos!
Hola desde storage
```
