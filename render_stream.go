package tviewmd

import (
	"errors"
	"strings"
	"sync"
)

// ErrStreamTooLarge is reported by Renderer.Err (and returned by Renderer.Write)
// when accumulated source reaches Options.MaxStreamBytes. Input beyond the cap
// is dropped; the buffered prefix remains renderable.
var ErrStreamTooLarge = errors.New("tviewmd: stream exceeded MaxStreamBytes")

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
// A Renderer is safe for concurrent use: Append and Write are serialized
// against String, Source, Reset and SetOptions, so a producer goroutine can
// stream tokens in while another goroutine flushes the rendered output. In a
// tview app, the usual pattern is to flush from a QueueUpdateDraw callback,
// which runs on the event-loop goroutine, and assign the result directly:
//
//	app.QueueUpdateDraw(func() { tv.SetText(r.String()) })
//
// Do not call QueueUpdateDraw — or QueueUpdate, which waits for the event loop
// just the same — from inside an event-loop callback: the loop cannot run the
// queued function while it is executing the current one, so the call
// deadlocks. When you are already on that goroutine, update the TextView
// directly.
//
// On a terminal resize, call SetWidth (or SetOptions) and then String: the
// next flush re-renders the existing source with the new layout. Options could
// previously only be passed to NewRenderer, which forced callers to Reset and
// re-feed the whole stream just to change the width.
type Renderer struct {
	mu sync.Mutex

	opts Options

	raw   strings.Builder
	out   string // cached rendered output, valid only when !dirty
	dirty bool
	err   error // sticky: set once MaxStreamBytes is reached
}

// NewRenderer returns a Renderer that renders with the given Options.
func NewRenderer(opts Options) *Renderer {
	return &Renderer{opts: opts, dirty: true}
}

// streamLimit resolves Options.MaxStreamBytes. Negative means unlimited; zero
// selects the default cap.
func streamLimit(opts Options) int {
	switch {
	case opts.MaxStreamBytes < 0:
		return -1
	case opts.MaxStreamBytes == 0:
		return DefaultMaxStreamBytes
	default:
		return opts.MaxStreamBytes
	}
}

// Append adds markdown source to the stream and marks the rendered output stale.
// Input beyond Options.MaxStreamBytes is dropped; Err then reports
// ErrStreamTooLarge.
func (r *Renderer) Append(s string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if limit := streamLimit(r.opts); limit >= 0 {
		if room := limit - r.raw.Len(); room < len(s) {
			if room > 0 {
				r.raw.WriteString(s[:room])
			}
			r.err = ErrStreamTooLarge
			r.dirty = true
			return
		}
	}
	r.raw.WriteString(s)
	r.dirty = true
}

// Write implements io.Writer, appending markdown source to the stream so a
// Renderer can be used directly as a streaming sink. A short write is reported
// with ErrStreamTooLarge once the configured cap is reached.
func (r *Renderer) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if limit := streamLimit(r.opts); limit >= 0 {
		if room := limit - r.raw.Len(); room < len(p) {
			if room > 0 {
				n, _ := r.raw.Write(p[:room])
				r.dirty = true
				r.err = ErrStreamTooLarge
				return n, ErrStreamTooLarge
			}
			r.err = ErrStreamTooLarge
			return 0, ErrStreamTooLarge
		}
	}
	n, _ := r.raw.Write(p)
	r.dirty = true
	return n, nil
}

// SetOptions replaces the render options and marks the output stale. The next
// String call re-renders the existing source with the new options; accumulated
// source is preserved.
func (r *Renderer) SetOptions(opts Options) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.opts = opts
	r.dirty = true
}

// SetWidth updates only the layout width, leaving the rest of the options
// untouched. It is the resize hook for a tview consumer.
func (r *Renderer) SetWidth(width int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.opts.Width = width
	r.dirty = true
}

// Err reports whether the stream hit Options.MaxStreamBytes. The error is
// sticky until Reset.
func (r *Renderer) Err() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.err
}

// String returns the current fully-rendered, tview-tagged document. When no new
// input has arrived since the last call, the cached output is returned unchanged.
func (r *Renderer) String() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.dirty {
		return r.out
	}
	r.out = Render(r.raw.String(), r.opts)
	r.dirty = false
	return r.out
}

// Source returns the raw markdown accumulated so far, truncated at the stream
// cap when one was reached.
func (r *Renderer) Source() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.raw.String()
}

// Reset discards all accumulated source, rendered output, and stream error.
// Options are preserved.
func (r *Renderer) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.raw.Reset()
	r.out = ""
	r.err = nil
	r.dirty = true
}
