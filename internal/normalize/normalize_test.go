package normalize

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeErrors(t *testing.T) {
	t.Parallel()

	t.Run("cancelled context writes nothing", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		dst := filepath.Join(t.TempDir(), "out")
		_, err := Normalize(ctx, fixture("dashboard-DA0001-baseline"), dst, V1())
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("err = %v, want context.Canceled", err)
		}
		if _, statErr := os.Stat(dst); !errors.Is(statErr, os.ErrNotExist) {
			t.Fatalf("dst was created despite cancellation")
		}
	})

	t.Run("non-empty destination is refused", func(t *testing.T) {
		t.Parallel()
		dst := t.TempDir()
		if err := os.WriteFile(filepath.Join(dst, "x"), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		_, err := Normalize(context.Background(), fixture("dashboard-DA0001-baseline"), dst, V1())
		if !errors.Is(err, ErrDestNotEmpty) {
			t.Fatalf("err = %v, want ErrDestNotEmpty", err)
		}
	})

	t.Run("source must be a directory", func(t *testing.T) {
		t.Parallel()
		_, err := Normalize(context.Background(), filepath.Join(fixture("dashboard-DA0001-baseline"), "fest.yaml"), filepath.Join(t.TempDir(), "out"), V1())
		if !errors.Is(err, ErrNotADirectory) {
			t.Fatalf("err = %v, want ErrNotADirectory", err)
		}
	})

	t.Run("symlink is rejected", func(t *testing.T) {
		t.Parallel()
		src := copyTree(t, fixture("dashboard-DA0001-baseline"))
		if err := os.Symlink("FESTIVAL_GOAL.md", filepath.Join(src, "link.md")); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
		_, err := Normalize(context.Background(), src, filepath.Join(t.TempDir(), "out"), V1())
		if !errors.Is(err, ErrSymlinkRejected) {
			t.Fatalf("err = %v, want ErrSymlinkRejected", err)
		}
	})

	t.Run("unknown field names the file", func(t *testing.T) {
		t.Parallel()
		src := copyTree(t, fixture("dashboard-DA0001-baseline"))
		rel := filepath.Join("001_IMPLEMENT", "01_data_layer", "02_design_data_layer.md")
		injectUnknownField(t, filepath.Join(src, rel))
		_, err := Normalize(context.Background(), src, filepath.Join(t.TempDir(), "out"), V1())
		if !errors.Is(err, ErrUnknownField) {
			t.Fatalf("err = %v, want ErrUnknownField", err)
		}
		if !strings.Contains(err.Error(), rel) || !strings.Contains(err.Error(), "fest_bogus") {
			t.Fatalf("error does not name file and key: %v", err)
		}
	})
}

func TestNormalizeBaseline(t *testing.T) {
	t.Parallel()
	src := fixture("dashboard-DA0001-baseline")
	dst := filepath.Join(t.TempDir(), "out")
	r, err := Normalize(context.Background(), src, dst, V1())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dst, ".fest")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal(".fest/ was copied")
	}
	if !r.StatusHistoryStripped {
		t.Fatal("status_history not stripped")
	}
	if want := countLines(t, src, "fest_status:", "fest_updated:", "fest_working_dir:"); r.FieldsStripped != want {
		t.Fatalf("FieldsStripped = %d, want %d", r.FieldsStripped, want)
	}
	if got := countLines(t, dst, "fest_status:", "fest_updated:", "fest_working_dir:"); got != 0 {
		t.Fatalf("%d state lines survived in output", got)
	}
	if r.Files == 0 || r.Excluded == 0 {
		t.Fatalf("implausible report: %+v", r)
	}
	out, err := os.ReadFile(filepath.Join(dst, "fest.yaml"))
	if err != nil || strings.Contains(string(out), "status_history") {
		t.Fatalf("fest.yaml still has status_history (err=%v)", err)
	}
}

func TestNormalizeStableAcrossExecution(t *testing.T) {
	t.Parallel()
	base := filepath.Join(t.TempDir(), "base")
	mut := filepath.Join(t.TempDir(), "mut")
	if _, err := Normalize(context.Background(), fixture("dashboard-DA0001-baseline"), base, V1()); err != nil {
		t.Fatal(err)
	}
	if _, err := Normalize(context.Background(), fixture("dashboard-DA0001-mutated"), mut, V1()); err != nil {
		t.Fatal(err)
	}
	a, b := treeDigest(t, base), treeDigest(t, mut)
	if len(a) != len(b) {
		t.Fatalf("file counts differ: %d vs %d", len(a), len(b))
	}
	for rel, h := range a {
		if b[rel] != h {
			t.Errorf("%s differs after execution", rel)
		}
	}
}

func TestNormalizeDeterministic(t *testing.T) {
	t.Parallel()
	one := filepath.Join(t.TempDir(), "one")
	two := filepath.Join(t.TempDir(), "two")
	for _, d := range []string{one, two} {
		if _, err := Normalize(context.Background(), fixture("dashboard-DA0001-baseline"), d, V1()); err != nil {
			t.Fatal(err)
		}
	}
	a, b := treeDigest(t, one), treeDigest(t, two)
	for rel, h := range a {
		if b[rel] != h {
			t.Errorf("%s not deterministic", rel)
		}
	}
}

// TestNormalizeFestRewriteIsStateOnly reproduces what fest does to a task doc
// on completion (observed 2026-08-21): status flip, fest_updated added, the
// hooks block re-serialized from flow to block style with a blank line, and a
// changed trailing newline. None of it is direction.
func TestNormalizeFestRewriteIsStateOnly(t *testing.T) {
	t.Parallel()
	before := "---\nfest_type: task\nfest_id: 01_x.md\nfest_status: pending\nfest_tracking: true\nhooks:\n  start:\n    pre: [direction_anchor]\n---\n\n# Task\n\n- [ ] done when"
	after := "---\nfest_type: task\nfest_id: 01_x.md\nfest_status: completed\nfest_updated: 2026-08-21T04:19:23.527557-06:00\nfest_tracking: true\nhooks:\n  start:\n    pre:\n      - direction_anchor\n\n---\n\n# Task\n\n- [ ] done when\n"
	withComment := "---\n# written by a template\nfest_tracking: true\nhooks: {start: {pre: [direction_anchor]}}\nfest_id: 01_x.md\nfest_type: task\n---\n\n# Task\n\n- [ ] done when\n\n\n"
	extraSeparator := "---\nfest_type: task\nfest_id: 01_x.md\nfest_tracking: true\nhooks:\n  start:\n    pre:\n      - direction_anchor\n\n---\n\n\n# Task\n\n- [ ] done when\n"
	var outs [][]byte
	for _, doc := range []string{before, after, withComment, extraSeparator} {
		out, _, err := filterFrontmatter([]byte(doc), V1())
		if err != nil {
			t.Fatal(err)
		}
		outs = append(outs, trimEOF(out))
	}
	for i := 1; i < len(outs); i++ {
		if string(outs[i]) != string(outs[0]) {
			t.Fatalf("variant %d normalizes differently:\n%s\n---\n%s", i, outs[0], outs[i])
		}
	}
	if !strings.Contains(string(outs[0]), "hooks:\n  start:\n    pre:\n      - direction_anchor\n") {
		t.Fatalf("canonical form unexpected:\n%s", outs[0])
	}
}
