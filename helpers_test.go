package tviewmd

import "strings"

// plain joins the visible text of a segment slice, ignoring styling/tags.
func plain(segs []Segment) string {
	var b strings.Builder
	for _, s := range segs {
		b.WriteString(s.Text)
	}
	return b.String()
}

// findBlock returns the first block of the given kind, or nil.
func findBlock(blocks []Block, kind BlockKind) *Block {
	for i := range blocks {
		if blocks[i].Kind == kind {
			return &blocks[i]
		}
	}
	return nil
}

func exampleBlocks(t testingT, src string) []Block {
	t.Helper()
	blocks, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse(%q) error: %v", src, err)
	}
	return blocks
}

// testingT is a minimal subset of *testing.T so the helpers above can be reused
// by both tests and the fuzz target without importing testing here.
type testingT interface {
	Helper()
	Fatalf(format string, args ...any)
}

func hasSegment(segs []Segment, text string) bool {
	for _, s := range segs {
		if strings.Contains(s.Text, text) {
			return true
		}
	}
	return false
}
