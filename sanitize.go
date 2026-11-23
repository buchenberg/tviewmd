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
