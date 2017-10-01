# Ideas
- command to show details about all running processes including the PID, uptime and environment
- command or config to mark processes (highlight its output either via background color or markes on the left (e.g. ┃, ║, ┋, …)
- command or config to scale processes
- command to simulate process crashes without bringing down the whole stack
- limit characters per row in output based on terminal width (with opt-out))
- configure JSON output and write rule (e.g. level=/FATAL|WARN|ERROR/i to color output)
- allow assigning one ore many groups to processes and then start a group via `prox start <group>` or tail group logs via `prox tail <group>`
- compile version into binary
- allow setting log level rather than passing verbose flag (maybe both?)