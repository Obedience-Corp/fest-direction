package direction

import (
	"context"
	"io"
	"os"
	"path/filepath"

	"github.com/Obedience-Corp/obey-shared/festivalbundle"

	"github.com/Obedience-Corp/fest-direction/internal/errs"
	"github.com/Obedience-Corp/fest-direction/internal/normalize"
)

const creator = "direction"

// Hash computes the direction hash and snapshot id of the work unit at src
// under policy p.
func Hash(ctx context.Context, src string, p normalize.Policy) (Result, error) {
	return HashWith(ctx, src, p, Options{})
}

// HashWith is Hash with Options. Both ids are bundle.ids produced by
// festivalbundle.Pack: the snapshot from the raw tree, the direction hash from
// the tree after normalize.Normalize. Pack applies the library's excludes and
// link vendoring before hashing, so the direction hash is exactly what
// `fest pack` would report for the normalized tree — a valid bundle.id, not a
// private digest. Nothing is ever written into src.
func HashWith(ctx context.Context, src string, p normalize.Policy, opts Options) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	src, err := filepath.Abs(src)
	if err != nil {
		return Result{}, errs.Wrap("hash: resolve "+src, err)
	}
	if st, err := os.Stat(src); err != nil {
		return Result{}, errs.Wrap("hash: stat "+src, err)
	} else if !st.IsDir() {
		return Result{}, errs.Wrap("hash "+src, ErrNotADirectory)
	}
	tmp, err := os.MkdirTemp("", "direction-*")
	if err != nil {
		return Result{}, errs.Wrap("hash: temp dir", err)
	}
	defer func() { _ = os.RemoveAll(tmp) }()

	m := readMeta(src)
	snapshot, err := pack(ctx, src, filepath.Join(tmp, "snapshot.festival"), m)
	if err != nil {
		return Result{}, errs.Wrap("hash: snapshot", err)
	}
	normalized := filepath.Join(tmp, "normalized")
	if _, err := normalize.Normalize(ctx, src, normalized, p); err != nil {
		return Result{}, errs.Wrap("hash", err)
	}
	directionPath := filepath.Join(tmp, "direction.festival")
	direction, err := pack(ctx, normalized, directionPath, m)
	if err != nil {
		return Result{}, errs.Wrap("hash: direction", err)
	}
	if opts.KeepNormalizedBundle != "" {
		if err := copyFile(directionPath, opts.KeepNormalizedBundle); err != nil {
			return Result{}, errs.Wrap("hash: keep normalized bundle", err)
		}
	}
	return Result{
		DirectionHash:        direction.Bundle.ID,
		NormalizationVersion: p.Version,
		SnapshotID:           snapshot.Bundle.ID,
		Kind:                 m.kind,
		Subject:              m.subject,
		Source:               src,
	}, nil
}

func pack(ctx context.Context, dir, out string, m meta) (*festivalbundle.Info, error) {
	return festivalbundle.Pack(ctx, dir, out, festivalbundle.PackOptions{
		Kind:    m.kind,
		Name:    m.name,
		Creator: creator,
		Subject: m.subject,
	})
}

func copyFile(from, to string) error {
	in, err := os.Open(from)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	outFile, err := os.Create(to)
	if err != nil {
		return err
	}
	if _, err := io.Copy(outFile, in); err != nil {
		_ = outFile.Close()
		return err
	}
	return outFile.Close()
}
