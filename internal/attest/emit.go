package attest

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Obedience-Corp/obey-shared/festivalbundle"

	"github.com/Obedience-Corp/fest-direction/internal/anchor"
	"github.com/Obedience-Corp/fest-direction/internal/direction"
	"github.com/Obedience-Corp/fest-direction/internal/errs"
	"github.com/Obedience-Corp/fest-direction/internal/normalize"
)

const maxTrailerCommits = 50

var (
	// ErrExists is returned when a statement file would be overwritten.
	ErrExists = errors.New("statement already exists")
	// ErrNotAWorkUnit is returned for a path that is neither a directory nor a .festival file.
	ErrNotAWorkUnit = errors.New("not a work-unit directory or .festival bundle")
)

// AttestOptions tune Attest.
type AttestOptions struct {
	// AnchorsFrom, when set, supplies anchor references from that
	// repository's records and trailer-bearing commits.
	AnchorsFrom *anchor.Repo
	Tool        Tool
	Now         func() time.Time
	// Policy defaults to normalize.Current().
	Policy *normalize.Policy
}

// Attest hashes the work unit or .festival bundle at src and returns a
// statement for it together with the hash result.
func Attest(ctx context.Context, src string, opts AttestOptions) (Statement, direction.Result, error) {
	if err := ctx.Err(); err != nil {
		return Statement{}, direction.Result{}, err
	}
	policy := normalize.Current()
	if opts.Policy != nil {
		policy = *opts.Policy
	}
	tree, isBundle, cleanup, err := workUnitTree(ctx, src, false)
	if err != nil {
		return Statement{}, direction.Result{}, err
	}
	defer cleanup()
	res, err := direction.Hash(ctx, tree, policy)
	if err != nil {
		return Statement{}, direction.Result{}, errs.Wrap("attest "+src, err)
	}
	if isBundle {
		if abs, err := filepath.Abs(src); err == nil {
			res.Source = abs
		}
	}
	anchors, err := collectAnchors(ctx, opts.AnchorsFrom, res)
	if err != nil {
		return Statement{}, direction.Result{}, err
	}
	now := time.Now
	if opts.Now != nil {
		now = opts.Now
	}
	rec := RecordFromResult(res, anchors, opts.Tool, now())
	if opts.AnchorsFrom != nil {
		if rel, err := opts.AnchorsFrom.Rel(res.Source); err == nil {
			rec.Source = rel
		}
	}
	stmt, err := NewStatement(rec, "")
	if err != nil {
		return Statement{}, direction.Result{}, err
	}
	return stmt, res, nil
}

// WriteStatement writes s to path, refusing to overwrite unless force.
func WriteStatement(path string, s Statement, force bool) error {
	if _, err := os.Stat(path); err == nil && !force {
		return errs.Wrap("write "+path, ErrExists)
	}
	b, err := Marshal(s)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return errs.Wrap("write statement: mkdir", err)
	}
	return errs.Wrap("write "+path, os.WriteFile(path, b, 0o644))
}

// DefaultStatementPath is <src basename>.intoto.json beside src.
func DefaultStatementPath(src string) string {
	abs, err := filepath.Abs(src)
	if err != nil {
		abs = src
	}
	base := strings.TrimSuffix(filepath.Base(abs), ".festival")
	return filepath.Join(filepath.Dir(abs), base+".intoto.json")
}

// workUnitTree returns a directory holding the work unit: src itself for a
// directory, or the extracted payload for a .festival bundle. skipVerify
// extracts a bundle without checking bundle.id (used by Verify after a
// mismatch has already been recorded).
func workUnitTree(ctx context.Context, src string, skipVerify bool) (tree string, isBundle bool, cleanup func(), err error) {
	cleanup = func() {}
	st, err := os.Stat(src)
	if err != nil {
		return "", false, cleanup, errs.Wrap("open "+src, err)
	}
	if st.IsDir() {
		return src, false, cleanup, nil
	}
	if !strings.HasSuffix(strings.ToLower(src), ".festival") {
		return "", false, cleanup, errs.Wrap(src, ErrNotAWorkUnit)
	}
	tmp, err := os.MkdirTemp("", "direction-attest-*")
	if err != nil {
		return "", true, cleanup, errs.Wrap("attest: temp dir", err)
	}
	cleanup = func() { _ = os.RemoveAll(tmp) }
	dest := filepath.Join(tmp, "payload")
	if _, err := festivalbundle.Unbundle(ctx, src, dest, festivalbundle.UnbundleOptions{SkipVerify: skipVerify}); err != nil {
		return "", true, cleanup, errs.Wrap("unbundle "+src, err)
	}
	return dest, true, cleanup, nil
}

// collectAnchors gathers record events and trailer-bearing commits for the
// direction hash, deduplicated by reference, record first.
func collectAnchors(ctx context.Context, repo *anchor.Repo, res direction.Result) ([]AnchorRef, error) {
	if repo == nil {
		return nil, nil
	}
	var refs []AnchorRef
	seen := map[string]bool{}
	rec, err := anchor.ReadRecord(ctx, *repo, res.DirectionHash)
	switch {
	case errors.Is(err, anchor.ErrNoRecord):
	case err != nil:
		return nil, err
	default:
		path, _ := anchor.RecordPath(*repo, res.DirectionHash)
		rel, _ := filepath.Rel(repo.Root, path)
		refs = append(refs, AnchorRef{Type: "record", Ref: filepath.ToSlash(rel)})
		for _, ev := range rec.Events {
			if !seen[ev.Head] {
				seen[ev.Head] = true
				refs = append(refs, AnchorRef{Type: "git-commit", Ref: ev.Head, At: ev.AnchoredAt})
			}
		}
	}
	commits, err := anchor.CommitsWithTrailer(ctx, *repo, res.DirectionHash, maxTrailerCommits)
	if err != nil {
		return nil, err
	}
	for _, c := range commits {
		if !seen[c.SHA] {
			seen[c.SHA] = true
			refs = append(refs, AnchorRef{Type: "git-commit", Ref: c.SHA, At: c.Date})
		}
	}
	return refs, nil
}
