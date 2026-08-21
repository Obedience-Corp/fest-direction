// Package ui resolves the shared Obedience Corp brand palette into Lip Gloss
// styles for the direction CLI. Output policy matches fest and camp: plain
// text when piped, under NO_COLOR, CI, TERM=dumb, or --no-color; the fire
// theme otherwise.
package ui

import (
	"io"
	"os"

	"github.com/Obedience-Corp/obey-shared/brand"
	brandgloss "github.com/Obedience-Corp/obey-shared/brand/lipgloss"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

// Options control palette resolution for one output stream.
type Options struct {
	// NoColor forces the plain palette regardless of terminal capability.
	NoColor bool
	// Mode selects the brand expression; the zero value is adaptive.
	Mode brand.Mode
	// Out is the stream being written; TTY detection applies to *os.File only.
	// Nil means os.Stdout.
	Out io.Writer
}

// Styles returns semantic styles resolved for opts.
func Styles(opts Options) brandgloss.Styles {
	return brandgloss.New(Palette(opts))
}

// Palette resolves the shared brand palette for opts. It never rewrites
// color tokens itself; brand owns the policy and Lip Gloss owns rendering.
func Palette(opts Options) brand.Palette {
	out := opts.Out
	if out == nil {
		out = os.Stdout
	}
	caps := brand.EnvironmentCapabilities(isTerminal(out), brand.ColorUnknown)
	if opts.NoColor {
		caps.NoColor = true
		caps.ColorDepth = brand.ColorNone
	}
	if !caps.ForcesPlain() {
		caps.DarkBackground = lipgloss.HasDarkBackground()
		caps.BackgroundKnown = true
	}
	return brand.Resolve(opts.Mode, caps)
}

func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	return ok && term.IsTerminal(int(f.Fd()))
}
