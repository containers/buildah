#!/bin/bash -x
set -e
read
export PATH=`pwd`:$PATH
systemctl restart ocid
read
: "[1m Check if we have some images to work with.[0m"
ocic image list
read
: "[1m Create a working container, and capture its name [0m"
read
echo '[container1=`buildah from ${1:-ubuntu}`]'
container1=`buildah from ${1:-ubuntu}`
read
: "[1m Mount that working container, and capture the mountpoint [0m"
read
echo '[mountpoint1=`buildah mount $container1`]'
mountpoint1=`buildah mount $container1`
read
: "[1m Add a file to the container [0m"
read
echo '[echo yay > $mountpoint1/file-in-root]'
echo yay > $mountpoint1/file-in-root
read
: "[1m Produce an image from the container [0m"
read
buildah commit "$container1" ${2:-first-new-image}
read
: "[1m Verify that our new image is there [0m"
read
ocic image list
read
: "[1m Unmount our working container and delete it [0m"
read
buildah umount "$container1"
buildah rm "$container1"
read
: "[1m Now try it with ocid not running! [0m"
read
systemctl stop ocid
read
: "[1m You know what?  Go ahead and use that image we just created, and capture its name [0m"
read
echo '[container2=`buildah from ${2:-first-new-image}`]'
container2=`buildah from ${2:-first-new-image}`
read
: "[1m Mount that new working container, and capture the mountpoint [0m"
read
echo '[mountpoint2=`buildah mount $container2`]'
mountpoint2=`buildah mount $container2`
read
: "[1m That file we added to the image is there, right? [0m"
read
cat $mountpoint2/file-in-root
read
: "[1m Add a file to the new container[0m"
read
echo '[echo yay > $mountpoint2/another-file-in-root]'
echo yay > $mountpoint2/another-file-in-root
read
: "[1m Produce an image from the new container[0m"
read
buildah commit "$container2" ${3:-second-new-image}
read
: "[1m Unmount our new working container and delete it [0m"
read
buildah umount "$container2"
buildah rm "$container2"
read
: "[1m Verify that our new new image is there[0m"
read
systemctl start ocid
ocic image list
read
: "[1m Clean up, because I ran this like fifty times while testing [0m"
read
ocic image remove --id=${2:-first-new-image}
ocic image remove --id=${3:-second-new-image}
