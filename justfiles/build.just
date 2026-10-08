# Build variants, install, clean

binary_name := "fest-direction"
bin_dir     := "bin"
cmd_path    := "./cmd/fest-direction"
version_pkg := "github.com/Obedience-Corp/fest-direction/internal/version"
version     := env_var_or_default("VERSION", `git describe --tags --exact-match HEAD 2>/dev/null || echo "dev"`)
commit      := `git rev-parse --short HEAD 2>/dev/null || echo "unknown"`
build_date  := `date -u +"%Y-%m-%dT%H:%M:%SZ"`
ldflags     := "-X " + version_pkg + ".Version=" + version + " -X " + version_pkg + ".Commit=" + commit + " -X " + version_pkg + ".BuildDate=" + build_date

[private]
default:
    @just --list --justfile {{source_file()}}

# Build the fest plugin binary into bin/
[no-cd]
local:
    go build -ldflags '{{ldflags}}' -o {{bin_dir}}/{{binary_name}} {{cmd_path}}

# Build without optimizations for debugging
[no-cd]
debug:
    go build -gcflags 'all=-N -l' -o {{bin_dir}}/{{binary_name}}-debug {{cmd_path}}

# Install the fest plugin binary to the directory go install uses
[no-cd]
install:
    go install -ldflags '{{ldflags}}' {{cmd_path}}

# Remove the fest plugin binary from the directory go install uses
[no-cd]
uninstall:
    #!/usr/bin/env bash
    set -euo pipefail
    rm -f "$(go list -f '{{{{.Target}}' {{cmd_path}})"

# Remove build artifacts
[no-cd]
clean:
    rm -rf {{bin_dir}}/ coverage.out coverage.html
