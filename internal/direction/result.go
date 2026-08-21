// Package direction computes the direction hash of a work unit: the bundle.id
// of its normalized tree, returned beside the snapshot bundle.id of the raw
// tree. Both ids come from obey-shared/festivalbundle — SPEC §7.1 is called,
// never reimplemented.
package direction

import (
	"errors"

	"github.com/Obedience-Corp/obey-shared/festivalbundle"
)

// Result is what Hash reports for one work unit.
type Result struct {
	DirectionHash        string                      `json:"direction_hash"`
	NormalizationVersion int                         `json:"normalization_version"`
	SnapshotID           string                      `json:"snapshot_id"`
	Kind                 string                      `json:"kind"`
	Subject              *festivalbundle.SubjectMeta `json:"subject,omitempty"`
	Source               string                      `json:"source"`
}

// Options tune Hash beyond the policy.
type Options struct {
	// KeepNormalizedBundle, when set, copies the normalized .festival bundle
	// to this path so the direction record can be inspected or archived.
	KeepNormalizedBundle string
}

var (
	// ErrNotADirectory is returned when src is not a directory.
	ErrNotADirectory = errors.New("source is not a directory")
)
