package anchor

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Obedience-Corp/fest-direction/internal/direction"
	"github.com/Obedience-Corp/fest-direction/internal/normalize"
)

func fixedClock() time.Time { return time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC) }

func TestAnchorRefusesUncommittedDirection(t *testing.T) {
	t.Parallel()
	repo, wu := repoWithFixture(t)
	task := filepath.Join(wu, "001_IMPLEMENT", "01_data_layer", "01_link_project.md")
	b, _ := os.ReadFile(task)
	if err := os.WriteFile(task, append(b, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	_, _, err := Anchor(context.Background(), repo, wu, normalize.V1(), Options{})
	if !errors.Is(err, ErrDirtyTree) {
		t.Fatalf("err = %v, want ErrDirtyTree", err)
	}
	if !strings.Contains(err.Error(), "01_link_project.md") {
		t.Fatalf("error does not name the dirty path: %v", err)
	}
	rec, _, err := Anchor(context.Background(), repo, wu, normalize.V1(), Options{Force: true, Now: fixedClock})
	if err != nil {
		t.Fatal(err)
	}
	if !rec.Events[0].Forced {
		t.Fatal("forced event not marked")
	}
}

func TestAnchorCleanWorkUnit(t *testing.T) {
	t.Parallel()
	repo, wu := repoWithFixture(t)
	ctx := context.Background()
	rec, res, err := Anchor(ctx, repo, wu, normalize.V1(), Options{Now: fixedClock, Tool: Tool{Name: "direction", Version: "test"}})
	if err != nil {
		t.Fatal(err)
	}
	head, _ := repo.Head(ctx)
	ev := rec.Events[0]
	if ev.Head != head || ev.Forced || ev.AnchoredAt != "2026-08-21T12:00:00Z" || ev.Tool.Name != "direction" {
		t.Fatalf("event = %+v (head %s)", ev, head)
	}
	if rec.Source != "festivals/dashboard-DA0001" || ev.Source != rec.Source {
		t.Fatalf("source = %q / %q", rec.Source, ev.Source)
	}
	if rec.DirectionHash != res.DirectionHash || ev.SnapshotID != res.SnapshotID || rec.Subject == nil || rec.Subject.ID != "DA0001" {
		t.Fatalf("record/result mismatch: %+v vs %+v", rec, res)
	}
	path, _ := RecordPath(repo, res.DirectionHash)
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("record not written: %v", err)
	}
	// The record itself is outside the work unit, so the work unit stays clean.
	if dirty, paths, _ := repo.Dirty(ctx, rec.Source); dirty {
		t.Fatalf("anchoring dirtied the work unit: %v", paths)
	}
	// A second anchor appends rather than replaces.
	again, _, err := Anchor(ctx, repo, wu, normalize.V1(), Options{Now: fixedClock})
	if err != nil || len(again.Events) != 2 {
		t.Fatalf("second anchor: %+v, %v", again, err)
	}
}

func TestAnchorCancelled(t *testing.T) {
	t.Parallel()
	repo, wu := repoWithFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := Anchor(ctx, repo, wu, normalize.V1(), Options{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}

func TestAnchorAllowsStateOnlyChanges(t *testing.T) {
	t.Parallel()
	repo, wu := repoWithFixture(t)
	ctx := context.Background()
	// Flip execution state in the working tree without committing: the plan is unchanged.
	task := filepath.Join(wu, "001_IMPLEMENT", "01_data_layer", "02_design_data_layer.md")
	b, _ := os.ReadFile(task)
	if err := os.WriteFile(task, []byte(strings.Replace(string(b), "fest_status: pending", "fest_status: completed", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	if dirty, _, _ := repo.Dirty(ctx, "festivals/dashboard-DA0001"); !dirty {
		t.Fatal("precondition: tree should be dirty")
	}
	rec, res, err := Anchor(ctx, repo, wu, normalize.V1(), Options{Now: fixedClock})
	if err != nil {
		t.Fatalf("state-only dirty tree was refused: %v", err)
	}
	if rec.Events[0].Forced {
		t.Fatal("state-only anchor must not be forced")
	}
	// HEAD's tree is the untouched fixture, so the recorded snapshot is its golden id.
	if res.SnapshotID != "sha256:4faf484b9bfb324cff4657bf0a79df2e6db2e14689b69cfbe4555a1981ba822e" {
		t.Fatalf("snapshot should be HEAD's, got %s", res.SnapshotID)
	}
}

func TestHeadTreeMatchesWorkingTree(t *testing.T) {
	t.Parallel()
	repo, wu := repoWithFixture(t)
	ctx := context.Background()
	committed, cleanup, err := repo.headTree(ctx, "festivals/dashboard-DA0001")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	a, err := direction.Hash(ctx, committed, normalize.V1())
	if err != nil {
		t.Fatal(err)
	}
	b, err := direction.Hash(ctx, wu, normalize.V1())
	if err != nil {
		t.Fatal(err)
	}
	if a.SnapshotID != b.SnapshotID || a.DirectionHash != b.DirectionHash {
		t.Fatalf("HEAD tree differs from clean working tree: %+v vs %+v", a, b)
	}
}
