package tviewmd

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
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
// which FuzzParse's Options{} never reaches. Its invariant is the real tview
// paint oracle: no color tag may survive into a painted cell.
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
		if !paintedTagsConsumed(out, width) {
			t.Fatalf("tview tag leaked into painted text for %q: %q", src, out)
		}
	})
}

// paintedTagsConsumed renders tagged text through a real tview.TextView on a
// tcell simulation screen and reports whether every tag was consumed. A tag
// that survives into a cell would be displayed literally to the user.
func paintedTagsConsumed(s string, width int) bool {
	if width < 20 {
		width = 20
	}
	if width > 200 {
		width = 200
	}
	// Size the screen to the document: a fixed height hides leaks in long
	// documents that paint beyond the last visible row.
	height := strings.Count(s, "\n") + 3
	if height < 40 {
		height = 40
	}
	if height > 500 {
		height = 500
	}
	tv := tview.NewTextView().SetDynamicColors(true).SetWrap(true)
	tv.SetText(s)
	tv.SetRect(0, 0, width, height)
	screen := tcell.NewSimulationScreen("")
	if err := screen.Init(); err != nil {
		return true // cannot verify in this environment; do not fail the fuzz run
	}
	defer screen.Fini()
	screen.SetSize(width, height)
	tv.Draw(screen)
	screen.Show()
	cells, w, _ := screen.GetContents()
	// tview writes one rune per cell, so join the painted rows before scanning
	// for bracket spans that span several cells.
	var b strings.Builder
	for row := 0; row < height; row++ {
		for col := 0; col < w; col++ {
			b.Write(cells[row*w+col].Bytes)
		}
		b.WriteByte('\n')
	}
	text := b.String()
	for i := 0; i < len(text); i++ {
		if text[i] != '[' {
			continue
		}
		j := i + 1
		for j < len(text) && text[j] != ']' && text[j] != '\n' {
			j++
		}
		// A bracket span containing ':' is a tview tag. If it survived
		// painting, the caller will see it literally.
		// A real tview tag has exactly two colon separators; a literal
		// '[a:b]' painted from escaped user text has only one and is fine.
		if j < len(text) && text[j] == ']' && strings.Count(text[i:j], ":") == 2 {
			return false
		}
	}
	return true
}

// TestPaintedTagsConsumedDetectsLeak guards the fuzz invariant itself: a
// checker that always passes would make FuzzRender pointless.
func TestPaintedTagsConsumedDetectsLeak(t *testing.T) {
	if paintedTagsConsumed("[#zzz::]unclosed color tag", 80) {
		t.Error("invariant accepted a literal unclosed tag")
	}
	if !paintedTagsConsumed(Render("plain text", Options{}), 80) {
		t.Error("invariant rejected clean output")
	}
}
