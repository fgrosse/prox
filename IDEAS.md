## Ideas for future features

### General
- fix all TODOs!
- update README
- review godoc
- check code coverage

### Output
- apply tags also to output send via "prox tail"

### Client / Server
- restart process (e.g. because its binary was rebuild)
  - `prox restart foobar`


## Ideas for after v1.0.0

### General
- start individual processes via `prox start foo bar baz`
- allow assigning one ore many groups to processes and then start a group via `prox start <group>` or tail group logs via `prox tail <group>`
- keep process logs in tmp dir and allow tailing logs from start
- add way to start everything except some specific processes
  - e.g. `prox start !foo !bar`

### Output
- limit characters per row in output based on terminal width (with opt-out))
- allow a single structured output tag to be applied to multiple fields

### Client / Server
- command or config to mark processes (highlight its output either via background color or marks on the left (e.g. ┃, ║, ┋, …)
  this could also use more advanced ad hoc rules if structured logging is enabled (e.g. host="foobar")
- command or config to scale processes (start new instances)
- command to simulate process crashes without bringing down the whole stack (can already be done via kill)
- watch for new binaries and restart automatically
- use gRPC instead of crappy own protocol
