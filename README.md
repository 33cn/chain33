[![pipeline status](https://gitlab.33.cn/chain33/chain33/badges/master/pipeline.svg)](https://gitlab.33.cn/chain33/chain33/commits/master)
[![coverage report](https://gitlab.33.cn/chain33/chain33/badges/master/coverage.svg)](https://gitlab.33.cn/chain33/chain33/commits/master)
[![Go Report Card](https://goreportcard.com/badge/gitlab.33.cn/chain33/chain33)](https://goreportcard.com/report/gitlab.33.cn/chain33/chain33)

# Chain33 Core Developer Edition

Official golang implementation of the chain33 blockchain framework

## Binary

**chain33-cli**: it is used test chain33 for a cli tool.

**chain33**: 33.cn blockchain product.

## Download

To install Chain33 Core Developer Edition on Mac, Windows, or Linux, please visit [our downloads page](www.33.cn).

## Building from source

Building chain33 requires both a Go (version 1.9 or later) and a protoc compiler.

* [Go](https://golang.org/doc/install) or [GOCN](https://golang.google.cn/dl/) version 1.9, with $GOPATH set to your preferred directory.

* [protoc](https://github.com/google/protobuf#protocol-compiler-installation) 3.1.0 or later and if you need to compile protos.

Clone this repository to $GOPATH

```shell
$ mkdir -p $GOPATH/src/gitlab.33.cn/chain33

$ git clone https://gitlab.33.cn/chain33/chain33.git $GOPATH/src/gitlab.33.cn/chain33/

$ cd $GOPATH/src/gitlab.33.cn/chain33
```

You can install them using your favourite package manager.

Once the dependencies are installed, run

```shell
$ make build
```

or, to build for all platform binary

```shell
$ make all
```

## Run

run chain33 binary application,you must with config file to run, for example

```shell
$ chain33 -f chain33.toml
```

## Test

run all unit tests

```shell
$ make test
```

## Feedback

Feedback is greatly appreciated.
At this stage, the maintainers are most interested in feedback centered on the user experience (UX) of chain33.
Do you have a good idea?
Do any of the commands have surprising effects, output, or results?
Let us know by filing an issue, describing what you did or wanted to do, what you expected to happen, and what actually happened.

## Contributing

Contributions are greatly appreciated.
The maintainers actively manage the issues list, and try to highlight issues suitable for newcomers.
To contribute, please read the contribution guidelines: https://golang.org/doc/contribute.html for more details.
Before starting any work, please either comment on an existing issue, or file a new one.
Note that the Go project uses the issue tracker for bug reports and proposals only. See https://golang.org/wiki/Questions for a list of places to ask questions about the Go language.

## License

