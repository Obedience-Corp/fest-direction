package cli

import (
	"io"

	"github.com/spf13/cobra"

	"github.com/Obedience-Corp/fest-direction/internal/anchor"
	"github.com/Obedience-Corp/fest-direction/internal/errs"
	"github.com/Obedience-Corp/fest-direction/internal/normalize"
)

func newTrailersCommand(a *app) *cobra.Command {
	var tree, workUnit string
	cmd := &cobra.Command{
		Use:   "trailers",
		Short: "Append Festival-Direction trailers for a captured git tree",
		Long: `Reads a commit message on stdin and writes it to stdout with
Festival-Direction and Festival-Normalization trailers.

--tree is a git tree SHA the caller already holds, such as the tree a
background job will pass to git commit-tree. The command hashes that tree,
not the index and not the working tree. The work unit is resolved the same
way as hook commit-msg: --work-unit, then $DIRECTION_WORK_UNIT, then
.direction/config.yaml (default_work_unit). With none configured, stdin is
copied through unchanged and the command exits 0.

A hash or git failure exits non-zero, writes the error to stderr, and writes
nothing to stdout. Pipe the original message to this command and pass its
successful stdout, unchanged, to git commit-tree. That stdout is the whole
message, not a trailer block to append, and it is unchanged when no work
unit is configured. commit-tree does not run the commit-msg hook.

If the configured path is missing from --tree because the work unit was
renamed as a whole since HEAD, the command hashes the new directory. It does
not rewrite .direction/config.yaml; the commit-msg hook does that.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return a.runTrailers(cmd, tree, workUnit)
		},
	}
	cmd.Flags().StringVar(&tree, "tree", "", "git tree SHA to hash (not the index)")
	cmd.Flags().StringVar(&workUnit, "work-unit", "", "work unit path (repo-relative or absolute)")
	_ = cmd.MarkFlagRequired("tree")
	return cmd
}

func (a *app) runTrailers(cmd *cobra.Command, tree, workUnit string) error {
	ctx := cmd.Context()
	msg, err := io.ReadAll(cmd.InOrStdin())
	if err != nil {
		return a.fail(cmd, errs.Wrap("trailers: read stdin", err))
	}
	repo, err := anchor.OpenRepo(ctx, ".")
	if err != nil {
		return a.fail(cmd, err)
	}
	rel, err := resolveWorkUnit(repo, workUnit)
	if err != nil {
		return a.fail(cmd, err)
	}
	out := msg
	if rel != "" {
		rewritten, err := anchor.TrailersForTree(ctx, repo, rel, tree, string(msg), normalize.Current())
		if err != nil {
			return a.fail(cmd, err)
		}
		out = []byte(rewritten)
	}
	if _, err := cmd.OutOrStdout().Write(out); err != nil {
		return a.fail(cmd, errs.Wrap("trailers: write stdout", err))
	}
	return nil
}
