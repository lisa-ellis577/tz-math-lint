package tzlint

import "testing"

func TestFixedDayArithmetic(t *testing.T) {
	src := []byte(`package sample

import "time"

func nextDay(t time.Time) time.Time {
	return t.Add(24 * time.Hour)
}
`)
	findings, err := Lint(src, "sample.go")
	if err != nil {
		t.Fatalf("Lint returned error: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(findings), findings)
	}
	if findings[0].Rule != "fixed-day-arithmetic" {
		t.Errorf("expected rule fixed-day-arithmetic, got %s", findings[0].Rule)
	}
	if findings[0].Line != 6 {
		t.Errorf("expected finding on line 6, got %d", findings[0].Line)
	}
}

func TestIgnoredLoadLocationError(t *testing.T) {
	src := []byte(`package sample

import "time"

func zone() *time.Location {
	loc, _ := time.LoadLocation("Europe/Berlin")
	return loc
}
`)
	findings, err := Lint(src, "sample.go")
	if err != nil {
		t.Fatalf("Lint returned error: %v", err)
	}
	if len(findings) != 1 || findings[0].Rule != "ignored-loadlocation-error" {
		t.Fatalf("expected a single ignored-loadlocation-error finding, got %+v", findings)
	}
	if findings[0].Severity != SeverityError {
		t.Errorf("expected severity error, got %s", findings[0].Severity)
	}
}

func TestCheckedLoadLocationErrorIsNotFlagged(t *testing.T) {
	src := []byte(`package sample

import "time"

func zone() (*time.Location, error) {
	loc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		return nil, err
	}
	return loc, nil
}
`)
	findings, err := Lint(src, "sample.go")
	if err != nil {
		t.Fatalf("Lint returned error: %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("expected no findings, got %+v", findings)
	}
}

func TestParseWithoutLocation(t *testing.T) {
	src := []byte(`package sample

import "time"

func parse(s string) (time.Time, error) {
	return time.Parse("2006-01-02", s)
}
`)
	findings, err := Lint(src, "sample.go")
	if err != nil {
		t.Fatalf("Lint returned error: %v", err)
	}
	if len(findings) != 1 || findings[0].Rule != "parse-without-location" {
		t.Fatalf("expected a single parse-without-location finding, got %+v", findings)
	}
}

func TestCleanSourceHasNoFindings(t *testing.T) {
	src := []byte(`package sample

import "time"

func nextDay(t time.Time) time.Time {
	return t.AddDate(0, 0, 1)
}
`)
	findings, err := Lint(src, "sample.go")
	if err != nil {
		t.Fatalf("Lint returned error: %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("expected no findings, got %+v", findings)
	}
}

func TestLintReturnsErrorOnInvalidSyntax(t *testing.T) {
	src := []byte(`package sample

func broken( {
`)
	if _, err := Lint(src, "sample.go"); err == nil {
		t.Fatal("expected a parse error, got nil")
	}
}
