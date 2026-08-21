# Format, vet, golangci-lint

[private]
default:
    @just --list --justfile {{source_file()}}

# gofmt in place
[no-cd]
fmt:
    gofmt -w .

# Fail if any file is not gofmt-clean
[no-cd]
fmt-check:
    @test -z "$(gofmt -l .)" || (gofmt -l . && echo "gofmt: files need formatting" && exit 1)

# go vet
[no-cd]
vet:
    go vet ./...

# golangci-lint (includes the fmt.Errorf ban via forbidigo)
[no-cd]
golangci:
    golangci-lint run ./...

# Fail if README or docs use a term the claims policy forbids (docs/claims.md)
[no-cd]
claims:
    #!/usr/bin/env sh
    set -eu
    hits="$(grep -rIinE 'tamper-proof|trustless|provably correct|immutable audit trail|prompt archive|reproducible ai' README.md docs/ | grep -v '<!-- avoided -->' || true)"
    if [ -n "$hits" ]; then echo "forbidden claim wording:"; echo "$hits"; exit 1; fi
    echo "claims: clean"

# Everything
[no-cd]
all: fmt-check vet golangci claims
