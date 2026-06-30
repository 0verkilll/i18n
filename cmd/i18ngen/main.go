// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

// Command i18ngen extracts translation keys from Go source files, generates
// locale file templates, diffs extracted keys against existing locale files,
// and scaffolds per-package i18n.go integration files.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/0verkilll/i18n/cmd/i18ngen/internal/differ"
	"github.com/0verkilll/i18n/cmd/i18ngen/internal/extractor"
	"github.com/0verkilll/i18n/cmd/i18ngen/internal/generator"
)

const (
	exitSuccess = 0
	exitIssues  = 1
	exitError   = 2
)

const usageText = `Usage: i18ngen <command> [flags] [paths...]

Commands:
  extract    Extract translation keys from Go source files
  generate   Generate locale file templates from extracted keys
  diff       Compare source keys against locale file
  init       Scaffold i18n.go for a package
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run dispatches to subcommand handlers and returns an exit code.
// Exit codes: 0 = success, 1 = issues found (diff), 2 = fatal error.
func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		writeMsg(stderr, usageText)
		return exitError
	}

	switch args[0] {
	case "extract":
		return runExtract(args[1:], stdout, stderr)
	case "generate":
		return runGenerate(args[1:], stdout, stderr)
	case "diff":
		return runDiff(args[1:], stdout, stderr)
	case "init":
		return runInit(args[1:], stdout, stderr)
	default:
		writeMsg(stderr, fmt.Sprintf("unknown command %q\n%s", args[0], usageText))
		return exitError
	}
}

// runExtract handles the extract subcommand.
func runExtract(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("extract", flag.ContinueOnError)
	fs.SetOutput(stderr)
	exclude := fs.String("exclude", "", "comma-separated glob patterns to skip")

	if err := fs.Parse(args); err != nil {
		return exitError
	}

	dirs := positionalDirs(fs)
	keys, err := extractor.Extract(dirs, splitCSV(*exclude))
	if err != nil {
		writeMsg(stderr, fmt.Sprintf("error: %v\n", err))
		return exitError
	}

	printExtractedKeys(stdout, keys)
	return exitSuccess
}

// runGenerate handles the generate subcommand.
func runGenerate(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("generate", flag.ContinueOnError)
	fs.SetOutput(stderr)
	format := fs.String("format", "", "output format: json, struct, binary, or init")
	packageName := fs.String("package", "main", "Go package name for struct/init output")
	outputFile := fs.String("o", "", "output file path (default: stdout)")
	exclude := fs.String("exclude", "", "comma-separated glob patterns to skip")

	if err := fs.Parse(args); err != nil {
		return exitError
	}

	if *format == "" {
		writeMsg(stderr, "error: -format flag is required (json, struct, binary, or init)\n")
		return exitError
	}

	dirs := positionalDirs(fs)
	keys, err := extractor.Extract(dirs, splitCSV(*exclude))
	if err != nil {
		writeMsg(stderr, fmt.Sprintf("error: %v\n", err))
		return exitError
	}

	genFn, selErr := selectGenerator(*format, *packageName, keys)
	if selErr != nil {
		writeMsg(stderr, fmt.Sprintf("error: %v\n", selErr))
		return exitError
	}

	return writeOutput(genFn, *outputFile, stdout, stderr)
}

// selectGenerator returns a write function for the given format.
func selectGenerator(format, packageName string, keys []extractor.ExtractedKey) (func(io.Writer) error, error) {
	switch format {
	case "json":
		return func(w io.Writer) error { return generator.GenerateJSON(keys, w) }, nil
	case "struct":
		return func(w io.Writer) error { return generator.GenerateStruct(keys, packageName, w) }, nil
	case "binary":
		return func(w io.Writer) error { return generator.GenerateBinary(keys, w) }, nil
	case "init":
		return func(w io.Writer) error { return generator.GenerateInit(keys, packageName, w) }, nil
	default:
		return nil, fmt.Errorf("unsupported format %q (use json, struct, binary, or init)", format)
	}
}

// runDiff handles the diff subcommand.
func runDiff(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("diff", flag.ContinueOnError)
	fs.SetOutput(stderr)
	localeFile := fs.String("locale", "", "path to JSON locale file (required)")
	format := fs.String("format", "text", "output format: text or json")
	exclude := fs.String("exclude", "", "comma-separated glob patterns to skip")

	if err := fs.Parse(args); err != nil {
		return exitError
	}

	if *localeFile == "" {
		writeMsg(stderr, "error: -locale flag is required\n")
		return exitError
	}

	dirs := positionalDirs(fs)
	keys, err := extractor.Extract(dirs, splitCSV(*exclude))
	if err != nil {
		writeMsg(stderr, fmt.Sprintf("error: %v\n", err))
		return exitError
	}

	return writeDiff(keys, *localeFile, *format, stdout, stderr)
}

// writeDiff compares source keys against a locale file and writes the result.
func writeDiff(keys []extractor.ExtractedKey, localeFile, format string, stdout, stderr io.Writer) int {
	result, err := differ.Diff(keys, localeFile)
	if err != nil {
		writeMsg(stderr, fmt.Sprintf("error: %v\n", err))
		return exitError
	}

	if err := formatDiff(result, format, stdout); err != nil {
		writeMsg(stderr, fmt.Sprintf("error: %v\n", err))
		return exitError
	}

	if result.HasIssues() {
		return exitIssues
	}
	return exitSuccess
}

// formatDiff writes the diff result in the requested format.
func formatDiff(result *differ.DiffResult, format string, w io.Writer) error {
	switch format {
	case "json":
		return differ.FormatJSON(result, w)
	default:
		return differ.FormatText(result, w)
	}
}

// runInit handles the init subcommand.
func runInit(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(stderr)
	packageName := fs.String("package", "", "Go package name (required)")
	outputFile := fs.String("o", "", "output file path (default: stdout)")
	exclude := fs.String("exclude", "", "comma-separated glob patterns to skip")

	if err := fs.Parse(args); err != nil {
		return exitError
	}

	if *packageName == "" {
		writeMsg(stderr, "error: -package flag is required\n")
		return exitError
	}

	dirs := positionalDirs(fs)
	keys, err := extractor.Extract(dirs, splitCSV(*exclude))
	if err != nil {
		writeMsg(stderr, fmt.Sprintf("error: %v\n", err))
		return exitError
	}

	genFn := func(w io.Writer) error {
		return generator.GenerateInit(keys, *packageName, w)
	}

	return writeOutput(genFn, *outputFile, stdout, stderr)
}

// writeOutput writes generated content to a file or stdout.
func writeOutput(genFn func(io.Writer) error, outputFile string, stdout, stderr io.Writer) int {
	if outputFile != "" {
		if err := generator.WriteToFile(outputFile, genFn); err != nil {
			writeMsg(stderr, fmt.Sprintf("error: %v\n", err))
			return exitError
		}
		return exitSuccess
	}

	if err := genFn(stdout); err != nil {
		writeMsg(stderr, fmt.Sprintf("error: %v\n", err))
		return exitError
	}
	return exitSuccess
}

// printExtractedKeys writes sorted keys to the writer. TD calls output
// "key\tdefault_value" (tab-separated). Struct field refs are prefixed
// with "struct:".
func printExtractedKeys(w io.Writer, keys []extractor.ExtractedKey) {
	sorted := make([]extractor.ExtractedKey, len(keys))
	copy(sorted, keys)
	sort.Slice(sorted, func(i, j int) bool {
		return formatKeyLine(sorted[i]) < formatKeyLine(sorted[j])
	})

	for _, k := range sorted {
		writeMsg(w, formatKeyLine(k)+"\n")
	}
}

// formatKeyLine returns the display line for a single extracted key.
func formatKeyLine(k extractor.ExtractedKey) string {
	switch k.Kind {
	case extractor.KindStructField:
		return "struct:" + k.Key
	case extractor.KindWithDefault:
		if k.DefaultValue != "" {
			return k.Key + "\t" + k.DefaultValue
		}
		return k.Key
	default:
		return k.Key
	}
}

// writeMsg writes a message to a writer. Write errors are intentionally
// discarded because callers write to stderr/stdout just before returning
// an exit code, and there is no recovery path for a broken output stream.
func writeMsg(w io.Writer, msg string) {
	_, _ = io.WriteString(w, msg) //nolint:errcheck,gosec // unrecoverable I/O
}

// positionalDirs returns positional directory arguments from the flag set,
// defaulting to "./" if none are provided.
func positionalDirs(fs *flag.FlagSet) []string {
	dirs := fs.Args()
	if len(dirs) == 0 {
		return []string{"./"}
	}
	return dirs
}

// splitCSV splits a comma-separated string into a slice, trimming whitespace
// and filtering out empty entries.
func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
