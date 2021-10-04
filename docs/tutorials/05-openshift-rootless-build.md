![buildah logo](https://cdn.rawgit.com/containers/buildah/main/logos/buildah-logo_large.png)

# Buildah Tutorial 5
## Using Buildah to build images in a rootless OpenShift container

This tutorial will walk you through setting up a container in OpenShift for building images.

The instructions have been tested on OpenShift 4.9.5 with Buildah 1.23.1.

Note that the VFS is used for storage instead of the more performant fuse-overlayfs or overlayfs. But the the latter do not work at the moment.

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

$ oc whoami -t | buildah login -u $(id -u -n) --password-stdin $REGISTRY_URL
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
FROM quay.io/buildah/stable:v1.23.1

RUN touch /etc/subgid /etc/subuid \
 && chmod g=u /etc/subgid /etc/subuid /etc/passwd \
 && echo build:10000:65536 > /etc/subuid \
 && echo build:10000:65536 > /etc/subgid

# Use chroot since the default runc does not work when running rootless
RUN echo "export BUILDAH_ISOLATION=chroot" >> /home/build/.bashrc

# Use VFS since fuse does not work
RUN mkdir -p /home/build/.config/containers \
 && (echo '[storage]';echo 'driver = "vfs"') > /home/build/.config/containers/storage.conf

USER build
WORKDIR /home/build

# Just keep the container running, allowing "oc rsh" access
CMD ["python3", "-m", "http.server"]
EOF

$ buildah build -t $REGISTRY_URL/image-build/buildah -f Containerfile-buildah
STEP 1: FROM quay.io/buildah/stable:v1.23.1
STEP 2: RUN touch /etc/subgid /etc/subuid  && chmod g=u /etc/subgid /etc/subuid /etc/passwd  && echo build:10000:65536 > /etc/subuid  && echo build:10000:65536 > /etc/subgid
--> a25dbbd3824
STEP 3: CMD ["python3", "-m", "http.server"]
STEP 4: COMMIT default-route-openshift-image-registry.../image-build/buildah
--> 9656f2677e3
9656f2677e3e760e071c93ca7cba116871f5549b28ad8595e9134679db2345fc

$ buildah push $REGISTRY_URL/image-build/buildah
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


#### Create DeploymentConfig

This is a simple DC just to get the container running.

Note that it drops CAP_KILL which is not required.

````console
$ oc create -f - <<EOF
apiVersion: apps.openshift.io/v1
kind: DeploymentConfig
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
EOF

deploymentconfig.apps.openshift.io/buildah created
````

#### The Buildah container

In the OpenShift console you can now open the Pod's Terminal and try building an image.

This is what the user/platform should look like:

````console
sh-5.0$ id
uid=1000(build) gid=1000(build) groups=1000(build)

sh-5.0$ uname -a
Linux buildah-1-8t74l 4.18.0-147.13.2.el8_1.x86_64 #1 SMP Wed May 13 15:19:35 UTC 2020 x86_64 x86_64 x86_64 GNU/Linux

sh-5.0$ capsh --print
Current: = cap_chown,cap_dac_override,cap_fowner,cap_fsetid,cap_setgid,cap_setuid,cap_setpcap,cap_net_bind_service=i
Bounding set =cap_chown,cap_dac_override,cap_fowner,cap_fsetid,cap_setgid,cap_setuid,cap_setpcap,cap_net_bind_service
Ambient set =
Current IAB: cap_chown,cap_dac_override,!cap_dac_read_search,cap_fowner,cap_fsetid,!cap_kill,cap_setgid,cap_setuid,cap_setpcap,!cap_linux_immutable,cap_net_bind_service,!cap_net_broadcast,!cap_net_admin,!cap_net_raw,!cap_ipc_lock,!cap_ipc_owner,!cap_sys_module,!cap_sys_rawio,!cap_sys_chroot,!cap_sys_ptrace,!cap_sys_pacct,!cap_sys_admin,!cap_sys_boot,!cap_sys_nice,!cap_sys_resource,!cap_sys_time,!cap_sys_tty_config,!cap_mknod,!cap_lease,!cap_audit_write,!cap_audit_control,!cap_setfcap,!cap_mac_override,!cap_mac_admin,!cap_syslog,!cap_wake_alarm,!cap_block_suspend,!cap_audit_read,!cap_perfmon,!cap_bpf
Securebits: 00/0x0/1'b0 (no-new-privs=0)
 secure-noroot: no (unlocked)
 secure-no-suid-fixup: no (unlocked)
 secure-keep-caps: no (unlocked)
 secure-no-ambient-raise: no (unlocked)
uid=1000(build)
gid=1000(build)
groups=
Guessed mode: UNCERTAIN (0)
````

This is what the Buildah data should look like:

````console
sh-5.0$ buildah version
Version:         1.23.1
Go Version:      go1.16.8
Image Spec:      1.0.1-dev
Runtime Spec:    1.0.2-dev
CNI Spec:        0.4.0
libcni Version:  v0.8.1
image Version:   5.16.0
Git Commit:
Built:           Tue Sep 28 18:26:37 2021
OS/Arch:         linux/amd64
BuildPlatform:   linux/amd64

sh-5.0$ buildah info
{
    "host": {
        "CgroupVersion": "v1",
        "Distribution": {
            "distribution": "fedora",
            "version": "35"
        },
        "MemTotal": 33726861312,
        "MenFree": 20319305728,
        "OCIRuntime": "crun",
        "SwapFree": 0,
        "SwapTotal": 0,
        "arch": "amd64",
        "cpus": 4,
        "hostname": "buildah-1-6hvsw",
        "kernel": "4.18.0-305.19.1.el8_4.x86_64",
        "os": "linux",
        "rootless": true,
        "uptime": "61h 10m 39.3s (Approximately 2.54 days)"
    },
    "store": {
        "ContainerStore": {
            "number": 0
        },
        "GraphDriverName": "vfs",
        "GraphOptions": null,
        "GraphRoot": "/home/build/.local/share/containers/storage",
        "GraphStatus": {},
        "ImageStore": {
            "number": 0
        },
        "RunRoot": "/var/tmp/containers-user-1000/containers"
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
FROM fedora:35
RUN ls -l /test-script.sh
RUN /test-script.sh "Hello world"
RUN dnf update -y | tee /output/update-output.txt
RUN dnf install -y gcc
EOF

sh-5.0$ mkdir output
````

And finally build the image, testing that everything works as expected:

````console
sh-5.0$ buildah -v /home/build/output:/output:rw -v /home/build/test-script.sh:/test-script.sh:ro build-using-dockerfile -t myimage -f Containerfile.test
FROM fedora:35
RUN ls -l /test-script.sh
RUN /test-script.sh "Hello world"
RUN dnf update -y | tee /output/update-output.txt
RUN dnf install -y gcc
EOF
sh-5.1$ mkdir output
sh-5.1$ buildah -v /home/build/output:/output:rw -v /home/build/test-script.sh:/test-script.sh:ro build-using-dockerfile -t myimage -f Containerfile.test
STEP 1/5: FROM fedora:35
Resolved "fedora" as an alias (/etc/containers/registries.conf.d/000-shortnames.conf)
Trying to pull registry.fedoraproject.org/fedora:35...
Getting image source signatures
Copying blob 791199e77b3d done
Copying config 1b52edb081 done
Writing manifest to image destination
Storing signatures
STEP 2/5: RUN ls -l /test-script.sh
-rwxr-xr-x. 1 root root 34 Nov 12 21:20 /test-script.sh
STEP 3/5: RUN /test-script.sh "Hello world"
Args Hello world
total 8
lrwxrwxrwx.   1 root   root      7 Jul 21 23:47 bin -> usr/bin
dr-xr-xr-x.   2 root   root      6 Jul 21 23:47 boot
drwxr-xr-x.   5 nobody nobody  360 Nov 12 21:17 dev
drwxr-xr-x.  42 root   root   4096 Nov  3 16:38 etc
drwxr-xr-x.   2 root   root      6 Jul 21 23:47 home
lrwxrwxrwx.   1 root   root      7 Jul 21 23:47 lib -> usr/lib
lrwxrwxrwx.   1 root   root      9 Jul 21 23:47 lib64 -> usr/lib64
drwx------.   2 root   root      6 Nov  3 16:37 lost+found
drwxr-xr-x.   2 root   root      6 Jul 21 23:47 media
drwxr-xr-x.   2 root   root      6 Jul 21 23:47 mnt
drwxr-xr-x.   2 root   root      6 Jul 21 23:47 opt
drwxr-xr-x.   2 root   root      6 Nov 12 21:21 output
dr-xr-xr-x. 352 nobody nobody    0 Nov 12 21:17 proc
dr-xr-x---.   2 root   root    196 Nov  3 16:38 root
drwxr-xr-x.   3 root   root     42 Nov 12 21:21 run
lrwxrwxrwx.   1 root   root      8 Jul 21 23:47 sbin -> usr/sbin
drwxr-xr-x.   2 root   root      6 Jul 21 23:47 srv
dr-xr-xr-x.  13 nobody nobody    0 Nov 12 20:27 sys
-rwxr-xr-x.   1 root   root     34 Nov 12 21:20 test-script.sh
drwxrwxrwt.   2 root   root      6 Nov  3 16:37 tmp
drwxr-xr-x.  12 root   root    144 Nov  3 16:38 usr
drwxr-xr-x.  18 root   root    235 Nov  3 16:38 var
STEP 4/5: RUN dnf update -y | tee /output/update-output.txt
Fedora 35 - x86_64                              7.1 MB/s |  61 MB     00:08
Fedora 35 openh264 (From Cisco) - x86_64        4.1 kB/s | 2.5 kB     00:00
Fedora Modular 35 - x86_64                      3.1 MB/s | 2.6 MB     00:00
Fedora 35 - x86_64 - Updates                    5.6 MB/s |  10 MB     00:01
Fedora Modular 35 - x86_64 - Updates            763 kB/s | 712 kB     00:00
Last metadata expiration check: 0:00:01 ago on Fri Nov 12 21:22:21 2021.
Dependencies resolved.
================================================================================
 Package                    Arch       Version                Repository   Size
================================================================================
Upgrading:
 glib2                      x86_64     2.70.1-1.fc35          updates     2.6 M
 glibc                      x86_64     2.34-8.fc35            updates     2.0 M
 glibc-common               x86_64     2.34-8.fc35            updates     406 k
 glibc-minimal-langpack     x86_64     2.34-8.fc35            updates     134 k
 gpgme                      x86_64     1.15.1-6.fc35          updates     206 k
 libgpg-error               x86_64     1.43-1.fc35            updates     216 k
 python3-gpg                x86_64     1.15.1-6.fc35          updates     261 k
 shadow-utils               x86_64     2:4.9-5.fc35           updates     1.1 M
 vim-minimal                x86_64     2:8.2.3582-1.fc35      updates     706 k
Installing weak dependencies:
 glibc-gconv-extra          x86_64     2.34-8.fc35            updates     1.6 M

Transaction Summary
================================================================================
Install  1 Package
Upgrade  9 Packages

Total download size: 9.3 M
Downloading Packages:
(1/10): glibc-2.34-8.fc35.x86_64.rpm            5.2 MB/s | 2.0 MB     00:00
(2/10): glibc-gconv-extra-2.34-8.fc35.x86_64.rp 3.9 MB/s | 1.6 MB     00:00
(3/10): glib2-2.70.1-1.fc35.x86_64.rpm          5.7 MB/s | 2.6 MB     00:00
(4/10): glibc-minimal-langpack-2.34-8.fc35.x86_ 2.1 MB/s | 134 kB     00:00
(5/10): glibc-common-2.34-8.fc35.x86_64.rpm     3.9 MB/s | 406 kB     00:00
(6/10): gpgme-1.15.1-6.fc35.x86_64.rpm          4.6 MB/s | 206 kB     00:00
(7/10): libgpg-error-1.43-1.fc35.x86_64.rpm     5.4 MB/s | 216 kB     00:00
(8/10): python3-gpg-1.15.1-6.fc35.x86_64.rpm    5.6 MB/s | 261 kB     00:00
(9/10): shadow-utils-4.9-5.fc35.x86_64.rpm       14 MB/s | 1.1 MB     00:00
(10/10): vim-minimal-8.2.3582-1.fc35.x86_64.rpm 8.2 MB/s | 706 kB     00:00
--------------------------------------------------------------------------------
Total                                           9.4 MB/s | 9.3 MB     00:00
Running transaction check
Transaction check succeeded.
Running transaction test
Transaction test succeeded.
Running transaction
  Preparing        :                                                        1/1
  Upgrading        : glibc-common-2.34-8.fc35.x86_64                       1/19
  Upgrading        : glibc-minimal-langpack-2.34-8.fc35.x86_64             2/19
  Running scriptlet: glibc-2.34-8.fc35.x86_64                              3/19
  Upgrading        : glibc-2.34-8.fc35.x86_64                              3/19
  Running scriptlet: glibc-2.34-8.fc35.x86_64                              3/19
  Installing       : glibc-gconv-extra-2.34-8.fc35.x86_64                  4/19
  Running scriptlet: glibc-gconv-extra-2.34-8.fc35.x86_64                  4/19
  Upgrading        : libgpg-error-1.43-1.fc35.x86_64                       5/19
  Upgrading        : gpgme-1.15.1-6.fc35.x86_64                            6/19
  Upgrading        : python3-gpg-1.15.1-6.fc35.x86_64                      7/19
  Upgrading        : glib2-2.70.1-1.fc35.x86_64                            8/19
  Upgrading        : shadow-utils-2:4.9-5.fc35.x86_64                      9/19
  Upgrading        : vim-minimal-2:8.2.3582-1.fc35.x86_64                 10/19
  Cleanup          : glib2-2.70.0-5.fc35.x86_64                           11/19
  Cleanup          : shadow-utils-2:4.9-3.fc35.x86_64                     12/19
  Cleanup          : python3-gpg-1.15.1-4.fc35.x86_64                     13/19
  Cleanup          : gpgme-1.15.1-4.fc35.x86_64                           14/19
  Cleanup          : vim-minimal-2:8.2.3568-1.fc35.x86_64                 15/19
  Cleanup          : libgpg-error-1.42-3.fc35.x86_64                      16/19
  Cleanup          : glibc-2.34-7.fc35.x86_64                             17/19
  Cleanup          : glibc-minimal-langpack-2.34-7.fc35.x86_64            18/19
  Cleanup          : glibc-common-2.34-7.fc35.x86_64                      19/19
  Running scriptlet: glibc-common-2.34-7.fc35.x86_64                      19/19
  Verifying        : glibc-gconv-extra-2.34-8.fc35.x86_64                  1/19
  Verifying        : glib2-2.70.1-1.fc35.x86_64                            2/19
  Verifying        : glib2-2.70.0-5.fc35.x86_64                            3/19
  Verifying        : glibc-2.34-8.fc35.x86_64                              4/19
  Verifying        : glibc-2.34-7.fc35.x86_64                              5/19
  Verifying        : glibc-common-2.34-8.fc35.x86_64                       6/19
  Verifying        : glibc-common-2.34-7.fc35.x86_64                       7/19
  Verifying        : glibc-minimal-langpack-2.34-8.fc35.x86_64             8/19
  Verifying        : glibc-minimal-langpack-2.34-7.fc35.x86_64             9/19
  Verifying        : gpgme-1.15.1-6.fc35.x86_64                           10/19
  Verifying        : gpgme-1.15.1-4.fc35.x86_64                           11/19
  Verifying        : libgpg-error-1.43-1.fc35.x86_64                      12/19
  Verifying        : libgpg-error-1.42-3.fc35.x86_64                      13/19
  Verifying        : python3-gpg-1.15.1-6.fc35.x86_64                     14/19
  Verifying        : python3-gpg-1.15.1-4.fc35.x86_64                     15/19
  Verifying        : shadow-utils-2:4.9-5.fc35.x86_64                     16/19
  Verifying        : shadow-utils-2:4.9-3.fc35.x86_64                     17/19
  Verifying        : vim-minimal-2:8.2.3582-1.fc35.x86_64                 18/19
  Verifying        : vim-minimal-2:8.2.3568-1.fc35.x86_64                 19/19

Upgraded:
  glib2-2.70.1-1.fc35.x86_64
  glibc-2.34-8.fc35.x86_64
  glibc-common-2.34-8.fc35.x86_64
  glibc-minimal-langpack-2.34-8.fc35.x86_64
  gpgme-1.15.1-6.fc35.x86_64
  libgpg-error-1.43-1.fc35.x86_64
  python3-gpg-1.15.1-6.fc35.x86_64
  shadow-utils-2:4.9-5.fc35.x86_64
  vim-minimal-2:8.2.3582-1.fc35.x86_64
Installed:
  glibc-gconv-extra-2.34-8.fc35.x86_64

Complete!
STEP 5/5: RUN dnf install -y gcc
Last metadata expiration check: 0:00:10 ago on Fri Nov 12 21:22:21 2021.
Dependencies resolved.
================================================================================
 Package                       Arch      Version               Repository  Size
================================================================================
Installing:
 gcc                           x86_64    11.2.1-1.fc35         fedora      31 M
Installing dependencies:
 binutils                      x86_64    2.37-10.fc35          fedora     6.0 M
 binutils-gold                 x86_64    2.37-10.fc35          fedora     728 k
 cpp                           x86_64    11.2.1-1.fc35         fedora      10 M
 elfutils-debuginfod-client    x86_64    0.185-5.fc35          fedora      36 k
 gc                            x86_64    8.0.4-6.fc35          fedora     103 k
 glibc-devel                   x86_64    2.34-8.fc35           updates    146 k
 glibc-headers-x86             noarch    2.34-8.fc35           updates    544 k
 guile22                       x86_64    2.2.7-3.fc35          fedora     6.4 M
 kernel-headers                x86_64    5.14.9-300.fc35       fedora     1.3 M
 libmpc                        x86_64    1.2.1-3.fc35          fedora      62 k
 libpkgconf                    x86_64    1.8.0-1.fc35          fedora      36 k
 libtool-ltdl                  x86_64    2.4.6-42.fc35         fedora      36 k
 libxcrypt-devel               x86_64    4.4.26-4.fc35         fedora      29 k
 make                          x86_64    1:4.3-6.fc35          fedora     533 k
 pkgconf                       x86_64    1.8.0-1.fc35          fedora      41 k
 pkgconf-m4                    noarch    1.8.0-1.fc35          fedora      14 k
 pkgconf-pkg-config            x86_64    1.8.0-1.fc35          fedora      10 k

Transaction Summary
================================================================================
Install  18 Packages

Total download size: 57 M
Installed size: 196 M
Downloading Packages:
(1/18): binutils-gold-2.37-10.fc35.x86_64.rpm   1.4 MB/s | 728 kB     00:00
(2/18): elfutils-debuginfod-client-0.185-5.fc35 565 kB/s |  36 kB     00:00
(3/18): gc-8.0.4-6.fc35.x86_64.rpm              1.4 MB/s | 103 kB     00:00
(4/18): binutils-2.37-10.fc35.x86_64.rpm        6.1 MB/s | 6.0 MB     00:00
(5/18): cpp-11.2.1-1.fc35.x86_64.rpm            9.2 MB/s |  10 MB     00:01
(6/18): kernel-headers-5.14.9-300.fc35.x86_64.r  11 MB/s | 1.3 MB     00:00
(7/18): libmpc-1.2.1-3.fc35.x86_64.rpm          785 kB/s |  62 kB     00:00
(8/18): guile22-2.2.7-3.fc35.x86_64.rpm          16 MB/s | 6.4 MB     00:00
(9/18): libpkgconf-1.8.0-1.fc35.x86_64.rpm      376 kB/s |  36 kB     00:00
(10/18): libtool-ltdl-2.4.6-42.fc35.x86_64.rpm  520 kB/s |  36 kB     00:00
(11/18): libxcrypt-devel-4.4.26-4.fc35.x86_64.r 429 kB/s |  29 kB     00:00
(12/18): pkgconf-1.8.0-1.fc35.x86_64.rpm        471 kB/s |  41 kB     00:00
(13/18): pkgconf-m4-1.8.0-1.fc35.noarch.rpm     148 kB/s |  14 kB     00:00
(14/18): pkgconf-pkg-config-1.8.0-1.fc35.x86_64 143 kB/s |  10 kB     00:00
(15/18): glibc-devel-2.34-8.fc35.x86_64.rpm     518 kB/s | 146 kB     00:00
(16/18): gcc-11.2.1-1.fc35.x86_64.rpm            21 MB/s |  31 MB     00:01
(17/18): make-4.3-6.fc35.x86_64.rpm             702 kB/s | 533 kB     00:00
(18/18): glibc-headers-x86-2.34-8.fc35.noarch.r 2.0 MB/s | 544 kB     00:00
--------------------------------------------------------------------------------
Total                                            19 MB/s |  57 MB     00:02
Running transaction check
Transaction check succeeded.
Running transaction test
Transaction test succeeded.
Running transaction
  Preparing        :                                                        1/1
  Installing       : libmpc-1.2.1-3.fc35.x86_64                            1/18
  Installing       : cpp-11.2.1-1.fc35.x86_64                              2/18
  Installing       : glibc-headers-x86-2.34-8.fc35.noarch                  3/18
  Installing       : pkgconf-m4-1.8.0-1.fc35.noarch                        4/18
  Installing       : libtool-ltdl-2.4.6-42.fc35.x86_64                     5/18
  Installing       : libpkgconf-1.8.0-1.fc35.x86_64                        6/18
  Installing       : pkgconf-1.8.0-1.fc35.x86_64                           7/18
  Installing       : pkgconf-pkg-config-1.8.0-1.fc35.x86_64                8/18
  Installing       : kernel-headers-5.14.9-300.fc35.x86_64                 9/18
  Installing       : libxcrypt-devel-4.4.26-4.fc35.x86_64                 10/18
  Installing       : glibc-devel-2.34-8.fc35.x86_64                       11/18
  Installing       : gc-8.0.4-6.fc35.x86_64                               12/18
  Installing       : guile22-2.2.7-3.fc35.x86_64                          13/18
  Installing       : make-1:4.3-6.fc35.x86_64                             14/18
  Installing       : elfutils-debuginfod-client-0.185-5.fc35.x86_64       15/18
  Installing       : binutils-gold-2.37-10.fc35.x86_64                    16/18
  Installing       : binutils-2.37-10.fc35.x86_64                         17/18
  Running scriptlet: binutils-2.37-10.fc35.x86_64                         17/18
  Installing       : gcc-11.2.1-1.fc35.x86_64                             18/18
  Running scriptlet: gcc-11.2.1-1.fc35.x86_64                             18/18
  Verifying        : binutils-2.37-10.fc35.x86_64                          1/18
  Verifying        : binutils-gold-2.37-10.fc35.x86_64                     2/18
  Verifying        : cpp-11.2.1-1.fc35.x86_64                              3/18
  Verifying        : elfutils-debuginfod-client-0.185-5.fc35.x86_64        4/18
  Verifying        : gc-8.0.4-6.fc35.x86_64                                5/18
  Verifying        : gcc-11.2.1-1.fc35.x86_64                              6/18
  Verifying        : guile22-2.2.7-3.fc35.x86_64                           7/18
  Verifying        : kernel-headers-5.14.9-300.fc35.x86_64                 8/18
  Verifying        : libmpc-1.2.1-3.fc35.x86_64                            9/18
  Verifying        : libpkgconf-1.8.0-1.fc35.x86_64                       10/18
  Verifying        : libtool-ltdl-2.4.6-42.fc35.x86_64                    11/18
  Verifying        : libxcrypt-devel-4.4.26-4.fc35.x86_64                 12/18
  Verifying        : make-1:4.3-6.fc35.x86_64                             13/18
  Verifying        : pkgconf-1.8.0-1.fc35.x86_64                          14/18
  Verifying        : pkgconf-m4-1.8.0-1.fc35.noarch                       15/18
  Verifying        : pkgconf-pkg-config-1.8.0-1.fc35.x86_64               16/18
  Verifying        : glibc-devel-2.34-8.fc35.x86_64                       17/18
  Verifying        : glibc-headers-x86-2.34-8.fc35.noarch                 18/18

Installed:
  binutils-2.37-10.fc35.x86_64
  binutils-gold-2.37-10.fc35.x86_64
  cpp-11.2.1-1.fc35.x86_64
  elfutils-debuginfod-client-0.185-5.fc35.x86_64
  gc-8.0.4-6.fc35.x86_64
  gcc-11.2.1-1.fc35.x86_64
  glibc-devel-2.34-8.fc35.x86_64
  glibc-headers-x86-2.34-8.fc35.noarch
  guile22-2.2.7-3.fc35.x86_64
  kernel-headers-5.14.9-300.fc35.x86_64
  libmpc-1.2.1-3.fc35.x86_64
  libpkgconf-1.8.0-1.fc35.x86_64
  libtool-ltdl-2.4.6-42.fc35.x86_64
  libxcrypt-devel-4.4.26-4.fc35.x86_64
  make-1:4.3-6.fc35.x86_64
  pkgconf-1.8.0-1.fc35.x86_64
  pkgconf-m4-1.8.0-1.fc35.noarch
  pkgconf-pkg-config-1.8.0-1.fc35.x86_64

Complete!
COMMIT myimage
Getting image source signatures
Copying blob cd62a89550d0 skipped: already exists
Copying blob 0f38b540528b done
Copying config c0458c205e done
Writing manifest to image destination
Storing signatures
--> c0458c205e5
Successfully tagged localhost/myimage:latest
c0458c205e533af9be1e5e9e665afb0d491f622a243deac76b4cbd0824bf23f6

sh-5.0$ buildah images
REPOSITORY                          TAG      IMAGE ID       CREATED          SIZE
localhost/myimage                   latest   d3a341d4fd99   22 seconds ago   544 MB
registry.fedoraproject.org/fedora   35       1b52edb08181   23 hours ago     159 MB

sh-5.0$ ls -l output/
total 4
-rw-r--r--. 1 build build 7186 Nov 12 21:22 update-output.txt
````
