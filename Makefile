GOCMD?=go
DOCKER?=docker

PKG_SRC=$(shell find . -type f -name '*.go')
VERSION=$(shell git describe --dirty --tags)
IMAGE_REPO=public.ecr.aws/p5v6t9h8/k8s-resource-collector
LDFLAGS=-ldflags "-X main.BuildVersion=$(VERSION)"

## Dev
format:
	goimports -w $$(find . -type f -name '*.go' -not -path "./vendor/*" -not -path "./.git/*")

lint:
	$(GOCMD) vet ./...
	golangci-lint run ./...

test:
	$(GOCMD) test -v -cover ./...

check: lint test

## Build
build:
	$(GOCMD) build $(LDFLAGS) -o collector ./cmd/collector

clean:
	-rm collector

image: Dockerfile
	$(DOCKER) build -f Dockerfile -t $(IMAGE_REPO):$(VERSION) .

push-image: image
	$(DOCKER) push $(IMAGE_REPO):$(VERSION)
