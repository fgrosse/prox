# Prox [![Build Status](https://travis-ci.org/fgrosse/prox.png)](https://travis-ci.org/fgrosse/prox) [![GitHub release](https://img.shields.io/badge/version-0.5-blue.svg?style=flat)](https://github.com/fgrosse/prox/releases)  [![License](https://img.shields.io/badge/license-MIT-4183c4.svg)](https://github.com/fgrosse/prox/blob/master/LICENSE) [![GoDoc](https://godoc.org/github.com/fgrosse/prox?status.svg)](https://godoc.org/github.com/fgrosse/prox) [![Coverage Status](https://coveralls.io/repos/github/fgrosse/prox/badge.svg?branch=master)](https://coveralls.io/github/fgrosse/prox?branch=master) [![Go Report Card](https://goreportcard.com/badge/github.com/fgrosse/prox)](https://goreportcard.com/report/github.com/fgrosse/prox)

Prox is a process runner for Procfile-based applications inspired by [foreman][foreman].
With `prox` you can run several processes defined in a `Procfile` concurrently
within a single terminal. All process outputs are prefixed with their corresponding
process names. The use case for this is often when developing applications that
consist of multiple processes (e.g. a database, a client and two workers). With
a process runner you can easily start this "stack" of applications and inspect its
output in a single shell while interacting with the application. 

TODO: asciinema

What makes prox special in comparison to other [other foreman clones](#similar-projects)
is that when prox starts the application, it will also listen on a unix socket for
requests. This makes it possible to interact with a running `prox` instance on
another shell, for instance to tail the logs of a subset of processes. This can
be especially useful when working with many processes where the merged output of
all applications can be rather spammy and is hard to be read by humans. Other
interactions include inspecting the state of all running processes, scaling a
process to more instances or stopping and restarting of processes.

Prox primary use case is as a development tool to run your stack locally and to
help you understand what it is doing or why a component has crashed. To do this
it is planned to introduce the `Proxfile` which serves as an opt-in alternative
to the `Procfile` with more advanced features such as parsing structured log output
to highlight relevant messages (not yet implemented).

## Installation

All you need to run prox is a single binary. You can either use a prebuilt
binary or build prox from source.

### Prebuilt binaries

Download the binary from the [releases page][releases] and extract the `prox` binary
somewhere in your path.

### Building from source

If you want to compile prox from source you need a working installation of Go
version 1.9 or greater and the `GOPATH` environment variable must be set.

You can either install prox directly via `go get` or use the `make install` target.
The preferred way is the installation via make since this compiles version information
into the `prox` binary. You can inspected the version of your prox binary via
`prox version`. This is helpful when reporting issues and debugging but is
otherwise of no use.

```bash
go get -d github.com/fgrosse/prox/cmd/prox
cd $GOPATH/src/github.com/fgrosse/prox
make install
```

## Usage

```
$ prox help
A process runner for Procfile-based applications

Usage:
  prox [flags]
  prox [command]

Available Commands:
  help        Help about any command
  ls          List information about currently running processes
  start       Run all processes
  tail        Follow the log output of running processes
  version     Print the version of prox and then exit

Flags:
  -e, --env string        path to the env file (default ".env")
  -h, --help              help for prox
      --no-colour         disable colored output
  -f, --procfile string   path to the Procfile (default "Procfile")
  -s, --socket string     path of the temporary unix socket file that clients can use to establish a connection (default ".prox.sock")
  -v, --verbose           enable detailed log output for debugging

Use "prox [command] --help" for more information about a command.
```

## Similar Projects

- [forego][forego]
- [goreman][goreman]
- [spm][spm]
- [overmind][overmind]

TODO: how is prox different from the other projects

TODO: maybe another word about motivation to write yet another Procfile runner

## License

Prox is licensed under the the MIT license. Please see the [LICENSE](LICENSE)
file for details. The individual licenses of the vendored dependencies can be
found in the [LICENSE-THIRD-PARTY](LICENSE-THIRD-PARTY) file.

## Contributing

Contributions are always welcome (use pull requests). Before you start working on
a bigger feature its always best to discuss ideas in a new github issue. For each
pull request make sure that you covered your changes and additions with unit tests.

Please keep in mind that I might not always be able to respond immediately but I
usually try to react within the week ☺.

[forego]: https://github.com/ddollar/forego
[goreman]: https://github.com/mattn/goreman
[spm]: https://github.com/bytegust/spm
[overmind]: https://github.com/DarthSim/overmind
[releases]: https://github.com/fgrosse/prox/releases