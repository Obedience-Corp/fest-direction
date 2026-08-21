package attest

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Obedience-Corp/fest-direction/internal/anchor"
	"github.com/Obedience-Corp/fest-direction/internal/normalize"
)

func TestAttestErrors(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := Attest(ctx, fixture("dashboard-DA0001-baseline"), AttestOptions{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled: %v", err)
	}
	if _, _, err := Attest(context.Background(), filepath.Join(fixture("dashboard-DA0001-baseline"), "fest.yaml"), AttestOptions{}); !errors.Is(err, ErrNotAWorkUnit) {
		t.Fatalf("file input: %v", err)
	}
	path := attestDir(t, fixture("workitem-note"))
	stmt, _, _ := Attest(context.Background(), fixture("workitem-note"), AttestOptions{})
	if err := WriteStatement(path, stmt, false); !errors.Is(err, ErrExists) {
		t.Fatalf("overwrite: %v", err)
	}
	if err := WriteStatement(path, stmt, true); err != nil {
		t.Fatalf("force: %v", err)
	}
}

func TestAttestBundleMatchesDirectory(t *testing.T) {
	t.Parallel()
	dir := fixture("dashboard-DA0001-baseline")
	bundle := packFixture(t, dir)
	fromDir, resDir, err := Attest(context.Background(), dir, AttestOptions{Now: fixedClock, Tool: testTool})
	if err != nil {
		t.Fatal(err)
	}
	fromBundle, resBundle, err := Attest(context.Background(), bundle, AttestOptions{Now: fixedClock, Tool: testTool})
	if err != nil {
		t.Fatal(err)
	}
	if resDir.DirectionHash != resBundle.DirectionHash || resDir.SnapshotID != resBundle.SnapshotID {
		t.Fatalf("bundle input hashes differently: %+v vs %+v", resDir, resBundle)
	}
	if fromDir.Subject[0].Digest["sha256"] != fromBundle.Subject[0].Digest["sha256"] {
		t.Fatal("subject digests differ")
	}
	if resBundle.Source != bundle {
		t.Fatalf("bundle source = %q", resBundle.Source)
	}
	if DefaultStatementPath(bundle) != filepath.Join(filepath.Dir(bundle), "fx.intoto.json") {
		t.Fatalf("default path = %s", DefaultStatementPath(bundle))
	}
}

// TestAttestCollectsAnchors is sequential: it prepends a built binary to PATH
// for the commit-msg shim.
func TestAttestCollectsAnchors(t *testing.T) {
	repo, wu := gitRepoWithFixture(t)
	ctx := context.Background()
	// A task-start style record, then a commit carrying the trailer.
	if _, _, err := anchor.Anchor(ctx, repo, wu, normalize.Current(), anchor.Options{Now: fixedClock}); err != nil {
		t.Fatal(err)
	}
	if _, err := anchor.Install(ctx, repo, anchor.InstallOptions{WorkUnit: "festivals/dashboard-DA0001"}); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(t.TempDir(), "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	if out, err := execCommand("go", "build", "-o", filepath.Join(bin, "direction"), "../../cmd/direction"); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	if err := os.WriteFile(filepath.Join(repo.Root, "notes.md"), []byte("work\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitC(t, repo.Root, "add", "-A")
	gitC(t, repo.Root, "commit", "-q", "-m", "[FE-DA0001] work under the plan")

	stmt, _, err := Attest(ctx, wu, AttestOptions{AnchorsFrom: &repo, Now: fixedClock, Tool: testTool})
	if err != nil {
		t.Fatal(err)
	}
	rec, err := ParseRecord(stmt)
	if err != nil {
		t.Fatal(err)
	}
	var records, commits int
	for _, a := range rec.Anchors {
		switch a.Type {
		case "record":
			records++
		case "git-commit":
			commits++
		}
	}
	if records != 1 || commits != 2 {
		t.Fatalf("anchors = %+v (records %d, commits %d; want 1 record, 2 commits: the anchored HEAD and the trailer commit)", rec.Anchors, records, commits)
	}
	if rec.Source != "festivals/dashboard-DA0001" {
		t.Fatalf("source = %q", rec.Source)
	}
}
