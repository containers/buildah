## buildah-run "1" "March 2017" "buildah"

## NAME
buildah run - Run a command inside of the container.

## SYNOPSIS
**buildah** **run** **containerID** [*options* [...] --] **command**

## DESCRIPTION
Launches a container and runs the specified command in that container using the
container's root filesystem as a root filesystem, using configuration settings
inherited from the container's image or as specified using previous calls to
the *buildah config* command.  If you execute *buildah run* and expect an
interactive shell, you need to specify the --tty flag.

## OPTIONS

**--add-host**=[]
   Add a custom host-to-IP mapping (host:ip)

   Add a line to /etc/hosts. The format is hostname:ip. The **--add-host**
option can be set multiple times.

**--cgroup-parent**=""
   Path to cgroups under which the cgroup for the container will be created. If the path is not absolute, the path is considered to be relative to the cgroups path of the init process. Cgroups will be created if they do not already exist.

**--cpu-period**=*0*
    Limit the CPU CFS (Completely Fair Scheduler) period

    Limit the container's CPU usage. This flag tell the kernel to restrict the container's CPU usage to the period you specify.

**--cpu-quota**=*0*
   Limit the CPU CFS (Completely Fair Scheduler) quota

   Limit the container's CPU usage. By default, containers run with the full
CPU resource. This flag tell the kernel to restrict the container's CPU usage
to the quota specified.

**--cpu-shares**=*0*
   CPU shares (relative weight)

   By default, all containers get the same proportion of CPU cycles. This proportion
can be modified by changing the container's CPU share weighting relative
to the weighting of all other running containers.

To modify the proportion from the default of 1024, use the **--cpu-shares**
flag to set the weighting to 2 or higher.

The proportion will only apply when CPU-intensive processes are running.
When tasks in one container are idle, other containers can use the
left-over CPU time. The actual amount of CPU time will vary depending on
the number of containers running on the system.

For example, consider three containers, one has a cpu-share of 1024 and
two others have a cpu-share setting of 512. When processes in all three
containers attempt to use 100% of CPU, the first container would receive
50% of the total CPU time. If you add a fourth container with a cpu-share
of 1024, the first container only gets 33% of the CPU. The remaining containers
receive 16.5%, 16.5% and 33% of the CPU.

On a multi-core system, the shares of CPU time are distributed over all CPU
cores. Even if a container is limited to less than 100% of CPU time, it can
use 100% of each individual CPU core.

For example, consider a system with more than three cores. If you start one
container **{C0}** with **-c=512** running one process, and another container
**{C1}** with **-c=1024** running two processes, this can result in the following
division of CPU shares:

    PID    container	CPU	CPU share
    100    {C0}		0	100% of CPU0
    101    {C1}		1	100% of CPU1
    102    {C1}		2	100% of CPU2

**--cpuset-cpus**=""
   CPUs in which to allow execution (0-3, 0,1)

**--cpuset-mems**=""
   Memory nodes (MEMs) in which to allow execution (0-3, 0,1). Only effective on NUMA systems.

   If you have four memory nodes on your system (0-3), use `--cpuset-mems=0,1`
then processes in your container will only use memory from the first
two memory nodes.

**--hostname**
Set the hostname inside of the running container.


**-m**, **--memory**=""
   Memory limit (format: <number>[<unit>], where unit = b, k, m or g)

   Allows you to constrain the memory available to a container. If the host
supports swap memory, then the **-m** memory setting can be larger than physical
RAM. If a limit of 0 is specified (not using **-m**), the container's memory is
not limited. The actual limit may be rounded up to a multiple of the operating
system's page size (the value would be very large, that's millions of trillions).

**--memory-swap**="LIMIT"
   A limit value equal to memory plus swap. Must be used with the  **-m**
(**--memory**) flag. The swap `LIMIT` should always be larger than **-m**
(**--memory**) value.  By default, the swap `LIMIT` will be set to double
the value of --memory.

   The format of `LIMIT` is `<number>[<unit>]`. Unit can be `b` (bytes),
`k` (kilobytes), `m` (megabytes), or `g` (gigabytes). If you don't specify a
unit, `b` is used. Set LIMIT to `-1` to enable unlimited swap.

**--runtime** *path*

The *path* to an alternate OCI-compatible runtime.

**--runtime-flag** *flag*

Adds global flags for the container runtime. To list the supported flags, please
consult manpages of your selected container runtime (`runc` is the default
runtime, the manpage to consult is `runc(8)`).
Note: Do not pass the leading `--` to the flag. To pass the runc flag `--log-format json`
to buildah run, the option given would be `--runtime-flag log-format=json`.

**--security-opt**=[]
   Security Options

    "label=user:USER"   : Set the label user for the container
    "label=role:ROLE"   : Set the label role for the container
    "label=type:TYPE"   : Set the label type for the container
    "label=level:LEVEL" : Set the label level for the container
    "label=disable"     : Turn off label confinement for the container
    "no-new-privileges" : Disable container processes from gaining additional privileges

    "seccomp=unconfined" : Turn off seccomp confinement for the container
    "seccomp=profile.json :  White listed syscalls seccomp Json file to be used as a seccomp filter

    "apparmor=unconfined" : Turn off apparmor confinement for the container
    "apparmor=your-profile" : Set the apparmor confinement profile for the container

**--tty**

By default a pseudo-TTY is allocated only when buildah's standard input is
attached to a pseudo-TTY.  Setting the `--tty` option to `true` will cause a
pseudo-TTY to be allocated inside the container connecting the user's "terminal"
with the stdin and stdout stream of the container.  Setting the `--tty` option to
`false` will prevent the pseudo-TTY from being allocated.

**--ulimit**=[]
    Ulimit options

**--volume, -v** *source*:*destination*:*flags*

Bind mount a location from the host into the container for its lifetime.

NOTE: End parsing of options with the `--` option, so that you can pass other 
options to the command inside of the container

## EXAMPLE

buildah run containerID -- ps -auxw

buildah run containerID --hostname myhost -- ps -auxw

buildah run --runtime-flag log-format=json containerID /bin/bash

buildah run --runtime-flag debug containerID /bin/bash

buildah run --tty containerID /bin/bash

buildah run --tty=false containerID ls /

## SEE ALSO
buildah(1)
