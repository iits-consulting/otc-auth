#!/bin/zsh
reset                       && \
gci write .                 && \
go vet .                    && \
goimports -w .              && \
gofmt -w .                  && \
gofumpt  -w .               && \
$HOME/go/bin/golangci-lint run -v