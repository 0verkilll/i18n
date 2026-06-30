// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

package analyzer

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// targetMethods is the set of function/method names that the scanner detects.
var targetMethods = map[string]bool{
	"Translate":         true,
	"TranslateWithArgs": true,
	"HasKey":            true,
	"T":                 true,
	"TF":                true,
	"Has":               true,
	"TD":                true,
	"Key":               true,
}

// formatMethods are the methods that accept variadic format arguments.
var formatMethods = map[string]bool{
	"TranslateWithArgs": true,
	"TF":                true,
}

// sourceKey represents a translation key extracted from Go source code.
type sourceKey struct {
	key      string
	file     string
	line     int
	col      int
	argCount int // Number of variadic arguments; -1 if not a format call.
}

// scanDirectories walks the given directories, parses Go files, and extracts
// translation key references. It applies exclude patterns to skip directories and files.
func scanDirectories(dirs, excludePatterns []string) ([]sourceKey, error) {
	var keys []sourceKey

	for _, dir := range dirs {
		collected, err := scanDirRecursive(dir, excludePatterns)
		if err != nil {
			return nil, err
		}
		keys = append(keys, collected...)
	}

	return keys, nil
}

// scanDirRecursive recursively walks a directory tree, parsing Go packages.
func scanDirRecursive(root string, excludePatterns []string) ([]sourceKey, error) {
	var keys []sourceKey

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
			return parseErr
		}
		keys = append(keys, dirKeys...)

		return nil
	})
	if err != nil {
		return nil, err
	}

	return keys, nil
}

// matchPattern wraps filepath.Match and treats errors as non-matches.
func matchPattern(pattern, name string) bool {
	matched, err := filepath.Match(pattern, name)
	return err == nil && matched
}

// shouldExclude checks whether a directory should be skipped based on exclude patterns.
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

// isGoSource reports whether the file name is a non-test Go source file.
func isGoSource(name string) bool {
	return strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go")
}

// scanSingleDir reads Go source files from a directory and extracts translation keys.
func scanSingleDir(dir string) ([]sourceKey, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	fset := token.NewFileSet()
	var keys []sourceKey

	for _, entry := range entries {
		if entry.IsDir() || !isGoSource(entry.Name()) {
			continue
		}

		filePath := filepath.Join(dir, entry.Name())
		file, parseErr := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
		if parseErr != nil {
			continue
		}

		fileKeys := extractKeysFromFile(fset, file)
		keys = append(keys, fileKeys...)
	}

	return keys, nil
}

// extractKeysFromFile walks a single file AST and finds translation key references.
func extractKeysFromFile(fset *token.FileSet, file *ast.File) []sourceKey {
	var keys []sourceKey

	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		funcName := extractFuncName(call)
		if funcName == "" || !targetMethods[funcName] {
			return true
		}

		if len(call.Args) == 0 {
			return true
		}

		lit, ok := call.Args[0].(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}

		keyValue, err := strconv.Unquote(lit.Value)
		if err != nil {
			return true
		}

		pos := fset.Position(lit.Pos())

		argCount := -1
		if formatMethods[funcName] {
			argCount = len(call.Args) - 1
		}

		keys = append(keys, sourceKey{
			key:      keyValue,
			file:     pos.Filename,
			line:     pos.Line,
			col:      pos.Column,
			argCount: argCount,
		})

		return true
	})

	return keys
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
