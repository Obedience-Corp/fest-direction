package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Obedience-Corp/fest-direction/internal/anchor"
	"github.com/Obedience-Corp/fest-direction/internal/direction"
)

func fixture(name string) string {
	return filepath.Join("..", "..", "testdata", "fixtures", name)
}

func run(ctx context.Context, args ...string) (string, error) {
	root := NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(args)
	err := root.ExecuteContext(ctx)
	return out.String(), err
}

func TestRootCommand(t *testing.T) {
	t.Parallel()

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()

	tests := []struct {
		name    string
		ctx     context.Context
		args    []string
		wantErr bool
		wantIs  error
		wantOut string
	}{
		{name: "cancelled context is refused before any verb runs", ctx: cancelled, args: []string{"version"}, wantErr: true, wantIs: context.Canceled},
		{name: "unknown verb is an error", ctx: context.Background(), args: []string{"nope"}, wantErr: true},
		{name: "version prints build metadata", ctx: context.Background(), args: []string{"--no-color", "version"}, wantOut: "dev"},
		{name: "help lists verbs", ctx: context.Background(), args: []string{"--help"}, wantOut: "version"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out, err := run(tc.ctx, tc.args...)
			if tc.wantErr != (err != nil) {
				t.Fatalf("err = %v, wantErr %v (output: %q)", err, tc.wantErr, out)
			}
			if tc.wantIs != nil && !errors.Is(err, tc.wantIs) {
				t.Fatalf("err = %v, want errors.Is %v", err, tc.wantIs)
			}
			if tc.wantOut != "" && !strings.Contains(out, tc.wantOut) {
				t.Fatalf("output %q missing %q", out, tc.wantOut)
			}
		})
	}
}

func TestHashCommand(t *testing.T) {
	t.Parallel()
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()

	tests := []struct {
		name    string
		ctx     context.Context
		args    []string
		wantErr bool
		wantIs  error
		wantOut []string
	}{
		{name: "cancelled context", ctx: cancelled, args: []string{"hash", fixture("dashboard-DA0001-baseline")}, wantErr: true, wantIs: context.Canceled},
		{name: "missing path is reported once", ctx: context.Background(), args: []string{"--no-color", "hash", "/nonexistent/work-unit"}, wantErr: true, wantOut: []string{"✗"}},
		{name: "baseline prints fields", ctx: context.Background(), args: []string{"--no-color", "hash", fixture("dashboard-DA0001-baseline")}, wantOut: []string{"direction", "sha256:", "normalization", "v2", "DA0001"}},
		{name: "note prints dash subject", ctx: context.Background(), args: []string{"--no-color", "hash", fixture("workitem-note")}, wantOut: []string{"note"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out, err := run(tc.ctx, tc.args...)
			if tc.wantErr != (err != nil) {
				t.Fatalf("err = %v, wantErr %v (output %q)", err, tc.wantErr, out)
			}
			if tc.wantIs != nil && !errors.Is(err, tc.wantIs) {
				t.Fatalf("err = %v, want %v", err, tc.wantIs)
			}
			for _, w := range tc.wantOut {
				if !strings.Contains(out, w) {
					t.Errorf("output missing %q:\n%s", w, out)
				}
			}
		})
	}
}

func TestHashCommandJSON(t *testing.T) {
	t.Parallel()
	out, err := run(context.Background(), "hash", "--json", fixture("dashboard-DA0001-baseline"))
	if err != nil {
		t.Fatal(err)
	}
	var res direction.Result
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, out)
	}
	if res.NormalizationVersion != 2 || !strings.HasPrefix(res.DirectionHash, "sha256:") || res.Subject == nil || res.Subject.ID != "DA0001" {
		t.Fatalf("unexpected result: %+v", res)
	}
}

func TestAnchorCommand(t *testing.T) {
	t.Parallel()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	repo := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		base := []string{"-C", repo, "-c", "user.name=t", "-c", "user.email=t@t", "-c", "commit.gpgsign=false"}
		if out, err := exec.Command("git", append(base, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	git("init", "-q", "-b", "main")
	wu := filepath.Join(repo, "dashboard-DA0001")
	if err := os.CopyFS(wu, os.DirFS(fixture("dashboard-DA0001-baseline"))); err != nil {
		t.Fatal(err)
	}
	git("add", "-A")
	git("commit", "-q", "-m", "fixture")

	out, err := run(context.Background(), "--no-color", "anchor", wu)
	if err != nil {
		t.Fatalf("anchor: %v\n%s", err, out)
	}
	for _, want := range []string{"direction", "head", "record", ".direction/anchors/sha256-", "commit "} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	out, err = run(context.Background(), "anchor", "--json", wu)
	if err != nil {
		t.Fatalf("anchor --json: %v", err)
	}
	var rec anchor.Record
	if err := json.Unmarshal([]byte(out), &rec); err != nil || len(rec.Events) != 2 {
		t.Fatalf("json record: %v (%d events)\n%s", err, len(rec.Events), out)
	}

	if err := os.WriteFile(filepath.Join(wu, "FESTIVAL_GOAL.md"), []byte("edited\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err = run(context.Background(), "--no-color", "anchor", wu)
	if err == nil || !strings.Contains(out, "✗") || !strings.Contains(out, "--force") {
		t.Fatalf("dirty anchor should fail with a hint: err=%v\n%s", err, out)
	}
	if _, err := run(context.Background(), "--no-color", "anchor", t.TempDir()); err == nil {
		t.Fatal("anchoring outside a repo should fail")
	}
}

func TestAttestAndVerifyCommands(t *testing.T) {
	t.Parallel()
	dir := fixture("dashboard-DA0001-baseline")
	stmt := filepath.Join(t.TempDir(), "s.intoto.json")
	out, err := run(context.Background(), "--no-color", "attest", dir, "-o", stmt)
	if err != nil || !strings.Contains(out, "written") || !strings.Contains(out, "DA0001.festival") {
		t.Fatalf("attest: %v\n%s", err, out)
	}
	if _, err := run(context.Background(), "--no-color", "attest", dir, "-o", stmt); err == nil {
		t.Fatal("second attest without --force should refuse")
	}
	out, err = run(context.Background(), "--no-color", "verify", dir, "--statement", stmt)
	if err != nil || !strings.Contains(out, "✓ verified") {
		t.Fatalf("verify: %v\n%s", err, out)
	}
	if _, err := run(context.Background(), "--no-color", "verify", dir); err == nil {
		t.Fatal("verify without --statement should fail")
	}
	// State-only change on a copy: snapshot moves, direction intact.
	cp := filepath.Join(t.TempDir(), "cp")
	if err := os.CopyFS(cp, os.DirFS(dir)); err != nil {
		t.Fatal(err)
	}
	task := filepath.Join(cp, "001_IMPLEMENT", "01_data_layer", "01_link_project.md")
	b, _ := os.ReadFile(task)
	if err := os.WriteFile(task, []byte(strings.Replace(string(b), "fest_status: pending", "fest_status: completed", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err = run(context.Background(), "--no-color", "verify", cp, "--statement", stmt)
	if err == nil || !strings.Contains(out, "✗ snapshot") || !strings.Contains(out, "direction intact") {
		t.Fatalf("state-only verify: err=%v\n%s", err, out)
	}
	out, err = run(context.Background(), "verify", cp, "--statement", stmt, "--json")
	if err == nil || !strings.Contains(out, `"state_only": true`) {
		t.Fatalf("json report: err=%v\n%s", err, out)
	}
}
