# buildah "1" "March 2017" "buildah"

## NAME
Buildah - A command line tool that facilitates building OCI container images.

## SYNOPSIS
buildah [OPTIONS] COMMAND [ARG...]


## DESCRIPTION
The Buildah package provides a command line tool which can be used to:

    * Create a working container, either from scratch or using an image as a starting point.
    * Mount a working container's root filesystem for manipulation.
    * Unmount a working container's root filesystem.
    * Use the updated contents of a container's root filesystem as a filesystem layer to create a new image.
    * Delete a working container or an image.
    * Rename a local container.


## OPTIONS

**--debug**

Print debugging information

**--help, -h**

Show help

**--registries-conf** *path*

Pathname of the configuration file which specifies which container registries should be
consulted when completing image names which do not include a registry or domain
portion.  It is not recommended that this option be used, as the default
behavior of using the system-wide configuration
(*/etc/containers/registries.conf*) is most often preferred.

**--registries-conf-dir** *path*

Pathname of the directory which contains configuration snippets which specify
registries which should be consulted when completing image names which do not
include a registry or domain portion.  It is not recommended that this option
be used, as the default behavior of using the system-wide configuration
(*/etc/containers/registries.d*) is most often preferred.

**--root** **value**

Storage root dir (default: "/var/lib/containers/storage" for UID 0, "$HOME/.local/share/containers/storage" for other users)
Default root dir is configured in /etc/containers/storage.conf

**--runroot** **value**

Storage state dir (default: "/var/run/containers/storage" for UID 0, "/var/run/user/$UID" for other users)
Default state dir is configured in /etc/containers/storage.conf

**--storage-driver** **value**

Storage driver.  The default storage driver for UID 0 is configured in /etc/containers/storage.conf (`$HOME/.config/containers/storage.conf` in rootless mode), and is *vfs* for other users.  The `STORAGE_DRIVER` environment variable overrides the default.  The --storage-driver specified driver overrides all.

Examples: "overlay", "devicemapper", "vfs"

Overriding this option will cause the *storage-opt* settings in /etc/containers/storage.conf to be ignored.  The user must
specify additional options via the `--storage-opt` flag.

**--storage-opt** **value**

Storage driver option, Default storage driver options are configured in /etc/containers/storage.conf (`$HOME/.config/containers/storage.conf` in rootless mode). The `STORAGE_OPTS` environment variable overrides the default. The --storage-opt specified options overrides all.

**--userns-uid-map** *mapping*

Specifies UID mappings which should be used to set ownership, at the
filesytem level, on the contents of images and containers.
Entries in this map take the form of one or more triples of a starting
in-container UID, a corresponding starting host-level UID, and the number of
consecutive IDs which the map entry represents.
This option overrides the *remap-uids* setting in the *options* section of
/etc/containers/storage.conf.

**--userns-gid-map** *mapping*

Specifies GID mappings which should be used to set ownership, at the
filesytem level, on the contents of images and containers.
Entries in this map take the form of one or more triples of a starting
in-container GID, a corresponding starting host-level GID, and the number of
consecutive IDs which the map entry represents.
This option overrides the *remap-gids* setting in the *options* section of
/etc/containers/storage.conf.

**--version, -v**

Print the version

## COMMANDS

| Command               | Description                                                                                          |
| --------------------- | ---------------------------------------------------                                                  |
| buildah-add(1)        | Add the contents of a file, URL, or a directory to the container.                                    |
| buildah-bud(1)        | Build an image using instructions from Dockerfiles.                                                  |
| buildah-commit(1)     | Create an image from a working container.                                                            |
| buildah-config(1)     | Update image configuration settings.                                                                 |
| buildah-containers(1) | List the working containers and their base images.                                                   |
| buildah-copy(1)       | Copies the contents of a file, URL, or directory into a container's working directory.               |
| buildah-from(1)       | Creates a new working container, either from scratch or using a specified image as a starting point. |
| buildah-images(1)     | List images in local storage.                                                                        |
| buildah-info(1)       | Display Buildah system information.                                                                  |
| buildah-inspect(1)    | Inspects the configuration of a container or image                                                   |
| buildah-mount(1)      | Mount the working container's root filesystem.                                                       |
| buildah-login(1)      | Login to a container registry.                                                                       |
| buildah-logout(1)     | Logout of a container registry                                                                       |
| buildah-pull(1)       | Pull an image from the specified location.                                                           |
| buildah-push(1)       | Push an image from local storage to elsewhere.                                                       |
| buildah-rename(1)     | Rename a local container.                                                                            |
| buildah-rm(1)         | Removes one or more working containers.                                                              |
| buildah-rmi(1)        | Removes one or more images.                                                                          |
| buildah-run(1)        | Run a command inside of the container.                                                               |
| buildah-tag(1)        | Add an additional name to a local image.                                                             |
| buildah-umount(1)     | Unmount a working container's root file system.                                                      |
| buildah-unshare(1)    | Launch a command in a user namespace with modified ID mappings.                                      |
| buildah-version(1)    | Display the Buildah Version Information                                                              |
| storage.conf(5)       | Syntax of Container Storage configuration file                                                       |


## Files

**storage.conf** (`/etc/containers/storage.conf`)

storage.conf is the storage configuration file for all tools using containers/storage

The storage configuration file specifies all of the available container storage options for tools using shared container storage.

**mounts.conf** (`/usr/share/containers/mounts.conf` and optionally `/etc/containers/mounts.conf`)

The mounts.conf files specify volume mount directories that are automatically mounted inside containers when executing the `buildah run` or `buildah build-using-dockerfile` commands.  Container process can then use this content.  The volume mount content does not get committed to the final image.

Usually these directories are used for passing secrets or credentials required by the package software to access remote package repositories.

For example, a mounts.conf with the line "`/usr/share/rhel/secrets:/run/secrets`", the content of `/usr/share/rhel/secrets` directory is mounted on `/run/secrets` inside the container.  This mountpoint allows Red Hat Enterprise Linux subscriptions from the host to be used within the container.

Note this is not a volume mount. The content of the volumes is copied into container storage, not bind mounted directly from the host.

**registries.conf** (`/etc/containers/registries.conf`)

registries.conf is the configuration file which specifies which container registries should be consulted when completing image names which do not include a registry or domain portion.

**registries.d** (`/etc/containers/registries.d`)

Directory which contains configuration snippets which specify registries which should be consulted when completing image names which do not include a registry or domain portion.

## SEE ALSO
podman(1), containers-mounts.conf(5), newuidmap(1), newgidmap(1), containers-registries.conf(5), containers-storage.conf(5)

## HISTORY
December 2017, Originally compiled by Tom Sweeney <tsweeney@redhat.com>
