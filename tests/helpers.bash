#!/bin/bash

BUILDAH_BINARY=${BUILDAH_BINARY:-$(dirname ${BASH_SOURCE})/../buildah}
IMGTYPE_BINARY=${IMGTYPE_BINARY:-$(dirname ${BASH_SOURCE})/../imgtype}
TESTSDIR=${TESTSDIR:-$(dirname ${BASH_SOURCE})}
STORAGE_DRIVER=${STORAGE_DRIVER:-vfs}
PATH=$(dirname ${BASH_SOURCE})/..:${PATH}

# Default timeout for a buildah command.
BUILDAH_TIMEOUT=${BUILDAH_TIMEOUT:-120}

function setup() {
	suffix=$(dd if=/dev/urandom bs=12 count=1 status=none | od -An -tx1 | sed -e 's, ,,g')
	TESTDIR=${BATS_TMPDIR}/tmp${suffix}
	rm -fr ${TESTDIR}
	mkdir -p ${TESTDIR}/{root,runroot}
}

function starthttpd() {
	pushd ${2:-${TESTDIR}} > /dev/null
	go build -o serve ${TESTSDIR}/serve/serve.go
	HTTP_SERVER_PORT=$((RANDOM+32768))
	./serve ${HTTP_SERVER_PORT} ${1:-${BATS_TMPDIR}} &
	HTTP_SERVER_PID=$!
	popd > /dev/null
}

function stophttpd() {
	if test -n "$HTTP_SERVER_PID" ; then
		kill -HUP ${HTTP_SERVER_PID}
		unset HTTP_SERVER_PID
		unset HTTP_SERVER_PORT
	fi
	true
}

function teardown() {
	stophttpd
	rm -fr ${TESTDIR}
}

function createrandom() {
	dd if=/dev/urandom bs=1 count=${2:-256} of=${1:-${BATS_TMPDIR}/randomfile} status=none
}

function buildah() {
	${BUILDAH_BINARY} --debug --registries-conf ${TESTSDIR}/registries.conf --root ${TESTDIR}/root --runroot ${TESTDIR}/runroot --storage-driver ${STORAGE_DRIVER} "$@"
}

function imgtype() {
	${IMGTYPE_BINARY} -debug -root ${TESTDIR}/root -runroot ${TESTDIR}/runroot -storage-driver ${STORAGE_DRIVER} "$@"
}

#################
#  run_buildah  #  Invoke buildah, with timeout, using BATS 'run'
#################
#
# This is the preferred mechanism for invoking buildah:
#
#  * we use 'timeout' to abort (with a diagnostic) if something
#    takes too long; this is preferable to a CI hang.
#  * we log the command run and its output. This doesn't normally
#    appear in BATS output, but it will if there's an error.
#  * we check exit status. Since the normal desired code is 0,
#    that's the default; but the first argument can override:
#
#     run_buildah 125  nonexistent-subcommand
#     run_buildah '?'  some-other-command       # let our caller check status
#
# Since we use the BATS 'run' mechanism, $output and $status will be
# defined for our caller.
#
function run_buildah() {
    # Number as first argument = expected exit code; default 0
    expected_rc=0
    case "$1" in
        [0-9])           expected_rc=$1; shift;;
        [1-9][0-9])      expected_rc=$1; shift;;
        [12][0-9][0-9])  expected_rc=$1; shift;;
        '?')             expected_rc=  ; shift;;  # ignore exit code
    esac

    # stdout is only emitted upon error; this echo is to help a debugger
    echo "\$ $BUILDAH_BINARY $*"
    run timeout --foreground -v --kill=10 $BUILDAH_TIMEOUT ${BUILDAH_BINARY} --debug --registries-conf ${TESTSDIR}/registries.conf --root ${TESTDIR}/root --runroot ${TESTDIR}/runroot --storage-driver ${STORAGE_DRIVER} "$@"
    # without "quotes", multiple lines are glommed together into one
    if [ -n "$output" ]; then
        echo "$output"
    fi
    if [ "$status" -ne 0 ]; then
        echo -n "[ rc=$status ";
        if [ -n "$expected_rc" ]; then
            if [ "$status" -eq "$expected_rc" ]; then
                echo -n "(expected) ";
            else
                echo -n "(** EXPECTED $expected_rc **) ";
            fi
        fi
        echo "]"
    fi

    if [ "$status" -eq 124 ]; then
        if expr "$output" : ".*timeout: sending" >/dev/null; then
            echo "*** TIMED OUT ***"
            false
        fi
    fi

    if [ -n "$expected_rc" ]; then
        if [ "$status" -ne "$expected_rc" ]; then
            die "exit code is $status; expected $expected_rc"
        fi
    fi
}

#########
#  die  #  Abort with helpful message
#########
function die() {
    echo "#/vvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvv"  >&2
    echo "#| FAIL: $*"                                           >&2
    echo "#\\^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^" >&2
    false
}

########
#  is  #  Compare actual vs expected string; fail w/diagnostic if mismatch
########
#
# Compares given string against expectations, using 'expr' to allow patterns.
#
# Examples:
#
#   is "$actual" "$expected" "descriptive test name"
#   is "apple" "orange"  "name of a test that will fail in most universes"
#   is "apple" "[a-z]\+" "this time it should pass"
#
function is() {
    local actual="$1"
    local expect="$2"
    local testname="${3:-FIXME}"

    if [ -z "$expect" ]; then
        if [ -z "$actual" ]; then
            return
        fi
        expect='[no output]'
    elif [ "$actual" = "$expect" ]; then
	return
    elif expr "$actual" : "$expect" >/dev/null; then
        return
    fi

    # This is a multi-line message, which may in turn contain multi-line
    # output, so let's format it ourself, readably
    local -a actual_split
    readarray -t actual_split <<<"$actual"
    printf "#/vvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvv\n" >&2
    printf "#|     FAIL: $testname\n"                          >&2
    printf "#| expected: '%s'\n" "$expect"                     >&2
    printf "#|   actual: '%s'\n" "${actual_split[0]}"          >&2
    local line
    for line in "${actual_split[@]:1}"; do
        printf "#|         > '%s'\n" "$line"                   >&2
    done
    printf "#\\^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^\n" >&2
    false
}


function check_options_flag_err() {
    flag="$1"
    [ "$status" -eq 1 ]
    [[ $output = *"No options ($flag) can be specified after"* ]]
}
