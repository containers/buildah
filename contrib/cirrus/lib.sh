

# Library of common, shared utility functions.  This file is intended
# to be sourced by other scripts, not called directly.

# BEGIN Global export of all variables
set -a

# Due to differences across platforms and runtime execution environments,
# handling of the (otherwise) default shell setup is non-uniform.  Rather
# than attempt to workaround differences, simply force-load/set required
# items every time this library is utilized.
USER="$(whoami)"
HOME="$(getent passwd $USER | cut -d : -f 6)"
# Some platforms set and make this read-only
[[ -n "$UID" ]] || \
    UID=$(getent passwd $USER | cut -d : -f 3)

# Automation library installed at image-build time,
# defining $AUTOMATION_LIB_PATH in this file.
if [[ -r "/etc/automation_environment" ]]; then
    source /etc/automation_environment
fi
# shellcheck disable=SC2154
if [[ -n "$AUTOMATION_LIB_PATH" ]]; then
        # shellcheck source=/usr/share/automation/lib/common_lib.sh
        source $AUTOMATION_LIB_PATH/common_lib.sh
else
    (
    echo "WARNING: It does not appear that containers/automation was installed."
    echo "         Functionality of most of this library will be negatively impacted"
    echo "         This ${BASH_SOURCE[0]} was loaded by ${BASH_SOURCE[1]}"
    ) > /dev/stderr
fi

# Required for proper GPG functioning under automation
GPG_TTY="${GPG_TTY:-/dev/null}"

# Essential default paths, many are overridden when executing under Cirrus-CI
# others are duplicated here, to assist in debugging.
GOPATH="${GOPATH:-/var/tmp/go}"
if type -P go &> /dev/null
then
    # required for go 1.12+
    GOCACHE="${GOCACHE:-$HOME/.cache/go-build}"
    eval "$(go env)"
    # Ensure compiled tooling is reachable
    PATH="$PATH:$GOPATH/bin"
fi
CIRRUS_WORKING_DIR="${CIRRUS_WORKING_DIR:-$GOPATH/src/github.com/containers/buildah}"
GOSRC="${GOSRC:-$CIRRUS_WORKING_DIR}"
PATH="$GOSRC/tests/tools/build:$HOME/bin:$GOPATH/bin:/usr/local/bin:/usr/lib/cri-o-runc/sbin:$PATH"
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
OS_RELEASE_VER="$(source /etc/os-release; echo $VERSION_ID | tr -d '.')"
# Combined to ease soe usage
OS_REL_VER="${OS_RELEASE_ID}-${OS_RELEASE_VER}"

# FQINs needed for testing
REGISTRY_FQIN=${REGISTRY_FQIN:-docker.io/library/registry}
ALPINE_FQIN=${ALPINE_FQIN:-docker.io/library/alpine}

# for in-container testing
IN_PODMAN_NAME="in_podman_$CIRRUS_TASK_ID"
IN_PODMAN="${IN_PODMAN:-false}"

# Downloaded, but not installed packages.
PACKAGE_DOWNLOAD_DIR=/var/cache/download

lilto() { err_retry 8 1000 "" "$@"; }  # just over 4 minutes max
bigto() { err_retry 7 5670 "" "$@"; }  # 12 minutes max

# Working with apt under Debian/Ubuntu automation is a PITA, make it easy
# Avoid some ways of getting stuck waiting for user input
export DEBIAN_FRONTEND=noninteractive
# Short-cut for frequently used base command
export APTGET='apt-get -qq --yes'
# Short timeout for quick-running packaging command
SHORT_APTGET="lilto $APTGET"
SHORT_DNFY="lilto dnf -y"
# Longer timeout for long-running packaging command
LONG_APTGET="bigto $APTGET"
LONG_DNFY="bigto dnf -y"

# Allow easy substitution for debugging if needed
CONTAINER_RUNTIME="showrun ${CONTAINER_RUNTIME:-podman}"

# END Global export of all variables
set +a

bad_os_id_ver() {
    die "Unknown/Unsupported distro. $OS_RELEASE_ID and/or version $OS_RELEASE_VER for $(basename $0)"
}

# Remove all files provided by the distro version of buildah.
# All VM cache-images used for testing include the distro buildah because it
# simplifies installing necessary dependencies which can change over time.
# For general CI testing however, calling this function makes sure the system
# can only run the compiled source version.
remove_packaged_buildah_files() {
    warn "Removing packaged buildah files to prevent conflicts with source build and testing."
    req_env_vars OS_RELEASE_ID

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
        # As of Ubuntu 2010, policy.json in buildah, not containers-common package
        if [[ "$OS_RELEASE_ID" == "ubuntu" ]] && \
            grep -q '/etc/containers'<<<"$fullpath"; then

            warn "Not removing $fullpath (from buildah package)"
            continue
        fi

        rm -vf "$fullpath"
    done

    if [[ -z "$CONTAINER" ]]; then
        # Be super extra sure and careful vs performant and completely safe
        sync && echo 3 > /proc/sys/vm/drop_caches
    fi
}

in_podman() {
    req_env_vars IN_PODMAN_NAME GOSRC GOPATH SECRET_ENV_RE HOME
    [[ -n "$@" ]] || \
        die "Must specify FQIN and command with arguments to execute"
    local envargs
    local envarg
    local envname
    local envval
    local xchars='[:punct:][:cntrl:][:space:]'
    local envrx='(^CI.*)|(^CIRRUS)|(^GOPATH)|(^GOCACHE)|(^GOSRC)|(^SCRIPT_BASE)|(.*_NAME)|(.*_FQIN)|(^IN_PODMAN_)|(^DISTRO)|(^BUILDAH)|(^STORAGE_)'
    local envfile=$(mktemp -p '' in_podman_env_tmp_XXXXXXXX)
    trap "rm -f $envfile" EXIT

    msg "Gathering env. vars. to pass-through into container."
    for envname in $(awk 'BEGIN{for(v in ENVIRON) print v}' | sort | \
                     egrep "$envrx" | egrep -v "$SECRET_ENV_RE" | \
                     egrep -v "^CIRRUS_.+(MESSAGE|TITLE)")
    do
        envval="${!envname}"
        [[ -n $(tr -d "$xchars" <<<"$envval") ]] || continue
        envarg=$(printf -- "$envname=%q" "$envval")
        echo "$envarg" | tee -a $envfile | indent
    done
    showrun podman run -i --name="$IN_PODMAN_NAME" \
                   --net=host \
                   --net="container:registry" \
                   --security-opt=label=disable \
                   --security-opt=seccomp=unconfined \
                   --cap-add=all \
                   --env-file=$envfile \
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
    req_env_vars CONTAINER_RUNTIME GOSRC
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
