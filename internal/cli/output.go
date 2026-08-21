package cli

import (
	"encoding/json"
	"errors"

	"github.com/spf13/cobra"

	"github.com/Obedience-Corp/fest-direction/internal/errs"
	"github.com/Obedience-Corp/fest-direction/internal/normalize"
	"github.com/Obedience-Corp/fest-direction/internal/ui"
)

// errPrinter styles messages bound for stderr.
func (a *app) errPrinter(cmd *cobra.Command) ui.Printer {
	return ui.NewPrinter(cmd.ErrOrStderr(), ui.Options{NoColor: a.noColor})
}

// fail reports err to the user once, with a hint for known causes, and
// returns ErrAlreadyPrinted so main exits non-zero without repeating it.
func (a *app) fail(cmd *cobra.Command, err error) error {
	p := a.errPrinter(cmd)
	p.Fail(err.Error())
	if errors.Is(err, normalize.ErrUnknownField) {
		p.Warn("add the field to normalize.Policy and bump Version, or fix the document")
	}
	return errs.ErrAlreadyPrinted
}

// writeJSON prints v as indented JSON on stdout and nothing else.
func writeJSON(cmd *cobra.Command, v any) error {
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return errs.Wrap("write json", enc.Encode(v))
}
