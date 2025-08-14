.POSIX:

PROJECT = gollm

code:
	code .

test-verbose:
	go test -v

test:
	go test

fmt:
	go fmt ./*go

build: build-linux build-macos build-windows

build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./bin/$(PROJECT)-linux-amd64 *.go

build-macos:
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o ./bin/$(PROJECT)-darwin-arm64 *.go
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o ./bin/$(PROJECT)-darwin-amd64 *.go

build-windows:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o ./bin/$(PROJECT)-windows-amd64.exe *.go
