#!/usr/bin/env just --justfile
# fest-direction — stable, verifiable direction records for Festival work units.
# The binary name is the fest plugin: `fest direction`.

set dotenv-load := true

binary_name := "fest-direction"
bin_dir     := "bin"
cmd_path    := "./cmd/fest-direction"
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

# Install the fest plugin binary to the directory go install uses
install:
    go install -ldflags '{{ldflags}}' {{cmd_path}}

# Tidy modules and verify the toolchain builds everything
bootstrap:
    go mod tidy
    go build ./...

# Run from source with build metadata injected
dev *ARGS:
    go run -ldflags '{{ldflags}}' {{cmd_path}} {{ARGS}}

# Build the fest plugin binary into bin/
build:
    go build -ldflags '{{ldflags}}' -o {{bin_dir}}/{{binary_name}} {{cmd_path}}

# Run the unit tests
test:
    go test ./...

# Format check, vet, and golangci-lint
lint:
    @just checks all

# Reproduce the golden direction-hash fixtures (phase 1 acceptance)
golden:
    @just tests golden
