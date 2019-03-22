#!/bin/bash
export PATH=$HOME/go/bin:$PATH
go get github.com/axw/gocov/gocov
go get -v -t ./...
go build -v
go test -coverprofile c.out -coverpkg github.com/ClusterLabs/hawk-apiserver,github.com/ClusterLabs/hawk-apiserver/api,github.com/ClusterLabs/hawk-apiserver/cib,github.com/ClusterLabs/hawk-apiserver/metrics,github.com/ClusterLabs/hawk-apiserver/server,github.com/ClusterLabs/hawk-apiserver/util
