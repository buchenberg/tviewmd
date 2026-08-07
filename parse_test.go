package tviewmd

import (
	"strings"
	"testing"
)

func TestParseHeadingLevels(t *testing.T) {
	for level := 1; level <= 6; level++ {
		src := string([]byte{'#', '#', '#', '#', '#', '#'}[:level]) + " Title"
		blocks := exampleBlocks(t, src)
		if len(blocks) != 1 || blocks[0].Kind != BlockHeading {
			t.Fatalf("level %d: expected one heading, got %+v", level, blocks)
		}
		if blocks[0].Level != level {
			t.Errorf("level %d: got Level=%d", level, blocks[0].Level)
		}
		if plain(blocks[0].Segments) != "Title" {
			t.Errorf("level %d: segments=%q", level, plain(blocks[0].Segments))
		}
	}
}

func TestParseParagraphAndEmphasis(t *testing.T) {
	blocks := exampleBlocks(t, "plain **bold** *italic* `code`")
	if len(blocks) != 1 || blocks[0].Kind != BlockParagraph {
		t.Fatalf("expected one paragraph, got %+v", blocks)
	}
	segs := blocks[0].Segments

	var foundBold, foundItalic, foundCode bool
	for _, s := range segs {
		if s.Style.Bold && s.Text == "bold" {
			foundBold = true
		}
		if s.Style.Italic && s.Text == "italic" {
			foundItalic = true
		}
		if s.Code && s.Text == "code" {
			foundCode = true
		}
	}
	if !foundBold {
		t.Errorf("missing bold segment: %+v", segs)
	}
	if !foundItalic {
		t.Errorf("missing italic segment: %+v", segs)
	}
	if !foundCode {
		t.Errorf("missing code segment: %+v", segs)
	}
	if plain(segs) != "plain bold italic code" {
		t.Errorf("plain text mismatch: %q", plain(segs))
	}
}

func TestParseNestedEmphasis(t *testing.T) {
	// ***boitalic*** is bold+italic (emphasis level 2 wrapping level 1).
	blocks := exampleBlocks(t, "***boitalic***")
	segs := blocks[0].Segments
	var bi bool
	for _, s := range segs {
		if s.Style.Bold && s.Style.Italic && s.Text == "boitalic" {
			bi = true
		}
	}
	if !bi {
		t.Errorf("nested bold+italic not found: %+v", segs)
	}
}

func TestParseStrikethrough(t *testing.T) {
	blocks := exampleBlocks(t, "~~gone~~")
	for _, s := range blocks[0].Segments {
		if s.Text == "gone" && s.Style.Strikethrough {
			return
		}
	}
	t.Errorf("strikethrough not found: %+v", blocks[0].Segments)
}

func TestParseFencedCodeWithLanguage(t *testing.T) {
	src := "```go\nfmt.Println(\"hi\")\n```\n"
	blocks := exampleBlocks(t, src)
	cb := findBlock(blocks, BlockCodeBlock)
	if cb == nil {
		t.Fatalf("no code block: %+v", blocks)
	}
	if cb.Code.Lang != "go" {
		t.Errorf("Lang=%q want go", cb.Code.Lang)
	}
	if cb.Code.Source != `fmt.Println("hi")` {
		t.Errorf("Source=%q", cb.Code.Source)
	}
}

func TestParseFencedCodeNoLanguage(t *testing.T) {
	blocks := exampleBlocks(t, "```\nplain\n```")
	cb := findBlock(blocks, BlockCodeBlock)
	if cb == nil {
		t.Fatalf("no code block")
	}
	if cb.Code.Lang != "" {
		t.Errorf("Lang=%q want empty", cb.Code.Lang)
	}
	if cb.Code.Source != "plain" {
		t.Errorf("Source=%q", cb.Code.Source)
	}
}

func TestParseUnorderedList(t *testing.T) {
	blocks := exampleBlocks(t, "- one\n- two\n- three")
	lb := findBlock(blocks, BlockList)
	if lb == nil {
		t.Fatalf("no list: %+v", blocks)
	}
	if len(lb.Items) != 3 {
		t.Fatalf("items=%d want 3", len(lb.Items))
	}
	for i, it := range lb.Items {
		if it.Ordered {
			t.Errorf("item %d ordered", i)
		}
		if plain(it.Segments) != "" && !hasSegment(it.Segments, "one") && !hasSegment(it.Segments, "two") && !hasSegment(it.Segments, "three") {
			// sanity: at least one item carries text
		}
	}
	if plain(lb.Items[0].Segments) != "one" {
		t.Errorf("item0=%q", plain(lb.Items[0].Segments))
	}
}

func TestParseOrderedListStart(t *testing.T) {
	blocks := exampleBlocks(t, "3. first\n4. second")
	lb := findBlock(blocks, BlockList)
	if lb == nil {
		t.Fatalf("no list")
	}
	if !lb.Items[0].Ordered {
		t.Error("expected ordered")
	}
	if lb.Items[0].Index != 3 || lb.Items[1].Index != 4 {
		t.Errorf("indexes=%d,%d want 3,4", lb.Items[0].Index, lb.Items[1].Index)
	}
}

func TestParseNestedList(t *testing.T) {
	blocks := exampleBlocks(t, "- top\n  - nested")
	lb := findBlock(blocks, BlockList)
	if lb == nil {
		t.Fatalf("no list")
	}
	if len(lb.Items) != 1 {
		t.Fatalf("expected 1 top item, got %d", len(lb.Items))
	}
	if len(lb.Items[0].Children) != 1 {
		t.Fatalf("expected 1 nested item, got %d", len(lb.Items[0].Children))
	}
	if plain(lb.Items[0].Children[0].Segments) != "nested" {
		t.Errorf("nested=%q", plain(lb.Items[0].Children[0].Segments))
	}
}

func TestParseTaskList(t *testing.T) {
	blocks := exampleBlocks(t, "- [x] done\n- [ ] todo")
	lb := findBlock(blocks, BlockList)
	if lb == nil {
		t.Fatalf("no list")
	}
	// Task checkbox renders as a leading ☒/☐ marker segment.
	if !strings.Contains(plain(lb.Items[0].Segments), "☒") {
		t.Errorf("checked marker missing: %q", plain(lb.Items[0].Segments))
	}
	if !strings.Contains(plain(lb.Items[1].Segments), "☐") {
		t.Errorf("unchecked marker missing: %q", plain(lb.Items[1].Segments))
	}
}

func TestParseBlockquote(t *testing.T) {
	blocks := exampleBlocks(t, "> a quote")
	bq := findBlock(blocks, BlockBlockquote)
	if bq == nil {
		t.Fatalf("no blockquote: %+v", blocks)
	}
	if plain(bq.Segments) != "a quote" {
		t.Errorf("quote=%q", plain(bq.Segments))
	}
}

func TestParseThematicBreak(t *testing.T) {
	blocks := exampleBlocks(t, "a\n\n---\n\nb")
	if findBlock(blocks, BlockThematicBreak) == nil {
		t.Errorf("no thematic break: %+v", blocks)
	}
}

func TestParseLink(t *testing.T) {
	blocks := exampleBlocks(t, "[Charm](https://charm.sh)")
	segs := blocks[0].Segments
	var linkSeg *Segment
	for i, s := range segs {
		if s.Text == "Charm" {
			linkSeg = &segs[i]
		}
	}
	if linkSeg == nil {
		t.Fatalf("link label not found: %+v", segs)
	}
	if linkSeg.Link != "https://charm.sh" {
		t.Errorf("Link=%q", linkSeg.Link)
	}
}

func TestParseAutolink(t *testing.T) {
	blocks := exampleBlocks(t, "<https://example.com>")
	segs := blocks[0].Segments
	var found bool
	for _, s := range segs {
		if s.Link == "https://example.com" {
			found = true
		}
	}
	if !found {
		t.Errorf("autolink url not found: %+v", segs)
	}
}

func TestParseTable(t *testing.T) {
	src := "| Name | Age |\n|:-----|----:|\n| Ada | 36 |\n| Linus | 54 |\n"
	blocks := exampleBlocks(t, src)
	tb := findBlock(blocks, BlockTable)
	if tb == nil {
		t.Fatalf("no table: %+v", blocks)
	}
	if len(tb.Table.Header) != 2 {
		t.Fatalf("header cols=%d want 2", len(tb.Table.Header))
	}
	if plain(tb.Table.Header[0].Segments) != "Name" {
		t.Errorf("h0=%q", plain(tb.Table.Header[0].Segments))
	}
	if len(tb.Table.Rows) != 2 {
		t.Fatalf("rows=%d want 2", len(tb.Table.Rows))
	}
	// Column alignments from the delimiter row.
	if len(tb.Table.Aligns) < 2 || tb.Table.Aligns[0] != AlignLeft || tb.Table.Aligns[1] != AlignRight {
		t.Errorf("aligns=%+v", tb.Table.Aligns)
	}
	// Per-cell alignment mirrors the column.
	if tb.Table.Header[0].Align != AlignLeft {
		t.Errorf("h0 align=%v", tb.Table.Header[0].Align)
	}
}

func TestParseEmpty(t *testing.T) {
	blocks, err := Parse("")
	if err != nil {
		t.Fatal(err)
	}
	if len(blocks) != 0 {
		t.Errorf("expected 0 blocks, got %d", len(blocks))
	}
}

func TestParseMultipleBlocks(t *testing.T) {
	blocks := exampleBlocks(t, "# H\n\npara\n\n- a\n- b")
	if len(blocks) != 3 {
		t.Fatalf("expected 3 blocks, got %d", len(blocks))
	}
	if blocks[0].Kind != BlockHeading || blocks[1].Kind != BlockParagraph || blocks[2].Kind != BlockList {
		t.Errorf("kinds=%v %v %v", blocks[0].Kind, blocks[1].Kind, blocks[2].Kind)
	}
}
