package anchor

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/Obedience-Corp/fest-direction/internal/errs"
)

const (
	shimMarker    = "# direction-hook v1"
	hookName      = "commit-msg"
	preCommitName = "pre-commit"
	configPath    = ".direction/config.yaml"
)

// hookShim is the script git runs. pre-commit retargets a renamed work unit
// before the commit tree is written; commit-msg appends the trailers.
func hookShim(command string) string {
	return "#!/bin/sh\n" + shimMarker + " — see docs/anchoring.md\n" +
		"here=\"$(dirname \"$0\")\"\n" +
		"chained=\"$here/" + command + ".before-direction\"\n" +
		"if [ -x \"$chained\" ]; then \"$chained\" \"$@\" || exit $?; fi\n" +
		"exec fest-direction hook " + command + " \"$@\"\n"
}

var (
	// ErrForeignHook is returned when a commit-msg or pre-commit hook not
	// written by direction is already installed and Force was not given.
	ErrForeignHook = errors.New("a git hook not written by direction already exists")
	// ErrNoShim is returned by Uninstall when no direction shim is installed.
	ErrNoShim = errors.New("no direction commit-msg shim installed")
)

// Config is .direction/config.yaml at the repository root.
type Config struct {
	DefaultWorkUnit string `yaml:"default_work_unit"`
}

// InstallOptions tune Install.
type InstallOptions struct {
	// WorkUnit, when set, is written to .direction/config.yaml as the default
	// (repo-relative, or absolute inside the repo).
	WorkUnit string
	// Force moves a foreign commit-msg hook aside (kept and chained) instead
	// of refusing.
	Force bool
}

// InstallResult reports what Install wrote.
type InstallResult struct {
	HookPath    string
	ConfigPath  string
	Chained     bool
	HooksPathCf string
}

// Install writes the commit-msg and pre-commit shims into the repository's
// hooks directory (honouring core.hooksPath) and optionally the default work unit.
func Install(ctx context.Context, repo Repo, opts InstallOptions) (InstallResult, error) {
	var res InstallResult
	hooksDir, err := repo.hooksDir(ctx)
	if err != nil {
		return res, err
	}
	res.HooksPathCf, _ = repo.configValue(ctx, "core.hooksPath")
	res.HookPath = filepath.Join(hooksDir, hookName)
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		return res, errs.Wrap("install: hooks dir", err)
	}
	chained, err := installShim(hooksDir, hookName, hookShim(hookName), opts.Force)
	if err != nil {
		return res, err
	}
	res.Chained = chained
	if _, err := installShim(hooksDir, preCommitName, hookShim(preCommitName), opts.Force); err != nil {
		return res, err
	}
	if opts.WorkUnit != "" {
		rel, err := repo.Rel(filepath.Join(repo.Root, opts.WorkUnit))
		if filepath.IsAbs(opts.WorkUnit) {
			rel, err = repo.Rel(opts.WorkUnit)
		}
		if err != nil {
			return res, err
		}
		res.ConfigPath = filepath.Join(repo.Root, filepath.FromSlash(configPath))
		if err := writeConfig(res.ConfigPath, Config{DefaultWorkUnit: rel}); err != nil {
			return res, err
		}
	}
	return res, nil
}

// installShim writes one hook. A foreign hook is refused unless force, in
// which case it is kept as <name>.before-direction and chained.
func installShim(hooksDir, name, contents string, force bool) (bool, error) {
	path := filepath.Join(hooksDir, name)
	chained := filepath.Join(hooksDir, name+".before-direction")
	existing, err := os.ReadFile(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
	case err != nil:
		return false, errs.Wrap("install: read existing hook", err)
	case !strings.Contains(string(existing), shimMarker):
		if !force {
			return false, errs.Wrap("install "+path, ErrForeignHook)
		}
		if err := os.Rename(path, chained); err != nil {
			return false, errs.Wrap("install: move foreign hook aside", err)
		}
	}
	kept := false
	if _, err := os.Stat(chained); err == nil {
		kept = true
	}
	return kept, errs.Wrap("install: write shim", os.WriteFile(path, []byte(contents), 0o755))
}

// Uninstall removes the shims and restores a chained hook if one was kept.
func Uninstall(ctx context.Context, repo Repo) error {
	hooksDir, err := repo.hooksDir(ctx)
	if err != nil {
		return err
	}
	found := false
	for _, name := range []string{hookName, preCommitName} {
		removed, err := uninstallShim(hooksDir, name)
		if err != nil {
			return err
		}
		found = found || removed
	}
	if !found {
		return errs.Wrap("uninstall", ErrNoShim)
	}
	return nil
}

func uninstallShim(hooksDir, name string) (bool, error) {
	path := filepath.Join(hooksDir, name)
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) || (err == nil && !strings.Contains(string(b), shimMarker)) {
		return false, nil
	}
	if err != nil {
		return false, errs.Wrap("uninstall: read hook", err)
	}
	if err := os.Remove(path); err != nil {
		return false, errs.Wrap("uninstall: remove shim", err)
	}
	chained := filepath.Join(hooksDir, name+".before-direction")
	if _, err := os.Stat(chained); err == nil {
		return true, errs.Wrap("uninstall: restore chained hook", os.Rename(chained, path))
	}
	return true, nil
}

// ReadConfig loads .direction/config.yaml; a missing file yields a zero Config.
func ReadConfig(repo Repo) (Config, error) {
	b, err := os.ReadFile(filepath.Join(repo.Root, filepath.FromSlash(configPath)))
	if errors.Is(err, fs.ErrNotExist) {
		return Config{}, nil
	}
	if err != nil {
		return Config{}, errs.Wrap("read "+configPath, err)
	}
	var c Config
	if err := yaml.Unmarshal(b, &c); err != nil {
		return Config{}, errs.Wrap("parse "+configPath, err)
	}
	return c, nil
}

func writeConfig(path string, c Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return errs.Wrap("write config: mkdir", err)
	}
	b, err := yaml.Marshal(c)
	if err != nil {
		return errs.Wrap("write config: encode", err)
	}
	return errs.Wrap("write config", os.WriteFile(path, b, 0o644))
}

func (r Repo) hooksDir(ctx context.Context) (string, error) {
	out, err := runGit(ctx, r.Root, "rev-parse", "--git-path", "hooks")
	if err != nil {
		return "", err
	}
	dir := strings.TrimSpace(out)
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(r.Root, dir)
	}
	return dir, nil
}

func (r Repo) configValue(ctx context.Context, key string) (string, error) {
	out, err := runGit(ctx, r.Root, "config", "--get", key)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}
