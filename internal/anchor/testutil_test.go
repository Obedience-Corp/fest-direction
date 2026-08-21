package anchor

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
}

func gitIn(t *testing.T, dir string, args ...string) string {
	t.Helper()
	base := []string{"-C", dir, "-c", "user.name=test", "-c", "user.email=test@example.com", "-c", "commit.gpgsign=false"}
	out, err := exec.Command("git", append(base, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return string(out)
}

// initRepo creates a repository with one empty commit and returns it.
func initRepo(t *testing.T) Repo {
	t.Helper()
	requireGit(t)
	dir := t.TempDir()
	gitIn(t, dir, "init", "-q", "-b", "main")
	gitIn(t, dir, "commit", "-q", "--allow-empty", "-m", "init")
	repo, err := OpenRepo(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	return repo
}

// repoWithFixture copies the DA0001 baseline into a fresh repo and commits it.
// Returns the repo and the absolute work-unit path.
func repoWithFixture(t *testing.T) (Repo, string) {
	t.Helper()
	repo := initRepo(t)
	wu := filepath.Join(repo.Root, "festivals", "dashboard-DA0001")
	src := filepath.Join("..", "..", "testdata", "fixtures", "dashboard-DA0001-baseline")
	if err := os.CopyFS(wu, os.DirFS(src)); err != nil {
		t.Fatal(err)
	}
	gitIn(t, repo.Root, "add", "-A")
	gitIn(t, repo.Root, "commit", "-q", "-m", "add fixture")
	return repo, wu
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
