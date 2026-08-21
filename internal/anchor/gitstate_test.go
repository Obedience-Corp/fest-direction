package anchor

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenRepoErrors(t *testing.T) {
	t.Parallel()
	requireGit(t)
	if _, err := OpenRepo(context.Background(), t.TempDir()); !errors.Is(err, ErrNotARepo) {
		t.Fatalf("plain dir: err = %v, want ErrNotARepo", err)
	}
	if _, err := OpenRepo(context.Background(), filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("missing path: expected an error")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := OpenRepo(ctx, t.TempDir()); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled: err = %v, want context.Canceled", err)
	}
}

func TestHeadOnUnbornBranch(t *testing.T) {
	t.Parallel()
	requireGit(t)
	dir := t.TempDir()
	gitIn(t, dir, "init", "-q", "-b", "main")
	repo, err := OpenRepo(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Head(context.Background()); !errors.Is(err, ErrNoCommits) {
		t.Fatalf("err = %v, want ErrNoCommits", err)
	}
}

func TestRelOutsideRepo(t *testing.T) {
	t.Parallel()
	repo := initRepo(t)
	if _, err := repo.Rel(t.TempDir()); !errors.Is(err, ErrOutsideRepo) {
		t.Fatalf("err = %v, want ErrOutsideRepo", err)
	}
	rel, err := repo.Rel(filepath.Join(repo.Root, "a", "b"))
	if err != nil || rel != "a/b" {
		t.Fatalf("Rel = %q, %v", rel, err)
	}
}

func TestDirty(t *testing.T) {
	t.Parallel()
	repo := initRepo(t)
	ctx := context.Background()
	head, err := repo.Head(ctx)
	if err != nil || len(head) != 40 {
		t.Fatalf("Head = %q, %v", head, err)
	}
	if dirty, _, err := repo.Dirty(ctx, "."); err != nil || dirty {
		t.Fatalf("fresh repo dirty=%v err=%v", dirty, err)
	}
	writeFile(t, filepath.Join(repo.Root, "unit", "plan.md"), "plan\n")
	writeFile(t, filepath.Join(repo.Root, "other", "note.md"), "note\n")
	gitIn(t, repo.Root, "add", "-A")
	gitIn(t, repo.Root, "commit", "-q", "-m", "files")

	writeFile(t, filepath.Join(repo.Root, "other", "note.md"), "changed\n")
	if dirty, _, err := repo.Dirty(ctx, "unit"); err != nil || dirty {
		t.Fatalf("change outside scope made unit dirty (dirty=%v err=%v)", dirty, err)
	}
	writeFile(t, filepath.Join(repo.Root, "unit", "plan.md"), "edited\n")
	dirty, paths, err := repo.Dirty(ctx, "unit")
	if err != nil || !dirty || len(paths) != 1 || paths[0] != "unit/plan.md" {
		t.Fatalf("modified: dirty=%v paths=%v err=%v", dirty, paths, err)
	}
	writeFile(t, filepath.Join(repo.Root, "unit", "new.md"), "new\n")
	if _, paths, _ := repo.Dirty(ctx, "unit"); len(paths) != 2 {
		t.Fatalf("untracked not listed: %v", paths)
	}
	if err := os.Remove(filepath.Join(repo.Root, "unit", "new.md")); err != nil {
		t.Fatal(err)
	}
}

func TestParsePorcelain(t *testing.T) {
	t.Parallel()
	got := parsePorcelain(" M a.md\n?? b/c.md\nR  old.md -> new.md\n")
	want := []string{"a.md", "b/c.md", "new.md"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}
