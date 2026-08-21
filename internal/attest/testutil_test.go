package attest

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Obedience-Corp/obey-shared/festivalbundle"

	"github.com/Obedience-Corp/fest-direction/internal/anchor"
)

func fixture(name string) string {
	return filepath.Join("..", "..", "testdata", "fixtures", name)
}

func copyTree(t *testing.T, src string) string {
	t.Helper()
	dst := filepath.Join(t.TempDir(), filepath.Base(src))
	if err := os.CopyFS(dst, os.DirFS(src)); err != nil {
		t.Fatal(err)
	}
	return dst
}

func fixedClock() time.Time { return time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC) }

var testTool = Tool{Name: "direction", Version: "test"}

// attestDir writes a statement for dir into a temp file and returns its path.
func attestDir(t *testing.T, dir string) string {
	t.Helper()
	stmt, _, err := Attest(context.Background(), dir, AttestOptions{Tool: testTool, Now: fixedClock})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "s.intoto.json")
	if err := WriteStatement(path, stmt, false); err != nil {
		t.Fatal(err)
	}
	return path
}

func packFixture(t *testing.T, dir string) string {
	t.Helper()
	out := filepath.Join(t.TempDir(), "fx.festival")
	if _, err := festivalbundle.Pack(context.Background(), dir, out, festivalbundle.PackOptions{Kind: festivalbundle.KindFestival}); err != nil {
		t.Fatal(err)
	}
	return out
}

func replaceInFile(t *testing.T, path, old, new string) {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	s := strings.Replace(string(b), old, new, 1)
	if s == string(b) {
		t.Fatalf("%q not found in %s", old, path)
	}
	if err := os.WriteFile(path, []byte(s), 0o644); err != nil {
		t.Fatal(err)
	}
}

func gitRepoWithFixture(t *testing.T) (anchor.Repo, string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	dir := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		base := []string{"-C", dir, "-c", "user.name=t", "-c", "user.email=t@t", "-c", "commit.gpgsign=false"}
		if out, err := exec.Command("git", append(base, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	git("init", "-q", "-b", "main")
	wu := filepath.Join(dir, "festivals", "dashboard-DA0001")
	if err := os.CopyFS(wu, os.DirFS(fixture("dashboard-DA0001-baseline"))); err != nil {
		t.Fatal(err)
	}
	git("add", "-A")
	git("commit", "-q", "-m", "fixture")
	repo, err := anchor.OpenRepo(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	return repo, wu
}
