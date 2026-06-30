// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

// Package analyzer provides static analysis for i18n translation key usage.
// It scans Go source files and JSON translation files to detect missing keys,
// unused keys, format string mismatches, and namespace violations.
package analyzer

import (
	"fmt"
	"sort"
	"strings"
)

// Severity levels for diagnostic issues.
const (
	SeverityError   = "error"
	SeverityWarning = "warning"
)

// Issue codes for each category of diagnostic finding.
const (
	CodeMissingKey        = "missing-key"
	CodeUnusedKey         = "unused-key"
	CodeFormatMismatch    = "format-mismatch"
	CodeOrphanedNamespace = "orphaned-namespace"
	CodeUnknownNamespace  = "unknown-namespace"
)

// Config holds all settings for an analysis run.
type Config struct {
	Namespace       string
	Format          string
	SourceDirs      []string
	LocalePaths     []string
	ExcludePatterns []string
}

// Issue represents a single diagnostic finding.
type Issue struct {
	File     string `json:"file"`
	Severity string `json:"severity"`
	Code     string `json:"code"`
	Message  string `json:"message"`
	Key      string `json:"key"`
	Line     int    `json:"line"`
	Col      int    `json:"col"`
}

// countFormatSpecifiers counts the number of format specifiers in a string,
// using the same %%-aware logic as ValidateFormatString in validate.go.
// A specifier is any '%' followed by a non-'%' byte; escaped '%%' pairs are skipped.
func countFormatSpecifiers(s string) int {
	count := 0
	i := 0
	n := len(s)
	for i < n {
		if s[i] == '%' {
			if i+1 < n {
				if s[i+1] == '%' {
					// Escaped %%, skip both
					i += 2
					continue
				}
				// Format specifier found
				count++
				i += 2
				continue
			}
			// '%' at end of string, not a valid specifier
			i++
			continue
		}
		i++
	}
	return count
}

// sortIssues sorts issues deterministically by file path, then line, then column.
func sortIssues(issues []Issue) {
	sort.Slice(issues, func(i, j int) bool {
		if issues[i].File != issues[j].File {
			return issues[i].File < issues[j].File
		}
		if issues[i].Line != issues[j].Line {
			return issues[i].Line < issues[j].Line
		}
		return issues[i].Col < issues[j].Col
	})
}

// Run executes a full analysis pass: loads locale files, scans Go source,
// and compares key sets to produce diagnostic issues. It returns an error
// only for fatal configuration or I/O problems, not for analysis findings.
func Run(cfg Config) ([]Issue, error) {
	localeData, err := loadLocaleFiles(cfg.LocalePaths)
	if err != nil {
		return nil, fmt.Errorf("loading locale files: %w", err)
	}

	sourceKeys, err := scanDirectories(cfg.SourceDirs, cfg.ExcludePatterns)
	if err != nil {
		return nil, fmt.Errorf("scanning source directories: %w", err)
	}

	issues := make([]Issue, 0, len(sourceKeys))

	issues = append(issues, detectMissingKeys(sourceKeys, localeData, cfg.Namespace)...)
	issues = append(issues, detectUnusedKeys(sourceKeys, localeData)...)
	issues = append(issues, detectFormatMismatches(sourceKeys, localeData)...)
	issues = append(issues, detectNamespaceIssues(sourceKeys, localeData)...)

	sortIssues(issues)
	return issues, nil
}

// detectMissingKeys finds keys referenced in source but absent from all locale files.
func detectMissingKeys(sourceKeys []sourceKey, localeData *localeKeyData, namespace string) []Issue {
	allLocaleKeys := make(map[string]bool, len(localeData.keys))
	for key := range localeData.keys {
		allLocaleKeys[key] = true
	}

	issues := make([]Issue, 0, len(sourceKeys))
	for _, sk := range sourceKeys {
		if namespace != "" && !strings.HasPrefix(sk.key, namespace+".") && sk.key != namespace {
			continue
		}

		if !allLocaleKeys[sk.key] {
			issues = append(issues, Issue{
				File:     sk.file,
				Line:     sk.line,
				Col:      sk.col,
				Severity: SeverityError,
				Code:     CodeMissingKey,
				Message:  fmt.Sprintf("translation key %q not found in any locale file", sk.key),
				Key:      sk.key,
			})
		}
	}

	return issues
}

// detectUnusedKeys finds keys in locale files that are never referenced in source.
func detectUnusedKeys(sourceKeys []sourceKey, localeData *localeKeyData) []Issue {
	sourceKeySet := make(map[string]bool, len(sourceKeys))
	for _, sk := range sourceKeys {
		sourceKeySet[sk.key] = true
	}

	unusedKeys := make([]string, 0, len(localeData.keys))
	for key := range localeData.keys {
		if !sourceKeySet[key] {
			unusedKeys = append(unusedKeys, key)
		}
	}
	sort.Strings(unusedKeys)

	issues := make([]Issue, 0, len(unusedKeys))
	for _, key := range unusedKeys {
		files := localeData.keys[key]

		sortedFiles := make([]string, 0, len(files))
		for f := range files {
			sortedFiles = append(sortedFiles, f)
		}
		sort.Strings(sortedFiles)

		issues = append(issues, Issue{
			File:     sortedFiles[0],
			Severity: SeverityWarning,
			Code:     CodeUnusedKey,
			Message:  fmt.Sprintf("translation key %q is defined but never used in source", key),
			Key:      key,
		})
	}

	return issues
}

// detectFormatMismatches finds TranslateWithArgs/TF calls where the argument
// count does not match the number of format specifiers in the translation value.
func detectFormatMismatches(sourceKeys []sourceKey, localeData *localeKeyData) []Issue {
	issues := make([]Issue, 0, len(sourceKeys))

	for _, sk := range sourceKeys {
		if sk.argCount < 0 {
			continue
		}

		specCount, ok := localeData.specifierCounts[sk.key]
		if !ok {
			continue
		}

		if sk.argCount != specCount {
			issues = append(issues, Issue{
				File:     sk.file,
				Line:     sk.line,
				Col:      sk.col,
				Severity: SeverityError,
				Code:     CodeFormatMismatch,
				Message:  fmt.Sprintf("format mismatch for key %q: translation has %d specifier(s) but call passes %d argument(s)", sk.key, specCount, sk.argCount),
				Key:      sk.key,
			})
		}
	}

	return issues
}

// collectOrphanedNamespaces returns sorted namespace prefixes present in locale
// data but not referenced in source.
func collectOrphanedNamespaces(sourceNamespaces map[string]bool, localeNamespaces map[string]map[string]bool) []Issue {
	var orphanedNS []string
	for ns := range localeNamespaces {
		if !sourceNamespaces[ns] {
			orphanedNS = append(orphanedNS, ns)
		}
	}
	sort.Strings(orphanedNS)

	issues := make([]Issue, 0, len(orphanedNS))
	for _, ns := range orphanedNS {
		files := localeNamespaces[ns]
		sortedFiles := make([]string, 0, len(files))
		for f := range files {
			sortedFiles = append(sortedFiles, f)
		}
		sort.Strings(sortedFiles)

		issues = append(issues, Issue{
			File:     sortedFiles[0],
			Severity: SeverityWarning,
			Code:     CodeOrphanedNamespace,
			Message:  fmt.Sprintf("namespace %q exists in locale files but is never referenced in source", ns),
			Key:      ns,
		})
	}
	return issues
}

// collectUnknownNamespaces returns sorted namespace prefixes used in source
// but absent from locale data.
func collectUnknownNamespaces(sourceNamespaces map[string]bool, localeNamespaces map[string]map[string]bool) []Issue {
	var unknownNS []string
	for ns := range sourceNamespaces {
		if _, exists := localeNamespaces[ns]; !exists {
			unknownNS = append(unknownNS, ns)
		}
	}
	sort.Strings(unknownNS)

	issues := make([]Issue, 0, len(unknownNS))
	for _, ns := range unknownNS {
		issues = append(issues, Issue{
			Severity: SeverityWarning,
			Code:     CodeUnknownNamespace,
			Message:  fmt.Sprintf("namespace %q is used in source but not found in any locale file", ns),
			Key:      ns,
		})
	}
	return issues
}

// detectNamespaceIssues reports orphaned and unknown namespaces.
func detectNamespaceIssues(sourceKeys []sourceKey, localeData *localeKeyData) []Issue {
	sourceNamespaces := make(map[string]bool)
	for _, sk := range sourceKeys {
		parts := strings.SplitN(sk.key, ".", 2)
		sourceNamespaces[parts[0]] = true
	}

	localeNamespaces := localeData.topLevelKeys

	issues := collectOrphanedNamespaces(sourceNamespaces, localeNamespaces)
	issues = append(issues, collectUnknownNamespaces(sourceNamespaces, localeNamespaces)...)

	return issues
}
