SHELL := /bin/bash

.PHONY: build build-static test

build:
	CGO_ENABLED=0 go build -o bin/rss_notify .

build-static:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
		-ldflags="-w -s" \
		-trimpath \
		-mod=readonly \
		-o bin/rss_notify_static .

test: check
	@echo "Running tests..."
	go test -race -v ./...

check:
	go fmt ./...
	go fix ./...
	go vet ./...
	golangci-lint fmt
	golangci-lint run --fix
