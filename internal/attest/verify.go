package attest

import (
	"context"
	"errors"
	"os"
	"strconv"
	"strings"

	"github.com/Obedience-Corp/obey-shared/festivalbundle"

	"github.com/Obedience-Corp/fest-direction/internal/direction"
	"github.com/Obedience-Corp/fest-direction/internal/errs"
	"github.com/Obedience-Corp/fest-direction/internal/normalize"
)

var (
	// ErrSnapshotMismatch: the tree's bytes differ from the attested snapshot.
	ErrSnapshotMismatch = errors.New("snapshot id does not match")
	// ErrDirectionMismatch: the plan itself differs from the attested direction.
	ErrDirectionMismatch = errors.New("direction hash does not match")
	// ErrVersionMismatch: the statement's normalization version is unknown to this build.
	ErrVersionMismatch = errors.New("normalization version unknown to this build")
)

// Check is one verification step.
type Check struct {
	Name string `json:"name"`
	OK   bool   `json:"ok"`
	Want string `json:"want,omitempty"`
	Got  string `json:"got,omitempty"`
	Err  string `json:"error,omitempty"`
}

// Report lists every check that ran, in order.
type Report struct {
	OK     bool              `json:"ok"`
	Checks []Check           `json:"checks"`
	Record *DirectionRecord  `json:"record,omitempty"`
	Result *direction.Result `json:"result,omitempty"`
	// StateOnly is true when the snapshot moved but the direction did not:
	// execution state changed, the plan is intact.
	StateOnly bool `json:"state_only"`
}

func (r *Report) add(c Check) { r.Checks = append(r.Checks, c) }

// Verify recomputes the hashes of the work unit or bundle at src and compares
// them to the statement at statementPath. Checks run in order — statement,
// predicate-type, normalization-version, snapshot, direction — and the report
// records each; the returned error is the first failure's sentinel.
func Verify(ctx context.Context, src, statementPath string) (Report, error) {
	var rep Report
	if err := ctx.Err(); err != nil {
		return rep, err
	}
	rec, policy, err := verifyStatement(&rep, statementPath)
	if err != nil {
		return rep, err
	}
	rep.Record = &rec
	res, snapshotErr, err := hashForVerify(ctx, src, policy)
	if err != nil {
		rep.add(Check{Name: "hash", Err: err.Error()})
		return rep, err
	}
	rep.Result = &res
	var first error
	snapshotOK := snapshotErr == nil && strings.TrimPrefix(rec.SnapshotID, "sha256:") == strings.TrimPrefix(res.SnapshotID, "sha256:")
	if !snapshotOK {
		first = errs.Wrap("verify: snapshot", ErrSnapshotMismatch)
	}
	rep.add(Check{Name: "snapshot", OK: snapshotOK, Want: rec.SnapshotID, Got: res.SnapshotID})
	directionOK := rec.DirectionHash == res.DirectionHash
	if !directionOK && first == nil {
		first = errs.Wrap("verify: direction", ErrDirectionMismatch)
	}
	rep.add(Check{Name: "direction", OK: directionOK, Want: rec.DirectionHash, Got: res.DirectionHash})
	rep.StateOnly = !snapshotOK && directionOK
	rep.OK = first == nil
	return rep, first
}

// verifyStatement runs the statement, predicate-type, and version checks.
func verifyStatement(rep *Report, statementPath string) (DirectionRecord, normalize.Policy, error) {
	b, err := os.ReadFile(statementPath)
	if err != nil {
		wrapped := errs.Wrap("verify: read "+statementPath, errors.Join(ErrMalformedStatement, err))
		rep.add(Check{Name: "statement", Err: wrapped.Error()})
		return DirectionRecord{}, normalize.Policy{}, wrapped
	}
	stmt, err := ParseStatement(b)
	if err != nil {
		rep.add(Check{Name: "statement", Err: err.Error()})
		return DirectionRecord{}, normalize.Policy{}, err
	}
	rep.add(Check{Name: "statement", OK: true, Got: stmt.Type})
	rec, err := ParseRecord(stmt)
	if err != nil {
		rep.add(Check{Name: "predicate-type", Want: PredicateTypeV1, Got: stmt.PredicateType, Err: err.Error()})
		return DirectionRecord{}, normalize.Policy{}, err
	}
	rep.add(Check{Name: "predicate-type", OK: true, Got: stmt.PredicateType})
	policy, err := normalize.ForVersion(rec.NormalizationVersion)
	if err != nil {
		wrapped := errs.Wrap("verify: normalization version", ErrVersionMismatch)
		rep.add(Check{Name: "normalization-version", Got: itoa(rec.NormalizationVersion), Err: wrapped.Error()})
		return DirectionRecord{}, normalize.Policy{}, wrapped
	}
	rep.add(Check{Name: "normalization-version", OK: true, Got: itoa(rec.NormalizationVersion)})
	return rec, policy, nil
}

// hashForVerify hashes src under policy. For a bundle whose bundle.id no
// longer matches its payload, snapshotErr reports that and the tree is
// re-extracted without verification so the direction check still runs.
func hashForVerify(ctx context.Context, src string, policy normalize.Policy) (direction.Result, error, error) {
	tree, _, cleanup, err := workUnitTree(ctx, src, false)
	var snapshotErr error
	if err != nil && errors.Is(err, festivalbundle.ErrHashMismatch) {
		cleanup()
		snapshotErr = err
		tree, _, cleanup, err = workUnitTree(ctx, src, true)
	}
	if err != nil {
		return direction.Result{}, nil, err
	}
	defer cleanup()
	res, err := direction.Hash(ctx, tree, policy)
	if err != nil {
		return direction.Result{}, nil, errs.Wrap("verify "+src, err)
	}
	return res, snapshotErr, nil
}

func itoa(n int) string { return strconv.Itoa(n) }
