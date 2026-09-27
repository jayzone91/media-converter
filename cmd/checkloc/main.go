package main

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const maxLines = 500

var checkedExtensions = map[string]struct{}{
	".css":   {},
	".go":    {},
	".html":  {},
	".js":    {},
	".jsx":   {},
	".scss":  {},
	".templ": {},
	".ts":    {},
	".tsx":   {},
}

var skippedDirectories = map[string]struct{}{
	".git":         {},
	".idea":        {},
	".vscode":      {},
	"build":        {},
	"dist":         {},
	"node_modules": {},
	"vendor":       {},
}

type violation struct {
	path  string
	lines int
}

func main() {
	violations, err := findViolations(".")
	if err != nil {
		fmt.Fprintf(
			os.Stderr,
			"LOC check failed: %v\n",
			err,
		)
		os.Exit(1)
	}

	if len(violations) == 0 {
		fmt.Printf(
			"LOC check passed: no source file exceeds %d lines.\n",
			maxLines,
		)
		return
	}

	fmt.Fprintf(
		os.Stderr,
		"LOC check failed: %d source file(s) exceed %d lines:\n\n",
		len(violations),
		maxLines,
	)

	for _, entry := range violations {
		fmt.Fprintf(
			os.Stderr,
			"  %4d  %s\n",
			entry.lines,
			entry.path,
		)
	}

	fmt.Fprintln(
		os.Stderr,
	)

	os.Exit(1)
}

func findViolations(
	root string,
) ([]violation, error) {
	var violations []violation

	err := filepath.WalkDir(
		root,
		func(
			path string,
			entry fs.DirEntry,
			walkErr error,
		) error {
			if walkErr != nil {
				return walkErr
			}

			if entry.IsDir() {
				if path == root {
					return nil
				}

				if shouldSkipDirectory(
					entry.Name(),
				) {
					return filepath.SkipDir
				}

				if isGeneratedStaticDirectory(
					path,
				) {
					return filepath.SkipDir
				}

				return nil
			}

			if !shouldCheckFile(
				path,
				entry.Name(),
			) {
				return nil
			}

			lines, err := countLines(
				path,
			)
			if err != nil {
				return fmt.Errorf(
					"count %s: %w",
					path,
					err,
				)
			}

			if lines <= maxLines {
				return nil
			}

			violations = append(
				violations,
				violation{
					path: filepath.ToSlash(
						path,
					),
					lines: lines,
				},
			)

			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	sort.Slice(
		violations,
		func(i, j int) bool {
			if violations[i].lines ==
				violations[j].lines {
				return violations[i].path <
					violations[j].path
			}

			return violations[i].lines >
				violations[j].lines
		},
	)

	return violations, nil
}

func shouldSkipDirectory(
	name string,
) bool {
	_, skip := skippedDirectories[name]

	return skip
}

func isGeneratedStaticDirectory(
	path string,
) bool {
	normalized := filepath.ToSlash(
		path,
	)

	return normalized ==
		"internal/web/static" ||
		strings.HasSuffix(
			normalized,
			"/internal/web/static",
		)
}

func shouldCheckFile(
	path string,
	name string,
) bool {
	if strings.HasSuffix(
		name,
		"_templ.go",
	) {
		return false
	}

	if strings.HasSuffix(
		name,
		".min.js",
	) ||
		strings.HasSuffix(
			name,
			".min.css",
		) {
		return false
	}

	if name == "Justfile" {
		return true
	}

	extension := strings.ToLower(
		filepath.Ext(path),
	)

	_, checked := checkedExtensions[extension]

	return checked
}

func countLines(
	path string,
) (int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(
		file,
	)

	buffer := make(
		[]byte,
		64*1024,
	)

	scanner.Buffer(
		buffer,
		4*1024*1024,
	)

	lines := 0

	for scanner.Scan() {
		lines++
	}

	if err := scanner.Err(); err != nil {
		return 0, err
	}

	if lines == 0 {
		info, err := file.Stat()
		if err != nil {
			return 0, err
		}

		if info.Size() > 0 {
			return 1, nil
		}
	}

	return lines, nil
}

var _ = errors.Is
