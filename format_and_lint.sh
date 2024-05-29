#!/bin/zsh

PATH="$PATH:$HOME/go/bin"

reset
go mod tidy
gci write .
go vet .
goimports -w .
gofmt -w .
gofumpt -w .
golangci-lint run -v
