# Harmonies

Next-generation cluster UI prototype

## Source installation + dependencies

Building requires Go v1.7+.

``` bash
go get -u github.com/krig/harmonies
```

The rest of the instructions assume that the current working directory
is `$GOPATH/src/github.com/krig/harmonies`.

## Generating an SSL certificate

``` bash
SSLGEN_KEY=harmonies.key SSLGEN_CERT=harmonies.pem ./tools/generate-ssl-cert
```

## Building the server

``` bash
go build
```

## Running the tests

``` bash
go test
```
