package anchor

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Obedience-Corp/fest-direction/internal/direction"
	"github.com/Obedience-Corp/fest-direction/internal/normalize"
)

func TestInjectTrailers(t *testing.T) {
	t.Parallel()
	repo, wu := repoWithFixture(t)
	ctx := context.Background()
	rel := "festivals/dashboard-DA0001"
	msg := filepath.Join(t.TempDir(), "COMMIT_EDITMSG")
	if err := os.WriteFile(msg, []byte("[FE-DA0001] edit plan\n\n# Please enter the commit message\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Stage a plan change; the trailer must describe the staged tree, which now equals the working tree.
	goal := filepath.Join(wu, "FESTIVAL_GOAL.md")
	b, _ := os.ReadFile(goal)
	if err := os.WriteFile(goal, append(b, []byte("\nA staged change to the plan.\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	gitIn(t, repo.Root, "add", "-A")
	changed, err := InjectTrailers(ctx, repo, rel, msg, normalize.Current())
	if err != nil || !changed {
		t.Fatalf("inject: changed=%v err=%v", changed, err)
	}
	out, _ := os.ReadFile(msg)
	tr, err := ParseTrailers(string(out))
	if err != nil {
		t.Fatalf("parse: %v\n%s", err, out)
	}
	want, err := direction.Hash(ctx, wu, normalize.Current())
	if err != nil {
		t.Fatal(err)
	}
	if tr.DirectionHash != want.DirectionHash || tr.NormalizationVersion != normalize.Version {
		t.Fatalf("trailer %+v != staged tree %s", tr, want.DirectionHash)
	}
	if !strings.Contains(string(out), "\n\n# Please enter") {
		t.Fatalf("comment tail lost:\n%s", out)
	}

	// Second run: nothing to change.
	if changed, err := InjectTrailers(ctx, repo, rel, msg, normalize.Current()); err != nil || changed {
		t.Fatalf("second inject: changed=%v err=%v", changed, err)
	}

	// An unstaged further edit must not leak into the trailer.
	if err := os.WriteFile(goal, append(b, []byte("\nUnstaged edit.\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	if changed, err := InjectTrailers(ctx, repo, rel, msg, normalize.Current()); err != nil || changed {
		t.Fatalf("unstaged edit changed the trailer: changed=%v err=%v", changed, err)
	}

	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := InjectTrailers(cancelled, repo, rel, msg, normalize.Current()); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled: %v", err)
	}
}
