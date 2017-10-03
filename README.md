# Prox [![Build Status](https://travis-ci.org/fgrosse/prox.png)](https://travis-ci.org/fgrosse/prox) [![GitHub release](https://img.shields.io/github/tag/fgrosse/prox.svg?style=flat)](https://github.com/fgrosse/prox/releases)  [![License](https://img.shields.io/github/license/fgrosse/prox.svg)](https://github.com/fgrosse/prox/blob/master/LICENSE) [![GoDoc](https://godoc.org/github.com/fgrosse/prox?status.svg)](https://godoc.org/github.com/fgrosse/prox)

Prox is a process runner for Procfile-based applications inspired by [foreman][foreman].
With `prox` you can run several processes defined in a `Procfile` concurrently
within a single terminal. All process outputs are prefixed with their corresponding
process names. One of tje major use cases for this arises during development of an
application that consist of multiple processes (e.g. microservices and storage backends).
With a process runner you can easily start this "stack" of applications and inspect its
output in a single shell while interacting with the application. 

What makes prox special in comparison to other [other foreman clones](#similar-projects)
is that when prox starts the application, it will also listen on a unix socket for
requests. This makes it possible to interact with a running `prox` instance in
another terminal, for instance to tail the logs of a subset of processes. This can
be especially useful when working with many processes where the merged output of
all applications can be rather spammy and is hard to be read by humans. Other
interactions include inspecting the state of all running processes, scaling a
process to more instances or stopping and restarting of processes.

Prox primary use case is as a development tool to run your stack locally and to
help you understand what it is doing or why a component has crashed. To do this
it is planned to introduce the `Proxfile` which serves as an opt-in alternative
to the `Procfile` with more advanced features such as parsing structured log output
to highlight relevant messages (not yet implemented).

You may ask why not just use *docker* for local development since it provides similar
client/server based functionality to run multiple processes, especially when
using docker-compose. The reason is ease of development and a fast development cycle
also for small code changes. It just takes longer than necessary to recompile a binary
and additionally build the docker image. Also the extra file system and process isolation
that are one of dockers many benefits in a production environment can become quite
a nuisance during local development.

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
`prox version`. This is helpful when reporting issues and debugging but it is
otherwise of no use.

```bash
go get -d github.com/fgrosse/prox/cmd/prox
cd $GOPATH/src/github.com/fgrosse/prox
make install
```

## Usage

You always need a `Procfile` which defines all processes that you want to run.

```bash
$ cat Procfile
# You can use comments, empty lines are ignored as well
worker: my-worker -v /etc/foo # prox uses your $PATH and passes arguments and flags as expected

foo-service: CONFIG_DIR=$PWD/config foo-serve # You can set and use environment variables per job
bar-service: bar-serve # Additionally all processes inherit your shells environment
baz-service: baz-serve # If there is a .env file it will be used to set variables for all processes
```

Optionally you can create a `.env` file which must contain a new-line delimited
list of key=value pairs which specify additional environment variables that are
exported to all processes defined within the `Procfile`.

```bash
$ cat .env
NAMESPACE=production
FOO_URL=file://home/fgrosse/src/github.com/foo/bar

# Again you may use empty lines and comments starting with '#'
ETCD_ENDPOINT=localhost:2379
LOG=*:debug,xxx:info,cache:error,db:info

# You can also use environment variables that you have defined earlier or that
# are defined in the shell that started prox.
PATH=/etc/foo/$NAMESPACE/baz:$PATH

# Spaces are allowed in values without any extra quoting
GREETING=hello world
```

Then change into the directory which contains your `Procfile` and `.env` and start
prox.

```bash
$ prox
echo1    │ I am a process
echo2    │ Hello World
redis    │ 14773:C 03 Oct 21:17:26.487 # oO0OoO0OoO0Oo Redis is starting oO0OoO0OoO0Oo
redis    │ 14773:C 03 Oct 21:17:26.487 # Redis version=4.0.1, bits=64, commit=00000000, modified=0, pid=14773, just started
…
```

In order to follow the logs of a specific process open another terminal.

```bash
prox tail redis
redis    │ 15249:M 03 Oct 21:21:13.044 # Server initialized
redis    │ 15249:M 03 Oct 21:21:13.045 * DB loaded from disk: 0.000 seconds
redis    │ 15249:M 03 Oct 21:21:13.045 * Ready to accept connections
…
``` 

For a detailed description of all prox commands and flags refer to the output
of `prox help`.

## Similar Projects

- [foreman][foreman]: the original process runner by [David Dollar][foreman-blog]
- [forego][forego]: a 1-1 port of foreman to Go
- [goreman][goreman]: another clone of foreman with some undocumented RPC functionality via TCP ports (Go)
- [honcho][honcho]: a Python port of foreman
- [spm][spm]: Simple Process Manager with client/server communication via unix sockets (Go)
- [overmind][overmind]: a process manager for Procfile-based applications that relies on tmux sessions
- [and more …][more-similar]

## Dependencies

Prox uses [go dep][go-dep] as dependency management tool. All vendored dependencies
are specified in the [Gopkg.toml](Gopkg.toml) file and are checked in in the [vendor](vendor)
directory. Prox itself mainly relies on the Go standard library, [zap][zap] for logging,
[cobra/viper][cobra] for the CLI and [pkg/errors][pkg-errors] for error wrapping.

## License

Prox is licensed under the BSD 2-clause License. Please see the [LICENSE](LICENSE)
file for details. The individual licenses of the vendored dependencies can be
found in the [LICENSE-THIRD-PARTY](LICENSE-THIRD-PARTY) file.

## Contributing

Contributions are always welcome (use pull requests). Before you start working on
a bigger feature its always best to discuss ideas in a new github issue. For each
pull request make sure that you covered your changes and additions with unit tests.

Please keep in mind that I might not always be able to respond immediately but I
usually try to react within the week ☺.

[foreman]: https://github.com/ddollar/foreman
[forego]: https://github.com/ddollar/forego
[honcho]: https://github.com/nickstenning/honcho
[goreman]: https://github.com/mattn/goreman
[spm]: https://github.com/bytegust/spm
[overmind]: https://github.com/DarthSim/overmind
[releases]: https://github.com/fgrosse/prox/releases
[foreman-blog]: http://blog.daviddollar.org/2011/05/06/introducing-foreman.html
[more-similar]: https://github.com/ddollar/foreman#ports
[go-dep]: https://github.com/golang/dep
[zap]: https://godoc.org/go.uber.org/zap
[cobra]: https://github.com/spf13/cobra
[pkg-errors]: https://github.com/pkg/errors