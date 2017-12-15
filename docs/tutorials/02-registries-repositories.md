![buildah logo](https://cdn.rawgit.com/projectatomic/buildah/master/logos/buildah-logo_large.png)

# Buildah Tutorial 2
## Using Buildah with container registries

The purpose of this tutorial is to demonstrate how Buildah can be used to move OCI compliant images in and out of private or public registries. 

In the [first tutorial](https://github.com/projectatomic/buildah/blob/master/docs/tutorials/01-intro.md) we built an image from scratch that we called `fedora-bashecho` and we pushed it to a local Docker repository using the `docker-daemon` protocol. We are going to use the same image to push to a private Docker registry.

First we must pull down a registry. As a shortcut we will save the container name that is returned from the `buildah from` command, into a bash variable called `registry`. This is just like we did in Tutorial 1:

    # registry=$(buildah from registry)

It is worth pointing out that the `from` command can also use other protocols beyond the default (and implicity assumed) order that first looks in local containers-storage (containers-storage:) and then looks in the Docker hub (docker:). For example, if you already had a registry container image in a local Docker registry then you could use the following:

    # registry=$(buildah from docker-daemon:registry:latest)

Then we need to start the registry. You should start the registry in a separate shell and leave it running there:

    # buildah run $registry
    
If you would like to see more details as to what is going on inside the registry, especially if you are having problems with the registry, you can run the registry container in debug mode as follows:

    # buildah --debug run $registry

You can use `--debug` on any Buildah command. 

The registry is running and is waiting for requests to process. Notice that this registry is a Docker registry that we pulled from Docker hub and we are running it for this example using `buildah run`. There is no Docker daemon running at this time.

Let's push our image to the private registry. By default, Buildah is set up to expect secure connections to a registry. Therefore we will need to turn the TLS verification off using the `--tls-verify` flag. We also need to tell Buildah that the registry is on this local host ( i.e. localhost) and listening on port 5000. Similar to  what you'd expect to do on multi-tenant Docker hub, we will explicitly specify that the registry is to store the image under the `ipbabble` repository - so as not to clash with other users' similarly named images.

    # buildah push --tls-verify=false fedora-bashecho docker://localhost:5000/ipbabble/fedora-bashecho:latest

[Skopeo](https://github.com/projectatomic/skopeo) is a ProjectAtomic tool that was created to inspect images in registries without having to pull the image from the registry. It has grown to have many other uses. We will verify that the image has been stored by using Skopeo to inspect the image in the registry:
 
    # skopeo inspect --tls-verify=false docker://localhost:5000/ipbabble/fedora-bashecho:latest
    {
        "Name": "localhost:5000/ipbabble/fedora-bashecho",
        "Digest": "sha256:6806f9385f97bc09f54b5c0ef583e58c3bc906c8c0b3e693d8782d0a0acf2137",
        "RepoTags": [
            "latest"
        ],
        "Created": "2017-12-05T21:38:12.311901938Z",
        "DockerVersion": "",
        "Labels": {
            "name": "fedora-bashecho"
        },
        "Architecture": "amd64",
        "Os": "linux",
        "Layers": [
            "sha256:0cb7556c714767b8da6e0299cbeab765abaddede84769475c023785ae66d10ca"
        ]
    }

We can verify that it is still portable with Docker by starting Docker again, as we did in the first tutorial. Then we can pull down the image and starting the container using Docker:

    # systemctl start docker
    # docker pull localhost:5000/ipbabble/fedora-bashecho
    Using default tag: latest
    Trying to pull repository localhost:5000/ipbabble/fedora-bashecho ... 
    sha256:6806f9385f97bc09f54b5c0ef583e58c3bc906c8c0b3e693d8782d0a0acf2137: Pulling from localhost:5000/ipbabble/fedora-bashecho
    0cb7556c7147: Pull complete 
    Digest: sha256:6806f9385f97bc09f54b5c0ef583e58c3bc906c8c0b3e693d8782d0a0acf2137
    Status: Downloaded newer image for localhost:5000/ipbabble/fedora-bashecho:latest

    # docker run localhost:5000/ipbabble/fedora-bashecho
    This is a new container named ipbabble [ 0 ]
    This is a new container named ipbabble [ 1 ]
    This is a new container named ipbabble [ 2 ]
    This is a new container named ipbabble [ 3 ]
    This is a new container named ipbabble [ 4 ]
    This is a new container named ipbabble [ 5 ]
    This is a new container named ipbabble [ 6 ]
    This is a new container named ipbabble [ 7 ]
    This is a new container named ipbabble [ 8 ]
    This is a new container named ipbabble [ 9 ]
    # systemctl stop docker

Pushing to Docker hub is just as easy. Of course you must have an account with credentials. In this example I'm using a Docker hub API key, which has the form "username:password" (example password has been edited for privacy), that I created with my Docker hub account. I use the `--creds` flag to use my API key. I also specify my local image name `fedora-bashecho` as my image source and I use the `docker` protocol with no host or port so that it will look at the default Docker hub registry:

    #  buildah push --creds ipbabble:5bbb9990-6eeb-1234-af1a-aaa80066887c fedora-bashecho docker://ipbabble/fedora-bashecho:latest

And let's inspect that with Skopeo:

    # skopeo inspect --creds ipbabble:5bbb9990-6eeb-1234-af1a-aaa80066887c docker://ipbabble/fedora-bashecho:latest
    {
        "Name": "docker.io/ipbabble/fedora-bashecho",
        "Digest": "sha256:6806f9385f97bc09f54b5c0ef583e58c3bc906c8c0b3e693d8782d0a0acf2137",
        "RepoTags": [
            "latest"
        ],
        "Created": "2017-12-05T21:38:12.311901938Z",
        "DockerVersion": "",
        "Labels": {
            "name": "fedora-bashecho"
        },
        "Architecture": "amd64",
        "Os": "linux",
        "Layers": [
            "sha256:0cb7556c714767b8da6e0299cbeab765abaddede84769475c023785ae66d10ca"
        ]
    }

We can use Buildah to pull down the image using the `buildah from` command. But before we do let's clean up our local containers-storage so that we don't have an existing fedora-bashecho - otherwise Buildah will know it already exists and not bother pulling it down.

    #  buildah images 
    IMAGE ID             IMAGE NAME                                               CREATED AT             SIZE
    d4cd7d73ee42         docker.io/library/registry:latest                        Dec 1, 2017 22:15      31.74 MB
    e31b0f0b0a63         docker.io/library/fedora-bashecho:latest                 Dec 5, 2017 21:38      772 B
    # buildah rmi fedora-bashecho
    untagged: docker.io/library/fedora-bashecho:latest
    e31b0f0b0a63e94c5a558d438d7490fab930a282a4736364360ab9b92cb25f3a
    #  buildah images 
    IMAGE ID             IMAGE NAME                                               CREATED AT             SIZE
    d4cd7d73ee42         docker.io/library/registry:latest                        Dec 1, 2017 22:15      31.74 MB

Okay, so we don't have a fedora-bashecho anymore. Let's pull the image from Docker hub:

    # buildah from ipbabble/fedora-bashecho 

If you don't want to bother doing the remove image step (`rmi`) you can use the flag `--pull-always` to force the image to be pulled again and overwrite any corresponding local image.

Now check that image is in the local containers-storage:

    # buildah images
    IMAGE ID             IMAGE NAME                                               CREATED AT             SIZE
    d4cd7d73ee42         docker.io/library/registry:latest                        Dec 1, 2017 22:15      31.74 MB
    864871ac1c45         docker.io/ipbabble/fedora-bashecho:latest                Dec 5, 2017 21:38      315.4 MB
    
Success!

If you have any suggestions or issues please post them at the [ProjectAtomic Buildah Issues page](https://github.com/projectatomic/buildah/issues).

For more information on Buildah and how you might contribute please visit the [Buildah home page on Github](https://github.com/projectatomic/buildah).
