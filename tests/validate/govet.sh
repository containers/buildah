#!/bin/bash
while ! test -x ./btrfs_tag.sh ; do
	cd ..
done
if ! test -x ./btrfs_tag.sh ; then
	echo govet.sh unable to locate top-level directory, failing.
	exit 1
fi
tags="$(./btrfs_tag.sh) $(./btrfs_installed_tag.sh) $(./libdm_tag.sh) $(./ostree_tag.sh) $(./selinux_tag.sh)"

for package in $(go list ./... | grep -v /vendor/ | grep -v /docker) ; do
	if ! go vet -tags "$tags" ${package} ; then
		echo Error: source package ${package} does not pass go vet.
		exit 1
	fi
done
exit 0
