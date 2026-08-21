package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Obedience-Corp/fest-direction/internal/direction"
	"github.com/Obedience-Corp/fest-direction/internal/normalize"
)

func newHashCommand(a *app) *cobra.Command {
	var asJSON bool
	var out string
	cmd := &cobra.Command{
		Use:   "hash <path>",
		Short: "Compute the direction hash and snapshot id of a work unit",
		Long: `hash normalizes the work unit at <path> — removing execution state
(fest_status, fest_updated, fest_working_dir, .fest/, TODO checkboxes,
fest.yaml status_history) — and reports the bundle.id of the normalized tree
as the direction hash, beside the snapshot bundle.id of the raw tree.
Nothing is written into <path>.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := direction.HashWith(cmd.Context(), args[0], normalize.V1(),
				direction.Options{KeepNormalizedBundle: out})
			if err != nil {
				return a.fail(cmd, err)
			}
			if asJSON {
				return writeJSON(cmd, res)
			}
			p := a.printer(cmd)
			p.Field("direction", res.DirectionHash)
			p.Field("normalization", fmt.Sprintf("v%d", res.NormalizationVersion))
			p.Field("snapshot", res.SnapshotID)
			p.Field("kind", res.Kind)
			p.Field("subject", subjectLabel(res))
			if out != "" {
				p.Field("normalized", out)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "print the result as JSON")
	cmd.Flags().StringVar(&out, "out", "", "also write the normalized .festival bundle to this path")
	return cmd
}

func subjectLabel(res direction.Result) string {
	if res.Subject == nil || res.Subject.ID == "" {
		return "—"
	}
	return res.Subject.ID
}
