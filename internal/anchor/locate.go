package anchor

import (
	"context"
	"errors"
	"strings"

	"github.com/Obedience-Corp/fest-direction/internal/errs"
)

// ErrWorkUnitMissing is returned when the configured work unit is absent from
// a tree and was not renamed as a whole since HEAD. A deletion, or a split
// across more than one directory, is this error. The commit-msg hook fails
// closed on it.
var ErrWorkUnitMissing = errors.New("work unit is not in the tree")

// resolveTreePath returns rel when that path is in treeish. When it is not,
// and every file under rel at HEAD was renamed into a single new directory,
// it returns that directory. treeish is a commit or a tree object.
func (r Repo) resolveTreePath(ctx context.Context, rel, treeish string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	ok, err := r.pathInTree(ctx, treeish, rel)
	if err != nil {
		return "", err
	}
	if ok {
		return rel, nil
	}
	moved, err := r.renamedWorkUnit(ctx, rel, treeish)
	if err != nil {
		return "", err
	}
	if moved == "" {
		return "", errs.Wrap("work unit "+rel+" is not in "+treeish+" and was not renamed as a whole", ErrWorkUnitMissing)
	}
	ok, err = r.pathInTree(ctx, treeish, moved)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", errs.Wrap("work unit "+rel+" renamed to "+moved+" which is not in "+treeish, ErrWorkUnitMissing)
	}
	return moved, nil
}

// pathInTree reports whether treeish contains rel. A missing path is false,
// not an error. Any other git failure is returned.
func (r Repo) pathInTree(ctx context.Context, treeish, rel string) (bool, error) {
	_, err := runGit(ctx, r.Root, "cat-file", "-e", treeish+":"+rel)
	if err == nil {
		return true, nil
	}
	var gf *gitFailure
	if errors.As(err, &gf) && strings.Contains(gf.stderr, "does not exist") {
		return false, nil
	}
	return false, err
}

// renamedWorkUnit returns the single directory that now holds every file that
// was under rel at HEAD, or "" when the diff is not that kind of move.
func (r Repo) renamedWorkUnit(ctx context.Context, rel, treeish string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	diff, err := runGit(ctx, r.Root, "-c", "diff.renameLimit=10000", "diff", "--find-renames", "--name-status", "-z", "HEAD", treeish)
	if err != nil {
		return "", err
	}
	renames, err := parseRenamePairs(diff)
	if err != nil {
		return "", err
	}
	listed, err := runGit(ctx, r.Root, "ls-tree", "-r", "--name-only", "-z", "HEAD", "--", rel)
	if err != nil {
		return "", err
	}
	return commonRenameRoot(rel, splitNUL(listed), renames)
}

// commonRenameRoot is the destination directory when every file under rel was
// renamed there. It returns "" when any file was deleted, left in place, or
// renamed somewhere else.
func commonRenameRoot(rel string, files []string, renames map[string]string) (string, error) {
	if len(files) == 0 {
		return "", nil
	}
	root := ""
	for _, file := range files {
		dest, ok := renames[file]
		if !ok {
			return "", nil
		}
		suffix, ok := relativeTo(file, rel)
		if !ok {
			return "", errs.Wrap("rename scan: "+file+" is outside "+rel, ErrWorkUnitMissing)
		}
		candidate, ok := strings.CutSuffix(dest, suffix)
		if !ok || candidate == "" {
			return "", nil
		}
		candidate = strings.TrimSuffix(candidate, "/")
		if candidate == "" {
			return "", nil
		}
		if root == "" {
			root = candidate
			continue
		}
		if root != candidate {
			return "", nil
		}
	}
	return root, nil
}

func relativeTo(path, root string) (string, bool) {
	if path == root {
		return "", true
	}
	rest, ok := strings.CutPrefix(path, root+"/")
	if !ok {
		return "", false
	}
	return "/" + rest, true
}

// parseRenamePairs reads a `git diff --name-status -z` stream and returns
// old path → new path for renames. Copies and other statuses are skipped.
func parseRenamePairs(out string) (map[string]string, error) {
	parts := splitNUL(out)
	renames := make(map[string]string)
	for i := 0; i < len(parts); {
		status := parts[i]
		i++
		n := 1
		if strings.HasPrefix(status, "R") || strings.HasPrefix(status, "C") {
			n = 2
		}
		if i+n > len(parts) {
			return nil, errs.Wrap("parse diff name-status", ErrGit)
		}
		if strings.HasPrefix(status, "R") {
			renames[parts[i]] = parts[i+1]
		}
		i += n
	}
	return renames, nil
}

func splitNUL(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, "\x00")
	if parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	return parts
}
