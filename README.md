# Hawk API Server

Next-generation cluster UI prototype

## Source installation + dependencies

Building requires Go v1.9.

``` bash
go get -u github.com/krig/hawk-apiserver
```

The rest of the instructions assume that the current working directory
is `$GOPATH/src/github.com/krig/hawk-apiserver`.

Generating `api_structs.go` requires the `cibToGoStruct` utility found
at https://github.com/liangxin1300/CibToGo to be installed. This file
is generated from the pacemaker schema, so a new schema version in
Pacemaker requires regenerating this file.

## Generating an SSL certificate

``` bash
SSLGEN_KEY=hawk.key SSLGEN_CERT=hawk.pem ./tools/generate-ssl-cert
```

## Building the server

``` bash
go generate
go build
```

## Running the tests

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


### Event subscription

There should be some way to subscribe to CIB events via the API.

Exactly what form this should take (WebSockets, long polling, etc.)
remains to be decided.


## TODO

* Unix socket reverse proxy support

* Cache for the file handler to avoid stat()ing on every request

