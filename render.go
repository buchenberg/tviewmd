package tviewmd

import "strings"

// Theme holds the color scheme used by renderers. Colors are expressed in any
// form the target backend understands (tview accepts named colors like "red",
// hex like "#ff8c42", and "default").
type Theme struct {
	// Heading colors per level (index 0 = H1). Used by the tview backend.
	Heading [6]string

	Link        string
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

// Options configure rendering. The zero value renders with the default theme,
// syntax highlighting enabled, and an 80-column width for rules and tables.
type Options struct {
	// Width is used for horizontal-rule length and table column sizing. 0 = 80.
	Width int

	// NoHighlight disables chroma syntax highlighting for fenced code blocks.
	NoHighlight bool

	// Theme is the color theme. The zero value uses DefaultTheme().
	Theme Theme
}

// Render parses src and renders it to tview color-tagged text. It is the
// convenience entry point for callers that do not need the intermediate []Block.
func Render(src string, opts Options) string {
	blocks, err := Parse(src)
	if err != nil || len(blocks) == 0 {
		return src
	}
	return RenderTView(blocks, opts)
}

// RenderPartial renders src for use during live streaming. Complete lines
// (terminated by \n) are passed through goldmark; the last incomplete line
// is appended as bracket-escaped raw text so tview's SetDynamicColors does
// not consume markdown link syntax or inline code brackets.
//
// When the stream flushes, call Render on the full text to replace the
// partial output with properly formatted content.
func RenderPartial(src string, opts Options) string {
	lastNL := strings.LastIndex(src, "\n")
	if lastNL < 0 {
		return escapeBrackets(src)
	}
	complete := src[:lastNL]
	pending := src[lastNL+1:]

	rendered := Render(complete, opts)
	if pending == "" {
		return rendered
	}
	if rendered == "" {
		return escapeBrackets(pending)
	}
	return rendered + "\n" + escapeBrackets(pending)
}

func escapeBrackets(s string) string {
	return strings.ReplaceAll(s, "[", "[[]")
}
