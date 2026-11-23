package tviewmd

import (
	"errors"
	"strings"
	"testing"

	"github.com/rivo/uniseg"
)

func TestRendererSetWidthReconfigures(t *testing.T) {
	const src = "| Feature description | Implementation status |\n|---|---|\n| streaming incremental parse with word wrap | blocked on width budget allocation |\n"
	r := NewRenderer(Options{Width: 120})
	r.Append(src)
	wide := r.String()
	r.SetWidth(40)
	narrow := r.String()
	if wide == narrow {
		t.Fatal("SetWidth did not change the rendered output")
	}
	if r.Source() != src {
		t.Fatal("SetWidth lost the accumulated source")
	}
	for i, line := range strings.Split(narrow, "\n") {
		if w := uniseg.StringWidth(stripTags(line)); w > 40 {
			t.Errorf("narrow render line %d is %d cells wide, want <= 40: %q", i, w, line)
		}
	}
}

func TestRendererSetOptionsChangesTheme(t *testing.T) {
	r := NewRenderer(Options{})
	r.Append("# Head\n")
	if out := r.String(); !strings.Contains(out, DefaultTheme().Heading[0]) {
		t.Fatalf("default theme not used: %q", out)
	}
	r.SetOptions(Options{Theme: NoColorTheme()})
	out := r.String()
	if strings.Contains(out, "#") {
		t.Errorf("SetOptions did not apply the new theme: %q", out)
	}
	if !strings.Contains(out, "default") {
		t.Errorf("SetOptions output has no inheriting tags: %q", out)
	}
}

func TestRendererStreamCap(t *testing.T) {
	r := NewRenderer(Options{MaxStreamBytes: 5})
	r.Append("hello world")
	if got := r.Source(); got != "hello" {
		t.Errorf("source = %q, want %q", got, "hello")
	}
	if !errors.Is(r.Err(), ErrStreamTooLarge) {
		t.Errorf("Err = %v, want ErrStreamTooLarge", r.Err())
	}
	if r.String() != "hello" {
		t.Errorf("rendered truncated source = %q", r.String())
	}
	r.Reset()
	if r.Err() != nil || r.Source() != "" {
		t.Errorf("Reset did not clear stream state: err=%v source=%q", r.Err(), r.Source())
	}
	r.Append("again")
	if r.Err() != nil || r.Source() != "again" {
		t.Errorf("stream unusable after Reset: err=%v source=%q", r.Err(), r.Source())
	}
}

func TestRendererWriteShortWrite(t *testing.T) {
	r := NewRenderer(Options{MaxStreamBytes: 3})
	n, err := r.Write([]byte("abcdef"))
	if n != 3 || !errors.Is(err, ErrStreamTooLarge) {
		t.Errorf("Write = (%d, %v), want (3, ErrStreamTooLarge)", n, err)
	}
	if got := r.Source(); got != "abc" {
		t.Errorf("source = %q, want %q", got, "abc")
	}
}

func TestRendererUnlimitedStream(t *testing.T) {
	r := NewRenderer(Options{MaxStreamBytes: -1})
	r.Append(strings.Repeat("x", DefaultMaxStreamBytes+1))
	if r.Err() != nil {
		t.Errorf("unlimited stream reported %v", r.Err())
	}
	if len(r.Source()) != DefaultMaxStreamBytes+1 {
		t.Errorf("unlimited stream truncated to %d bytes", len(r.Source()))
	}
}

func TestRendererStreamCapBoundaries(t *testing.T) {
	r := NewRenderer(Options{MaxStreamBytes: 4})
	r.Append("abcd") // exact fit, no error
	if r.Err() != nil || r.Source() != "abcd" {
		t.Errorf("exact fit failed: err=%v source=%q", r.Err(), r.Source())
	}
	r.Append("e") // zero room
	if !errors.Is(r.Err(), ErrStreamTooLarge) || r.Source() != "abcd" {
		t.Errorf("zero-room append: err=%v source=%q", r.Err(), r.Source())
	}
	n, err := r.Write([]byte("fg"))
	if n != 0 || !errors.Is(err, ErrStreamTooLarge) {
		t.Errorf("write on full stream = (%d, %v)", n, err)
	}
}

func TestRendererConcurrentReconfiguration(t *testing.T) {
	r := NewRenderer(Options{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 200; i++ {
			_, _ = r.Write([]byte("chunk "))
			r.SetWidth(40 + i%20)
			r.SetOptions(Options{Width: 60, Theme: NoColorTheme()})
			_ = r.Err()
		}
	}()
	for i := 0; i < 200; i++ {
		_ = r.String()
		_ = r.Source()
	}
	<-done
	r.Reset()
	if r.Err() != nil {
		t.Errorf("reset did not clear error: %v", r.Err())
	}
}
