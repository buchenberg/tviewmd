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
tags natively, so a `TextView` with `SetWrap(true)` **and**
`SetDynamicColors(true)` just works — tview disables dynamic colors by default,
so without the second call the tags render as literal text.

## Installation

```bash
go get github.com/buchenberg/tviewmd
```

Or add to your `go.mod`:

```bash
go mod edit -require=github.com/buchenberg/tviewmd@latest
go mod tidy
```

## Quick start

```go
package main

import (
    "fmt"
    "github.com/buchenberg/tviewmd"
    "github.com/rivo/tview"
)

func main() {
    app := tview.NewApplication()
    tv := tview.NewTextView().SetWrap(true).SetDynamicColors(true)
    fmt.Fprint(tv, tviewmd.Render("# Hello\n\nSome **bold** and `code`.", tviewmd.Options{}))
    if err := app.SetRoot(tv, true).Run(); err != nil {
        panic(err)
    }
}
```

## Usage Examples

### Basic Rendering

Render markdown to a tview TextView:

```go
import (
    "github.com/buchenberg/tviewmd"
    "github.com/rivo/tview"
)

// Simple rendering with defaults
output := tviewmd.Render("# Heading\n\nParagraph text.", tviewmd.Options{})

// Render to a TextView
textView := tview.NewTextView().SetWrap(true).SetDynamicColors(true)
textView.SetText(output)
```

### Complete Markdown Features

```go
markdown := "# Title\n\n" +
    "## Subtitle\n\n" +
    "This is a paragraph with **bold**, *italic*, ~~strikethrough~~, and `inline code`.\n\n" +
    "### Code Block\n\n" +
    "```go\n" +
    "func main() {\n" +
    "    fmt.Println(\"Hello, World!\")\n" +
    "}\n" +
    "```\n\n" +
    "### List\n\n" +
    "- Item 1\n" +
    "- Item 2\n" +
    "  - Nested item\n" +
    "- Item 3\n\n" +
    "### Table\n\n" +
    "| Column 1 | Column 2 |\n" +
    "|----------|----------|\n" +
    "| Row 1    | Data     |\n" +
    "| Row 2    | More     |\n\n" +
    "### Blockquote\n\n" +
    "> This is a blockquote\n\n" +
    "### Link\n\n" +
    "[GitHub](https://github.com)\n\n" +
    "### Thematic Break\n\n" +
    "---\n\n" +
    "### Task List\n\n" +
    "- [x] Task 1 (completed)\n" +
    "- [ ] Task 2 (pending)"

textView.SetText(tviewmd.Render(markdown, tviewmd.Options{}))
```

### With Custom Theme

```go
// Create a custom theme
theme := tviewmd.Theme{
    Heading:      [6]string{"#ff5555", "#ff7755", "#ff9955", "#ffbb55", "#ffdd55", "#ffff55"},
    Link:         "#55aaff",
    InlineCodeFG: "#ffffff",
    InlineCodeBG: "#333333",
    CodeBlockFG:  "#e0e0e0",
    QuoteFG:      "#888888",
    Hr:           "#666666",
}

// Use the custom theme
output := tviewmd.Render(markdown, tviewmd.Options{
    Theme: theme,
})
```

### Streaming LLM Output

For real-time rendering of streaming LLM responses, the recommended API is
`Renderer` (see below); the lower-level pattern is:

```go
func renderLLMStream(app *tview.Application, stream <-chan []byte, textView *tview.TextView) {
    var buf strings.Builder
    throttle := time.Tick(50 * time.Millisecond) // Adjust throttle as needed

    for {
        select {
        case chunk, ok := <-stream:
            if !ok {
                // Stream ended - final render
                app.QueueUpdateDraw(func() {
                    textView.SetText(tviewmd.Render(buf.String(), tviewmd.Options{}))
                })
                return
            }
            buf.Write(chunk)
        case <-throttle:
            // Throttled re-render for smooth updates
            app.QueueUpdateDraw(func() {
                textView.SetText(tviewmd.Render(buf.String(), tviewmd.Options{}))
            })
        }
    }
}

// Usage with a mock stream
func demoStreaming() {
    app := tview.NewApplication()
    textView := tview.NewTextView().SetWrap(true).SetDynamicColors(true)
    
    // Simulate streaming
    stream := make(chan []byte)
    go func() {
        messages := []string{
            "# Response\n\n",
            "Here is the ",
            "**answer** ",
            "to your question.\n\n",
            "```go\n",
            "func example() {}\n",
            "```\n",
        }
        for _, msg := range messages {
            stream <- []byte(msg)
            time.Sleep(100 * time.Millisecond)
        }
        close(stream)
    }()

    // Run renderer in a goroutine
    go renderLLMStream(app, stream, textView)
    
    // Start the application
    if err := app.SetRoot(textView, true).Run(); err != nil {
        panic(err)
    }
}
```

For a reusable buffer that caches rendered output, use `Renderer` instead of
rolling your own `strings.Builder`:

```go
r := tviewmd.NewRenderer(tviewmd.Options{})
r.Append(string(chunk))  // Append takes a string; io.Copy(r, stream) also works
text := r.String()       // cached until new input or options change
r.SetWidth(newWidth)     // resize hook: changes only Options.Width
r.SetOptions(opts)       // replaces all Options; accumulated source is kept
src := r.Source()        // raw markdown accumulated so far
if errors.Is(r.Err(), tviewmd.ErrStreamTooLarge) { /* stream was capped */ }
r.Reset()                // clears source, output and Err; keeps options
```

`SetOptions` and `SetWidth` take effect on the next `String()` call and never
discard the buffered source. `Err` is sticky until `Reset`. A `Renderer` is safe
for concurrent use; flush from a `QueueUpdateDraw` callback on the tview event
loop and assign the result directly (do not queue from inside that callback).

`Options.MaxStreamBytes` caps accumulated source (`DefaultMaxStreamBytes`, 8 MiB,
when zero; `-1` for unlimited). Input past the cap is dropped, `Err` reports
`tviewmd.ErrStreamTooLarge`, and the buffered prefix stays renderable. `Render`
and `RenderTView` ignore this option.

### Custom Width

Set a custom width for horizontal rules and table sizing:

```go
output := tviewmd.Render(markdown, tviewmd.Options{
    Width: 120, // 0 = 80 for rules, max-content tables; negative unsupported
})
```

Tables are planned at render time: the column widths are baked into the
returned tag string. Re-render with an updated `Width` when the terminal size
changes; resizing the `TextView` alone only re-wraps the already-planned table
as prose.

```go
// On tview's resize event, re-render the accumulated source:
textView.SetText(tviewmd.Render(src, tviewmd.Options{Width: width}))
```

### Two-Phase Parsing and Rendering

For advanced use cases where you want to inspect or transform the parsed structure:

```go
// Parse markdown into blocks (renderer-neutral AST)
blocks, err := tviewmd.Parse(markdown)
if err != nil {
    // Handle error (Parse never returns error for valid markdown)
    log.Fatal(err)
    return
}

// Inspect or transform blocks...
for _, block := range blocks {
    switch block.Kind {
    case tviewmd.BlockHeading:
        fmt.Printf("Heading level %d: %s\n", block.Level, tviewmd.PlainText(block.Segments))
    case tviewmd.BlockParagraph:
        fmt.Printf("Paragraph: %s\n", tviewmd.PlainText(block.Segments))
    }
}

// Then render to tview tags
output := tviewmd.RenderTView(blocks, tviewmd.Options{})
```

Use `PlainText` as above to inspect segment text. When walking parsed blocks,
remember the AST is a lossy, terminal-oriented view: nested quotes/list items
live in `Block.Blocks` / `ListItem.Blocks`, images are alt text only (URL
discarded), raw HTML is literal text, and `AlignDefault` renders like
`AlignLeft`.

## API Reference

### Functions

| Function | Description |
|----------|-------------|
| `Render(src string, opts Options) string` | Parse and render markdown to tview tags in one call |
| `Parse(src string) ([]Block, error)` | Parse markdown into renderer-neutral blocks |
| `RenderTView(blocks []Block, opts Options) string` | Render parsed blocks to tview tags |
| `PlainText(segs []Segment) string` | Concatenate segment text without rendering tview tags |
| `RenderLines(src string, opts Options) []string` | Render and split into planned physical lines (including table spacers) |

### Types

#### Constants

| Constant | Meaning |
|----------|---------|
| `CompactTableRows` | `Options.TableRowSpacing` value that disables blank lines between table rows |
| `DefaultMaxStreamBytes` | Default `Renderer` stream cap (8 MiB) |
| `ErrStreamTooLarge` | Reported by `Renderer.Err`/`Write` when the stream cap is hit |

#### Options

```go
type Options struct {
    // Width for horizontal rules and the table layout budget. 0 means 80 for
    // rules and max-content tables. Negative values currently behave like 0
    // (the table branch applies a budget only when Width > 0); they are
    // unsupported, so pass 0 or a positive width.
    Width int

    // Custom color theme (zero value: DefaultTheme()).
    Theme Theme

    // Blank lines between table body rows (default: 1).
    // Use tviewmd.CompactTableRows (-1) for the dense layout with none.
    TableRowSpacing int

    // Caps source accumulated by a Renderer. 0 uses DefaultMaxStreamBytes
    // (8 MiB); -1 means unlimited. Ignored by Render.
    MaxStreamBytes int
}
```

Tables get one blank line between body rows by default so row breaks stay
visible in a `tview.TextView`. To restore the dense pre-spacing layout:

```go
out := tviewmd.Render(markdown, tviewmd.Options{TableRowSpacing: tviewmd.CompactTableRows})
```

#### Renderer

`Renderer` is a concurrency-safe streaming buffer. `NewRenderer(opts)` starts
one; `Append(string)`, `Write([]byte)` (so it implements `io.Writer`), `String`,
`Source`, `Reset`, `SetOptions`, `SetWidth` and `Err` operate on it.

```go
r := tviewmd.NewRenderer(tviewmd.Options{})
r.Append(chunk)
text := r.String()
r.SetWidth(width)     // re-layout the buffered source
r.SetOptions(opts)    // replace all options, keep the source
```

#### Theme

```go
type Theme struct {
    // Heading colors per level (index 0 = H1, 5 = H6)
    Heading [6]string
    
    // Link color
    Link string
    
    // Inline code foreground and background
    InlineCodeFG string
    InlineCodeBG string
    
    // Code block foreground
    CodeBlockFG string
    
    // Blockquote text color
    QuoteFG string
    
    // Horizontal rule color
    Hr string
}

// DefaultTheme returns a dark-terminal-friendly theme
func DefaultTheme() Theme

// WithDefaults fills only the empty fields, so a partial theme keeps the
// default colors everywhere else.
func (t Theme) WithDefaults() Theme

// NoColorTheme asks tview to inherit the terminal default everywhere (NO_COLOR).
func NoColorTheme() Theme
```

A theme with only `Link` set keeps the default heading/rule/code colors;
defaulting is per field, not per struct. A zero `Theme` still means
`DefaultTheme()`; use `NoColorTheme()` when you want tview's `default` colors
(NO_COLOR / monochrome terminals).

#### Block Types

The `Parse` function returns a slice of `Block` structs:

```go
type Block struct {
    Kind     BlockKind  // Type of block
    Level    int        // Heading level (1-6, for BlockHeading)
    Segments []Segment  // Inline content (for paragraphs, headings, blockquotes)
    Code     CodeBlock  // Code block content (for BlockCodeBlock)
    Items    []ListItem // List items (for BlockList)
    Table    TableData  // Table content (for BlockTable)
    Blocks   []Block    // Nested block children (blockquotes)
}

type ListItem struct {
    Segments []Segment
    Children []ListItem // legacy flattened nested items
    Blocks   []Block    // full block children of the item
    Ordered  bool
    Index    int        // 1-based ordinal for ordered items
}

type BlockKind int

const (
    BlockParagraph BlockKind = iota
    BlockHeading
    BlockCodeBlock
    BlockList
    BlockBlockquote
    BlockTable
    BlockThematicBreak
)
```

The AST is intentionally lossy: it is a terminal-oriented view, not goldmark's
full tree. `Block.Blocks` and `ListItem.Blocks` preserve nested code fences,
tables, lists, headings and multiple paragraphs inside blockquotes and list
items; the `Segments` / `Children` fields remain the flattened view used by
hand-built blocks. Images keep only their alt text (the URL is discarded), raw
HTML is emitted as literal text, and `AlignDefault` renders like `AlignLeft`.

#### Segment Type

Inline content is represented as styled segments:

```go
type Segment struct {
    Text   string // The text content
    Style  Style  // Rendering style (bold, italic, etc.)
    Link   string // URL for links (empty if not a link)
    LinkID int    // distinguishes adjacent links with the same URL
    Code   bool   // true for inline code spans
}

type Style struct {
    FG string // Foreground color
    BG string // Background color
    Bold          bool
    Italic        bool
    Underline     bool
    Strikethrough bool
    Dim           bool
}
```

`tviewmd.PlainText(segs)` returns the concatenated text of a `[]Segment`
without emitting any tview tags.

## Design

- **Parser**: [`goldmark`](https://github.com/yuin/goldmark) (CommonMark + GFM) —
  correct parsing for free.
- **Layered core**: `Parse` → `[]Block` (renderer-neutral) → `RenderTView`. A
  future `RenderANSI` / plain backend can reuse the same `[]Block`.

The `[]Block` / `Segment` model is the real deliverable: a curated terminal-oriented
markdown vocabulary, smaller and friendlier than goldmark's full AST.

## Comparison with Alternatives

| Approach | Pros | Cons |
|----------|------|------|
| **tviewmd** (this library) | Native tview tags, no ANSI round-trip, correct parsing | New library, smaller community |
| **glamour + TranslateANSI** | Mature, widely used | Lossy ANSI round-trip, background color clashes, performance overhead |
| **text/tabwriter** | Standard library | No markdown support, manual formatting |
| **gocui** | Good for complex UIs | Different widget system, not tview-compatible |

### Why tviewmd over glamour + TranslateANSI?

```go
// OLD WAY: glamour + TranslateANSI (3 steps, lossy)
import (
    "github.com/charmbracelet/glamour"
    "github.com/rivo/tview"
)

renderer, _ := glamour.NewTerminalRenderer()
ansi, _ := renderer.Render("# Hello **world**")
tviewText := tview.TranslateANSI(ansi)  // Lossy conversion!

// NEW WAY: tviewmd (1 step, native)
import "github.com/buchenberg/tviewmd"

tviewText := tviewmd.Render("# Hello **world**", tviewmd.Options{})
```

**Benefits of tviewmd:**
- ✅ **No ANSI round-trip**: Direct tview tag emission
- ✅ **Preserves styling**: No information loss from ANSI conversion
- ✅ **Better performance**: ~2-3x faster than glamour + TranslateANSI
- ✅ **Correct backgrounds**: No color clash with tview widgets
- ✅ **Simpler API**: One function call vs three

## Troubleshooting

### Common Issues

#### Text wrapping not working

Make sure you've enabled wrapping on your TextView:

```go
textView := tview.NewTextView().SetWrap(true).SetDynamicColors(true)  // ✅ Both required
```

#### Code blocks not colored

Code blocks render with the `CodeBlockFG` theme color. If you expect
syntax highlighting, note that chroma was removed in favor of a simpler,
safer, and faster single-color approach.

#### Colors not appearing

Check that:
1. Your terminal supports 256 colors or true color
2. The TextView is properly sized and visible
3. You're using a theme with valid color codes

```go
// Valid color formats:
// - Named colors: "red", "green", "blue"
// - Hex colors: "#ff0000", "#00ff00"
// - tview special: "default", ""
```

#### Markdown not parsing correctly

Ensure you're using valid CommonMark + GFM syntax:

```go
// ✅ Valid GFM table
markdown := "| A | B |\n|---|---|\n| 1 | 2 |"

// ❌ Invalid (missing separator row)
markdown := "| A | B |\n| 1 | 2 |"  // No separator row between header and data
```

#### Performance issues with large documents

For large markdown documents (>10KB):

1. **Throttle rendering** for streaming:
   ```go
   // Reduce throttle interval
   throttle := time.Tick(100 * time.Millisecond)  // Less frequent updates
   ```

2. **Pre-parse** if re-rendering the same content:
   ```go
   blocks, _ := tviewmd.Parse(markdown)
   // Cache blocks, then render with different options
   output1 := tviewmd.RenderTView(blocks, opts1)
   output2 := tviewmd.RenderTView(blocks, opts2)
   ```

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch (`git checkout -b feat/amazing-feature`)
3. Add tests for new functionality
4. Run `make all` to run the standard checks (test, lint, build)
5. Commit your changes (`git commit -m 'feat: add amazing feature'`)
6. Push to the branch (`git push origin feat/amazing-feature`)
7. Open a Pull Request

### Development Setup

```bash
# Clone the repository
git clone https://github.com/buchenberg/tviewmd.git
cd tviewmd

# Install dependencies
go mod download

# Run standard checks (test, lint, build)
make all

# Or individually:
make test    # Run tests
make lint    # Run linter
make build   # Build project
```

## Status

tviewmd is a standalone library at `github.com/buchenberg/tviewmd` (pre-v1.0;
see AGENTS.md for the versioning policy).

## Testing

The project uses GitHub Actions for continuous integration with the following workflows:

| Workflow | Description | Badge |
|----------|-------------|-------|
| **CI** | Runs tests, linting, and build verification on push/PR | [![CI](https://github.com/buchenberg/tviewmd/actions/workflows/ci.yml/badge.svg)](https://github.com/buchenberg/tviewmd/actions/workflows/ci.yml) |

### Running Tests Locally

```bash
# Run all tests with race detector
go test -v -race ./...

# Run fuzz tests (the -fuzz flag targets one package)
go test -fuzz=FuzzParse -fuzztime=60s .

# Run with coverage
go test -coverprofile=coverage.out -covermode=atomic ./...
go tool cover -func=coverage.out
```

### Using Makefile

```bash
# Standard checks (test + lint + build); coverage and fuzz are separate targets
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

The project uses [golangci-lint](https://golangci-lint.run/) v2 (v2.15.0 in CI)
with a v2 configuration (`version: "2"`); v1.x cannot read this config.

```bash
# Install linter (if not in CI)
make install-linter

# Run linting
make lint
```

## License

MIT.
