#!/bin/bash
if test $(find . -name "*.go" -not -path "*/vendor/*" -print0 | xargs -n 1 -0 gofmt -s -l | wc -l) -ne 0 ; then
	echo Error: source files are not formatted according to recommendations.  Run \"gofmt -s -w\" on:
	find . -name "*.go" -not -path "*/vendor/*" -print0 | xargs -n 1 -0 gofmt -s -l
	exit 1
fi
exit 0
