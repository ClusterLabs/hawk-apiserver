#!/bin/bash
mkdir -p ~/go
export GOPATH=~/go
go get -t ./...
go build
go test
