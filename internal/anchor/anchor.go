package anchor

import (
	"context"
	"time"

	"github.com/Obedience-Corp/fest-direction/internal/direction"
	"github.com/Obedience-Corp/fest-direction/internal/errs"
	"github.com/Obedience-Corp/fest-direction/internal/normalize"
)

// Options tune Anchor.
type Options struct {
	// Force anchors a work unit whose direction is uncommitted; the event is
	// marked forced and records the working tree rather than HEAD.
	Force bool
	// Tool is recorded on the event.
	Tool Tool
	// Now supplies the timestamp; nil means time.Now.
	Now func() time.Time
}

func (o Options) now() time.Time {
	if o.Now != nil {
		return o.Now()
	}
	return time.Now()
}

// Anchor records the direction hash of the work unit at workUnit against the
// current HEAD of repo. The hash must correspond to a tree a reader can later
// retrieve, so it is computed from HEAD's version of the work unit. A working
// tree that differs from HEAD only in execution state (status flips, progress
// events) has the same direction hash and anchors normally; a working tree
// whose direction differs is refused unless opts.Force.
func Anchor(ctx context.Context, repo Repo, workUnit string, p normalize.Policy, opts Options) (Record, direction.Result, error) {
	if err := ctx.Err(); err != nil {
		return Record{}, direction.Result{}, err
	}
	rel, err := repo.Rel(workUnit)
	if err != nil {
		return Record{}, direction.Result{}, err
	}
	head, err := repo.Head(ctx)
	if err != nil {
		return Record{}, direction.Result{}, err
	}
	res, forced, err := resolveResult(ctx, repo, rel, workUnit, p, opts.Force)
	if err != nil {
		return Record{}, direction.Result{}, err
	}
	rec := Record{
		DirectionHash:        res.DirectionHash,
		NormalizationVersion: res.NormalizationVersion,
		Kind:                 res.Kind,
		Subject:              res.Subject,
		Source:               rel,
	}
	ev := Event{
		Head:       head,
		Source:     rel,
		SnapshotID: res.SnapshotID,
		AnchoredAt: opts.now().UTC().Format(time.RFC3339),
		Forced:     forced,
		Tool:       opts.Tool,
	}
	stored, err := AppendEvent(ctx, repo, rec, ev)
	if err != nil {
		return Record{}, direction.Result{}, err
	}
	return stored, res, nil
}

// resolveResult hashes HEAD's tree and, when the working tree is dirty,
// decides whether the difference is state-only (anchor HEAD) or direction
// (refuse, or anchor the working tree when forced).
func resolveResult(ctx context.Context, repo Repo, rel, workUnit string, p normalize.Policy, force bool) (direction.Result, bool, error) {
	dirty, paths, err := repo.Dirty(ctx, rel)
	if err != nil {
		return direction.Result{}, false, err
	}
	committed, cleanup, err := repo.headTree(ctx, rel)
	if err != nil {
		return direction.Result{}, false, err
	}
	defer cleanup()
	headRes, err := direction.Hash(ctx, committed, p)
	if err != nil {
		return direction.Result{}, false, errs.Wrap("anchor "+rel+" (HEAD)", err)
	}
	if !dirty {
		return headRes, false, nil
	}
	workRes, err := direction.Hash(ctx, workUnit, p)
	if err != nil {
		return direction.Result{}, false, errs.Wrap("anchor "+rel, err)
	}
	if workRes.DirectionHash == headRes.DirectionHash {
		return headRes, false, nil
	}
	if !force {
		return direction.Result{}, false, errs.Wrap("anchor "+rel+": direction differs from HEAD ("+summarizePaths(paths)+")", ErrDirtyTree)
	}
	return workRes, true, nil
}
