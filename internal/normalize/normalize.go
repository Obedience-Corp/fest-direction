package normalize

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/Obedience-Corp/fest-direction/internal/errs"
)

// Report counts what Normalize did.
type Report struct {
	Files                 int
	FieldsStripped        int
	CheckboxesReset       int
	Excluded              int
	StatusHistoryStripped bool
}

var (
	// ErrDestNotEmpty is returned when dst exists and has entries.
	ErrDestNotEmpty = errors.New("destination exists and is not empty")
	// ErrSymlinkRejected mirrors SPEC §4.3: symlinks are not part of a bundle.
	ErrSymlinkRejected = errors.New("symlinks are not allowed")
	// ErrNotADirectory is returned when src is not a directory.
	ErrNotADirectory = errors.New("source is not a directory")
)

// Normalize copies the work-unit tree at src into dst with execution state
// removed under policy p: the policy's excluded names are skipped, *.md frontmatter is
// filtered, TODO.md checkboxes are reset, and fest.yaml loses
// metadata.status_history. Relative paths are preserved exactly because SPEC
// §7.1 hashes them. dst must be absent or an empty directory.
func Normalize(ctx context.Context, src, dst string, p Policy) (Report, error) {
	var r Report
	if err := ctx.Err(); err != nil {
		return r, err
	}
	src, err := filepath.Abs(src)
	if err != nil {
		return r, errs.Wrap("normalize: resolve source", err)
	}
	st, err := os.Stat(src)
	if err != nil {
		return r, errs.Wrap("normalize: stat source", err)
	}
	if !st.IsDir() {
		return r, errs.Wrap("normalize "+src, ErrNotADirectory)
	}
	if err := ensureEmptyDir(dst); err != nil {
		return r, err
	}
	w := &walker{ctx: ctx, src: src, dst: dst, policy: p, report: &r}
	return r, filepath.WalkDir(src, w.visit)
}

func ensureEmptyDir(dst string) error {
	entries, err := os.ReadDir(dst)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return errs.Wrap("normalize: create destination", os.MkdirAll(dst, 0o755))
	case err != nil:
		return errs.Wrap("normalize: read destination", err)
	case len(entries) > 0:
		return errs.Wrap("normalize "+dst, ErrDestNotEmpty)
	}
	return nil
}

// walker carries the per-run inputs so the WalkDir callback stays small.
type walker struct {
	ctx    context.Context
	src    string
	dst    string
	policy Policy
	report *Report
}

func (w *walker) visit(path string, d fs.DirEntry, walkErr error) error {
	if walkErr != nil {
		return errs.Wrap("normalize: walk "+path, walkErr)
	}
	if err := w.ctx.Err(); err != nil {
		return err
	}
	rel, err := filepath.Rel(w.src, path)
	if err != nil {
		return errs.Wrap("normalize: relative path for "+path, err)
	}
	if rel == "." {
		return nil
	}
	if w.policy.Excluded(d.Name()) {
		w.report.Excluded++
		if d.IsDir() {
			return filepath.SkipDir
		}
		return nil
	}
	if d.Type()&fs.ModeSymlink != 0 {
		return errs.Wrap("normalize "+rel, ErrSymlinkRejected)
	}
	if d.IsDir() {
		return errs.Wrap("normalize: mkdir "+rel, os.MkdirAll(filepath.Join(w.dst, rel), 0o755))
	}
	if !d.Type().IsRegular() {
		return nil
	}
	return w.file(rel)
}

// file applies the rules to one regular file and writes it under dst.
func (w *walker) file(rel string) error {
	data, err := os.ReadFile(filepath.Join(w.src, rel))
	if err != nil {
		return errs.Wrap("normalize "+rel, err)
	}
	if data, err = w.apply(rel, data); err != nil {
		return err
	}
	target := filepath.Join(w.dst, rel)
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return errs.Wrap("normalize: mkdir for "+rel, err)
	}
	if err := os.WriteFile(target, data, 0o644); err != nil {
		return errs.Wrap("normalize: write "+rel, err)
	}
	w.report.Files++
	return nil
}

// apply runs the rules in order: frontmatter + EOF, TODO checkboxes, fest.yaml.
func (w *walker) apply(rel string, data []byte) ([]byte, error) {
	if filepath.Ext(rel) == ".md" {
		out, n, err := filterFrontmatter(data, w.policy)
		if err != nil {
			return nil, errs.Wrap("normalize "+rel, err)
		}
		data = trimEOF(out)
		w.report.FieldsStripped += n
	}
	if filepath.Base(rel) == "TODO.md" {
		out, n := resetCheckboxes(data)
		data = out
		w.report.CheckboxesReset += n
	}
	if rel == "fest.yaml" {
		out, removed, err := stripStatusHistory(data)
		if err != nil {
			return nil, errs.Wrap("normalize "+rel, err)
		}
		data = out
		w.report.StatusHistoryStripped = removed
	}
	return data, nil
}

// trimEOF ends a document with exactly one newline. Editors and fest's own
// writer disagree about trailing whitespace; none of it is direction.
func trimEOF(b []byte) []byte {
	t := bytes.TrimRight(b, " \t\r\n")
	if len(t) == 0 {
		return t
	}
	return append(t, '\n')
}
