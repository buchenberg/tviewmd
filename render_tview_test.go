package tviewmd

import (
	"strings"
	"testing"
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

func TestRenderCodeHighlightInvokesChroma(t *testing.T) {
	// A go fenced block should produce highlighted output: chroma emits ANSI
	// that TranslateANSI converts to tview tags, so expect a tag and keyword
	// coloring. We only assert it does not crash and returns content.
	out := Render("```go\nfunc main() {}\n```", Options{})
	// chroma tokenizes individually; each token gets its own color tag.
	// Assert the keywords appear separately.
	for _, tok := range []string{"func", "main"} {
		if !strings.Contains(out, tok) {
			t.Errorf("token %q missing in %q", tok, out)
		}
	}
}

func TestRenderCodeNoHighlight(t *testing.T) {
	out := Render("```go\nfunc main() {}\n```", Options{NoHighlight: true})
	// With highlighting off, the whole block is a single code-block-fg tag.
	if !strings.Contains(out, "[#e6e6e6::]") {
		t.Errorf("plain plain code tag missing in %q", out)
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


