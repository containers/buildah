#!/bin/bash
for package in $(go list ./... | grep -v /vendor/) ; do
	if ! go vet ${package} ; then
		echo Error: source package ${package} does not pass go vet.
		exit 1
	fi
done
exit 0
