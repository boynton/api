# api
An HTTP-oriented API design and prototyping tool.

This tool can read and write several API description formats, generate concise summaries.

## Installation

On a Mac, use homebrew:

    $ brew tap boynton/tap
    $ brew install api
    
The current executables are also available as assets in the current [GitHub Release](https://github.com/boynton/api/releases/latest).

To install from source, clone this repo and type "make". The build requires [Go](https://golang.org).

## Usage

Invoked with no arguments, `api` shows basic usage:

```
$ api
usage: api [-vlfhpq] [-w warnlev] [-ns namespace] [-e entityid] [-d outdir] [-g generator] [-a key=val]* [-t tag]* file ...
  -a value
        Additional named arguments for a generator
  -d string
        The directory to generate output into (defaults to none, i.e. echo to stdout)
  -e string
        Show the specified entity.
  -f    Force overwrite if output file exists
  -g string
        The generator for output (default "api")
  -h    Show more help information
  -l    List the entities in the model
  -ns string
    	The namespace to force if absent. Also used by the default api generator to flatten to a single namespace
  -p	Parse input, display parse tree, and exit.
  -q    Quiet tool output, make it less verbose
  -t value
        Tag of entities to include. Prefix tag with '-' to exclude that tag
  -v	Suppress validation of the assembled model.
  -w string
        Warnings. 'show' or 'supress' or 'error'. Default is 'show' (default "show")
```
For additional help and dezcription of supported generators, use the -h flag:

```
rio:api lee$ ./bin/api -h

Supported API description formats for each input file extension:
   .api      api (the default for this tool
   .smithy   smithy
   .json     api, smithy, openapi, swagger (inferred by looking at the file contents)

The '' and 'namespace' options allow specifying those attributes for input formats
that do not require or support them. Otherwise a default is used based on the model being parsed.

Supported generators and options used from config if present
- api: Prints the native API representation to stdout. This is the default.
- json: Prints the parsed API data representation in JSON to stdout
- smithy: Prints the Smithy IDL representation to stdout.
- smithy-ast: Prints the Smithy AST representation to stdout
- openapi: Prints the OpenAPI Spec v3 representation to stdout
- plantuml: Prints the PlantUML representation of the API to stdout.
- sadl: Prints the SADL (an older format similar to api) to stdout. Useful for some additional generators.
- html: Prints html to stdout
   "-a detail-generator=api" - to generate the detail entries with "api" instead of "smithy", which is the default
- markdown: Prints markdown to stdout
   "-a detail-generator=api" - to generate the detail entries with "api" instead of "smithy", which is the default

For any generator the following additional parameters are accepted:
- "-a sort" - causes the operations and types to be alphabetically sorted, by default the original order is preserved
```

In general, it takes an arbitrary set of input files, parses them, assembles them into a single model, and then uses
a generator to produce output. The generator defaults to the `api` tool's native description language. The API description
language is oriented towards HTTP-based APIs (not RPC), and supports a common subset of several other description languages
like Smithy and OpenAPI.

```
$ cat examples/hello.api
namespace examples
service HelloService
version "1.0"

//
// A minimal hello world action
//
operation Hello (method=GET, url="/hello") {
    input {
        caller String (query="caller")
    }
    output (status=200) {
        greeting String (payload)
    }
}
```

To parse and echo the result with the tool's native format:

```
$ api examples/hello.api
namespace examples
service HelloService
version "1.0"

//
// A minimal hello world action
//
operation Hello (method=GET, url="/hello") {
    input {
        caller String (query="caller")
    }
    output (status=200) {
        greeting String (payload)
    }
}

```

To show the tool's data representation in JSON:
```
$ api -g json examples/hello.api
{
  "id": "examples#HelloService",
  "version": "1.0",
  "operations": [
    {
      "comment": "A minimal hello world action",
      "id": "examples#Hello",
      "httpMethod": "GET",
      "httpUri": "/hello",
      "input": {
        "id": "examples#HelloInput",
        "fields": [
          {
            "name": "caller",
            "type": "base#String",
            "default": "Mystery Caller",
            "httpQuery": "caller"
          }
        ]
      },
      "output": {
        "id": "examples#HelloOutput",
        "httpStatus": 200,
        "fields": [
          {
            "name": "greeting",
            "type": "base#String",
            "httpPayload": true
          }
        ]
      }
    }
  ]
}
```

To convert this to [Smithy](https://awslabs.github.io/smithy/):
```
$ api -g smithy examples/hello.api > /tmp/hello.smithy
$ cat /tmp/hello.smithy
$version: "2"

namespace examples

service HelloService {
    version: "1.0"
    operations: [
        Hello
    ]
}

@readonly
@http(method: "GET", uri: "/hello", code: 200)
operation Hello {
    input := {
        @httpQuery("caller")
        caller: String = "Mystery Caller"
    }

    output := {
        @httpPayload
        greeting: String
    }
}
```

To parse the smithy back into api's native format:

```
$ api /tmp/hello.smithy
namespace examples
service HelloService
version "1.0"


//
// A minimal hello world action
//
operation Hello (method=GET, url="/hello") {
    input {
        caller String (query="caller")
    }
    output (status=200) {
        greeting String (payload)
    }
}
```

