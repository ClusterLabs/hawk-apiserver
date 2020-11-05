# Hawk API Server

[![Build Status](https://travis-ci.org/ClusterLabs/hawk-apiserver.svg?branch=master)](https://travis-ci.org/ClusterLabs/hawk-apiserver)
[![GoDoc](https://godoc.org/github.com/ClusterLabs/hawk-apiserver?status.svg)](https://godoc.org/github.com/ClusterLabs/hawk-apiserver)
[![Go Report Card](https://goreportcard.com/badge/github.com/ClusterLabs/hawk-apiserver)](https://goreportcard.com/report/github.com/ClusterLabs/hawk-apiserver)

HTTPS API server / minimalist web proxy for Hawk.

# Table of content

- [Rationale](#Rationale)
- [Devel](#devel)
- [Usage](#usage)
- [Features](#features)

# Rationale
This project currently provides a minimalistic web server which
handles SSL certificate termination, proxying and static file serving
for [HAWK](https://github.com/ClusterLabs/hawk)

The **primary goal** for this project is to provide the minimal web server
needed by Hawk while consuming as few system resources as
possible. Second, it provides the `/monitor` API endpoint which
handles long-lived connections from the frontend to enable instant
updates of the interface on cluster events.


# Devel


### Dependencies:

- following pkgs: `libqb-devel libpacemaker-devel`.
Use `go build .` and other standards golang commands to test the project.

## Generating an SSL certificate

``` bash
SSLGEN_KEY=hawk.key SSLGEN_CERT=hawk.pem ./tools/generate-ssl-cert
```

# Usage:

The `hawk-api-server` is used currently mainly for hawk usage purposes.

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
# Features:

- HTTPS server
- reverse proxy
- ``/monitor`  API endpoint which handles long-lived connections from the frontend to enable instant
              updates of the interface on cluster events.


### Authentication

* Basic auth: Get user:password from HTTP headers. Map to system
  user. Verify that system user is a member of the haclient group.

* Cookie auth (cookie created by hawk rails app): If a valid cookie is
  found in the HTTP headers, this is accepted as authentication.
  Session cookie is stored in attrd.
