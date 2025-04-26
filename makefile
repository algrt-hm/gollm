.POSIX:

PROJECT = gollm

fmt:
	go fmt ./*go

test:
	go test -v

build: build-linux build-macos

build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./bin/$(PROJECT)-linux-amd64 *.go

build-macos:
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o ./bin/$(PROJECT)-darwin-arm64 *.go
