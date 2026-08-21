package ui

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Obedience-Corp/obey-shared/brand"
)

func TestPaletteOutputPolicy(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		opts     Options
		wantMode brand.Mode
	}{
		{name: "--no-color forces plain even for a tty-less writer", opts: Options{NoColor: true, Out: &bytes.Buffer{}}, wantMode: brand.ModePlain},
		{name: "non-tty writer forces plain", opts: Options{Out: &bytes.Buffer{}}, wantMode: brand.ModePlain},
		{name: "explicit plain mode stays plain", opts: Options{Mode: brand.ModePlain, Out: &bytes.Buffer{}}, wantMode: brand.ModePlain},
		{name: "explicit dark mode on a pipe still degrades to plain", opts: Options{Mode: brand.ModeDark, Out: &bytes.Buffer{}}, wantMode: brand.ModePlain},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := Palette(tc.opts)
			if got.Mode != tc.wantMode {
				t.Fatalf("Palette(%+v).Mode = %q, want %q", tc.opts, got.Mode, tc.wantMode)
			}
			if got.ColorEnabled {
				t.Fatalf("ColorEnabled = true for plain output")
			}
		})
	}
}

func TestPrinterPlainOutput(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	p := NewPrinter(&buf, Options{NoColor: true})
	p.Success("hashed")
	p.Warn("dirty tree")
	p.Fail("mismatch")
	p.Field("direction", "sha256:abc")

	out := buf.String()
	for _, want := range []string{"✓ hashed", "⚠ dirty tree", "✗ mismatch", "direction", "sha256:abc"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "\x1b[") {
		t.Errorf("plain output contains ANSI escapes:\n%q", out)
	}
}
