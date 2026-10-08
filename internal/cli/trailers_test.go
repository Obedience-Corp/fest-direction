package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Obedience-Corp/fest-direction/internal/anchor"
	"github.com/Obedience-Corp/fest-direction/internal/direction"
	"github.com/Obedience-Corp/fest-direction/internal/normalize"
)

func runIO(ctx context.Context, stdin string, args ...string) (stdout, stderr string, err error) {
	root := NewRootCommand()
	var out, errb bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errb)
	root.SetIn(strings.NewReader(stdin))
	root.SetArgs(args)
	err = root.ExecuteContext(ctx)
	return out.String(), errb.String(), err
}

func TestTrailersCommand(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	repo := t.TempDir()
	git := func(args ...string) string {
		t.Helper()
		base := []string{"-C", repo, "-c", "user.name=t", "-c", "user.email=t@t", "-c", "commit.gpgsign=false"}
		out, err := exec.CommandContext(t.Context(), "git", append(base, args...)...).CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return string(out)
	}
	git("init", "-q", "-b", "main")
	const rel = "festivals/dashboard-DA0001"
	wu := filepath.Join(repo, rel)
	if err := os.CopyFS(wu, os.DirFS(fixture("dashboard-DA0001-baseline"))); err != nil {
		t.Fatal(err)
	}
	git("add", "-A")
	git("commit", "-q", "-m", "fixture")
	tree := strings.TrimSpace(git("rev-parse", "HEAD^{tree}"))

	want, err := direction.Hash(t.Context(), wu, normalize.Current())
	if err != nil {
		t.Fatal(err)
	}
	goal := filepath.Join(wu, "FESTIVAL_GOAL.md")
	body, err := os.ReadFile(goal)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(goal, append(body, []byte("\nStaged after the tree was captured.\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	git("add", "-A")
	dirty, err := direction.Hash(t.Context(), wu, normalize.Current())
	if err != nil {
		t.Fatal(err)
	}
	if dirty.DirectionHash == want.DirectionHash {
		t.Fatal("staged edit did not change the direction hash; the test would not prove which tree was hashed")
	}

	plain := "[FE-DA0001] subject\n\nbody\n"
	already := plain + "\n" + anchor.FormatTrailers(want)
	const badTree = "0123456789abcdef0123456789abcdef01234567"
	cancel := func(ctx context.Context) context.Context {
		c, stop := context.WithCancel(ctx)
		stop()
		return c
	}
	keep := func(ctx context.Context) context.Context { return ctx }

	tests := []struct {
		name       string
		ctx        func(context.Context) context.Context
		tree       string
		workUnit   string
		stdin      string
		wantErr    bool
		wantIs     error
		wantStderr bool
		wantExact  string
		wantHash   bool
	}{
		{
			name:     "cancelled context",
			ctx:      cancel,
			tree:     tree,
			workUnit: rel,
			stdin:    "do-not-leak\n",
			wantErr:  true,
			wantIs:   context.Canceled,
		},
		{
			name:       "bad tree writes nothing",
			ctx:        keep,
			tree:       badTree,
			workUnit:   rel,
			stdin:      "do-not-leak\n",
			wantErr:    true,
			wantStderr: true,
		},
		{
			name:      "no work unit copies stdin",
			ctx:       keep,
			tree:      badTree,
			stdin:     "subject only, no trailers\n",
			wantExact: "subject only, no trailers\n",
		},
		{
			name:     "known tree gets trailers",
			ctx:      keep,
			tree:     tree,
			workUnit: rel,
			stdin:    plain,
			wantHash: true,
		},
		{
			name:      "existing trailers are not duplicated",
			ctx:       keep,
			tree:      tree,
			workUnit:  rel,
			stdin:     already,
			wantExact: already,
			wantHash:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Chdir(repo)
			t.Setenv("DIRECTION_WORK_UNIT", "")
			args := []string{"--no-color", "trailers", "--tree", tc.tree}
			if tc.workUnit != "" {
				args = append(args, "--work-unit", tc.workUnit)
			}
			stdout, stderr, err := runIO(tc.ctx(t.Context()), tc.stdin, args...)
			if tc.wantErr != (err != nil) {
				t.Fatalf("err = %v, wantErr %v\nstdout: %q\nstderr: %q", err, tc.wantErr, stdout, stderr)
			}
			if tc.wantIs != nil && !errors.Is(err, tc.wantIs) {
				t.Fatalf("err = %v, want %v", err, tc.wantIs)
			}
			if tc.wantErr {
				if stdout != "" {
					t.Fatalf("stdout = %q, want empty so a failed hash is not a partial message", stdout)
				}
				if tc.wantStderr && !strings.Contains(stderr, "✗") {
					t.Fatalf("stderr = %q, want the error", stderr)
				}
				return
			}
			if stderr != "" {
				t.Fatalf("stderr = %q, want empty", stderr)
			}
			if tc.wantExact != "" && stdout != tc.wantExact {
				t.Fatalf("stdout = %q, want %q", stdout, tc.wantExact)
			}
			if !tc.wantHash {
				if strings.Contains(stdout, anchor.TrailerDirection) {
					t.Fatalf("stdout gained trailers:\n%s", stdout)
				}
				return
			}
			if strings.Count(stdout, anchor.TrailerDirection+":") != 1 || strings.Count(stdout, anchor.TrailerNormalization+":") != 1 {
				t.Fatalf("trailers duplicated or missing:\n%s", stdout)
			}
			got, err := anchor.ParseTrailers(stdout)
			if err != nil {
				t.Fatalf("parse: %v\n%s", err, stdout)
			}
			if got.DirectionHash != want.DirectionHash || got.NormalizationVersion != want.NormalizationVersion {
				t.Fatalf("trailer %+v, want hash of tree %s (%s) not the staged tree %s", got, tree, want.DirectionHash, dirty.DirectionHash)
			}
			if !strings.Contains(stdout, "[FE-DA0001] subject") {
				t.Fatalf("subject lost:\n%s", stdout)
			}
		})
	}
}
