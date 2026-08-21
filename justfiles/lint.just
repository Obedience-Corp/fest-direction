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

# Everything
[no-cd]
all: fmt-check vet golangci
