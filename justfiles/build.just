# Build variants, install, clean

binary_name := "direction"
bin_dir     := "bin"
cmd_path    := "./cmd/direction"
version_pkg := "github.com/Obedience-Corp/fest-direction/internal/version"
version     := env_var_or_default("VERSION", `git describe --tags --exact-match HEAD 2>/dev/null || echo "dev"`)
commit      := `git rev-parse --short HEAD 2>/dev/null || echo "unknown"`
build_date  := `date -u +"%Y-%m-%dT%H:%M:%SZ"`
ldflags     := "-X " + version_pkg + ".Version=" + version + " -X " + version_pkg + ".Commit=" + commit + " -X " + version_pkg + ".BuildDate=" + build_date

[private]
default:
    @just --list --justfile {{source_file()}}

# Build direction and the fest-direction plugin name into bin/
[no-cd]
local:
    go build -ldflags '{{ldflags}}' -o {{bin_dir}}/{{binary_name}} {{cmd_path}}
    cp -f {{bin_dir}}/{{binary_name}} {{bin_dir}}/fest-direction

# Build without optimizations for debugging
[no-cd]
debug:
    go build -gcflags 'all=-N -l' -o {{bin_dir}}/{{binary_name}}-debug {{cmd_path}}

# Install direction and the fest-direction plugin name to $GOBIN
[no-cd]
install:
    #!/usr/bin/env bash
    set -euo pipefail
    go install -ldflags '{{ldflags}}' {{cmd_path}}
    dest="$(go env GOBIN)"
    if [[ -z "$dest" ]]; then dest="$(go env GOPATH)/bin"; fi
    cp -f "$dest/direction" "$dest/fest-direction"

# Remove both names from $GOBIN
[no-cd]
uninstall:
    #!/usr/bin/env bash
    set -euo pipefail
    dest="$(go env GOBIN)"
    if [[ -z "$dest" ]]; then dest="$(go env GOPATH)/bin"; fi
    rm -f "$dest/{{binary_name}}" "$dest/fest-direction"

# Remove build artifacts
[no-cd]
clean:
    rm -rf {{bin_dir}}/ coverage.out coverage.html
