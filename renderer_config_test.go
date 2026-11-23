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
