package direction

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func fixture(name string) string {
	return filepath.Join("..", "..", "testdata", "fixtures", name)
}

func copyTree(t *testing.T, src string) string {
	t.Helper()
	dst := filepath.Join(t.TempDir(), filepath.Base(src))
	if err := os.CopyFS(dst, os.DirFS(src)); err != nil {
		t.Fatalf("copy %s: %v", src, err)
	}
	return dst
}

var pocCheckbox = regexp.MustCompile(`(?m)^- \[[xX]\]`)

// applyPoCRules reproduces the design's line-wise proof-of-concept exactly:
// drop .fest/, delete ^fest_status: and ^fest_updated: lines from every *.md,
// reset top-level TODO.md checkboxes. It is deliberately not the v1 rule set.
func applyPoCRules(t *testing.T, dir string) {
	t.Helper()
	if err := os.RemoveAll(filepath.Join(dir, ".fest")); err != nil {
		t.Fatal(err)
	}
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Ext(path) != ".md" {
			return err
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var kept []string
		for _, line := range strings.SplitAfter(string(b), "\n") {
			if strings.HasPrefix(line, "fest_status:") || strings.HasPrefix(line, "fest_updated:") {
				continue
			}
			kept = append(kept, line)
		}
		out := []byte(strings.Join(kept, ""))
		if filepath.Base(path) == "TODO.md" {
			out = pocCheckbox.ReplaceAll(out, []byte("- [ ]"))
		}
		if !bytes.Equal(out, b) {
			return os.WriteFile(path, out, 0o644)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func injectUnknownField(t *testing.T, file string) {
	t.Helper()
	b, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	s := strings.Replace(string(b), "fest_tracking: true\n", "fest_tracking: true\nfest_bogus: 1\n", 1)
	if s == string(b) {
		t.Fatalf("marker not found in %s", file)
	}
	if err := os.WriteFile(file, []byte(s), 0o644); err != nil {
		t.Fatal(err)
	}
}
