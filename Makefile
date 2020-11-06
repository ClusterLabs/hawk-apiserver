# this is the what ends up in the RPM "Version" field and embedded in the --version CLI flag
VERSION ?= $(shell .ci/get_version_from_git.sh)

default: build test
build:
	go vet ./...
	go build -ldflags "-s -w -X main.version=$(VERSION)"
	go mod tidy
test:
	go test ./... -v

.PHONY: build test
