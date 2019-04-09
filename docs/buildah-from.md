# buildah-from "1" "March 2017" "buildah"

## NAME
buildah\-from - Creates a new working container, either from scratch or using a specified image as a starting point.

## SYNOPSIS
**buildah from** [*options*] *image*

## DESCRIPTION
Creates a working container based upon the specified image name.  If the
supplied image name is "scratch" a new empty container is created.  Image names
use a "transport":"details" format.

Multiple transports are supported:

  **dir:**_path_
  An existing local directory _path_ containing the manifest, layer tarballs, and signatures in individual files. This is a non-standardized format, primarily useful for debugging or noninvasive image inspection.

  **docker://**_docker-reference_ (Default)
  An image in a registry implementing the "Docker Registry HTTP API V2". By default, uses the authorization state in `$XDG\_RUNTIME\_DIR/containers/auth.json`, which is set using `(buildah login)`. If the authorization state is not found there, `$HOME/.docker/config.json` is checked, which is set using `(docker login)`.
  If _docker-reference_ does not include a registry name, *localhost* will be consulted first, followed by any registries named in the registries configuration.

  **docker-archive:**_path_
  An image is retrieved as a `docker load` formatted file.

  **docker-daemon:**_docker-reference_
  An image _docker-reference_ stored in the docker daemon's internal storage.  _docker-reference_ must include either a tag or a digest.  Alternatively, when reading images, the format can also be docker-daemon:algo:digest (an image ID).

  **oci:**_path_**:**_tag_**
  An image tag in a directory compliant with "Open Container Image Layout Specification" at _path_.

  **oci-archive:**_path_**:**_tag_
  An image _tag_ in a directory compliant with "Open Container Image Layout Specification" at _path_.

  **ostree:**_image_[**@**_/absolute/repo/path_]
  An image in local OSTree repository.  _/absolute/repo/path_ defaults to _/ostree/repo_.

### DEPENDENCIES

Buildah resolves the path to the registry to pull from by using the /etc/containers/registries.conf
file, registries.conf(5).  If the `buildah from` command fails with an "image not known" error,
first verify that the registries.conf file is installed and configured appropriately.

## RETURN VALUE
The container ID of the container that was created.  On error 1 is returned.

## OPTIONS

**--add-host**=[]

Add a custom host-to-IP mapping (host:ip)

Add a line to /etc/hosts. The format is hostname:ip. The **--add-host** option can be set multiple times.

**--authfile** *path*

Path of the authentication file. Default is ${XDG\_RUNTIME\_DIR}/containers/auth.json, which is set using `buildah login`.
If the authorization state is not found there, $HOME/.docker/config.json is checked, which is set using `docker login`.

**--cap-add**=*CAP\_xxx*

Add the specified capability to the default set of capabilities which will be
supplied for subsequent *buildah run* invocations which use this container.
Certain capabilities are granted by default; this option can be used to add
more.

**--cap-drop**=*CAP\_xxx*

Remove the specified capability from the default set of capabilities which will
be supplied for subsequent *buildah run* invocations which use this container.
The CAP\_AUDIT\_WRITE, CAP\_CHOWN, CAP\_DAC\_OVERRIDE, CAP\_FOWNER,
CAP\_FSETID, CAP\_KILL, CAP\_MKNOD, CAP\_NET\_BIND\_SERVICE, CAP\_SETFCAP,
CAP\_SETGID, CAP\_SETPCAP, CAP\_SETUID, and CAP\_SYS\_CHROOT capabilities are
granted by default; this option can be used to remove them.

If a capability is specified to both the **--cap-add** and **--cap-drop**
options, it will be dropped, regardless of the order in which the options were
given.

**--cert-dir** *path*

Use certificates at *path* (\*.crt, \*.cert, \*.key) to connect to the registry.
The default certificates directory is _/etc/containers/certs.d_.

**--cgroup-parent**=""

Path to cgroups under which the cgroup for the container will be created. If the path is not absolute, the path is considered to be relative to the cgroups path of the init process. Cgroups will be created if they do not already exist.

**--cidfile** *ContainerIDFile*

Write the container ID to the file.

**--cni-config-dir**=*directory*

Location of CNI configuration files which will dictate which plugins will be
used to configure network interfaces and routing when the container is
subsequently used for `buildah run`, if processes to be started will be run in
their own network namespaces, and networking is not disabled.

**--cni-plugin-path**=*directory[:directory[:directory[...]]]*

List of directories in which the CNI plugins which will be used for configuring
network namespaces can be found.

**--cpu-period**=*0*

Limit the CPU CFS (Completely Fair Scheduler) period

Limit the container's CPU usage. This flag tell the kernel to restrict the container's CPU usage to the period you specify.

**--cpu-quota**=*0*

Limit the CPU CFS (Completely Fair Scheduler) quota

Limit the container's CPU usage. By default, containers run with the full
CPU resource. This flag tell the kernel to restrict the container's CPU usage
to the quota you specify.

**--cpu-shares, -c**=*0*

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

**--creds** *creds*

The [username[:password]] to use to authenticate with the registry if required.
If one or both values are not supplied, a command line prompt will appear and the
value can be entered.  The password is entered without echo.

**--http-proxy**

By default proxy environment variables are passed into the container if set
for the buildah process.  This can be disabled by setting the `--http-proxy`
option to `false`.  The environment variables passed in include `http_proxy`,
`https_proxy`, `ftp_proxy`, `no_proxy`, and also the upper case versions of
those.

Defaults to `true`

**--ipc** *how*

Sets the configuration for IPC namespaces when the container is subsequently
used for `buildah run`.
The configured value can be "" (the empty string) or "container" to indicate
that a new IPC namespace should be created, or it can be "host" to indicate
that the IPC namespace in which `buildah` itself is being run should be reused,
or it can be the path to an IPC namespace which is already in use by
another process.

**--isolation** *type*

Controls what type of isolation is used for running processes under `buildah
run`.  Recognized types include *oci* (OCI-compatible runtime, the default),
*rootless* (OCI-compatible runtime invoked using a modified
configuration, with *--no-new-keyring* added to its *create*
invocation, with network and UTS namespaces disabled, and IPC, PID,
and user namespaces enabled; the default for unprivileged users), and
*chroot* (an internal wrapper that leans more toward chroot(1) than
container technology).

Note: You can also override the default isolation type by setting the
BUILDAH\_ISOLATION environment variable.  `export BUILDAH_ISOLATION=oci`

**--memory, -m**=""

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

**--name** *name*

A *name* for the working container

**--net** *how*
**--network** *how*

Sets the configuration for network namespaces when the container is subsequently
used for `buildah run`.
The configured value can be "" (the empty string) or "container" to indicate
that a new network namespace should be created, or it can be "host" to indicate
that the network namespace in which `buildah` itself is being run should be
reused, or it can be the path to a network namespace which is already in use by
another process.

**--pid** *how*

Sets the configuration for PID namespaces when the container is subsequently
used for `buildah run`.
The configured value can be "" (the empty string) or "container" to indicate
that a new PID namespace should be created, or it can be "host" to indicate
that the PID namespace in which `buildah` itself is being run should be reused,
or it can be the path to a PID namespace which is already in use by another
process.

**--pull**

When the flag is enabled, attempt to pull the latest image from the registries
listed in registries.conf if a local image does not exist or the image is newer
than the one in storage. Raise an error if the image is not in any listed
registry and is not present locally.

If the flag is disabled (with *--pull=false*), do not pull the image from the
registry, use only the local version. Raise an error if the image is not
present locally.

Defaults to *true*.

**--pull-always**

Pull the image from the first registry it is found in as listed in registries.conf.
Raise an error if not found in the registries, even if the image is present locally.

**--quiet, -q**

If an image needs to be pulled from the registry, suppress progress output.

**--security-opt**=[]

Security Options

  "label=user:USER"   : Set the label user for the container
  "label=role:ROLE"   : Set the label role for the container
  "label=type:TYPE"   : Set the label type for the container
  "label=level:LEVEL" : Set the label level for the container
  "label=disable"     : Turn off label confinement for the container
  "no-new-privileges" : Not supported

  "seccomp=unconfined" : Turn off seccomp confinement for the container
  "seccomp=profile.json :  White listed syscalls seccomp Json file to be used as a seccomp filter

  "apparmor=unconfined" : Turn off apparmor confinement for the container
  "apparmor=your-profile" : Set the apparmor confinement profile for the container

**--shm-size**=""

Size of `/dev/shm`. The format is `<number><unit>`. `number` must be greater than `0`.
Unit is optional and can be `b` (bytes), `k` (kilobytes), `m`(megabytes), or `g` (gigabytes).
If you omit the unit, the system uses bytes. If you omit the size entirely, the system uses `64m`.

**--tls-verify** *bool-value*

Require HTTPS and verify certificates when talking to container registries (defaults to true)

**--ulimit** *type*=*soft-limit*[:*hard-limit*]

Specifies resource limits to apply to processes launched during `buildah run`.
This option can be specified multiple times.  Recognized resource types
include:
  "core": maximimum core dump size (ulimit -c)
  "cpu": maximum CPU time (ulimit -t)
  "data": maximum size of a process's data segment (ulimit -d)
  "fsize": maximum size of new files (ulimit -f)
  "locks": maximum number of file locks (ulimit -x)
  "memlock": maximum amount of locked memory (ulimit -l)
  "msgqueue": maximum amount of data in message queues (ulimit -q)
  "nice": niceness adjustment (nice -n, ulimit -e)
  "nofile": maximum number of open files (ulimit -n)
  "nofile": maximum number of open files (1048576); when run by root
  "nproc": maximum number of processes (ulimit -u)
  "nproc": maximum number of processes (1048576); when run by root
  "rss": maximum size of a process's (ulimit -m)
  "rtprio": maximum real-time scheduling priority (ulimit -r)
  "rttime": maximum amount of real-time execution between blocking syscalls
  "sigpending": maximum number of pending signals (ulimit -i)
  "stack": maximum stack size (ulimit -s)

**--userns** *how*

Sets the configuration for user namespaces when the container is subsequently
used for `buildah run`.
The configured value can be "" (the empty string) or "container" to indicate
that a new user namespace should be created, it can be "host" to indicate that
the user namespace in which `buildah` itself is being run should be reused, or
it can be the path to an user namespace which is already in use by another
process.

**--userns-uid-map** *mapping*

Directly specifies a UID mapping which should be used to set ownership, at the
filesytem level, on the container's contents.
Commands run using `buildah run` will default to being run in their own user
namespaces, configured using the UID and GID maps.

Entries in this map take the form of one or more triples of a starting
in-container UID, a corresponding starting host-level UID, and the number of
consecutive IDs which the map entry represents.

This option overrides the *remap-uids* setting in the *options* section of
/etc/containers/storage.conf.

If this option is not specified, but a global --userns-uid-map setting is
supplied, settings from the global option will be used.

If none of --userns-uid-map-user, --userns-gid-map-group, or --userns-uid-map
are specified, but --userns-gid-map is specified, the UID map will be set to
use the same numeric values as the GID map.

**--userns-gid-map** *mapping*

Directly specifies a GID mapping which should be used to set ownership, at the
filesytem level, on the container's contents.
Commands run using `buildah run` will default to being run in their own user
namespaces, configured using the UID and GID maps.

Entries in this map take the form of one or more triples of a starting
in-container GID, a corresponding starting host-level GID, and the number of
consecutive IDs which the map entry represents.

This option overrides the *remap-gids* setting in the *options* section of
/etc/containers/storage.conf.

If this option is not specified, but a global --userns-gid-map setting is
supplied, settings from the global option will be used.

If none of --userns-uid-map-user, --userns-gid-map-group, or --userns-gid-map
are specified, but --userns-uid-map is specified, the GID map will be set to
use the same numeric values as the UID map.

**--userns-uid-map-user** *user*

Specifies that a UID mapping which should be used to set ownership, at the
filesytem level, on the container's contents, can be found in entries in the
`/etc/subuid` file which correspond to the specified user.
Commands run using `buildah run` will default to being run in their own user
namespaces, configured using the UID and GID maps.
If --userns-gid-map-group is specified, but --userns-uid-map-user is not
specified, `buildah` will assume that the specified group name is also a
suitable user name to use as the default setting for this option.

**--userns-gid-map-group** *group*

Specifies that a GID mapping which should be used to set ownership, at the
filesytem level, on the container's contents, can be found in entries in the
`/etc/subgid` file which correspond to the specified group.
Commands run using `buildah run` will default to being run in their own user
namespaces, configured using the UID and GID maps.
If --userns-uid-map-user is specified, but --userns-gid-map-group is not
specified, `buildah` will assume that the specified user name is also a
suitable group name to use as the default setting for this option.

**--uts** *how*

Sets the configuration for UTS namespaces when the container is subsequently
used for `buildah run`.
The configured value can be "" (the empty string) or "container" to indicate
that a new UTS namespace should be created, or it can be "host" to indicate
that the UTS namespace in which `buildah` itself is being run should be reused,
or it can be the path to a UTS namespace which is already in use by another
process.

**--volume, -v**[=*[HOST-DIR:CONTAINER-DIR[:OPTIONS]]*]

   Create a bind mount. If you specify, ` -v /HOST-DIR:/CONTAINER-DIR`, Buildah
   bind mounts `/HOST-DIR` in the host to `/CONTAINER-DIR` in the Buildah
   container. The `OPTIONS` are a comma delimited list and can be:

   * [rw|ro]
   * [z|Z]
   * [`[r]shared`|`[r]slave`|`[r]private`|`[r]unbindable`]

The `CONTAINER-DIR` must be an absolute path such as `/src/docs`. The `HOST-DIR`
must be an absolute path as well. Buildah bind-mounts the `HOST-DIR` to the
path you specify. For example, if you supply `/foo` as the host path,
Buildah copies the contents of `/foo` to the container filesystem on the host
and bind mounts that into the container.

You can specify multiple  **-v** options to mount one or more mounts to a
container.

You can add the `:ro` or `:rw` suffix to a volume to mount it read-only or
read-write mode, respectively. By default, the volumes are mounted read-write.
See examples.

Labeling systems like SELinux require that proper labels are placed on volume
content mounted into a container. Without a label, the security system might
prevent the processes running inside the container from using the content. By
default, Buildah does not change the labels set by the OS.

To change a label in the container context, you can add either of two suffixes
`:z` or `:Z` to the volume mount. These suffixes tell Buildah to relabel file
objects on the shared volumes. The `z` option tells Buildah that two containers
share the volume content. As a result, Buildah labels the content with a shared
content label. Shared volume labels allow all containers to read/write content.
The `Z` option tells Buildah to label the content with a private unshared label.
Only the current container can use a private volume.

By default bind mounted volumes are `private`. That means any mounts done
inside container will not be visible on the host and vice versa. This behavior can
be changed by specifying a volume mount propagation property.

When the mount propagation policy is set to `shared`, any mounts completed inside
the container on that volume will be visible to both the host and container. When
the mount propagation policy is set to `slave`, one way mount propagation is enabled
and any mounts completed on the host for that volume will be visible only inside of the container.
To control the mount propagation property of the volume use the `:[r]shared`,
`:[r]slave`, `[r]private` or `[r]unbindable`propagation flag. The propagation property can
be specified only for bind mounted volumes and not for internal volumes or
named volumes. For mount propagation to work on the source mount point (the mount point
where source dir is mounted on) it has to have the right propagation properties. For
shared volumes, the source mount point has to be shared. And for slave volumes,
the source mount has to be either shared or slave.

Use `df <source-dir>` to determine the source mount and then use
`findmnt -o TARGET,PROPAGATION <source-mount-dir>` to determine propagation
properties of source mount, if `findmnt` utility is not available, the source mount point
can be determined by looking at the mount entry in `/proc/self/mountinfo`. Look
at `optional fields` and see if any propagaion properties are specified.
`shared:X` means the mount is `shared`, `master:X` means the mount is `slave` and if
nothing is there that means the mount is `private`.

To change propagation properties of a mount point use the `mount` command. For
example, to bind mount the source directory `/foo` do
`mount --bind /foo /foo` and `mount --make-private --make-shared /foo`. This
will convert /foo into a `shared` mount point.  The propagation properties of the source
mount can be changed directly. For instance if `/` is the source mount for
`/foo`, then use `mount --make-shared /` to convert `/` into a `shared` mount.

## EXAMPLE

buildah from --pull imagename

buildah from --pull docker://myregistry.example.com/imagename

buildah from docker-daemon:imagename:imagetag

buildah from --name mycontainer docker-archive:filename

buildah from oci-archive:filename

buildah from --name mycontainer dir:directoryname

buildah from --pull-always --name "mycontainer" docker://myregistry.example.com/imagename

buildah from --tls-verify=false myregistry/myrepository/imagename:imagetag

buildah from --creds=myusername:mypassword --cert-dir ~/auth myregistry/myrepository/imagename:imagetag

buildah from --authfile=/tmp/auths/myauths.json myregistry/myrepository/imagename:imagetag

buildah from --memory 40m --cpu-shares 2 --cpuset-cpus 0,2 --security-opt label=level:s0:c100,c200 myregistry/myrepository/imagename:imagetag

buildah from --ulimit nofile=1024:1028 --cgroup-parent /path/to/cgroup/parent myregistry/myrepository/imagename:imagetag

buildah from --volume /home/test:/myvol:ro,Z myregistry/myrepository/imagename:imagetag

## Files

**registries.conf** (`/etc/containers/registries.conf`)

registries.conf is the configuration file which specifies which container registries should be consulted when completing image names which do not include a registry or domain portion.

**policy.json** (`/etc/containers/policy.json`)

Signature policy file.  This defines the trust policy for container images.  Controls which container registries can be used for image, and whether or not the tool should trust the images.

## SEE ALSO
buildah(1), buildah-pull(1), buildah-login(1), docker-login(1), namespaces(7), pid\_namespaces(7), policy.json(5), registries.conf(5), user\_namespaces(7)
