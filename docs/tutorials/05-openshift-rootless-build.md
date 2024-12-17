![buildah logo](https://cdn.rawgit.com/containers/buildah/main/logos/buildah-logo_large.png)

# Buildah Tutorial 5
## Using Buildah to build images in a rootless OpenShift container

This tutorial will walk you through setting up a container in OpenShift for building images.

The instructions have been tested on OpenShift 4.16 with Buildah 1.36.0.

Note that we can use overlay for copy-on-write storage if we ensure that the storage used in
the builder container ends up in an emptyDir volume.

### Prepare a new namespace

Create a new project in OpenShift called `image-build`.

Make the registry URL available to the following steps.

*Note that you need to change this so it matches your OpenShift installation.*

````console
$ export REGISTRY_URL=default-route-openshift-image-registry.apps.whatever.com
````

Login to OpenShift and its registry:

````console
$ oc login -n image-build
Username: ...
Password: ...
Login successful.

You have access to N projects, the list has been suppressed. You can list all projects with 'oc projects'

Using project "image-build".

$ oc whoami -t | buildah login --tls-verify=false -u $(id -u -n) --password-stdin $REGISTRY_URL
Login Succeeded!
````


### Make builder image

This is the image that will host the building. It uses the Buildah stable official image, which is based on Fedora 35.

The image starts a python web server. This allows us to interact with the container via the OpenShift console terminal, demonstrating that building an image works.

First create an ImageStream to hold the image:

````console
$ oc create -f - <<EOF
apiVersion: image.openshift.io/v1
kind: ImageStream
metadata:
  name: buildah
EOF

imagestream.image.openshift.io/buildah created
````

Then create the image.

Note that no packages are updated - this should ensure that this tutorial is actually working.
If you are making anything for use in the real world, make sure to update it frequently for security fixes!

````console
$ cat > Containerfile-buildah <<EOF
FROM quay.io/buildah/stable:v1.36.0

RUN touch /etc/subgid /etc/subuid \
 && chmod g=u /etc/subgid /etc/subuid /etc/passwd \
 && echo build:10000:65536 > /etc/subuid \
 && echo build:10000:65536 > /etc/subgid

# Use chroot since the default runc does not work when running rootless
RUN echo "export BUILDAH_ISOLATION=chroot" >> /home/build/.bashrc

# Use overlay
RUN mkdir -p /home/build/.config/containers \
 && (echo '[storage]';echo 'driver = "overlay"') > /home/build/.config/containers/storage.conf

USER build
WORKDIR /home/build

# Just keep the container running, allowing "oc rsh" access
CMD ["python3", "-m", "http.server"]
EOF

$ buildah build --layers -t $REGISTRY_URL/image-build/buildah -f Containerfile-buildah
STEP 1/7: FROM quay.io/buildah/stable:v1.36.0
STEP 2/7: RUN touch /etc/subgid /etc/subuid  && chmod g=u /etc/subgid /etc/subuid /etc/passwd  && echo build:10000:65536 > /etc/subuid  && echo build:10000:65536 > /etc/subgid
--> e6ee6fcc2d94
STEP 3/7: RUN echo "export BUILDAH_ISOLATION=chroot" >> /home/build/.bashrc
--> 4327c8743bcc
STEP 4/7: RUN mkdir -p /home/build/.config/containers  && (echo '[storage]';echo 'driver = "overlay"') > /home/build/.config/containers/storage.conf
--> c405cbcd1132
STEP 5/7: USER build
--> 2c97c1162233
STEP 6/7: WORKDIR /home/build
--> 78d6367c298f
STEP 7/7: CMD ["python3", "-m", "http.server"]
COMMIT default-route-openshift-image-registry.apps-crc.testing/image-build/buildah
--> a872961c3fa9

$ buildah push --tls-verify=false $REGISTRY_URL/image-build/buildah
Getting image source signatures
...
Storing signatures
````


### Create Service Account for building images

Create a service account which is solely used for image building.

````console
$ oc create -f - <<EOF
apiVersion: v1
kind: ServiceAccount
metadata:
  name: buildah-sa
EOF

serviceaccount/buildah-sa created
````

You need to assign it the ability to run as the standard `anyuid` [SCC](https://docs.openshift.com/container-platform/4.3/authentication/managing-security-context-constraints.html).

````console
$ oc adm policy add-scc-to-user anyuid -z buildah-sa
clusterrole.rbac.authorization.k8s.io/system:openshift:scc:anyuid added: "buildah-sa"
````

This will give the container *cap_kill*, *cap_setgid*, and *cap_setuid* capabilities which are extras compared to the `restricted` SCC.
Note that *cap_kill* is dropped by the DeploymentConfig, but the two others are required to execute commands with different user ids as an image is built.


With this in place, when you get the Pod running (in a little while!), its YAML state will contain:

````
kind: Pod
metadata:
  ...
  openshift.io/scc: anyuid
...
````

Which tells you that the Pod has been launched with the correct permissions.


#### Create ReplicationController

This is a simple RC just to get the container running.

Note that it drops CAP_KILL which is not required.

````console
$ oc create -f - <<EOF
apiVersion: v1
kind: ReplicationController
metadata:
  name: buildah
spec:
  selector:
    app: image-builder
  replicas: 1
  template:
    metadata:
      labels:
        app: image-builder
    spec:
      serviceAccount: buildah-sa
      containers:
        - name: buildah
          image: image-registry.openshift-image-registry.svc:5000/image-build/buildah
          securityContext:
            capabilities:
              drop:
                - KILL
          volumeMounts:
            - name: containersstorage
              mountPath: /home/build/.local/share/containers/storage
      volumes:
        - name: containersstorage
          emptyDir:
            medium: ""
EOF

replicationcontroller/buildah created
````

#### The Buildah container

In the OpenShift console you can now open the Pod's terminal (or run `oc rsh rc/buildah` at the command line) and try building an image.

This is what the user/platform should look like:

````console
sh-5.0$ id
uid=1000(build) gid=1000(build) groups=1000(build)

sh-5.0$ uname -a
Linux buildah-vtwfs 5.14.0-427.22.1.el9_4.x86_64 #1 SMP PREEMPT_DYNAMIC Mon Jun 10 09:23:36 EDT 2024 x86_64 GNU/Linux

sh-5.0$ capsh --print
Current: =
Bounding set =cap_chown,cap_dac_override,cap_fowner,cap_fsetid,cap_setgid,cap_setuid,cap_setpcap,cap_net_bind_service
Ambient set =
Current IAB: !cap_dac_read_search,!cap_kill,!cap_linux_immutable,!cap_net_broadcast,!cap_net_admin,!cap_net_raw,!cap_ipc_lock,!cap_ipc_owner,!cap_sys_module,!cap_sys_rawio,!cap_sys_chroot,!cap_sys_ptrace,!cap_sys_pacct,!cap_sys_admin,!cap_sys_boot,!cap_sys_nice,!cap_sys_resource,!cap_sys_time,!cap_sys_tty_config,!cap_mknod,!cap_lease,!cap_audit_write,!cap_audit_control,!cap_setfcap,!cap_mac_override,!cap_mac_admin,!cap_syslog,!cap_wake_alarm,!cap_block_suspend,!cap_audit_read,!cap_perfmon,!cap_bpf,!cap_checkpoint_restore
Securebits: 00/0x0/1'b0 (no-new-privs=0)
 secure-noroot: no (unlocked)
 secure-no-suid-fixup: no (unlocked)
 secure-keep-caps: no (unlocked)
 secure-no-ambient-raise: no (unlocked)
uid=1000(build) euid=1000(build)
gid=1000(build)
groups=1000(build)
Guessed mode: HYBRID (4)
````

This is what the Buildah data should look like:

````console
sh-5.0$ buildah version
Version:         1.36.0
Go Version:      go1.22.3
Image Spec:      1.1.0
Runtime Spec:    1.2.0
CNI Spec:        1.0.0
libcni Version:
image Version:   5.31.0
Git Commit:
Built:           Mon May 27 13:11:54 2024
OS/Arch:         linux/amd64
BuildPlatform:   linux/amd64

sh-5.0$ buildah info
{
    "host": {
        "CgroupVersion": "v2",
        "Distribution": {
            "distribution": "fedora",
            "version": "40"
        },
        "MemFree": 570695680,
        "MemTotal": 10916950016,
        "OCIRuntime": "crun",
        "SwapFree": 0,
        "SwapTotal": 0,
        "arch": "amd64",
        "cpus": 4,
        "hostname": "buildah-hgdcd-debug-dsvsf",
        "kernel": "5.14.0-427.22.1.el9_4.x86_64",
        "os": "linux",
        "rootless": true,
        "uptime": "1h 6m 11.06s (Approximately 0.04 days)",
        "variant": ""
    },
    "store": {
        "ContainerStore": {
            "number": 0
        },
        "GraphDriverName": "overlay",
        "GraphOptions": null,
        "GraphRoot": "/home/build/.local/share/containers/storage",
        "GraphStatus": {
            "Backing Filesystem": "xfs",
            "Native Overlay Diff": "true",
            "Supports d_type": "true",
            "Supports shifting": "false",
            "Supports volatile": "true",
            "Using metacopy": "false"
        },
        "ImageStore": {
            "number": 0
        },
        "RunRoot": "/var/tmp/storage-run-1000/containers"
    }
}
````

#### Building an image

Now create some files for testing.

This container test file exercises at least some of the critical parts of building an image (package update/installation, execution of commands, and use of volumes).

````console
sh-5.0$ cat > test-script.sh <<EOF
#/bin/bash
echo "Args \$*"
ls -l /
EOF

sh-5.0$ chmod +x test-script.sh

sh-5.0$ cat > Containerfile.test <<EOF
FROM fedora:40
RUN ls -l /test-script.sh
RUN /test-script.sh "Hello world"
RUN dnf update -y | tee /output/update-output.txt
RUN dnf install -y gcc
EOF

sh-5.0$ mkdir output
````

And finally build the image, testing that everything works as expected:

````console
sh-5.0$ buildah build --layers -v /home/build/output:/output:rw -v /home/build/test-script.sh:/test-script.sh:ro -t myimage -f Containerfile.test
FROM fedora:40
RUN ls -l /test-script.sh
RUN /test-script.sh "Hello world"
RUN dnf update -y | tee /output/update-output.txt
RUN dnf install -y gcc
EOF
sh-5.1$ mkdir output
sh-5.1$ buildah -v /home/build/output:/output:rw -v /home/build/test-script.sh:/test-script.sh:ro build-using-dockerfile -t myimage -f Containerfile.test
STEP 1/5: FROM fedora:40
Resolved "fedora" as an alias (/etc/containers/registries.conf.d/000-shortnames.conf)
Trying to pull registry.fedoraproject.org/fedora:40...
Getting image source signatures
Copying blob 6d5785fdf371 done   |
Copying config b8638217aa done   |
Writing manifest to image destination
STEP 2/5: RUN ls -l /test-script.sh
-rwxr-xr-x. 1 root root 34 Aug  5 18:33 /test-script.sh
--> a73b603bca4d
STEP 3/5: RUN /test-script.sh "Hello world"
Args Hello world
total 8
dr-xr-xr-x.   2 root   root      6 Jan 24  2024 afs
lrwxrwxrwx.   1 root   root      7 Jan 24  2024 bin -> usr/bin
dr-xr-xr-x.   2 root   root      6 Jan 24  2024 boot
drwxr-xr-x.   5 nobody nobody  380 Aug  5 18:32 dev
drwxr-xr-x.   1 root   root     41 Aug  5 18:35 etc
drwxr-xr-x.   2 root   root      6 Jan 24  2024 home
lrwxrwxrwx.   1 root   root      7 Jan 24  2024 lib -> usr/lib
lrwxrwxrwx.   1 root   root      9 Jan 24  2024 lib64 -> usr/lib64
drwxr-xr-x.   2 root   root      6 Jan 24  2024 media
drwxr-xr-x.   2 root   root      6 Jan 24  2024 mnt
drwxr-xr-x.   2 root   root      6 Jan 24  2024 opt
drwxr-xr-x.   2 root   root      6 Aug  5 18:34 output
dr-xr-xr-x. 465 nobody nobody    0 Aug  5 18:32 proc
dr-xr-x---.   2 root   root     91 Aug  5 05:47 root
drwxr-xr-x.   1 root   root     42 Aug  5 18:35 run
lrwxrwxrwx.   1 root   root      8 Jan 24  2024 sbin -> usr/sbin
drwxr-xr-x.   2 root   root      6 Jan 24  2024 srv
dr-xr-xr-x.  13 nobody nobody    0 Aug  5 17:26 sys
-rwxr-xr-x.   1 root   root     34 Aug  5 18:33 test-script.sh
drwxrwxrwt.   2 root   root      6 Jan 24  2024 tmp
drwxr-xr-x.  12 root   root    144 Aug  5 05:47 usr
drwxr-xr-x.  18 root   root   4096 Aug  5 05:47 var
--> 3a1192fe0ecf
STEP 4/5: RUN dnf update -y | tee /output/update-output.txt
Fedora 40 - x86_64                              9.4 MB/s |  20 MB     00:02
Fedora 40 openh264 (From Cisco) - x86_64        3.5 kB/s | 1.4 kB     00:00
Fedora 40 - x86_64 - Updates                    9.9 MB/s | 9.1 MB     00:00
Dependencies resolved.
Nothing to do.
Complete!
--> 5026f20f0ad1
STEP 5/5: RUN dnf install -y gcc
Last metadata expiration check: 0:00:06 ago on Mon Aug  5 18:35:25 2024.
Dependencies resolved.
================================================================================
 Package                        Arch       Version            Repository   Size
================================================================================
Installing:
 gcc                            x86_64     14.2.1-1.fc40      updates      37 M
Installing dependencies:
 binutils                       x86_64     2.41-37.fc40       updates     6.2 M
 binutils-gold                  x86_64     2.41-37.fc40       updates     781 k
 cpp                            x86_64     14.2.1-1.fc40      updates      12 M
 elfutils-debuginfod-client     x86_64     0.191-4.fc40       fedora       38 k
 gc                             x86_64     8.2.2-6.fc40       fedora      110 k
 glibc-devel                    x86_64     2.39-17.fc40       updates     114 k
 glibc-headers-x86              noarch     2.39-17.fc40       updates     608 k
 guile30                        x86_64     3.0.7-12.fc40      fedora      8.1 M
 jansson                        x86_64     2.13.1-9.fc40      fedora       44 k
 kernel-headers                 x86_64     6.9.4-200.fc40     updates     1.6 M
 libmpc                         x86_64     1.3.1-5.fc40       fedora       71 k
 libpkgconf                     x86_64     2.1.1-1.fc40       updates      38 k
 libxcrypt-devel                x86_64     4.4.36-5.fc40      fedora       29 k
 make                           x86_64     1:4.4.1-6.fc40     fedora      588 k
 pkgconf                        x86_64     2.1.1-1.fc40       updates      44 k
 pkgconf-m4                     noarch     2.1.1-1.fc40       updates      14 k
 pkgconf-pkg-config             x86_64     2.1.1-1.fc40       updates     9.9 k

Transaction Summary
================================================================================
Install  18 Packages

Total download size: 67 M
Installed size: 230 M
Downloading Packages:
(1/18): elfutils-debuginfod-client-0.191-4.fc40  57 kB/s |  38 kB     00:00
(2/18): gc-8.2.2-6.fc40.x86_64.rpm              146 kB/s | 110 kB     00:00
(3/18): jansson-2.13.1-9.fc40.x86_64.rpm        254 kB/s |  44 kB     00:00
(4/18): libmpc-1.3.1-5.fc40.x86_64.rpm          420 kB/s |  71 kB     00:00
(5/18): libxcrypt-devel-4.4.36-5.fc40.x86_64.rp 336 kB/s |  29 kB     00:00
(6/18): make-4.4.1-6.fc40.x86_64.rpm            739 kB/s | 588 kB     00:00
(7/18): binutils-2.41-37.fc40.x86_64.rpm        6.4 MB/s | 6.2 MB     00:00
(8/18): cpp-14.2.1-1.fc40.x86_64.rpm             28 MB/s |  12 MB     00:00
(9/18): binutils-gold-2.41-37.fc40.x86_64.rpm   732 kB/s | 781 kB     00:01
(10/18): glibc-devel-2.39-17.fc40.x86_64.rpm    982 kB/s | 114 kB     00:00
(11/18): glibc-headers-x86-2.39-17.fc40.noarch. 3.1 MB/s | 608 kB     00:00
(12/18): kernel-headers-6.9.4-200.fc40.x86_64.r 4.1 MB/s | 1.6 MB     00:00
(13/18): libpkgconf-2.1.1-1.fc40.x86_64.rpm     376 kB/s |  38 kB     00:00
(14/18): gcc-14.2.1-1.fc40.x86_64.rpm            27 MB/s |  37 MB     00:01
(15/18): pkgconf-2.1.1-1.fc40.x86_64.rpm        403 kB/s |  44 kB     00:00
(16/18): pkgconf-m4-2.1.1-1.fc40.noarch.rpm     180 kB/s |  14 kB     00:00
(17/18): pkgconf-pkg-config-2.1.1-1.fc40.x86_64 120 kB/s | 9.9 kB     00:00
(18/18): guile30-3.0.7-12.fc40.x86_64.rpm       685 kB/s | 8.1 MB     00:12
--------------------------------------------------------------------------------
Total                                           5.4 MB/s |  67 MB     00:12
Running transaction check
Transaction check succeeded.
Running transaction test
Transaction test succeeded.
Running transaction
  Preparing        :                                                        1/1
  Installing       : libmpc-1.3.1-5.fc40.x86_64                            1/18
  Installing       : jansson-2.13.1-9.fc40.x86_64                          2/18
  Installing       : cpp-14.2.1-1.fc40.x86_64                              3/18
  Installing       : pkgconf-m4-2.1.1-1.fc40.noarch                        4/18
  Installing       : libpkgconf-2.1.1-1.fc40.x86_64                        5/18
  Installing       : pkgconf-2.1.1-1.fc40.x86_64                           6/18
  Installing       : pkgconf-pkg-config-2.1.1-1.fc40.x86_64                7/18
  Installing       : kernel-headers-6.9.4-200.fc40.x86_64                  8/18
  Installing       : glibc-headers-x86-2.39-17.fc40.noarch                 9/18
  Installing       : glibc-devel-2.39-17.fc40.x86_64                      10/18
  Installing       : libxcrypt-devel-4.4.36-5.fc40.x86_64                 11/18
  Installing       : gc-8.2.2-6.fc40.x86_64                               12/18
  Installing       : guile30-3.0.7-12.fc40.x86_64                         13/18
  Installing       : make-1:4.4.1-6.fc40.x86_64                           14/18
  Installing       : elfutils-debuginfod-client-0.191-4.fc40.x86_64       15/18
  Installing       : binutils-gold-2.41-37.fc40.x86_64                    16/18
  Running scriptlet: binutils-gold-2.41-37.fc40.x86_64                    16/18
  Installing       : binutils-2.41-37.fc40.x86_64                         17/18
  Running scriptlet: binutils-2.41-37.fc40.x86_64                         17/18
  Installing       : gcc-14.2.1-1.fc40.x86_64                             18/18
  Running scriptlet: gcc-14.2.1-1.fc40.x86_64                             18/18

Installed:
  binutils-2.41-37.fc40.x86_64
  binutils-gold-2.41-37.fc40.x86_64
  cpp-14.2.1-1.fc40.x86_64
  elfutils-debuginfod-client-0.191-4.fc40.x86_64
  gc-8.2.2-6.fc40.x86_64
  gcc-14.2.1-1.fc40.x86_64
  glibc-devel-2.39-17.fc40.x86_64
  glibc-headers-x86-2.39-17.fc40.noarch
  guile30-3.0.7-12.fc40.x86_64
  jansson-2.13.1-9.fc40.x86_64
  kernel-headers-6.9.4-200.fc40.x86_64
  libmpc-1.3.1-5.fc40.x86_64
  libpkgconf-2.1.1-1.fc40.x86_64
  libxcrypt-devel-4.4.36-5.fc40.x86_64
  make-1:4.4.1-6.fc40.x86_64
  pkgconf-2.1.1-1.fc40.x86_64
  pkgconf-m4-2.1.1-1.fc40.noarch
  pkgconf-pkg-config-2.1.1-1.fc40.x86_64

Complete!
COMMIT myimage
--> 087a83c62eb7
Successfully tagged localhost/myimage:latest
087a83c62eb73def45ef4542460c18b897a8d018299f15f69a2a6b678d56fcec

sh-5.0$ buildah images
REPOSITORY                          TAG      IMAGE ID       CREATED          SIZE
localhost/myimage                   latest   087a83c62eb7   45 seconds ago   533 MB
registry.fedoraproject.org/fedora   40       b8638217aa4e   13 hours ago     233 MB

sh-5.0$ ls -l output/
total 4
-rw-r--r--. 1 build build 288 Aug  5 18:35 update-output.txt
````
