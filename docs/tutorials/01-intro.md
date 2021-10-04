![buildah logo](https://cdn.rawgit.com/containers/buildah/main/logos/buildah-logo_large.png)

# Buildah Tutorial 1
## Building OCI container images

The purpose of this tutorial is to demonstrate how Buildah can be used to build container images compliant with the [Open Container Initiative](https://www.opencontainers.org/) (OCI) [image specification](https://github.com/opencontainers/image-spec). Images can be built based on existing images, from scratch, and using Dockerfiles. OCI images built using the Buildah command line tool (CLI) and the underlying OCI based technologies (e.g. [containers/image](https://github.com/containers/image) and [containers/storage](https://github.com/containers/storage)) are portable and can therefore run in a Docker environment.

In brief the `containers/image` project provides mechanisms to copy (push, pull), inspect, and sign container images. The `containers/storage` project provides mechanisms for storing filesystem layers, container images, and containers. Buildah is a CLI that takes advantage of these underlying projects and therefore allows you to build, move, and manage container images and containers.

Buildah works on a number of Linux distributions, but is not supported on Windows or Mac platforms at this time.  Buildah specializes mainly in building OCI images while [Podman](https://podman.io) provides a broader set of commands and functions that help you to maintain, modify and run OCI images and containers.  For more information on the difference between the projects please refer to the [Buildah and Podman relationship](https://github.com/containers/buildah#buildah-and-podman-relationship) section on the main README.md.

## Configure and Install Buildah

Note that installation instructions below assume you are running a Linux distro that uses `dnf` as its package manager, and have all prerequisites fulfilled. See Buildah's [installation instructions][buildah-install] for a full list of prerequisites, and the `buildah` installation section in the [official Red Hat documentation][rh-repo-docs] for RHEL-specific instructions.

[buildah-install]:../../install.md
[rh-repo-docs]:https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/8/html/building_running_and_managing_containers/

First step is to install Buildah. Run as root because you will need to be root for installing the Buildah package:

    $ sudo -s

Then install buildah by running:

    # dnf -y install buildah

## Rootless User Configuration

If you plan to run Buildah as a user without root privileges, i.e. a "rootless user", the administrator of the system might have to do a bit of additional configuration beforehand.  The setup required for this is listed on the Podman GitHub site [here](https://github.com/containers/podman/blob/main/docs/tutorials/rootless_tutorial.md).  Buildah has the same setup and configuration requirements that Podman does for rootless users.

## Post Installation Verification

After installing Buildah we can see there are no images installed. The `buildah images` command will list all the images:

    # buildah images

We can also see that there are also no working containers by running:

    # buildah containers

When you build a working container from an existing image, Buildah defaults to appending '-working-container' to the image's name to construct a name for the container. The Buildah CLI conveniently returns the name of the new container. You can take advantage of this by assigning the returned value to a shell variable using standard shell assignment:

    # container=$(buildah from fedora)

It is not required to assign the container's name to a shell variable. Running `buildah from fedora` is sufficient. It just helps simplify commands later. To see the name of the container that we stored in the shell variable:

    # echo $container

What can we do with this new container? Let's try running bash:

    # buildah run $container bash

Notice we get a new shell prompt because we are running a bash shell inside of the container. It should be noted that `buildah run` is primarily intended for debugging and running commands as part of the build process. A more full-featured engine like Podman or a container runtime interface service like [CRI-O](https://github.com/kubernetes-sigs/cri-o) is more suited for starting containers in production.

Be sure to `exit` out of the container and let's try running something else:

    # buildah run $container java

Oops. Java is not installed. A message containing something like the following was returned.

    runc create failed: unable to start start container process: exec: "java": executable file not found in $PATH

Let's try installing it inside the container using:

    # buildah run $container -- dnf -y install java

The `--` syntax basically tells Buildah: there are no more `buildah run` command options after this point. The options after this point are for the command that's started inside the container. It is required if the command we specify includes command line options which are not meant for Buildah.

Now running `buildah run $container java` will show that Java has been installed. It will return the standard Java `Usage` output.

## Building a container from scratch

One of the advantages of using `buildah` to build OCI compliant container images is that you can easily build a container image from scratch and therefore exclude unnecessary packages from your image. Most final container images for production probably don't need a package manager like `dnf`.

Let's build a container and image from scratch. The special "image" name "scratch" tells Buildah to create an empty container.  The container has a small amount of metadata about the container but no real Linux content.

    # newcontainer=$(buildah from scratch)

You can see this new empty container by running:

    # buildah containers

You should see output similar to the following:

    CONTAINER ID  BUILDER  IMAGE ID     IMAGE NAME                       CONTAINER NAME
    82af3b9a9488     *     3d85fcda5754 docker.io/library/fedora:latest  fedora-working-container
    ac8fa6be0f0a     *                  scratch                          working-container

Its container name is working-container by default and it's stored in the `$newcontainer` variable. Notice the image name (IMAGE NAME) is "scratch". This is a special value that indicates that the working container wasn't based on an image. When we run:

    # buildah images

We don't see the "scratch" image listed. There is no corresponding scratch image. A container based on "scratch" starts from nothing.

So does this container actually do anything? Let's see.

    # buildah run $newcontainer bash

Nope. This really is empty. The package installer `dnf` is not even inside this container. It's essentially an empty layer on top of the kernel. So what can be done with that? Thankfully there is a `buildah mount` command.

    # scratchmnt=$(buildah mount $newcontainer)

Note: If attempting to mount in rootless mode, the command fails. Mounting a container can only be done in a mount namespace that you own. Create and enter a user namespace and mount namespace by executing the `buildah unshare` command. See buildah-mount(1) man page for more information.

    $ buildah unshare
    # scratchmnt=$(buildah mount $newcontainer)

By echoing `$scratchmnt` we can see the path for the [overlay mount point](https://wiki.archlinux.org/index.php/Overlay_filesystem), which is used as the root file system for the container.

    # echo $scratchmnt
    /var/lib/containers/storage/overlay/b78d0e11957d15b5d1fe776293bd40a36c28825fb6cf76f407b4d0a95b2a200d/merged

Notice that the overlay mount point is somewhere under `/var/lib/containers/storage` if you started out as root, and under your home directory's `.local/share/containers/storage` directory if you're in rootless mode. (See above on `containers/storage` or for more information see [containers/storage](https://github.com/containers/storage).)

Now that we have a new empty container we can install or remove software packages or simply copy content into that container. So let's install `bash` and `coreutils` so that we can run bash scripts. This could easily be `nginx` or other packages needed for your container.

**NOTE:** the version in the example below (35) relates to a Fedora version which is the Linux platform this example was run on.  If you are running dnf on the host to populate the container, the version you specify must be valid for the host or dnf will throw an error.  I.e. If you were to run this on a RHEL platform, you'd need to specify `--releasever 8.1` or similar instead of `--releasever 35`.  If you want the container to be a particular Linux platform, change `scratch` in the first line of the example to the platform you want, i.e. `# newcontainer=$(buildah from fedora)`, and then you can specify an appropriate version number for that Linux platform.

    # dnf install --installroot $scratchmnt --releasever 35 bash coreutils --setopt install_weak_deps=false -y

Let's try it out (showing the prompt in this example to demonstrate the difference):

    # buildah run $newcontainer sh
    sh-5.1# cd /usr/bin
    sh-5.1# ls
    sh-5.1# exit

Notice we now have a `/usr/bin` directory in the newcontainer's root file system. Let's first copy a simple file from our host into the container. Create a file called runecho.sh which contains the following:

    #!/usr/bin/env bash
    for i in `seq 0 9`;
    do
    	echo "This is a new container from ipbabble [" $i "]"
    done

Change the permissions on the file so that it can be run:

    # chmod +x runecho.sh

With `buildah` files can be copied into the new container.  We can then use `buildah run` to run that command within the container by specifying the command.  We can also configure the image we'll create from this container to run the command directly when we run it using [Podman](https://github.com/containers/podman) and its `podman run` command. In short the `buildah run` command is equivalent to the "RUN" command in a Dockerfile (it always needs to be told what to run), whereas `podman run` is equivalent to the `docker run` command (it can look at the image's configuration to see what to run).  Now let's copy this new command into the container's `/usr/bin` directory, configure the command to be run when the image is run by `podman`, and create an image from the container's root file system and configuration settings:

    # To test with Podman, first install via:
    # dnf -y install podman
    # buildah copy $newcontainer ./runecho.sh /usr/bin
    # buildah config --cmd /usr/bin/runecho.sh $newcontainer
    # buildah commit $newcontainer newimage

We've got a new image named "newimage". The container is still there because we didn't remove it.
Now run the command in the container with Buildah specifying the command to run in the container:

    # buildah run $newcontainer /usr/bin/runecho.sh
    This is a new container from ipbabble [ 0 ]
    This is a new container from ipbabble [ 1 ]
    This is a new container from ipbabble [ 2 ]
    This is a new container from ipbabble [ 3 ]
    This is a new container from ipbabble [ 4 ]
    This is a new container from ipbabble [ 5 ]
    This is a new container from ipbabble [ 6 ]
    This is a new container from ipbabble [ 7 ]
    This is a new container from ipbabble [ 8 ]
    This is a new container from ipbabble [ 9 ]

Now use Podman to run the command in a new container based on our new image (no command required):

    # podman run --rm newimage
    This is a new container from ipbabble [ 0 ]
    This is a new container from ipbabble [ 1 ]
    This is a new container from ipbabble [ 2 ]
    This is a new container from ipbabble [ 3 ]
    This is a new container from ipbabble [ 4 ]
    This is a new container from ipbabble [ 5 ]
    This is a new container from ipbabble [ 6 ]
    This is a new container from ipbabble [ 7 ]
    This is a new container from ipbabble [ 8 ]
    This is a new container from ipbabble [ 9 ]

It works! Congratulations, you have built a new OCI container image from scratch that uses bash scripting.

Back to Buildah, let's add some more configuration information.

    # buildah config --created-by "ipbabble"  $newcontainer
    # buildah config --author "wgh at redhat.com @ipbabble" --label name=fedora35-bashecho $newcontainer

We can inspect the working container's metadata using the `inspect` command:

    # buildah inspect $newcontainer

We should probably unmount the working container's rootfs.  We will need to commit the container again to create an image that includes the two configuration changes we just made:

     # buildah unmount $newcontainer
     # buildah commit $newcontainer fedora-bashecho
     # buildah images

And you can see there is a new image called `localhost/fedora-bashecho:latest`. You can inspect the new image using:

    # buildah inspect --type=image fedora-bashecho

Later when you want to create a new container or containers from this image, you simply need need to do `buildah from fedora-bashecho`. This will create a new container based on this image for you.

Now that you have the new image you can remove the scratch container called working-container:

    # buildah rm $newcontainer

or

    # buildah rm working-container

## OCI images built using Buildah are portable

Let's test if this new OCI image is really portable to another container engine like Docker. First you should install Docker and start it. Notice that Docker requires a running daemon process in order to run any client commands. Buildah and Podman have no daemon requirement.

    # dnf -y install docker
    # systemctl start docker

Let's copy that image from where containers/storage stores it to where the Docker daemon stores its images, so that we can run it using Docker. We can achieve this using `buildah push`. This copies the image to Docker's storage area which is located under `/var/lib/docker`. Docker's storage is managed by the Docker daemon. This needs to be explicitly stated by telling Buildah to push the image to the Docker daemon using `docker-daemon:`.

    # buildah push fedora-bashecho docker-daemon:fedora-bashecho:latest

Under the covers, the containers/image library calls into the containers/storage library to read the image's contents from where buildah keeps them, and sends them to the local Docker daemon, which writes them to where it keeps them. This can take a little while. And usually you won't need to do this. If you're using `buildah` you are probably not using Docker. This is just for demo purposes. Let's try it:

    # docker run --rm fedora-bashecho
    This is a new container from ipbabble [ 0 ]
    This is a new container from ipbabble [ 1 ]
    This is a new container from ipbabble [ 2 ]
    This is a new container from ipbabble [ 3 ]
    This is a new container from ipbabble [ 4 ]
    This is a new container from ipbabble [ 5 ]
    This is a new container from ipbabble [ 6 ]
    This is a new container from ipbabble [ 7 ]
    This is a new container from ipbabble [ 8 ]
    This is a new container from ipbabble [ 9 ]

OCI container images built with `buildah` are completely standard as expected. So now it might be time to run:

    # dnf -y remove docker

## Using Containerfiles/Dockerfiles with Buildah

What if you have been using Docker for a while and have some existing Dockerfiles? Not a problem. Buildah can build images using a Dockerfile. The `build` command takes a Dockerfile as input and produces an OCI image.

Find one of your Dockerfiles or create a file called Dockerfile. Use the following example or some variation if you'd like:

    # Base on the most recently released Fedora
    FROM fedora:latest
    MAINTAINER ipbabble email buildahboy@redhat.com # not a real email

    # Install updates and httpd
    RUN echo "Updating all fedora packages"; dnf -y update; dnf -y clean all
    RUN echo "Installing httpd"; dnf -y install httpd && dnf -y clean all

    # Expose the default httpd port 80
    EXPOSE 80

    # Run the httpd
    CMD ["/usr/sbin/httpd", "-DFOREGROUND"]

Now run `buildah build` with the name of the Dockerfile and the name to be given to the created image (e.g. fedora-httpd):

    # buildah build -f Dockerfile -t fedora-httpd .

or, because `buildah build` defaults to `Dockerfile` and using the current directory as the build context:

    # buildah build -t fedora-httpd

You will see all the steps of the Dockerfile executing. Afterwards `buildah images` will show you the new image. Now we can create a container from the image and test it with `podman run`:

    # podman run --rm -p 8123:80 fedora-httpd

While that container is running, in another shell run:

    # curl localhost:8123

You will see the standard Apache webpage.

Why not try and modify the Dockerfile. Do not install httpd, but instead ADD the runecho.sh file and have it run as the CMD.

## Congratulations

Well done. You have learned a lot about Buildah using this short tutorial. Hopefully you followed along with the examples and found them to be sufficient. Be sure to look at Buildah's man pages to see the other useful commands you can use. Have fun playing.

If you have any suggestions or issues please post them at the [Buildah Issues page](https://github.com/containers/buildah/issues).

For more information on Buildah and how you might contribute please visit the [Buildah home page on GitHub](https://github.com/containers/buildah).
