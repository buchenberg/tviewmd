package tviewmd

// BlockKind enumerates block-level markdown elements.
type BlockKind int

const (
	// BlockParagraph is a run of inline content (text + emphasis + code + links).
	BlockParagraph BlockKind = iota
	// BlockHeading is a level 1-6 heading. Level holds the depth.
	BlockHeading
	// BlockCodeBlock is a fenced or indented code block. Code holds language+source.
	BlockCodeBlock
	// BlockList is an ordered or unordered list. Items holds the entries.
	BlockList
	// BlockBlockquote is a quoted block. Segments holds the flattened inline content.
	BlockBlockquote
	// BlockTable is a GFM table. Table holds header, rows, and column alignments.
	BlockTable
	// BlockThematicBreak is a horizontal rule.
	BlockThematicBreak
)

// Style is renderer-neutral inline formatting. Empty color fields mean
// "inherit the surrounding style"; the renderer maps the flags to its native
// emphasis syntax.
type Style struct {
	FG string
	BG string

	Bold          bool
	Italic        bool
	Underline     bool
	Strikethrough bool
	Dim           bool
}

// Segment is a single styled inline run of text. Exactly one rendering hint
// applies: Code, Link, or a plain Style.
type Segment struct {
	Text  string
	Style Style

	// Link, when non-empty, marks Text as the link label and Link as the URL.
	Link string

	// Code marks the segment as inline code. Renderers may use a distinct
	// background/foreground for code spans.
	Code bool
}

// TextAlign controls table column alignment.
type TextAlign int

const (
	// AlignDefault left-aligns (the CommonMark default rendering).
	AlignDefault TextAlign = iota
	AlignLeft
	AlignCenter
	AlignRight
)

// ListItem is one entry in a List block.
type ListItem struct {
	Segments []Segment
	Children []ListItem // nested list items
	Ordered  bool
	// Index is the 1-based ordinal for ordered items.
	Index int
}

// CodeBlock holds a fenced or indented code block.
type CodeBlock struct {
	Lang   string // language hint from the info string (may be empty)
	Source string
}

// TableCell is one cell of a table row.
type TableCell struct {
	Segments []Segment
	Align    TextAlign
}

// TableData holds a parsed GFM table.
type TableData struct {
	Header []TableCell
	Rows   [][]TableCell
	// Aligns is the per-column alignment declared in the delimiter row.
	Aligns []TextAlign
}

// Block is one block-level element. The field matching Kind is populated;
// the others are zero-valued.
type Block struct {
	Kind     BlockKind
	Level    int // heading level 1-6 (BlockHeading only)
	Segments []Segment // paragraph, heading, blockquote inline content
	Code     CodeBlock // BlockCodeBlock only
	Items    []ListItem // BlockList only
	Table    TableData // BlockTable only
}
