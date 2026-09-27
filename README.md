# tz-math-lint

A static linter for a specific, recurring class of bug: code that does
timezone arithmetic as if every day were exactly 24 hours and every
zone lookup always succeeds. Both assumptions are wrong twice a year in
any location that observes DST, and wrong permanently if a container
image ships without tzdata.

It works on the Go AST, not on a running program, so a finding is
something you can point at in a code review: a rule name, a message,
and a line and column number.

## Why this exists

These three patterns show up in real codebases and pass code review
because they compile and work fine most of the year:

```go
// Looks like "same time tomorrow". On the day the clocks change,
// it's actually the same time tomorrow plus or minus an hour.
next := t.Add(24 * time.Hour)

// If "America/Sao_Paulo" is misspelled, or the container has no
// tzdata, loc is nil and every derived time silently drifts.
loc, _ := time.LoadLocation("America/Sao_Paulo")

// If layout has no zone directive, Parse assumes UTC. Fine if the
// input actually is UTC; a quiet bug if it's local time.
ts, _ := time.Parse("2006-01-02 15:04:05", raw)
```

None of these throw a compile error, a vet warning, or (usually) a
failing test, because most test runs happen to fall outside a DST
transition. tz-math-lint looks for the syntactic shape of the mistake
so it gets caught before it ships.

## Rules

| Rule | Severity | Flags |
|---|---|---|
| `fixed-day-arithmetic` | warning | `N * time.Hour` where N is a nonzero multiple of 24, e.g. `24 * time.Hour`, `48 * time.Hour` |
| `ignored-loadlocation-error` | error | `time.LoadLocation(...)` whose error return is discarded |
| `parse-without-location` | warning | any call to `time.Parse` (suggests `time.ParseInLocation` if the input isn't actually UTC) |

## Usage

As a library:

```go
src, err := os.ReadFile("scheduler.go")
if err != nil {
	log.Fatal(err)
}

findings, err := tzlint.Lint(src, "scheduler.go")
if err != nil {
	log.Fatal(err)
}

for _, f := range findings {
	fmt.Printf("scheduler.go:%d:%d: %s: %s [%s]\n",
		f.Line, f.Column, f.Severity, f.Message, f.Rule)
}
```

As a command:

```
go run ./cmd/tzlint scheduler.go billing.go

scheduler.go:6:10: warning: 24 * time.Hour assumes every day is 24 hours; use AddDate(0, 0, n) so DST transitions add up correctly [fixed-day-arithmetic]
billing.go:14:2: error: time.LoadLocation error is discarded; a missing or misspelled zone name fails silently instead of returning an error [ignored-loadlocation-error]
```

Arguments can also be a directory, walked recursively for `*.go`
files (skipping `vendor` and hidden directories like `.git`), or a
glob pattern understood by `filepath.Glob` (no `**`, but `*` and `?`
work as usual):

```
go run ./cmd/tzlint ./internal
go run ./cmd/tzlint 'pkg/*/*.go'
```

The command exits nonzero if any finding is severity `error`.

## Design

`Lint(src []byte, filename string) ([]Finding, error)` is the only
entry point, and every rule underneath it is a plain function of
`(*token.FileSet, *ast.File) -> []Finding`. There's no shared mutable
state, no filesystem access below the CLI layer, and no global
registry beyond the `Rules` slice that lists what's registered. Given
the same source bytes, `Lint` always returns the same findings in the
same order — that's what makes it possible to unit test a rule with
nothing but a string literal, no fixtures or temp files required.

## Status

Early skeleton: three rules, no configuration, no output mode besides
plain text. See the issue tracker for what's next.

## License

MIT, see [LICENSE](LICENSE).
