VERSION=$(shell ./version.sh)

all: fmt test linux32 linux64 darwin64

help:
	@echo "Please use 'make <target>' where <target> is one of"
	@echo "  linux32          to build binary for linux/i386"
	@echo "  linux64          to build binary for linux/amd64"
	@echo "  darwin64         to build binary for darwin/amd64"
	@echo "  test             to run all tests"
	@echo "  fmt              to format code"

fmt:
	go fmt ./...

test:
	go test -v ./...

linux32:
	mkdir -p bin
	GOARCH=386 GOOS=linux go build -ldflags "-X main.version '$(VERSION)'"
	mv GrimReaper bin/GrimReaper-linux_i386

linux64:
	mkdir -p bin
	GOARCH=amd64 GOOS=linux go build -ldflags "-X main.version '$(VERSION)'"
	mv GrimReaper bin/GrimReaper-linux_amd64

darwin64:
	mkdir -p bin
	GOARCH=amd64 GOOS=darwin go build -ldflags "-X main.version '$(VERSION)'"
	mv GrimReaper bin/GrimReaper-darwin_amd64
