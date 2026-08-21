package anchor

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallAndUninstall(t *testing.T) {
	t.Parallel()
	repo := initRepo(t)
	ctx := context.Background()

	if err := Uninstall(ctx, repo); !errors.Is(err, ErrNoShim) {
		t.Fatalf("uninstall before install: %v", err)
	}
	res, err := Install(ctx, repo, InstallOptions{WorkUnit: "festivals/x"})
	if err != nil {
		t.Fatal(err)
	}
	shim, err := os.ReadFile(res.HookPath)
	if err != nil || !strings.Contains(string(shim), shimMarker) || !strings.Contains(string(shim), "direction hook commit-msg") {
		t.Fatalf("shim: %v\n%s", err, shim)
	}
	if st, _ := os.Stat(res.HookPath); st.Mode()&0o111 == 0 {
		t.Fatal("shim not executable")
	}
	cfg, err := ReadConfig(repo)
	if err != nil || cfg.DefaultWorkUnit != "festivals/x" {
		t.Fatalf("config: %+v, %v", cfg, err)
	}
	// Reinstall is idempotent.
	if _, err := Install(ctx, repo, InstallOptions{}); err != nil {
		t.Fatalf("reinstall: %v", err)
	}
	if err := Uninstall(ctx, repo); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(res.HookPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("shim still present after uninstall")
	}
}

func TestInstallForeignHook(t *testing.T) {
	t.Parallel()
	repo := initRepo(t)
	ctx := context.Background()
	hooksDir, _ := repo.hooksDir(ctx)
	foreign := filepath.Join(hooksDir, hookName)
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(foreign, []byte("#!/bin/sh\necho foreign\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(ctx, repo, InstallOptions{}); !errors.Is(err, ErrForeignHook) {
		t.Fatalf("expected ErrForeignHook, got %v", err)
	}
	res, err := Install(ctx, repo, InstallOptions{Force: true})
	if err != nil || !res.Chained {
		t.Fatalf("force install: %+v, %v", res, err)
	}
	if b, err := os.ReadFile(filepath.Join(hooksDir, chainedName)); err != nil || !strings.Contains(string(b), "foreign") {
		t.Fatalf("foreign hook not preserved: %v", err)
	}
	if err := Uninstall(ctx, repo); err != nil {
		t.Fatal(err)
	}
	if b, err := os.ReadFile(foreign); err != nil || !strings.Contains(string(b), "foreign") {
		t.Fatalf("foreign hook not restored: %v", err)
	}
}
