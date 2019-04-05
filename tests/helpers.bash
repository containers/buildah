#!/bin/bash

BUILDAH_BINARY=${BUILDAH_BINARY:-$(dirname ${BASH_SOURCE})/../buildah}
IMGTYPE_BINARY=${IMGTYPE_BINARY:-$(dirname ${BASH_SOURCE})/../imgtype}
TESTSDIR=${TESTSDIR:-$(dirname ${BASH_SOURCE})}
STORAGE_DRIVER=${STORAGE_DRIVER:-vfs}
PATH=$(dirname ${BASH_SOURCE})/..:${PATH}

# Default timeout for a buildah command.
BUILDAH_TIMEOUT=${BUILDAH_TIMEOUT:-300}

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

    # Remember command args, for possible use in later diagnostic messages
    MOST_RECENT_BUILDAH_COMMAND="buildah $*"

    # stdout is only emitted upon error; this echo is to help a debugger
    echo "\$ $BUILDAH_BINARY $*"
    run timeout --foreground --kill=10 $BUILDAH_TIMEOUT ${BUILDAH_BINARY} --debug --registries-conf ${TESTSDIR}/registries.conf --root ${TESTDIR}/root --runroot ${TESTDIR}/runroot --storage-driver ${STORAGE_DRIVER} "$@"
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

    if [ "$status" -eq 124 -o "$status" -eq 137 ]; then
        # FIXME: 'timeout -v' requires coreutils-8.29; travis seems to have
        #        an older version. If/when travis updates, please add -v
        #        to the 'timeout' command above, and un-comment this out:
        # if expr "$output" : ".*timeout: sending" >/dev/null; then
        echo "*** TIMED OUT ***"
        false
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

###################
#  expect_output  #  Compare actual vs expected string; fail if mismatch
###################
#
# Compares $output against the given string argument. Optional second
# argument is descriptive text to show as the error message (default:
# the command most recently run by 'run_buildah'). This text can be
# useful to isolate a failure when there are multiple identical
# run_buildah invocations, and the difference is solely in the
# config or setup; see, e.g., run.bats:run-cmd().
#
# By default we run an exact string comparison; use --substring to
# look for the given string anywhere in $output.
#
# By default we look in "$output", which is set in run_buildah().
# To override, use --from="some-other-string" (e.g. "${lines[0]}")
#
# Examples:
#
#   expect_output "this is exactly what we expect"
#   expect_output "foo=bar"  "description of this particular test"
#   expect_output --from="${lines[0]}"  "expected first line"
#
function expect_output() {
    # By default we examine $output, the result of run_buildah
    local actual="$output"
    local check_substring=

    # option processing: recognize --from="...", --substring
    local opt
    for opt; do
        local value=$(expr "$opt" : '[^=]*=\(.*\)')
        case "$opt" in
            --from=*)       actual="$value";   shift;;
            --substring)    check_substring=1; shift;;
            --)             shift; break;;
            -*)             die "Invalid option '$opt'" ;;
            *)              break;;
        esac
    done

    local expect="$1"
    local testname="${2:-${MOST_RECENT_BUILDAH_COMMAND:-[no test name given]}}"

    if [ -z "$expect" ]; then
        if [ -z "$actual" ]; then
            return
        fi
        expect='[no output]'
    elif [ "$actual" = "$expect" ]; then
	return
    elif [ -n "$check_substring" ]; then
        if [[ "$actual" =~ $expect ]]; then
            return
        fi
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

#######################
#  expect_line_count  #  Check the expected number of output lines
#######################
#
# ...from the most recent run_buildah command
#
function expect_line_count() {
    local expect="$1"
    local testname="${2:-${MOST_RECENT_BUILDAH_COMMAND:-[no test name given]}}"

    local actual="${#lines[@]}"
    if [ "$actual" -eq "$expect" ]; then
        return
    fi

    printf "#/vvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvv\n"          >&2
    printf "#| FAIL: $testname\n"                                       >&2
    printf "#| Expected %d lines of output, got %d\n" $expect $actual   >&2
    printf "#| Output was:\n"                                           >&2
    local line
    for line in "${lines[@]}"; do
        printf "#| >%s\n" "$line"                                       >&2
    done
    printf "#\\^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^\n"         >&2
    false
}

function check_options_flag_err() {
    flag="$1"
    [ "$status" -eq 1 ]
    [[ $output = *"No options ($flag) can be specified after"* ]]
}
