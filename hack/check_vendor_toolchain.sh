#!/bin/bash
topdir=$(dirname ${BASH_SOURCE})/..
wantversion=$(sed -e '/^go /!d' -e '/^go /s,.* ,,g' ${topdir}/go.mod)
goversion=$(go env GOVERSION)
haveversion=${goversion/go}
if test "${wantversion%.*}" != "${haveversion%.*}" ; then
    echo go.mod uses Go "${wantversion%.*}" \("${wantversion}"\), but environment provides "${haveversion%.*}" \("${haveversion}"\).
    echo "$@"
    exit 1
fi
exit 0
