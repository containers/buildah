#!/bin/bash

set -e

source $(dirname $0)/lib.sh

req_env_vars CI GOSRC OS_RELEASE_ID

case $1 in
    audit)
        case $OS_RELEASE_ID in
            ubuntu) showrun cat /var/log/kern.log ;;
            fedora) showrun cat /var/log/audit/audit.log ;;
            *) bad_os_id_ver ;;
        esac
        ;;
    df) showrun df -lhTx tmpfs ;;
    journal) showrun journalctl -b ;;
    podman) showrun podman system info ;;
    buildah_version) showrun $GOSRC/bin/buildah version;;
    buildah_info) showrun $GOSRC/bin/buildah info;;
    golang) showrun go version;;
    packages)
        # These names are common to Fedora and Ubuntu
        PKG_NAMES=(\
                    buildah
                    conmon
                    container-selinux
                    containernetworking-plugins
                    containers-common
                    crun
                    cri-o-runc
                    libseccomp
                    libseccomp2
                    podman
                    runc
                    skopeo
                    slirp4netns
        )
        case $OS_RELEASE_ID in
            fedora*)
                PKG_LST_CMD='rpm -q --qf=%{N}-%{V}-%{R}-%{ARCH}\n'
                ;;
            ubuntu*)
                PKG_LST_CMD='dpkg-query --show --showformat=${Package}-${Version}-${Architecture}\n'
                ;;
            *) bad_os_id_ver ;;
        esac
        # Any not-present packages will be listed as such
        $PKG_LST_CMD ${PKG_NAMES[@]} | sort -u
        ;;
    *) die "Warning, $(basename $0) doesn't know how to handle the parameter '$1'"
esac
