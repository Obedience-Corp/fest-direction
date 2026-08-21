# Format, vet, golangci-lint

[private]
default:
    @just --list --justfile {{source_file()}}

# gofmt in place
fmt:
    gofmt -w .

# Fail if any file is not gofmt-clean
fmt-check:
    @test -z "$(gofmt -l .)" || (gofmt -l . && echo "gofmt: files need formatting" && exit 1)

# go vet
vet:
    go vet ./...

# golangci-lint (includes the fmt.Errorf ban via forbidigo)
golangci:
    golangci-lint run ./...

# Everything
all: fmt-check vet golangci
