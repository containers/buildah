#!/bin/sh

VERSION=$(grep "Version =" buildah.go | sed -e 's/\tVersion = //; s/"//g')
git archive --format=tar.gz --prefix=buildah_$VERSION.orig/ HEAD \
    > ../buildah_$VERSION.orig.tar.gz
