package anchor

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/Obedience-Corp/fest-direction/internal/direction"
	"github.com/Obedience-Corp/fest-direction/internal/errs"
	"github.com/Obedience-Corp/fest-direction/internal/normalize"
)

// InjectTrailers hashes the staged version of the repo-relative work unit rel
// — exactly what the commit in progress will contain — and writes the
// direction trailers into the commit message at msgPath. It reports whether
// the file changed. Meant to run as git's commit-msg hook.
func InjectTrailers(ctx context.Context, repo Repo, rel, msgPath string, p normalize.Policy) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	tree, used, cleanup, err := repo.indexTree(ctx, rel)
	if err != nil {
		return false, err
	}
	defer cleanup()
	msg, err := os.ReadFile(msgPath)
	if err != nil {
		return false, errs.Wrap("commit-msg: read "+msgPath, err)
	}
	out, changed, err := hashAndAppend(ctx, tree, string(msg), p)
	if err != nil {
		return false, errs.Wrap("commit-msg "+rel, err)
	}
	if changed {
		if err := os.WriteFile(msgPath, []byte(out), 0o644); err != nil {
			return false, errs.Wrap("commit-msg: write "+msgPath, err)
		}
	}
	// Disk-only fallback for clones whose pre-commit shim is not installed yet.
	// git writes the commit tree before commit-msg, so staging here cannot
	// enter this commit. pre-commit stages the retarget in time.
	if err := repo.retargetConfiguredUnit(ctx, rel, used, false); err != nil {
		return false, err
	}
	return changed, nil
}

// RetargetStagedRename points default_work_unit at a work unit that was renamed
// as a whole in the index, and stages the config. A missing path that is not
// such a rename is left for the commit-msg hook to reject.
func (r Repo) RetargetStagedRename(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	cfg, err := ReadConfig(r)
	if err != nil || cfg.DefaultWorkUnit == "" {
		return err
	}
	out, err := runGit(ctx, r.Root, "write-tree")
	if err != nil {
		return err
	}
	used, err := r.resolveTreePath(ctx, cfg.DefaultWorkUnit, strings.TrimSpace(out))
	if errors.Is(err, ErrWorkUnitMissing) {
		return nil
	}
	if err != nil {
		return err
	}
	return r.retargetConfiguredUnit(ctx, cfg.DefaultWorkUnit, used, true)
}

// retargetConfiguredUnit points default_work_unit from → to. stage adds the
// file to the index. A config that names some other path is left alone.
func (r Repo) retargetConfiguredUnit(ctx context.Context, from, to string, stage bool) error {
	if from == to {
		return nil
	}
	cfg, err := ReadConfig(r)
	if err != nil {
		return err
	}
	if cfg.DefaultWorkUnit != from {
		return nil
	}
	path := filepath.Join(r.Root, filepath.FromSlash(configPath))
	if err := writeConfig(path, Config{DefaultWorkUnit: to}); err != nil {
		return err
	}
	if !stage {
		return nil
	}
	_, err = runGit(ctx, r.Root, "add", "--", configPath)
	return err
}

// TrailersForTree hashes rel inside the git tree treeish and returns message
// with direction trailers applied. treeish is a tree the caller already holds
// (the tree a background job will pass to git commit-tree). It is not the
// index and not the working tree. On failure the returned message is empty.
func TrailersForTree(ctx context.Context, repo Repo, rel, treeish, message string, p normalize.Policy) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	used, err := repo.resolveTreePath(ctx, rel, treeish)
	if err != nil {
		return "", err
	}
	dir, cleanup, err := repo.treeAt(ctx, used, treeish)
	if err != nil {
		return "", err
	}
	defer cleanup()
	out, _, err := hashAndAppend(ctx, dir, message, p)
	if err != nil {
		return "", errs.Wrap("trailers "+rel, err)
	}
	return out, nil
}

// hashAndAppend hashes the materialized work unit at dir and applies direction
// trailers to message. The commit-msg hook and TrailersForTree both use it;
// only the tree they materialize differs.
func hashAndAppend(ctx context.Context, dir, message string, p normalize.Policy) (string, bool, error) {
	res, err := direction.Hash(ctx, dir, p)
	if err != nil {
		return "", false, err
	}
	out, changed := AppendTrailers(message, res)
	return out, changed, nil
}
