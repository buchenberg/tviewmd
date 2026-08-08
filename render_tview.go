package tviewmd

import (
	"fmt"
	"strings"

	"github.com/mattn/go-runewidth"
)

// RenderTView renders parsed blocks to tview color-tagged text suitable for a
// tview.TextView configured with SetWrap(true). The output contains tview
// color tags ([fg:bg:flags] ... [-:-:-]); the TextView performs word-wrapping.
func RenderTView(blocks []Block, opts Options) string {
	if opts.Theme == (Theme{}) {
		opts.Theme = DefaultTheme()
	}
	width := opts.Width
	if width <= 0 {
		width = 80
	}
	var b strings.Builder
	for i, blk := range blocks {
		if i > 0 {
			b.WriteString("\n")
		}
		renderBlock(&b, blk, opts, width, 0)
	}
	return strings.TrimRight(b.String(), "\n")
}

func renderBlock(b *strings.Builder, blk Block, opts Options, width, depth int) {
	switch blk.Kind {
	case BlockHeading:
		col := opts.Theme.Heading[0]
		if blk.Level >= 1 && blk.Level <= 6 {
			col = opts.Theme.Heading[blk.Level-1]
		}
		b.WriteString(tag(col, "", "b"))
		b.WriteString(renderSegments(blk.Segments, opts))
		b.WriteString(closeTag())
		b.WriteString("\n")
	case BlockParagraph:
		b.WriteString(renderSegments(blk.Segments, opts))
		b.WriteString("\n")
	case BlockThematicBreak:
		b.WriteString(tag(opts.Theme.Hr, "", ""))
		b.WriteString(strings.Repeat("─", max(3, width)))
		b.WriteString(closeTag())
		b.WriteString("\n")
	case BlockCodeBlock:
		b.WriteString(renderCode(blk.Code, opts))
		b.WriteString("\n")
	case BlockBlockquote:
		b.WriteString(tag(opts.Theme.QuoteFG, "", "i"))
		b.WriteString("▎ ")
		b.WriteString(renderSegments(blk.Segments, opts))
		b.WriteString(closeTag())
		b.WriteString("\n")
	case BlockList:
		renderList(b, blk.Items, opts, depth)
	case BlockTable:
		renderTable(b, blk.Table, opts)
	}
}

func renderList(b *strings.Builder, items []ListItem, opts Options, depth int) {
	indent := strings.Repeat("  ", depth)
	for _, it := range items {
		marker := "• "
		if it.Ordered {
			marker = fmt.Sprintf("%d. ", it.Index)
		}
		b.WriteString(indent)
		b.WriteString(marker)
		b.WriteString(renderSegments(it.Segments, opts))
		b.WriteString("\n")
		if len(it.Children) > 0 {
			renderList(b, it.Children, opts, depth+1)
		}
	}
}

func renderCode(cb CodeBlock, opts Options) string {
	source := strings.TrimRight(cb.Source, "\n")
	fg := opts.Theme.CodeBlockFG
	if cb.Lang != "" && !opts.NoHighlight {
		return highlightCode(cb.Lang, source, fg)
	}
	return tag(fg, "", "") + source + closeTag()
}

func renderTable(b *strings.Builder, t TableData, opts Options) {
	cols := tableCols(t)
	if cols == 0 {
		return
	}
	widths := make([]int, cols)
	for i, c := range t.Header {
		if i < cols {
			widths[i] = max(widths[i], cellPlainWidth(c))
		}
	}
	for _, row := range t.Rows {
		for i := 0; i < cols && i < len(row); i++ {
			widths[i] = max(widths[i], cellPlainWidth(row[i]))
		}
	}
	// Header row (bold).
	if len(t.Header) > 0 {
		b.WriteString(tag("", "", "b"))
		b.WriteString(renderTableRow(t.Header, widths))
		b.WriteString(closeTag())
		b.WriteString("\n")
	}
	// Separator.
	b.WriteString(tag(opts.Theme.Hr, "", ""))
	for i := 0; i < cols; i++ {
		if i > 0 {
			b.WriteString(" ")
		}
		b.WriteString(strings.Repeat("─", widths[i]))
	}
	b.WriteString(closeTag())
	b.WriteString("\n")
	// Body.
	for _, row := range t.Rows {
		padded := make([]TableCell, cols)
		for i := 0; i < cols; i++ {
			if i < len(row) {
				padded[i] = row[i]
			}
		}
		b.WriteString(renderTableRow(padded, widths))
		b.WriteString("\n")
	}
}

// renderTableRow emits one row as space-separated, padded cells. The visible
// content is styled via renderSegments; padding is computed from the plain-text
// width so color tags do not affect column alignment.
func renderTableRow(cells []TableCell, widths []int) string {
	var b strings.Builder
	for i, c := range cells {
		if i > 0 {
			b.WriteString(" ")
		}
		content := renderSegments(c.Segments, Options{})
		plainW := cellPlainWidth(c)
		colW := 0
		if i < len(widths) {
			colW = widths[i]
		}
		left, right := alignPad(c.Align, plainW, colW)
		b.WriteString(strings.Repeat(" ", left))
		b.WriteString(content)
		b.WriteString(strings.Repeat(" ", right))
	}
	return b.String()
}

func alignPad(align TextAlign, contentW, colW int) (left, right int) {
	if contentW >= colW {
		return 0, 0
	}
	gap := colW - contentW
	switch align {
	case AlignRight:
		return gap, 0
	case AlignCenter:
		return gap / 2, gap - gap/2
	default:
		return 0, gap
	}
}

func tableCols(t TableData) int {
	n := len(t.Header)
	for _, row := range t.Rows {
		if len(row) > n {
			n = len(row)
		}
	}
	return n
}

func cellPlainWidth(c TableCell) int {
	n := 0
	for _, s := range c.Segments {
		n += runewidth.StringWidth(s.Text)
	}
	return n
}

// renderSegments emits inline content for a block/cell. It accepts Options only
// for theming of code spans and links within the run.
func renderSegments(segs []Segment, opts Options) string {
	var b strings.Builder
	for _, s := range segs {
		b.WriteString(renderSegment(s, opts))
	}
	return b.String()
}

func renderSegment(s Segment, opts Options) string {
	if s.Text == "" {
		return ""
	}
	if s.Link != "" {
		col := opts.Theme.Link
		if opts.Theme == (Theme{}) {
			col = DefaultTheme().Link
		}
		if s.Text == s.Link {
			return tag(col, "", "u") + s.Text + closeTag()
		}
		return tag(col, "", "u") + s.Text + closeTag() + " " + tag(col, "", "") + "(" + s.Link + ")" + closeTag()
	}
	if s.Code {
		return tag(opts.Theme.InlineCodeFG, opts.Theme.InlineCodeBG, "") + s.Text + closeTag()
	}
	if s.Style == (Style{}) {
		return s.Text
	}
	return tag(s.Style.FG, s.Style.BG, flagsOf(s.Style)) + s.Text + closeTag()
}

// tag formats a tview color tag [fg:bg:flags]. Empty fields are left blank so
// tview inherits the surrounding style.
func tag(fg, bg, flags string) string {
	return fmt.Sprintf("[%s:%s:%s]", fg, bg, flags)
}

// closeTag resets all style attributes.
func closeTag() string { return "[-:-:-]" }

func flagsOf(s Style) string {
	var f strings.Builder
	if s.Bold {
		f.WriteByte('b')
	}
	if s.Italic {
		f.WriteByte('i')
	}
	if s.Underline {
		f.WriteByte('u')
	}
	if s.Strikethrough {
		f.WriteByte('s')
	}
	if s.Dim {
		f.WriteByte('d')
	}
	return f.String()
}
