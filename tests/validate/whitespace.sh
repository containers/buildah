#!/bin/sh
#
#  Check for one or more whitespace characters at the end of a line in a markdown or text file.
#  gofmt is already going to be doing the same for source code.
#
status=0
if find * -name '*.md' -o -name "*.txt" | grep -v vendor/ | xargs egrep -q '[[:space:]]+$' ; then
	echo "** ERROR: dangling whitespace found in these files: **"
	find * -name '*.md' -o -name "*.txt" | grep -v vendor/ | xargs egrep -n '[[:space:]]+$'
	echo "** ERROR: try running \"sed -i -E -e 's,[[:space:]]+$,,'\" on the affected files **"
	status=1
fi
exit $status
