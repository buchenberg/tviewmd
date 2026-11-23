package tviewmd

import (
	"strings"
	"testing"
)

func TestSanitizeText(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{"clean", "hello world", "hello world"},
		{"unicode", "héllo ✍️", "héllo ✍️"},
		{"newlines and tabs", "a\nb\tc", "a\nb\tc"},
		{"simple CSI", "a\x1b[31mb\x1b[0mc", "abc"},
		{"other CSI", "a\x1b[2Jb", "ab"},
		{"OSC with BEL", "a\x1b]0;title\x07b", "ab"},
		{"OSC with ST", "a\x1b]0;title\x1b\\b", "ab"},
		{"bare ESC", "a\x1bb", "ab"},
		{"NUL and DEL", "a\x00b\x7fc", "abc"},
		{"C1 control", "a\u009bb", "ab"},
		{"raw 8-bit C1", "a\x9bb", "ab"},
		{"mixed", "keep \x1b[1mmore\x00", "keep more"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := sanitizeText(tc.in); got != tc.want {
				t.Errorf("sanitizeText(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestRenderSanitizesUntrustedText: markdown can carry ANSI from an LLM or a
// pasted log; it must not reach the terminal through the TextView.
func TestRenderSanitizesUntrustedText(t *testing.T) {
	src := "hello \x1b[31mred\x1b[0m world\n\n```\n\x1b[2J\n```\n"
	out := Render(src, Options{})
	for _, forbidden := range []string{"\x1b", "31m", "0m", "2J"} {
		if strings.Contains(out, forbidden) {
			t.Errorf("render leaked %q:\n%s", forbidden, out)
		}
	}
	if !strings.Contains(out, "hello") || !strings.Contains(out, "world") {
		t.Errorf("sanitizing dropped visible text:\n%s", out)
	}
}

func TestParseSanitizesSegmentText(t *testing.T) {
	blocks, err := Parse("a\x1b[31mb")
	if err != nil {
		t.Fatal(err)
	}
	if len(blocks) != 1 || len(blocks[0].Segments) != 1 {
		t.Fatalf("unexpected blocks: %+v", blocks)
	}
	if got := blocks[0].Segments[0].Text; got != "ab" {
		t.Errorf("segment text = %q, want %q", got, "ab")
	}
}

// TestRenderTViewSanitizesHandBuiltBlocks covers callers that construct Blocks
// directly instead of going through Parse.
func TestRenderTViewSanitizesHandBuiltBlocks(t *testing.T) {
	blocks := []Block{{Kind: BlockParagraph, Segments: []Segment{{Text: "a\x1b[31mb"}}}}
	out := RenderTView(blocks, Options{})
	if strings.Contains(out, "\x1b") || strings.Contains(out, "31m") {
		t.Errorf("hand-built block leaked ANSI: %q", out)
	}
}

func TestSanitizeIncompleteSequences(t *testing.T) {
	cases := []struct{ in, want string }{
		{"a\x1b[31", "a"},
		{"a\x1b]0;title", "a"},
		{"a\x1b", "a"},
	}
	for _, tc := range cases {
		if got := sanitizeText(tc.in); got != tc.want {
			t.Errorf("sanitizeText(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
