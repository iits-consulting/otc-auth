reset && gofmt -w . && gofumpt -l -w . && gci write . && go vet . && $HOME/go/bin/golangci-lint run --enable-all -v