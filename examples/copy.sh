#!/bin/bash -x
set -e
: "[1m Build a temporary directory; make sure ocid is running.[0m"
export PATH=`pwd`:$PATH
d=`mktemp -d`
trap 'cd /;rm -fr "$d"' EXIT
cd "$d"
systemctl restart ocid
read
: "[1m Check if we have some images to work with.[0m"
read
ocic image list
read
: "[1m Create a working container, and capture its name [0m"
read
echo '[container1=`buildah from ${1:-alpine}`]'
container1=`buildah from ${1:-alpine}`
read
: "[1m Mount that working container, and capture the mountpoint [0m"
read
echo '[mountpoint1=`buildah mount $container1`]'
mountpoint1=`buildah mount $container1`
read
: "[1m List random files in the container [0m"
read
echo '[find $mountpoint1 -name "random*"]'
find $mountpoint1 -name "random*"
read
: "[1m Ensure the default destination for copying files is / [0m"
read
echo '[buildah config $container1 --workingdir /]'
buildah config $container1 --workingdir /
read
: "[1m Add a file to the container [0m"
read
echo '[dd if=/dev/urandom of=random1 bs=512 count=1]'
echo '[buildah copy $container1 random1]'
dd if=/dev/urandom of=random1 bs=512 count=1
buildah copy $container1 random1
read
: "[1m Change the default destination for copying files [0m"
read
echo '[buildah config $container1 --workingdir /tmp]'
buildah config $container1 --workingdir /tmp
read
: "[1m Add another new file to the container [0m"
read
echo '[dd if=/dev/urandom of=random2 bs=512 count=1]'
echo '[buildah copy $container1 random2]'
dd if=/dev/urandom of=random2 bs=512 count=1
buildah copy $container1 random2
read
: "[1m Copy a subdirectory with some files in it [0m"
read
echo '[mkdir -p randomsubdir]'
echo '[dd if=/dev/urandom of=randomsubdir/random3 bs=512 count=1]'
echo '[dd if=/dev/urandom of=randomsubdir/random4 bs=512 count=1]'
echo '[buildah copy $container1 randomsubdir]'
mkdir -p randomsubdir
dd if=/dev/urandom of=randomsubdir/random3 bs=512 count=1
dd if=/dev/urandom of=randomsubdir/random4 bs=512 count=1
buildah copy $container1 randomsubdir
read
: "[1m List some of the container's contents [0m"
read
echo '[find $mountpoint1 -name "random*"]'
find $mountpoint1 -name "random*"
read
: "[1m Download a tarball [0m"
read
echo '[wget -c https://releases.pagure.org/tmpwatch/tmpwatch-2.9.17.tar.bz2]'
wget -c https://releases.pagure.org/tmpwatch/tmpwatch-2.9.17.tar.bz2
read
: "[1m Copy that tarball to the container [0m"
read
echo '[mkdir -p $mountpoint1/tmpwatch]'
echo '[buildah copy $container1 --dest /tmpwatch tmpwatch-2.9.17.tar.bz2]'
mkdir -p $mountpoint1/tmpwatch
buildah copy $container1 --dest /tmpwatch tmpwatch-2.9.17.tar.bz2
read
: "[1m Download another tarball to the container [0m"
read
echo '[buildah copy $container1 --dest /tmpwatch https://releases.pagure.org/newt/newt-0.52.19.tar.gz]'
buildah copy $container1 --dest /tmpwatch https://releases.pagure.org/newt/newt-0.52.19.tar.gz
read
: "[1m List the contents of the target directory [0m"
read
echo '[find $mountpoint1/tmpwatch]'
find $mountpoint1/tmpwatch
read
: "[1m Now 'add' the downloaded tarball to the container [0m"
read
echo '[buildah add $container1 --dest /tmpwatch tmpwatch-2.9.17.tar.bz2]'
buildah add $container1 --dest /tmpwatch tmpwatch-2.9.17.tar.bz2
read
: "[1m List the contents of the target directory again [0m"
read
echo '[find $mountpoint1/tmpwatch]'
find $mountpoint1/tmpwatch
read
: "[1m Clean up, because I ran this like fifty times while testing [0m"
read
echo '[buildah delete $container1]'
buildah rm $container1
