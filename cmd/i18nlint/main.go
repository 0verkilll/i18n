// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

// Command i18nlint statically analyzes Go source files and JSON translation
// files to report missing keys, unused keys, format string argument mismatches,
// and namespace violations.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/0verkilll/i18n/cmd/i18nlint/internal/analyzer"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

// parseFlags parses command-line flags and returns the flag set.
func parseFlags(args []string) (*flag.FlagSet, error) {
	fs := flag.NewFlagSet("i18nlint", flag.ContinueOnError)
	fs.String("locales", "", "comma-separated list of locale directory or file paths")
	fs.String("format", analyzer.FormatText, "output format: text or json")
	fs.String("exclude", "", "comma-separated glob patterns for directories/files to skip")
	fs.String("namespace", "", "restrict analysis to a specific namespace prefix")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	return fs, nil
}

// buildConfig constructs an analyzer.Config from the parsed flags.
func buildConfig(fs *flag.FlagSet) (analyzer.Config, error) {
	format := fs.Lookup("format").Value.String()
	if format != analyzer.FormatText && format != analyzer.FormatJSON {
		return analyzer.Config{}, fmt.Errorf("unsupported format %q (use \"text\" or \"json\")", format)
	}

	sourceDirs := fs.Args()
	if len(sourceDirs) == 0 {
		sourceDirs = []string{"./"}
	}

	localesStr := fs.Lookup("locales").Value.String()
	var localePaths []string
	if localesStr != "" {
		localePaths = strings.Split(localesStr, ",")
	}
	if len(localePaths) == 0 {
		return analyzer.Config{}, fmt.Errorf("no locale paths specified (use -locales flag)")
	}

	var excludePatterns []string
	if excludeStr := fs.Lookup("exclude").Value.String(); excludeStr != "" {
		excludePatterns = strings.Split(excludeStr, ",")
	}

	return analyzer.Config{
		SourceDirs:      sourceDirs,
		LocalePaths:     localePaths,
		ExcludePatterns: excludePatterns,
		Namespace:       fs.Lookup("namespace").Value.String(),
		Format:          format,
	}, nil
}

// reportIssues writes issues in the requested format. Returns an error on I/O failure.
func reportIssues(format string, issues []analyzer.Issue) error {
	switch format {
	case analyzer.FormatJSON:
		return analyzer.ReportJSON(os.Stdout, issues)
	default:
		return analyzer.ReportText(os.Stdout, issues)
	}
}

// run parses flags, executes the analysis, and returns an exit code.
// Exit codes: 0 = clean, 1 = issues found, 2 = fatal error.
func run(args []string) int {
	fs, err := parseFlags(args)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 2
	}

	cfg, err := buildConfig(fs)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 2
	}

	issues, err := analyzer.Run(cfg)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 2
	}

	if err := reportIssues(cfg.Format, issues); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 2
	}

	if len(issues) > 0 {
		return 1
	}
	return 0
}
