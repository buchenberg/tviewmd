package tviewmd

import (
	"fmt"
	"slices"
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
		ambient := tag(col, "", "b")
		b.WriteString(ambient)
		b.WriteString(renderSegments(blk.Segments, opts, ambient))
		b.WriteString(closeTag())
		b.WriteString("\n")
	case BlockParagraph:
		b.WriteString(renderSegments(blk.Segments, opts, ""))
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
		ambient := tag(opts.Theme.QuoteFG, "", "i")
		b.WriteString(ambient)
		b.WriteString("▎ ")
		b.WriteString(renderSegments(blk.Segments, opts, ambient))
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
		b.WriteString(renderSegments(it.Segments, opts, ""))
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
	// Width-constrained tables: with a width budget, plan column widths that
	// fit it; when the plan differs from max-content, rows re-flow as
	// multi-line grids instead of leaving tview to fold them as prose.
	if opts.Width > 0 {
		// A width budget below the separator overhead still constrains the
		// table to the narrowest legal plan (all columns at 1) rather than
		// silently falling back to max-content: solveColumnWidths reads a
		// budget of 0 as "no constraint", which is the unset-Width sentinel.
		budget := max(opts.Width-(cols-1), 1)
		if planned := solveColumnWidths(t, budget); !slices.Equal(planned, widths) {
			renderTableWrapped(b, t, planned, opts)
			return
		}
	}
	// Header row (bold).
	if len(t.Header) > 0 {
		ambient := tag("", "", "b")
		b.WriteString(ambient)
		b.WriteString(renderTableRow(t.Header, widths, opts, ambient))
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
		b.WriteString(renderTableRow(padded, widths, opts, ""))
		b.WriteString("\n")
	}
}

// renderTableRow emits one row as space-separated, padded cells. The visible
// content is styled via renderSegments; padding is computed from the painted
// width (segment text plus any link decoration renderSegmentGroup draws) so
// color tags do not affect column alignment.
func renderTableRow(cells []TableCell, widths []int, opts Options, ambient string) string {
	var b strings.Builder
	for i, c := range cells {
		if i > 0 {
			b.WriteString(" ")
		}
		content := renderSegments(c.Segments, opts, ambient)
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
	natural := naturalColumnWidths(t)
	sum := 0
	for _, w := range natural {
		sum += w
	}
	// Nothing to fit: either no budget exists, or max-content already fits.
	if budget <= 0 || sum <= budget {
		return natural
	}
	floors := make([]int, len(natural))
	for i := range floors {
		floors[i] = colTokenFloor(t, i, natural[i])
	}
	// Soft shrink, then hard-break. In both phases stepReduce takes one cell
	// per pass from the widest column still above its limit -- the floors for
	// the soft phase (spending the budget where it buys the least and keeping
	// identical tables on identical plans), then the hard-break minimum of 1
	// when even the floors must give way. Width 0 would hide a non-empty
	// column entirely, so the second loop bottoms out at all-ones and any
	// remaining overflow is final.
	widths := slices.Clone(natural)
	for sum > budget && stepReduce(widths, floors) {
		sum--
	}
	hard := make([]int, len(widths))
	for i := range hard {
		hard[i] = 1
	}
	for sum > budget && stepReduce(widths, hard) {
		sum--
	}
	return widths
}

// stepReduce takes one cell from the widest column whose width exceeds
// lim[i] (leftmost on ties) and reports whether it found one; the caller
// stops at a phase's limit when it returns false. Deterministic by
// construction -- the widest-first rule with leftmost tie-breaking.
func stepReduce(widths, lim []int) bool {
	best := -1
	for i := range widths {
		if widths[i] > lim[i] && (best < 0 || widths[i] > widths[best]) {
			best = i
		}
	}
	if best < 0 {
		return false
	}
	widths[best]--
	return true
}

// naturalColumnWidths returns each column's max painted width over the
// header and all body rows -- the max-content plan the width-constrained
// layout shrinks from, and the direct plan when no budget exists.
func naturalColumnWidths(t TableData) []int {
	cols := tableCols(t)
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
	return widths
}

// colTokenFloor returns the width of the widest unbreakable run in column col
// -- the width a wrapped layout cannot go below, since it breaks between
// words, not inside them. Runs are measured from cellTokens, so link
// decorations count exactly as they paint (a URL never shrinks until the
// hard-break phase) and token boundaries match the wrapper's own splitting;
// a strings.Fields-style split would disagree on non-ASCII whitespace and
// would miss decorations entirely. natural (the column's max-content width)
// decides emptiness: a column that paints nothing floors at 0, while any
// column that paints something floors at 1.
func colTokenFloor(t TableData, col, natural int) int {
	widest := 0
	measure := func(c TableCell) {
		run := 0
		for _, tok := range cellTokens(c) {
			tw := uniseg.StringWidth(tok.text)
			if tok.glue {
				run += tw // glued fragments continue the same unbreakable word
				continue
			}
			widest = max(widest, run)
			run = tw
		}
		widest = max(widest, run)
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

// renderTableWrapped paints a table whose planned column widths are narrower
// than max-content. Each cell wraps within its column at word boundaries, a
// logical row grows to as many physical lines as its tallest cell needs, and
// every physical line keeps the space-separated column skeleton, so the
// columns stay aligned without relying on tview to wrap anything.
func renderTableWrapped(b *strings.Builder, t TableData, widths []int, opts Options) {
	cols := len(widths)
	if len(t.Header) > 0 {
		ambient := tag("", "", "b")
		for _, line := range renderWrappedRow(t.Header, widths, opts, ambient) {
			b.WriteString(ambient)
			b.WriteString(line)
			b.WriteString(closeTag())
			b.WriteString("\n")
		}
	}
	b.WriteString(tag(opts.Theme.Hr, "", ""))
	for i := 0; i < cols; i++ {
		if i > 0 {
			b.WriteString(" ")
		}
		b.WriteString(strings.Repeat("─", widths[i]))
	}
	b.WriteString(closeTag())
	b.WriteString("\n")
	for _, row := range t.Rows {
		padded := make([]TableCell, cols)
		for i := 0; i < cols && i < len(row); i++ {
			padded[i] = row[i]
		}
		for _, line := range renderWrappedRow(padded, widths, opts, "") {
			b.WriteString(line)
			b.WriteString("\n")
		}
	}
}

// cellToken is one unbreakable piece of a cell's painted content in the
// width-constrained layout. text never contains ASCII whitespace. pre marks a
// forced line break before this token (a "\n" in the source cell text); glue
// marks a token that continues the previous one with no intervening source
// whitespace -- goldmark splits styled mid-word runs ("**bo**ld" -> "bo"+"ld")
// and adjacent segments paint back-to-back, so a glued token must not get a
// space painted before it and must not be separated from its predecessor by
// wrapping unless nothing else fits.
type cellToken struct {
	text  string
	seg   Segment
	pre   bool
	glue  bool
	plain bool // decoration token: link color without the label's underline
}

// isASCIISpaceByte reports whether b is an ASCII whitespace byte. Non-ASCII
// whitespace (e.g. U+00A0) is deliberately NOT a token boundary -- it is
// measured as painted content, and a token containing it simply hard-breaks
// at cluster boundaries if it exceeds its column.
func isASCIISpaceByte(b byte) bool {
	switch b {
	case ' ', '\t', '\n', '\r', '\v', '\f':
		return true
	}
	return false
}

// cellTokens flattens a cell into word-level tokens carrying their source
// segment's treatment. Runs of ASCII whitespace separate tokens (runs
// containing "\n" force a break before the next token); whitespace itself is
// dropped and re-synthesized as single spaces when lines are joined, so a
// re-flowed cell may paint fewer separating spaces than its source had. Link
// decorations are emitted as explicit "(url)" tokens after each link run,
// exactly where renderSegments would paint them, using its condition verbatim.
func cellTokens(c TableCell) []cellToken {
	var toks []cellToken
	var open *openToken
	carryGlue := false
	pendingBreak := false
	// linkText accumulates the raw (untrimmed) text of the current link run,
	// mirroring the run walk in renderSegments, to evaluate the decoration
	// condition once per run rather than once per treatment group.
	var linkText strings.Builder
	for si, s := range c.Segments {
		if si > 0 && !sameSegmentTreatment(c.Segments[si-1], s) {
			// The previous treatment group ends mid-cell. Its still-open token
			// belongs to that group and must be flushed BEFORE the group's
			// decoration token is appended: the decoration paints after the
			// link text it labels, never before it.
			toks = flushOpen(toks, open)
			open = nil
			prev := c.Segments[si-1]
			if prev.Link != s.Link {
				// The link run ends here; a label may span several treatment
				// groups ("[**do**cs](u)"), so only the run's last group gets
				// the decoration.
				appendDecoration(&toks, linkText.String(), prev)
				linkText.Reset()
			}
		}
		linkText.WriteString(s.Text)
		toks, open, pendingBreak, carryGlue = scanSegmentTokens(toks, open, s, pendingBreak, carryGlue)
	}
	// Flush the final group; the carry's life ends here, so no reseat.
	toks = flushOpen(toks, open)
	if n := len(c.Segments); n > 0 {
		appendDecoration(&toks, linkText.String(), c.Segments[n-1])
	}
	return toks
}

// appendDecoration appends a link run's decoration token when renderSegments'
// emission condition holds (run text non-empty, link set, and the text not the
// bare link target). Mirroring that condition verbatim keeps measurement and
// paint coupled. The token is plain: the fast path paints the decoration in
// the link color without the label's underline, and the wrapped layout must
// match.
func appendDecoration(toks *[]cellToken, raw string, seg Segment) {
	if raw != "" && seg.Link != "" && raw != seg.Link {
		plain := cellToken{text: "(" + seg.Link + ")", seg: seg, plain: true}
		*toks = append(*toks, plain)
	}
}

// openToken is the token currently being assembled by scanSegmentTokens. Its
// text buffers byte-exactly: a strings.Builder copies the source bytes, while
// converting a single byte to a string would re-encode its value as a rune
// and corrupt every multi-byte sequence ("caf\u00e9" -> "caf\u00c3\u00a9").
type openToken struct {
	seg       Segment
	pre, glue bool
	b         strings.Builder
}

// flushOpen appends the open token, if any, to toks; the caller clears the
// carry by assigning nil.
func flushOpen(toks []cellToken, open *openToken) []cellToken {
	if open == nil {
		return toks
	}
	tok := cellToken{text: open.b.String(), seg: open.seg, pre: open.pre, glue: open.glue}
	return append(toks, tok)
}

// scanSegmentTokens consumes one segment's text into tokens, threading the
// carry state across segment boundaries: open is the token under
// construction (nil at a word boundary), carryGlue says the next token would
// continue the current word with no source whitespace between (a mid-word
// style change: the boundary flush in cellTokens already split it, and the
// new token glues so no phantom space appears), and pendingBreak says a
// "\n" is waiting to force a break before the next token.
func scanSegmentTokens(
	toks []cellToken, open *openToken, s Segment, pendingBreak, carryGlue bool,
) ([]cellToken, *openToken, bool, bool) {
	for i := 0; i < len(s.Text); i++ {
		ch := s.Text[i]
		if isASCIISpaceByte(ch) {
			toks = flushOpen(toks, open)
			open = nil
			if ch == '\n' {
				pendingBreak = true
			}
			carryGlue = false
			continue
		}
		if open == nil {
			open = &openToken{seg: s, pre: pendingBreak, glue: carryGlue}
			pendingBreak = false
		}
		open.b.WriteByte(ch)
		carryGlue = true
	}
	return toks, open, pendingBreak, carryGlue
}

// splitWideToken splits text into chunks of at most width display cells at
// grapheme-cluster boundaries -- the hard-break of last resort for tokens
// wider than their column. A single cluster wider than width (a two-cell
// emoji in a one-cell column) is emitted on its own line and accepted as
// overflow; the loop always terminates because every iteration consumes at
// least one cluster.
func splitWideToken(text string, width int) []string {
	if width <= 0 || uniseg.StringWidth(text) <= width {
		return []string{text}
	}
	var chunks []string
	var cur strings.Builder
	curW := 0
	state := -1
	rest := text
	for len(rest) > 0 {
		var cluster string
		var w int
		cluster, rest, w, state = uniseg.FirstGraphemeClusterInString(rest, state)
		if curW+w > width && cur.Len() > 0 {
			chunks = append(chunks, cur.String())
			cur.Reset()
			curW = 0
		}
		cur.WriteString(cluster)
		curW += w
	}
	if cur.Len() > 0 {
		chunks = append(chunks, cur.String())
	}
	return chunks
}

// wrapUnit is one placement unit for the line builder: a complete word --
// possibly assembled from several style-fragment tokens joined by glue -- or a
// single fragment when the word is too wide for the column. Moving glued
// fragments as one unit keeps styled words whole whenever a legal plan holds
// them ("**bo**ld" never emits "a bo" / "ld").
type wrapUnit struct {
	toks []cellToken
	w    int // painted width; glued fragments touch, so no inner gaps
	glue bool
	pre  bool
}

// wrapUnits groups tokens into placement units: consecutive glued tokens are
// fragments of one word and must move together when the word fits a line. A
// word wider than the column falls back to one unit per fragment, letting the
// greedy pass split it where it must.
func wrapUnits(toks []cellToken, width int) []wrapUnit {
	units := make([]wrapUnit, 0, len(toks))
	for i := 0; i < len(toks); {
		j := i
		w := uniseg.StringWidth(toks[i].text)
		for j+1 < len(toks) && toks[j+1].glue {
			j++
			w += uniseg.StringWidth(toks[j].text)
		}
		if j > i && w > width {
			for k := i; k <= j; k++ {
				units = append(units, wrapUnit{
					toks: toks[k : k+1],
					w:    uniseg.StringWidth(toks[k].text),
					glue: toks[k].glue,
					pre:  toks[k].pre,
				})
			}
			i = j + 1
			continue
		}
		units = append(units, wrapUnit{toks: toks[i : j+1], w: w, glue: toks[i].glue, pre: toks[i].pre})
		i = j + 1
	}
	return units
}

// wrapTokens lays tokens out into lines of at most width display cells.
// Breaks happen between placement units (see wrapUnits); a gap cost is only
// charged when something already precedes the unit on its line, so a
// line-initial unit costs its own width alone. A token wider than width is
// cluster-split up front by preSplitWideTokens; width <= 0 disables wrapping
// entirely (one line, whatever it costs).
func wrapTokens(toks []cellToken, width int) [][]cellToken {
	if width <= 0 {
		return [][]cellToken{toks}
	}
	units := wrapUnits(preSplitWideTokens(toks, width), width)
	var lines [][]cellToken
	var cur []cellToken
	curW := 0
	for _, u := range units {
		gap := 0
		if len(cur) > 0 && !u.glue {
			gap = 1
		}
		if len(cur) > 0 && (u.pre || curW+gap+u.w > width) {
			lines = append(lines, cur)
			cur = nil
			curW = 0
			gap = 0
		}
		cur = append(cur, u.toks...)
		curW += gap + u.w
	}
	if len(cur) > 0 || len(lines) == 0 {
		lines = append(lines, cur)
	}
	return lines
}

// preSplitWideTokens returns toks with every token wider than width split
// into cluster-bounded chunks. Every non-final chunk fills a line of its own
// downstream; the final chunk joins the greedy pass like any other token.
// Because a split token's first chunk fills the width, it can never share a
// line with predecessors -- the same layout as splitting inline, without the
// loop nesting in the line builder.
func preSplitWideTokens(toks []cellToken, width int) []cellToken {
	out := make([]cellToken, 0, len(toks))
	for _, tok := range toks {
		if uniseg.StringWidth(tok.text) <= width {
			out = append(out, tok)
			continue
		}
		for k, chunk := range splitWideToken(tok.text, width) {
			out = append(out, cellToken{
				text:  chunk,
				seg:   tok.seg,
				pre:   tok.pre && k == 0,
				glue:  tok.glue && k == 0,
				plain: tok.plain,
			})
		}
	}
	return out
}

// tokenLinePaintedWidth returns the display width a painted line of tokens
// occupies: token widths plus one cell for every inter-token space (glued
// tokens paint none). It must agree with what renderTokenLine paints.
func tokenLinePaintedWidth(toks []cellToken) int {
	w := 0
	for i, tok := range toks {
		if i > 0 && !tok.glue {
			w++
		}
		w += uniseg.StringWidth(tok.text)
	}
	return w
}

// renderTokenGroup renders one same-treatment run of tokens as a single
// escaped unit. It mirrors renderSegmentGroup's style branches exactly but
// never appends a link decoration: in the wrapped layout the decoration is an
// explicit token (cellTokens), so appending one here would paint it twice.
// plain selects the decoration's styling: link color without the underline
// the label carries.
func renderTokenGroup(s0 Segment, plain bool, raw string, opts Options, ambient string) string {
	if raw == "" {
		return ""
	}
	text := tview.Escape(raw)
	if s0.Link != "" {
		col := opts.Theme.Link
		if opts.Theme == (Theme{}) {
			col = DefaultTheme().Link
		}
		flags := "u"
		if plain {
			flags = ""
		}
		return tag(col, "", flags) + text + closeTag() + ambient
	}
	if s0.Code {
		return tag(opts.Theme.InlineCodeFG, opts.Theme.InlineCodeBG, "") + text + closeTag() + ambient
	}
	if s0.Style == (Style{}) {
		return text
	}
	return tag(s0.Style.FG, s0.Style.BG, flagsOf(s0.Style)) + text + closeTag() + ambient
}

// renderTokenLine paints one physical line of tokens: consecutive tokens with
// the same treatment merge and escape as a single unit (the same merged-escape
// rule as renderSegments -- a tag-shaped bracket pair must be neutralized per
// line regardless of how goldmark split it), with one space between tokens
// except where glue says the source had none.
func renderTokenLine(toks []cellToken, opts Options, ambient string) string {
	var b strings.Builder
	for i := 0; i < len(toks); {
		j := i
		for j+1 < len(toks) && sameSegmentTreatment(toks[j].seg, toks[j+1].seg) && toks[j].plain == toks[j+1].plain {
			j++
		}
		group := toks[i : j+1]
		var tb strings.Builder
		for k, tok := range group {
			if k > 0 && !tok.glue {
				tb.WriteString(" ")
			}
			tb.WriteString(tok.text)
		}
		if i > 0 && !group[0].glue {
			b.WriteString(" ")
		}
		b.WriteString(renderTokenGroup(group[0].seg, group[0].plain, tb.String(), opts, ambient))
		i = j + 1
	}
	return b.String()
}

// renderWrappedRow renders one logical row as its physical lines: every cell
// wraps to its column width, the row is as tall as its tallest cell, and each
// line pads every cell to that cell's column width so the columns line up on
// every painted line. Alignment honors the cell's own Align, like the
// unwrapped renderTableRow.
func renderWrappedRow(cells []TableCell, widths []int, opts Options, ambient string) []string {
	cellLines := make([][][]cellToken, len(cells))
	rowH := 1
	for i, c := range cells {
		w := 1
		if i < len(widths) {
			w = max(widths[i], 1)
		}
		cellLines[i] = wrapTokens(cellTokens(c), w)
		rowH = max(rowH, len(cellLines[i]))
	}
	out := make([]string, rowH)
	for p := 0; p < rowH; p++ {
		var b strings.Builder
		for i := range cells {
			if i > 0 {
				b.WriteString(" ")
			}
			content := ""
			pw := 0
			if p < len(cellLines[i]) {
				content = renderTokenLine(cellLines[i][p], opts, ambient)
				pw = tokenLinePaintedWidth(cellLines[i][p])
			}
			colW := 0
			if i < len(widths) {
				colW = widths[i]
			}
			left, right := alignPad(cells[i].Align, pw, colW)
			b.WriteString(strings.Repeat(" ", left))
			b.WriteString(content)
			b.WriteString(strings.Repeat(" ", right))
		}
		out[p] = b.String()
	}
	return out
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
	return uniseg.StringWidth(segmentsText(c.Segments))
}

// cellPaintedWidth returns the display width of everything a cell paints, not
// just its segment text: renderSegments appends a visible " (url)" decoration
// after any link run whose label differs from its target, so a column planned
// from cellPlainWidth alone comes out one decoration narrower than the rows
// actually painted in it, and header, separator and body rows stop lining up.
// This is a paint/measure coupling: layout must measure exactly the glyphs the
// renderer draws, so this function walks the same link runs renderSegments
// walks and applies its decoration condition verbatim (link set, run text
// non-empty, run text != target). The decoration is measured in its unescaped
// form -- tview.Escape doubles '[' in the string while the screen paints a
// single '[' -- and uniseg keeps the count cluster-aware, matching the oracle
// tview/tcell paint with.
func cellPaintedWidth(c TableCell) int {
	w := cellPlainWidth(c)
	for i := 0; i < len(c.Segments); {
		link := c.Segments[i].Link
		end := linkRunEnd(c.Segments, i)
		if link != "" {
			if raw := segmentsText(c.Segments[i : end+1]); raw != "" && raw != link {
				w += uniseg.StringWidth(" (" + link + ")")
			}
		}
		i = end + 1
	}
	return w
}

// renderSegments emits inline content for a block/cell. Adjacent segments that
// share the same treatment (link, code, style) are merged and their visible
// text escaped as a single unit. Per-segment escaping is insufficient: goldmark
// splits a failed link like "[red]boom" into "[", "red", "]boom", and tview
// consumes a "[...]" tag shape across segment boundaries, so the full run must
// be escaped together or the bracket pair would inject a style directive.
func renderSegments(segs []Segment, opts Options, ambient string) string {
	var b strings.Builder
	runStart := 0
	for i := 0; i < len(segs); {
		j := i
		for j+1 < len(segs) && sameSegmentTreatment(segs[j], segs[j+1]) {
			j++
		}
		b.WriteString(renderSegmentGroup(segs[i:j+1], opts, ambient))
		// A link's decoration belongs after the label's last visible text, not
		// after each treatment group: "[**do**cs](u)" is one link split into
		// "do" and "cs", and must paint "docs (u)", once.
		if link := segs[i].Link; link != "" && (j+1 == len(segs) || segs[j+1].Link != link) {
			if raw := segmentsText(segs[runStart : j+1]); raw != "" && raw != link {
				b.WriteString(linkDecoration(link, opts))
				b.WriteString(ambient)
			}
		}
		if next := j + 1; next < len(segs) && segs[next].Link != segs[j].Link {
			runStart = next
		}
		i = j + 1
	}
	return b.String()
}

// segmentsText concatenates the visible text of segs.
func segmentsText(segs []Segment) string {
	var b strings.Builder
	for _, s := range segs {
		b.WriteString(s.Text)
	}
	return b.String()
}

// linkColor returns the theme's link color, falling back to the default theme
// when opts carries no theme.
func linkColor(opts Options) string {
	if opts.Theme == (Theme{}) {
		return DefaultTheme().Link
	}
	return opts.Theme.Link
}

// linkDecoration paints the " (url)" suffix shown after a link whose label
// differs from its target: link color, no label underline, and no trailing
// style state (the caller resumes any ambient style).
func linkDecoration(link string, opts Options) string {
	return " " + tag(linkColor(opts), "", "") + "(" + tview.Escape(link) + ")" + closeTag()
}

// linkRunEnd returns the index of the last segment in the maximal run of
// consecutive segments sharing the same non-empty link, or i when segs[i] has
// no link. A link label may span several treatment groups ("[**do**cs](u)"),
// and a link's decoration is emitted after the run, not after each group.
func linkRunEnd(segs []Segment, i int) int {
	if segs[i].Link == "" {
		return i
	}
	j := i
	for j+1 < len(segs) && segs[j+1].Link == segs[i].Link {
		j++
	}
	return j
}

// sameSegmentTreatment reports whether two segments render identically, so they
// can be merged for escaping purposes.
func sameSegmentTreatment(a, b Segment) bool {
	return a.Link == b.Link && a.Code == b.Code && a.Style == b.Style
}

// renderSegmentGroup renders a run of same-styled segments. The visible text is
// concatenated and passed through tview.Escape once, so tag-shaped bracket
// spans are neutralized regardless of how goldmark split them. Only the tags
// this function itself emits are left unescaped. Link decorations are emitted
// by the caller (renderSegments) once per link run.
func renderSegmentGroup(group []Segment, opts Options, ambient string) string {
	raw := segmentsText(group)
	if raw == "" {
		return ""
	}
	text := tview.Escape(raw)
	s0 := group[0]
	if s0.Link != "" {
		return tag(linkColor(opts), "", "u") + text + closeTag() + ambient
	}
	if s0.Code {
		return tag(opts.Theme.InlineCodeFG, opts.Theme.InlineCodeBG, "") + text + closeTag() + ambient
	}
	if s0.Style == (Style{}) {
		return text
	}
	return tag(s0.Style.FG, s0.Style.BG, flagsOf(s0.Style)) + text + closeTag() + ambient
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
