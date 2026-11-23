package tviewmd

import (
	"strings"
	"testing"
)

// Nested block children used to be silently dropped by the inline flatteners:
// code, lists and tables inside a quote or a list item never reached the output.
func TestNestedBlocksRender(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want []string
	}{
		{"quote code", "> ```go\n> x := 1\n> ```\n", []string{"▎", "x := 1"}},
		{"quote list", "> - alpha\n> - beta\n", []string{"▎ • alpha", "▎ • beta"}},
		{"quote table", "> | A | B |\n> |---|---|\n> | x | y |\n", []string{"▎ A B", "▎ x y"}},
		{"list code", "- item\n  ```go\n  x := 1\n  ```\n", []string{"• item", "  x := 1"}},
		{"list table", "- item\n  | A |\n  |---|\n  | x |\n", []string{"• item", "  A", "  x"}},
		{"quote paragraphs", "> a\n>\n> b\n", []string{"▎ a", "▎ b"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := stripTags(Render(tc.src, Options{}))
			for _, want := range tc.want {
				if !strings.Contains(out, want) {
					t.Errorf("output missing %q:\n%s", want, out)
				}
			}
		})
	}
}

func TestParseKeepsNestedBlockKinds(t *testing.T) {
	blocks, err := Parse("> ```go\n> x := 1\n> ```\n\n- item\n  - child\n")
	if err != nil {
		t.Fatal(err)
	}
	if len(blocks) != 2 {
		t.Fatalf("got %d top-level blocks, want 2", len(blocks))
	}
	if blocks[0].Kind != BlockBlockquote || len(blocks[0].Blocks) != 1 || blocks[0].Blocks[0].Kind != BlockCodeBlock {
		t.Errorf("quote lost its code block: %+v", blocks[0])
	}
	if blocks[1].Kind != BlockList || len(blocks[1].Items) != 1 || len(blocks[1].Items[0].Blocks) == 0 {
		t.Errorf("list item lost its nested blocks: %+v", blocks[1])
	}
}

// TestNestedListIndentExact pins the layout that regressed when list items
// began rendering their full Blocks: each nesting level is exactly two cells.
func TestNestedListIndentExact(t *testing.T) {
	got := stripTags(Render("- parent\n  - child\n    - grandchild\n", Options{TableRowSpacing: CompactTableRows}))
	want := "• parent\n  • child\n    • grandchild"
	if got != want {
		t.Errorf("nested list indent =\n%q\nwant\n%q", got, want)
	}
}

// TestQuoteWithOnlyEmptyFenceKeepsBar: suppressing tag-only fence output must
// not suppress a blockquote's visible quote bar.
func TestQuoteWithOnlyEmptyFenceKeepsBar(t *testing.T) {
	got := stripTags(Render("> ```\n> ```\n", Options{}))
	if !strings.Contains(got, "▎") {
		t.Errorf("empty-fence quote lost its bar: %q", got)
	}
}
