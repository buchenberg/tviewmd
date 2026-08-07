package tviewmd

import "testing"

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
	}
	for _, s := range seed {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, src string) {
		blocks, err := Parse(src)
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		// Rendering must also be panic-free.
		_ = RenderTView(blocks, Options{})
	})
}
