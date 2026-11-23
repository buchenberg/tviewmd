package tviewmd

import (
	"strings"
	"testing"
)

const spacingTable = "| A | B |\n|---|---|\n| one | x |\n| two | y |\n"

func tableLines(t *testing.T, src string, opts Options) []string {
	t.Helper()
	out := Render(src, opts)
	return strings.Split(stripTags(out), "\n")
}

func lineIndex(lines []string, contains string) int {
	for i, line := range lines {
		if strings.Contains(line, contains) {
			return i
		}
	}
	return -1
}

// TestTableRowSpacingDefault: the zero value of Options paints one blank line
// between body rows so a dense table still shows where a row breaks.
func TestTableRowSpacingDefault(t *testing.T) {
	lines := tableLines(t, spacingTable, Options{})
	one := lineIndex(lines, "one")
	two := lineIndex(lines, "two")
	if one < 0 || two < 0 {
		t.Fatalf("body rows missing in %q", lines)
	}
	if want := one + 2; two != want {
		t.Errorf("rows are %d lines apart, want %d (one blank line): %q", two-one, want-one+1, lines)
	}
	if lines[one+1] != "" {
		t.Errorf("line between rows = %q, want blank: %q", lines[one+1], lines)
	}
	if got := lines[len(lines)-1]; got == "" {
		t.Errorf("render ends with a blank line: %q", lines)
	}
}

// TestTableRowSpacingCompact: CompactTableRows restores the dense pre-spacing
// layout with no blank lines between body rows.
func TestTableRowSpacingCompact(t *testing.T) {
	lines := tableLines(t, spacingTable, Options{TableRowSpacing: CompactTableRows})
	one := lineIndex(lines, "one")
	two := lineIndex(lines, "two")
	if one < 0 || two < 0 {
		t.Fatalf("body rows missing in %q", lines)
	}
	if two != one+1 {
		t.Errorf("compact rows are %d lines apart, want 1: %q", two-one, lines)
	}
}

// TestTableRowSpacingCustom: positive values add that many blank lines.
func TestTableRowSpacingCustom(t *testing.T) {
	lines := tableLines(t, spacingTable, Options{TableRowSpacing: 2})
	one := lineIndex(lines, "one")
	two := lineIndex(lines, "two")
	if one < 0 || two < 0 {
		t.Fatalf("body rows missing in %q", lines)
	}
	if two != one+3 {
		t.Errorf("rows are %d lines apart, want 3 (two blank lines): %q", two-one, lines)
	}
	if lines[one+1] != "" || lines[one+2] != "" {
		t.Errorf("want two blank lines between rows, got %q: %q", lines[one+1:one+3], lines)
	}
}

// TestTableRowSpacingWrapped: the width-constrained layout also separates
// logical rows, not just physical lines within one wrapped row.
func TestTableRowSpacingWrapped(t *testing.T) {
	src := "" +
		"| Feature | Status |\n" +
		"|---|---|\n" +
		"| a very long description that has to wrap across lines | blocked on width budget allocation |\n" +
		"| short | landed |\n"
	lines := tableLines(t, src, Options{Width: 34})
	first := lineIndex(lines, "a very long")
	second := lineIndex(lines, "short")
	if first < 0 || second < 0 {
		t.Fatalf("wrapped body rows missing in %q", lines)
	}
	if second <= first+1 {
		t.Fatalf("first row did not wrap before the second: %q", lines)
	}
	if got := lines[second-1]; got != "" {
		t.Errorf("line before second logical row = %q, want blank: %q", got, lines)
	}
}
