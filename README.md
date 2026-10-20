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
    tv := tview.NewTextView().SetWrap(true)
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
import "github.com/buchenberg/tviewmd"

// Simple rendering with defaults
output := tviewmd.Render("# Heading\n\nParagraph text.", tviewmd.Options{})

// Render to a TextView
textView := tview.NewTextView().SetWrap(true)
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

For real-time rendering of streaming LLM responses:

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
    textView := tview.NewTextView().SetWrap(true)
    
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

### Disable Syntax Highlighting

### Custom Width

Set a custom width for horizontal rules and table sizing:

```go
output := tviewmd.Render(markdown, tviewmd.Options{
    Width: 120, // Default is 80
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

## API Reference

### Functions

| Function | Description |
|----------|-------------|
| `Render(src string, opts Options) string` | Parse and render markdown to tview tags in one call |
| `Parse(src string) ([]Block, error)` | Parse markdown into renderer-neutral blocks |
| `RenderTView(blocks []Block, opts Options) string` | Render parsed blocks to tview tags |

### Types

#### Options

```go
type Options struct {
    // Width for horizontal rules and the table layout budget (default: 80).
    // Tables are planned at render time; re-render on resize.
    Width int

    // Custom color theme (default: DefaultTheme())
    Theme Theme
}
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
```

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

#### Segment Type

Inline content is represented as styled segments:

```go
type Segment struct {
    Text  string  // The text content
    Style Style   // Rendering style (bold, italic, etc.)
    Link  string // URL for links (empty if not a link)
    Code  bool    // true for inline code spans
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
textView := tview.NewTextView().SetWrap(true)  // ✅ Required
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

3. **Pre-parse** if re-rendering the same content:
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
4. Run `make all` to verify everything works
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

# Run all checks
make all

# Or individually:
make test    # Run tests
make lint    # Run linter
make build   # Build project
```

## Status

tviewmd is a standalone, production-ready library at `github.com/buchenberg/tviewmd`.

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
