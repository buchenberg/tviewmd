package tviewmd

import "testing"

// TestRenderEmptyInput pins the empty-source path: Parse yields no blocks for
// empty or whitespace-only input, so Render must return an empty document there
// rather than echoing the raw source (which could carry unescaped tview tags).
func TestRenderEmptyInput(t *testing.T) {
	t.Parallel()

	for _, src := range []string{"", "\n", "\n\n", "   \n\t\n  "} {
		if got := Render(src, Options{}); got != "" {
			t.Errorf("Render(%q)=%q, want empty output", src, got)
		}
	}
}
