package anchor

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Obedience-Corp/fest-direction/internal/errs"
)

// ErrSymlinkInHistory is returned when HEAD's tree contains a symlink under
// the work unit; bundles cannot contain them (SPEC §4.3).
var ErrSymlinkInHistory = errors.New("committed tree contains a symlink")

// headTree materializes HEAD's version of the repo-relative path rel. This is
// how anchoring hashes exactly what a reader can later retrieve, regardless
// of the working tree.
func (r Repo) headTree(ctx context.Context, rel string) (string, func(), error) {
	return r.treeAt(ctx, rel, "HEAD")
}

// indexTree materializes the staged version of rel — what a commit in
// progress will contain — by writing the index as a tree object first.
func (r Repo) indexTree(ctx context.Context, rel string) (string, func(), error) {
	out, err := runGit(ctx, r.Root, "write-tree")
	if err != nil {
		return "", nil, err
	}
	return r.treeAt(ctx, rel, strings.TrimSpace(out))
}

// treeAt extracts treeish:rel into a private temp directory and returns the
// work-unit root inside it. The caller must invoke cleanup.
func (r Repo) treeAt(ctx context.Context, rel, treeish string) (string, func(), error) {
	if err := ctx.Err(); err != nil {
		return "", nil, err
	}
	tmp, err := os.MkdirTemp("", "direction-tree-*")
	if err != nil {
		return "", nil, errs.Wrap("head tree: temp dir", err)
	}
	cleanup := func() { _ = os.RemoveAll(tmp) }
	cmd := exec.CommandContext(ctx, "git", "-C", r.Root, "archive", "--format=tar", treeish, "--", rel)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cleanup()
		return "", nil, errs.Wrap("tree: pipe", err)
	}
	if err := cmd.Start(); err != nil {
		cleanup()
		return "", nil, errs.Wrap("tree: start git archive", err)
	}
	extractErr := extractTar(stdout, tmp)
	waitErr := cmd.Wait()
	if ctx.Err() != nil {
		cleanup()
		return "", nil, ctx.Err()
	}
	if waitErr != nil {
		cleanup()
		return "", nil, errs.Wrap("git archive "+treeish+" -- "+rel, &gitFailure{args: "archive " + treeish + " -- " + rel, stderr: strings.TrimSpace(stderr.String()), code: exitCode(waitErr)})
	}
	if extractErr != nil {
		cleanup()
		return "", nil, extractErr
	}
	return filepath.Join(tmp, filepath.FromSlash(rel)), cleanup, nil
}

func exitCode(err error) int {
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return exit.ExitCode()
	}
	return -1
}

// extractTar writes regular files and directories from a tar stream under
// dst, refusing paths that escape it and symlinks.
func extractTar(r io.Reader, dst string) error {
	tr := tar.NewReader(r)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return errs.Wrap("head tree: read archive", err)
		}
		target := filepath.Join(dst, filepath.FromSlash(hdr.Name))
		if target != dst && !strings.HasPrefix(target, dst+string(filepath.Separator)) {
			return errs.Wrap("head tree: unsafe path "+hdr.Name, errs.ErrInvalidInput)
		}
		if err := writeEntry(tr, hdr, target); err != nil {
			return err
		}
	}
}

func writeEntry(tr *tar.Reader, hdr *tar.Header, target string) error {
	switch hdr.Typeflag {
	case tar.TypeDir:
		return errs.Wrap("head tree: mkdir "+hdr.Name, os.MkdirAll(target, 0o755))
	case tar.TypeSymlink, tar.TypeLink:
		return errs.Wrap("head tree: "+hdr.Name, ErrSymlinkInHistory)
	case tar.TypeReg:
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return errs.Wrap("head tree: mkdir for "+hdr.Name, err)
		}
		f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err != nil {
			return errs.Wrap("head tree: create "+hdr.Name, err)
		}
		if _, err := io.Copy(f, tr); err != nil {
			_ = f.Close()
			return errs.Wrap("head tree: write "+hdr.Name, err)
		}
		return errs.Wrap("head tree: close "+hdr.Name, f.Close())
	default:
		return nil // pax headers and other entries carry no content
	}
}
