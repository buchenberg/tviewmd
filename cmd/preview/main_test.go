package main

import "testing"

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
			gotWidth, gotFilter := parseArgs(tc.args)
			if gotWidth != tc.wantWidth {
				t.Errorf("parseArgs(%q) width=%d, want %d", tc.args, gotWidth, tc.wantWidth)
			}
			if gotFilter != tc.wantFilter {
				t.Errorf("parseArgs(%q) filter=%q, want %q", tc.args, gotFilter, tc.wantFilter)
			}
		})
	}
}
