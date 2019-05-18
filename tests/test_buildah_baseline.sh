#!/bin/bash
# test_buildah_baseline.sh 
# A script to be run at the command line with Buildah installed.
# This should be run against a new kit to provide base level testing
# on a freshly installed machine with no images or containers in 
# play.  This currently needs to be run as root.
#
# Commands based on the tutorial provided by William Henry.
#
# To run this command:
#
# /bin/bash -v test_buildah_baseline.sh 

########
# Next two commands should return blanks 
########
buildah images
buildah containers

########
# Run ls in redis container, this should work 
########
ctrid=$(buildah from registry.access.redhat.com/rhscl/redis-32-rhel7)
buildah run $ctrid ls /


########
# Validate touch works after installing httpd, solved selinux
# issue that should now work.
########
ctr=$(buildah from scratch)
mnt=$(buildah mount $ctr)
dnf -y install --installroot=$mnt --releasever=30 httpd
buildah run $ctr touch /test

########
# Create Fedora based container
########
container=$(buildah from fedora)
echo $container

########
# Run container and display contents in /etc 
########
buildah run $container -- ls -alF /etc

########
# Run Java in the container - should FAIL
########
buildah run $container java

########
# Install java onto the container
########
buildah run $container -- dnf -y install java

########
# Run Java in the container - should show java usage
########
buildah run $container java

########
# Create a scratch container 
########
newcontainer=$(buildah from scratch)

########
# Check and find two containers
########
buildah containers

########
# Check images, no "scratch" image
########
buildah images

########
# Run the container - should FAIL
########
buildah run $newcontainer bash

########
# Mount the container's root file system
########
scratchmnt=$(buildah mount $newcontainer)

########
# Show the location, should be /var/lib/containers/storage/overlay/{id}/dif
########
echo $scratchmnt

########
# Install Fedora 30 bash and coreutils
########
dnf install --installroot $scratchmnt --release 30 bash coreutils --setopt install_weak_deps=false -y

########
# Check /usr/bin on the new container
########
buildah run $newcontainer -- ls -alF /usr/bin

########
# Create shell script to test on
########
FILE=./runecho.sh
/bin/cat <<EOM >$FILE
#!/bin/bash
for i in {1..9};
do
    echo "This is a new container from ipbabble [" \$i "]"
done
EOM
chmod +x $FILE

########
# Copy and run file on scratch container 
########
buildah copy $newcontainer $FILE /usr/bin
buildah config --cmd /usr/bin/runecho.sh $newcontainer
buildah run $newcontainer /usr/bin/runecho.sh 

########
# Add configuration information
########
buildah config --created-by "ipbabble"  $newcontainer
buildah config --author "wgh at redhat.com @ipbabble" --label name=fedora30-bashecho $newcontainer

########
# Inspect the container, verifying above was put into it
########
buildah inspect $newcontainer

########
# Unmount the container
########
buildah unmount $newcontainer

########
# Commit the image
########
buildah commit $newcontainer fedora-bashecho

########
# Check the images there should be a fedora-bashecho:latest image
########
buildah images

########
# Inspect the fedora-bashecho image
########
buildah inspect --type=image fedora-bashecho

########
# Remove the container
########
buildah rm $newcontainer

########
# Install Docker, but not for long!
########
dnf -y install docker
systemctl start docker

########
# Push fedora-bashecho to the Docker daemon 
########
buildah push fedora-bashecho docker-daemon:fedora-bashecho:latest

########
# Run fedora-bashecho from Docker
########
docker run fedora-bashecho

########
# Time to remove Docker
########
dnf -y remove docker

########
# Build Dockerfiles for OnBuild Test
# (Thanks @clcollins!)
########
FILE=./Dockerfile
/bin/cat <<EOM >$FILE
FROM alpine
RUN touch /foo
ONBUILD RUN touch /bar
EOM
chmod +x $FILE

FILE=./Dockerfile-2
/bin/cat <<EOM >$FILE
FROM onbuild-image
RUN touch /baz
EOM
chmod +x $FILE

########
# Build with Dockerfiles 
########
buildah bud -f ./Dockerfile --format=docker -t onbuild-image .
buildah bud -f ./Dockerfile-2 --format=docker -t result-image .

########
# Build a container to see if the /bar file has been created. 
########
ctr=$(buildah from result-image)

########
# Validate that the /bar file has been created in the container. 
########
buildah run $ctr ls -alF /bar /foo /baz

########
# Build Dockerfile for WhaleSays
########
FILE=./Dockerfile
/bin/cat <<EOM >$FILE
FROM docker/whalesay:latest
RUN apt-get -y update && apt-get install -y fortunes
CMD /usr/games/fortune -a | cowsay
EOM
chmod +x $FILE

########
# Build with the Dockerfile
########
buildah bud -f Dockerfile -t whale-says . 

########
# Create a whalesays container 
########
whalesays=$(buildah from whale-says)

########
# Run the container to see what the whale says
########
buildah run $whalesays bash -c '/usr/games/fortune -a | cowsay'

########
# Clean up Buildah
########
buildah rm --all
buildah rmi --all
