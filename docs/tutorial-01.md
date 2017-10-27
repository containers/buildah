# buildah Tutorial 1
## Building OCI container images

The purpose of this tutorial is to deomonstrate how buildah can be used to build container images compliant with the [Open Container Initiative](https://www.opencontainers.org/) (OCI) [image specification](https://github.com/opencontainers/image-spec). Images can be built from existing images, from scratch, and using Dockerfiles. OCI images built using the buildah command line tool (CLI) and the underlying OCI based technologies (e.g. [containers/image](https://github.com/containers/image) and [containers/storage](https://github.com/containers/storage)) are portable and can therefore run in a docker environment.

In brief the `containers/image` project provides mecahnisms to copy, push, pull, inspect and sign container images. The `containers/storage` project provides mecahnisms for storing filesystem layers, container images, and containers. `buildah` is a CLI that takes advantage of these underlying projects and therefore allows you to build, move, and manage container images and containers.  

First step is to install buildah:

    dnf -y install buildah

After installing buildah we can see there are no images installed. The `buildah images` command will list all the images:

    buildah images

We can also see that there are also no containers by running:

    buildah containers
  
When you build a working container from an existing image, buildah defaults to appending '-working-container' to the image's name to construct a name for the container. The buildah CLI conveniently returns the name of the new container. You can take advantage of this by assigning the returned value to a shell varible using standard shell assignment :

    container=$(buildah from fedora)

Of course you can always just run `build from fedora` and then list the containers to see the contianer name. As mentioned previously it will be fedora-working-container in this case.

See the name:  

    echo $container

What can we do with this new container? Let's try running bash:

    buildah run $container bash
    
Notice we get a new shell prompt because we running a bash shell inside of the container. It should be noted that `buildah run` is not intended for running production containers. It is for helping debug during the build process. 

Let's try running something else:

    buildah run $container java

Oops. Java is not installed. A message containing something like the following was returned.

    container_linux.go:274: starting container process caused "exec: \"java\": executable file not found in $PATH"
    
Lets try installing it using:
    
    buildah run $container -- dnf install java

The `--` syntax basically says: no more `buildah run` command options after this point. It is required if the command we specify includes command line options which are not meant for buildah. 

Now running `java` will show that Java has been installed. It will return the `Usage`:

## Building a container from scratch

One of the advantages of using `buildah` to build OCI compliant container images is that you can easily build a container image from scratch and therefore exclude unnecessary packages from your image. E.g. most final container images for production probably don't need a package manager like `dnf`. 

Let's build a container from scratch. The special "image" name scratch tells buildah to create an empty container:

    newcontainer=$(buildah from scratch)
  
You can see this new empty container by running:

    buildah containers
  
Its container name is working-container by default. And it's stored in the `$newcontainer` variable. Notice the image name is scratch. And when we run:

    buildah images
  
We don't see it listed. That is because "scratch" is not an image. It is an empty container.

So does this container actually do anything? Let's see.

    buildah run $newcontainer bash
    
Nope. This really is empty. The package installer `dnf` is not even inside this container. It's essentially an empty layer on top of the kernel. So what can be done with that?  Thankfully there is a `buildah mount` command.

    scratchmnt=$(buildah mount $newcontainer)
    
By echoing `$scratchmnt` we can see the path for the overlay image. 

    # echo $scratchmnt
    /var/lib/containers/storage/overlay/b78d0e11957d15b5d1fe776293bd40a36c28825fb6cf76f407b4d0a95b2a200d/diff  
    
Notice that the overlay image is under `/var/lib/containers/storage` as one would expect. (See above on `containers/storage` or for more information see [containers/storage](https://github.com/containers/storage).) 

Now that we have a new empty container we can install or remove software packages or simply copy content into that container. So let's install `bash` and `coreutils` so that we can run bash scripts. This could easily be `nginx` or other packages needed for your container.

    dnf install --installroot $scratchmnt --release 26 bash coreutils --setopt install_weak_deps=false -y

Let's try it out:

    # buildah run $newcontainer bash
    bash-4.4# cd /usr/bin
    bash-4.4# exit

Notice we have a `/usr/bin` directoy in the newcontianers image layer. Let's first copy a simple file. Create a file called runecho.sh contains the following:

    #!/bin/bash
    for i in `seq 0 9`;
    do
    	echo "This is a new container from ipbabble [" $i "]"
    done

Change the permissions on the file so that it can be run:

    chmod +x runecho.sh
    
With `buildah` files can be copied into the new image and we can also configure the image to run commands. Let's copy this new command into the container's `/usr/bin` directory and configure the container to run the command when the container is run: 

    buildah copy $newcontainer ./runecho.sh /usr/bin
    buildah config --cmd /usr/bin/runecho.sh $newcontainer
    
Now run the container:

    # buildah run $newcontainer
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

It works! Congratulations, you have built a new OCI container from scratch that uses bash scripting. Let's add some more configuration information.

    buildah config --created-by "ipbabble"  $newcontainer
    buildah config --author "wgh at redhat.com @ipbabble" --label fedora26-bashecho:latest $newcontainer
 
We can inspect the container's metadata using the `inspect` command:
 
    buildah inspect $newcontainer

We should probably unmount and commit the image:

     buildah unmount $newcontainer
     buildah commit $newcontainer fedora-bashecho
     buildah images
     
And you can see there is a new image called `fedora-bashecho:latest`. You can inspect the new image using:

    buildah inspect --type=image fedora-bashecho

## OCI images built using buildah are portable

Let's test if this new OCI image is really portable to another OCI technology like docker. First you shouuld install docker and start it.

    dnf -y install docker
    systemctl start docker
    
The we need to push the image from the `containers/image` across to docker's storage area which is under `/var/lib/docker`. This is managed by the docker dameon. We can do this telling buildah to push to the docker protocol called docker-daemon.

    buildah push fedora-bashecho docker-daemon:fedora-bashecho:latest

Underneath `buildah` the `containers/storage` copies the image blob over to the docker storage. This can take a little while. And usually you won't need to do this. If you're using `buildah` you are probably not using docker. This is just for demo purposes. Letr's try it:

    docker run fedora-bashecho 
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

    dnf -y remove docker

## Using Dockerfiles with buildah

What if you have been using docker for while and have some exisiting Dockerfiles. Not a problem. `buildah` can build images using a Dockerfile. The `build-using-dockerfile`, or `bud` for short, takes a Dockefile as input and produces an OCI image.

Find one of your Dockerfiles or create a file called Dockerfile. Use the following example or some variation if you'd like:

    # Base on the Fedora
    FROM fedora:latest
    MAINTAINER W Henry email ipbabble@gmail.com

    # Update image and install hhtpd
    RUN echo "Updating all fedora packages"; dnf -y update; dnf -y clean all
    RUN echo "Installing httpd"; dnf -y install httpd
 
    # Expose the default httpd port 80
    EXPOSE 80

    # Run the httpd
    CMD ["/usr/sbin/httpd", "-DFOREGROUND"]

Now run `buildah bud` with the name of the Dockerfile and the image name (fedora-hhtpd):

    buildah bud -f Dockerfile -t fedora-httpd

You will see all the steps of the Dockerfile executing. Afterware `buildah images` will show you the new image. Now we need to create the container and test it with `buidlah run` it:

    httpcontainer=$(buildah from fedora-httpd)
    buildah run $httpcontainer
    
While that container is running, in another shell run:

    curl localhost
    
You will see the standard apache webpage.

Why not try and modify the Dockerfile. Do not install httpd, but instead ADD the runecho.sh file and have it run as the CMD. 
    
