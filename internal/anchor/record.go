package anchor

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Obedience-Corp/obey-shared/festivalbundle"

	"github.com/Obedience-Corp/fest-direction/internal/errs"
)

const (
	recordsDir   = ".direction/anchors"
	hashPrefix   = "sha256:"
	recordSuffix = ".json"
)

var (
	// ErrInvalidHash is returned for a hash not of the form sha256:<64 hex>.
	ErrInvalidHash = errors.New("invalid direction hash")
	// ErrNoRecord is returned when no anchor record exists for a hash.
	ErrNoRecord = errors.New("no anchor record")
	// ErrRecordConflict is returned when an existing record disagrees on
	// identity (hash or normalization version) with the event being added.
	ErrRecordConflict = errors.New("anchor record conflict")

	hashRE = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
)

// Record is the durable local evidence for one direction hash: identity plus
// an append-only list of anchor events. The first event is the pre-execution
// anchor. Records live under <repo>/.direction/anchors/ and are committed.
type Record struct {
	DirectionHash        string                      `json:"direction_hash"`
	NormalizationVersion int                         `json:"normalization_version"`
	Kind                 string                      `json:"kind"`
	Subject              *festivalbundle.SubjectMeta `json:"subject,omitempty"`
	Source               string                      `json:"source"` // repo-relative path at first anchor
	Events               []Event                     `json:"events"`
}

// Event is one anchoring of the record's direction hash at a git HEAD.
type Event struct {
	Head       string `json:"head"`
	Source     string `json:"source"` // repo-relative path at this event; may move across lifecycle dirs
	SnapshotID string `json:"snapshot_id"`
	AnchoredAt string `json:"anchored_at"` // RFC3339 UTC
	Forced     bool   `json:"forced,omitempty"`
	Tool       Tool   `json:"tool"`
}

// Tool identifies the writer.
type Tool struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ValidHash reports whether h is a well-formed direction or snapshot hash.
func ValidHash(h string) bool { return hashRE.MatchString(h) }

// RecordPath returns the on-disk path for a direction hash's record.
func RecordPath(repo Repo, directionHash string) (string, error) {
	if !ValidHash(directionHash) {
		return "", errs.Wrap("record path "+directionHash, ErrInvalidHash)
	}
	name := "sha256-" + strings.TrimPrefix(directionHash, hashPrefix) + recordSuffix
	return filepath.Join(repo.Root, filepath.FromSlash(recordsDir), name), nil
}

// ReadRecord loads the record for directionHash.
func ReadRecord(ctx context.Context, repo Repo, directionHash string) (Record, error) {
	if err := ctx.Err(); err != nil {
		return Record{}, err
	}
	path, err := RecordPath(repo, directionHash)
	if err != nil {
		return Record{}, err
	}
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return Record{}, errs.Wrap("read record "+directionHash, ErrNoRecord)
	}
	if err != nil {
		return Record{}, errs.Wrap("read record "+path, err)
	}
	var rec Record
	if err := json.Unmarshal(b, &rec); err != nil {
		return Record{}, errs.Wrap("parse record "+path, err)
	}
	return rec, nil
}

// AppendEvent adds ev to rec's on-disk record, creating it on first use, and
// returns the stored record. Identity fields must agree with an existing
// record; Source may differ per event because work units move between
// lifecycle directories without their direction changing.
func AppendEvent(ctx context.Context, repo Repo, rec Record, ev Event) (Record, error) {
	if err := ctx.Err(); err != nil {
		return Record{}, err
	}
	path, err := RecordPath(repo, rec.DirectionHash)
	if err != nil {
		return Record{}, err
	}
	existing, err := ReadRecord(ctx, repo, rec.DirectionHash)
	switch {
	case errors.Is(err, ErrNoRecord):
		existing = rec
		existing.Events = nil
	case err != nil:
		return Record{}, err
	case existing.NormalizationVersion != rec.NormalizationVersion:
		return Record{}, errs.Wrap("append event: normalization version", ErrRecordConflict)
	}
	existing.Events = append(existing.Events, ev)
	if err := writeAtomic(path, existing); err != nil {
		return Record{}, err
	}
	return existing, nil
}

func writeAtomic(path string, rec Record) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return errs.Wrap("write record: mkdir", err)
	}
	data, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return errs.Wrap("write record: encode", err)
	}
	data = append(data, '\n')
	tmp, err := os.CreateTemp(filepath.Dir(path), ".record-*")
	if err != nil {
		return errs.Wrap("write record: temp", err)
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return errs.Wrap("write record", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return errs.Wrap("write record: close", err)
	}
	if err := os.Chmod(tmpName, 0o644); err != nil {
		_ = os.Remove(tmpName)
		return errs.Wrap("write record: chmod", err)
	}
	return errs.Wrap("write record: rename", os.Rename(tmpName, path))
}
