

# Library of common, shared utility functions.  This file is intended
# to be sourced by other scripts, not called directly.

# Global details persist here
source /etc/environment  # not always loaded under all circumstances

# Automation environment doesn't automatically load for Ubuntu 18
if [[ -r '/usr/share/automation/environment' ]]; then
    source '/usr/share/automation/environment'
fi

# Under some contexts these values are not set, make sure they are.
export USER="$(whoami)"
export HOME="$(getent passwd $USER | cut -d : -f 6)"
[[ -n "$UID" ]] || export UID=$(getent passwd $USER | cut -d : -f 3)
export GID=$(getent passwd $USER | cut -d : -f 4)
# Not cross-compiling by default
CROSS_TARGET="${CROSS_TARGET:-}"

# Essential default paths, many are overridden when executing under Cirrus-CI
# others are duplicated here, to assist in debugging.
export GOPATH="${GOPATH:-/var/tmp/go}"
if type -P go &> /dev/null
then
    # required for go 1.12+
    export GOCACHE="${GOCACHE:-$HOME/.cache/go-build}"
    eval "$(go env)"
    # required by make and other tools
    export $(go env | cut -d '=' -f 1)
    # Ensure compiled tooling is reachable
    export PATH="$PATH:$GOPATH/bin"
fi
export CIRRUS_WORKING_DIR="${CIRRUS_WORKING_DIR:-$GOPATH/src/github.com/containers/buildah}"
export GOSRC="${GOSRC:-$CIRRUS_WORKING_DIR}"
export PATH="$GOSRC/tests/tools/build:$HOME/bin:$GOPATH/bin:/usr/local/bin:$PATH"
SCRIPT_BASE=${SCRIPT_BASE:-./contrib/cirrus}

cd $GOSRC
if type -P git &> /dev/null
then
    CIRRUS_CHANGE_IN_REPO=${CIRRUS_CHANGE_IN_REPO:-$(git show-ref --hash=8 HEAD || date +%s)}
else # pick something unique and obviously not from Cirrus
    CIRRUS_CHANGE_IN_REPO=${CIRRUS_CHANGE_IN_REPO:-unknown$(date +%s)}
fi

export CI="${CI:-false}"
CIRRUS_CI="${CIRRUS_CI:-false}"
CONTINUOUS_INTEGRATION="${CONTINUOUS_INTEGRATION:-false}"
CIRRUS_REPO_NAME=${CIRRUS_REPO_NAME:-buildah}
CIRRUS_BASE_SHA=${CIRRUS_BASE_SHA:-unknown$(date +%d)}  # difficult to reliably discover
CIRRUS_BUILD_ID=${CIRRUS_BUILD_ID:-unknown$(date +%s)}  # must be short and unique enough
CIRRUS_TASK_ID=${CIRRUS_BUILD_ID:-unknown$(date +%d)}   # to prevent state thrashing when
                                                        # debugging with `hack/get_ci_vm.sh`

# Unsafe env. vars for display
SECRET_ENV_RE='(IRCID)|(ACCOUNT)|(^GC[EP]..+)|(SSH)'

# GCE image-name compatible string representation of distribution name
OS_RELEASE_ID="$(source /etc/os-release; echo $ID)"
# GCE image-name compatible string representation of distribution _major_ version
OS_RELEASE_VER="$(source /etc/os-release; echo $VERSION_ID | cut -d '.' -f 1)"
# Combined to ease soe usage
OS_REL_VER="${OS_RELEASE_ID}-${OS_RELEASE_VER}"

# FQINs needed for testing
REGISTRY_FQIN=${REGISTRY_FQIN:-docker.io/library/registry}
ALPINE_FQIN=${ALPINE_FQIN:-docker.io/library/alpine}

# for in-container testing
# TODO: Use $DEST_BRANCH as image tag after automated-builds working
IN_PODMAN_IMAGE="quay.io/libpod/in_podman:master"
IN_PODMAN_NAME="in_podman_$CIRRUS_TASK_ID"
IN_PODMAN="${IN_PODMAN:-false}"

# Working with apt under Debian/Ubuntu automation is a PITA, make it easy
# Avoid some ways of getting stuck waiting for user input
export DEBIAN_FRONTEND=noninteractive
# Short-cut for frequently used base command
export APTGET='apt-get -qq --yes'
# Short timeout for quick-running packaging command
SHORT_APTGET="timeout_attempt_delay_command 24s 5 30s $APTGET"
SHORT_DNFY="timeout_attempt_delay_command 60s 2 5s dnf -y"
# Longer timeout for long-running packaging command
LONG_APTGET="timeout_attempt_delay_command 300s 5 30s $APTGET"
LONG_DNFY="timeout_attempt_delay_command 300s 3 60s dnf -y"

# Allow easy substitution for debugging if needed
CONTAINER_RUNTIME="showrun ${CONTAINER_RUNTIME:-podman}"

# Pass in a list of one or more envariable names; exit non-zero with
# helpful error message if any value is empty
req_env_var() {
    # Provide context. If invoked from function use its name; else script name
    local caller=${FUNCNAME[1]}
    if [[ -n "$caller" ]]; then
        # Indicate that it's a function name
        caller="$caller()"
    else
        # Not called from a function: use script name
        caller=$(basename $0)
    fi

    # Usage check
    [[ -n "$1" ]] || die 1 "FATAL: req_env_var: invoked without arguments"

    # Each input arg is an envariable name, e.g. HOME PATH etc. Expand each.
    # If any is empty, bail out and explain why.
    for i; do
        if [[ -z "${!i}" ]]; then
            die 9 "FATAL: $caller requires \$$i to be non-empty"
        fi
    done
}

show_env_vars() {
    echo "Showing selection of environment variable definitions:"
    _ENV_VAR_NAMES=$(awk 'BEGIN{for(v in ENVIRON) print v}' | \
        egrep -v "(^PATH$)|(^BASH_FUNC)|(^[[:punct:][:space:]]+)|$SECRET_ENV_RE" | \
        sort -u)
    for _env_var_name in $_ENV_VAR_NAMES
    do
        # Supports older BASH versions
        printf "    ${_env_var_name}=%q\n" "$(printenv $_env_var_name)"
    done
}

die() {
    echo "************************************************"
    echo ">>>>> ${2:-FATAL ERROR (but no message given!) in ${FUNCNAME[1]}()}"
    echo "************************************************"
    exit ${1:-1}
}

warn() {
    echo ">>>>> WARNING: ${1:-WARNING (but no message given!) in ${FUNCNAME[1]}()}" > /dev/stderr
}

bad_os_id_ver() {
    echo "Unknown/Unsupported distro. $OS_RELEASE_ID and/or version $OS_RELEASE_VER for $(basename $0)"
    exit 42
}

timeout_attempt_delay_command() {
    TIMEOUT=$1
    ATTEMPTS=$2
    DELAY=$3
    shift 3
    STDOUTERR=$(mktemp -p '' $(basename $0)_XXXXX)
    req_env_var ATTEMPTS DELAY
    echo "Retrying $ATTEMPTS times with a $DELAY delay, and $TIMEOUT timeout for command: $@"
    for (( COUNT=1 ; COUNT <= $ATTEMPTS ; COUNT++ ))
    do
        echo "##### (attempt #$COUNT)" &>> "$STDOUTERR"
        if timeout --foreground $TIMEOUT "$@" &>> "$STDOUTERR"
        then
            echo "##### (success after #$COUNT attempts)" &>> "$STDOUTERR"
            break
        else
            echo "##### (failed with exit: $?)" &>> "$STDOUTERR"
            sleep $DELAY
        fi
    done
    cat "$STDOUTERR"
    rm -f "$STDOUTERR"
    if (( COUNT > $ATTEMPTS ))
    then
        echo "##### (exceeded $ATTEMPTS attempts)"
        exit 125
    fi
}

showrun() {
    local -a context
    context=($(caller 0))
    echo "+ $@  # ${context[2]}:${context[0]} in ${context[1]}()" > /dev/stderr
    "$@"
}

# workaround issue 1945 (remove when resolved)
remove_storage_mountopt() {
    local FILEPATH=/etc/containers/storage.conf
    warn "remove_storage_mountopt() is overwriting $FILEPATH"
    # This file normally comes from containers-common package
    cat <<EOF> $FILEPATH
[storage]
runroot = "/var/run/containers/storage"
graphroot = "/var/lib/containers/storage"
[storage.options]
mountopt = ""
EOF
    cat $FILEPATH
}

# Remove all files provided by the distro version of buildah.
# All VM cache-images used for testing include the distro buildah because it
# simplifies installing necessary dependencies which can change over time.
# For general CI testing however, calling this function makes sure the system
# can only run the compiled source version.
remove_packaged_buildah_files() {
    warn "Removing packaged buildah files to prevent conflicts with source build and testing."
    req_env_var OS_RELEASE_ID

    if [[ "$OS_RELEASE_ID" =~ "ubuntu" ]]
    then
        LISTING_CMD="dpkg-query -L buildah"
    else
        LISTING_CMD='rpm -ql buildah'
    fi

    # yum/dnf/dpkg may list system directories, only remove files
    $LISTING_CMD | while read fullpath
    do
        # Sub-directories may contain unrelated/valuable stuff
        if [[ -d "$fullpath" ]]; then continue; fi
        rm -vf "$fullpath"
    done

    if [[ -z "$CONTAINER" ]]; then
        # Be super extra sure and careful vs performant and completely safe
        sync && echo 3 > /proc/sys/vm/drop_caches
    fi
}

in_podman() {
    req_env_var IN_PODMAN_NAME GOSRC GOPATH SECRET_ENV_RE HOME
    [[ -n "$@" ]] || \
        die 7 "Must specify FQIN and command with arguments to execute"
    local envargs
    local envname
    local envvalue
    local envrx='(^CIRRUS_.+)|(^BUILDAH_+)|(^STORAGE_)|(^CI$)|(^CROSS_TARGET$)|(^IN_PODMAN_.+)'
    for envname in $(awk 'BEGIN{for(v in ENVIRON) print v}' | \
                     egrep "$envrx" | \
                     egrep -v "CIRRUS_.+_MESSAGE" | \
                     egrep -v "CIRRUS_.+_TITLE" | \
                     egrep -v "$SECRET_ENV_RE")
    do
        envvalue="${!envname}"
        [[ -z "$envname" ]] || [[ -z "$envvalue" ]] || \
            envargs="${envargs:+$envargs }-e $envname=$envvalue"
    done
    # Back in the days of testing under PAPR, containers were run with super-privledges.
    # That behavior is preserved here with a few updates for modern podman behaviors.
    # The only other additions/changes are passthrough of CI-related env. vars ($envargs),
    # some path related updates, and mounting cgroups RW instead of the RO default.
    showrun podman run -i --name $IN_PODMAN_NAME \
                   $envargs \
                   --net=host \
                   --net="container:registry" \
                   --security-opt label=disable \
                   --security-opt seccomp=unconfined \
                   --cap-add=all \
                   -e "GOPATH=$GOPATH" \
                   -e "GOSRC=$GOSRC" \
                   -e "IN_PODMAN=false" \
                   -e "CONTAINER=podman" \
                   -e "CGROUP_MANAGER=cgroupfs" \
                   -v "$HOME/auth:$HOME/auth:ro" \
                   -v /sys/fs/cgroup:/sys/fs/cgroup:rw \
                   -v /dev/fuse:/dev/fuse:rw \
                   -v "$GOSRC:$GOSRC:z" \
                   --workdir "$GOSRC" \
                   "$@"
}

verify_local_registry(){
    # On the unexpected/rare chance of a name-clash
    local CUSTOM_FQIN=localhost:5000/my-alpine-$RANDOM
    echo "Verifying local 'registry' container is operational"
    showrun podman version
    showrun podman info
    showrun podman ps --all
    showrun podman images
    showrun ls -alF $HOME/auth
    showrun podman pull $ALPINE_FQIN
    showrun podman login --tls-verify=false localhost:5000 --username testuser --password testpassword
    showrun podman tag $ALPINE_FQIN $CUSTOM_FQIN
    showrun podman push --tls-verify=false --creds=testuser:testpassword $CUSTOM_FQIN
    showrun podman ps --all
    showrun podman images
    showrun podman rmi $ALPINE_FQIN
    showrun podman rmi $CUSTOM_FQIN
    showrun podman pull --tls-verify=false --creds=testuser:testpassword $CUSTOM_FQIN
    showrun podman ps --all
    showrun podman images
    echo "Success, local registry is working, cleaning up."
    showrun podman rmi $CUSTOM_FQIN
}

execute_local_registry() {
    if nc -4 -z 127.0.0.1 5000
    then
        warn "Found listener on localhost:5000, NOT starting up local registry server."
        verify_local_registry
        return 0
    fi
    req_env_var CONTAINER_RUNTIME GOSRC
    local authdirpath=$HOME/auth
    cd $GOSRC

    echo "Creating a self signed certificate and get it in the right places"
    mkdir -p $authdirpath
    openssl req \
        -newkey rsa:4096 -nodes -sha256 -x509 -days 2 \
        -subj "/C=US/ST=Foo/L=Bar/O=Red Hat, Inc./CN=registry host certificate" \
        -addext subjectAltName=DNS:localhost \
        -keyout $authdirpath/domain.key \
        -out $authdirpath/domain.crt

    cp $authdirpath/domain.crt $authdirpath/domain.cert

    echo "Creating http credentials file"
    showrun htpasswd -Bbn testuser testpassword > $authdirpath/htpasswd

    echo "Starting up the local 'registry' container"
    showrun podman run -d -p 5000:5000 --name registry \
        -v $authdirpath:$authdirpath:Z \
        -e "REGISTRY_AUTH=htpasswd" \
        -e "REGISTRY_AUTH_HTPASSWD_REALM=Registry Realm" \
        -e REGISTRY_AUTH_HTPASSWD_PATH=$authdirpath/htpasswd \
        -e REGISTRY_HTTP_TLS_CERTIFICATE=$authdirpath/domain.crt \
        -e REGISTRY_HTTP_TLS_KEY=$authdirpath/domain.key \
        $REGISTRY_FQIN

    verify_local_registry
}
