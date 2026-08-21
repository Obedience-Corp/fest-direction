package attest

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

func failedChecks(r Report) []string {
	var out []string
	for _, c := range r.Checks {
		if !c.OK {
			out = append(out, c.Name)
		}
	}
	return out
}

func TestVerifyStatementErrors(t *testing.T) {
	t.Parallel()
	dir := fixture("dashboard-DA0001-baseline")
	good := attestDir(t, dir)
	goodJSON, _ := os.ReadFile(good)
	write := func(t *testing.T, content string) string {
		t.Helper()
		p := filepath.Join(t.TempDir(), "s.json")
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	tests := []struct {
		name   string
		stmt   string
		wantIs error
		check  string
	}{
		{name: "malformed", stmt: write(t, "not json"), wantIs: ErrMalformedStatement, check: "statement"},
		{name: "missing file", stmt: filepath.Join(t.TempDir(), "nope.json"), wantIs: ErrMalformedStatement, check: "statement"},
		{name: "foreign predicate", stmt: write(t, strings.Replace(string(goodJSON), PredicateTypeV1, "https://slsa.dev/provenance/v1", 1)), wantIs: ErrPredicateType, check: "predicate-type"},
		{name: "unknown version", stmt: write(t, strings.Replace(string(goodJSON), `"normalization_version": 2`, `"normalization_version": 99`, 1)), wantIs: ErrVersionMismatch, check: "normalization-version"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rep, err := Verify(context.Background(), dir, tc.stmt)
			if !errors.Is(err, tc.wantIs) {
				t.Fatalf("err = %v, want %v", err, tc.wantIs)
			}
			if fc := failedChecks(rep); len(fc) != 1 || fc[0] != tc.check {
				t.Fatalf("failed checks = %v, want [%s]", fc, tc.check)
			}
			if rep.Result != nil {
				t.Fatal("hashes should not be computed after an early failure")
			}
		})
	}
}

func TestVerifyOutcomes(t *testing.T) {
	t.Parallel()
	t.Run("unmodified passes", func(t *testing.T) {
		t.Parallel()
		dir := fixture("dashboard-DA0001-baseline")
		rep, err := Verify(context.Background(), dir, attestDir(t, dir))
		if err != nil || !rep.OK || len(rep.Checks) != 5 {
			t.Fatalf("rep=%+v err=%v", rep, err)
		}
	})
	t.Run("body edit fails snapshot and direction", func(t *testing.T) {
		t.Parallel()
		dir := copyTree(t, fixture("dashboard-DA0001-baseline"))
		stmt := attestDir(t, dir)
		goal := filepath.Join(dir, "FESTIVAL_GOAL.md")
		b, _ := os.ReadFile(goal)
		if err := os.WriteFile(goal, append(b, []byte("\nEdited plan.\n")...), 0o644); err != nil {
			t.Fatal(err)
		}
		rep, err := Verify(context.Background(), dir, stmt)
		if !errors.Is(err, ErrSnapshotMismatch) {
			t.Fatalf("err = %v", err)
		}
		if fc := failedChecks(rep); strings.Join(fc, ",") != "snapshot,direction" || rep.StateOnly {
			t.Fatalf("failed = %v stateOnly=%v", fc, rep.StateOnly)
		}
	})
	t.Run("state-only edit fails snapshot, direction intact", func(t *testing.T) {
		t.Parallel()
		dir := copyTree(t, fixture("dashboard-DA0001-baseline"))
		stmt := attestDir(t, dir)
		replaceInFile(t, filepath.Join(dir, "001_IMPLEMENT", "01_data_layer", "01_link_project.md"), "fest_status: pending", "fest_status: completed")
		rep, err := Verify(context.Background(), dir, stmt)
		if !errors.Is(err, ErrSnapshotMismatch) {
			t.Fatalf("err = %v", err)
		}
		if fc := failedChecks(rep); strings.Join(fc, ",") != "snapshot" || !rep.StateOnly {
			t.Fatalf("failed = %v stateOnly=%v", fc, rep.StateOnly)
		}
	})
	t.Run("tampered direction_hash fails only direction", func(t *testing.T) {
		t.Parallel()
		dir := fixture("dashboard-DA0001-baseline")
		stmt := attestDir(t, dir)
		b, _ := os.ReadFile(stmt)
		s := string(b)
		const key = `"direction_hash": "sha256:`
		i := strings.Index(s, key)
		if i < 0 {
			t.Fatalf("key not found in statement:\n%s", s)
		}
		tampered := s[:i+len(key)] + strings.Repeat("0", 64) + s[i+len(key)+64:]
		if err := os.WriteFile(stmt, []byte(tampered), 0o644); err != nil {
			t.Fatal(err)
		}
		rep, err := Verify(context.Background(), dir, stmt)
		if !errors.Is(err, ErrDirectionMismatch) {
			t.Fatalf("err = %v", err)
		}
		if fc := failedChecks(rep); strings.Join(fc, ",") != "direction" {
			t.Fatalf("failed = %v", fc)
		}
	})
	t.Run("tampered bundle payload fails snapshot via unbundle", func(t *testing.T) {
		t.Parallel()
		dir := fixture("dashboard-DA0001-baseline")
		bundle := packFixture(t, dir)
		stmt := attestDir(t, dir)
		// Flip one byte inside the zip's payload data without touching info.json.
		b, _ := os.ReadFile(bundle)
		i := strings.Index(string(b), "Primary Goal")
		if i < 0 {
			t.Skip("bundle is compressed; plain-text payload not addressable")
		}
		b[i] = 'X'
		if err := os.WriteFile(bundle, b, 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := Verify(context.Background(), bundle, stmt); err == nil {
			t.Fatal("expected a failure on a tampered bundle")
		}
	})
	t.Run("v1 statement verifies under v1 rules", func(t *testing.T) {
		t.Parallel()
		dir := fixture("dashboard-DA0001-baseline")
		v1 := normalize.V1()
		res, err := direction.Hash(context.Background(), dir, v1)
		if err != nil {
			t.Fatal(err)
		}
		stmt, err := NewStatement(RecordFromResult(res, nil, testTool, fixedClock()), "")
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(t.TempDir(), "v1.json")
		if err := WriteStatement(path, stmt, false); err != nil {
			t.Fatal(err)
		}
		rep, err := Verify(context.Background(), dir, path)
		if err != nil || !rep.OK || rep.Result.NormalizationVersion != 1 {
			t.Fatalf("v1 verify: rep=%+v err=%v", rep, err)
		}
	})
	t.Run("cancelled", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if _, err := Verify(ctx, fixture("workitem-note"), "x"); !errors.Is(err, context.Canceled) {
			t.Fatalf("err = %v", err)
		}
	})
}
