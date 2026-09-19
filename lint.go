// Package tzlint statically analyzes Go source for timezone and DST
// arithmetic mistakes: fixed-length-day math, discarded LoadLocation
// errors, and zone-less parsing. It works on parsed source only, never
// on a running program, so every finding is reproducible from the AST
// alone.
package tzlint

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
)

// Severity ranks how confident a rule is that it found a real bug.
type Severity int

const (
	SeverityWarning Severity = iota
	SeverityError
)

func (s Severity) String() string {
	if s == SeverityError {
		return "error"
	}
	return "warning"
}

// Finding is one reported problem, positioned the way compilers do:
// 1-indexed line and column within the file that was linted.
type Finding struct {
	Rule     string
	Message  string
	Line     int
	Column   int
	Severity Severity
}

// Rule inspects a parsed file and returns whatever it finds. Check must
// not mutate fset or file; Lint relies on that to run rules in any order.
type Rule struct {
	Name  string
	Check func(fset *token.FileSet, file *ast.File) []Finding
}

// Rules is the full set applied by Lint, in registration order. The
// order doesn't affect output: Lint sorts findings by position.
var Rules = []Rule{
	{Name: "fixed-day-arithmetic", Check: checkFixedDayArithmetic},
	{Name: "ignored-loadlocation-error", Check: checkIgnoredLoadLocationError},
	{Name: "parse-without-location", Check: checkParseWithoutLocation},
}

// Lint parses src and runs every rule against it. It takes bytes rather
// than a file path so callers (and tests) never need a filesystem: pass
// a snippet in memory and get findings back. filename is only used to
// label parse errors and positions; it does not need to exist on disk.
func Lint(src []byte, filename string) ([]Finding, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", filename, err)
	}

	var findings []Finding
	for _, rule := range Rules {
		findings = append(findings, rule.Check(fset, file)...)
	}

	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Line != findings[j].Line {
			return findings[i].Line < findings[j].Line
		}
		return findings[i].Column < findings[j].Column
	})

	return findings, nil
}
