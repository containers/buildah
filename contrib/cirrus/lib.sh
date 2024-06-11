

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

# All CI jobs use a local registry
export CI_USE_REGISTRY_CACHE=true

# Regex defining all CI-related env. vars. necessary for all possible
# testing operations on all platforms and versions.  This is necessary
# to avoid needlessly passing through global/system values across
# contexts, such as host->container or root->rootless user
#
# List of envariables which must be EXACT matches
#   N/B: Don't include BUILDAH_ISOLATION, STORAGE_DRIVER, or CGROUP_MANAGER
#   here because they will negatively affect execution of the rootless
#   integration tests.
PASSTHROUGH_ENV_EXACT='DEST_BRANCH|DISTRO_NV|GOPATH|GOSRC|ROOTLESS_USER|SCRIPT_BASE|IN_PODMAN_IMAGE'

# List of envariable patterns which must match AT THE BEGINNING of the name.
PASSTHROUGH_ENV_ATSTART='CI|TEST'

# List of envariable patterns which can match ANYWHERE in the name
PASSTHROUGH_ENV_ANYWHERE='_NAME|_FQIN'

# Combine into one
PASSTHROUGH_ENV_RE="(^($PASSTHROUGH_ENV_EXACT)\$)|(^($PASSTHROUGH_ENV_ATSTART))|($PASSTHROUGH_ENV_ANYWHERE)"

# Unsafe env. vars for display
SECRET_ENV_RE='ACCOUNT|GC[EP]..|SSH|PASSWORD|SECRET|TOKEN'

# FQINs needed for testing
REGISTRY_FQIN=${REGISTRY_FQIN:-quay.io/libpod/registry:2.8.2}
ALPINE_FQIN=${ALPINE_FQIN:-quay.io/libpod/alpine}

# for in-container testing
IN_PODMAN_NAME="in_podman_$CIRRUS_TASK_ID"
IN_PODMAN="${IN_PODMAN:-false}"

# rootless_user
ROOTLESS_USER="rootlessuser"

# Downloaded, but not installed packages.
PACKAGE_DOWNLOAD_DIR=/var/cache/download

lilto() { err_retry 8 1000 "" "$@"; }  # just over 4 minutes max
bigto() { err_retry 7 5670 "" "$@"; }  # 12 minutes max

# Working with apt under automation is a PITA, make it easy
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

    if [[ "$OS_RELEASE_ID" =~ "debian" ]]
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

# Return a list of environment variables that should be passed through
# to lower levels (tests in containers, or via ssh to rootless).
# We return the variable names only, not their values. It is up to our
# caller to reference values.
passthrough_envars(){
    warn "Will pass env. vars. matching the following regex:
    $PASSTHROUGH_ENV_RE"
    compgen -A variable | \
        grep -Ev "$SECRET_ENV_RE" | \
        grep -Ev "^PASSTHROUGH_" | \
        grep -E  "$PASSTHROUGH_ENV_RE"
}

in_podman() {
    req_env_vars IN_PODMAN_NAME GOSRC GOPATH SECRET_ENV_RE HOME
    [[ -n "$@" ]] || \
        die "Must specify FQIN and command with arguments to execute"

    # Line-separated arguments which include shell-escaped special characters
    declare -a envargs
    while read -r var; do
        # Pass "-e VAR" on the command line, not "-e VAR=value". Podman can
        # do a much better job of transmitting the value than we can,
        # especially when value includes spaces.
        envargs+=("-e" "$var")
    done <<<"$(passthrough_envars)"

    showrun podman run -i --name="$IN_PODMAN_NAME" \
                   --net=host \
                   --privileged \
                   --cgroupns=host \
                   "${envargs[@]}" \
                   -e BUILDAH_ISOLATION \
                   -e STORAGE_DRIVER \
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

setup_rootless() {
    req_env_vars GOPATH GOSRC SECRET_ENV_RE

    local rootless_uid
    local rootless_gid
    local env_var_val
    local akfilepath
    local sshcmd

    # Only do this once; established by setup_environment.sh
    # shellcheck disable=SC2154
    if passwd --status $ROOTLESS_USER
    then
        if [[ $PRIV_NAME = "rootless" ]]; then
            msg "Updating $ROOTLESS_USER user permissions on possibly changed libpod code"
            chown -R $ROOTLESS_USER:$ROOTLESS_USER "$GOPATH" "$GOSRC"
            return 0
        fi
    fi
    msg "************************************************************"
    msg "Setting up rootless user '$ROOTLESS_USER'"
    msg "************************************************************"
    cd $GOSRC || exit 1
    # Guarantee independence from specific values
    rootless_uid=$[RANDOM+1000]
    rootless_gid=$[RANDOM+1000]
    msg "creating $rootless_uid:$rootless_gid $ROOTLESS_USER user"
    groupadd -g $rootless_gid $ROOTLESS_USER
    useradd -g $rootless_gid -u $rootless_uid --no-user-group --create-home $ROOTLESS_USER

    # We also set up rootless user for image-scp tests (running as root)
    if [[ $PRIV_NAME = "rootless" ]]; then
        chown -R $ROOTLESS_USER:$ROOTLESS_USER "$GOPATH" "$GOSRC"
    fi
    echo "$ROOTLESS_USER ALL=(root) NOPASSWD: ALL" > /etc/sudoers.d/ci-rootless

    mkdir -p "$HOME/.ssh" "/home/$ROOTLESS_USER/.ssh"

    msg "Creating ssh key pairs"
    [[ -r "$HOME/.ssh/id_rsa" ]] || \
        ssh-keygen -t rsa -P "" -f "$HOME/.ssh/id_rsa"
    ssh-keygen -t ed25519 -P "" -f "/home/$ROOTLESS_USER/.ssh/id_ed25519"
    ssh-keygen -t rsa -P "" -f "/home/$ROOTLESS_USER/.ssh/id_rsa"

    msg "Setup authorized_keys"
    cat $HOME/.ssh/*.pub /home/$ROOTLESS_USER/.ssh/*.pub >> $HOME/.ssh/authorized_keys
    cat $HOME/.ssh/*.pub /home/$ROOTLESS_USER/.ssh/*.pub >> /home/$ROOTLESS_USER/.ssh/authorized_keys

    msg "Ensure the ssh daemon is up and running within 5 minutes"
    systemctl start sshd
    lilto systemctl is-active sshd

    msg "Configure ssh file permissions"
    chmod -R 700 "$HOME/.ssh"
    chmod -R 700 "/home/$ROOTLESS_USER/.ssh"
    chown -R $ROOTLESS_USER:$ROOTLESS_USER "/home/$ROOTLESS_USER/.ssh"

    msg "   setup known_hosts for $USER"
    ssh-keyscan localhost > /root/.ssh/known_hosts

    msg "   setup known_hosts for $ROOTLESS_USER"
    install -Z -m 700 -o $ROOTLESS_USER -g $ROOTLESS_USER \
        /root/.ssh/known_hosts /home/$ROOTLESS_USER/.ssh/known_hosts

    msg "Setting up pass-through env. vars for $ROOTLESS_USER"
    while read -r env_var; do
        # N/B: Some values contain spaces and other potential nasty-bits
        # (i.e. $CIRRUS_COMMIT_MESSAGE).  The %q conversion ensures proper
        # bash-style escaping.
        printf -- "export %s=%q\n" "${env_var}" "${!env_var}" | tee -a /home/$ROOTLESS_USER/ci_environment
    done <<<"$(passthrough_envars)"
}
