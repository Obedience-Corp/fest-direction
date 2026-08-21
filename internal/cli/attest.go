package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Obedience-Corp/fest-direction/internal/anchor"
	"github.com/Obedience-Corp/fest-direction/internal/attest"
	"github.com/Obedience-Corp/fest-direction/internal/version"
)

func newAttestCommand(a *app) *cobra.Command {
	var out, anchorsFrom string
	var force, asJSON bool
	cmd := &cobra.Command{
		Use:   "attest <path>",
		Short: "Emit an in-toto statement with the direction-record predicate",
		Long: `attest hashes a work-unit directory or a .festival bundle and writes a detached
in-toto Statement v1 (<name>.intoto.json beside it) whose subject digest is
the bundle's snapshot id and whose predicate carries the direction hash, the
normalization version, identity, and — with --anchors-from — the anchor
records and trailer-bearing commits that make it evidence.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			opts := attest.AttestOptions{Tool: attest.Tool{Name: "direction", Version: version.Version}}
			if anchorsFrom != "" {
				repo, err := anchor.OpenRepo(ctx, anchorsFrom)
				if err != nil {
					return a.fail(cmd, err)
				}
				opts.AnchorsFrom = &repo
			}
			stmt, res, err := attest.Attest(ctx, args[0], opts)
			if err != nil {
				return a.fail(cmd, err)
			}
			path := out
			if path == "" {
				path = attest.DefaultStatementPath(args[0])
			}
			if err := attest.WriteStatement(path, stmt, force); err != nil {
				return a.fail(cmd, err)
			}
			if asJSON {
				return writeJSON(cmd, stmt)
			}
			rec, _ := attest.ParseRecord(stmt)
			p := a.printer(cmd)
			p.Field("subject", stmt.Subject[0].Name)
			p.Field("digest", "sha256:"+stmt.Subject[0].Digest["sha256"])
			p.Field("direction", res.DirectionHash)
			p.Field("normalization", fmt.Sprintf("v%d", res.NormalizationVersion))
			p.Field("anchors", fmt.Sprintf("%d", len(rec.Anchors)))
			p.Field("written", path)
			return nil
		},
	}
	cmd.Flags().StringVarP(&out, "out", "o", "", "statement path (default: <name>.intoto.json beside the source)")
	cmd.Flags().StringVar(&anchorsFrom, "anchors-from", "", "repository whose records and trailers supply anchor references")
	cmd.Flags().BoolVar(&force, "force", false, "overwrite an existing statement")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print the statement as JSON")
	return cmd
}
