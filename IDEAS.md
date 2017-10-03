## Ideas for future features

### General

- limit characters per row in output based on terminal width (with opt-out))
- configure JSON output and write rule (e.g. level=/FATAL|WARN|ERROR/i to color output)
- allow assigning one ore many groups to processes and then start a group via `prox start <group>` or tail group logs via `prox tail <group>`
- keep process logs in tmp dir and allow tailing logs from start

### Client / Server

- command or config to mark processes (highlight its output either via background color or marks on the left (e.g. ┃, ║, ┋, …)
- command or config to scale processes (start new instances)
- command to simulate process crashes without bringing down the whole stack (can already be done via kill)