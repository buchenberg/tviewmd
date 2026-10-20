package tviewmd

import (
	"strings"
	"sync"
	"testing"
)

// TestRendererParityWithRender is the core guarantee: rendering a streamed
// buffer through Renderer must produce exactly the same output as a single
// Render call on the complete source. Streaming must never diverge from the
// final result.
func TestRendererParityWithRender(t *testing.T) {
	t.Parallel()

	src := "# Title\n\nSome **bold** and `code`.\n\n- one\n- two\n\n| A | B |\n|---|---|\n| 1 | 2 |\n"
	want := Render(src, Options{})

	r := NewRenderer(Options{})
	// Stream in irregular chunks, including splits inside markdown syntax.
	chunks := []string{
		"# Tit", "le\n\nSome **bo", "ld** and `co", "de`.\n\n- on", "e\n- two\n\n",
		"| A | B ", "|\n|", "---", "|---|\n", "| 1 | 2 |\n",
	}
	for _, c := range chunks {
		r.Append(c)
	}
	if got := r.String(); got != want {
		t.Errorf("Renderer output diverged from Render.\n got: %q\nwant: %q", got, want)
	}
}

// TestRendererWriteImplementsIOWriter verifies a Renderer can be used as an
// io.Writer sink for streamed content.
func TestRendererWriteImplementsIOWriter(t *testing.T) {
	t.Parallel()

	r := NewRenderer(Options{})
	n, err := r.Write([]byte("hello **world**"))
	if err != nil {
		t.Fatalf("Write error: %v", err)
	}
	if n != len("hello **world**") {
		t.Errorf("Write returned %d, want %d", n, len("hello **world**"))
	}
	if got, want := r.String(), Render("hello **world**", Options{}); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestRendererSourceAccumulates checks Source returns the raw accumulated
// markdown.
func TestRendererSourceAccumulates(t *testing.T) {
	t.Parallel()

	r := NewRenderer(Options{})
	r.Append("foo")
	r.Append("bar")
	if got := r.Source(); got != "foobar" {
		t.Errorf("Source()=%q, want %q", got, "foobar")
	}
}

// TestRendererCachesWhenUnchanged verifies the change-detection cache: String()
// after no new input returns the cached output without re-rendering (observable
// via the internal dirty flag in this white-box test).
func TestRendererCachesWhenUnchanged(t *testing.T) {
	t.Parallel()

	r := NewRenderer(Options{})
	r.Append("# hi")
	first := r.String()
	if r.dirty {
		t.Error("dirty should be false after String()")
	}
	second := r.String()
	if first != second {
		t.Errorf("cached output changed between calls:\n first=%q\nsecond=%q", first, second)
	}
	r.Append(" more")
	if !r.dirty {
		t.Error("dirty should be true after Append()")
	}
	third := r.String()
	if third == first {
		t.Errorf("output did not change after Append; got %q", third)
	}
}

// TestRendererReset verifies Reset clears both the source and the rendered cache.
func TestRendererReset(t *testing.T) {
	t.Parallel()

	r := NewRenderer(Options{})
	r.Append("# hi")
	_ = r.String()
	r.Reset()
	if got := r.Source(); got != "" {
		t.Errorf("Source() after Reset=%q, want empty", got)
	}
	if got := r.String(); got != "" {
		t.Errorf("String() after Reset=%q, want empty", got)
	}
}

// TestRendererEmpty verifies a fresh Renderer renders empty output.
func TestRendererEmpty(t *testing.T) {
	t.Parallel()

	if got := NewRenderer(Options{}).String(); got != "" {
		t.Errorf("empty Renderer String()=%q, want empty", got)
	}
}

// TestRendererDoesNotLeakAcrossFlushes verifies that streamed markdown that is
// re-rendered on each flush never leaves a style directive open. The trailing
// content must render in the baseline style. This is the streaming analogue of
// the underline-leak regression.
func TestRendererDoesNotLeakAcrossFlushes(t *testing.T) {
	t.Parallel()

	const sentinel = "QQQ"
	baseline, ok := cellStyleFor(t, "plain "+sentinel, 'Q')
	if !ok {
		t.Fatal("missing baseline sentinel")
	}

	r := NewRenderer(Options{})
	for _, c := range []string{"A [Charm](", "https://charm.sh)", " then plain " + sentinel} {
		r.Append(c)
	}
	got, ok := cellStyleFor(t, r.String(), 'Q')
	if !ok {
		t.Fatalf("missing sentinel cell in:\n%s", r.String())
	}
	if got != baseline {
		t.Errorf("style leaked across streamed link:\n got=%#v\nwant=%#v", got, baseline)
	}
}

// TestRendererStreamingBracketsDoNotInject verifies that brackets streamed in
// chunks (split across flushes) are escaped once assembled and never inject a
// color directive.
func TestRendererStreamingBracketsDoNotInject(t *testing.T) {
	t.Parallel()

	r := NewRenderer(Options{})
	for _, c := range []string{"see [", "gre", "en] now"} {
		r.Append(c)
	}
	if got := screenText(t, r.String()); !strings.Contains(got, "see [green] now") {
		t.Errorf("streamed bracket span not preserved:\n on-screen: %q", got)
	}
}

// TestRendererConcurrentAppendAndString exercises the mutex that guards the raw
// buffer: one goroutine streams chunks in while another flushes rendered output
// and reads the source. Run under -race, this fails if any of Append, Write,
// String, or Source touches r.raw / r.out / r.dirty without the lock.
func TestRendererConcurrentAppendAndString(t *testing.T) {
	t.Parallel()

	opts := Options{Width: 40}
	r := NewRenderer(opts)
	chunks := []string{
		"# Title\n\n",
		"A **bold** paragraph with `code`.\n\n",
		"| A | B |\n|---|---|\n| 1 | 2 |\n",
		"- one\n",
	}
	const rounds = 40

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < rounds; i++ {
			r.Append(chunks[i%len(chunks)])
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < rounds; i++ {
			_ = r.String()
			_ = r.Source()
		}
	}()
	wg.Wait()

	// After the producer finishes, the cache must agree with a full re-render
	// of the accumulated source: no flush may have observed a torn buffer.
	if got, want := r.String(), Render(r.Source(), opts); got != want {
		t.Errorf("concurrent flush diverged from Render\n got: %q\nwant: %q", got, want)
	}
}
