package direction

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Obedience-Corp/obey-shared/festivalbundle"

	"github.com/Obedience-Corp/fest-direction/internal/normalize"
)

const (
	goldenBaselineSnapshot = "sha256:4faf484b9bfb324cff4657bf0a79df2e6db2e14689b69cfbe4555a1981ba822e"
	goldenMutatedSnapshot  = "sha256:5c7eced663c2efabbae61079f881aa9547c727eac6a29ca65de9ae2a80aaa9cd"
	goldenPoCDirection     = "sha256:50ddaea058b7b8ee46f89422da257d80ccb863ca640d887474544c752cdc854a"
	// goldenV1Direction is the v1 direction hash of dashboard-DA0001. It is
	// recorded in testdata/fixtures/README.md; any normalization drift fails
	// here loudly instead of silently moving every anchor.
	goldenV1Direction = "sha256:4ee0bd159d49c964e342148f9da8618d1690bcf5bce38c48db87c6ce88041ba9"
)

func TestHashErrors(t *testing.T) {
	t.Parallel()
	t.Run("file path is not a directory", func(t *testing.T) {
		t.Parallel()
		_, err := Hash(context.Background(), filepath.Join(fixture("dashboard-DA0001-baseline"), "fest.yaml"), normalize.V1())
		if !errors.Is(err, ErrNotADirectory) {
			t.Fatalf("err = %v, want ErrNotADirectory", err)
		}
	})

	t.Run("unknown field fails closed", func(t *testing.T) {
		t.Parallel()
		src := copyTree(t, fixture("dashboard-DA0001-baseline"))
		injectUnknownField(t, filepath.Join(src, "001_IMPLEMENT", "01_data_layer", "03_implement_websocket.md"))
		_, err := Hash(context.Background(), src, normalize.V1())
		if !errors.Is(err, normalize.ErrUnknownField) {
			t.Fatalf("err = %v, want ErrUnknownField", err)
		}
	})
}

func TestReadMeta(t *testing.T) {
	t.Parallel()
	ritual := t.TempDir()
	if err := os.WriteFile(filepath.Join(ritual, "fest.yaml"), []byte("metadata:\n  id: RI-X0001\n  name: weekly\n  festival_type: ritual\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name     string
		src      string
		wantKind string
		wantID   string
	}{
		{name: "festival from fest.yaml", src: fixture("dashboard-DA0001-baseline"), wantKind: festivalbundle.KindFestival, wantID: "DA0001"},
		{name: "ritual from festival_type", src: ritual, wantKind: festivalbundle.KindRitual, wantID: "RI-X0001"},
		{name: "note from .workitem", src: fixture("workitem-note"), wantKind: festivalbundle.KindNote, wantID: "note-direction-fixture-2026-08-21"},
		{name: "bare directory is a workitem", src: t.TempDir(), wantKind: festivalbundle.KindWorkitem},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			m := readMeta(tc.src)
			if m.kind != tc.wantKind {
				t.Fatalf("kind = %q, want %q", m.kind, tc.wantKind)
			}
			if tc.wantID != "" && (m.subject == nil || m.subject.ID != tc.wantID) {
				t.Fatalf("subject = %+v, want id %q", m.subject, tc.wantID)
			}
		})
	}
}

func TestGoldenStableAcrossExecution(t *testing.T) {
	t.Parallel()
	base, err := Hash(context.Background(), fixture("dashboard-DA0001-baseline"), normalize.V1())
	if err != nil {
		t.Fatal(err)
	}
	mut, err := Hash(context.Background(), fixture("dashboard-DA0001-mutated"), normalize.V1())
	if err != nil {
		t.Fatal(err)
	}
	if base.SnapshotID != goldenBaselineSnapshot {
		t.Errorf("baseline snapshot = %s, want %s", base.SnapshotID, goldenBaselineSnapshot)
	}
	if mut.SnapshotID != goldenMutatedSnapshot {
		t.Errorf("mutated snapshot = %s, want %s", mut.SnapshotID, goldenMutatedSnapshot)
	}
	if base.SnapshotID == mut.SnapshotID {
		t.Errorf("snapshots should differ after execution")
	}
	if base.DirectionHash != mut.DirectionHash {
		t.Errorf("direction hash moved across execution: %s vs %s", base.DirectionHash, mut.DirectionHash)
	}
	if base.DirectionHash == base.SnapshotID {
		t.Errorf("direction hash equals snapshot for a festival with state")
	}
	if base.NormalizationVersion != 1 || base.Kind != festivalbundle.KindFestival || base.Subject == nil || base.Subject.ID != "DA0001" {
		t.Errorf("unexpected result metadata: %+v", base)
	}
	t.Logf("v1 direction hash: %s", base.DirectionHash)
}

func TestGoldenV1(t *testing.T) {
	t.Parallel()
	if goldenV1Direction == "REPLACE_ME" {
		t.Skip("v1 golden not yet recorded")
	}
	res, err := Hash(context.Background(), fixture("dashboard-DA0001-baseline"), normalize.V1())
	if err != nil {
		t.Fatal(err)
	}
	if res.DirectionHash != goldenV1Direction {
		t.Fatalf("v1 direction hash = %s, want %s — normalization drifted; bump normalize.Version and re-record", res.DirectionHash, goldenV1Direction)
	}
}

func TestGoldenPoCReproduction(t *testing.T) {
	t.Parallel()
	for _, fx := range []string{"dashboard-DA0001-baseline", "dashboard-DA0001-mutated"} {
		dir := copyTree(t, fixture(fx))
		applyPoCRules(t, dir)
		info, err := festivalbundle.Pack(context.Background(), dir, filepath.Join(t.TempDir(), "poc.festival"), festivalbundle.PackOptions{Kind: festivalbundle.KindFestival})
		if err != nil {
			t.Fatal(err)
		}
		if info.Bundle.ID != goldenPoCDirection {
			t.Errorf("%s: PoC rules → %s, want %s", fx, info.Bundle.ID, goldenPoCDirection)
		}
	}
}

func TestGoldenNonFestivalKindEqualsSnapshot(t *testing.T) {
	t.Parallel()
	res, err := Hash(context.Background(), fixture("workitem-note"), normalize.V1())
	if err != nil {
		t.Fatal(err)
	}
	if res.Kind != festivalbundle.KindNote {
		t.Errorf("kind = %q, want note", res.Kind)
	}
	if res.DirectionHash != res.SnapshotID {
		t.Errorf("note: direction %s != snapshot %s", res.DirectionHash, res.SnapshotID)
	}
}

func TestHashKeepsNormalizedBundle(t *testing.T) {
	t.Parallel()
	out := filepath.Join(t.TempDir(), "normalized.festival")
	res, err := HashWith(context.Background(), fixture("dashboard-DA0001-baseline"), normalize.V1(), Options{KeepNormalizedBundle: out})
	if err != nil {
		t.Fatal(err)
	}
	info, err := festivalbundle.ReadInfo(context.Background(), out)
	if err != nil {
		t.Fatal(err)
	}
	if info.Bundle.ID != res.DirectionHash {
		t.Fatalf("kept bundle id %s != direction hash %s", info.Bundle.ID, res.DirectionHash)
	}
	if err := festivalbundle.Verify(context.Background(), out); err != nil {
		t.Fatalf("kept bundle does not verify: %v", err)
	}
}

// TestHashCancelledLeavesNoTempDirs is deliberately sequential and uses a
// private TMPDIR so parallel siblings creating direction-* dirs cannot skew it.
func TestHashCancelledLeavesNoTempDirs(t *testing.T) {
	tmpRoot := t.TempDir()
	t.Setenv("TMPDIR", tmpRoot)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := Hash(ctx, fixture("dashboard-DA0001-baseline"), normalize.V1())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	entries, err := os.ReadDir(tmpRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("temp dirs left behind: %d", len(entries))
	}
}

// TestHashCleansTempDirsOnSuccess runs sequentially for the same reason.
func TestHashCleansTempDirsOnSuccess(t *testing.T) {
	tmpRoot := t.TempDir()
	t.Setenv("TMPDIR", tmpRoot)
	if _, err := Hash(context.Background(), fixture("workitem-note"), normalize.V1()); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(tmpRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("temp dirs left behind after success: %d", len(entries))
	}
}
