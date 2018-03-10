#!/bin/bash -x

ctr1=`buildah from ${1:-fedora}`

## Get all updates and install our minimal httpd server
buildah run $ctr1 -- dnf update -y
buildah run $ctr1 -- dnf install -y lighttpd

## Include some buildtime annotations
buildah config --annotation "com.example.build.host=$(uname -n)" $ctr1

## Run our server and expose the port
buildah config --cmd "/usr/sbin/lighttpd -D -f /etc/lighttpd/lighttpd.conf" $ctr1
buildah config --port 80 $ctr1

## Commit this container to an image name
buildah commit $ctr1 ${2:-$USER/lighttpd}
