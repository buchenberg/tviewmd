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

// TestPartialThemeEmptyInlineCodeBGStaysTransparent covers the v0.4.0
// regression: per-field defaulting replaced an explicitly empty background
// with DefaultTheme's highlight, so code spans gained a lighter background
// they were not supposed to have. An empty background in a partial theme must
// stay empty so the caller gets only the foreground color change.
func TestPartialThemeEmptyInlineCodeBGStaysTransparent(t *testing.T) {
	theme := Theme{
		Link:         "#00ffff",
		InlineCodeFG: "#9988bb",
		InlineCodeBG: "",
	}
	if got := theme.WithDefaults().InlineCodeBG; got != "" {
		t.Errorf("empty InlineCodeBG was defaulted to %q, want transparent", got)
	}

	out := Render("`code`", Options{Theme: theme})
	if want := "[#9988bb::]code"; !strings.Contains(out, want) {
		t.Errorf("code span missing transparent-background tag %q:\n%s", want, out)
	}
	if strings.Contains(out, DefaultTheme().InlineCodeBG) {
		t.Errorf("code span gained the default highlight background:\n%s", out)
	}
}

// TestPartialThemeExplicitInlineCodeBGKept makes sure opting back into a
// highlight still works, as does asking tview to inherit the terminal color.
func TestPartialThemeExplicitInlineCodeBGKept(t *testing.T) {
	for _, bg := range []string{"#123456", "default"} {
		theme := Theme{InlineCodeFG: "#ffffff", InlineCodeBG: bg}
		if got := theme.WithDefaults().InlineCodeBG; got != bg {
			t.Errorf("WithDefaults changed explicit InlineCodeBG %q to %q", bg, got)
		}
		out := Render("`code`", Options{Theme: theme})
		if want := "[#ffffff:" + bg + ":]code"; !strings.Contains(out, want) {
			t.Errorf("code span missing tag %q:\n%s", want, out)
		}
	}
}

// TestZeroThemeWithDefaultsStillUsesEveryDefault pins the zero-value meaning
// of Theme now that WithDefaults short-circuits on an all-empty struct.
func TestZeroThemeWithDefaultsStillUsesEveryDefault(t *testing.T) {
	if got := (Theme{}).WithDefaults(); got != DefaultTheme() {
		t.Errorf("zero theme WithDefaults = %+v, want %+v", got, DefaultTheme())
	}
}

func TestThemeWithDefaultsExplicitFieldsUntouched(t *testing.T) {
	custom := Theme{
		Heading:      [6]string{"a", "b", "c", "d", "e", "f"},
		Link:         "l",
		InlineCodeFG: "fg",
		InlineCodeBG: "bg",
		CodeBlockFG:  "code",
		QuoteFG:      "quote",
		Hr:           "hr",
	}
	if got := custom.WithDefaults(); got != custom {
		t.Errorf("WithDefaults changed a fully explicit theme:\n got=%+v\nwant=%+v", got, custom)
	}
}
