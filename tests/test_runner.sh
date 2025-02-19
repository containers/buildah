#!/usr/bin/env bash
set -e

cd "$(dirname "$(readlink -f "$BASH_SOURCE")")"

# Default to using /tmp for test space.
export TMPDIR=${TMPDIR:-/tmp}

function execute() {
	>&2 echo "++ $@"
	eval "$@"
}

# Run the tests.
execute time bats -j $(nproc) --tap "${@:-.}"
