#!/bin/bash
export PATH=$HOME/go/bin:$PATH
go get -v -t ./...
go build -v
go test -cover

