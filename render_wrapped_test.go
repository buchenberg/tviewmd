package tviewmd

import (
	"strings"
	"testing"

	"github.com/rivo/uniseg"
)

// This file covers the width-constrained ("wrapped") painter directly. Before
// it existed the wrapped branch was reached by a single end-to-end repro test
// and renderTokenGroup's link, code, and style branches never ran at all.

// lineVisible returns the visible text of one painted line with runs of spaces
// collapsed, so padding does not obscure word-level assertions.
func lineVisible(s string) string {
	return strings.Join(strings.Fields(stripTags(s)), " ")
}

// TestRenderWrappedCellStyles drives every styling branch of the wrapped
// painter: link, inline code, bold, and italic cells reflowed at a width that
// cannot hold them on one line.
func TestRenderWrappedCellStyles(t *testing.T) {
	t.Parallel()

	theme := DefaultTheme()
	cases := []struct {
		name    string
		segs    []Segment
		visible string // joined visible text of all physical lines
		tag     string // tag fragment the painted output must carry
	}{
		{
			name:    "link",
			segs:    []Segment{{Text: "label", Link: "https://x.io"}},
			visible: "label (https://x.io)",
			tag:     theme.Link,
		},
		{
			name:    "code",
			segs:    []Segment{{Text: "x := 1", Code: true}, {Text: " tail words here"}},
			visible: "x := 1 tail words here",
			tag:     theme.InlineCodeBG,
		},
		{
			name:    "bold",
			segs:    []Segment{{Text: "bold", Style: Style{Bold: true}}, {Text: " tail words here"}},
			visible: "bold tail words here",
			tag:     "::b]",
		},
		{
			name:    "italic",
			segs:    []Segment{{Text: "it", Style: Style{Italic: true}}, {Text: " tail words here"}},
			visible: "it tail words here",
			tag:     "::i]",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			lines := renderWrappedRow([]TableCell{{Segments: tc.segs}}, []int{16}, Options{Theme: theme}, "")
			if len(lines) < 2 {
				t.Fatalf("cell did not wrap: %q", lines)
			}
			var visible strings.Builder
			for i, line := range lines {
				if i > 0 {
					visible.WriteString(" ")
				}
				visible.WriteString(lineVisible(line))
				if got := uniseg.StringWidth(stripTags(line)); got > 16 {
					t.Errorf("line %d paints %d cells, want <= 16: %q", i, got, stripTags(line))
				}
			}
			if got := visible.String(); got != tc.visible {
				t.Errorf("visible text = %q, want %q", got, tc.visible)
			}
			if raw := strings.Join(lines, ""); !strings.Contains(raw, tc.tag) {
				t.Errorf("missing style tag %q in wrapped output:\n%s", tc.tag, raw)
			}
		})
	}
}

// TestRenderTokenGroupBranches pins the pieces the end-to-end wrapped tests
// cannot reach: the empty run, the zero-theme link fallback, and the generic
// style branch including the strikethrough/dim flags.
func TestRenderTokenGroupBranches(t *testing.T) {
	t.Parallel()

	if got := renderTokenGroup(Segment{}, false, "", Options{}, ""); got != "" {
		t.Errorf("empty run rendered %q, want empty", got)
	}
	if got := renderTokenGroup(Segment{Link: "u"}, false, "label", Options{}, ""); !strings.Contains(got, DefaultTheme().Link) {
		t.Errorf("zero theme did not fall back to the default link color: %q", got)
	}
	if got := renderTokenGroup(Segment{Code: true}, false, "c", Options{Theme: DefaultTheme()}, ""); !strings.Contains(got, DefaultTheme().InlineCodeBG) {
		t.Errorf("code run missing inline-code background: %q", got)
	}
	if got := renderTokenGroup(Segment{}, false, "x", Options{}, ""); got != "x" {
		t.Errorf("plain run rendered %q, want %q", got, "x")
	}
	styled := Segment{Style: Style{Bold: true, Strikethrough: true, Dim: true}}
	if got := renderTokenGroup(styled, false, "s", Options{}, ""); !strings.Contains(got, "::bsd]") {
		t.Errorf("style run missing bold/strikethrough/dim flags: %q", got)
	}
	styled = Segment{Style: Style{Italic: true, Underline: true}}
	if got := renderTokenGroup(styled, false, "s", Options{}, ""); !strings.Contains(got, "::iu]") {
		t.Errorf("style run missing italic/underline flags: %q", got)
	}
}

// TestLinkColorFallsBackToDefault covers the theme fallback used by the wrapped
// painter when a caller passes a zero Options.
func TestLinkColorFallsBackToDefault(t *testing.T) {
	t.Parallel()

	if got := linkColor(Options{}); got != DefaultTheme().Link {
		t.Errorf("linkColor(zero) = %q, want %q", got, DefaultTheme().Link)
	}
	if got := linkColor(Options{Theme: Theme{Link: "x"}}); got != "x" {
		t.Errorf("linkColor(custom) = %q, want %q", got, "x")
	}
}

// TestTokenLineWidthAgreesWithPaint pins the paint/measure pair: the row
// planner pads against tokenLinePaintedWidth, so it must count exactly the
// cells renderTokenLine emits for the same tokens -- including glue, the link
// decoration token, and code spans.
func TestTokenLineWidthAgreesWithPaint(t *testing.T) {
	t.Parallel()

	theme := DefaultTheme()
	cases := []struct {
		name string
		cell TableCell
	}{
		{"styled linked word", TableCell{Segments: []Segment{
			{Text: "do", Link: "u", Style: Style{Bold: true}},
			{Text: "cs", Link: "u"},
		}}},
		{"plain words", mkCell("aa bb")},
		{"code span", TableCell{Segments: []Segment{
			{Text: "x", Code: true},
			{Text: " tail"},
		}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			toks := cellTokens(tc.cell)
			painted := renderTokenLine(toks, Options{Theme: theme}, "")
			got := uniseg.StringWidth(stripTags(painted))
			if want := tokenLinePaintedWidth(toks); got != want {
				t.Errorf("painted %d cells, measured %d\ntokens: %v\npainted: %q",
					got, want, gotTexts(toks), painted)
			}
		})
	}
}

// TestWrapUnitsSplitsOversizeWord: when a glued word cannot fit a column at
// all, the units fall back to one fragment each so the greedy pass can split
// it where it must.
func TestWrapUnitsSplitsOversizeWord(t *testing.T) {
	t.Parallel()

	toks := []cellToken{{text: "ab"}, {text: "cd", glue: true}, {text: "ef", glue: true}}
	units := wrapUnits(toks, 4) // "abcdef" is 6 cells, wider than the column
	if len(units) != len(toks) {
		t.Fatalf("wrapUnits = %d units, want one per fragment (%d)", len(units), len(toks))
	}
	for i, u := range units {
		if len(u.toks) != 1 || u.toks[0].text != toks[i].text {
			t.Errorf("unit %d = %v, want fragment %q", i, gotTexts(u.toks), toks[i].text)
		}
	}
}

// TestWrapTokensHonorsForcedBreak: a token flagged pre starts a new line even
// when it would fit, which is how a "\n" in a source cell keeps its break.
func TestWrapTokensHonorsForcedBreak(t *testing.T) {
	t.Parallel()

	lines := wrapTokens([]cellToken{{text: "aa"}, {text: "bb", pre: true}}, 20)
	if len(lines) != 2 || len(lines[0]) != 1 || len(lines[1]) != 1 {
		t.Fatalf("wrapTokens(aa, pre-bb) = %v, want two lines", lines)
	}
}

// TestWrapTokensDisabled: width <= 0 disables wrapping entirely, which is the
// contract for callers that do their own folding.
func TestWrapTokensDisabled(t *testing.T) {
	t.Parallel()

	toks := []cellToken{{text: "aa"}, {text: "bb"}}
	lines := wrapTokens(toks, 0)
	if len(lines) != 1 || len(lines[0]) != len(toks) {
		t.Fatalf("wrapTokens(width 0) = %v, want one line with both tokens", lines)
	}
}

// TestSplitWideTokenGuards: a non-positive width has no legal chunk size, so
// the token is returned whole rather than split into empty pieces.
func TestSplitWideTokenGuards(t *testing.T) {
	t.Parallel()

	for _, width := range []int{0, -1} {
		got := splitWideToken("abcdef", width)
		if len(got) != 1 || got[0] != "abcdef" {
			t.Errorf("splitWideToken(width %d) = %v, want the whole token", width, got)
		}
	}
}

// TestPreSplitWideTokensFillsItsLines: every non-final hard-break chunk fills
// the column exactly, which is the invariant the line builder relies on when
// it lets a split token start at column zero.
func TestPreSplitWideTokensFillsItsLines(t *testing.T) {
	t.Parallel()

	const width = 10
	out := preSplitWideTokens([]cellToken{{text: strings.Repeat("x", 25)}}, width)
	if len(out) != 3 {
		t.Fatalf("chunks = %v, want three", gotTexts(out))
	}
	for i, tok := range out[:len(out)-1] {
		if got := uniseg.StringWidth(tok.text); got != width {
			t.Errorf("chunk %d is %d cells, want %d", i, got, width)
		}
	}
	if got := uniseg.StringWidth(out[len(out)-1].text); got == 0 || got > width {
		t.Errorf("final chunk is %d cells, want 1..%d", got, width)
	}
}

// TestAlignPad covers right and center alignment; the left default is already
// exercised by the table tests.
func TestAlignPad(t *testing.T) {
	t.Parallel()

	cases := []struct {
		align       TextAlign
		contentW    int
		colW        int
		left, right int
	}{
		{AlignRight, 3, 8, 5, 0},
		{AlignCenter, 3, 8, 2, 3},
		{AlignCenter, 4, 8, 2, 2},
		{AlignLeft, 3, 8, 0, 5},
		{AlignDefault, 3, 8, 0, 5},
		{AlignRight, 8, 8, 0, 0},
		{AlignRight, 9, 8, 0, 0},
	}
	for _, tc := range cases {
		left, right := alignPad(tc.align, tc.contentW, tc.colW)
		if left != tc.left || right != tc.right {
			t.Errorf("alignPad(%v, %d, %d) = (%d, %d), want (%d, %d)",
				tc.align, tc.contentW, tc.colW, left, right, tc.left, tc.right)
		}
	}
}

// TestRenderWrappedRowPadsEveryLine: a row is as tall as its tallest cell and
// every physical line is padded to the same painted width, so the column
// skeleton stays aligned on wrapped lines.
func TestRenderWrappedRowPadsEveryLine(t *testing.T) {
	t.Parallel()

	row := []TableCell{
		{Segments: []Segment{{Text: "aa bb cc"}}},
		mkCell("x"),
	}
	widths := []int{5, 3}
	lines := renderWrappedRow(row, widths, Options{}, "")
	if len(lines) != 2 {
		t.Fatalf("row painted %d lines, want 2: %q", len(lines), lines)
	}
	wantW := widths[0] + 1 + widths[1]
	for i, line := range lines {
		if got := uniseg.StringWidth(stripTags(line)); got != wantW {
			t.Errorf("line %d paints %d cells, want %d: %q", i, got, wantW, stripTags(line))
		}
	}
}
