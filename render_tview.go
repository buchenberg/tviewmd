package tviewmd

import (
	"fmt"
	"strings"

	"github.com/rivo/tview"
	"github.com/rivo/uniseg"
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
	return tag(opts.Theme.CodeBlockFG, "", "") + tview.Escape(source) + closeTag()
}

//nolint:gocognit // Column widths, separators and padding are laid out together; splitting scatters the rules.
func renderTable(b *strings.Builder, t TableData, opts Options) {
	cols := tableCols(t)
	if cols == 0 {
		return
	}
	widths := make([]int, cols)
	for i, c := range t.Header {
		if i < cols {
			widths[i] = max(widths[i], cellPaintedWidth(c))
		}
	}
	for _, row := range t.Rows {
		for i := 0; i < cols && i < len(row); i++ {
			widths[i] = max(widths[i], cellPaintedWidth(row[i]))
		}
	}
	// Header row (bold).
	if len(t.Header) > 0 {
		b.WriteString(tag("", "", "b"))
		b.WriteString(renderTableRow(t.Header, widths, opts))
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
		b.WriteString(renderTableRow(padded, widths, opts))
		b.WriteString("\n")
	}
}

// renderTableRow emits one row as space-separated, padded cells. The visible
// content is styled via renderSegments; padding is computed from the painted
// width (segment text plus any link decoration renderSegmentGroup draws) so
// color tags do not affect column alignment.
func renderTableRow(cells []TableCell, widths []int, opts Options) string {
	var b strings.Builder
	for i, c := range cells {
		if i > 0 {
			b.WriteString(" ")
		}
		content := renderSegments(c.Segments, opts)
		paintedW := cellPaintedWidth(c)
		colW := 0
		if i < len(widths) {
			colW = widths[i]
		}
		left, right := alignPad(c.Align, paintedW, colW)
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

// solveColumnWidths decides the final display width of each column of t given
// budget -- the total screen cells available for column content, separators
// excluded. It is pure: no rendering, no I/O; the caller owns painting the
// planned widths.
//
// Columns start at natural width (max painted content over header+rows). When
// that fits, or there is no budget at all, the natural widths are the answer.
// Otherwise widths shrink one cell at a time from the widest column that still
// has slack above its floor, where the floor is the column's widest
// unbreakable token: a TextView wraps between tokens, so a narrower column
// could not be honored anyway. If even the floors do not fit, the floors
// themselves give way, largest first, down to 1 per non-empty column; below
// that the table overflows, which the caller/wrapper accepts. The
// widest-first rule is deterministic (leftmost wins ties) and each pass
// removes exactly one cell from the sum, so the loops cost O(total excess)
// and cannot spin -- column sums here are small table widths.
func solveColumnWidths(t TableData, budget int) []int {
	cols := tableCols(t)
	if cols == 0 {
		return nil
	}
	natural := make([]int, cols)
	for i, c := range t.Header {
		if i < cols {
			natural[i] = max(natural[i], cellPaintedWidth(c))
		}
	}
	for _, row := range t.Rows {
		for i := 0; i < cols && i < len(row); i++ {
			natural[i] = max(natural[i], cellPaintedWidth(row[i]))
		}
	}
	sum := 0
	for _, w := range natural {
		sum += w
	}
	// Nothing to fit: either no budget exists, or max-content already fits.
	if budget <= 0 || sum <= budget {
		return natural
	}
	floors := make([]int, cols)
	for i := 0; i < cols; i++ {
		floors[i] = colTokenFloor(t, i, natural[i])
	}
	widths := append([]int(nil), natural...)
	// Soft shrink: one cell per pass from the widest column above its floor
	// (leftmost on ties), spending the budget where it buys the least and
	// keeping identical tables on identical plans.
	for sum > budget {
		best := -1
		for i := 0; i < cols; i++ {
			if widths[i] > floors[i] && (best < 0 || widths[i] > widths[best]) {
				best = i
			}
		}
		if best < 0 {
			break // every column sits on its floor; the floors must give way
		}
		widths[best]--
		sum--
	}
	// Hard-break: reduce the floors themselves, largest first under the same
	// tie rule. A non-empty column never drops below 1 -- width 0 would hide
	// it entirely -- so with a budget under the non-empty column count the
	// loop bottoms out at all-ones and the remaining overflow is final.
	for sum > budget {
		best := -1
		for i := 0; i < cols; i++ {
			if widths[i] > 1 && (best < 0 || widths[i] > widths[best]) {
				best = i
			}
		}
		if best < 0 {
			break // all non-empty columns at 1; overflow accepted
		}
		widths[best]--
		sum--
	}
	return widths
}

// colTokenFloor returns the width of the widest unbreakable token in column
// col -- the width a TextView can never wrap below, since it breaks between
// whitespace-separated runs, not inside them. Token widths use uniseg, so
// they are cluster-aware exactly like cellPlainWidth. natural (the column's
// max-content width) decides emptiness: a column with no content anywhere
// floors at 0 so it claims no budget and gets no padding, while any column
// that paints something floors at 1, the smallest useful column.
func colTokenFloor(t TableData, col, natural int) int {
	widest := 0
	measure := func(c TableCell) {
		var b strings.Builder
		for _, s := range c.Segments {
			b.WriteString(s.Text)
		}
		// Fields splits on all whitespace, "\n" included, so hard-wrapped
		// cell text cannot smuggle one token across a line break.
		for _, tok := range strings.Fields(b.String()) {
			widest = max(widest, uniseg.StringWidth(tok))
		}
	}
	if col < len(t.Header) {
		measure(t.Header[col])
	}
	for _, row := range t.Rows {
		if col < len(row) {
			measure(row[col])
		}
	}
	if natural == 0 {
		return 0
	}
	return max(widest, 1)
}

// cellPlainWidth returns the display width of a cell's visible text. It is
// measured with uniseg over the concatenated segment text: uniseg is
// grapheme-cluster aware, which is what tview/tcell use when painting a cell.
// Per-rune width arithmetic (go-runewidth) under-counts any emoji-presentation
// sequence -- a base rune plus U+FE0F (VS16) is zero-width on its own and only
// promotes the preceding cluster to two cells, which summing runes can never
// see (1 + 0 = 1). Under-counting a column makes every row that contains such a
// cell one cell wider than its separator, so borders come out ragged.
//
// The segments are joined before measuring so a cluster split across segment
// boundaries (for example a combining mark that starts a new segment) is still
// measured once.
func cellPlainWidth(c TableCell) int {
	var b strings.Builder
	for _, s := range c.Segments {
		b.WriteString(s.Text)
	}
	return uniseg.StringWidth(b.String())
}

// cellPaintedWidth returns the display width of everything a cell paints, not
// just its segment text: renderSegmentGroup appends a visible " (url)"
// decoration after any link group whose label differs from its target, so a
// column planned from cellPlainWidth alone comes out one decoration narrower
// than the rows actually painted in it, and header, separator and body rows
// stop lining up. This is a paint/measure coupling: layout must measure
// exactly the glyphs the renderer draws, so this function walks the same
// sameSegmentTreatment groups renderSegments walks and applies
// renderSegmentGroup's decoration condition verbatim (link set, group text
// non-empty, group text != target). The decoration is measured in its
// unescaped form -- tview.Escape doubles '[' in the string while the screen
// paints a single '[' -- and uniseg keeps the count cluster-aware, matching
// the oracle tview/tcell paint with.
func cellPaintedWidth(c TableCell) int {
	w := cellPlainWidth(c)
	for i := 0; i < len(c.Segments); {
		j := i
		for j+1 < len(c.Segments) && sameSegmentTreatment(c.Segments[j], c.Segments[j+1]) {
			j++
		}
		group := c.Segments[i : j+1]
		var b strings.Builder
		for _, s := range group {
			b.WriteString(s.Text)
		}
		raw := b.String()
		if raw != "" && group[0].Link != "" && raw != group[0].Link {
			w += uniseg.StringWidth(" (" + group[0].Link + ")")
		}
		i = j + 1
	}
	return w
}

// renderSegments emits inline content for a block/cell. Adjacent segments that
// share the same treatment (link, code, style) are merged and their visible
// text escaped as a single unit. Per-segment escaping is insufficient: goldmark
// splits a failed link like "[red]boom" into "[", "red", "]boom", and tview
// consumes a "[...]" tag shape across segment boundaries, so the full run must
// be escaped together or the bracket pair would inject a style directive.
func renderSegments(segs []Segment, opts Options) string {
	var b strings.Builder
	for i := 0; i < len(segs); {
		j := i
		for j+1 < len(segs) && sameSegmentTreatment(segs[j], segs[j+1]) {
			j++
		}
		b.WriteString(renderSegmentGroup(segs[i:j+1], opts))
		i = j + 1
	}
	return b.String()
}

// sameSegmentTreatment reports whether two segments render identically, so they
// can be merged for escaping purposes.
func sameSegmentTreatment(a, b Segment) bool {
	return a.Link == b.Link && a.Code == b.Code && a.Style == b.Style
}

// renderSegmentGroup renders a run of same-styled segments. The visible text is
// concatenated and passed through tview.Escape once, so tag-shaped bracket
// spans are neutralized regardless of how goldmark split them. Only the tags
// this function itself emits are left unescaped.
func renderSegmentGroup(group []Segment, opts Options) string {
	var tb strings.Builder
	for _, s := range group {
		tb.WriteString(s.Text)
	}
	raw := tb.String()
	if raw == "" {
		return ""
	}
	text := tview.Escape(raw)
	s0 := group[0]
	if s0.Link != "" {
		col := opts.Theme.Link
		if opts.Theme == (Theme{}) {
			col = DefaultTheme().Link
		}
		if raw == s0.Link {
			return tag(col, "", "u") + text + closeTag()
		}
		return tag(col, "", "u") + text + closeTag() + " " + tag(col, "", "") + "(" + tview.Escape(s0.Link) + ")" + closeTag()
	}
	if s0.Code {
		return tag(opts.Theme.InlineCodeFG, opts.Theme.InlineCodeBG, "") + text + closeTag()
	}
	if s0.Style == (Style{}) {
		return text
	}
	return tag(s0.Style.FG, s0.Style.BG, flagsOf(s0.Style)) + text + closeTag()
}

// tag formats a tview color tag [fg:bg:flags]. Empty fields inherit the
// current style. All three fields are always present with colon separators
// so closeTag() reliably resets the same channels across all tags.
func tag(fg, bg, flags string) string {
	return fmt.Sprintf("[%s:%s:%s]", fg, bg, flags)
}

// closeTag resets all style attributes including flags (bold/underline/italic)
// and color. The trailing [::U] is required because tcell stores the underline
// style in a separate field (Style.ulStyle) that tview's [-:-:-] reset does not
// touch — without it, any underlined run (e.g. links) leaks its underline to
// every cell rendered afterwards. [::U] maps to tcell's Underline(false), which
// is the only tag that clears ulStyle back to UnderlineStyleNone.
func closeTag() string { return "[-:-:-][::U]" }

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
