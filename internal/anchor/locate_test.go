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

func TestParseRenamePairs(t *testing.T) {
	t.Parallel()
	in := "M\x00README.md\x00R100\x00festivals/active/demo/fest.yaml\x00festivals/.dungeon/demo/fest.yaml\x00A\x00notes\x00"
	got, err := parseRenamePairs(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got["festivals/active/demo/fest.yaml"] != "festivals/.dungeon/demo/fest.yaml" {
		t.Fatalf("renames: %#v", got)
	}
	if _, err := parseRenamePairs("R100\x00only-old\x00"); err == nil {
		t.Fatal("truncated rename record was accepted")
	}
}

func TestCommonRenameRoot(t *testing.T) {
	t.Parallel()
	rel := "festivals/active/demo"
	files := []string{rel + "/fest.yaml", rel + "/TODO.md"}
	renames := map[string]string{
		files[0]: "festivals/.dungeon/demo/fest.yaml",
		files[1]: "festivals/.dungeon/demo/TODO.md",
	}
	root, err := commonRenameRoot(rel, files, renames)
	if err != nil || root != "festivals/.dungeon/demo" {
		t.Fatalf("root=%q err=%v", root, err)
	}
	renames[files[1]] = "elsewhere/TODO.md"
	root, err = commonRenameRoot(rel, files, renames)
	if err != nil || root != "" {
		t.Fatalf("split move: root=%q err=%v", root, err)
	}
	delete(renames, files[1])
	root, err = commonRenameRoot(rel, files, renames)
	if err != nil || root != "" {
		t.Fatalf("deleted file: root=%q err=%v", root, err)
	}
}

func TestInjectTrailersFollowsDirectoryRename(t *testing.T) {
	repo, _ := repoWithFixture(t)
	ctx := t.Context()
	old := "festivals/dashboard-DA0001"
	writeFile(t, filepath.Join(repo.Root, ".direction/config.yaml"), "default_work_unit: "+old+"\n")
	gitIn(t, repo.Root, "add", "-A")
	gitIn(t, repo.Root, "commit", "-q", "-m", "configure work unit")

	neu := "festivals/.dungeon/completed/2026-10-08/dashboard-DA0001"
	if err := os.MkdirAll(filepath.Join(repo.Root, filepath.Dir(neu)), 0o755); err != nil {
		t.Fatal(err)
	}
	gitIn(t, repo.Root, "mv", old, neu)
	writeFile(t, filepath.Join(repo.Root, neu, "notes.md"), "added during the move\n")
	goal := filepath.Join(repo.Root, neu, "FESTIVAL_GOAL.md")
	b, err := os.ReadFile(goal)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(goal, append(b, []byte("\nMoved.\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	gitIn(t, repo.Root, "add", "-A")
	if err := repo.RetargetStagedRename(ctx); err != nil {
		t.Fatal(err)
	}
	staged := gitIn(t, repo.Root, "diff", "--cached", "--", ".direction/config.yaml")
	if !strings.Contains(staged, "+default_work_unit: "+neu) {
		t.Fatalf("pre-commit did not stage the new work unit:\n%s", staged)
	}

	msg := filepath.Join(t.TempDir(), "COMMIT_EDITMSG")
	if err := os.WriteFile(msg, []byte("dungeon the festival\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	changed, err := InjectTrailers(ctx, repo, old, msg, normalize.Current())
	if err != nil || !changed {
		t.Fatalf("inject: changed=%v err=%v", changed, err)
	}
	out, err := os.ReadFile(msg)
	if err != nil {
		t.Fatal(err)
	}
	tr, err := ParseTrailers(string(out))
	if err != nil {
		t.Fatalf("parse: %v\n%s", err, out)
	}
	want, err := direction.Hash(ctx, filepath.Join(repo.Root, neu), normalize.Current())
	if err != nil {
		t.Fatal(err)
	}
	if tr.DirectionHash != want.DirectionHash {
		t.Fatalf("trailer %s != moved tree %s", tr.DirectionHash, want.DirectionHash)
	}
	cfg, err := os.ReadFile(filepath.Join(repo.Root, ".direction/config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(cfg), "default_work_unit: "+neu+"\n") {
		t.Fatalf("config not retargeted:\n%s", cfg)
	}

	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := repo.resolveTreePath(cancelled, old, "HEAD"); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled: %v", err)
	}
}

func TestInjectTrailersDeletedWorkUnit(t *testing.T) {
	repo, _ := repoWithFixture(t)
	ctx := t.Context()
	rel := "festivals/dashboard-DA0001"
	writeFile(t, filepath.Join(repo.Root, ".direction/config.yaml"), "default_work_unit: "+rel+"\n")
	gitIn(t, repo.Root, "add", ".direction/config.yaml")
	gitIn(t, repo.Root, "rm", "-r", "-q", rel)
	msg := filepath.Join(t.TempDir(), "COMMIT_EDITMSG")
	if err := os.WriteFile(msg, []byte("remove the festival\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := InjectTrailers(ctx, repo, rel, msg, normalize.Current())
	if !errors.Is(err, ErrWorkUnitMissing) {
		t.Fatalf("delete: %v", err)
	}
	cfg, err := os.ReadFile(filepath.Join(repo.Root, ".direction/config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(cfg), rel) {
		t.Fatalf("config changed on a failed delete:\n%s", cfg)
	}
}

func TestInjectTrailersSplitMove(t *testing.T) {
	repo, _ := repoWithFixture(t)
	ctx := t.Context()
	rel := "festivals/dashboard-DA0001"
	gitIn(t, repo.Root, "mv", rel+"/fest.yaml", "fest.yaml")
	gitIn(t, repo.Root, "mv", rel, "elsewhere")
	msg := filepath.Join(t.TempDir(), "COMMIT_EDITMSG")
	if err := os.WriteFile(msg, []byte("split\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := InjectTrailers(ctx, repo, rel, msg, normalize.Current())
	if !errors.Is(err, ErrWorkUnitMissing) {
		t.Fatalf("split: %v", err)
	}
}
