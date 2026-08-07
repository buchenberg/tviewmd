package tviewmd

import (
	"bytes"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/rivo/tview"
)

// highlighter holds lazily-initialized chroma formatter + style. Chroma's
// registry is process-global, so one shared instance is correct.
type highlighter struct {
	formatter chroma.Formatter
	style     *chroma.Style
}

var hl = func() *highlighter {
	return &highlighter{
		formatter: formatters.Get("terminal256"),
		style:     styles.Get("monokai"),
	}
}()

// highlightCode returns tview-tagged, syntax-highlighted source for the given
// language. If the language is unknown or highlighting fails, it falls back to
// the code-block foreground color with no highlighting.
func highlightCode(lang, source, fg string) string {
	if hl.formatter == nil || source == "" {
		return tag(fg, "", "") + source + closeTag()
	}
	lexer := lexers.Get(lang)
	if lexer == nil {
		lexer = lexers.Analyse(source)
	}
	if lexer == nil {
		return tag(fg, "", "") + source + closeTag()
	}
	lexer = chroma.Coalesce(lexer)
	it, err := lexer.Tokenise(nil, source)
	if err != nil {
		return tag(fg, "", "") + source + closeTag()
	}
	var buf bytes.Buffer
	if err := hl.formatter.Format(&buf, hl.style, it); err != nil {
		return tag(fg, "", "") + source + closeTag()
	}
	return tview.TranslateANSI(buf.String())
}
