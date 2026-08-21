package normalize

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFilterFrontmatterErrors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		doc    string
		wantIs error
		wantIn string
	}{
		{name: "unknown fest_ key fails closed", doc: "---\nfest_type: task\nfest_bogus: 1\n---\nbody\n", wantIs: ErrUnknownField, wantIn: "fest_bogus"},
		{name: "list root is invalid", doc: "---\n- a\n- b\n---\nbody\n", wantIs: ErrInvalidFrontmatter},
		{name: "scalar root is invalid", doc: "---\njust text\n---\nbody\n", wantIs: ErrInvalidFrontmatter},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, _, err := filterFrontmatter([]byte(tc.doc), V1())
			if !errors.Is(err, tc.wantIs) {
				t.Fatalf("err = %v, want %v", err, tc.wantIs)
			}
			if tc.wantIn != "" && !strings.Contains(err.Error(), tc.wantIn) {
				t.Fatalf("err %q does not name %q", err, tc.wantIn)
			}
		})
	}
}

func TestFilterFrontmatter(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		doc           string
		wantStripped  int
		wantContains  []string
		wantAbsent    []string
		wantUnchanged bool
	}{
		{name: "no frontmatter is untouched", doc: "# Title\n\nfest_status: not frontmatter\n", wantUnchanged: true},
		{name: "unterminated fence is not frontmatter", doc: "---\nfest_status: pending\nbody", wantUnchanged: true},
		{name: "state keys stripped, direction and body kept", doc: "---\nfest_type: task\nfest_status: pending\nfest_updated: 2026-01-01\nfest_working_dir: projects/x\n---\n\n# Task\n\n- [x] done\n", wantStripped: 3, wantContains: []string{"fest_type: task", "\n---\n\n# Task\n\n- [x] done\n"}, wantAbsent: []string{"fest_status", "fest_updated", "fest_working_dir"}},
		{name: "template metadata survives, comments are canonicalized away", doc: "---\n# Template metadata\nid: QUALITY_GATE_TESTING\naliases:\n  - testing-verify\nfest_type: gate\nfest_status: pending\n---\n\n# Task: Testing\n", wantStripped: 1, wantContains: []string{"id: QUALITY_GATE_TESTING", "- testing-verify", "fest_type: gate"}, wantAbsent: []string{"fest_status", "# Template metadata"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out, n, err := filterFrontmatter([]byte(tc.doc), V1())
			if err != nil {
				t.Fatal(err)
			}
			if tc.wantUnchanged {
				if !bytes.Equal(out, []byte(tc.doc)) || n != 0 {
					t.Fatalf("expected unchanged, got %q (n=%d)", out, n)
				}
				return
			}
			if n != tc.wantStripped {
				t.Fatalf("stripped = %d, want %d", n, tc.wantStripped)
			}
			for _, s := range tc.wantContains {
				if !strings.Contains(string(out), s) {
					t.Errorf("output missing %q:\n%s", s, out)
				}
			}
			for _, s := range tc.wantAbsent {
				if strings.Contains(string(out), s) {
					t.Errorf("output still has %q:\n%s", s, out)
				}
			}
			again, n2, err := filterFrontmatter(out, V1())
			if err != nil || n2 != 0 || !bytes.Equal(again, out) {
				t.Fatalf("second pass changed output (n=%d, err=%v)", n2, err)
			}
		})
	}
}

func TestFilterFrontmatterFixtureBodyPreserved(t *testing.T) {
	t.Parallel()
	file := filepath.Join(fixture("dashboard-DA0001-baseline"), "001_IMPLEMENT", "01_data_layer", "01_link_project.md")
	doc, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	out, n, err := filterFrontmatter(doc, V1())
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("stripped = %d, want 1 (fest_status)", n)
	}
	_, wantBody, _ := splitFrontmatter(doc)
	_, gotBody, _ := splitFrontmatter(out)
	if !bytes.Equal(wantBody, gotBody) {
		t.Fatalf("body changed")
	}
	if strings.Contains(string(out), "fest_status:") {
		t.Fatalf("fest_status survived:\n%s", out)
	}
}

func TestFilterFrontmatterEveryFixtureDoc(t *testing.T) {
	t.Parallel()
	for _, fx := range []string{"dashboard-DA0001-baseline", "dashboard-DA0001-mutated"} {
		err := filepath.WalkDir(fixture(fx), func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || filepath.Ext(path) != ".md" {
				return err
			}
			doc, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if _, _, err := filterFrontmatter(doc, V1()); err != nil {
				t.Errorf("%s: %v", path, err)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}
