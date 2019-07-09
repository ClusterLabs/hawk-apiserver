# Hawk API Server

[![Build Status](https://travis-ci.org/ClusterLabs/hawk-apiserver.svg?branch=master)](https://travis-ci.org/ClusterLabs/hawk-apiserver)
[![Maintainability](https://api.codeclimate.com/v1/badges/9f372cd2a2dac74962e2/maintainability)](https://codeclimate.com/github/ClusterLabs/hawk-apiserver/maintainability)
[![Test Coverage](https://api.codeclimate.com/v1/badges/9f372cd2a2dac74962e2/test_coverage)](https://codeclimate.com/github/ClusterLabs/hawk-apiserver/test_coverage)
[![GoDoc](https://godoc.org/github.com/ClusterLabs/hawk-apiserver?status.svg)](https://godoc.org/github.com/ClusterLabs/hawk-apiserver)
[![Go Report Card](https://goreportcard.com/badge/github.com/ClusterLabs/hawk-apiserver)](https://goreportcard.com/report/github.com/ClusterLabs/hawk-apiserver)

HTTPS API server / minimalist web proxy for Hawk.

# Table of content

- [Rationale](#Rationale)
- [Build](#build)
- [Devel](#devel)
- [Testing](#testing)
- [openapi specification](#openapi)

# Rationale
This project currently provides a minimalistic web server which
handles SSL certificate termination, proxying and static file serving
for Hawk, a HA cluster dashboard and control interface written in Ruby
on Rails.

The primary goal for this project is to provide the minimal web server
needed by Hawk while consuming as few system resources as
possible. Second, it provides the `/monitor` API endpoint which
handles long-lived connections from the frontend to enable instant
updates of the interface on cluster events.

In the future, the API server will provide a complete and documented
REST API for Pacemaker and Pacemaker-based HA clusters, including the
ability to provide status information and metrics for inclusion in
other dashboards or for monitoring tools like Prometheus.

# Build 

### Dependencies:

- GO >= 1.9
- following pkgs: `libqb-devel libpacemaker-devel`.

``` bash
go get -u github.com/ClusterLabs/hawk-apiserver
```

The rest of the instructions assume that the current working directory
is `$GOPATH/src/github.com/ClusterLabs/hawk-apiserver`.

# Devel

## Generating the API structures

Generating `api_structs.go` requires the `cibToGoStruct` utility found
at https://github.com/liangxin1300/CibToGo to be installed. This file
is generated from the pacemaker schema, so a new schema version in
Pacemaker requires regenerating this file.

Since a version of `api_structs.go` is already checked into the
repository, running this utility is only necessary if this file hasn't
been updated to match the current version of Pacemaker.

## Using the enter script

The `/enter` script can be used to set up a build and test environment
inside a Docker container, so that all the dependencies are in place
and there is no need to install things in the host system. If Docker
is installed and the user has the necessary permissions to run
containers, `/enter` should bring you to a prepared environment ready
to run the rest of the commands.

## Generating an SSL certificate

``` bash
SSLGEN_KEY=hawk.key SSLGEN_CERT=hawk.pem ./tools/generate-ssl-cert
```

## Building the server

``` bash
go generate
go build
```

# Testing

``` bash
go test
```

## Configuration

Pass `-config <config>` as an argument to give the server a
configuration file. The format is a json dictionary with key / value
pairs.

The available configuration values are described below. If a value is
set both in the configuration file and in a command line argument, the
command line argument takes precedence.

* `key`: Path to SSL key. (argument: -key)

* `cert`: Path to SSL certificate. (argument: -cert)

* `port`: TCP port to listen to for connections. (argument: -port)

* `route`: List of json maps that configure the routing table.

The route format is very limited and adapted to serving hawk, but
enable reconfiguration of the exact paths to certificates, files and
sockets.

Example:

``` json
{
  "key": "/etc/hawk/hawk.key",
  "cert": "/etc/hawk/hawk.pem",
  "port": 7630,
  "route": [
    {
      "handler": "api/v1",
      "path": "/api/v1"
    },
    {
      "handler": "monitor",
      "path": "/monitor"
    },
    {
      "handler": "file",
      "path": "/",
      "target": "/usr/share/hawk/public"
    },
    {
      "handler": "proxy",
      "path": "/",
      "target": "unix:///var/run/hawk/app.sock"
    }
  ]
}
```

## API

Testing using curl:

``` bash
curl --insecure -u hacluster:<pass> https://<server>:<port>/api/v1/cib
```

### Authentication

* Basic auth: Get user:password from HTTP headers. Map to system
  user. Verify that system user is a member of the haclient group.

* Cookie auth (cookie created by hawk rails app): If a valid cookie is
  found in the HTTP headers, this is accepted as authentication.
  Session cookie is stored in attrd.

* TODO: SAML2

### Endpoints

HTTP verbs:

* POST: Create new resources
* GET: Retrieve resource
* PUT: Update resource (should be idempotent)
* PATCH: Update resource (not yet supported)
* DELETE: Delete a resource

``` bash
GET                 /api/v1/features
GET/POST/PUT/DELETE /api/v1/cib
GET/POST/PUT/DELETE /api/v1/cib/attributes
GET                 /api/v1/cib/status
GET/POST/PUT/DELETE /api/v1/cib/configuration
GET/POST/PUT/DELETE /api/v1/cib/configuration/crm_config
GET/POST/PUT/DELETE /api/v1/cib/configuration/crm_config/{id}
GET/POST/PUT/DELETE /api/v1/cib/configuration/nodes
GET/POST/PUT/DELETE /api/v1/cib/configuration/nodes/{id}
GET/POST/PUT/DELETE /api/v1/cib/configuration/resources
GET/POST/PUT/DELETE /api/v1/cib/configuration/resources/{id}
GET/POST/PUT/DELETE /api/v1/cib/configuration/constraints
GET/POST/PUT/DELETE /api/v1/cib/configuration/constraints/{id}
GET/POST/PUT/DELETE /api/v1/cib/configuration/fencing
GET/POST/PUT/DELETE /api/v1/cib/configuration/acls
GET/POST/PUT/DELETE /api/v1/cib/configuration/acls/{id}
GET/POST/PUT/DELETE /api/v1/cib/configuration/tags
GET/POST/PUT/DELETE /api/v1/cib/configuration/tags/{id}
GET/POST/PUT/DELETE /api/v1/cib/configuration/alerts
GET/POST/PUT/DELETE /api/v1/cib/configuration/alerts/{id}
GET/POST/PUT/DELETE /api/v1/cib/configuration/op_defaults
GET/POST/PUT/DELETE /api/v1/cib/configuration/op_defaults/{id}
GET/POST/PUT/DELETE /api/v1/cib/configuration/rsc_defaults
GET/POST/PUT/DELETE /api/v1/cib/configuration/rsc_defaults/{id}
```


### Shadow CIBs and simulation

The web server SHOULD support the Shadow CIB feature, which includes
the simulator interface. If the Shadow CIB feature is supported, the
object returned by `GET /api/v1/features` MUST include `shadow: true`.

All endpoints can also receive these extra arguments to determine
which CIB is being accessed:

* `CIB_file`: (DEV only) Refers to a path on the server.
* `CIB_shadow`: Identifier for the shadow CIB residing on the server.

Adds the following API endpoints which creates or deletes the server
shadow CIB identifier. Use the CIB_shadow parameter together with the
regular configuration API to modify the actual contents of the shadow
CIB.

``` bash
GET/POST/PUT/DELETE /api/v1/shadow
GET/POST/PUT/DELETE /api/v1/shadow/{id}
PUT /api/v1/shadow/{id}
```

Adds the following API endpoints to invoke the simulator:

``` bash
POST/PUT /api/v1/shadow/{id}/simulate
```

The simulator accepts the special identifier `live` as ID to simulate
using the live CIB as input.

The simulator run and the results are returned as part of the request
body.

### Openapi

This project support the openapi specification V2.0, via [go-swagger](https://github.com/go-swagger/go-swagger)

