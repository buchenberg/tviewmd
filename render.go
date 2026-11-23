package tviewmd

import (
	"strings"

	"github.com/rivo/tview"
)

// Theme holds the color scheme used by renderers. Colors are expressed in any
// form the target backend understands (tview accepts named colors like "red",
// hex like "#ff8c42", and "default").
type Theme struct {
	// Heading colors per level (index 0 = H1). Used by the tview backend.
	Heading [6]string

	Link         string
	InlineCodeFG string
	InlineCodeBG string
	CodeBlockFG  string
	QuoteFG      string
	Hr           string
}

// DefaultTheme returns a dark-terminal-friendly theme tuned for readability on
// a black/default background.
func DefaultTheme() Theme {
	return Theme{
		Heading:      [6]string{"#ff8c42", "#ffae57", "#ffd166", "#a3e635", "#56cffd", "#c08bff"},
		Link:         "#56cffd",
		InlineCodeFG: "#ffffff",
		InlineCodeBG: "#3a3a3a",
		CodeBlockFG:  "#e6e6e6",
		QuoteFG:      "#9ca3af",
		Hr:           "#555555",
	}
}

// WithDefaults fills each empty color field of t from DefaultTheme, leaving
// explicitly set fields untouched. Defaulting is per field, not per struct: a
// theme that changes only Link keeps the default heading, code and rule colors.
func (t Theme) WithDefaults() Theme {
	d := DefaultTheme()
	for i := range t.Heading {
		if t.Heading[i] == "" {
			t.Heading[i] = d.Heading[i]
		}
	}
	if t.Link == "" {
		t.Link = d.Link
	}
	if t.InlineCodeFG == "" {
		t.InlineCodeFG = d.InlineCodeFG
	}
	if t.InlineCodeBG == "" {
		t.InlineCodeBG = d.InlineCodeBG
	}
	if t.CodeBlockFG == "" {
		t.CodeBlockFG = d.CodeBlockFG
	}
	if t.QuoteFG == "" {
		t.QuoteFG = d.QuoteFG
	}
	if t.Hr == "" {
		t.Hr = d.Hr
	}
	return t
}

// NoColorTheme returns a theme that asks tview to inherit the terminal's
// default color everywhere. It is the hook for NO_COLOR environments and
// monochrome terminals, which cannot be expressed with the zero Theme (that
// value means "use DefaultTheme").
// inheritColor is tview's name for the terminal's default color.
const inheritColor = "default"

// NoColorTheme returns a theme that asks tview to inherit the terminal's
// default color everywhere. It is the hook for NO_COLOR environments and
// monochrome terminals, which cannot be expressed with the zero Theme.
func NoColorTheme() Theme {
	t := Theme{
		Link:         inheritColor,
		InlineCodeFG: inheritColor,
		InlineCodeBG: inheritColor,
		CodeBlockFG:  inheritColor,
		QuoteFG:      inheritColor,
		Hr:           inheritColor,
	}
	for i := range t.Heading {
		t.Heading[i] = inheritColor
	}
	return t
}

// Options configure rendering. The zero value renders with the default theme
// and an 80-column width for rules; tables use max-content column widths
// unless Width constrains them.
type Options struct {
	// Width is the horizontal-rule length and the table layout budget. 0 means
	// 80 for rules and max-content (no constrained layout) for tables.
	//
	// Tables are planned when text is rendered, not when it is painted: their
	// column widths are baked into the returned tag string. Callers that want
	// tables to track the terminal width must re-render with an updated Width
	// when the available width changes; resizing the TextView alone only
	// re-wraps the already-planned table as prose.
	Width int

	// Theme is the color theme. The zero value uses DefaultTheme().
	Theme Theme

	// TableRowSpacing is the number of blank lines painted between table body
	// rows. The zero value uses one blank line, which makes row breaks legible
	// in a tview.TextView; -1 selects the compact layout with no blank lines;
	// larger positive values add that many blank lines. Spacing is applied
	// between logical rows only -- never after the last row, so Render's
	// trailing-newline trimming is unaffected. Values above 64 are clamped.
	TableRowSpacing int

	// MaxStreamBytes bounds how much raw source a Renderer accumulates.
	// Zero uses DefaultMaxStreamBytes (8 MiB); -1 means unlimited. Render and
	// RenderBytes ignore it -- it exists to stop a runaway token stream from
	// growing the buffer and making every String() re-render ever more
	// expensive.
	MaxStreamBytes int
}

// CompactTableRows disables blank lines between table rows. Pass it as
// Options.TableRowSpacing to get the pre-spacing, dense table layout.
const CompactTableRows = -1

// DefaultMaxStreamBytes is the stream cap used when Options.MaxStreamBytes is
// zero. It is deliberately generous: a runaway-stream guard, not a normal
// response limit.
const DefaultMaxStreamBytes = 8 << 20

// Render parses src and renders it to tview color-tagged text. It is the
// convenience entry point for callers that do not need the intermediate []Block.
//
// Render is safe to call on each streaming update: it fully re-renders the
// accumulated buffer and escapes literal brackets, so incremental re-rendering
// never produces unbalanced or injected tview tags. For a buffered streaming
// accumulator see Renderer.
func Render(src string, opts Options) string {
	blocks, err := Parse(src)
	switch {
	case err != nil:
		// Parse documents that it never fails, but a future backend might. Escape
		// rather than echo src: raw brackets would be read as tview tags.
		return tview.Escape(src)
	case len(blocks) == 0:
		return ""
	}
	return RenderTView(blocks, opts)
}

// RenderLines renders src and returns its planned physical lines, including
// the blank spacer lines between table rows. It is a convenience for consumers
// that need a row count (for example a TUI anchoring a prompt below a message)
// without re-wrapping the already-planned output.
//
// With a positive Options.Width, tables are planned to fit that budget, so the
// returned lines are the rows a TextView at that width paints. With Width 0 the
// TextView may wrap further; callers that need an exact painted count must set
// Width or use a SimulationScreen.
func RenderLines(src string, opts Options) []string {
	out := Render(src, opts)
	if out == "" {
		return nil
	}
	return strings.Split(out, "\n")
}
