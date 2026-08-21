package attest

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/Obedience-Corp/obey-shared/festivalbundle"

	"github.com/Obedience-Corp/fest-direction/internal/anchor"
	"github.com/Obedience-Corp/fest-direction/internal/direction"
	"github.com/Obedience-Corp/fest-direction/internal/errs"
)

// PredicateTypeV1 identifies the direction-record predicate. It is namespaced
// under the repository that owns it; a move to a brand domain is a new
// version once any statement has been published.
const PredicateTypeV1 = "https://github.com/Obedience-Corp/fest-direction/predicate/direction-record/v1"

var (
	// ErrInvalidHash marks a hash field that is not sha256:<64 hex>.
	ErrInvalidHash = errors.New("invalid hash in direction record")
	// ErrPredicateType marks a statement whose predicate is not a direction record.
	ErrPredicateType = errors.New("not a direction-record statement")
)

// DirectionRecord is the predicate: the direction hash, the rules that
// produced it, the snapshot it was taken beside, identity, and the anchors
// that make it evidence.
type DirectionRecord struct {
	DirectionHash        string                      `json:"direction_hash"`
	NormalizationVersion int                         `json:"normalization_version"`
	SnapshotID           string                      `json:"snapshot_id"`
	Kind                 string                      `json:"kind"`
	Subject              *festivalbundle.SubjectMeta `json:"subject,omitempty"`
	Source               string                      `json:"source,omitempty"`
	Anchors              []AnchorRef                 `json:"anchors,omitempty"`
	Tool                 Tool                        `json:"tool"`
	CreatedAt            string                      `json:"created_at"`
}

// AnchorRef points at one place the direction hash was anchored.
type AnchorRef struct {
	Type string `json:"type"` // "record" | "git-commit"
	Ref  string `json:"ref"`  // record path (repo-relative) or commit sha
	At   string `json:"at,omitempty"`
}

// Tool identifies the producer.
type Tool struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// RecordFromResult builds the predicate for r.
func RecordFromResult(r direction.Result, anchors []AnchorRef, tool Tool, now time.Time) DirectionRecord {
	return DirectionRecord{
		DirectionHash:        r.DirectionHash,
		NormalizationVersion: r.NormalizationVersion,
		SnapshotID:           r.SnapshotID,
		Kind:                 r.Kind,
		Subject:              r.Subject,
		Source:               r.Source,
		Anchors:              anchors,
		Tool:                 tool,
		CreatedAt:            now.UTC().Format(time.RFC3339),
	}
}

// NewStatement wraps rec in a Statement whose subject is the bundle. The
// subject digest is the Festival Bundle SPEC §7.1 payload hash — a real
// SHA-256 over the canonical payload record stream — not a digest of the
// .festival zip bytes, which vary with packed_at.
func NewStatement(rec DirectionRecord, subjectName string) (Statement, error) {
	if !anchor.ValidHash(rec.DirectionHash) {
		return Statement{}, errs.Wrap("statement: direction_hash "+rec.DirectionHash, ErrInvalidHash)
	}
	if !anchor.ValidHash(rec.SnapshotID) {
		return Statement{}, errs.Wrap("statement: snapshot_id "+rec.SnapshotID, ErrInvalidHash)
	}
	pred, err := json.Marshal(rec)
	if err != nil {
		return Statement{}, errs.Wrap("statement: encode predicate", err)
	}
	if subjectName == "" {
		subjectName = defaultSubjectName(rec)
	}
	return Statement{
		Type:          StatementType,
		Subject:       []Subject{{Name: subjectName, Digest: map[string]string{"sha256": strings.TrimPrefix(rec.SnapshotID, "sha256:")}}},
		PredicateType: PredicateTypeV1,
		Predicate:     pred,
	}, nil
}

// ParseRecord extracts and validates the direction record from s.
func ParseRecord(s Statement) (DirectionRecord, error) {
	if s.PredicateType != PredicateTypeV1 {
		return DirectionRecord{}, errs.Wrap("predicateType "+s.PredicateType, ErrPredicateType)
	}
	var rec DirectionRecord
	if err := json.Unmarshal(s.Predicate, &rec); err != nil {
		return DirectionRecord{}, errs.Wrap("predicate: decode", errors.Join(ErrMalformedStatement, err))
	}
	if !anchor.ValidHash(rec.DirectionHash) || !anchor.ValidHash(rec.SnapshotID) {
		return DirectionRecord{}, errs.Wrap("predicate hashes", ErrInvalidHash)
	}
	if rec.NormalizationVersion < 1 {
		return DirectionRecord{}, errs.Wrap("predicate: normalization_version", ErrMalformedStatement)
	}
	return rec, nil
}

func defaultSubjectName(rec DirectionRecord) string {
	if rec.Subject != nil && rec.Subject.ID != "" {
		return rec.Subject.ID + ".festival"
	}
	if rec.Kind != "" {
		return rec.Kind + ".festival"
	}
	return "work-unit.festival"
}
