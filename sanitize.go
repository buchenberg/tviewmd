package tviewmd

import (
	"strings"
	"unicode/utf8"
)

// sanitizeText removes terminal control sequences and C0/C1 control characters
// from untrusted text. Newlines and tabs are preserved; everything else in the
// 0x00-0x1F and 0x7F-0x9F ranges is dropped. ESC introduces a CSI (`ESC [ ...`
// up to a final byte in 0x40-0x7E) or an OSC (`ESC ] ...` up to BEL or ST), and
// the whole sequence is removed rather than leaving its parameters behind as
// printable text.
//
// tview.Escape only neutralizes bracketed color tags; it does not sanitize
// control characters. Since tviewmd's main consumer renders LLM output, text
// reaching the terminal must not be able to issue SGR or other escape codes.
//
//nolint:funlen,gocognit,gocyclo // an explicit byte-state machine; splitting it would obscure the CSI/OSC handling
func sanitizeText(s string) string {
	var b strings.Builder
	changed := false
	for i := 0; i < len(s); {
		c := s[i]
		switch {
		case c == '\n' || c == '\t':
			if changed {
				b.WriteByte(c)
			}
			i++
		case c == '\r':
			// CommonMark treats a bare CR as a line ending. Deleting it would
			// silently join two lines; normalize to LF (CRLF stays one LF).
			if !changed {
				b.Grow(len(s))
				b.WriteString(s[:i])
				changed = true
			}
			if i+1 < len(s) && s[i+1] == '\n' {
				i++ // skip CR; the following LF becomes the line break
			} else {
				b.WriteByte('\n')
				i++
			}
		case c >= 0x20 && c != 0x7f:
			if c < 0x80 {
				if changed {
					b.WriteByte(c)
				}
				i++
				continue
			}
			r, size := utf8.DecodeRuneInString(s[i:])
			if r == utf8.RuneError && size == 1 { // raw 8-bit byte
				if !changed {
					b.Grow(len(s))
					b.WriteString(s[:i])
					changed = true
				}
				i++
				continue
			}
			if r >= 0x80 && r <= 0x9f { // C1 controls encoded as UTF-8
				if !changed {
					b.Grow(len(s))
					b.WriteString(s[:i])
					changed = true
				}
				i += size
				continue
			}
			if changed {
				b.WriteString(s[i : i+size])
			}
			i += size
		case c == 0x1b: // ESC
			if !changed {
				b.Grow(len(s))
				b.WriteString(s[:i])
				changed = true
			}
			i++
			if i < len(s) && s[i] == '[' { // CSI
				i++
				for i < len(s) {
					final := s[i]
					i++
					if final >= 0x40 && final <= 0x7e {
						break
					}
				}
			} else if i < len(s) && s[i] == ']' { // OSC
				i++
				for i < len(s) {
					switch {
					case s[i] == 0x07: // BEL
						i++
					case s[i] == 0x1b && i+1 < len(s) && s[i+1] == '\\': // ST
						i += 2
					default:
						i++
						continue
					}
					break
				}
			}
		default: // other C0 controls and DEL
			if !changed {
				b.Grow(len(s))
				b.WriteString(s[:i])
				changed = true
			}
			i++
		}
	}
	if !changed {
		return s
	}
	return b.String()
}

// sanitizeBlocks returns a copy of blocks with every untrusted string
// sanitized. RenderTView calls this once so width planning and painting see
// identical text, including for Blocks constructed directly by callers.
func sanitizeBlocks(blocks []Block) []Block {
	out := make([]Block, len(blocks))
	for i, b := range blocks {
		out[i] = b
		out[i].Segments = sanitizeSegments(b.Segments)
		out[i].Code.Source = sanitizeText(b.Code.Source)
		out[i].Items = sanitizeItems(b.Items)
		out[i].Table = sanitizeTable(b.Table)
		out[i].Blocks = sanitizeBlocks(b.Blocks)
	}
	return out
}

func sanitizeSegments(segs []Segment) []Segment {
	if segs == nil {
		return nil
	}
	out := make([]Segment, len(segs))
	copy(out, segs)
	for i := range out {
		out[i].Text = sanitizeText(out[i].Text)
		out[i].Link = sanitizeText(out[i].Link)
	}
	return out
}

func sanitizeItems(items []ListItem) []ListItem {
	if items == nil {
		return nil
	}
	out := make([]ListItem, len(items))
	copy(out, items)
	for i := range out {
		out[i].Segments = sanitizeSegments(items[i].Segments)
		out[i].Children = sanitizeItems(items[i].Children)
		out[i].Blocks = sanitizeBlocks(items[i].Blocks)
	}
	return out
}

func sanitizeTable(t TableData) TableData {
	out := TableData{Aligns: t.Aligns}
	out.Header = make([]TableCell, len(t.Header))
	for i, c := range t.Header {
		out.Header[i] = TableCell{Segments: sanitizeSegments(c.Segments), Align: c.Align}
	}
	out.Rows = make([][]TableCell, len(t.Rows))
	for i, row := range t.Rows {
		out.Rows[i] = make([]TableCell, len(row))
		for j, c := range row {
			out.Rows[i][j] = TableCell{Segments: sanitizeSegments(c.Segments), Align: c.Align}
		}
	}
	return out
}
