// Package anchor turns a direction hash into evidence: it records the hash
// against a git HEAD before work begins (anchor records), and carries it on
// every commit (trailers). Git is driven through os/exec with context
// propagation; nothing here links against a git library.
package anchor

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Obedience-Corp/fest-direction/internal/errs"
)

var (
	// ErrNotARepo is returned when a path is not inside a git work tree.
	ErrNotARepo = errors.New("not inside a git repository")
	// ErrNoCommits is returned when HEAD does not exist yet.
	ErrNoCommits = errors.New("repository has no commits")
	// ErrOutsideRepo is returned for a path outside the repository root.
	ErrOutsideRepo = errors.New("path is outside the repository")
	// ErrGit wraps a failed git invocation; the operation carries stderr.
	ErrGit = errors.New("git command failed")
	// ErrDirtyTree is returned when the work unit has uncommitted changes.
	ErrDirtyTree = errors.New("work unit has uncommitted changes")
)

// Repo is a git work tree identified by its root.
type Repo struct {
	Root string
}

// gitFailure is a non-zero git exit with its stderr and code.
type gitFailure struct {
	args   string
	stderr string
	code   int
}

func (g *gitFailure) Error() string { return "git " + g.args + ": " + g.stderr }
func (g *gitFailure) Unwrap() error { return ErrGit }

// OpenRepo resolves the repository containing path.
func OpenRepo(ctx context.Context, path string) (Repo, error) {
	if _, err := os.Stat(path); err != nil {
		return Repo{}, errs.Wrap("open repo", err)
	}
	out, err := runGit(ctx, path, "rev-parse", "--show-toplevel")
	var gf *gitFailure
	if errors.As(err, &gf) && strings.Contains(gf.stderr, "not a git repository") {
		return Repo{}, errs.Wrap("open repo "+path, ErrNotARepo)
	}
	if err != nil {
		return Repo{}, err
	}
	root, err := filepath.EvalSymlinks(strings.TrimSpace(out))
	if err != nil {
		return Repo{}, errs.Wrap("open repo: resolve root", err)
	}
	return Repo{Root: root}, nil
}

// Head returns the full commit sha of HEAD.
func (r Repo) Head(ctx context.Context) (string, error) {
	out, err := runGit(ctx, r.Root, "rev-parse", "--verify", "-q", "HEAD")
	var gf *gitFailure
	if errors.As(err, &gf) && gf.code == 1 {
		return "", errs.Wrap("head", ErrNoCommits)
	}
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// Dirty reports whether sub (repo-relative, "." for the whole tree) has
// modified, staged, or untracked paths, and lists them.
func (r Repo) Dirty(ctx context.Context, sub string) (bool, []string, error) {
	if sub == "" {
		sub = "."
	}
	out, err := runGit(ctx, r.Root, "status", "--porcelain=v1", "--untracked-files=all", "--", sub)
	if err != nil {
		return false, nil, err
	}
	paths := parsePorcelain(out)
	return len(paths) > 0, paths, nil
}

// Rel converts path to a repo-relative, slash-separated path.
func (r Repo) Rel(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", errs.Wrap("rel "+path, err)
	}
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		abs = resolved
	}
	rel, err := filepath.Rel(r.Root, abs)
	if err != nil {
		return "", errs.Wrap("rel "+path, err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errs.Wrap("rel "+path, ErrOutsideRepo)
	}
	return filepath.ToSlash(rel), nil
}

func parsePorcelain(out string) []string {
	var paths []string
	for _, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		if len(line) < 4 {
			continue
		}
		p := line[3:]
		if i := strings.Index(p, " -> "); i >= 0 {
			p = p[i+4:]
		}
		paths = append(paths, strings.Trim(p, `"`))
	}
	return paths
}

// runGit executes git in dir and returns raw stdout. Callers trim as needed:
// porcelain output depends on leading spaces.
func runGit(ctx context.Context, dir string, args ...string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			return "", errs.Wrap("git "+strings.Join(args, " "), &gitFailure{args: strings.Join(args, " "), stderr: strings.TrimSpace(stderr.String()), code: exit.ExitCode()})
		}
		return "", errs.Wrap("git "+strings.Join(args, " "), err)
	}
	return stdout.String(), nil
}

// summarizePaths renders up to three paths plus a count for error text.
func summarizePaths(paths []string) string {
	const show = 3
	if len(paths) <= show {
		return strings.Join(paths, ", ")
	}
	return strings.Join(paths[:show], ", ") + " +" + strconv.Itoa(len(paths)-show) + " more"
}
