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
    stop_git_daemon
    stop_registry

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

    local storage=
    for img in "$@"; do
        if [[ "$img" =~ '[vfs@' ]] ; then
            storage="$img"
            continue
        fi
        img=$(normalize_image_name "$img")
        echo "# [checking for: $img]" >&2
        fname=$(tr -c a-zA-Z0-9.- - <<< "$img")
        if [ -d $_BUILDAH_IMAGE_CACHEDIR/$fname ]; then
            echo "# [restoring from cache: $_BUILDAH_IMAGE_CACHEDIR / $img]" >&2
            copy dir:$_BUILDAH_IMAGE_CACHEDIR/$fname containers-storage:"$storage""$img"
        else
            rm -fr $_BUILDAH_IMAGE_CACHEDIR/$fname
            echo "# [copy docker://$img dir:$_BUILDAH_IMAGE_CACHEDIR/$fname]" >&2
            for attempt in $(seq 3) ; do
                if copy docker://"$img" dir:$_BUILDAH_IMAGE_CACHEDIR/$fname ; then
                    break
                fi
                sleep 5
            done
            echo "# [copy dir:$_BUILDAH_IMAGE_CACHEDIR/$fname containers-storage:$storage$img]" >&2
            copy dir:$_BUILDAH_IMAGE_CACHEDIR/$fname containers-storage:"$storage""$img"
        fi
    done
}

function createrandom() {
    dd if=/dev/urandom bs=1 count=${2:-256} of=${1:-${BATS_TMPDIR}/randomfile} status=none
}

###################
#  random_string  #  Returns a pseudorandom human-readable string
###################
#
# Numeric argument, if present, is desired length of string
#
function random_string() {
    local length=${1:-10}

    head /dev/urandom | tr -dc a-zA-Z0-9 | head -c$length
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
    command ${PODMAN_BINARY:-podman} ${PODMAN_REGISTRY_OPTS} ${ROOTDIR_OPTS} "$@"
}

# There are various scenarios where we would like to execute `tests` as rootless user, however certain commands like `buildah mount`
# do not work in rootless session since a normal user cannot mount a filesystem unless they're in a user namespace along with its
# own mount namespace. In order to run such specific commands from a rootless session we must perform `buildah unshare`.
# Following function makes sure that invoked command is triggered inside a `buildah unshare` session if env is rootless.
function run_unshared() {
    if is_rootless; then
        $BUILDAH_BINARY unshare "$@"
    else
        command "$@"
    fi
}

function mkdir() {
    run_unshared mkdir "$@"
}

function touch() {
    run_unshared touch "$@"
}

function cp() {
    run_unshared cp "$@"
}

function rm() {
    run_unshared rm "$@"
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

    # If session is rootless and `buildah mount` is invoked, perform unshare,
    # since normal user cannot mount a filesystem unless they're in a user namespace along with its own mount namespace.
    if is_rootless; then
        if [[ "$1" =~ mount ]]; then
            set "unshare" "$BUILDAH_BINARY" ${BUILDAH_REGISTRY_OPTS} ${ROOTDIR_OPTS} "$@"
        fi
    fi

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
    elif [[ $operator == '!=' ]]; then
        # Same special case as above
        if [ "$actual_string" != "$expect_string" ]; then
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

#################################
#  skip_if_rootless_environment # `mount` or its variant needs unshare
#################################
function skip_if_rootless_environment() {
    if is_rootless; then
        skip "${1:-test is being invoked from rootless environment and might need unshare}"
    fi
}

#################################
#  skip_if_root_environment     #
#################################
function skip_if_root_environment() {
    if ! is_rootless; then
        skip "${1:-test is being invoked from root environment}"
    fi
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

##################################
#  skip_if_rootless_and_cgroupv1 #
##################################
function skip_if_rootless_and_cgroupv1() {
    if test "$BUILDAH_ISOLATION" = "rootless"; then
        if ! is_cgroupsv2; then
            skip "${1:-test does not work when \$BUILDAH_ISOLATION = rootless} and not cgroupv2"
        fi
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

#######################
#  skip_if_no_podman  #  we need 'podman' to test how we interact with podman
#######################
function skip_if_no_podman() {
    run which ${PODMAN_BINARY:-podman}
    if [[ $status -ne 0 ]]; then
        skip "podman is not installed"
    fi
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

#######################
#  skip_if_cgroupsv1  #  Some tests don't work with cgroupsv1
#######################
function skip_if_cgroupsv1() {
    if ! is_cgroupsv2; then
        skip "${1:-test does not work with cgroups v1}"
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

# Bring up a registry server using buildah with vfs and chroot as a cheap
# substitute for podman, accessible only to user $1 using password $2 on the
# local system at a dynamically-allocated port.
# Requires openssl.
# A user name and password can be supplied as the two parameters, or default
# values of "testuser" and "testpassword" will be used.
# Sets REGISTRY_PID, REGISTRY_PORT (to append to "localhost:"), and
# REGISTRY_DIR (where the CA cert can be found) on success.
function start_registry() {
  local testuser="${1:-testuser}"
  local testpassword="${2:-testpassword}"
  local REGISTRY_IMAGE=quay.io/libpod/registry:2.8
  local config='
version: 0.1
log:
  fields:
    service: registry
storage:
  cache:
    blobdescriptor: inmemory
  filesystem:
    rootdirectory: /var/lib/registry
http:
  addr: :0
  headers:
    X-Content-Type-Options: [nosniff]
  tls:
    certificate: /etc/docker/registry/localhost.crt
    key: /etc/docker/registry/localhost.key
health:
  storagedriver:
    enabled: true
    interval: 10s
    threshold: 3
auth:
  htpasswd:
    realm: buildah-realm
    path: /etc/docker/registry/htpasswd
'
  # roughly equivalent to "htpasswd -nbB testuser testpassword", the registry uses
  # the same package this does for verifying passwords against hashes in htpasswd files
  htpasswd=${testuser}:$(buildah passwd ${testpassword})

  # generate the htpasswd and config.yml files for the registry
  mkdir -p "${TESTDIR}"/registry/root "${TESTDIR}"/registry/run "${TESTDIR}"/registry/certs "${TESTDIR}"/registry/config
  cat > "${TESTDIR}"/registry/config/htpasswd <<< "$htpasswd"
  cat > "${TESTDIR}"/registry/config/config.yml <<< "$config"
  chmod 644 "${TESTDIR}"/registry/config/htpasswd "${TESTDIR}"/registry/config/config.yml

  # generate a new key and certificate
  if ! openssl req -newkey rsa:4096 -nodes -sha256 -keyout "${TESTDIR}"/registry/certs/localhost.key -x509 -days 2 -addext "subjectAltName = DNS:localhost" -out "${TESTDIR}"/registry/certs/localhost.crt -subj "/CN=localhost" ; then
    die error creating new key and certificate
  fi
  chmod 644 "${TESTDIR}"/registry/certs/localhost.crt
  chmod 600 "${TESTDIR}"/registry/certs/localhost.key
  # use a copy of the server's certificate for validation from a client
  cp "${TESTDIR}"/registry/certs/localhost.crt "${TESTDIR}"/registry/

  # create a container in its own storage
  _prefetch "[vfs@${TESTDIR}/registry/root+${TESTDIR}/registry/run]" ${REGISTRY_IMAGE}
  ctr=$(${BUILDAH_BINARY} --storage-driver vfs --root "${TESTDIR}"/registry/root --runroot "${TESTDIR}"/registry/run from --quiet --pull-never ${REGISTRY_IMAGE})
  ${BUILDAH_BINARY} --storage-driver vfs --root "${TESTDIR}"/registry/root --runroot "${TESTDIR}"/registry/run copy $ctr "${TESTDIR}"/registry/config/htpasswd "${TESTDIR}"/registry/config/config.yml "${TESTDIR}"/registry/certs/localhost.key "${TESTDIR}"/registry/certs/localhost.crt /etc/docker/registry/

  # fire it up
  coproc ${BUILDAH_BINARY} --storage-driver vfs --root "${TESTDIR}"/registry/root --runroot "${TESTDIR}"/registry/run run --net host "$ctr" /entrypoint.sh /etc/docker/registry/config.yml 2> "${TESTDIR}"/registry/registry.log

  # record the coprocess's ID and try to parse the listening port from the log
  # we're separating all of this from the storage for any test that might call
  # this function and using vfs to minimize the cleanup required
  REGISTRY_PID="${COPROC_PID}"
  REGISTRY_DIR="${TESTDIR}"/registry
  REGISTRY_PORT=
  local waited=0
  while [ -z "${REGISTRY_PORT}" ] ; do
    if [ $waited -ge $BUILDAH_TIMEOUT ] ; then
      echo Could not determine listening port from log:
      sed -e 's/^/  >/' ${TESTDIR}/registry/registry.log
      stop_registry
      false
    fi
    waited=$((waited+1))
    sleep 1
    REGISTRY_PORT=$(sed -ne 's^.*listening on.*:\([0-9]\+\),.*^\1^p' ${TESTDIR}/registry/registry.log)
  done

  # push the registry image we just started... to itself, as a confidence check
  if ! ${BUILDAH_BINARY} --storage-driver vfs --root "${REGISTRY_DIR}"/root --runroot "${REGISTRY_DIR}"/run push --cert-dir "${REGISTRY_DIR}" --creds "${testuser}":"${testpassword}" "${REGISTRY_IMAGE}" localhost:"${REGISTRY_PORT}"/registry; then
    echo error pushing to /registry repository at localhost:$REGISTRY_PORT
    stop_registry
    false
  fi
}

function stop_registry() {
  if test -n "${REGISTRY_PID}" ; then
    kill "${REGISTRY_PID}"
    wait "${REGISTRY_PID}" || true
  fi
  unset REGISTRY_PID
  unset REGISTRY_PORT
  if test -n "${REGISTRY_DIR}" ; then
    ${BUILDAH_BINARY} --storage-driver vfs --root "${REGISTRY_DIR}"/root --runroot "${REGISTRY_DIR}"/run rmi -a -f
    rm -fr "${REGISTRY_DIR}"
  fi
  unset REGISTRY_DIR
}
