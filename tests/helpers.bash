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

function setup() {
    pushd "$(dirname "$(readlink -f "$BASH_SOURCE")")"

    suffix=$(dd if=/dev/urandom bs=12 count=1 status=none | od -An -tx1 | sed -e 's, ,,g')
    TESTDIR=${BATS_TMPDIR}/tmp${suffix}
    rm -fr ${TESTDIR}
    mkdir -p ${TESTDIR}/{root,runroot,sigstore,registries.d,cache}
    echo "default-docker:                                                           " >> ${TESTDIR}/registries.d/default.yaml
    echo "  sigstore-staging: file://${TESTDIR}/sigstore                            " >> ${TESTDIR}/registries.d/default.yaml
    echo "docker:                                                                   " >> ${TESTDIR}/registries.d/default.yaml
    echo "  registry.access.redhat.com:                                             " >> ${TESTDIR}/registries.d/default.yaml
    echo "    sigstore: https://access.redhat.com/webassets/docker/content/sigstore " >> ${TESTDIR}/registries.d/default.yaml
    echo "  registry.redhat.io:                                                     " >> ${TESTDIR}/registries.d/default.yaml
    echo "    sigstore: https://registry.redhat.io/containers/sigstore              " >> ${TESTDIR}/registries.d/default.yaml
    # Common options for all buildah and podman invocations
    ROOTDIR_OPTS="--root ${TESTDIR}/root --runroot ${TESTDIR}/runroot --storage-driver ${STORAGE_DRIVER}"
    BUILDAH_REGISTRY_OPTS="--registries-conf ${TESTSDIR}/registries.conf --registries-conf-dir ${TESTDIR}/registries.d --short-name-alias-conf ${TESTDIR}/cache/shortnames.conf"
    PODMAN_REGISTRY_OPTS="--registries-conf ${TESTSDIR}/registries.conf"
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
    stop_git_daemon

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
    ${COPY_BINARY} ${ROOTDIR_OPTS} ${BUILDAH_REGISTRY_OPTS} "$@"
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
    # output, so let's format it ourselves, readably
    local -a actual_split
    readarray -t actual_split <<<"$actual"
    printf "#/vvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvv\n" >&2
    printf "#|     FAIL: %s\n" "$testname"                     >&2
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
    [ "$status" -eq 125 ]
    [[ $output = *"No options ($flag) can be specified after"* ]]
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

function start_git_daemon() {
  daemondir=${TESTDIR}/git-daemon
  mkdir -p ${daemondir}/repo
  gzip -dc < ${1:-${TESTSDIR}/git-daemon/repo.tar.gz} | tar x -C ${daemondir}/repo
  GITPORT=$(($RANDOM + 32768))
  git daemon --detach --pid-file=${TESTDIR}/git-daemon/pid --reuseaddr --port=${GITPORT} --base-path=${daemondir} ${daemondir}
}

function stop_git_daemon() {
  if test -s ${TESTDIR}/git-daemon/pid ; then
    kill $(cat ${TESTDIR}/git-daemon/pid)
    rm -f ${TESTDIR}/git-daemon/pid
  fi
}
