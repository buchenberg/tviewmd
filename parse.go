package tviewmd

import (
	"strings"

	"github.com/yuin/goldmark"
	gast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	extast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)

// gmd is the shared goldmark parser configured for CommonMark + GFM
// (tables, strikethrough, autolinks, task lists). Constructed once; goldmark
// parsers are safe for concurrent use.
var gmd = goldmark.New(goldmark.WithExtensions(extension.GFM))

// Parse converts markdown source into a slice of renderer-neutral Blocks.
// Parse never returns an error for goldmark (markdown is permissive), but the
// error result is retained for API symmetry with future parser backends.
func Parse(src string) ([]Block, error) {
	source := []byte(src)
	doc := gmd.Parser().Parse(text.NewReader(source))
	var blocks []Block
	for child := doc.FirstChild(); child != nil; child = child.NextSibling() {
		blocks = append(blocks, parseBlock(child, source)...)
	}
	return blocks, nil
}

// parseBlock maps a goldmark block node to zero or more Blocks. Some goldmark
// nodes (list items, table rows) are handled by their parent parsers.
func parseBlock(n gast.Node, source []byte) []Block {
	switch v := n.(type) {
	case *gast.Heading:
		return []Block{{Kind: BlockHeading, Level: v.Level, Segments: collectInline(v, source, Style{})}}
	case *gast.Paragraph, *gast.TextBlock:
		return []Block{{Kind: BlockParagraph, Segments: collectInline(n, source, Style{})}}
	case *gast.ThematicBreak:
		return []Block{{Kind: BlockThematicBreak}}
	case *gast.FencedCodeBlock:
		return []Block{{Kind: BlockCodeBlock, Code: fencedCode(v, source)}}
	case *gast.CodeBlock:
		return []Block{{Kind: BlockCodeBlock, Code: CodeBlock{Source: string(v.Lines().Value(source))}}}
	case *gast.Blockquote:
		return []Block{{Kind: BlockBlockquote, Segments: collectBlockquote(v, source)}}
	case *gast.List:
		return []Block{{Kind: BlockList, Items: parseList(v, source)}}
	case *extast.Table:
		return []Block{{Kind: BlockTable, Table: parseTable(v, source)}}
	case *gast.HTMLBlock:
		return []Block{{Kind: BlockParagraph, Segments: []Segment{{Text: string(v.Text(source))}}}}
	}
	return nil
}

// fencedCode extracts language + source text from a fenced code block.
func fencedCode(v *gast.FencedCodeBlock, source []byte) CodeBlock {
	lang := ""
	if l := v.Language(source); l != nil {
		lang = string(l)
	}
	return CodeBlock{Lang: lang, Source: strings.TrimRight(string(v.Lines().Value(source)), "\n")}
}

// collectBlockquote flattens a blockquote's block children into inline segments,
// inserting a newline between distinct paragraphs.
func collectBlockquote(bq *gast.Blockquote, source []byte) []Segment {
	var segs []Segment
	first := true
	for c := bq.FirstChild(); c != nil; c = c.NextSibling() {
		for _, blk := range parseBlock(c, source) {
			if len(blk.Segments) == 0 {
				continue
			}
			if !first {
				segs = append(segs, Segment{Text: "\n"})
			}
			segs = append(segs, blk.Segments...)
			first = false
		}
	}
	return segs
}

// parseList maps a goldmark List into ListItems, recursing into nested lists.
func parseList(list *gast.List, source []byte) []ListItem {
	items := make([]ListItem, 0)
	idx := 1
	if list.Start > 0 {
		idx = list.Start
	}
	for li := list.FirstChild(); li != nil; li = li.NextSibling() {
		item := ListItem{Ordered: list.IsOrdered(), Index: idx}
		for c := li.FirstChild(); c != nil; c = c.NextSibling() {
			if sub, ok := c.(*gast.List); ok {
				item.Children = append(item.Children, parseList(sub, source)...)
				continue
			}
			segs := collectInline(c, source, Style{})
			if len(item.Segments) > 0 && len(segs) > 0 {
				item.Segments = append(item.Segments, Segment{Text: " "})
			}
			item.Segments = append(item.Segments, segs...)
		}
		if item.Ordered {
			idx++
		}
		items = append(items, item)
	}
	return items
}

// parseTable maps a GFM Table into header + body rows.
func parseTable(t *extast.Table, source []byte) TableData {
	out := TableData{Aligns: convertAligns(t.Alignments)}
	for row := t.FirstChild(); row != nil; row = row.NextSibling() {
		switch r := row.(type) {
		case *extast.TableHeader:
			for c := r.FirstChild(); c != nil; c = c.NextSibling() {
				if cell, ok := c.(*extast.TableCell); ok {
					out.Header = append(out.Header, tableCell(cell, source))
				}
			}
		case *extast.TableRow:
			cells := make([]TableCell, 0)
			for c := r.FirstChild(); c != nil; c = c.NextSibling() {
				if cell, ok := c.(*extast.TableCell); ok {
					cells = append(cells, tableCell(cell, source))
				}
			}
			out.Rows = append(out.Rows, cells)
		}
	}
	return out
}

func tableCell(cell *extast.TableCell, source []byte) TableCell {
	return TableCell{
		Segments: collectInline(cell, source, Style{}),
		Align:    convertAlign(cell.Alignment),
	}
}

func convertAlign(a extast.Alignment) TextAlign {
	switch a {
	case extast.AlignLeft:
		return AlignLeft
	case extast.AlignRight:
		return AlignRight
	case extast.AlignCenter:
		return AlignCenter
	default:
		return AlignDefault
	}
}

func convertAligns(as []extast.Alignment) []TextAlign {
	out := make([]TextAlign, len(as))
	for i, a := range as {
		out[i] = convertAlign(a)
	}
	return out
}

// collectInline gathers the inline children of a node into styled Segments,
// starting from a base Style that parent emphasis may have set.
func collectInline(n gast.Node, source []byte, base Style) []Segment {
	var segs []Segment
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		segs = append(segs, inlineNode(c, source, base)...)
	}
	return segs
}

// inlineNode maps a single inline node to Segments, inheriting and extending
// the base style for nested emphasis.
func inlineNode(n gast.Node, source []byte, base Style) []Segment {
	switch v := n.(type) {
	case *gast.Text:
		var out []Segment
		if val := string(v.Value(source)); val != "" {
			out = append(out, Segment{Text: val, Style: base})
		}
		if v.SoftLineBreak() {
			out = append(out, Segment{Text: " ", Style: base})
		}
		if v.HardLineBreak() {
			out = append(out, Segment{Text: "\n", Style: base})
		}
		return out
	case *gast.String:
		return []Segment{{Text: string(v.Value), Style: base}}
	case *gast.CodeSpan:
		var b strings.Builder
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			switch t := c.(type) {
			case *gast.Text:
				b.Write(t.Value(source))
			case *gast.String:
				b.Write(t.Value)
			}
		}
		return []Segment{{Text: b.String(), Style: base, Code: true}}
	case *gast.Emphasis:
		st := base
		if v.Level == 1 {
			st.Italic = true
		} else {
			st.Bold = true
		}
		return collectInline(n, source, st)
	case *gast.Link:
		segs := collectInline(n, source, base)
		for i := range segs {
			segs[i].Link = string(v.Destination)
		}
		return segs
	case *gast.AutoLink:
		return []Segment{{Text: string(v.Label(source)), Style: base, Link: string(v.URL(source))}}
	case *gast.RawHTML:
		return []Segment{{Text: string(v.Text(source)), Style: base}}
	case *extast.Strikethrough:
		st := base
		st.Strikethrough = true
		return collectInline(n, source, st)
	case *extast.TaskCheckBox:
		box := "☐ "
		if v.IsChecked {
			box = "☒ "
		}
		return []Segment{{Text: box, Style: base}}
	case *gast.Image:
		// Terminal rendering can't show images; emit the alt text instead.
		return collectInline(n, source, base)
	}
	return nil
}
