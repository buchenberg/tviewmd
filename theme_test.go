package tviewmd

import (
	"strings"
	"testing"
)

func TestThemeWithDefaultsPerField(t *testing.T) {
	got := (Theme{Link: "#123456"}).WithDefaults()
	d := DefaultTheme()
	if got.Link != "#123456" {
		t.Errorf("explicit link color changed: %q", got.Link)
	}
	if got.Heading[0] != d.Heading[0] {
		t.Errorf("default heading color lost: %q, want %q", got.Heading[0], d.Heading[0])
	}
	if got.Hr != d.Hr {
		t.Errorf("default hr color lost: %q, want %q", got.Hr, d.Hr)
	}
}

// TestPartialThemeKeepsDefaults: setting one color used to blank every other
// field, because defaulting was all-or-nothing on the whole Theme struct.
func TestPartialThemeKeepsDefaults(t *testing.T) {
	src := "# Head\n\n[link](https://x)\n\n---\n"
	out := Render(src, Options{Theme: Theme{Link: "#123456"}})
	for _, want := range []string{"#123456", DefaultTheme().Heading[0], DefaultTheme().Hr} {
		if !strings.Contains(out, want) {
			t.Errorf("partial theme output missing %q:\n%s", want, out)
		}
	}
}

// TestNoColorTheme: NO_COLOR has an expressible theme instead of the zero
// struct, which means "use all the default colors".
func TestNoColorTheme(t *testing.T) {
	src := "# Head\n\n`code`\n\n> quote\n\n---\n"
	out := Render(src, Options{Theme: NoColorTheme()})
	if strings.Contains(out, "#") {
		t.Errorf("no-color theme painted a hex color:\n%s", out)
	}
	if !strings.Contains(out, "default") {
		t.Errorf("no-color theme produced no inheriting tags:\n%s", out)
	}
}

func TestZeroThemeStillUsesDefaults(t *testing.T) {
	src := "# Head\n\n---\n"
	zero := Render(src, Options{})
	explicit := Render(src, Options{Theme: DefaultTheme()})
	if zero != explicit {
		t.Errorf("zero theme differs from explicit DefaultTheme:\nzero:\n%s\ndefault:\n%s", zero, explicit)
	}
}
