package tviewmd

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/rivo/uniseg"
)

// These two tests pin the table-layout behavior that the rework delivered:
// wide tables lay out to Options.Width without folding, and painted width
// accounting covers the visible link decoration. They were the failing
// repros for "small tables render fine, wide tables wrap mid-row and
// misalign" and must keep passing.

// paintedScreenRows draws tagged text into a simulation screen of width w with
// wrapping enabled and returns the painted, right-trimmed screen lines with
// trailing blank lines removed.
func paintedScreenRows(t *testing.T, tagged string, w, h int) []string {
	t.Helper()
	tv := tview.NewTextView().SetDynamicColors(true).SetWrap(true)
	tv.SetText(tagged)
	tv.SetRect(0, 0, w, h)
	screen := tcell.NewSimulationScreen("")
	if err := screen.Init(); err != nil {
		t.Fatalf("sim screen init: %v", err)
	}
	t.Cleanup(screen.Fini)
	screen.SetSize(w, h)
	tv.Draw(screen)
	screen.Show()
	cells, sw, _ := screen.GetContents()
	rows := make([]string, h)
	for row := 0; row < h; row++ {
		var b strings.Builder
		for col := 0; col < sw; col++ {
			c := cells[row*sw+col]
			if len(c.Bytes) == 0 {
				b.WriteByte(' ')
				continue
			}
			b.Write(c.Bytes)
		}
		rows[row] = strings.TrimRight(b.String(), " ")
	}
	for len(rows) > 0 && rows[len(rows)-1] == "" {
		rows = rows[:len(rows)-1]
	}
	return rows
}

// TestTableWideContentLaysOutToWidth: when Options.Width is set, the renderer
// must lay the table out to fit that width. The invariant is not "one painted
// line per logical row" — a 49-cell body cell cannot fit any legal plan for a
// 60-cell budget, and a wrapped grid legitimately paints several lines per
// row. What must hold instead: (1) no painted line exceeds the budget, so
// tview never re-wraps a table line as prose (the folding that shreds
// alignment); (2) no word is split mid-word — line breaks land between words;
// (3) the column boundary the separator advertises is a blank cell on every
// painted row — no cell spills into its right-hand neighbor. Before the
// rework, column widths were natural max-content widths and Options.Width was
// ignored: every row of this table painted 83 cells and the TextView wrapped
// each as prose into ragged halves.
func TestTableWideContentLaysOutToWidth(t *testing.T) {
	t.Parallel()

	src := "" +
		"| Feature description summary | Implementation status details |\n" +
		"|------------------------------|-------------------------------|\n" +
		"| Streaming incremental parse with word wrap policy | Blocked on width budget allocation |\n" +
		"| Cluster aware width measurement | Landed in the table layout rework |\n"
	out := Render(src, Options{Width: 60})
	// (1) No line wider than the budget. Measure the RAW render lines, not
	// the painted screen: a SimulationScreen is 60 cells wide by
	// construction, so screen rows can never exceed the width even when
	// the render itself emits 80-cell lines that tview then folds.
	for i, line := range strings.Split(out, "\n") {
		if w := uniseg.StringWidth(stripTags(line)); w > 60 {
			t.Errorf("render line %d paints %d cells, want <= 60: %q", i, w, stripTags(line))
		}
	}
	rows := paintedScreenRows(t, out, 60, 24)

	if len(rows) < 4 {
		t.Fatalf("painted %d screen lines, want at least header+separator+2 body; wrapped output:\n%s",
			len(rows), strings.Join(rows, "\n"))
	}
	// (2) Words survive intact: check the PAINTED rows, so a break inside a
	// word (which also splits it across lines) is visible -- the raw tagged
	// string still contains every word contiguously even when it folds.
	painted := strings.Join(rows, "\n")
	for _, word := range []string{"Feature", "description", "summary", "Implementation", "status", "details",
		"Streaming", "incremental", "policy", "Cluster", "measurement", "Landed", "rework"} {
		if !strings.Contains(painted, word) {
			t.Errorf("word %q split mid-word or missing from layout:\n%s", word, painted)
		}
	}
	// (3) The separator row is all box-drawing dashes with exactly one space
	// at the column boundary; every other painted row must be blank at that
	// screen cell, i.e. column 2 starts exactly where the separator says.
	// Locate the separator by content -- a wrapping header may occupy more
	// than one painted line, so a fixed index would be brittle.
	sepIdx := -1
	for i, r := range rows {
		if strings.Contains(r, "─") && strings.Trim(strings.Map(dropDashSpace, r), " ") == "" {
			sepIdx = i
			break
		}
	}
	if sepIdx < 0 {
		t.Fatalf("no separator row found in painted output:\n%s", painted)
	}
	sep := rows[sepIdx]
	byteIdx := strings.IndexByte(sep, ' ')
	if byteIdx < 0 {
		t.Fatalf("separator row has no column boundary: %q", sep)
	}
	boundary := uniseg.StringWidth(sep[:byteIdx]) // byte index -> screen cell
	for i, r := range rows {
		if i == sepIdx {
			continue
		}
		if !rowBlankAtCell(r, boundary) {
			t.Errorf("row %d paints non-blank at column-boundary cell %d — a cell spills into its neighbor:\n%q", i, boundary, r)
		}
	}
}

// dropDashSpace maps box-drawing dashes and spaces to spaces, so trimming the
// result identifies a separator row.
func dropDashSpace(r rune) rune {
	if r == '─' {
		return ' '
	}
	return r
}

// stripTags returns the visible text of one tagged render line: an escaped
// "[[" paints one '[' and counts as one cell, and bracketed colour tags paint
// nothing.
func stripTags(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] != '[' {
			b.WriteByte(s[i])
			continue
		}
		if i+1 < len(s) && s[i+1] == '[' {
			b.WriteByte('[')
			i++
			continue
		}
		end := strings.IndexByte(s[i:], ']')
		if end < 0 {
			b.WriteByte(s[i])
			continue
		}
		i += end
	}
	return b.String()
}

// rowBlankAtCell reports whether painted row is blank (space, or the row
// ended) at screen cell index cell. Walks grapheme clusters so wide glyphs
// occupy their true cell count.
func rowBlankAtCell(row string, cell int) bool {
	w := 0
	state := -1
	rest := row
	for len(rest) > 0 {
		var cluster string
		var cw int
		cluster, rest, cw, state = uniseg.FirstGraphemeClusterInString(rest, state)
		if w == cell {
			return cluster == " "
		}
		if w+cw > cell {
			return false // a wide glyph covers the boundary cell
		}
		w += cw
	}
	return true // row ended at or before the boundary cell
}

// TestTableLinkDecorationKeepsRowAligned: cells render a visible " (url)"
// decoration when a link's label differs from its target, but cellPlainWidth
// measures only the segment text, so the column is planned one size and
// painted another. Even with no wrapping at all, the body row comes out wider
// than the header/separator and every later column shifts. Width accounting
// must cover exactly the glyphs that get painted.
func TestTableLinkDecorationKeepsRowAligned(t *testing.T) {
	t.Parallel()

	src := "" +
		"| Link | Note |\n" +
		"|------|------|\n" +
		"| [docs](https://example.com/some/very/long/path) | text |\n"
	out := Render(src, Options{})
	ws := screenRowWidths(t, out, 120, 8) // wrap disabled: pure width comparison

	// Row 0 = header, row 1 = separator, row 2 = body. The final column is
	// deliberately equal-width in every row ("Note"/"text"), so the invariant
	// "all rows paint to the same extent" is expressible: screenRowWidths
	// measures up to the last non-space cell and cannot see trailing padding.
	if ws[0] == 0 || ws[2] == 0 {
		t.Fatalf("screen rows missing: widths = %v", ws)
	}
	if ws[0] != ws[2] || ws[1] != ws[0] {
		t.Errorf("header row paints %d cells, separator %d, body %d — the (url) decoration is not measured:\n%s",
			ws[0], ws[1], ws[2], out)
	}
}
