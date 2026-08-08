package tviewmd

import "strings"

// Renderer incrementally renders a markdown stream to tview color-tagged text.
//
// It is the streaming-oriented counterpart to Render: callers Append (or Write)
// text as it arrives — e.g. from an LLM token stream — and call String on a
// throttled tick to obtain the current fully-rendered document for a
// tview.TextView.SetText call.
//
// Unlike a naive line-splitting approach, String renders the entire accumulated
// buffer through the same Parse+RenderTView pipeline as Render. Markdown is
// block-oriented (a paragraph, list, or code fence is only complete once its
// closing construct arrives), so any line- or block-boundary split can render
// the settled portion differently from a final full render and produces a
// visible raw→formatted flash. Rendering the whole buffer avoids both; a
// trailing unclosed construct (an open code fence, a "**" mid-emphasis) is
// handled gracefully by goldmark and resolves as soon as its closer streams in.
// The Parse+RenderTView pipeline is sub-millisecond for typical LLM responses,
// so full re-render per flush is cheap; repeated calls without new input return
// the cached output.
//
// A Renderer is not safe for concurrent use. Drive it from a single goroutine —
// in a tview app, from a QueueUpdateDraw/QueueUpdate callback.
type Renderer struct {
	opts Options

	raw  strings.Builder
	out  string // cached rendered output, valid only when !dirty
	dirty bool
}

// NewRenderer returns a Renderer that renders with the given Options.
func NewRenderer(opts Options) *Renderer {
	return &Renderer{opts: opts, dirty: true}
}

// Append adds markdown source to the stream and marks the rendered output stale.
func (r *Renderer) Append(s string) {
	r.raw.WriteString(s)
	r.dirty = true
}

// Write implements io.Writer, appending markdown source to the stream so a
// Renderer can be used directly as a streaming sink.
func (r *Renderer) Write(p []byte) (int, error) {
	r.dirty = true
	return r.raw.Write(p)
}

// String returns the current fully-rendered, tview-tagged document. When no new
// input has arrived since the last call, the cached output is returned unchanged.
func (r *Renderer) String() string {
	if !r.dirty {
		return r.out
	}
	r.out = Render(r.raw.String(), r.opts)
	r.dirty = false
	return r.out
}

// Source returns the raw markdown accumulated so far.
func (r *Renderer) Source() string {
	return r.raw.String()
}

// Reset discards all accumulated source and rendered output.
func (r *Renderer) Reset() {
	r.raw.Reset()
	r.out = ""
	r.dirty = true
}
