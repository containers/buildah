#!/bin/bash
if pkg-config ostree-1 2> /dev/null ; then
	echo constainers_image_ostree
else
	echo containers_image_ostree_stub
fi
