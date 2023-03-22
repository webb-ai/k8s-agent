GOCMD?=go

PKG_SRC=$(shell find . -type f -name '*.go')
VERSION=$(shell git describe --dirty --tags)
LDFLAGS=-ldflags "-X main.BuildVersion=$(VERSION)"

## Dev
format:
	goimports -w $$(find . -type f -name '*.go' -not -path "./vendor/*" -not -path "./.git/*")

lint:
	$(GOCMD) vet ./...
	golangci-lint run ./...

test:
	$(GOCMD) test -v -cover ./...

check: format lint test

## Build
build:
	$(GOCMD) build $(LDFLAGS) -o collector ./cmd/collector

clean:
	-rm collector