package cli

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Obedience-Corp/fest-direction/internal/anchor"
)

// TestHookEndToEnd builds the binary, installs the shim in a temp repository,
// makes a real git commit, and reads the trailers back with git itself.
func TestHookEndToEnd(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	bin := filepath.Join(t.TempDir(), "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	build := exec.Command("go", "build", "-o", filepath.Join(bin, "fest-direction"), "../../cmd/fest-direction")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	repo := t.TempDir()
	git := func(args ...string) string {
		t.Helper()
		base := []string{"-C", repo, "-c", "user.name=t", "-c", "user.email=t@t", "-c", "commit.gpgsign=false"}
		out, err := exec.Command("git", append(base, args...)...).CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return string(out)
	}
	git("init", "-q", "-b", "main")
	wu := filepath.Join(repo, "festivals", "dashboard-DA0001")
	if err := os.CopyFS(wu, os.DirFS(fixture("dashboard-DA0001-baseline"))); err != nil {
		t.Fatal(err)
	}
	git("add", "-A")
	git("commit", "-q", "-m", "fixture")

	out, err := run(context.Background(), "--no-color", "hook", "install", "--repo", repo, "--work-unit", "festivals/dashboard-DA0001")
	if err != nil {
		t.Fatalf("install: %v\n%s", err, out)
	}
	if !strings.Contains(out, "installed") {
		t.Fatalf("unexpected install output:\n%s", out)
	}

	// A real commit that changes the plan: the trailer must describe the committed tree.
	goal := filepath.Join(wu, "FESTIVAL_GOAL.md")
	b, _ := os.ReadFile(goal)
	if err := os.WriteFile(goal, append(b, []byte("\nRevised goal.\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	git("add", "-A")
	git("commit", "-q", "-m", "[FE-DA0001] revise goal")
	body := git("log", "-1", "--format=%B")
	tr, err := anchor.ParseTrailers(body)
	if err != nil {
		t.Fatalf("commit lacks trailers: %v\n%s", err, body)
	}
	want, err := run(context.Background(), "hash", "--json", wu)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(want, tr.DirectionHash) {
		t.Fatalf("trailer %s does not match direction hash of the committed tree:\n%s", tr.DirectionHash, want)
	}
	// A commit outside the work unit still states the plan in force.
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git("add", "-A")
	git("commit", "-q", "-m", "docs: readme")
	if _, err := anchor.ParseTrailers(git("log", "-1", "--format=%B")); err != nil {
		t.Fatalf("unrelated commit lacks trailers: %v", err)
	}
	// A whole-directory move of the configured work unit must still commit,
	// and the committed config must name the new path.
	if err := os.MkdirAll(filepath.Join(repo, "festivals", ".dungeon"), 0o755); err != nil {
		t.Fatal(err)
	}
	git("mv", "festivals/dashboard-DA0001", "festivals/.dungeon/dashboard-DA0001")
	git("commit", "-q", "-m", "dungeon festival")
	if _, err := anchor.ParseTrailers(git("log", "-1", "--format=%B")); err != nil {
		t.Fatalf("move commit lacks trailers: %v", err)
	}
	cfg := git("show", "HEAD:.direction/config.yaml")
	if !strings.Contains(cfg, "default_work_unit: festivals/.dungeon/dashboard-DA0001\n") {
		t.Fatalf("committed config did not follow the move:\n%s", cfg)
	}
	if status := git("status", "--short"); status != "" {
		t.Fatalf("move commit left the tree dirty:\n%s", status)
	}
	// Uninstall: later commits carry none.
	if _, err := run(context.Background(), "--no-color", "hook", "uninstall", "--repo", repo); err != nil {
		t.Fatal(err)
	}
	git("commit", "-q", "--allow-empty", "-m", "after uninstall")
	if _, err := anchor.ParseTrailers(git("log", "-1", "--format=%B")); err == nil {
		t.Fatal("commit after uninstall still carries trailers")
	}
}
