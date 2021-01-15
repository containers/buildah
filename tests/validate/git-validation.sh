#!/bin/bash

set -e

REPO_ROOT=`realpath $(dirname $0)/../../`
TOOL_DIR="$REPO_ROOT/tests/tools/build"

export PATH="$TOOL_DIR:$PATH"
if [[ -z "$(type -P git-validation)" ]]; then
	echo git-validation is not in PATH "$PATH".
	exit 1
fi

GITVALIDATE_EPOCH="${GITVALIDATE_EPOCH:-474febf0dcb3c76c23a88f3b83834c81f20a7e5c}"

OUTPUT_OPTIONS="-q"
if [[ "$CI" == 'true' ]]; then
    OUTPUT_OPTIONS="-v"
fi

set -x
exec git-validation \
    $OUTPUT_OPTIONS \
    -run DCO,short-subject \
    ${GITVALIDATE_EPOCH:+-range "${GITVALIDATE_EPOCH}..${GITVALIDATE_TIP:-HEAD}"} \
    ${GITVALIDATE_FLAGS}
