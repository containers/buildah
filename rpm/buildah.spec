%global with_debug 1

%if 0%{?with_debug}
%global _find_debuginfo_dwz_opts %{nil}
%global _dwz_low_mem_die_limit 0
%else
%global debug_package   %{nil}
%endif

# RHEL's default %%gobuild macro doesn't account for the BUILDTAGS variable, so we
# set it separately here and do not depend on RHEL's go-[s]rpm-macros package
# until that's fixed.
# c9s bz: https://bugzilla.redhat.com/show_bug.cgi?id=2227328
# c8s bz: https://bugzilla.redhat.com/show_bug.cgi?id=2227331
%if %{defined rhel}
%define gobuild(o:) go build -buildmode pie -compiler gc -tags="rpm_crashtraceback libtrust_openssl ${BUILDTAGS:-}" -ldflags "-linkmode=external -compressdwarf=false ${LDFLAGS:-} -B 0x$(head -c20 /dev/urandom|od -An -tx1|tr -d ' \\n') -extldflags '%__global_ldflags'" -a -v -x %{?**};
%endif

%global gomodulesmode GO111MODULE=on

%if 0%{defined fedora}
%define build_with_btrfs 1
%endif

%global git0 https://github.com/containers/%{name}

Name: buildah
# Set different Epoch for copr
%if %{defined copr_username}
Epoch: 102
%else
Epoch: 2
%endif
# DO NOT TOUCH the Version string!
# The TRUE source of this specfile is:
# https://github.com/containers/skopeo/blob/main/rpm/skopeo.spec
# If that's what you're reading, Version must be 0, and will be updated by Packit for
# copr and koji builds.
# If you're reading this on dist-git, the version is automatically filled in by Packit.
Version: 0
# The `AND` needs to be uppercase in the License for SPDX compatibility
License: Apache-2.0 AND BSD-2-Clause AND BSD-3-Clause AND ISC AND MIT AND MPL-2.0
Release: %autorelease
%if %{defined golang_arches_future}
ExclusiveArch: %{golang_arches_future}
%else
ExclusiveArch: aarch64 ppc64le s390x x86_64
%endif
Summary: A command line tool used for creating OCI Images
URL: https://%{name}.io
# Tarball fetched from upstream
Source: %{git0}/archive/v%{version}.tar.gz
BuildRequires: device-mapper-devel
BuildRequires: git-core
BuildRequires: golang >= 1.16.6
BuildRequires: glib2-devel
BuildRequires: glibc-static
%if !%{defined gobuild}
BuildRequires: go-rpm-macros
%endif
BuildRequires: gpgme-devel
BuildRequires: libassuan-devel
BuildRequires: make
BuildRequires: ostree-devel
%if %{defined build_with_btrfs}
BuildRequires: btrfs-progs-devel
%endif
BuildRequires: shadow-utils-subid-devel
Requires: containers-common-extra
%if %{defined fedora}
BuildRequires: libseccomp-static
%else
BuildRequires: libseccomp-devel
%endif
Requires: libseccomp >= 2.4.1-0
Suggests: cpp

%description
The %{name} package provides a command line tool which can be used to
* create a working container from scratch
or
* create a working container from an image as a starting point
* mount/umount a working container's root file system for manipulation
* save container's root file system layer to create a new image
* delete a working container or an image

%package tests
Summary: Tests for %{name}

Requires: %{name} = %{epoch}:%{version}-%{release}
%if %{defined fedora}
Requires: bats
%endif
Requires: bzip2
Requires: podman
Requires: golang
Requires: jq
Requires: httpd-tools
Requires: openssl
Requires: nmap-ncat
Requires: git-daemon

%description tests
%{summary}

This package contains system tests for %{name}

%prep
%autosetup -Sgit -n %{name}-%{version}

%build
%set_build_flags
export CGO_CFLAGS=$CFLAGS

# These extra flags present in $CFLAGS have been skipped for now as they break the build
CGO_CFLAGS=$(echo $CGO_CFLAGS | sed 's/-flto=auto//g')
CGO_CFLAGS=$(echo $CGO_CFLAGS | sed 's/-Wp,D_GLIBCXX_ASSERTIONS//g')
CGO_CFLAGS=$(echo $CGO_CFLAGS | sed 's/-specs=\/usr\/lib\/rpm\/redhat\/redhat-annobin-cc1//g')

%ifarch x86_64
export CGO_CFLAGS+=" -m64 -mtune=generic -fcf-protection=full"
%endif

export CNI_VERSION=`grep '^# github.com/containernetworking/cni ' src/modules.txt | sed 's,.* ,,'`
export LDFLAGS="-X main.buildInfo=`date +%s` -X main.cniVersion=${CNI_VERSION}"

export BUILDTAGS="seccomp exclude_graphdriver_devicemapper $(hack/systemd_tag.sh) $(hack/libsubid_tag.sh)"
%if !%{defined build_with_btrfs}
export BUILDTAGS+=" btrfs_noversion exclude_graphdriver_btrfs"
%endif

%gobuild -o bin/%{name} ./cmd/%{name}
%gobuild -o bin/imgtype ./tests/imgtype
%gobuild -o bin/copy ./tests/copy
%gobuild -o bin/tutorial ./tests/tutorial
%gobuild -o bin/inet ./tests/inet
%{__make} docs

%install
make DESTDIR=%{buildroot} PREFIX=%{_prefix} install install.completions

install -d -p %{buildroot}/%{_datadir}/%{name}/test/system
cp -pav tests/. %{buildroot}/%{_datadir}/%{name}/test/system
cp bin/imgtype %{buildroot}/%{_bindir}/%{name}-imgtype
cp bin/copy    %{buildroot}/%{_bindir}/%{name}-copy
cp bin/tutorial %{buildroot}/%{_bindir}/%{name}-tutorial
cp bin/inet     %{buildroot}/%{_bindir}/%{name}-inet

rm %{buildroot}%{_datadir}/%{name}/test/system/tools/build/*

#define license tag if not already defined
%{!?_licensedir:%global license %doc}

%files
%license LICENSE vendor/modules.txt
%doc README.md
%{_bindir}/%{name}
%{_mandir}/man1/%{name}*
%dir %{_datadir}/bash-completion
%dir %{_datadir}/bash-completion/completions
%{_datadir}/bash-completion/completions/%{name}

%files tests
%license LICENSE
%{_bindir}/%{name}-imgtype
%{_bindir}/%{name}-copy
%{_bindir}/%{name}-tutorial
%{_bindir}/%{name}-inet
%{_datadir}/%{name}/test

%changelog
%if %{defined autochangelog}
%autochangelog
%else
# NOTE: This changelog will be visible on CentOS 8 Stream builds
# Other envs are capable of handling autochangelog
* Fri Jun 16 2023 RH Container Bot <rhcontainerbot@fedoraproject.org>
- Placeholder changelog for envs that are not autochangelog-ready.
- Contact upstream if you need to report an issue with the build.
%endif
