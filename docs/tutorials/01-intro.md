![buildah logo](https://cdn.rawgit.com/containers/buildah/master/logos/buildah-logo_large.png)

# Buildah Tutorial 1
## Building OCI container images

The purpose of this tutorial is to demonstrate how Buildah can be used to build container images compliant with the [Open Container Initiative](https://www.opencontainers.org/) (OCI) [image specification](https://github.com/opencontainers/image-spec). Images can be built from existing images, from scratch, and using Dockerfiles. OCI images built using the Buildah command line tool (CLI) and the underlying OCI based technologies (e.g. [containers/image](https://github.com/containers/image) and [containers/storage](https://github.com/containers/storage)) are portable and can therefore run in a Docker environment.

In brief the `containers/image` project provides mechanisms to copy, push, pull, inspect and sign container images. The `containers/storage` project provides mechanisms for storing filesystem layers, container images, and containers. Buildah is a CLI that takes advantage of these underlying projects and therefore allows you to build, move, and manage container images and containers.

Note that installation instructions below assume you are running a Linux distro that uses `dnf` as its package manager, and have all prerequisites fulfilled. See Buildah's [installation instructions][buildah-install] for a full list of prerequisites, and the `buildah` installation section in the [official Red Hat documentation][rh-repo-docs] for RHEL-specific instructions.

[buildah-install]:../../install.md
[rh-repo-docs]:https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux_atomic_host/7/html/managing_containers/finding_running_and_building_containers_without_docker

First step is to install Buildah. Run as root because you will need to be root for running Buildah commands:

    $ sudo -s

Then install buildah by running:

    # dnf -y install buildah

After installing Buildah we can see there are no images installed. The `buildah images` command will list all the images:

    # buildah images

We can also see that there are also no containers by running:

    # buildah containers

When you build a working container from an existing image, Buildah defaults to appending '-working-container' to the image's name to construct a name for the container. The Buildah CLI conveniently returns the name of the new container. You can take advantage of this by assigning the returned value to a shell varible using standard shell assignment :

    # container=$(buildah from fedora)

It is not required to assign a shell variable. Running `buildah from fedora` is sufficient. It just helps simplify commands later. To see the name of the container that we stored in the shell variable:

    # echo $container

What can we do with this new container? Let's try running bash:

    # buildah run $container bash

Notice we get a new shell prompt because we are running a bash shell inside of the container. It should be noted that `buildah run` is primarily intended for helping debug during the build process. A runtime like runc or a container interface like [CRI-O](https://github.com/kubernetes-sigs/cri-o) is more suited for starting containers in production.

Be sure to `exit` out of the container and let's try running something else:

    # buildah run $container java

Oops. Java is not installed. A message containing something like the following was returned.

    container_linux.go:274: starting container process caused "exec: \"java\": executable file not found in $PATH"

Lets try installing it using:

    # buildah run $container -- dnf -y install java

The `--` syntax basically tells Buildah: there are no more `buildah run` command options after this point. The options after this point are for inside the containers shell. It is required if the command we specify includes command line options which are not meant for Buildah.

Now running `buildah run $container java` will show that Java has been installed. It will return the standard Java `Usage` output.

## Building a container from scratch

One of the advantages of using `buildah` to build OCI compliant container images is that you can easily build a container image from scratch and therefore exclude unnecessary packages from your image. E.g. most final container images for production probably don't need a package manager like `dnf`.

Let's build a container from scratch. The special "image" name "scratch" tells Buildah to create an empty container.  The container has a small amount of metadata about the container but no real Linux content.

    # newcontainer=$(buildah from scratch)

You can see this new empty container by running:

    # buildah containers

You should see output similar to the following:

    CONTAINER ID  BUILDER  IMAGE ID     IMAGE NAME                       CONTAINER NAME
    82af3b9a9488     *     3d85fcda5754 docker.io/library/fedora:latest  fedora-working-container
    ac8fa6be0f0a     *                  scratch                          working-container

Its container name is working-container by default and it's stored in the `$newcontainer` variable. Notice the image name (IMAGE NAME) is "scratch". This just indicates that there is no real image yet. i.e. It is containers/storage but there is no representation in containers/image. So when we run:

    # buildah images

We don't see the image listed. There is no corresponding scratch image. It is an empty container.

So does this container actually do anything? Let's see.

    # buildah run $newcontainer bash

Nope. This really is empty. The package installer `dnf` is not even inside this container. It's essentially an empty layer on top of the kernel. So what can be done with that?  Thankfully there is a `buildah mount` command.

    # scratchmnt=$(buildah mount $newcontainer)

By echoing `$scratchmnt` we can see the path for the [overlay image](https://wiki.archlinux.org/index.php/Overlay_filesystem), which gives you a link directly to the root file system of the container.

    # echo $scratchmnt
    /var/lib/containers/storage/overlay/b78d0e11957d15b5d1fe776293bd40a36c28825fb6cf76f407b4d0a95b2a200d/merged

Notice that the overlay image is under `/var/lib/containers/storage` as one would expect. (See above on `containers/storage` or for more information see [containers/storage](https://github.com/containers/storage).)

Now that we have a new empty container we can install or remove software packages or simply copy content into that container. So let's install `bash` and `coreutils` so that we can run bash scripts. This could easily be `nginx` or other packages needed for your container.

    # dnf install --installroot $scratchmnt --releasever 29 bash coreutils --setopt install_weak_deps=false -y

Let's try it out (showing the prompt in this example to demonstrate the difference):

    # buildah run $newcontainer bash
    bash-4.4# cd /usr/bin
    bash-4.4# ls
    bash-4.4# exit

Notice we have a `/usr/bin` directory in the newcontainer's image layer. Let's first copy a simple file from our host into the container. Create a file called runecho.sh which contains the following:

    #!/bin/bash
    for i in `seq 0 9`;
    do
    	echo "This is a new container from ipbabble [" $i "]"
    done

Change the permissions on the file so that it can be run:

    # chmod +x runecho.sh


With `buildah` files can be copied into the new image.  We can then use `buildah run` to run that command within the container by specifying the command.  We can also configure the image to run the command directly using [Podman](https://github.com/containers/libpod) and its `podman run` command. In short the `buildah run` command is equivalent to the "RUN" command in a Dockerfile, whereas `podman run` is equivalent to the `docker run` command.  Now let's copy this new command into the container's `/usr/bin` directory and configure the container to run the command when the container is run via podman:

    # To test with Podman, first install via:
    # dnf -y install podman
    # buildah copy $newcontainer ./runecho.sh /usr/bin
    # buildah config --cmd /usr/bin/runecho.sh $newcontainer

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

Now run the command in the container with Podman (no command required):

    # podman run $newcontainer
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

It works! Congratulations, you have built a new OCI container from scratch that uses bash scripting.

Back to Buildah, let's add some more configuration information.

    # buildah config --created-by "ipbabble"  $newcontainer
    # buildah config --author "wgh at redhat.com @ipbabble" --label name=fedora29-bashecho $newcontainer

We can inspect the container's metadata using the `inspect` command:

    # buildah inspect $newcontainer

We should probably unmount and commit the image:

     # buildah unmount $newcontainer
     # buildah commit $newcontainer fedora-bashecho
     # buildah images

And you can see there is a new image called `fedora-bashecho:latest`. You can inspect the new image using:

    # buildah inspect --type=image fedora-bashecho

Later when you want to create a new container or containers from this image, you simply need need to do `buildah from fedora-bashecho`. This will create a new containers based on this image for you.

Now that you have the new image you can remove the scratch container called working-container:

    # buildah rm $newcontainer

or

    # buildah rm working-container

## OCI images built using Buildah are portable

Let's test if this new OCI image is really portable to another OCI technology like Docker. First you should install Docker and start it. Notice that Docker requires a daemon process (that's quite big) in order to run any client commands. Buildah has no daemon requirement.

    # dnf -y install docker
    # systemctl start docker

Let's copy that image from where containers/storage stores it to where the Docker daemon stores its images, so that we can run it using Docker. We can achieve this using `buildah push`. This copies the image to Docker's repository area which is located under `/var/lib/docker`. Docker's repository is managed by the Docker daemon. This needs to be explicitly stated by telling Buildah to push to the Docker repository protocol using `docker-daemon:`.

    # buildah push fedora-bashecho docker-daemon:fedora-bashecho:latest

Under the covers, the containers/image library calls into the containers/storage library to read the image's contents, and sends them to the local Docker daemon. This can take a little while. And usually you won't need to do this. If you're using `buildah` you are probably not using Docker. This is just for demo purposes. Let's try it:

    # docker run fedora-bashecho
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

## Using Dockerfiles with Buildah

What if you have been using Docker for a while and have some existing Dockerfiles. Not a problem. Buildah can build images using a Dockerfile. The `build-using-dockerfile`, or `bud` for short, takes a Dockerfile as input and produces an OCI image.

Find one of your Dockerfiles or create a file called Dockerfile. Use the following example or some variation if you'd like:

    # Base on the Fedora
    FROM fedora:latest
    MAINTAINER  ipbabble email buildahboy@redhat.com # not a real email

    # Update image and install httpd
    RUN echo "Updating all fedora packages"; dnf -y update; dnf -y clean all
    RUN echo "Installing httpd"; dnf -y install httpd

    # Expose the default httpd port 80
    EXPOSE 80

    # Run the httpd
    CMD ["/usr/sbin/httpd", "-DFOREGROUND"]

Now run `buildah bud` with the name of the Dockerfile and the name to be given to the created image (e.g. fedora-httpd):

    # buildah bud -f Dockerfile -t fedora-httpd .

or, because `buildah bud` defaults to Dockerfile (note the period at the end of the example):

    # buildah bud -t fedora-httpd .

You will see all the steps of the Dockerfile executing. Afterwards `buildah images` will show you the new image. Now we need to create the container using `buildah from` and test it with `buildah run`:

    # httpcontainer=$(buildah from fedora-httpd)
    # buildah run $httpcontainer

While that container is running, in another shell run:

    # curl localhost

You will see the standard Apache webpage.

Why not try and modify the Dockerfile. Do not install httpd, but instead ADD the runecho.sh file and have it run as the CMD.

## Congratulations

Well done. You have learned a lot about Buildah using this short tutorial. Hopefully you followed along with the examples and found them to be sufficient. Be sure to look at Buildah's man pages to see the other useful commands you can use. Have fun playing.

If you have any suggestions or issues please post them at the [Buildah Issues page](https://github.com/containers/buildah/issues).

For more information on Buildah and how you might contribute please visit the [Buildah home page on Github](https://github.com/containers/buildah).
