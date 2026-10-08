#!/usr/bin/env just --justfile
# direction CLI — stable, verifiable direction records for Festival work units

set dotenv-load := true

binary_name := "direction"
bin_dir     := "bin"
cmd_path    := "./cmd/direction"
version_pkg := "github.com/Obedience-Corp/fest-direction/internal/version"
version     := env_var_or_default("VERSION", `git describe --tags --exact-match HEAD 2>/dev/null || echo "dev"`)
commit      := `git rev-parse --short HEAD 2>/dev/null || echo "unknown"`
build_date  := `date -u +"%Y-%m-%dT%H:%M:%SZ"`
ldflags     := "-X " + version_pkg + ".Version=" + version + " -X " + version_pkg + ".Commit=" + commit + " -X " + version_pkg + ".BuildDate=" + build_date

[doc('Build variants, install, clean')]
mod bin 'justfiles/build.just'

[doc('Tests, race, coverage, golden fixtures')]
mod tests 'justfiles/test.just'

[doc('Format, vet, golangci-lint')]
mod checks 'justfiles/lint.just'

[private]
default:
    @just --list --unsorted

# Install direction and the fest-direction plugin name to $GOBIN
install:
    #!/usr/bin/env bash
    set -euo pipefail
    go install -ldflags '{{ldflags}}' {{cmd_path}}
    # go install writes to GOBIN, otherwise the first GOPATH entry.
    # The list separator is ';' on Windows and ':' elsewhere. Choosing it from
    # the string would split a drive letter: C:\Users\alice\go has no semicolon.
    dest="$(go env GOBIN)"
    if [[ -z "$dest" ]]; then
        gopath="$(go env GOPATH)"
        case "$(go env GOHOSTOS)" in
            windows) dest="${gopath%%;*}/bin" ;;
            *) dest="${gopath%%:*}/bin" ;;
        esac
    fi
    exe="$(go env GOEXE)"
    cp -f "$dest/direction$exe" "$dest/fest-direction$exe"

# Tidy modules and verify the toolchain builds everything
bootstrap:
    go mod tidy
    go build ./...

# Run from source with build metadata injected
dev *ARGS:
    go run -ldflags '{{ldflags}}' {{cmd_path}} {{ARGS}}

# Build direction and the fest-direction plugin name into bin/
build:
    go build -ldflags '{{ldflags}}' -o {{bin_dir}}/{{binary_name}} {{cmd_path}}
    cp -f {{bin_dir}}/{{binary_name}} {{bin_dir}}/fest-direction

# Run the unit tests
test:
    go test ./...

# Format check, vet, and golangci-lint
lint:
    @just checks all

# Reproduce the golden direction-hash fixtures (phase 1 acceptance)
golden:
    @just tests golden
