// Command tzlint runs the tzlint rules against Go source given on the
// command line and prints findings in compiler-style file:line:col
// format. Arguments may be individual files, glob patterns, or
// directories; directories are walked recursively for *.go files.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tzlint "github.com/lisa-ellis577/tz-math-lint"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: tzlint <file.go|dir|glob> [more...]")
		os.Exit(2)
	}

	paths, err := resolvePaths(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	exitCode := 0
	for _, path := range paths {
		src, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", path, err)
			exitCode = 1
			continue
		}

		findings, err := tzlint.Lint(src, path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", path, err)
			exitCode = 1
			continue
		}

		for _, f := range findings {
			fmt.Printf("%s:%d:%d: %s: %s [%s]\n", path, f.Line, f.Column, f.Severity, f.Message, f.Rule)
			if f.Severity == tzlint.SeverityError {
				exitCode = 1
			}
		}
	}

	os.Exit(exitCode)
}

// resolvePaths expands the command-line arguments into a sorted,
// deduplicated list of .go files. Each argument is treated, in order,
// as a directory to walk, a glob pattern, or a literal file path;
// whichever of those actually matches something wins.
func resolvePaths(args []string) ([]string, error) {
	seen := make(map[string]bool)
	var out []string

	add := func(path string) {
		if !seen[path] {
			seen[path] = true
			out = append(out, path)
		}
	}

	for _, arg := range args {
		if info, err := os.Stat(arg); err == nil && info.IsDir() {
			if walkErr := walkGoFiles(arg, add); walkErr != nil {
				return nil, fmt.Errorf("%s: %w", arg, walkErr)
			}
			continue
		}

		matches, globErr := filepath.Glob(arg)
		if globErr != nil {
			return nil, fmt.Errorf("%s: %w", arg, globErr)
		}
		if len(matches) == 0 {
			// No directory and no glob match: treat as a literal path so
			// a missing file still surfaces its own read error later.
			add(arg)
			continue
		}

		for _, m := range matches {
			if info, err := os.Stat(m); err == nil && info.IsDir() {
				if walkErr := walkGoFiles(m, add); walkErr != nil {
					return nil, fmt.Errorf("%s: %w", m, walkErr)
				}
				continue
			}
			add(m)
		}
	}

	sort.Strings(out)
	return out, nil
}

// walkGoFiles calls add for every .go file under root, skipping hidden
// directories (".git", ".idea", ...) and "vendor", since neither holds
// code this linter should report on.
func walkGoFiles(root string, add func(string)) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			name := info.Name()
			if path != root && (strings.HasPrefix(name, ".") || name == "vendor") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, ".go") {
			add(path)
		}
		return nil
	})
}
