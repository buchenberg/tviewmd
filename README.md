# tviewmd

Native markdown rendering for [`tview`](https://github.com/rivo/tview) `TextView`s.

`tviewmd` parses markdown (CommonMark + GFM) into a renderer-neutral block/segment
model, then emits **tview color tags** directly — no ANSI round-trip, no
`glamour`/`TranslateANSI` double translation.

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

## License

MIT.
