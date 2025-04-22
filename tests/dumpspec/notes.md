Global flags:
 crun: --cgroup-manager=MANAGER --debug --log=FILE --log-format={text|json} --log-level --root=DIR --rootless={true|false|auto} --systemd-cgroup
 runc: --debug --log=FILE --log-format={text|json} --root=DIR --systemd-cgroup --rootless={true|false|auto}

create [-b|--bundle dir] [--console-socket[=]path] [--pid-file[=]path] [--no-pivot] [--preserve-fds[=]N] containerID
 runc: [--pidfd-socket=path] [--no-new-keyring]
 crun: [-f|--config file] [--no-subreaper (ignored)] [--no-new-keyring]
 runsc: [--pidfd-socket=path]
* Start keeping track of containerID under --root or $XDG_RUNTIME_DIR/$runtimeName
* If console socket given, allocate a pseudoterminal, connect to it, and pass a TTY descriptor.
* If not, pass stdio down directly.
* Prepare, but have babysitter wait before starting process.

start containerID
* Start process connected to stdio or terminal.

state containerID
 crun: [-a|--all] [-r|--regex regex]
 runsc: [-all|--all] [-pid int (in parent pid namespace)]
* Output a JSON-encoded github.com/opencontainers/runtime-spec/specs-go.State value on stdout.

kill containerID [signal]
 crun: [-a|--all] [-r|--regex regex]
 runsc: [-all|--all] [-pid int (in parent pid namespace)]
* Send signal to process tree.

delete containerID
 runc: [-f|--force (SIGKILL first if need be)]
 crun: [-f|--force (SIGKILL first if need be)] [-r|--regex regex]
 runsc: [-force|--force]

runc: checkpoint events exec features list pause ps resume restore run spec state update
crun: checkpoint exec features list pause ps resume restore run spec state update
runsc: checkpoint do events exec flags list pause port-forward ps restore resume run spec state wait
