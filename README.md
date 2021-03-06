**THIS PACKAGE IS DEPRECATED**

Since I ended up ditching Avro in favor of Protobuf for everything I do, this repo is not mantained anymore. I would accept fixes and PRs if someone wanted to fix something.

---

# Schemagen [![GoDoc](https://godoc.org/github.com/burdiyan/schemagen?status.svg)](https://godoc.org/github.com/burdiyan/schemagen)

This is a tool that fetches Avro schemas from [Confluent Schema Registry](https://github.com/confluentinc/schema-registry) and compiles them to Go code.

Code generation is entirely based on [gogen-avro](https://github.com/alanctgardner/gogen-avro).

Additionally, `schemagen` will generate [goka.Codec](https://godoc.org/github.com/lovoo/goka#Codec) compatible type for all Avro schemas of type `record`.

## Installation

Right now the only way to install `schemagen` is to build it from source:

```
go install github.com/burdiyan/schemagen/cmd/...
```

## Getting Started

1. Create a file named `.schemagen.yaml` in the root of your project.
2. Specify Schema Registry URL, subjects and versions of the schema you want to download and compile.
3. Run `schemagen` to download the schemas from Schema Registry and compile them.

### Config Example

```
kind: Avro
registry: http://confluent-schema-registry.default.svc.cluster.local:8081
schemas:
  - subject: my-topic-value
    version: latest
    package: country # This is the name of the Go package that will be generated.
  - subject: another-topic-value
    version: "2"
    package: anothertopic
compile: true
outputDir: ./foo
```
