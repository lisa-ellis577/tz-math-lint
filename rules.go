package tzlint

import (
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
)

// checkFixedDayArithmetic flags expressions like 24*time.Hour or
// 48*time.Hour. They assume every day is exactly 24 hours, which is
// false on the day a location switches into or out of DST. AddDate is
// the calendar-aware replacement.
func checkFixedDayArithmetic(fset *token.FileSet, file *ast.File) []Finding {
	var findings []Finding
	ast.Inspect(file, func(n ast.Node) bool {
		be, ok := n.(*ast.BinaryExpr)
		if !ok || be.Op != token.MUL {
			return true
		}

		lit, ok := literalMultipleOfADay(be)
		if !ok {
			return true
		}

		pos := fset.Position(be.Pos())
		findings = append(findings, Finding{
			Rule:     "fixed-day-arithmetic",
			Severity: SeverityWarning,
			Line:     pos.Line,
			Column:   pos.Column,
			Message: fmt.Sprintf(
				"%s * time.Hour assumes every day is 24 hours; use AddDate(0, 0, n) so DST transitions add up correctly",
				lit,
			),
		})
		return true
	})
	return findings
}

// literalMultipleOfADay reports whether be multiplies time.Hour by an
// integer literal that is a nonzero multiple of 24, in either operand
// order, and returns that literal's source text.
func literalMultipleOfADay(be *ast.BinaryExpr) (string, bool) {
	pairs := [][2]ast.Expr{{be.X, be.Y}, {be.Y, be.X}}
	for _, pair := range pairs {
		lit, ok := pair[0].(*ast.BasicLit)
		if !ok || lit.Kind != token.INT {
			continue
		}
		if !isTimeHour(pair[1]) {
			continue
		}
		hours, err := strconv.Atoi(lit.Value)
		if err != nil || hours == 0 || hours%24 != 0 {
			continue
		}
		return lit.Value, true
	}
	return "", false
}

func isTimeHour(e ast.Expr) bool {
	return isTimeSelector(e, "Hour")
}

func isTimeSelector(e ast.Expr, name string) bool {
	sel, ok := e.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != name {
		return false
	}
	ident, ok := sel.X.(*ast.Ident)
	return ok && ident.Name == "time"
}

// checkIgnoredLoadLocationError flags calls to time.LoadLocation whose
// error return is thrown away. LoadLocation fails whenever the IANA
// zone database entry is missing (e.g. a typo, or a minimal container
// image with no tzdata), and the caller silently keeps going with a nil
// or zero location instead of the zone it asked for.
func checkIgnoredLoadLocationError(fset *token.FileSet, file *ast.File) []Finding {
	var findings []Finding
	ast.Inspect(file, func(n ast.Node) bool {
		switch stmt := n.(type) {
		case *ast.ExprStmt:
			if call, ok := stmt.X.(*ast.CallExpr); ok && isLoadLocationCall(call) {
				findings = append(findings, loadLocationFinding(fset, call.Pos()))
			}
		case *ast.AssignStmt:
			if len(stmt.Rhs) != 1 || len(stmt.Lhs) != 2 {
				return true
			}
			call, ok := stmt.Rhs[0].(*ast.CallExpr)
			if !ok || !isLoadLocationCall(call) {
				return true
			}
			if ident, ok := stmt.Lhs[1].(*ast.Ident); ok && ident.Name == "_" {
				findings = append(findings, loadLocationFinding(fset, call.Pos()))
			}
		}
		return true
	})
	return findings
}

func isLoadLocationCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "LoadLocation" {
		return false
	}
	ident, ok := sel.X.(*ast.Ident)
	return ok && ident.Name == "time"
}

func loadLocationFinding(fset *token.FileSet, pos token.Pos) Finding {
	p := fset.Position(pos)
	return Finding{
		Rule:     "ignored-loadlocation-error",
		Severity: SeverityError,
		Line:     p.Line,
		Column:   p.Column,
		Message:  "time.LoadLocation error is discarded; a missing or misspelled zone name fails silently instead of returning an error",
	}
}

// checkParseWithoutLocation flags time.Parse calls. time.Parse defaults
// to UTC whenever the layout string has no zone directive, which is a
// common source of off-by-several-hours bugs when the input actually
// meant local time. time.ParseInLocation makes the assumption explicit.
func checkParseWithoutLocation(fset *token.FileSet, file *ast.File) []Finding {
	var findings []Finding
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "Parse" {
			return true
		}
		ident, ok := sel.X.(*ast.Ident)
		if !ok || ident.Name != "time" {
			return true
		}

		pos := fset.Position(call.Pos())
		findings = append(findings, Finding{
			Rule:     "parse-without-location",
			Severity: SeverityWarning,
			Line:     pos.Line,
			Column:   pos.Column,
			Message:  "time.Parse falls back to UTC when the layout has no zone offset; confirm that's intended or switch to time.ParseInLocation",
		})
		return true
	})
	return findings
}
