// Package cli wires the direction command tree. It owns flags, output, and
// exit codes; domain logic lives in the packages it calls. Verbs are added
// here as their packages land: hash (phase 1), anchor (phase 2), attest and
// verify (phase 3).
package cli

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/Obedience-Corp/fest-direction/internal/ui"
	"github.com/Obedience-Corp/fest-direction/internal/version"
)

// app carries process-wide options resolved from persistent flags.
type app struct {
	noColor bool
}

func (a *app) printer(cmd *cobra.Command) ui.Printer {
	return ui.NewPrinter(cmd.OutOrStdout(), ui.Options{NoColor: a.noColor})
}

// Execute runs the command tree under ctx.
func Execute(ctx context.Context) error {
	return NewRootCommand().ExecuteContext(ctx)
}

// NewRootCommand builds the root command with its persistent flags and verbs.
func NewRootCommand() *cobra.Command {
	a := &app{}
	root := &cobra.Command{
		Use:   "direction",
		Short: "Stable, verifiable direction records for Festival work units",
		Long: `direction computes a direction hash over a work unit with execution state
removed, anchors it in git before and during execution, and carries it in a
detached in-toto attestation beside the bundle's snapshot hash (bundle.id).

Verbs: hash, anchor, hook, attest, verify.`,
		Version:       version.String(),
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Context().Err()
		},
	}
	root.PersistentFlags().BoolVar(&a.noColor, "no-color", false, "disable colored output")
	root.AddCommand(newVersionCommand(a))
	root.AddCommand(newHashCommand(a))
	root.AddCommand(newAnchorCommand(a))
	root.AddCommand(newHookCommand(a))
	root.AddCommand(newAttestCommand(a))
	root.AddCommand(newVerifyCommand(a))
	return root
}
