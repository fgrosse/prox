# Prox [![Build Status](https://travis-ci.org/fgrosse/prox.png)](https://travis-ci.org/fgrosse/prox) [![GitHub release](https://img.shields.io/github/tag/fgrosse/prox.svg?style=flat)](https://github.com/fgrosse/prox/releases)  [![License](https://img.shields.io/github/license/fgrosse/prox.svg)](https://github.com/fgrosse/prox/blob/master/LICENSE) [![GoDoc](https://godoc.org/github.com/fgrosse/prox?status.svg)](https://godoc.org/github.com/fgrosse/prox)

Prox is a process runner for Procfile-based applications inspired by [foreman][foreman].
With `prox` you can run several processes defined in a `Procfile` concurrently
within a single terminal. All process outputs are merged but prefixed with their
corresponding process names. One of the major use cases for this is the local
development of an application that consist of multiple processes (e.g.
microservices and storage backends). With a process runner you can easily start
this "stack" of applications and inspect its output in a single shell while
interacting with the application.

You may ask why not just use *docker* for local development since it provides
similar functionality to run multiple processes, especially when using docker-compose.
The reason is ease of development and a fast development cycle also for small
code changes. It just takes longer than necessary to recompile a binary and
additionally build the docker image. Also the extra file system and process isolation
that are one of dockers many benefits in a production environment can become quite
a nuisance during local development.

## Features

Prox primary use case is as a development tool to run your entire application stack
on your local machine. Apart from running all components, Proxs primary goal is to
help you understand what the application is doing and sometimes help to debug why
a component has crashed.

### Error reporting

Like other process managers, prox will stop the entire stack when one of the
managed processes has crashed. This way the system fails fast and it is the
developers task to understand and fix the problem. This usually entails searching
through the log output for the first fatal error which caused the system to go down.
Prox helps with this by reporting the name and exit code of the process that was
the root cause for the stack shutdown.

### Log parsing

Today it is good practice for applications to emit structured log output so it
can be parsed and used easily. Prox detects if a process encodes its logs as JSON
and can use this information to reformat and color the output. By default this is
used to highlight error messages but the user can specify custom formatting as well.

In the future log parsing can also be used during error reporting to print the
last error message of the component which crashed the stack.

### Prox Server

Another thing that distinguishes Prox from [other foreman clones](#similar-projects)
is that when prox starts the application, it will also listen on a unix socket for
requests. This makes it possible to interact with a running Prox instance in
another terminal, for instance to tail the logs of a subset of processes. This can
be useful when working with many processes where the merged output of all
applications can be rather spammy and is hard to be read by humans.

The current version of the prox server only implements tailing but you can take
a look at the [IDEAS.md](IDEAS.md) file for other functionality that might be
implemented later on.

## Proxfile

Advanced users can use a slightly more complicated `Proxfile` which serves as an
opt-in alternative to the `Procfile` but with more features (see [usage below](#advanced-proxfile-usage)).

## Installation

All you need to run prox is a single binary. You can either use a prebuilt
binary or build prox from source.

### Prebuilt binaries

Download the binary from the [releases page][releases] and extract the `prox` binary
somewhere in your path.

### Building from source

If you want to compile prox from source you need a working installation of Go
version 1.9 or greater.

You can either install prox directly via `go get` or use the `make install` target.
The preferred way is the installation via make since this compiles version information
into the `prox` binary. You can inspected the version of your prox binary via
`prox version`. This is helpful when reporting issues and debugging but it is
otherwise of no use.

```bash
go get -v github.com/fgrosse/prox/cmd/prox
cd $GOPATH/src/github.com/fgrosse/prox
make install
```

## Usage

You always need a `Procfile` or `Proxfile` which defines all processes that you want to run.

### Simple Procfile usage

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

In order to follow the logs of a _specific_ process open another terminal.

```bash
prox tail redis
redis    │ 15249:M 03 Oct 21:21:13.044 # Server initialized
redis    │ 15249:M 03 Oct 21:21:13.045 * DB loaded from disk: 0.000 seconds
redis    │ 15249:M 03 Oct 21:21:13.045 * Ready to accept connections
…
``` 

For a detailed description of all prox commands and flags refer to the output
of `prox help`.

### Advanced Proxfile usage

Instead of using a standard `Procfile` and `.env` file you can combine both in a
`Proxfile`. Additionally this gives you access to more features such as custom
coloring of structured log output.

```bash
$ cat Proxfile
version: 1 # The Proxfile file format is versioned

# Internally the Proxfile is parsed as YAML.
# You can use comments, empty lines are ignored as well.

processes:
  redis: redis-server # Like the Procfile you specify processes as "name: shell script"

  foo-service:
    script: foo-service --debug -a -b 42
    env:
      - "CONFIG_DIR=$PWD/config foo-serve"

  echo:
    script: "echo $LISTEN_ADDR"
    env:
      - "LISTEN_ADDR=localhost:1232"

  example-3:
    script: my-app
    tags:
      errors:
        color: red
        condition:
          field: level
          value: "/error|fatal/i"
```

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
[cobra/viper][cobra] for the CLI and [pkg/errors][pkg-errors] for error wrapping as well
as [hashicorp/go-multierror][multi-errors].

## License

Prox is licensed under the BSD 2-clause License. Please see the [LICENSE](LICENSE)
file for details. The individual licenses of the vendored dependencies can be
found in the [LICENSE-THIRD-PARTY](LICENSE-THIRD-PARTY) file.

## Contributing

Contributions are always welcome (use pull requests). Before you start working on
a bigger feature its always best to discuss ideas in a new github issue. For each
pull request make sure that you covered your changes and additions with unit tests.

Please keep in mind that I might not always be able to respond immediately but I
usually try to react within a week or two ☺.

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
[multi-errors]: https://github.com/hashicorp/go-multierror
