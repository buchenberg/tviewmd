package tviewmd

import (
	"strings"
	"testing"
)

// FuzzParse ensures the goldmark walker never panics on arbitrary input. A
// panic here would crash the TUI on any model output, so this is the most
// important correctness guard for the renderer.
func FuzzParse(f *testing.F) {
	seed := []string{
		"",
		"# heading",
		"```go\nfunc(){}\n```",
		"| a | b |\n|---|---|\n| 1 | 2 |",
		"- [x] done\n- [ ] todo",
		"> quote\n> more",
		"***weird*** ***nesting** *italic*",
		"line1\nline2\n\npara2",
		"[l](u) <auto> `c` ~~s~~",
		"----",
		"#{1,2,3}",
		"\x00\x01binary\x02",
		"\x1b[31mred\x1b[0m",
		"**✨ wide** and 中文",
	}
	for _, s := range seed {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, src string) {
		blocks, _ := Parse(src) // Parse is permissive; today err is always nil.
		_ = RenderTView(blocks, Options{})
	})
}

// FuzzRender exercises the width-constrained table path and alternate themes,
// which FuzzParse's Options{} never reaches. It also checks that emitted tview
// tags are balanced: an unbalanced tag would paint literally in a TextView.
func FuzzRender(f *testing.F) {
	f.Add("# head\n\n[a](u)\n\n| a | b |\n|---|---|\n| 1 | 2 |\n", 0, 0)
	f.Add("> - a\n> ```go\n> x\n> ```\n", 34, 1)
	f.Add("`c` ~~s~~ ✍️ 中文", -1, 2)
	f.Fuzz(func(t *testing.T, src string, width, theme int) {
		opts := Options{Width: width, TableRowSpacing: theme - 1}
		switch theme {
		case 1:
			opts.Theme = NoColorTheme()
		case 2:
			opts.Theme = Theme{Link: "#123456"}
		}
		out := Render(src, opts)
		if !tagsBalanced(out) {
			t.Fatalf("unbalanced tview tags in %q", out)
		}
	})
}

// tagsBalanced reports whether every tag-shaped bracket span in out is closed.
// `[[` is an escaped literal bracket; a lone '[' with no following ']' is also
// just literal text in tview. Only '[fg:bg:flags]' shapes are tags.
func tagsBalanced(s string) bool {
	for i := 0; i < len(s); {
		if s[i] != '[' {
			i++
			continue
		}
		if i+1 < len(s) && s[i+1] == '[' {
			i += 2
			continue
		}
		j := i + 1
		for j < len(s) && s[j] != ']' {
			j++
		}
		if j == len(s) {
			break // literal '[' with no matching bracket
		}
		if !strings.Contains(s[i:j], ":") {
			i = j + 1 // literal bracket span, not a tag
			continue
		}
		i = j + 1
	}
	return true
}
