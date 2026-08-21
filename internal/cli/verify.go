package cli

import (
	"github.com/spf13/cobra"

	"github.com/Obedience-Corp/fest-direction/internal/attest"
	"github.com/Obedience-Corp/fest-direction/internal/errs"
	"github.com/Obedience-Corp/fest-direction/internal/ui"
)

func newVerifyCommand(a *app) *cobra.Command {
	var statement string
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "verify <path> --statement <file>",
		Short: "Recompute both hashes of a work unit or bundle and check them against a statement",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rep, err := attest.Verify(cmd.Context(), args[0], statement)
			if asJSON {
				if werr := writeJSON(cmd, rep); werr != nil {
					return werr
				}
				if err != nil {
					return errs.ErrAlreadyPrinted
				}
				return nil
			}
			renderReport(a.printer(cmd), rep)
			if err != nil {
				return errs.ErrAlreadyPrinted
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&statement, "statement", "", "statement file to verify against (required)")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print the report as JSON")
	_ = cmd.MarkFlagRequired("statement")
	return cmd
}

func renderReport(p ui.Printer, rep attest.Report) {
	for _, c := range rep.Checks {
		switch {
		case c.OK:
			p.Success(c.Name + "  " + c.Got)
		case c.Err != "" && c.Want == "":
			p.Fail(c.Name + "  " + c.Err)
		default:
			p.Fail(c.Name + "  want " + c.Want + " got " + c.Got)
		}
	}
	if rep.StateOnly {
		p.Warn("execution state changed; direction intact")
	}
	if rep.OK {
		p.Success("verified")
		return
	}
	for _, c := range rep.Checks {
		if !c.OK {
			p.Fail("verification failed: " + c.Name)
			return
		}
	}
}
