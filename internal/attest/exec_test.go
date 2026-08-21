package attest

import (
	"os/exec"
	"testing"
)

func execCommand(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).CombinedOutput()
}

func gitC(t *testing.T, dir string, args ...string) {
	t.Helper()
	base := []string{"-C", dir, "-c", "user.name=t", "-c", "user.email=t@t", "-c", "commit.gpgsign=false"}
	if out, err := exec.Command("git", append(base, args...)...).CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}
