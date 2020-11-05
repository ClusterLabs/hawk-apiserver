# this is the what ends up in the RPM "Version" field and embedded in the --version CLI flag
VERSION ?= $(shell .ci/get_version_from_git.sh)

# this will be used as the build date by the Go compile task
DATE = $(shell date --iso-8601=seconds)

default: build test
build:
	go vet ./...
	go build -ldflags "-s -w -X main.version=$(VERSION) -X main.buildDate=$(DATE)"
	go mod tidy
test:
	go test ./... -v

.PHONY: build test
