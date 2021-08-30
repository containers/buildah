#!/usr/bin/env bash

BUILDAH_BINARY=${BUILDAH_BINARY:-$(dirname ${BASH_SOURCE})/../bin/buildah}
IMGTYPE_BINARY=${IMGTYPE_BINARY:-$(dirname ${BASH_SOURCE})/../bin/imgtype}
COPY_BINARY=${COPY_BINARY:-$(dirname ${BASH_SOURCE})/../bin/copy}
TESTSDIR=${TESTSDIR:-$(dirname ${BASH_SOURCE})}
STORAGE_DRIVER=${STORAGE_DRIVER:-vfs}
PATH=$(dirname ${BASH_SOURCE})/../bin:${PATH}
OCI=$(${BUILDAH_BINARY} info --format '{{.host.OCIRuntime}}' || command -v runc || command -v crun)
# Default timeout for a buildah command.
BUILDAH_TIMEOUT=${BUILDAH_TIMEOUT:-300}

# We don't invoke gnupg directly in many places, but this avoids ENOTTY errors
# when we invoke it directly in batch mode, and CI runs us without a terminal
# attached.
export GPG_TTY=/dev/null

function setup(){
    setup_tests
}

function setup_tests() {
    pushd "$(dirname "$(readlink -f "$BASH_SOURCE")")"

    # buildah/podman: "repository name must be lowercase".
    # me: "but it's a local file path, not a repository name!"
    # buildah/podman: "i dont care. no caps anywhere!"
    TESTDIR=$(mktemp -d --dry-run --tmpdir=${BATS_TMPDIR:-${TMPDIR:-/tmp}} buildah_tests.XXXXXX | tr A-Z a-z)
    mkdir --mode=0700 $TESTDIR

    mkdir -p ${TESTDIR}/{root,runroot,sigstore,registries.d}
    cat >${TESTDIR}/registries.d/default.yaml <<EOF
default-docker:
  sigstore-staging: file://${TESTDIR}/sigstore
docker:
  registry.access.redhat.com:
    sigstore: https://access.redhat.com/webassets/docker/content/sigstore
  registry.redhat.io:
    sigstore: https://registry.redhat.io/containers/sigstore
EOF

    # Common options for all buildah and podman invocations
    ROOTDIR_OPTS="--root ${TESTDIR}/root --runroot ${TESTDIR}/runroot --storage-driver ${STORAGE_DRIVER}"
    BUILDAH_REGISTRY_OPTS="--registries-conf ${TESTSDIR}/registries.conf --registries-conf-dir ${TESTDIR}/registries.d --short-name-alias-conf ${TESTDIR}/cache/shortnames.conf"
    PODMAN_REGISTRY_OPTS="--registries-conf ${TESTSDIR}/registries.conf"
}

function starthttpd() {
    pushd ${2:-${TESTDIR}} > /dev/null
    go build -o serve ${TESTSDIR}/serve/serve.go
    portfile=$(mktemp)
    if test -z "${portfile}"; then
        echo error creating temporaty file
        exit 1
    fi
    ./serve ${1:-${BATS_TMPDIR}} 0 ${portfile} &
    HTTP_SERVER_PID=$!
    waited=0
    while ! test -s ${portfile} ; do
        sleep 0.1
        if test $((++waited)) -ge 300 ; then
            echo test http server did not start within timeout
            exit 1
        fi
    done
    HTTP_SERVER_PORT=$(cat ${portfile})
    rm -f ${portfile}
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

function teardown(){
    teardown_tests
}

function teardown_tests() {
    stophttpd

    # Workaround for #1991 - buildah + overlayfs leaks mount points.
    # Many tests leave behind /var/tmp/.../root/overlay and sub-mounts;
    # let's find those and clean them up, otherwise 'rm -rf' fails.
    # 'sort -r' guarantees that we umount deepest subpaths first.
    mount |\
        awk '$3 ~ testdir { print $3 }' testdir="^${TESTDIR}/" |\
        sort -r |\
        xargs --no-run-if-empty --max-lines=1 umount

    rm -fr ${TESTDIR}

    popd
}

function normalize_image_name() {
    for img in "$@"; do
        if [[ "${img##*/}" == "$img" ]] ; then
            echo -n docker.io/library/"$img"
        elif [[ docker.io/"${img##*/}" == "$img" ]] ; then
            echo -n docker.io/library/"${img##*/}"
        else
            echo -n "$img"
        fi
    done
}

function _prefetch() {
    if [ -z "${_BUILDAH_IMAGE_CACHEDIR}" ]; then
        _pgid=$(sed -ne 's/^NSpgid:\s*//p' /proc/$$/status)
        export _BUILDAH_IMAGE_CACHEDIR=${BATS_TMPDIR}/buildah-image-cache.$_pgid
        mkdir -p ${_BUILDAH_IMAGE_CACHEDIR}
    fi

    for img in "$@"; do
        img=$(normalize_image_name "$img")
        echo "# [checking for: $img]" >&2
        fname=$(tr -c a-zA-Z0-9.- - <<< "$img")
        if [ -d $_BUILDAH_IMAGE_CACHEDIR/$fname ]; then
            echo "# [restoring from cache: $_BUILDAH_IMAGE_CACHEDIR / $img]" >&2
            copy dir:$_BUILDAH_IMAGE_CACHEDIR/$fname containers-storage:"$img"
        else
            rm -fr $_BUILDAH_IMAGE_CACHEDIR/$fname
            echo "# [copy docker://$img dir:$_BUILDAH_IMAGE_CACHEDIR/$fname]" >&2
            for attempt in $(seq 3) ; do
                if copy docker://"$img" dir:$_BUILDAH_IMAGE_CACHEDIR/$fname ; then
                    break
                fi
                sleep 5
            done
            echo "# [copy dir:$_BUILDAH_IMAGE_CACHEDIR/$fname containers-storage:$img]" >&2
            copy dir:$_BUILDAH_IMAGE_CACHEDIR/$fname containers-storage:"$img"
        fi
    done
}

function createrandom() {
    dd if=/dev/urandom bs=1 count=${2:-256} of=${1:-${BATS_TMPDIR}/randomfile} status=none
}

function buildah() {
    ${BUILDAH_BINARY} ${BUILDAH_REGISTRY_OPTS} ${ROOTDIR_OPTS} "$@"
}

function imgtype() {
    ${IMGTYPE_BINARY} ${ROOTDIR_OPTS} "$@"
}

function copy() {
    ${COPY_BINARY} --max-parallel-downloads=1 ${ROOTDIR_OPTS} ${BUILDAH_REGISTRY_OPTS} "$@"
}

function podman() {
    command podman ${PODMAN_REGISTRY_OPTS} ${ROOTDIR_OPTS} "$@"
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
    # --retry as first argument = retry 3 times on error (eg registry flakes)
    local expected_rc=0
    local retry=1
    case "$1" in
        [0-9])           expected_rc=$1; shift;;
        [1-9][0-9])      expected_rc=$1; shift;;
        [12][0-9][0-9])  expected_rc=$1; shift;;
        '?')             expected_rc=  ; shift;;  # ignore exit code
        --retry)         retry=3;        shift;;  # retry network flakes
    esac

    # Remember command args, for possible use in later diagnostic messages
    MOST_RECENT_BUILDAH_COMMAND="buildah $*"

    while [ $retry -gt 0 ]; do
        retry=$(( retry - 1 ))

        # stdout is only emitted upon error; this echo is to help a debugger
        echo "\$ $BUILDAH_BINARY $*"
        run env CONTAINERS_CONF=${CONTAINERS_CONF:-$(dirname ${BASH_SOURCE})/containers.conf} timeout --foreground --kill=10 $BUILDAH_TIMEOUT ${BUILDAH_BINARY} ${BUILDAH_REGISTRY_OPTS} ${ROOTDIR_OPTS} "$@"
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
            # This does not get the benefit of a retry
            false
        fi

        if [ -n "$expected_rc" ]; then
            if [ "$status" -eq "$expected_rc" ]; then
                return
            elif [ $retry -gt 0 ]; then
                echo "[ RETRYING ]" >&2
                sleep 30
            else
                die "exit code is $status; expected $expected_rc"
            fi
        fi
    done
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

############
#  assert  #  Compare actual vs expected string; fail if mismatch
############
#
# Compares string (default: $output) against the given string argument.
# By default we do an exact-match comparison against $output, but there
# are two different ways to invoke us, each with an optional description:
#
#      xpect               "EXPECT" [DESCRIPTION]
#      xpect "RESULT" "OP" "EXPECT" [DESCRIPTION]
#
# The first form (one or two arguments) does an exact-match comparison
# of "$output" against "EXPECT". The second (three or four args) compares
# the first parameter against EXPECT, using the given OPerator. If present,
# DESCRIPTION will be displayed on test failure.
#
# Examples:
#
#   xpect "this is exactly what we expect"
#   xpect "${lines[0]}" =~ "^abc"  "first line begins with abc"
#
function assert() {
    local actual_string="$output"
    local operator='=='
    local expect_string="$1"
    local testname="$2"

    case "${#*}" in
        0)   die "Internal error: 'assert' requires one or more arguments" ;;
        1|2) ;;
        3|4) actual_string="$1"
             operator="$2"
             expect_string="$3"
             testname="$4"
             ;;
        *)   die "Internal error: too many arguments to 'assert" ;;
    esac

    # Comparisons.
    # Special case: there is no !~ operator, so fake it via '! x =~ y'
    local not=
    local actual_op="$operator"
    if [[ $operator == '!~' ]]; then
        not='!'
        actual_op='=~'
    fi
    if [[ $operator == '=' || $operator == '==' ]]; then
        # Special case: we can't use '=' or '==' inside [[ ... ]] because
        # the right-hand side is treated as a pattern... and '[xy]' will
        # not compare literally. There seems to be no way to turn that off.
        if [ "$actual_string" = "$expect_string" ]; then
            return
        fi
    else
        if eval "[[ $not \$actual_string $actual_op \$expect_string ]]"; then
            return
        elif [ $? -gt 1 ]; then
            die "Internal error: could not process 'actual' $operator 'expect'"
        fi
    fi

    # Test has failed. Get a descriptive test name.
    if [ -z "$testname" ]; then
        testname="${MOST_RECENT_BUILDAH_COMMAND:-[no test name given]}"
    fi

    # Display optimization: the typical case for 'expect' is an
    # exact match ('='), but there are also '=~' or '!~' or '-ge'
    # and the like. Omit the '=' but show the others; and always
    # align subsequent output lines for ease of comparison.
    local op=''
    local ws=''
    if [ "$operator" != '==' ]; then
        op="$operator "
        ws=$(printf "%*s" ${#op} "")
    fi

    # This is a multi-line message, which may in turn contain multi-line
    # output, so let's format it ourself, readably
    local actual_split
    IFS=$'\n' read -rd '' -a actual_split <<<"$actual_string" || true
    printf "#/vvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvv\n"    >&2
    printf "#|     FAIL: %s\n" "$testname"                        >&2
    printf "#| expected: %s'%s'\n" "$op" "$expect_string"         >&2
    printf "#|   actual: %s'%s'\n" "$ws" "${actual_split[0]}"     >&2
    local line
    for line in "${actual_split[@]:1}"; do
        printf "#|         > %s'%s'\n" "$ws" "$line"              >&2
    done
    printf "#\\^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^\n"   >&2
    false
}

###################
#  expect_output  #  [obsolete; kept for compatibility]
###################
#
# An earlier version of assert().
#
function expect_output() {
    # By default we examine $output, the result of run_buildah
    local actual="$output"
    local operator='=='

    # option processing: recognize --from="...", --substring
    local opt
    for opt; do
        local value=$(expr "$opt" : '[^=]*=\(.*\)')
        case "$opt" in
            --from=*)       actual="$value";   shift;;
            --substring)    operator='=~';     shift;;
            --)             shift; break;;
            -*)             die "Invalid option '$opt'" ;;
            *)              break;;
        esac
    done

    assert "$actual" "$operator" "$@"
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
    [ "$status" -eq 125 ]
    [[ $output = *"No options ($flag) can be specified after"* ]]
}

#################
#  is_rootless  #  Check if we run as normal user
#################
function is_rootless() {
    [ "$(id -u)" -ne 0 ]
}

####################
#  skip_if_chroot  #
####################
function skip_if_chroot() {
    if test "$BUILDAH_ISOLATION" = "chroot"; then
        skip "${1:-test does not work when \$BUILDAH_ISOLATION = chroot}"
    fi
}

######################
#  skip_if_rootless  #
######################
function skip_if_rootless() {
    if test "$BUILDAH_ISOLATION" = "rootless"; then
        skip "${1:-test does not work when \$BUILDAH_ISOLATION = rootless}"
    fi
}

########################
#  skip_if_no_runtime  #  'buildah run' can't work without a runtime
########################
function skip_if_no_runtime() {
    if type -p "${OCI}" &> /dev/null; then
        return
    fi

    skip "runtime \"$OCI\" not found"
}

##################
#  is_cgroupsv2  #  Returns true if host system has cgroupsv2 enabled
##################
function is_cgroupsv2() {
    local cgroupfs_t=$(stat -f -c %T /sys/fs/cgroup)
    test "$cgroupfs_t" = "cgroup2fs"
}

#######################
#  skip_if_cgroupsv2  #  Some tests don't work with cgroupsv2
#######################
function skip_if_cgroupsv2() {
    if is_cgroupsv2; then
        skip "${1:-test does not work with cgroups v2}"
    fi
}

##########################
#  skip_if_in_container  #
##########################
function skip_if_in_container() {
    if test "$CONTAINER" = "podman"; then
        skip "This test is not working inside a container"
    fi
}

#######################
#  skip_if_no_docker  #
#######################
function skip_if_no_docker() {
  which docker                  || skip "docker is not installed"
  systemctl -q is-active docker || skip "docker.service is not active"

  # Confirm that this is really truly docker, not podman.
  docker_version=$(docker --version)
  if [[ $docker_version =~ podman ]]; then
    skip "this test needs actual docker, not podman-docker"
  fi
}
