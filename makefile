.POSIX:

PROJECT = gollm
BINARY = $(PROJECT)
SOURCES = $(wildcard *.go)

# Default target:
# Build only if sources are newer than binary
$(BINARY): $(SOURCES)
	go build -o $(BINARY) .

help:
	@echo "make        - Build (if needed) and run gollm"
	@echo "make build  - Build gollm"
	@echo "make test   - Run tests"
	@echo "make fmt    - Format code"
	@echo "make update - Update dependencies"
	@echo "make release - Bump tag and push release tag"

build: $(BINARY)

update:
	go get -u github.com/openai/openai-go
	go get -u github.com/google/generative-ai-go/genai
	go get -u "github.com/anthropics/anthropic-sdk-go"

code:
	code .

test-verbose:
	go test -v

test:
	go test

fmt:
	gofmt -w .

release:
	./scripts/release.sh

build-all-platforms: build-linux build-macos build-windows

build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./bin/$(PROJECT)-linux-amd64 *.go

build-macos: build-macos-arm64 build-macos-amd64

build-macos-arm64:
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o ./bin/$(PROJECT)-darwin-arm64 *.go

build-macos-amd64:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o ./bin/$(PROJECT)-darwin-amd64 *.go

build-windows:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o ./bin/$(PROJECT)-windows-amd64.exe *.go
