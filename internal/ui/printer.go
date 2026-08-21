package ui

import (
	"fmt"
	"io"

	brandgloss "github.com/Obedience-Corp/obey-shared/brand/lipgloss"
)

// Printer writes styled status lines. It is the one place the CLI composes
// glyphs and colors, so verbs stay free of presentation detail.
type Printer struct {
	out    io.Writer
	styles brandgloss.Styles
}

// NewPrinter resolves styles for out and returns a Printer bound to it.
func NewPrinter(out io.Writer, opts Options) Printer {
	if opts.Out == nil {
		opts.Out = out
	}
	return Printer{out: out, styles: Styles(opts)}
}

// Success prints a ✓ line.
func (p Printer) Success(msg string) { p.line(p.styles.OK.Render("✓"), msg) }

// Warn prints a ⚠ line.
func (p Printer) Warn(msg string) { p.line(p.styles.Warn.Render("⚠"), msg) }

// Fail prints a ✗ line.
func (p Printer) Fail(msg string) { p.line(p.styles.Err.Render("✗"), msg) }

// Field prints an aligned label/value pair.
func (p Printer) Field(label, value string) {
	_, _ = fmt.Fprintf(p.out, "%s%s\n",
		p.styles.Muted.Render(fmt.Sprintf("%-12s", label)),
		p.styles.Normal.Render(value))
}

func (p Printer) line(glyph, msg string) {
	_, _ = fmt.Fprintf(p.out, "%s %s\n", glyph, msg)
}
