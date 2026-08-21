package normalize

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestStripStatusHistoryErrors(t *testing.T) {
	t.Parallel()
	for _, doc := range []string{"- a\n- b\n", "just a scalar\n", ""} {
		if _, _, err := stripStatusHistory([]byte(doc)); !errors.Is(err, ErrInvalidFestYAML) {
			t.Errorf("%q: err = %v, want ErrInvalidFestYAML", doc, err)
		}
	}
}

func TestStripStatusHistoryFixture(t *testing.T) {
	t.Parallel()
	b, err := os.ReadFile(filepath.Join(fixture("dashboard-DA0001-baseline"), "fest.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	out, removed, err := stripStatusHistory(b)
	if err != nil {
		t.Fatal(err)
	}
	if !removed {
		t.Fatal("status_history not found in fixture fest.yaml")
	}
	s := string(out)
	if strings.Contains(s, "status_history") {
		t.Fatalf("status_history survived:\n%s", s)
	}
	for _, want := range []string{"goal:", "quality_gates:", "type_config:"} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q", want)
		}
	}
	var reparsed map[string]any
	if err := yaml.Unmarshal(out, &reparsed); err != nil {
		t.Fatalf("output is not valid YAML: %v", err)
	}
	again, removed2, err := stripStatusHistory(out)
	if err != nil || removed2 || string(again) != s {
		t.Fatalf("second pass not idempotent (removed=%v err=%v)", removed2, err)
	}
}

func TestStripStatusHistoryWithoutMetadata(t *testing.T) {
	t.Parallel()
	_, removed, err := stripStatusHistory([]byte("version: \"1.0\"\n"))
	if err != nil || removed {
		t.Fatalf("removed=%v err=%v, want false/nil", removed, err)
	}
}

func TestResetCheckboxes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		in    string
		want  string
		wantN int
	}{
		{name: "nothing to reset", in: "- [ ] open\ntext\n", want: "- [ ] open\ntext\n"},
		{name: "lower and upper x", in: "- [x] a\n- [X] b\n", want: "- [ ] a\n- [ ] b\n", wantN: 2},
		{name: "star bullets and nesting", in: "* [x] a\n  - [x] nested\n", want: "* [ ] a\n  - [ ] nested\n", wantN: 2},
		{name: "inline text untouched", in: "- [x] keep [x] this\n", want: "- [ ] keep [x] this\n", wantN: 1},
		{name: "not a list item", in: "[x] bare\n", want: "[x] bare\n"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, n := resetCheckboxes([]byte(tc.in))
			if string(got) != tc.want || n != tc.wantN {
				t.Fatalf("got %q (n=%d), want %q (n=%d)", got, n, tc.want, tc.wantN)
			}
		})
	}
}

func TestExcluded(t *testing.T) {
	t.Parallel()
	for name, want := range map[string]bool{".fest": true, ".bundles": true, ".git": true, ".env": true, "fest.yaml": false, "TODO.md": false, ".envrc": false} {
		if got := excluded(name); got != want {
			t.Errorf("excluded(%q) = %v, want %v", name, got, want)
		}
	}
}
