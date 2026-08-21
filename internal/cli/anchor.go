package cli

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/Obedience-Corp/fest-direction/internal/anchor"
	"github.com/Obedience-Corp/fest-direction/internal/errs"
	"github.com/Obedience-Corp/fest-direction/internal/normalize"
	"github.com/Obedience-Corp/fest-direction/internal/version"
)

func newAnchorCommand(a *app) *cobra.Command {
	var force, asJSON bool
	cmd := &cobra.Command{
		Use:   "anchor <path>",
		Short: "Record the direction hash of a work unit against the current git HEAD",
		Long: `anchor computes the direction hash of HEAD's version of the work unit —
the tree a reader can retrieve later — and appends an event to
<repo>/.direction/anchors/sha256-<hash>.json. Uncommitted execution state
(status flips, progress events) is fine; an uncommitted change to the plan
itself is refused unless --force. The first event for a hash is the
pre-execution anchor. Commit the record.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			repo, err := anchor.OpenRepo(ctx, args[0])
			if err != nil {
				return a.fail(cmd, err)
			}
			rec, _, err := anchor.Anchor(ctx, repo, args[0], normalize.Current(), anchor.Options{
				Force: force,
				Tool:  anchor.Tool{Name: "direction", Version: version.Version},
			})
			if err != nil {
				if errors.Is(err, anchor.ErrDirtyTree) {
					a.errPrinter(cmd).Fail(err.Error() + "; commit the plan first, or pass --force to anchor the working tree")
					return errs.ErrAlreadyPrinted
				}
				return a.fail(cmd, err)
			}
			if asJSON {
				return writeJSON(cmd, rec)
			}
			path, _ := anchor.RecordPath(repo, rec.DirectionHash)
			relPath, _ := filepath.Rel(repo.Root, path)
			ev := rec.Events[len(rec.Events)-1]
			p := a.printer(cmd)
			p.Field("direction", rec.DirectionHash)
			p.Field("normalization", fmt.Sprintf("v%d", rec.NormalizationVersion))
			p.Field("snapshot", ev.SnapshotID)
			p.Field("head", ev.Head)
			p.Field("record", filepath.ToSlash(relPath))
			p.Field("events", fmt.Sprintf("%d", len(rec.Events)))
			if ev.Forced {
				p.Warn("anchored an uncommitted direction (--force); the event is marked forced and does not correspond to HEAD")
			}
			p.Warn("commit " + filepath.ToSlash(relPath) + " with your next commit — the record is the evidence")
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "anchor even if the work unit has uncommitted changes")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print the record as JSON")
	return cmd
}
