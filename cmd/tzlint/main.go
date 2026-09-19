// Command tzlint runs the tzlint rules against Go source files given on
// the command line and prints findings in compiler-style
// file:line:col format.
package main

import (
	"fmt"
	"os"

	tzlint "github.com/lisa-ellis577/tz-math-lint"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: tzlint <file.go> [more files...]")
		os.Exit(2)
	}

	exitCode := 0
	for _, path := range os.Args[1:] {
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
