package anchor

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Obedience-Corp/fest-direction/internal/direction"
)

var sampleResult = direction.Result{DirectionHash: testHash, NormalizationVersion: 2}

func TestParseTrailersErrors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		msg    string
		wantIs error
	}{
		{name: "no trailer block", msg: "[FE-DA0004] subject\n\nbody text\n", wantIs: ErrNoTrailer},
		{name: "other trailers only", msg: "subject\n\nSigned-off-by: a <a@b>\n", wantIs: ErrNoTrailer},
		{name: "bad hash", msg: "subject\n\nFestival-Direction: sha256:nope\nFestival-Normalization: 2\n", wantIs: ErrMalformedTrailer},
		{name: "bad version", msg: "subject\n\nFestival-Direction: " + testHash + "\nFestival-Normalization: two\n", wantIs: ErrMalformedTrailer},
		{name: "missing version", msg: "subject\n\nFestival-Direction: " + testHash + "\n", wantIs: ErrMalformedTrailer},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if _, err := ParseTrailers(tc.msg); !errors.Is(err, tc.wantIs) {
				t.Fatalf("err = %v, want %v", err, tc.wantIs)
			}
		})
	}
}

func TestAppendAndParseTrailers(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		msg         string
		wantContain []string
	}{
		{name: "plain message gets a new block", msg: "[FE-FEST-a3b2c1] Fix login\n\nSome body.\n", wantContain: []string{"[FE-FEST-a3b2c1] Fix login\n\nSome body.\n\nFestival-Direction: " + testHash + "\nFestival-Normalization: 2\n"}},
		{name: "subject only", msg: "just a subject", wantContain: []string{"just a subject\n\nFestival-Direction: "}},
		{name: "joins an existing trailer block", msg: "subject\n\nbody\n\nSigned-off-by: a <a@b>\n", wantContain: []string{"Signed-off-by: a <a@b>\nFestival-Direction: " + testHash}},
		{name: "replaces stale values", msg: "subject\n\nFestival-Direction: sha256:" + strings.Repeat("0", 64) + "\nFestival-Normalization: 1\n", wantContain: []string{"Festival-Direction: " + testHash + "\nFestival-Normalization: 2\n"}},
		{name: "keeps git comment tail after the trailers", msg: "subject\n\nbody\n\n# Please enter the commit message\n# Changes to be committed:\n#\tmodified: x\n", wantContain: []string{"body\n\nFestival-Direction: " + testHash + "\nFestival-Normalization: 2\n\n# Please enter"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out, changed := AppendTrailers(tc.msg, sampleResult)
			if !changed {
				t.Fatalf("expected change")
			}
			for _, w := range tc.wantContain {
				if !strings.Contains(out, w) {
					t.Fatalf("output missing %q:\n%s", w, out)
				}
			}
			if !strings.HasPrefix(out, strings.SplitN(tc.msg, "\n", 2)[0]) {
				t.Fatalf("subject changed:\n%s", out)
			}
			got, err := ParseTrailers(out)
			if err != nil || got.DirectionHash != testHash || got.NormalizationVersion != 2 {
				t.Fatalf("parse back: %+v, %v", got, err)
			}
			again, changed := AppendTrailers(out, sampleResult)
			if changed || again != out {
				t.Fatalf("not idempotent")
			}
		})
	}
}

func TestTrailersAgreeWithGit(t *testing.T) {
	t.Parallel()
	requireGit(t)
	out, _ := AppendTrailers("[FE-DA0004] subject\n\nbody\n\nSigned-off-by: a <a@b>\n", sampleResult)
	file := filepath.Join(t.TempDir(), "msg")
	if err := os.WriteFile(file, []byte(out), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "interpret-trailers", "--parse", file)
	parsed, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	s := string(parsed)
	for _, w := range []string{"Festival-Direction: " + testHash, "Festival-Normalization: 2", "Signed-off-by: a <a@b>"} {
		if !strings.Contains(s, w) {
			t.Fatalf("git did not see %q in:\n%s", w, s)
		}
	}
}
