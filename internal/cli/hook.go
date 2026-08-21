package cli

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/Obedience-Corp/fest-direction/internal/anchor"
	"github.com/Obedience-Corp/fest-direction/internal/errs"
	"github.com/Obedience-Corp/fest-direction/internal/normalize"
)

const workUnitEnv = "DIRECTION_WORK_UNIT"

func newHookCommand(a *app) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hook",
		Short: "Git hook entry points: commit-msg trailers, install, uninstall",
	}
	cmd.AddCommand(newHookCommitMsgCommand(a), newHookInstallCommand(a), newHookUninstallCommand(a))
	return cmd
}

func newHookCommitMsgCommand(a *app) *cobra.Command {
	var workUnit string
	cmd := &cobra.Command{
		Use:   "commit-msg <message-file>",
		Short: "Append Festival-Direction trailers for the staged work unit (git commit-msg hook)",
		Long: `Resolves the work unit from --work-unit, then $DIRECTION_WORK_UNIT, then
.direction/config.yaml (default_work_unit). With none configured the hook is a
no-op so unrelated repositories are never blocked. Otherwise it hashes the
staged version of the work unit and writes the trailers into the message.
Errors abort the commit (fail closed); git commit --no-verify bypasses it.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			repo, err := anchor.OpenRepo(ctx, ".")
			if err != nil {
				return a.fail(cmd, err)
			}
			rel, err := resolveWorkUnit(repo, workUnit)
			if err != nil {
				return a.fail(cmd, err)
			}
			if rel == "" {
				return nil
			}
			if _, err := anchor.InjectTrailers(ctx, repo, rel, args[0], normalize.Current()); err != nil {
				return a.fail(cmd, err)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&workUnit, "work-unit", "", "work unit path (repo-relative or absolute)")
	return cmd
}

func resolveWorkUnit(repo anchor.Repo, flag string) (string, error) {
	candidate := flag
	if candidate == "" {
		candidate = os.Getenv(workUnitEnv)
	}
	if candidate == "" {
		cfg, err := anchor.ReadConfig(repo)
		if err != nil {
			return "", err
		}
		candidate = cfg.DefaultWorkUnit
	}
	if candidate == "" {
		return "", nil
	}
	if !filepath.IsAbs(candidate) {
		candidate = filepath.Join(repo.Root, candidate)
	}
	return repo.Rel(candidate)
}

func newHookInstallCommand(a *app) *cobra.Command {
	var repoPath, workUnit string
	var force bool
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install the commit-msg shim (and optionally the default work unit)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			repo, err := anchor.OpenRepo(ctx, repoPath)
			if err != nil {
				return a.fail(cmd, err)
			}
			res, err := anchor.Install(ctx, repo, anchor.InstallOptions{WorkUnit: workUnit, Force: force})
			if err != nil {
				p := a.errPrinter(cmd)
				p.Fail(err.Error())
				if errors.Is(err, anchor.ErrForeignHook) {
					p.Warn("pass --force to keep the existing hook as commit-msg.before-direction and chain it")
				}
				return errs.ErrAlreadyPrinted
			}
			p := a.printer(cmd)
			p.Success("installed " + res.HookPath)
			if res.ConfigPath != "" {
				p.Field("work unit", workUnit)
				p.Field("config", res.ConfigPath)
				p.Warn("commit .direction/config.yaml so clones know the work unit; the shim itself is per clone")
			}
			if res.Chained {
				p.Warn("an existing commit-msg hook is chained before the shim")
			}
			if res.HooksPathCf != "" {
				p.Warn("core.hooksPath is set (" + res.HooksPathCf + "); the shim was written there")
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&repoPath, "repo", ".", "repository path")
	cmd.Flags().StringVar(&workUnit, "work-unit", "", "default work unit to record in .direction/config.yaml")
	cmd.Flags().BoolVar(&force, "force", false, "keep and chain an existing foreign commit-msg hook")
	return cmd
}

func newHookUninstallCommand(a *app) *cobra.Command {
	var repoPath string
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Remove the commit-msg shim, restoring a chained hook if any",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			repo, err := anchor.OpenRepo(cmd.Context(), repoPath)
			if err != nil {
				return a.fail(cmd, err)
			}
			if err := anchor.Uninstall(cmd.Context(), repo); err != nil {
				return a.fail(cmd, err)
			}
			a.printer(cmd).Success("removed the direction commit-msg shim")
			return nil
		},
	}
	cmd.Flags().StringVar(&repoPath, "repo", ".", "repository path")
	return cmd
}
