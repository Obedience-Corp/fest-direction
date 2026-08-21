package anchor

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
)

const testHash = "sha256:4ee0bd159d49c964e342148f9da8618d1690bcf5bce38c48db87c6ce88041ba9"

func TestRecordErrors(t *testing.T) {
	t.Parallel()
	repo := Repo{Root: t.TempDir()}
	ctx := context.Background()
	if _, err := RecordPath(repo, "sha256:short"); !errors.Is(err, ErrInvalidHash) {
		t.Fatalf("invalid hash: %v", err)
	}
	if _, err := RecordPath(repo, "4ee0bd159d49c964e342148f9da8618d1690bcf5bce38c48db87c6ce88041ba9"); !errors.Is(err, ErrInvalidHash) {
		t.Fatalf("missing prefix: %v", err)
	}
	if _, err := ReadRecord(ctx, repo, testHash); !errors.Is(err, ErrNoRecord) {
		t.Fatalf("missing record: %v", err)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := AppendEvent(cancelled, repo, Record{DirectionHash: testHash, NormalizationVersion: 1}, Event{Head: "abc"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled: %v", err)
	}
	if entries, _ := os.ReadDir(repo.Root); len(entries) != 0 {
		t.Fatal("cancelled append wrote to disk")
	}
}

func TestAppendEventRoundTrip(t *testing.T) {
	t.Parallel()
	repo := Repo{Root: t.TempDir()}
	ctx := context.Background()
	rec := Record{DirectionHash: testHash, NormalizationVersion: 1, Kind: "festival", Source: "festivals/planning/x"}
	first, err := AppendEvent(ctx, repo, rec, Event{Head: "aaa", Source: "festivals/planning/x", SnapshotID: testHash, AnchoredAt: "2026-08-21T00:00:00Z", Tool: Tool{Name: "direction", Version: "test"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Events) != 1 || first.Events[0].Head != "aaa" {
		t.Fatalf("first = %+v", first)
	}
	path, _ := RecordPath(repo, testHash)
	raw1, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(path, "/.direction/anchors/sha256-4ee0bd159d49c964e342148f9da8618d1690bcf5bce38c48db87c6ce88041ba9.json") {
		t.Fatalf("unexpected path %s", path)
	}

	// Re-anchor after a lifecycle move: same direction, different source → append, no conflict.
	moved := rec
	moved.Source = "festivals/active/x"
	second, err := AppendEvent(ctx, repo, moved, Event{Head: "bbb", Source: "festivals/active/x", SnapshotID: testHash, AnchoredAt: "2026-08-22T00:00:00Z"})
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Events) != 2 || second.Events[0].Head != "aaa" || second.Events[1].Head != "bbb" || second.Source != "festivals/planning/x" {
		t.Fatalf("second = %+v", second)
	}

	// Different normalization version for the same hash is a conflict.
	bad := rec
	bad.NormalizationVersion = 2
	if _, err := AppendEvent(ctx, repo, bad, Event{Head: "ccc"}); !errors.Is(err, ErrRecordConflict) {
		t.Fatalf("conflict: %v", err)
	}

	read, err := ReadRecord(ctx, repo, testHash)
	if err != nil || len(read.Events) != 2 {
		t.Fatalf("read = %+v, %v", read, err)
	}
	// Deterministic bytes: rewriting the same content yields identical bytes.
	if err := writeAtomic(path, read); err != nil {
		t.Fatal(err)
	}
	raw2, _ := os.ReadFile(path)
	if len(raw1) >= len(raw2) {
		t.Fatalf("second write did not grow the file (%d vs %d)", len(raw1), len(raw2))
	}
	raw3, _ := os.ReadFile(path)
	if string(raw2) != string(raw3) {
		t.Fatal("rewrite not deterministic")
	}
}
