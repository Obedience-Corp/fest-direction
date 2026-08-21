package attest

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Obedience-Corp/fest-direction/internal/direction"
	"github.com/Obedience-Corp/fest-direction/internal/normalize"
)

var update = flag.Bool("update", false, "rewrite golden files")

func TestParseStatementErrors(t *testing.T) {
	t.Parallel()
	good, _, err := Attest(context.Background(), fixture("dashboard-DA0001-baseline"), AttestOptions{Tool: testTool, Now: fixedClock})
	if err != nil {
		t.Fatal(err)
	}
	okJSON, _ := Marshal(good)
	tests := []struct {
		name string
		doc  string
	}{
		{name: "not json", doc: "{"},
		{name: "wrong type", doc: strings.Replace(string(okJSON), StatementType, "https://in-toto.io/Statement/v0.1", 1)},
		{name: "no subject", doc: strings.Replace(string(okJSON), `"subject": [`, `"subject": [], "x": [`, 1)},
		{name: "prefixed digest", doc: strings.Replace(string(okJSON), `"sha256": "`, `"sha256": "sha256:`, 1)},
		{name: "empty predicate type", doc: strings.Replace(string(okJSON), PredicateTypeV1, "", 1)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if _, err := ParseStatement([]byte(tc.doc)); !errors.Is(err, ErrMalformedStatement) {
				t.Fatalf("err = %v, want ErrMalformedStatement", err)
			}
		})
	}
}

func TestStatementGolden(t *testing.T) {
	t.Parallel()
	stmt, res, err := Attest(context.Background(), fixture("dashboard-DA0001-baseline"), AttestOptions{Tool: testTool, Now: fixedClock})
	if err != nil {
		t.Fatal(err)
	}
	// Source is machine-specific; pin it for the golden.
	rec, err := ParseRecord(stmt)
	if err != nil {
		t.Fatal(err)
	}
	rec.Source = "testdata/fixtures/dashboard-DA0001-baseline"
	stmt, err = NewStatement(rec, "")
	if err != nil {
		t.Fatal(err)
	}
	got, err := Marshal(stmt)
	if err != nil {
		t.Fatal(err)
	}
	golden := filepath.Join("testdata", "statement_golden.json")
	if *update {
		if err := os.WriteFile(golden, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("golden missing (run with -update once): %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("statement differs from golden:\n%s", got)
	}
	back, err := ParseStatement(got)
	if err != nil {
		t.Fatal(err)
	}
	again, _ := Marshal(back)
	if !bytes.Equal(again, got) {
		t.Fatal("marshal → parse → marshal is not byte-stable")
	}
	if back.Subject[0].Name != "DA0001.festival" || "sha256:"+back.Subject[0].Digest["sha256"] != res.SnapshotID {
		t.Fatalf("subject = %+v, snapshot %s", back.Subject[0], res.SnapshotID)
	}
}

func TestRecordRoundTrip(t *testing.T) {
	t.Parallel()
	res := direction.Result{DirectionHash: "sha256:" + strings.Repeat("a", 64), SnapshotID: "sha256:" + strings.Repeat("b", 64), NormalizationVersion: normalize.Version, Kind: "note"}
	rec := RecordFromResult(res, []AnchorRef{{Type: "git-commit", Ref: "abc"}}, testTool, fixedClock())
	stmt, err := NewStatement(rec, "")
	if err != nil {
		t.Fatal(err)
	}
	if stmt.Subject[0].Name != "note.festival" {
		t.Fatalf("default name = %q", stmt.Subject[0].Name)
	}
	back, err := ParseRecord(stmt)
	if err != nil || back.DirectionHash != rec.DirectionHash || len(back.Anchors) != 1 || back.CreatedAt != "2026-08-21T12:00:00Z" {
		t.Fatalf("round trip: %+v, %v", back, err)
	}
	bad := rec
	bad.DirectionHash = "sha256:short"
	if _, err := NewStatement(bad, ""); !errors.Is(err, ErrInvalidHash) {
		t.Fatalf("invalid hash: %v", err)
	}
	foreign := stmt
	foreign.PredicateType = "https://slsa.dev/provenance/v1"
	if _, err := ParseRecord(foreign); !errors.Is(err, ErrPredicateType) {
		t.Fatalf("foreign predicate: %v", err)
	}
}
