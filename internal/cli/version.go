package cli

import (
	"github.com/spf13/cobra"

	"github.com/Obedience-Corp/fest-direction/internal/version"
)

func newVersionCommand(a *app) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			p := a.printer(cmd)
			p.Field("direction", version.Version)
			p.Field("commit", version.Commit)
			p.Field("built", version.BuildDate)
			return nil
		},
	}
}
