package main

import (
	"strings"
	"testing"
)

// TestParseArgs covers the argument contract: a leading integer is the paint
// width, a leading non-integer is the sample-name filter (previously dropped
// together with the discarded Atoi error), and a second argument always
// overrides the filter.
func TestParseArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       []string
		wantWidth  int
		wantFilter string
	}{
		{name: "no arguments", args: nil, wantWidth: 100},
		{name: "width only", args: []string{"120"}, wantWidth: 120},
		{name: "filter only", args: []string{"docs"}, wantWidth: 100, wantFilter: "docs"},
		{name: "width and filter", args: []string{"120", "docs"}, wantWidth: 120, wantFilter: "docs"},
		{name: "filter then extra argument", args: []string{"docs", "120"}, wantWidth: 100, wantFilter: "docs"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gotWidth, gotFilter, err := parseArgs(tc.args)
			if err != nil {
				t.Fatalf("parseArgs(%q) error: %v", tc.args, err)
			}
			if gotWidth != tc.wantWidth {
				t.Errorf("parseArgs(%q) width=%d, want %d", tc.args, gotWidth, tc.wantWidth)
			}
			if gotFilter != tc.wantFilter {
				t.Errorf("parseArgs(%q) filter=%q, want %q", tc.args, gotFilter, tc.wantFilter)
			}
		})
	}
}

// TestParseArgsRejectsNonPositiveWidth: a simulation screen cannot be allocated
// with no columns, so a zero or negative width must fail before paint runs
// instead of panicking inside tcell.
func TestParseArgsRejectsNonPositiveWidth(t *testing.T) {
	t.Parallel()

	for _, arg := range []string{"0", "-1", "-100"} {
		_, _, err := parseArgs([]string{arg})
		if err == nil {
			t.Errorf("parseArgs(%q) accepted a non-positive width", arg)
			continue
		}
		if !strings.Contains(err.Error(), "width must be positive") {
			t.Errorf("parseArgs(%q) error = %v, want an argument error", arg, err)
		}
	}
}

// TestPaintWideGlyphHasNoPhantomSpace: a wide glyph owns two screen columns, and
// tcell leaves its continuation cell empty. Dumping that empty cell as a space
// would paint one column too many ("界 X" instead of "界X"), misaligning the
// Unicode sample.
func TestPaintWideGlyphHasNoPhantomSpace(t *testing.T) {
	t.Parallel()

	rows := paint("界X", 10, 2)
	if len(rows) == 0 || rows[0] != "界X" {
		t.Errorf("paint(\"界X\") = %q, want %q", rows, "界X")
	}
}
