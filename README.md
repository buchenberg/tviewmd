# tviewmd

[![CI](https://github.com/buchenberg/tviewmd/actions/workflows/ci.yml/badge.svg)](https://github.com/buchenberg/tviewmd/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/buchenberg/tviewmd.svg)](https://pkg.go.dev/github.com/buchenberg/tviewmd)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

Native markdown rendering for [`tview`](https://github.com/rivo/tview) `TextView`s.

`tviewmd` parses markdown (CommonMark + GFM) into a renderer-neutral block/segment
model, then emits **tview color tags** directly.

## Why

tview apps today render markdown via `glamour`, which produces ANSI that must be
re-parsed into tview tags with `tview.TranslateANSI`. That hop is lossy, wasteful,
and glamour's block backgrounds clash with tview widgets. `tviewmd` emits tview
tags natively so a `TextView.SetWrap(true)` just works.

## Quick start

```go
import "github.com/buchenberg/tviewmd"

tv := tview.NewTextView().SetWrap(true)
fmt.Fprint(tv, tviewmd.Render("# Hello\n\nSome **bold** and `code`.", tviewmd.Options{}))
```

## Design

- **Parser**: [`goldmark`](https://github.com/yuin/goldmark) (CommonMark + GFM) —
  correct parsing for free.
- **Code highlighting**: [`chroma`](https://github.com/alecthomas/chroma)
  `terminal256` formatter, translated to tview tags.
- **Layered core**: `Parse` → `[]Block` (renderer-neutral) → `RenderTView`. A
  future `RenderANSI` / plain backend can reuse the same `[]Block`.

The `[]Block` / `Segment` model is the real deliverable: a curated terminal-oriented
markdown vocabulary, smaller and friendlier than goldmark's full AST.

## Status

In-tree development under the `yaah` repository (`md/`), extracted via a Go
workspace. When the test bar is met it lifts to its own repo with zero import
rewrites (the module path is already `github.com/buchenberg/tviewmd`).

## Testing

The project uses GitHub Actions for continuous integration with the following workflows:

| Workflow | Description | Badge |
|----------|-------------|-------|
| **CI** | Runs tests, linting, and build verification on push/PR | [![CI](https://github.com/buchenberg/tviewmd/actions/workflows/ci.yml/badge.svg)](https://github.com/buchenberg/tviewmd/actions/workflows/ci.yml) |

### Running Tests Locally

```bash
# Run all tests with race detector
go test -v -race ./...

# Run fuzz tests
go test -fuzz=FuzzParse -fuzztime=60s ./...

# Run with coverage
go test -coverprofile=coverage.out -covermode=atomic ./...
go tool cover -func=coverage.out
```

### Using Makefile

```bash
# Run all checks (test + lint + build)
make all

# Individual targets
make test      # Run tests with race detector
make lint      # Run golangci-lint
make build     # Build the project
make coverage  # Generate coverage report
make fuzz      # Run fuzz tests
make clean     # Clean build artifacts
```

## Linting

The project uses [golangci-lint](https://golangci-lint.run/) with a comprehensive configuration.

```bash
# Install linter (if not in CI)
make install-linter

# Run linting
make lint
```

## License

MIT.
