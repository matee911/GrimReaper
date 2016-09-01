VERSION=$(shell ./version.sh)
LDFLAGS=-ldflags '-X "main.version=$(VERSION)"'

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
	go test -v ./... -bench=. -benchtime 1s -benchmem

linux32:
	mkdir -p bin/linux_i386
	GOARCH=386 GOOS=linux go build $(LDFLAGS)
	mv GrimReaper bin/linux_i386/grimreaper

linux64:
	mkdir -p bin/linux_amd64
	GOARCH=amd64 GOOS=linux go build $(LDFLAGS)
	mv GrimReaper bin/linux_amd64/grimreaper

darwin64:
	mkdir -p bin/darwin_amd64
	GOARCH=amd64 GOOS=darwin go build $(LDFLAGS)
	mv GrimReaper bin/darwin_amd64/grimreaper
