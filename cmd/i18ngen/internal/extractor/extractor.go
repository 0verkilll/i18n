// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

package extractor

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// KeyKind classifies how a translation key was extracted from source code.
type KeyKind int

const (
	// KindStandard is a plain Translate, T, Has, HasKey, or Key call.
	KindStandard KeyKind = iota
	// KindFormat is a TranslateWithArgs or TF call (records arg count).
	KindFormat
	// KindPlural is a TranslatePlural call.
	KindPlural
	// KindGender is a TranslateGender call.
	KindGender
	// KindStructField is a Msg.Get().FieldName pattern.
	KindStructField
	// KindWithDefault is a TD call or WithDefaults map entry.
	KindWithDefault
)

// ExtractedKey represents a translation key found in source code.
type ExtractedKey struct {
	Key          string  // the translation key
	DefaultValue string  // default value if provided (from TD or WithDefaults)
	File         string  // source file path
	Pattern      string  // which pattern matched (e.g., "Translate", "T", "WithDefaults")
	Line         int     // line number
	Col          int     // column number
	ArgCount     int     // number of variadic arguments; -1 if not a format call
	Kind         KeyKind // distinguishes extraction source
}

// targetMethods is the set of method names that the extractor detects.
var targetMethods = map[string]bool{
	"Translate":         true,
	"TranslateWithArgs": true,
	"TranslatePlural":   true,
	"TranslateGender":   true,
	"HasKey":            true,
	"T":                 true,
	"TF":                true,
	"TD":                true,
	"Has":               true,
	"Key":               true,
}

// formatMethods are methods that accept variadic format arguments.
var formatMethods = map[string]bool{
	"TranslateWithArgs": true,
	"TF":                true,
}

// pluralMethods detect plural translation calls.
var pluralMethods = map[string]bool{
	"TranslatePlural": true,
}

// genderMethods detect gender translation calls.
var genderMethods = map[string]bool{
	"TranslateGender": true,
}

// defaultMethods extract a second argument as a default value.
var defaultMethods = map[string]bool{
	"TD": true,
}

// Extract scans Go source files in the given directories and returns all
// translation keys found. It deduplicates and sorts the result by key.
func Extract(dirs, excludePatterns []string) ([]ExtractedKey, error) {
	var all []ExtractedKey

	for _, dir := range dirs {
		keys, err := scanDirRecursive(dir, excludePatterns)
		if err != nil {
			return nil, fmt.Errorf("extracting keys from %q: %w", dir, err)
		}
		all = append(all, keys...)
	}

	all = deduplicateKeys(all)
	sort.Slice(all, func(i, j int) bool {
		return all[i].Key < all[j].Key
	})

	return all, nil
}

// ExtractFromSource parses a Go source string and extracts translation keys.
// This is used for testing without file I/O.
func ExtractFromSource(filename, src string) ([]ExtractedKey, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parsing source %q: %w", filename, err)
	}
	return extractKeysFromFile(fset, file), nil
}

// scanDirRecursive walks a directory tree and extracts keys from Go files.
func scanDirRecursive(root string, excludePatterns []string) ([]ExtractedKey, error) {
	var keys []ExtractedKey

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return nil
		}
		if shouldExclude(path, root, excludePatterns) {
			return filepath.SkipDir
		}
		dirKeys, parseErr := scanSingleDir(path)
		if parseErr != nil {
			return fmt.Errorf("scanning directory %q: %w", path, parseErr)
		}
		keys = append(keys, dirKeys...)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return keys, nil
}

// scanSingleDir reads Go source files from a directory and extracts keys.
func scanSingleDir(dir string) ([]ExtractedKey, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading directory %q: %w", dir, err)
	}

	fset := token.NewFileSet()
	var keys []ExtractedKey

	for _, entry := range entries {
		if entry.IsDir() || !isGoSource(entry.Name()) {
			continue
		}
		filePath := filepath.Join(dir, entry.Name())
		file, parseErr := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
		if parseErr != nil {
			continue
		}
		keys = append(keys, extractKeysFromFile(fset, file)...)
	}

	return keys, nil
}

// extractKeysFromFile walks a file AST and finds translation key references.
func extractKeysFromFile(fset *token.FileSet, file *ast.File) []ExtractedKey {
	var keys []ExtractedKey

	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		keys = appendCallKeys(keys, fset, call)
		return true
	})

	keys = appendStructFieldKeys(keys, fset, file)
	return keys
}

// appendCallKeys inspects a call expression and appends any extracted keys.
func appendCallKeys(keys []ExtractedKey, fset *token.FileSet, call *ast.CallExpr) []ExtractedKey {
	funcName := extractFuncName(call)

	if funcName == "WithDefaults" {
		return appendWithDefaultsKeys(keys, fset, call)
	}

	if funcName == "" || !targetMethods[funcName] {
		return keys
	}

	return appendMethodKey(keys, fset, call, funcName)
}

// appendMethodKey extracts a key from a standard method call (Translate, T, etc.).
func appendMethodKey(keys []ExtractedKey, fset *token.FileSet, call *ast.CallExpr, funcName string) []ExtractedKey {
	if len(call.Args) == 0 {
		return keys
	}

	lit, ok := call.Args[0].(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return keys
	}

	keyValue, err := strconv.Unquote(lit.Value)
	if err != nil {
		return keys
	}

	pos := fset.Position(lit.Pos())
	ek := ExtractedKey{
		Key:      keyValue,
		File:     pos.Filename,
		Pattern:  funcName,
		Line:     pos.Line,
		Col:      pos.Column,
		ArgCount: -1,
		Kind:     classifyMethod(funcName),
	}

	if formatMethods[funcName] {
		ek.ArgCount = len(call.Args) - 1
	}

	if defaultMethods[funcName] {
		ek.DefaultValue = extractDefaultArg(call)
	}

	keys = append(keys, ek)
	return keys
}

// classifyMethod returns the KeyKind for a given method name.
func classifyMethod(name string) KeyKind {
	switch {
	case formatMethods[name]:
		return KindFormat
	case pluralMethods[name]:
		return KindPlural
	case genderMethods[name]:
		return KindGender
	case defaultMethods[name]:
		return KindWithDefault
	default:
		return KindStandard
	}
}

// extractDefaultArg extracts the second string literal argument from a call.
func extractDefaultArg(call *ast.CallExpr) string {
	if len(call.Args) < 2 {
		return ""
	}
	lit, ok := call.Args[1].(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return ""
	}
	val, err := strconv.Unquote(lit.Value)
	if err != nil {
		return ""
	}
	return val
}

// appendWithDefaultsKeys extracts key-value pairs from a WithDefaults(map[string]string{...}) call.
func appendWithDefaultsKeys(keys []ExtractedKey, fset *token.FileSet, call *ast.CallExpr) []ExtractedKey {
	if len(call.Args) == 0 {
		return keys
	}

	comp, ok := call.Args[0].(*ast.CompositeLit)
	if !ok {
		return keys
	}

	for _, elt := range comp.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		ek := extractMapEntry(fset, kv)
		if ek != nil {
			keys = append(keys, *ek)
		}
	}

	return keys
}

// extractMapEntry extracts a key-value pair from a map literal element.
func extractMapEntry(fset *token.FileSet, kv *ast.KeyValueExpr) *ExtractedKey {
	keyLit, ok := kv.Key.(*ast.BasicLit)
	if !ok || keyLit.Kind != token.STRING {
		return nil
	}
	valLit, ok := kv.Value.(*ast.BasicLit)
	if !ok || valLit.Kind != token.STRING {
		return nil
	}

	keyStr, err := strconv.Unquote(keyLit.Value)
	if err != nil {
		return nil
	}
	valStr, err := strconv.Unquote(valLit.Value)
	if err != nil {
		return nil
	}

	pos := fset.Position(keyLit.Pos())
	return &ExtractedKey{
		Key:          keyStr,
		DefaultValue: valStr,
		File:         pos.Filename,
		Pattern:      "WithDefaults",
		Line:         pos.Line,
		Col:          pos.Column,
		ArgCount:     -1,
		Kind:         KindWithDefault,
	}
}

// appendStructFieldKeys detects Msg.Get().FieldName patterns in the AST.
func appendStructFieldKeys(keys []ExtractedKey, fset *token.FileSet, file *ast.File) []ExtractedKey {
	ast.Inspect(file, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if !isGetCallReceiver(sel.X) {
			return true
		}
		pos := fset.Position(sel.Sel.Pos())
		keys = append(keys, ExtractedKey{
			Key:      sel.Sel.Name,
			File:     pos.Filename,
			Pattern:  "StructField",
			Line:     pos.Line,
			Col:      pos.Column,
			ArgCount: -1,
			Kind:     KindStructField,
		})
		return true
	})
	return keys
}

// isGetCallReceiver checks if an expression is a .Get() call on a receiver.
func isGetCallReceiver(expr ast.Expr) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	return sel.Sel.Name == "Get"
}

// extractFuncName extracts the function or method name from a call expression.
func extractFuncName(call *ast.CallExpr) string {
	switch fn := call.Fun.(type) {
	case *ast.SelectorExpr:
		return fn.Sel.Name
	case *ast.Ident:
		return fn.Name
	default:
		return ""
	}
}

// isGoSource reports whether the file name is a non-test Go source file.
func isGoSource(name string) bool {
	return strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go")
}

// shouldExclude checks whether a directory should be skipped.
func shouldExclude(path, root string, patterns []string) bool {
	if path == root {
		return false
	}

	dirName := filepath.Base(path)
	rel, err := filepath.Rel(root, path)
	if err != nil {
		rel = dirName
	}

	for _, pattern := range patterns {
		if matchPattern(pattern, dirName) || matchPattern(pattern, rel) {
			return true
		}
	}

	return false
}

// matchPattern wraps filepath.Match and treats errors as non-matches.
func matchPattern(pattern, name string) bool {
	matched, err := filepath.Match(pattern, name)
	return err == nil && matched
}

// deduplicateKeys removes duplicate keys, keeping the first occurrence.
func deduplicateKeys(keys []ExtractedKey) []ExtractedKey {
	seen := make(map[string]bool, len(keys))
	result := make([]ExtractedKey, 0, len(keys))

	for _, k := range keys {
		dedupeKey := k.Key + "|" + fmt.Sprintf("%d", k.Kind)
		if seen[dedupeKey] {
			continue
		}
		seen[dedupeKey] = true
		result = append(result, k)
	}

	return result
}
