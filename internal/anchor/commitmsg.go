package anchor

import (
	"context"
	"os"

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
	tree, cleanup, err := repo.indexTree(ctx, rel)
	if err != nil {
		return false, err
	}
	defer cleanup()
	res, err := direction.Hash(ctx, tree, p)
	if err != nil {
		return false, errs.Wrap("commit-msg "+rel, err)
	}
	msg, err := os.ReadFile(msgPath)
	if err != nil {
		return false, errs.Wrap("commit-msg: read "+msgPath, err)
	}
	out, changed := AppendTrailers(string(msg), res)
	if !changed {
		return false, nil
	}
	return true, errs.Wrap("commit-msg: write "+msgPath, os.WriteFile(msgPath, []byte(out), 0o644))
}
