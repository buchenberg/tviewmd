package tviewmd

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestRenderBoldTag(t *testing.T) {
	out := Render("**hi**", Options{})
	if !strings.Contains(out, "::b]") {
		t.Errorf("bold tag missing in %q", out)
	}
	if !strings.Contains(out, "hi") {
		t.Errorf("text missing in %q", out)
	}
}

func TestRenderItalicTag(t *testing.T) {
	out := Render("*hi*", Options{})
	if !strings.Contains(out, "::i]") {
		t.Errorf("italic tag missing in %q", out)
	}
}

func TestRenderStrikethroughTag(t *testing.T) {
	out := Render("~~hi~~", Options{})
	if !strings.Contains(out, "::s]") {
		t.Errorf("strike tag missing in %q", out)
	}
}

func TestRenderInlineCode(t *testing.T) {
	out := Render("`x`", Options{})
	// Inline code uses the InlineCodeBG color in the tag.
	if !strings.Contains(out, "]") {
		t.Errorf("plain code tag missing in %q", out)
	}
	if !strings.Contains(out, "x") {
		t.Errorf("code text missing in %q", out)
	}
}

func TestRenderHeadingColor(t *testing.T) {
	out := Render("## Sub", Options{})
	// H2 color from DefaultTheme is #ffae57.
	if !strings.Contains(out, "#ffae57") {
		t.Errorf("h2 color missing in %q", out)
	}
}

func TestRenderThematicBreakWidth(t *testing.T) {
	out := Render("a\n\n---\n\nb", Options{Width: 20})
	if !strings.Contains(out, strings.Repeat("─", 20)) {
		t.Errorf("20-char rule missing in %q", out)
	}
}

func TestRenderCodeBlockPlain(t *testing.T) {
	out := Render("```go\nfunc main() {}\n```", Options{})
	// Code blocks render with a single CodeBlockFG tag around escaped source.
	if !strings.Contains(out, "[#e6e6e6::]") {
		t.Errorf("code block fg tag missing in %q", out)
	}
	for _, tok := range []string{"func", "main"} {
		if !strings.Contains(out, tok) {
			t.Errorf("token %q missing in %q", tok, out)
		}
	}
}

func TestRenderListMarkers(t *testing.T) {
	out := Render("- a\n- b\n1. c", Options{})
	if !strings.Contains(out, "• a") {
		t.Errorf("bullet missing in %q", out)
	}
	if !strings.Contains(out, "1. c") {
		t.Errorf("ordered marker missing in %q", out)
	}
}

func TestRenderBlockquote(t *testing.T) {
	out := Render("> quoted", Options{})
	if !strings.Contains(out, "▎") {
		t.Errorf("quote bar missing in %q", out)
	}
	if !strings.Contains(out, "quoted") {
		t.Errorf("quote text missing in %q", out)
	}
}

func TestRenderLink(t *testing.T) {
	out := Render("[Charm](https://charm.sh)", Options{})
	if !strings.Contains(out, "Charm") {
		t.Errorf("link label missing in %q", out)
	}
	if !strings.Contains(out, "https://charm.sh") {
		t.Errorf("link url missing in %q", out)
	}
}

func TestRenderTableAlignment(t *testing.T) {
	src := "| A | B |\n|:--|--:|\n| short | x |\n"
	out := Render(src, Options{})
	// Right-aligned column B should left-pad "x" to match header width "B".
	if !strings.Contains(out, "x") {
		t.Errorf("cell missing in %q", out)
	}
	if !strings.Contains(out, "─") {
		t.Errorf("separator missing in %q", out)
	}
}

func TestRenderRoundTripPreservesText(t *testing.T) {
	src := "# Title\n\nSome paragraph with **bold** and a `code span`.\n\n- one\n- two\n"
	out := Render(src, Options{})
	for _, want := range []string{"Title", "Some paragraph with", "bold", "code span", "one", "two"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in rendered output:\n%s", want, out)
		}
	}
}

func TestRenderEmpty(t *testing.T) {
	if got := Render("", Options{}); got != "" {
		t.Errorf("expected empty render, got %q", got)
	}
}

func TestRenderPlainFallback(t *testing.T) {
	// A string with no markdown structure should round-trip as plain text.
	if got := Render("just text here", Options{}); got != "just text here" {
		t.Errorf("plain fallback=%q", got)
	}
}

func TestRenderTViewTrimsTrailingNewline(t *testing.T) {
	out := Render("para", Options{})
	if strings.HasSuffix(out, "\n") {
		t.Errorf("trailing newline not trimmed: %q", out)
	}
}

// cellStyleFor renders tagged text in a real tview.TextView onto a tcell
// SimulationScreen and returns the style of the first cell whose bytes begin
// with r. It is the gold-standard way to assert how tview actually renders the
// library's color-tagged output, including attributes tcell stores outside the
// AttrMask (notably the separate underline-style state).
func cellStyleFor(t *testing.T, tagged string, r rune) (tcell.Style, bool) {
	t.Helper()
	tv := tview.NewTextView().SetDynamicColors(true).SetWrap(true)
	tv.SetText(tagged)
	tv.SetRect(0, 0, 40, 2)
	screen := tcell.NewSimulationScreen("")
	if err := screen.Init(); err != nil {
		t.Fatalf("sim screen init: %v", err)
	}
	t.Cleanup(screen.Fini)
	screen.SetSize(40, 2)
	tv.Draw(screen)
	screen.Show() // flush back buffer to front so GetContents reflects the draw
	cells, _, _ := screen.GetContents()
	for _, c := range cells {
		if len(c.Bytes) > 0 && rune(c.Bytes[0]) == r {
			return c.Style, true
		}
	}
	return tcell.StyleDefault, false
}

// TestRenderLinkDoesNotLeakUnderline guards against the bug where an underlined
// run (e.g. a link) left its underline active for all following content.
//
// tcell keeps the underline style in a separate Style.ulStyle field that
// tview's [-:-:-] reset does not clear, so checking the AttrMask is not enough.
// tcell.Style is a plain struct, so == compares every field (including the
// unexported ulStyle) — if the post-link tail matches the plain baseline, the
// renderer is closing its tags correctly. This needs no unsafe access.
//
// The tail uses 'Q' as a sentinel because the visible text of links/autolinks
// (URLs, labels) never contains it, so cellStyleFor can't accidentally match a
// cell that is legitimately still styled.
func TestRenderLinkDoesNotLeakUnderline(t *testing.T) {
	t.Parallel()

	const tail = "QQQ"

	baseline, ok := cellStyleFor(t, "plain "+tail, 'Q')
	if !ok {
		t.Fatal("could not find baseline sentinel cell")
	}

	out := Render("[Charm](https://charm.sh) then plain "+tail, Options{})
	got, ok := cellStyleFor(t, out, 'Q')
	if !ok {
		t.Fatalf("could not find trailing sentinel cell in rendered output:\n%s", out)
	}

	if got != baseline {
		t.Errorf("style leaked past a link into trailing text\ntail style after link: %#v\nbaseline tail style:    %#v\noutput: %q", got, baseline, out)
	}
}

// TestRenderClosesUnderlineForAllStyles ensures the close tag resets underline
// regardless of which other attributes were active on the segment.
func TestRenderClosesUnderlineForAllStyles(t *testing.T) {
	t.Parallel()

	const tail = "QQQ"
	baseline, ok := cellStyleFor(t, "plain "+tail, 'Q')
	if !ok {
		t.Fatal("missing baseline sentinel")
	}

	cases := []struct {
		name string
		src  string
	}{
		{"link", "[l](u) " + tail},
		{"autolink", "<https://x.io> " + tail},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out := Render(tc.src, Options{})
			got, ok := cellStyleFor(t, out, 'Q')
			if !ok {
				t.Fatalf("missing trailing sentinel in:\n%s", out)
			}
			if got != baseline {
				t.Errorf("style leaked:\n got=%#v\nwant=%#v", got, baseline)
			}
		})
	}
}

// screenText renders tagged text in a real tview.TextView and returns the
// visible (tag-stripped) cell text of the first row.
func screenText(t *testing.T, tagged string) string {
	t.Helper()
	tv := tview.NewTextView().SetDynamicColors(true).SetWrap(true)
	tv.SetText(tagged)
	tv.SetRect(0, 0, 80, 4)
	screen := tcell.NewSimulationScreen("")
	if err := screen.Init(); err != nil {
		t.Fatalf("sim screen init: %v", err)
	}
	t.Cleanup(screen.Fini)
	screen.SetSize(80, 4)
	tv.Draw(screen)
	screen.Show()
	cells, _, _ := screen.GetContents()
	var b strings.Builder
	for _, c := range cells {
		if len(c.Bytes) > 0 {
			b.WriteString(string(c.Bytes))
		}
	}
	return b.String()
}

// TestRenderEscapesBrackets guards against bracket injection: literal bracket
// sequences in the markdown source that look like tview color/region tags must
// be rendered as visible text, not consumed by SetDynamicColors. goldmark may
// split such a span across several segments, so this is checked end-to-end.
func TestRenderEscapesBrackets(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		src     string
		visible string // the bracket-bearing text that must appear on screen
	}{
		{"color word", "[red]boom", "[red]boom"},
		{"green mid sentence", "see [green] now", "see [green] now"},
		{"numeric", "price is [5] dollars", "price is [5] dollars"},
		{"hex", "[#ff0000]x", "[#ff0000]x"},
		{"region tag", `["quoted"]`, `["quoted"]`},
		{"unclosed", "a [boom", "a [boom"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out := Render(tc.src, Options{})
			if got := screenText(t, out); !strings.Contains(got, tc.visible) {
				t.Errorf("visible text lost the bracket span from %q:\n on-screen: %q\n tagged:   %q", tc.src, got, out)
			}
		})
	}
}

// TestRenderBracketInjectionDoesNotRecolor checks end-to-end that a literal
// "[red]" in the source does not actually turn following text red in a TextView.
func TestRenderBracketInjectionDoesNotRecolor(t *testing.T) {
	t.Parallel()

	baseline, ok := cellStyleFor(t, "plain boom", 'b')
	if !ok {
		t.Fatal("missing baseline")
	}
	got, ok := cellStyleFor(t, Render("[red]boom", Options{}), 'b')
	if !ok {
		t.Fatal("missing 'b' cell after escaping")
	}
	if got != baseline {
		t.Errorf("literal [red] recolored text:\n got=%#v\nwant=%#v", got, baseline)
	}
}

// TestRenderTableCellInlineCodeThemed ensures table cells receive the render
// options' theme (previously renderTableRow discarded opts, so code spans and
// links inside tables rendered without the theme colors).
func TestRenderTableCellInlineCodeThemed(t *testing.T) {
	t.Parallel()

	theme := DefaultTheme()
	src := "| A | B |\n|---|---|\n| `x` | y |\n"
	out := Render(src, Options{Theme: theme})

	if !strings.Contains(out, theme.InlineCodeBG) {
		t.Errorf("table cell inline code missing InlineCodeBG %q in output:\n%s", theme.InlineCodeBG, out)
	}
	if !strings.Contains(out, theme.InlineCodeFG) {
		t.Errorf("table cell inline code missing InlineCodeFG %q in output:\n%s", theme.InlineCodeFG, out)
	}
}

// TestRenderTableCellLinkThemed ensures links inside table cells use the
// caller's theme rather than the default.
func TestRenderTableCellLinkThemed(t *testing.T) {
	t.Parallel()

	theme := DefaultTheme()
	theme.Link = "#123456" // deliberately non-default
	src := "| A |\n|---|\n| [l](https://x.io) |\n"
	out := Render(src, Options{Theme: theme})

	if !strings.Contains(out, "#123456") {
		t.Errorf("table cell link missing custom theme color:\n%s", out)
	}
}


