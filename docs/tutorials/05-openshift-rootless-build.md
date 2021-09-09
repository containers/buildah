![buildah logo](https://cdn.rawgit.com/containers/buildah/main/logos/buildah-logo_large.png)

# Buildah Tutorial 5
## Using Buildah to build images in a rootless OpenShift container

This tutorial will walk you through setting up a container in OpenShift for building images.

The instructions have been tested on OpenShift 4.3.28 with Buildah 1.14.8.

Note that the VFS volume mounting is used instead of the more performant fuse. But the the latter does not work at the moment.

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

$ oc whoami -t | podman login -u $(id -u -n) --password-stdin $REGISTRY_URL
Login Succeeded!
````


### Make builder image

This is the image that will host the building. It uses the Buildah stable official image, which is based on Fedora 32.

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
FROM quay.io/buildah/stable:v1.14.8

RUN touch /etc/subgid /etc/subuid \
 && chmod g=u /etc/subgid /etc/subuid /etc/passwd \
 && echo build:10000:65536 > /etc/subuid \
 && echo build:10000:65536 > /etc/subgid

# Use chroot since the default runc does not work when running rootless
RUN echo "export BUILDAH_ISOLATION=chroot" >> /home/build/.bashrc

# Use VFS since fuse does not work
RUN mkdir -p /home/build/.config/containers \
 && echo "driver=\"vfs\"" > /home/build/.config/containers/storage.conf

USER build
WORKDIR /home/build

# Just keep the container running, allowing "oc rsh" access
CMD ["python3", "-m", "http.server"]
EOF

$ podman build -t $REGISTRY_URL/image-build/buildah -f Containerfile-buildah
STEP 1: FROM quay.io/buildah/stable:v1.14.8
STEP 2: RUN touch /etc/subgid /etc/subuid  && chmod g=u /etc/subgid /etc/subuid /etc/passwd  && echo build:10000:65536 > /etc/subuid  && echo build:10000:65536 > /etc/subgid
--> a25dbbd3824
STEP 3: CMD ["python3", "-m", "http.server"]
STEP 4: COMMIT default-route-openshift-image-registry.../image-build/buildah
--> 9656f2677e3
9656f2677e3e760e071c93ca7cba116871f5549b28ad8595e9134679db2345fc

$ podman push $REGISTRY_URL/image-build/buildah
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
Current: = cap_chown,cap_dac_override,cap_fowner,cap_fsetid,cap_setgid,cap_setuid,cap_setpcap,cap_net_bind_service,cap_net_raw,cap_sys_chroot+i
Bounding set =cap_chown,cap_dac_override,cap_fowner,cap_fsetid,cap_setgid,cap_setuid,cap_setpcap,cap_net_bind_service,cap_net_raw,cap_sys_chroot
Ambient set =
Securebits: 00/0x0/1'b0
 secure-noroot: no (unlocked)
 secure-no-suid-fixup: no (unlocked)
 secure-keep-caps: no (unlocked)
 secure-no-ambient-raise: no (unlocked)
uid=1000(build)
gid=1000(build)
groups=
````

This is what the Buildah data should look like:

````console
sh-5.0$ buildah version
Version:         1.14.8
Go Version:      go1.14
Image Spec:      1.0.1-dev
Runtime Spec:    1.0.1-dev
CNI Spec:        0.4.0
libcni Version:
image Version:   5.4.3
Git Commit:
Built:           Thu Jan  1 00:00:00 1970
OS/Arch:         linux/amd64

sh-5.0$ buildah info
{
    "host": {
        "CgroupVersion": "v1",
        "Distribution": {
            "distribution": "fedora",
            "version": "32"
        },
        "MemTotal": 33726861312,
        "MenFree": 20319305728,
        "OCIRuntime": "runc",
        "SwapFree": 0,
        "SwapTotal": 0,
        "arch": "amd64",
        "cpus": 4,
        "hostname": "buildah-1-6hvsw",
        "kernel": "4.18.0-147.13.2.el8_1.x86_64",
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
        "RunRoot": "/var/tmp/1000/containers"
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
FROM fedora:33
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
STEP 1: FROM fedora:33
Getting image source signatures
Copying blob 453ed60def9c done
Copying config 71d10e102a done
Writing manifest to image destination
Storing signatures
STEP 2: RUN ls -l /test-script.sh
-rwxr-xr-x. 1 root root 34 Jul  8 07:47 /test-script.sh
STEP 3: RUN /test-script.sh "Hello world"
Args Hello world
total 8
lrwxrwxrwx.   1 root   root      7 Jan 28 18:30 bin -> usr/bin
dr-xr-xr-x.   2 root   root      6 Jan 28 18:30 boot
drwxr-xr-x.   5 nobody nobody  360 Jul  8 07:39 dev
drwxr-xr-x.  42 root   root   4096 Jul  7 09:07 etc
drwxr-xr-x.   2 root   root      6 Jan 28 18:30 home
lrwxrwxrwx.   1 root   root      7 Jan 28 18:30 lib -> usr/lib
lrwxrwxrwx.   1 root   root      9 Jan 28 18:30 lib64 -> usr/lib64
drwx------.   2 root   root      6 Jul  7 09:06 lost+found
drwxr-xr-x.   2 root   root      6 Jan 28 18:30 media
drwxr-xr-x.   2 root   root      6 Jan 28 18:30 mnt
drwxr-xr-x.   2 root   root      6 Jan 28 18:30 opt
drwxr-xr-x.   2 root   root      6 Jul  8 07:46 output
dr-xr-xr-x. 311 nobody nobody    0 Jul  8 07:39 proc
dr-xr-x---.   2 root   root    196 Jul  7 09:07 root
drwxr-xr-x.   3 root   root     42 Jul  8 07:47 run
lrwxrwxrwx.   1 root   root      8 Jan 28 18:30 sbin -> usr/sbin
drwxr-xr-x.   2 root   root      6 Jan 28 18:30 srv
dr-xr-xr-x.  13 nobody nobody    0 Jul  5 17:57 sys
-rwxr-xr-x.   1 root   root     34 Jul  8 07:47 test-script.sh
drwxrwxrwt.   2 root   root     32 Jul  7 09:07 tmp
drwxr-xr-x.  12 root   root    144 Jul  7 09:07 usr
drwxr-xr-x.  18 root   root    235 Jul  7 09:07 var
STEP 4: RUN dnf update -y | tee /output/update-output.txt
Fedora 33 openh264 (From Cisco) - x86_64        817  B/s | 5.1 kB     00:06
Fedora - Modular Rawhide - Developmental packag 3.0 MB/s | 3.1 MB     00:01
Fedora - Rawhide - Developmental packages for t  19 MB/s |  72 MB     00:03
Dependencies resolved.
Nothing to do.
Complete!
STEP 5: RUN dnf install -y gcc
Last metadata expiration check: 0:00:30 ago on Wed Jul  8 07:48:12 2020.
Dependencies resolved.
==================================================================================================================================================================================================================================================
 Package                                                       Architecture                                       Version                                                               Repository                                           Size
==================================================================================================================================================================================================================================================
Installing:
 gcc                                                           x86_64                                             10.1.1-2.fc33                                                         rawhide                                              30 M
Installing dependencies:
 binutils                                                      x86_64                                             2.34.0-7.fc33                                                         rawhide                                             5.4 M
 binutils-gold                                                 x86_64                                             2.34.0-7.fc33                                                         rawhide                                             857 k
 cpp                                                           x86_64                                             10.1.1-2.fc33                                                         rawhide                                             9.3 M
 glibc-devel                                                   x86_64                                             2.31.9000-17.fc33                                                     rawhide                                             1.0 M
 glibc-headers-x86                                             noarch                                             2.31.9000-17.fc33                                                     rawhide                                             472 k
 isl                                                           x86_64                                             0.16.1-10.fc32                                                        rawhide                                             872 k
 kernel-headers                                                x86_64                                             5.8.0-0.rc4.git0.1.fc33                                               rawhide                                             1.2 M
 libmpc                                                        x86_64                                             1.1.0-8.fc32                                                          rawhide                                              59 k
 libxcrypt-devel                                               x86_64                                             4.4.16-5.fc33                                                         rawhide                                              31 k

Transaction Summary
==================================================================================================================================================================================================================================================
Install  10 Packages

Total download size: 49 M
Installed size: 147 M
Downloading Packages:
(1/10): binutils-gold-2.34.0-7.fc33.x86_64.rpm                                                                                                                                                                    3.3 MB/s | 857 kB     00:00
(2/10): binutils-2.34.0-7.fc33.x86_64.rpm                                                                                                                                                                          16 MB/s | 5.4 MB     00:00
(3/10): cpp-10.1.1-2.fc33.x86_64.rpm                                                                                                                                                                              9.3 MB/s | 9.3 MB     00:01
(4/10): gcc-10.1.1-2.fc33.x86_64.rpm                                                                                                                                                                               33 MB/s |  30 MB     00:00
(5/10): glibc-devel-2.31.9000-17.fc33.x86_64.rpm                                                                                                                                                                  1.2 MB/s | 1.0 MB     00:00
(6/10): glibc-headers-x86-2.31.9000-17.fc33.noarch.rpm                                                                                                                                                            2.6 MB/s | 472 kB     00:00
(7/10): isl-0.16.1-10.fc32.x86_64.rpm                                                                                                                                                                              12 MB/s | 872 kB     00:00
(8/10): kernel-headers-5.8.0-0.rc4.git0.1.fc33.x86_64.rpm                                                                                                                                                          11 MB/s | 1.2 MB     00:00
(9/10): libmpc-1.1.0-8.fc32.x86_64.rpm                                                                                                                                                                            534 kB/s |  59 kB     00:00
(10/10): libxcrypt-devel-4.4.16-5.fc33.x86_64.rpm                                                                                                                                                                 589 kB/s |  31 kB     00:00
--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
Total                                                                                                                                                                                                              35 MB/s |  49 MB     00:01
Running transaction check
Transaction check succeeded.
Running transaction test
Transaction test succeeded.
Running transaction
  Preparing        :                                                                                                                                                                                                                          1/1
  Installing       : binutils-gold-2.34.0-7.fc33.x86_64                                                                                                                                                                                      1/10
  Installing       : binutils-2.34.0-7.fc33.x86_64                                                                                                                                                                                           2/10
  Running scriptlet: binutils-2.34.0-7.fc33.x86_64                                                                                                                                                                                           2/10
  Installing       : libmpc-1.1.0-8.fc32.x86_64                                                                                                                                                                                              3/10
  Installing       : cpp-10.1.1-2.fc33.x86_64                                                                                                                                                                                                4/10
  Installing       : kernel-headers-5.8.0-0.rc4.git0.1.fc33.x86_64                                                                                                                                                                           5/10
  Installing       : isl-0.16.1-10.fc32.x86_64                                                                                                                                                                                               6/10
  Installing       : glibc-headers-x86-2.31.9000-17.fc33.noarch                                                                                                                                                                              7/10
  Installing       : libxcrypt-devel-4.4.16-5.fc33.x86_64                                                                                                                                                                                    8/10
  Installing       : glibc-devel-2.31.9000-17.fc33.x86_64                                                                                                                                                                                    9/10
  Installing       : gcc-10.1.1-2.fc33.x86_64                                                                                                                                                                                               10/10
  Running scriptlet: gcc-10.1.1-2.fc33.x86_64                                                                                                                                                                                               10/10
  Verifying        : binutils-2.34.0-7.fc33.x86_64                                                                                                                                                                                           1/10
  Verifying        : binutils-gold-2.34.0-7.fc33.x86_64                                                                                                                                                                                      2/10
  Verifying        : cpp-10.1.1-2.fc33.x86_64                                                                                                                                                                                                3/10
  Verifying        : gcc-10.1.1-2.fc33.x86_64                                                                                                                                                                                                4/10
  Verifying        : glibc-devel-2.31.9000-17.fc33.x86_64                                                                                                                                                                                    5/10
  Verifying        : glibc-headers-x86-2.31.9000-17.fc33.noarch                                                                                                                                                                              6/10
  Verifying        : isl-0.16.1-10.fc32.x86_64                                                                                                                                                                                               7/10
  Verifying        : kernel-headers-5.8.0-0.rc4.git0.1.fc33.x86_64                                                                                                                                                                           8/10
  Verifying        : libmpc-1.1.0-8.fc32.x86_64                                                                                                                                                                                              9/10
  Verifying        : libxcrypt-devel-4.4.16-5.fc33.x86_64                                                                                                                                                                                   10/10

Installed:
  binutils-2.34.0-7.fc33.x86_64    binutils-gold-2.34.0-7.fc33.x86_64               cpp-10.1.1-2.fc33.x86_64      gcc-10.1.1-2.fc33.x86_64                glibc-devel-2.31.9000-17.fc33.x86_64    glibc-headers-x86-2.31.9000-17.fc33.noarch
  isl-0.16.1-10.fc32.x86_64        kernel-headers-5.8.0-0.rc4.git0.1.fc33.x86_64    libmpc-1.1.0-8.fc32.x86_64    libxcrypt-devel-4.4.16-5.fc33.x86_64

Complete!
STEP 6: COMMIT myimage
Getting image source signatures
Copying blob fd46c60e883a skipped: already exists
Copying blob f3157b126b5d done
Copying config d3a341d4fd done
Writing manifest to image destination
Storing signatures
--> d3a341d4fd9
d3a341d4fd993fb4ee84f102e5915fe9ab544f4cd72fd9947beec9e745f12302

sh-5.0$ buildah images
REPOSITORY                          TAG      IMAGE ID       CREATED          SIZE
localhost/myimage                   latest   d3a341d4fd99   22 seconds ago   475 MB
registry.fedoraproject.org/fedora   33       71d10e102a30   23 hours ago     191 MB

sh-5.0$ ls -l output/
total 4
-rw-r--r--. 1 build build 288 Jul  8 07:48 update-output.txt
````
