# JSON validation service example

This repository contains a simple RESTful service for validating JSON documents
against a preloaded JSON schema.


## Prerequisites

* Go compiler
* For running the server copy `conf.example.json` to `conf.json` and customize


## Usage

Start the server (by default reads configuration from `conf.json`):

    make run

Use `curl` for testing:

    ### [SUCCESS] Upload test schema
    $ curl localhost:8080/schema/test1 -d @examples/test1-schema.json
    {"action":"uploadSchema","id":"test1","status":"success"}

    ### [ERROR] Upload another schema to the same schema ID
    $ curl localhost:8080/schema/test1 -d '{"type": "object"}'
    {"action":"uploadSchema","id":"test1","status":"error","message":"already exists"}

    ### [SUCCESS] Validate test JSON against test schema
    $ curl localhost:8080/validate/test1 -d @examples/test1.json
    {"action":"validateDocument","id":"test1","status":"success"}

    ### [ERROR] Validate test JSON against undefined schema
    $ curl localhost:8080/validate/test2 -d @examples/test1.json
    {"action":"validateDocument","id":"test2","status":"error","message":"not found"}

    ### [ERROR] Validate invalid test JSON
    $ curl localhost:8080/validate/test1 -d @examples/test1-invalid.json
    {"action":"validateDocument","id":"test1","status":"error","message":"jsonschema: '/chunks/1' does not validate with test1.json/properties/chunks/items/type: expected object, but got null"}

    ### [SUCCESS] Upload another schema
    curl localhost:8080/schema/integer -d '{"type": "integer"}'
    {"action":"uploadSchema","id":"integer","status":"success"}

    ### [SUCCESS] Validate JSON number against integer schema
    $ curl localhost:8080/validate/integer -d '3'
    {"action":"validateDocument","id":"test1","status":"success"}

    ### [ERROR] Validate JSON string against integer schema
    curl localhost:8080/validate/integer -d '"3"'
    {"action":"validateDocument","id":"integer","status":"error","message":"jsonschema: '' does not validate with integer.json/type: expected integer, but got string"}

Run unit tests:

    make check


## Solution

* Using https://github.com/santhosh-tekuri/jsonschema for JSON schema validation

  * In case of errors, the library reveals the absolute paths of the system
    where the code is running or the schema file is located.
    This is not ideal and for that reason the validation errors from the library
    are first sanitized by replacing absolute paths with short code names before
    returning the error message to the client.

* Before validating JSON, remove all `null` values of the given JSON in order to
  accept null for e.g `integer` type field. Implementation:
    * Unmarshal given JSON into `interface{}` and traverse the
      value according to [the doc](https://pkg.go.dev/encoding/json@go1.19.2#Unmarshal):

      > JSON object is stored in map[string]interface{}
      >
      > JSON array is stored in []interface{}

    * Recurse into maps (JSON object) and slices (JSON array)
    * With maps, remove keys where value is nil
    * With arrays **do not** remove nulls, because that changes array structure
      (number of elements), which may be important

* Schema storage has common interface and 2 implementation:
    * Memory storage - simple map. This is used, when no other storage
    implementation is configured
    * File system storage - stores the schema files in the configured
    directory.


## Repository structure

```
├── Makefile
├── README.md
├── cmd
│   └── service
│       └── main.go             # Read conf, create services, start server, listens for signals
├── conf.example.json           # Example conf file
├── conf.json                   # (Created) The conf file for execution
├── data                        # (Created) The storage dir for example conf
├── examples                    # Examples for testing
│   ├── test1-invalid.json
│   ├── test1-schema.json
│   └── test1.json
├── go.mod                      # Go module definition
├── go.sum
├── internal
│   ├── conf
│   │   └── conf.go             # Conf data structure (refers to others), load conf
│   ├── schema
│   │   ├── handler.go          # Defines handlers dealing with schemas and validation
│   │   ├── handler_test.go     # HTTP unit tests
│   │   └── service.go          # Main logic: store and load schemas, validate JSON
│   ├── server
│   │   └── server.go           # HTTP server, register given handlers, graceful shutdown
│   └── storage
│       ├── filestorage.go      # Filesystem storage (persistent)
│       ├── memory.go           # In-memory storage (non-persistent)
│       └── storage.go          # Package facade - create implementation according to conf
└── service                     # (Created) The executable
```
