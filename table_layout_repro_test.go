package tviewmd

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/rivo/uniseg"
)

// These two tests encode the DESIRED table-layout behavior and currently FAIL:
// they are the regression repro for "small tables render fine, wide tables
// wrap mid-row and misalign". They must pass after the table layout rework.

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

// TestTableWideContentStaysOnOneLinePerRow: when Options.Width is set, the
// renderer must lay the table out to fit that width. A logical table row
// (header, separator, or body) must never be wider than the width budget, so
// tview's word wrap never breaks a row in half. Today column widths are
// natural (max content) widths and Options.Width is ignored: every row of this
// table is 83 cells wide, the TextView wraps each one as prose, and the
// painted screen shows ~8 ragged lines instead of 4 aligned rows.
func TestTableWideContentStaysOnOneLinePerRow(t *testing.T) {
	t.Parallel()

	src := "" +
		"| Feature description summary | Implementation status details |\n" +
		"|------------------------------|-------------------------------|\n" +
		"| Streaming incremental parse with word wrap policy | Blocked on width budget allocation |\n" +
		"| Cluster aware width measurement | Landed in the table layout rework |\n"
	out := Render(src, Options{Width: 60})
	rows := paintedScreenRows(t, out, 60, 24)

	const wantRows = 4 // header, separator, two body rows — one line each
	if len(rows) != wantRows {
		t.Errorf("painted %d screen lines, want %d (one per logical row); wrapped output:\n%s",
			len(rows), wantRows, strings.Join(rows, "\n"))
	}
	if len(rows) == wantRows {
		// Columns must line up: the separator must span exactly the header row.
		if w0, w1 := rowLen(rows[0]), rowLen(rows[1]); w0 != w1 {
			t.Errorf("header row spans %d cells but separator spans %d:\n%s\n%s", w0, w1, rows[0], rows[1])
		}
	}
}

// rowLen counts display cells, cluster-aware, of one painted screen line.
func rowLen(row string) int {
	return uniseg.StringWidth(row)
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
