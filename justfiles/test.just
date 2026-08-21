# Tests, race, coverage, golden fixtures

[private]
default:
    @just --list --justfile {{source_file()}}

# Unit tests
unit:
    go test ./...

# Unit tests with the race detector
race:
    go test -race ./...

# Coverage profile and HTML report
cover:
    go test -coverprofile=coverage.out ./...
    go tool cover -html=coverage.out -o coverage.html
    @echo "coverage.html written"

# Golden direction-hash fixtures: baseline == mutated, snapshot ids differ,
# PoC-rules reproduction of sha256:50ddaea0…
golden:
    go test ./... -run 'Golden' -count=1 -v
