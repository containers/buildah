![buildah logo](../../logos/buildah-logo_large.png)

# Buildah Tutorial 3
## Using ONBUILD in Buildah

The purpose of this tutorial is to demonstrate how Buildah can use a Dockerfile with the ONBUILD instruction within it or how the ONBUILD instruction can be used with the `buildah config` command.  The ONBUILD instruction stores a command in the meta data of a container image which is then invoked when a secondary container image is created.  The image can have multiple ONBUILD instructions.  Note: The ONBUILD instructions do not change the content of the image that contain the instructions, only the container images that are created from this image are changed based on the FROM command.

Container images that are compliant with the [Open Container Initiative][] (OCI) [image specification][] do not support the ONBUILD instruction.  Images that are created by Buildah are OCI compliant by default.  Therefore only containers that are created by Buildah that use the Docker format can use the ONBUILD instruction.  The OCI format can be overridden in Buildah by specifying the Docker format with the `--format=docker` option or by setting the BUILDAH_FORMAT environment variable to 'docker'.  Regardless of the format selected, Buildah is capable of working seamlessly with either OCI or Docker compliant images and containers.

On to the tutorial.  The first step is to install Buildah.  In short, the `buildah run` command emulates the RUN command that is found in a Dockerfile while the `podman run` command emulates the `docker run` command.  For the purpose of this tutorial Buildah's run command will be used.  As an aside, Podman is aimed at managing containers, images, and pods while Buildah focuses on the building of containers.  For more info on Podman, please go to [GitHub][].

## Setup

The following assumes installation on Fedora.

Run as root because you will need to be root for running Buildah commands (at the time of this writing work is underway to allow non-root access):

    $ sudo -s

Then install Buildah by running:

    # dnf -y install buildah

After installing Buildah check to see that there are no images installed. The `buildah images` command will list all the images:

    # buildah images

We can also see that there are also no containers by running:

    # buildah containers

## Examples

The two examples that will be shown are relatively simple, but they illustrate how a command or a number of commands can be setup in a master image such that they will be added to a secondary container image that is created from it.  This is extremely useful if you need to setup an environment where your containers have 75% of the same content, but need a few individual tweaks.  This can be helpful in setting up a environment for maven or java development containers for instance.  In this way you can create a single Dockerfile with all the common setup steps as ONBUILD commands and then really minimize the buildah commands or instructions in a second Dockerfile that would be necessary to complete the creation of the container image.

NOTE: In the examples below the option `--format=docker` is used in several places.  If you wanted to omit that, you could define the `BUILDAH_FORMAT` and set it to 'docker'.  On Fedora that command would be `export BUILDAH_FORMAT docker`.
## ONBUILD in a Dockerfile - Example 1


The first example was provided by Chris Collins (GitHub @clcollins), the idea is a file `/bar` will be created in the secondary container image only, and not in the primary image.

First create two Dockerfiles:

```
$ cat << EOF > Dockerfile
FROM fedora:latest
RUN touch /foo
ONBUILD RUN touch /bar
EOF

$ cat << EOF > Dockerfile-2
FROM onbuild-image
RUN touch /baz
EOF
```

Now to create the first container and verify that ONBUILD has been set:

```
# buildah build --format=docker -f Dockerfile -t onbuild-image .
# buildah inspect --format '{{.Docker.Config.OnBuild}}' onbuild-image
[RUN touch /bar]
```

The second container is now created and the `/bar` file will be created within it:

```
# buildah build --format=docker -f Dockerfile-2 -t result-image .
STEP 1: FROM onbuild-image
STEP 2: RUN touch /bar    # Note /bar created here based on the ONBUILD in Dockerfile
STEP 3: RUN touch /baz
STEP 4: COMMIT containers-storage:[overlay@/var/lib/containers/storage+/run/containers/storage:overlay.override_kernel_check=true]localhost/result-image:latest
{output edited for brevity}
$ container=$(sudo buildah from result-image:latest)
# buildah run $container ls /bar /foo /baz
/bar  /baz  /foo
```

## ONBUILD via `buildah config` - Example 1

Instead of using a Dockerfile to create the onbuild-image, Buildah allows you to build an image and configure it directly with the same commands that can be found in a Dockerfile.  This allows for easy on the fly manipulation of your image.  Let's look at the previous example without the use of a Dockerfile when building the primary container image.

First a Fedora container will be created with `buildah from`, then the `/foo` file will be added with `buildah run`.  The `buildah config` command will configure  ONBUILD to add `/bar` when a container image is created from the primary image, and finally the image will be saved with `buildah commit`.

```
# buildah from --format=docker --name onbuild-container fedora:latest
# buildah run onbuild-container touch /foo
# buildah config --onbuild="RUN touch /bar" onbuild-container
# buildah commit --format=docker onbuild-container onbuild-image
{output edited for brevity}
# buildah inspect --format '{{.Docker.Config.OnBuild}}' onbuild-image
[RUN touch /bar]
```
The onbuild-image has been created, so now create a container from it using the same commands as the first example using the second Dockerfile:

```
# buildah build --format=docker -f Dockerfile-2 -t result-image .
STEP 1: FROM onbuild-image
STEP 2: RUN touch /bar    # Note /bar created here based on the ONBUILD in Dockerfile
STEP 3: RUN touch /baz
STEP 4: COMMIT containers-storage:[overlay@/var/lib/containers/storage+/run/containers/storage:overlay.override_kernel_check=true]localhost/result-image:latest
{output edited for brevity}
$ container=$(sudo buildah from result-image:latest)
# buildah run $container ls /bar /foo /baz
/bar  /baz  /foo
```
Or for bonus points, piece the secondary container image together with Buildah commands directly:

```
# buildah from --format=docker --name result-container onbuild-image
result-container
# buildah run result-container touch /baz
# buildah run result-container ls /bar /foo /baz
/bar  /baz  /foo
```

## ONBUILD via `buildah config` - Example 2

For this example the ONBUILD instructions in the primary container image will be used to copy a shell script and then run it in the secondary container image.  For the script, we'll make use of the shell script from the [Introduction Tutorial][].  First create a file in the local directory called `runecho.sh` containing the following:

```
#!/usr/bin/env bash

for i in `seq 0 9`;
do
	echo "This is a new container from ipbabble [" $i "]"
done
```
Change the permissions on the file so that it can be run:

```
$ chmod +x runecho.sh
```

Now create a second primary container image.  This image has multiple ONBUILD instructions, the first ONBUILD instruction copies the file into the image and a second ONBUILD instruction to then run it.  We're going to do this example using only Buildah commands.  A Dockerfile could be translated easily and used from these commands, or these commands could be saved to a script directly.

```
# buildah from --format=docker --name onbuild-container-2 fedora:latest
onbuild-container-2
# buildah config --onbuild="COPY ./runecho.sh /usr/bin/runecho.sh" onbuild-container-2
# buildah config --onbuild="RUN /usr/bin/runecho.sh" onbuild-container-2
# buildah commit --format=docker onbuild-container-2 onbuild-image-2
{output edited for brevity}
# buildah inspect --format '{{.Docker.Config.OnBuild}}' onbuild-image-2
[COPY ./runecho.sh /usr/bin/runecho.sh RUN /usr/bin/runecho.sh &> /tmp/runecho.txt]
```

Now the result container can be created from the primary container image onbuild-image-2.  The runecho.sh script will be copied to the containers /usr/bin directory and then run from there when the secondary container image is created.

```
# buildah from --format=docker --name result-container-2 onbuild-image-2
STEP 1: COPY ./runecho.sh /usr/bin/runecho.sh
STEP 2: RUN /usr/bin/runecho.sh
This is a new container pull ipbabble [ 1 ]
This is a new container pull ipbabble [ 2 ]
This is a new container pull ipbabble [ 3 ]
This is a new container pull ipbabble [ 4 ]
This is a new container pull ipbabble [ 5 ]
This is a new container pull ipbabble [ 6 ]
This is a new container pull ipbabble [ 7 ]
This is a new container pull ipbabble [ 8 ]
This is a new container pull ipbabble [ 9 ]
result-container-2
```
As result-container has the script stored in /usr/bin still and it can be run at anytime.

```
# buildah run result-container-2 /usr/bin/runecho.sh
This is a new container pull ipbabble [ 1 ]
This is a new container pull ipbabble [ 2 ]
This is a new container pull ipbabble [ 3 ]
This is a new container pull ipbabble [ 4 ]
This is a new container pull ipbabble [ 5 ]
This is a new container pull ipbabble [ 6 ]
This is a new container pull ipbabble [ 7 ]
This is a new container pull ipbabble [ 8 ]
This is a new container pull ipbabble [ 9 ]

```
Again these aren't the most extensive examples, but they both illustrate how a master image can be setup and then a secondary container image can then be created with just a few steps.  This way the steps that are set up with the ONBUILD instructions don't have to be typed in each and every time that you need to setup your container.

## Congratulations

Well done. You have learned about Buildah's ONBUILD functionality using this short tutorial. Hopefully you followed along with the examples and found them to be sufficient. Be sure to look at Buildah's man pages to see the other useful commands you can use. Have fun playing.

If you have any suggestions or issues please post them at the [Buildah Issues page](https://github.com/containers/buildah/issues).

For more information on Buildah and how you might contribute please visit the [Buildah home page on GitHub](https://github.com/containers/buildah).

[GitHub]: https://github.com/containers/podman/
[image specification]: https://github.com/opencontainers/runtime-spec
[Introduction Tutorial]: 01-intro.md
[Open Container Initiative]: https://www.opencontainers.org/
