package tviewmd

import (
	"strings"
	"testing"
)

func TestNegativeWidthIsDocumentedBehavior(t *testing.T) {
	out := Render("a\n\n---\n\n| a | b |\n|---|---|\n| 1 | 2 |", Options{Width: -5})
	if !strings.Contains(out, strings.Repeat("─", 80)) {
		t.Errorf("negative Width did not fall back to an 80-cell rule: %q", out)
	}
	if !strings.Contains(out, "1 2") {
		t.Errorf("table lost its content: %q", out)
	}
}

func TestHandBuiltBlocksDoNotPanic(t *testing.T) {
	blocks := []Block{
		{Kind: BlockKind(99)},
		{Kind: BlockHeading, Level: 0, Segments: []Segment{{Text: "level zero"}}},
		{Kind: BlockHeading, Level: 7, Segments: []Segment{{Text: "level seven"}}},
		{Kind: BlockParagraph},
		{Kind: BlockTable},
	}
	out := RenderTView(blocks, Options{})
	if !strings.Contains(out, "level zero") || !strings.Contains(out, "level seven") {
		t.Errorf("hand-built headings lost: %q", out)
	}
}

func TestLinkURLBracketsAreEscaped(t *testing.T) {
	// goldmark ends the URL at the first ']'; any leftover bracket text must
	// not become an injected tview tag. The visible label and decoration stay.
	raw := Render("[label](https://x/[y])", Options{})
	out := stripTags(raw)
	if !strings.Contains(out, "label (https://x/)") {
		t.Errorf("link decoration lost: %q", out)
	}
	if !tagsBalanced(Render("[label](https://x/[y])", Options{})) {
		t.Errorf("bracketed link text injected a tag")
	}
}

func TestRawHTMLIsLiteralText(t *testing.T) {
	out := stripTags(Render("<b>bold</b>\n\n<span>inline</span>", Options{}))
	if !strings.Contains(out, "<b>bold</b>") || !strings.Contains(out, "<span>inline</span>") {
		t.Errorf("raw HTML was not emitted as literal text: %q", out)
	}
}

func TestImageEmitsAltTextOnly(t *testing.T) {
	out := stripTags(Render("![alt text](https://x/image.png)", Options{}))
	if !strings.Contains(out, "alt text") {
		t.Errorf("image alt text missing: %q", out)
	}
	if strings.Contains(out, "image.png") {
		t.Errorf("image URL leaked into text (documented lossy behavior changed): %q", out)
	}
}

func TestLineBreaks(t *testing.T) {
	if got := stripTags(Render("a\nb", Options{})); got != "a b" {
		t.Errorf("soft break = %q, want %q", got, "a b")
	}
	if got := stripTags(Render("a  \nb", Options{})); got != "a\nb" {
		t.Errorf("hard break = %q, want %q", got, "a\nb")
	}
}

// Emphasis used to be parsed but discarded whenever a link or code hint was
// present on the segment.
func TestEmphasisSurvivesLinksAndCode(t *testing.T) {
	link := Render("**[bold link](https://x)**", Options{})
	if !strings.Contains(link, "b]") || !strings.Contains(link, "u") {
		t.Errorf("bold link lost its flags: %q", link)
	}
	code := Render("**`coded`**", Options{})
	if !strings.Contains(code, "b]") {
		t.Errorf("bold code span lost its flag: %q", code)
	}
}

func TestAdjacentSameURLLinksStayDistinct(t *testing.T) {
	out := stripTags(Render("[a](u)[b](u)", Options{}))
	if got := strings.Count(out, "(u)"); got != 2 {
		t.Errorf("got %d link decorations, want 2: %q", got, out)
	}
}

func TestEmptyFenceEmitsNoTags(t *testing.T) {
	if got := Render("```", Options{}); got != "" {
		t.Errorf("empty fence rendered %q, want empty", got)
	}
}

func TestPlainText(t *testing.T) {
	segs := []Segment{{Text: "one"}, {Text: "two"}}
	if got := PlainText(segs); got != "onetwo" {
		t.Errorf("PlainText = %q, want %q", got, "onetwo")
	}
}

func TestTagsBalancedAcrossFeatures(t *testing.T) {
	src := "# H\n\n**b** [l](u) `c`\n\n| a |\n|---|\n| \x1b[31mx\x1b[0m |\n"
	if out := Render(src, Options{}); !tagsBalanced(out) {
		t.Errorf("unbalanced tags: %q", out)
	}
}

func TestRenderLinesIncludesTableSpacers(t *testing.T) {
	src := "| A |\n|---|\n| x |\n| y |\n"
	lines := RenderLines(src, Options{})
	// header, separator, two body rows, one blank spacer = 5
	if len(lines) != 5 {
		t.Errorf("RenderLines returned %d lines, want 5: %q", len(lines), lines)
	}
	if lines[3] != "" {
		t.Errorf("expected spacer at line 3: %q", lines)
	}
	if got := RenderLines("", Options{}); got != nil {
		t.Errorf("empty source gave lines %q, want nil", got)
	}
	if got, want := RenderLines(src, Options{TableRowSpacing: CompactTableRows}), RenderLines(src, Options{}); len(got) != len(want)-1 {
		t.Errorf("compact layout = %d lines, want %d", len(got), len(want)-1)
	}
}
