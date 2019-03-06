#!/bin/bash
export PATH=$HOME/go/bin:$PATH
go get github.com/axw/gocov/gocov
go get -v -t ./...
go build -v
go test -coverprofile c.out

