%global _find_debuginfo_dwz_opts %{nil}
%global _dwz_low_mem_die_limit 0

%if %{defined copr_username}
%define copr_build 1
%endif

%if 0%{?rhel} > 7 && ! 0%{?fedora}
%define gobuild(o:) \
go build -buildmode pie -compiler gc -tags="rpm_crashtraceback libtrust_openssl ${BUILDTAGS:-}" -ldflags "${LDFLAGS:-} -compressdwarf=false -B 0x$(head -c20 /dev/urandom|od -An -tx1|tr -d ' \\n') -extldflags '%__global_ldflags'" -a -v %{?**};
%else
%if ! 0%{?gobuild:1}
%define gobuild(o:) GO111MODULE=off go build -buildmode pie -compiler gc -tags="rpm_crashtraceback ${BUILDTAGS:-}" -ldflags "${LDFLAGS:-} -B 0x$(head -c20 /dev/urandom|od -An -tx1|tr -d ' \\n') -extldflags '-Wl,-z,relro -Wl,-z,now -specs=/usr/lib/rpm/redhat/redhat-hardened-ld '" -a -v %{?**};
%endif
%endif

%global import_path github.com/containers/buildah
%global branch release-1.24
%global commit0 60e6bc0f7338b4b7a0ec044f75f8dd3ba0fa58fb
%global shortcommit0 %(c=%{commit0}; echo ${c:0:7})

Epoch: 1
Name: buildah
Version: 1.24.6
Release: 8%{?dist}
Summary: A command line tool used for creating OCI Images
License: ASL 2.0
URL: https://%{name}.io
# https://fedoraproject.org/wiki/PackagingDrafts/Go#Go_Language_Architectures
ExclusiveArch: %{go_arches}
%if 0%{?branch:1}
Source0: https://%{import_path}/tarball/%{commit0}/%{branch}-%{shortcommit0}.tar.gz
%else
Source0: https://%{import_path}/archive/%{commit0}/%{name}-%{version}-%{shortcommit0}.tar.gz
%endif
BuildRequires: golang >= 1.17.7
BuildRequires: git
BuildRequires: glib2-devel
BuildRequires: libseccomp-devel
BuildRequires: ostree-devel
BuildRequires: glibc-static
BuildRequires: go-md2man
BuildRequires: gpgme-devel
BuildRequires: device-mapper-devel
BuildRequires: libassuan-devel
BuildRequires: make
Requires: runc >= 1.0.0-26
Requires: containers-common >= 2:1-2
Recommends: container-selinux
Requires: slirp4netns >= 0.3-0

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
Requires: bzip2
Requires: podman
Requires: golang
Requires: jq
Requires: httpd-tools
Requires: openssl
Requires: nmap-ncat

%description tests
%{summary}

This package contains system tests for %{name}

%prep
%if 0%{?branch:1}
%autosetup -Sgit -n containers-%{name}-%{shortcommit0}
%else
%autosetup -Sgit -n %{name}-%{commit0}
%endif
sed -i 's/GOMD2MAN =/GOMD2MAN ?=/' docs/Makefile
sed -i '/docs install/d' Makefile

%build
mkdir _build
pushd _build
mkdir -p src/github.com/containers
ln -s $(dirs +1 -l) src/%{import_path}
popd

mv vendor src

export GOPATH=$(pwd)/_build:$(pwd)
export BUILDTAGS='seccomp selinux btrfs_noversion exclude_graphdriver_btrfs'
export GO111MODULE=off
export CGO_CFLAGS="%{optflags} -D_GNU_SOURCE -D_LARGEFILE_SOURCE -D_LARGEFILE64_SOURCE -D_FILE_OFFSET_BITS=64"
export CNI_VERSION=`grep '^# github.com/containernetworking/cni ' src/modules.txt | sed 's,.* ,,'`
export LDFLAGS="$LDFLAGS -X main.buildInfo=`date +%s` -X main.cniVersion=${CNI_VERSION}"
rm -f src/github.com/containers/storage/drivers/register/register_btrfs.go
%gobuild -o bin/%{name} %{import_path}/cmd/%{name}
%gobuild -o imgtype %{import_path}/tests/imgtype
%gobuild -o bin/copy %{import_path}/tests/copy
GOMD2MAN=go-md2man %{__make} -C docs

%install
export GOPATH=$(pwd)/_build:$(pwd):%{gopath}
make DESTDIR=%{buildroot} PREFIX=%{_prefix} install install.completions
install -d -p %{buildroot}/%{_datadir}/%{name}/test/system
cp -pav tests/. %{buildroot}/%{_datadir}/%{name}/test/system
cp imgtype %{buildroot}/%{_bindir}/%{name}-imgtype
cp bin/copy %{buildroot}/%{_bindir}/%{name}-copy
make DESTDIR=%{buildroot} PREFIX=%{_prefix} -C docs install
rm -f %{buildroot}%{_mandir}/man5/{Containerfile*,containerignore*}

#define license tag if not already defined
%{!?_licensedir:%global license %doc}

%files
%license LICENSE
%doc README.md
%{_bindir}/%{name}
%{_mandir}/man[15]/*
%dir %{_datadir}/bash-completion
%dir %{_datadir}/bash-completion/completions
%{_datadir}/bash-completion/completions/%{name}

%files tests
%license LICENSE
%{_bindir}/%{name}-imgtype
%{_bindir}/%{name}-copy
%{_datadir}/%{name}/test

%changelog
* Wed Jan 03 2024 Jindrich Novy <jnovy@redhat.com> - 1:1.24.6-8
- rebuild
- Resolves: RHEL-18261

* Mon Aug 28 2023 Jindrich Novy <jnovy@redhat.com> - 1:1.24.6-7
- rebuild for CVE-2023-29406
- Related: #2176055

* Thu Jun 15 2023 Jindrich Novy <jnovy@redhat.com> - 1:1.24.6-6
- rebuild for following CVEs:
CVE-2022-41724 CVE-2022-41725 CVE-2023-24538 CVE-2023-24534 CVE-2023-24536 CVE-2022-41723 CVE-2023-24539 CVE-2023-24540 CVE-2023-29400
- Resolves: #2179943
- Resolves: #2187341
- Resolves: #2187359
- Resolves: #2203677
- Resolves: #2207505

* Tue Mar 14 2023 Jindrich Novy <jnovy@redhat.com> - 1:1.24.6-5
- remove Containerfile man page
- Related: #2176055

* Mon Oct 31 2022 Jindrich Novy <jnovy@redhat.com> - 1:1.24.6-3
- update to the latest content of https://github.com/containers/buildah/tree/release-1.24
  (https://github.com/containers/buildah/commit/1320eb7)
- Related: #2129766

* Mon Sep 26 2022 Jindrich Novy <jnovy@redhat.com> - 1:1.24.6-2
- update to the latest content of https://github.com/containers/buildah/tree/release-1.24
  (https://github.com/containers/buildah/commit/4198a78)
- Related: #2123641

* Wed Sep 21 2022 Jindrich Novy <jnovy@redhat.com> - 1:1.24.6-1
- update to the latest content of https://github.com/containers/buildah/tree/release-1.24
  (https://github.com/containers/buildah/commit/efed577)
- Related: #2123641

* Tue Sep 20 2022 Jindrich Novy <jnovy@redhat.com> - 1:1.24.5-3
- update to the latest content of https://github.com/containers/buildah/tree/release-1.24
  (https://github.com/containers/buildah/commit/0ee422c)
- Related: #2123641

* Wed Aug 24 2022 Jindrich Novy <jnovy@redhat.com> - 1:1.24.5-2
- update to the latest content of https://github.com/containers/buildah/tree/release-1.24
  (https://github.com/containers/buildah/commit/8cc4586)
- Related: #2061390

* Thu Jul 28 2022 Jindrich Novy <jnovy@redhat.com> - 1:1.24.5-1
- update to the latest content of https://github.com/containers/buildah/tree/release-1.24
  (https://github.com/containers/buildah/commit/83c5f26)
- Related: #2061390

* Fri Jul 15 2022 Jindrich Novy <jnovy@redhat.com> - 1:1.24.4-3
- update to the latest content of https://github.com/containers/buildah/tree/release-1.24
  (https://github.com/containers/buildah/commit/e99286b)
- Related: #2061390

* Wed Jul 13 2022 Jindrich Novy <jnovy@redhat.com> - 1:1.24.4-2
- update to the latest content of https://github.com/containers/buildah/tree/release-1.24
  (https://github.com/containers/buildah/commit/63813ef)
- Related: #2061390

* Thu May 12 2022 Jindrich Novy <jnovy@redhat.com> - 1:1.24.4-1
- update to the latest content of https://github.com/containers/buildah/tree/release-1.24
  (https://github.com/containers/buildah/commit/c1bdaba)
- Related: #2061390

* Wed May 11 2022 Jindrich Novy <jnovy@redhat.com> - 1:1.24.3-1
- update to the latest content of https://github.com/containers/buildah/tree/release-1.24
  (https://github.com/containers/buildah/commit/d29224c)
- Related: #2061390

* Fri Apr 29 2022 Jindrich Novy <jnovy@redhat.com> - 1:1.24.2-5
- update to the latest content of https://github.com/containers/buildah/tree/release-1.24
  (https://github.com/containers/buildah/commit/9c456a3)
- Related: #2061390

* Fri Apr 29 2022 Jindrich Novy <jnovy@redhat.com> - 1:1.24.2-4
- use release-1.24 maintenance branch for the 4.0 stable stream
- Related: #2061390

* Fri Apr 08 2022 Jindrich Novy <jnovy@redhat.com> - 1:1.25.1-2
- bump golang BR to 1.17.7
- Related: #2061390

* Thu Mar 31 2022 Jindrich Novy <jnovy@redhat.com> - 1:1.25.1-1
- update to https://github.com/containers/buildah/releases/tag/v1.25.1
- Related: #2061390

* Wed Mar 30 2022 Jindrich Novy <jnovy@redhat.com> - 1:1.25.0-1
- update to https://github.com/containers/buildah/releases/tag/v1.25.0
- Related: #2061390

* Mon Feb 21 2022 Lokesh Mandvekar <lsm5@redhat.com> - 1:1.24.2-2
- Add patch to fix bash symtax for gating tests
- Upstream PR: https://github.com/containers/buildah/pull/3792
- Related: #2001445

* Thu Feb 17 2022 Jindrich Novy <jnovy@redhat.com> - 1:1.24.2-1
- update to https://github.com/containers/buildah/releases/tag/v1.24.2
- Related: #2001445

* Fri Feb 04 2022 Jindrich Novy <jnovy@redhat.com> - 1:1.24.1-1
- update to https://github.com/containers/buildah/releases/tag/v1.24.1
- Related: #2001445

* Wed Feb 02 2022 Jindrich Novy <jnovy@redhat.com> - 1:1.24.0-1
- update to https://github.com/containers/buildah/releases/tag/v1.24.0
- Related: #2001445

* Tue Jan 25 2022 Jindrich Novy <jnovy@redhat.com> - 1:1.23.2-1
- update to the latest content of https://github.com/containers/buildah/tree/release-1.23
  (https://github.com/containers/buildah/commit/83a66a7)
- Related: #2001445

* Mon Nov 22 2021 Jindrich Novy <jnovy@redhat.com> - 1:1.23.1-3
- update to the latest content of https://github.com/containers/buildah/tree/release-1.23
  (https://github.com/containers/buildah/commit/867c1bc)
- Related: #2001445

* Mon Oct 18 2021 Jindrich Novy <jnovy@redhat.com> - 1:1.23.1-2
- respect Epoch in subpackage dependencies
- Related: #2001445

* Fri Oct 15 2021 Jindrich Novy <jnovy@redhat.com> - 1:1.23.1-1
- bump Epoch to preserve upgrade path
- Related: #2001445

* Wed Oct 13 2021 Jindrich Novy <jnovy@redhat.com> - 1.23.1-0.1
- update to the latest content of https://github.com/containers/buildah/tree/release-1.23
  (https://github.com/containers/buildah/commit/87a0565)
- Related: #2001445

* Wed Oct 13 2021 Jindrich Novy <jnovy@redhat.com> - 1.24.0-0.18
- update to the latest content of https://github.com/containers/buildah/tree/main
  (https://github.com/containers/buildah/commit/982717a)
- Related: #2001445

* Mon Oct 11 2021 Jindrich Novy <jnovy@redhat.com> - 1.24.0-0.17
- update to the latest content of https://github.com/containers/buildah/tree/main
  (https://github.com/containers/buildah/commit/50869a7)
- Related: #2001445

* Fri Oct 08 2021 Jindrich Novy <jnovy@redhat.com> - 1.24.0-0.16
- update to the latest content of https://github.com/containers/buildah/tree/main
  (https://github.com/containers/buildah/commit/211972a)
- Related: #2001445

* Thu Oct 07 2021 Jindrich Novy <jnovy@redhat.com> - 1.24.0-0.15
- update to the latest content of https://github.com/containers/buildah/tree/main
  (https://github.com/containers/buildah/commit/c044ad6)
- Related: #2001445

* Wed Oct 06 2021 Jindrich Novy <jnovy@redhat.com> - 1.24.0-0.14
- update to the latest content of https://github.com/containers/buildah/tree/main
  (https://github.com/containers/buildah/commit/7807a0e)
- Related: #2001445

* Mon Oct 04 2021 Jindrich Novy <jnovy@redhat.com> - 1.24.0-0.13
- update to the latest content of https://github.com/containers/buildah/tree/main
  (https://github.com/containers/buildah/commit/0cd9445)
- Related: #2001445

* Fri Oct 01 2021 Jindrich Novy <jnovy@redhat.com> - 1.24.0-0.12
- update to the latest content of https://github.com/containers/buildah/tree/main
  (https://github.com/containers/buildah/commit/954c481)
- Related: #2001445

* Thu Sep 30 2021 Jindrich Novy <jnovy@redhat.com> - 1.24.0-0.11
- include all man pages
- Related: #2001445

* Thu Sep 30 2021 Jindrich Novy <jnovy@redhat.com> - 1.24.0-0.10
- update to the latest content of https://github.com/containers/buildah/tree/main
  (https://github.com/containers/buildah/commit/d2ef199)
- Related: #2001445

* Wed Sep 29 2021 Jindrich Novy <jnovy@redhat.com> - 1.24.0-0.9
- update to the latest content of https://github.com/containers/buildah/tree/main
  (https://github.com/containers/buildah/commit/455f2f1)
- Related: #2001445

* Mon Sep 27 2021 Jindrich Novy <jnovy@redhat.com> - 1.24.0-0.8
- update to the latest content of https://github.com/containers/buildah/tree/main
  (https://github.com/containers/buildah/commit/8548885)
- Related: #2001445

* Fri Sep 24 2021 Jindrich Novy <jnovy@redhat.com> - 1.24.0-0.7
- update to the latest content of https://github.com/containers/buildah/tree/main
  (https://github.com/containers/buildah/commit/9a49348)
- Related: #2001445

* Thu Sep 23 2021 Jindrich Novy <jnovy@redhat.com> - 1.24.0-0.6
- update to the latest content of https://github.com/containers/buildah/tree/main
  (https://github.com/containers/buildah/commit/a72aad4)
- Related: #2001445

* Wed Sep 22 2021 Jindrich Novy <jnovy@redhat.com> - 1.24.0-0.5
- update to the latest content of https://github.com/containers/buildah/tree/main
  (https://github.com/containers/buildah/commit/b8757e9)
- Related: #2001445

* Tue Sep 21 2021 Jindrich Novy <jnovy@redhat.com> - 1.24.0-0.4
- update to the latest content of https://github.com/containers/buildah/tree/main
  (https://github.com/containers/buildah/commit/fb84638)
- Related: #2001445

* Mon Sep 20 2021 Jindrich Novy <jnovy@redhat.com> - 1.24.0-0.3
- update to the latest content of https://github.com/containers/buildah/tree/main
  (https://github.com/containers/buildah/commit/f3f3c55)
- Related: #2001445

* Fri Sep 17 2021 Jindrich Novy <jnovy@redhat.com> - 1.24.0-0.2
- update to the latest content of https://github.com/containers/buildah/tree/main
  (https://github.com/containers/buildah/commit/753716a)
- Related: #2001445

* Wed Sep 15 2021 Jindrich Novy <jnovy@redhat.com> - 1.24.0-0.1
- update to the latest content of https://github.com/containers/buildah/tree/main
  (https://github.com/containers/buildah/commit/69b3e56)
- Related: #2001445

* Mon Sep 13 2021 Jindrich Novy <jnovy@redhat.com> - 1.23.0-0.2
- update to the latest content of https://github.com/containers/buildah/tree/main
  (https://github.com/containers/buildah/commit/e9bc224)
- Related: #2001445

* Fri Sep 10 2021 Jindrich Novy <jnovy@redhat.com> - 1.23.0-0.1
- update to the latest content of https://github.com/containers/buildah/tree/main
  (https://github.com/containers/buildah/commit/a5aba5c)
- Related: #2001445

* Wed Aug 25 2021 Jindrich Novy <jnovy@redhat.com> - 1.22.3-2
- update to the latest content of https://github.com/containers/buildah/tree/release-1.22
  (https://github.com/containers/buildah/commit/4d20222)
- Related: #1934415

* Fri Aug 20 2021 Lokesh Mandvekar <lsm5@redhat.com> - 1.22.3-1
- update to v1.22.3
- Related: #1934415

* Mon Aug 16 2021 Jindrich Novy <jnovy@redhat.com> - 1.22.0-3
- update to the latest content of https://github.com/containers/buildah/tree/release-1.22
  (https://github.com/containers/buildah/commit/98960f2)
- Related: #1934415

* Thu Aug 05 2021 Jindrich Novy <jnovy@redhat.com> - 1.22.0-2
- update to the latest content of https://github.com/containers/buildah/tree/release-1.22
  (https://github.com/containers/buildah/commit/71b8003)
- Related: #1934415

* Tue Aug 03 2021 Jindrich Novy <jnovy@redhat.com> - 1.22.0-1
- update to 1.22.0 release and switch to the release-1.22 maint branch
- Related: #1934415

* Mon Aug 02 2021 Jindrich Novy <jnovy@redhat.com> - 1.22.0-0.4
- update to the latest content of https://github.com/containers/buildah/tree/main
  (https://github.com/containers/buildah/commit/56ff12f)
- Related: #1934415

* Thu Jul 29 2021 Jindrich Novy <jnovy@redhat.com> - 1.22.0-0.3
- update to the latest content of https://github.com/containers/buildah/tree/main
  (https://github.com/containers/buildah/commit/f517d85)
- Related: #1934415

* Wed Jul 28 2021 Jindrich Novy <jnovy@redhat.com> - 1.22.0-0.2
- update to the latest content of https://github.com/containers/buildah/tree/main
  (https://github.com/containers/buildah/commit/42dbc97)
- Related: #1934415

* Wed Jul 28 2021 Jindrich Novy <jnovy@redhat.com> - 1.21.0-1
- update to the latest content of https://github.com/containers/buildah/tree/main
  (https://github.com/containers/buildah/commit/42dbc97)
- Related: #1934415

* Tue Jul 27 2021 Jindrich Novy <jnovy@redhat.com> - 1.22.0-0.1
- switch to main branch
- Related: #1934415

* Mon Jul 26 2021 Jindrich Novy <jnovy@redhat.com> - 1.21.4-2
- add buildah-copy helper
- Related: #1934415

* Mon Jul 26 2021 Jindrich Novy <jnovy@redhat.com> - 1.21.4-1
- update to the latest content of https://github.com/containers/buildah/tree/release-1.21
  (https://github.com/containers/buildah/commit/9c83683)
- Related: #1934415

* Wed Jul 21 2021 Jindrich Novy <jnovy@redhat.com> - 1.21.3-2
- update to the latest content of https://github.com/containers/buildah/tree/release-1.21
  (https://github.com/containers/buildah/commit/30a10f3)
- Related: #1934415

* Thu Jul 15 2021 Jindrich Novy <jnovy@redhat.com> - 1.21.3-1
- update to the latest content of https://github.com/containers/buildah/tree/release-1.21
  (https://github.com/containers/buildah/commit/7f9540d)
- Related: #1934415

* Thu Jul 01 2021 Jindrich Novy <jnovy@redhat.com> - 1.21.1-0.17
- update to buildah 1.21.1 from the release-1.21 upstream branch
- Related: #1934415

* Tue Jun 29 2021 Jindrich Novy <jnovy@redhat.com> - 1.22.0-0.16
- update to the latest content of https://github.com/containers/buildah/tree/main
  (https://github.com/containers/buildah/commit/3ed5d8e)
- Related: #1934415

* Mon Jun 28 2021 Jindrich Novy <jnovy@redhat.com> - 1.22.0-0.15
- update to the latest content of https://github.com/containers/buildah/tree/main
  (https://github.com/containers/buildah/commit/c7d828f)
- Related: #1934415

* Fri Jun 25 2021 Jindrich Novy <jnovy@redhat.com> - 1.22.0-0.14
- "buildah version" produces correct output
- Related: #1934415

* Fri Jun 25 2021 Jindrich Novy <jnovy@redhat.com> - 1.22.0-0.13
- update to the latest content of https://github.com/containers/buildah/tree/main
  (https://github.com/containers/buildah/commit/6bc611d)
- Related: #1934415

* Thu Jun 24 2021 Jindrich Novy <jnovy@redhat.com> - 1.22.0-0.12
- update to the latest content of https://github.com/containers/buildah/tree/main
  (https://github.com/containers/buildah/commit/3a0b52f)
- Related: #1934415

* Wed Jun 23 2021 Jindrich Novy <jnovy@redhat.com> - 1.22.0-0.11
- update to the latest content of https://github.com/containers/buildah/tree/master
  (https://github.com/containers/buildah/commit/6d5d1ae)
- Related: #1934415

* Tue Jun 22 2021 Jindrich Novy <jnovy@redhat.com> - 1.22.0-0.10
- update to the latest content of https://github.com/containers/buildah/tree/master
  (https://github.com/containers/buildah/commit/802a904)
- Related: #1934415

* Mon Jun 21 2021 Jindrich Novy <jnovy@redhat.com> - 1.22.0-0.9
- update to the latest content of https://github.com/containers/buildah/tree/master
  (https://github.com/containers/buildah/commit/5181b9c)
- Related: #1934415

* Fri Jun 18 2021 Jindrich Novy <jnovy@redhat.com> - 1.22.0-0.8
- update to the latest content of https://github.com/containers/buildah/tree/master
  (https://github.com/containers/buildah/commit/db16262)
- Related: #1934415

* Thu Jun 17 2021 Jindrich Novy <jnovy@redhat.com> - 1.22.0-0.7
- update to the latest content of https://github.com/containers/buildah/tree/master
  (https://github.com/containers/buildah/commit/30c07b7)
- Related: #1934415

* Wed Jun 16 2021 Jindrich Novy <jnovy@redhat.com> - 1.22.0-0.6
- update to the latest content of https://github.com/containers/buildah/tree/master
  (https://github.com/containers/buildah/commit/d99221f)
- Related: #1934415

* Mon Jun 14 2021 Jindrich Novy <jnovy@redhat.com> - 1.22.0-0.5
- update to the latest content of https://github.com/containers/buildah/tree/master
  (https://github.com/containers/buildah/commit/8d08247)
- Related: #1934415

* Thu Jun 10 2021 Jindrich Novy <jnovy@redhat.com> - 1.22.0-0.4
- update to the latest content of https://github.com/containers/buildah/tree/master
  (https://github.com/containers/buildah/commit/9c7f50b)
- Related: #1934415

* Mon Jun 07 2021 Jindrich Novy <jnovy@redhat.com> - 1.22.0-0.3
- update to the latest content of https://github.com/containers/buildah/tree/master
  (https://github.com/containers/buildah/commit/d08dbe7)
- Related: #1934415

* Thu Jun 03 2021 Jindrich Novy <jnovy@redhat.com> - 1.22.0-0.2
- update to the latest content of https://github.com/containers/buildah/tree/master
  (https://github.com/containers/buildah/commit/bbbe10a)
- Related: #1934415

* Wed Jun 02 2021 Jindrich Novy <jnovy@redhat.com> - 1.21.1-0.8
- update to the latest content of https://github.com/containers/buildah/tree/master
  (https://github.com/containers/buildah/commit/4fa566e)
- Related: #1934415

* Mon May 31 2021 Jindrich Novy <jnovy@redhat.com> - 1.21.1-0.7
- update to the latest content of https://github.com/containers/buildah/tree/master
  (https://github.com/containers/buildah/commit/8a6d840)
- Related: #1934415

* Wed May 26 2021 Jindrich Novy <jnovy@redhat.com> - 1.21.1-0.6
- update to the latest content of https://github.com/containers/buildah/tree/master
  (https://github.com/containers/buildah/commit/d677bf0)
- Related: #1934415

* Tue May 25 2021 Jindrich Novy <jnovy@redhat.com> - 1.21.1-0.5
- update to the latest content of https://github.com/containers/buildah/tree/master
  (https://github.com/containers/buildah/commit/df14b1c)
- Related: #1934415

* Fri May 21 2021 Jindrich Novy <jnovy@redhat.com> - 1.21.1-0.4
- update to the latest content of https://github.com/containers/buildah/tree/master
  (https://github.com/containers/buildah/commit/19d3065)
- Related: #1934415

* Fri May 21 2021 Jindrich Novy <jnovy@redhat.com> - 1.21.1-0.3
- update to the latest content of https://github.com/containers/buildah/tree/master
  (https://github.com/containers/buildah/commit/2a83637)
- Related: #1934415

* Thu May 20 2021 Jindrich Novy <jnovy@redhat.com> - 1.21.1-0.2
- update to the latest content of https://github.com/containers/buildah/tree/master
  (https://github.com/containers/buildah/commit/f629ded)
- Related: #1934415

* Wed May 19 2021 Jindrich Novy <jnovy@redhat.com> - 1.20.2-0.8
- update to the latest content of https://github.com/containers/buildah/tree/master
  (https://github.com/containers/buildah/commit/c3a3fe8)
- Related: #1934415

* Wed May 19 2021 Jindrich Novy <jnovy@redhat.com> - 1.20.2-0.7
- update to the latest content of https://github.com/containers/buildah/tree/master
  (https://github.com/containers/buildah/commit/f30b420)
- Related: #1934415

* Mon May 17 2021 Jindrich Novy <jnovy@redhat.com> - 1.20.2-0.6
- update to the latest content of https://github.com/containers/buildah/tree/master
  (https://github.com/containers/buildah/commit/162fbaf)
- Related: #1934415

* Thu May 13 2021 Jindrich Novy <jnovy@redhat.com> - 1.20.2-0.5
- update to the latest content of https://github.com/containers/buildah/tree/master
  (https://github.com/containers/buildah/commit/5119393)
- Related: #1934415

* Wed May 12 2021 Jindrich Novy <jnovy@redhat.com> - 1.20.2-0.4
- update to the latest content of https://github.com/containers/buildah/tree/master
  (https://github.com/containers/buildah/commit/a0853c3)
- Related: #1934415

* Tue May 11 2021 Jindrich Novy <jnovy@redhat.com> - 1.20.2-0.3
- update to the latest content of https://github.com/containers/buildah/tree/master
  (https://github.com/containers/buildah/commit/135d63d)
- Related: #1934415

* Tue May 11 2021 Jindrich Novy <jnovy@redhat.com> - 1.20.2-2
- fix release to reflect a development version
- Related: #1934415

* Fri May 07 2021 Jindrich Novy <jnovy@redhat.com> - 1.20.2-1
- update to the latest content of https://github.com/containers/buildah/tree/master
  (https://github.com/containers/buildah/commit/22fc573)
- Related: #1934415

* Mon Apr 26 2021 Jindrich Novy <jnovy@redhat.com> - 1.20.1-1
- update to https://github.com/containers/buildah/releases/tag/v1.20.1
- sync tests with Fedora
- Related: #1934415

* Fri Mar 26 2021 Jindrich Novy <jnovy@redhat.com> - 1.20.0-1
- update to https://github.com/containers/buildah/releases/tag/v1.20.0
- Related: #1934415

* Tue Mar 09 2021 Jindrich Novy <jnovy@redhat.com> - 1.19.8-1
- update to https://github.com/containers/buildah/releases/tag/v1.19.8
- Related: #1934415

* Fri Mar 05 2021 Jindrich Novy <jnovy@redhat.com> - 1.19.7-1
- update to https://github.com/containers/buildah/releases/tag/v1.19.7
- Related: #1934415

* Fri Feb 19 2021 Jindrich Novy <jnovy@redhat.com> - 1.19.6-1
- update to the latest content of https://github.com/containers/buildah/tree/release-1.19
  (https://github.com/containers/buildah/commit/7aedb16)
- Related: #1883490

* Thu Feb 18 2021 Jindrich Novy <jnovy@redhat.com> - 1.19.5-6
- update to the latest content of https://github.com/containers/buildah/tree/release-1.19
  (https://github.com/containers/buildah/commit/dcd385e)
- Related: #1883490

* Thu Feb 18 2021 Jindrich Novy <jnovy@redhat.com> - 1.19.5-5
- update to the latest content of https://github.com/containers/buildah/tree/release-1.19
  (https://github.com/containers/buildah/commit/016b90d)
- Related: #1883490

* Tue Feb 16 2021 Jindrich Novy <jnovy@redhat.com> - 1.19.5-4
- update to the latest content of https://github.com/containers/buildah/tree/release-1.19
  (https://github.com/containers/buildah/commit/e5384ed)
- Related: #1883490

* Mon Feb 15 2021 Jindrich Novy <jnovy@redhat.com> - 1.19.5-3
- update to the latest content of https://github.com/containers/buildah/tree/release-1.19
  (https://github.com/containers/buildah/commit/9dd415b)
- Related: #1883490

* Fri Feb 12 2021 Jindrich Novy <jnovy@redhat.com> - 1.19.5-2
- update to the latest content of https://github.com/containers/buildah/tree/release-1.19
  (https://github.com/containers/buildah/commit/db783f4)
- Related: #1883490

* Wed Feb 10 2021 Jindrich Novy <jnovy@redhat.com> - 1.19.5-1
- update to the latest content of https://github.com/containers/buildah/tree/release-1.19
  (https://github.com/containers/buildah/commit/1fb260f)
- Related: #1883490

* Tue Feb 09 2021 Jindrich Novy <jnovy@redhat.com> - 1.19.4-1
- update to the latest content of https://github.com/containers/buildah/tree/release-1.19
  (https://github.com/containers/buildah/commit/76beccc)
- Related: #1883490

* Sun Feb 07 2021 Jindrich Novy <jnovy@redhat.com> - 1.19.3-3
- update to the latest content of https://github.com/containers/buildah/tree/release-1.19
  (https://github.com/containers/buildah/commit/a96b716)
- Related: #1883490

* Sat Feb 06 2021 Jindrich Novy <jnovy@redhat.com> - 1.19.3-2
- update to the latest content of https://github.com/containers/buildah/tree/release-1.19
  (https://github.com/containers/buildah/commit/17521db)
- Related: #1883490

* Sun Jan 31 2021 Jindrich Novy <jnovy@redhat.com> - 1.19.3-1
- update to the latest content of https://github.com/containers/buildah/tree/release-1.19
  (https://github.com/containers/buildah/commit/af31e45)
- Related: #1883490

* Fri Jan 29 2021 Jindrich Novy <jnovy@redhat.com> - 1.19.2-2
- update to the latest content of https://github.com/containers/buildah/tree/release-1.19
  (https://github.com/containers/buildah/commit/06e091b)
- Related: #1883490

* Fri Jan 15 2021 Jindrich Novy <jnovy@redhat.com> - 1.19.2-1
- update to https://github.com/containers/buildah/releases/tag/v1.19.2
- Related: #1883490

* Fri Jan 15 2021 Jindrich Novy <jnovy@redhat.com> - 1.19.1-1
- update to https://github.com/containers/buildah/releases/tag/v1.19.1
- Related: #1883490

* Wed Jan 13 2021 Jindrich Novy <jnovy@redhat.com> - 1.19.0-2
- fix gating test issue with openssl cert
- Related: #1914884

* Sat Jan 09 2021 Jindrich Novy <jnovy@redhat.com> - 1.19.0-1
- update to https://github.com/containers/buildah/releases/tag/v1.19.0
- Related: #1883490

* Tue Dec 08 2020 Jindrich Novy <jnovy@redhat.com> - 1.18.0-1
- make build log more readable
- always build with debuginfo enabled
- Related: #1883490

* Thu Nov 05 2020 Jindrich Novy <jnovy@redhat.com> - 1.17.0-2
- simplify spec file
- use short commit ID in tarball name
- Related: #1883490

* Fri Oct 30 2020 Jindrich Novy <jnovy@redhat.com> - 1.17.0-1
- update to https://github.com/containers/buildah/releases/tag/v1.17.0
- Related: #1883490

* Fri Oct 23 2020 Jindrich Novy <jnovy@redhat.com> - 1.16.5-2
- use shortcommit ID in branch tarball name
- Related: #1883490

* Thu Oct 22 2020 Jindrich Novy <jnovy@redhat.com> - 1.16.5-1
- synchronize with stream-container-tools-rhel8
- Related: #1883490

* Thu Oct 22 2020 Jindrich Novy <jnovy@redhat.com> - 1.16.4-3
- update source tarball
- Related: #1883490

* Thu Oct 22 2020 Jindrich Novy <jnovy@redhat.com> - 1.16.4-2
- use the mainline 1.16.4 buildah release as starting point for 8.4.0
  not the content from upstream branch yet
- Related: #1883490

* Wed Oct 21 2020 Jindrich Novy <jnovy@redhat.com> - 1.16.4-1
- synchronize with stream-container-tools-rhel8
- Related: #1883490

* Tue Aug 11 2020 Jindrich Novy <jnovy@redhat.com> - 1.15.1-2
- propagate proper CFLAGS to CGO_CFLAGS to assure code hardening and optimization
- Related: #1821193

* Mon Aug 10 2020 Jindrich Novy <jnovy@redhat.com> - 1.15.1-1
- update to https://github.com/containers/buildah/releases/tag/v1.15.1
- Related: #1821193

* Fri Jul 17 2020 Jindrich Novy <jnovy@redhat.com> - 1.15.0-2
- fix "CVE-2020-14040 buildah: golang.org/x/text: possibility to trigger an infinite loop in encoding/unicode could lead to crash [rhel-8]"
- Resolves: #1854717

* Thu Jun 18 2020 Jindrich Novy <jnovy@redhat.com> - 1.15.0-1
- update to https://github.com/containers/buildah/releases/tag/v1.15.0
- Related: #1821193

* Wed Jun 10 2020 Jindrich Novy <jnovy@redhat.com> - 1.14.9-2
- exclude i686 arch
- Related: #1821193

* Tue May 19 2020 Jindrich Novy <jnovy@redhat.com> - 1.14.9-1
- update to https://github.com/containers/buildah/releases/tag/v1.14.9
- Related: #1821193

* Tue May 12 2020 Jindrich Novy <jnovy@redhat.com> - 1.14.8-1
- synchronize containter-tools 8.3.0 with 8.2.1
- Related: #1821193

* Wed Apr 01 2020 Jindrich Novy <jnovy@redhat.com> - 1.11.6-8
- fix "CVE-2020-10696 buildah: crafted input tar file may lead to local file overwriting during image build process"
- Resolves: #1819810

* Mon Feb 24 2020 Jindrich Novy <jnovy@redhat.com> - 1.11.6-7
- fix "COPY command takes long time with buildah"
- Resolves: #1806120

* Mon Feb 17 2020 Jindrich Novy <jnovy@redhat.com> - 1.11.6-6
- fix CVE-2020-1702
- Resolves: #1801926

* Thu Feb 13 2020 Jindrich Novy <jnovy@redhat.com> - 1.11.6-5
- adding the first phase of FIPS fix
- Related: #1784952

* Wed Dec 11 2019 Jindrich Novy <jnovy@redhat.com> - 1.11.6-4
- compile in FIPS mode
- Related: RHELPLAN-25139

* Mon Dec 09 2019 Jindrich Novy <jnovy@redhat.com> - 1.11.6-3
- be sure to use golang >= 1.12.12-4
- Related: RHELPLAN-25139

* Fri Dec 06 2019 Jindrich Novy <jnovy@redhat.com> - 1.11.6-2
- fix chroot: unmount with MNT_DETACH instead of UnmountMountpoints()
- bug reference 1772179
- Related: RHELPLAN-25139

* Thu Dec 05 2019 Jindrich Novy <jnovy@redhat.com> - 1.11.6-1
- update to buildah 1.11.6
- Related: RHELPLAN-25139

* Thu Nov 21 2019 Jindrich Novy <jnovy@redhat.com> - 1.11.5-1
- update to buildah 1.11.5
- Related: RHELPLAN-25139

* Thu Nov 07 2019 Jindrich Novy <jnovy@redhat.com> - 1.11.4-2
- fix %%gobuild macro to not to ignore BUILDTAGS
- Related: RHELPLAN-25139

* Thu Nov 07 2019 Jindrich Novy <jnovy@redhat.com> - 1.11.4-1
- update to 1.11.4
- Related: RHELPLAN-25139

* Tue Sep 17 2019 Jindrich Novy <jnovy@redhat.com> - 1.9.0-5
- Use autosetup macro again.

* Thu Sep 12 2019 Jindrich Novy <jnovy@redhat.com> - 1.9.0-4
- Fix CVE-2019-10214 (#1734653).

* Sat Jun 15 2019 Lokesh Mandvekar <lsm5@redhat.com> - 1.9.0-3
- Resolves: #1721247 - enable fips mode

* Sat Jun 15 2019 Lokesh Mandvekar <lsm5@redhat.com> - 1.9.0-2
- Resolves: #1720654 - tests subpackage depends on golang explicitly

* Sat Jun 15 2019 Lokesh Mandvekar <lsm5@redhat.com> - 1.9.0-1
- Resolves: #1720654 - rebase to v1.9.0

* Fri Jun 14 2019 Lokesh Mandvekar <lsm5@redhat.com> - 1.8.3-1
- Resolves: #1720654 - rebase to v1.8.3

* Tue Apr  9 2019 Eduardo Santiago <santiago@redhat.com> - 1.8-0.git021d607
- package system tests

* Tue Dec 18 2018 Frantisek Kluknavsky <fkluknav@redhat.com> - 1.5-3.gite94b4f9
- re-enable debuginfo

* Mon Dec 17 2018 Frantisek Kluknavsky <fkluknav@redhat.com> - 1.5-2.gite94b4f9
- go toolset not in scl anymore

* Fri Nov 23 2018 Frantisek Kluknavsky <fkluknav@redhat.com> - 1.5-1.gite94b4f9
- rebase

* Mon Nov 19 2018 Frantisek Kluknavsky <fkluknav@redhat.com> - 1.4-3.git608fa84
- fedora-like go compiler macro in buildrequires is enough

* Wed Oct 10 2018 Frantisek Kluknavsky <fkluknav@redhat.com> - 1.4-2.git608fa84
- rebase

* Mon Aug 13 2018 Lokesh Mandvekar <lsm5@redhat.com> - 1.3-3.git4888163
- Resolves: #1615611 - rebuild with gobuild tag 'no_openssl'

* Wed Aug 08 2018 Lokesh Mandvekar <lsm5@redhat.com> - 1.3-2.git4888163
- Resolves: #1614009 - built with updated scl-ized go-toolset dep
- build with %%gobuild

* Sun Aug 5 2018 Dan Walsh <dwalsh@redhat.com> - 1.3-1
- Bump to v1.3
- Vendor in lates containers/image
- build-using-dockerfile: let -t include transports again
- Block use of /proc/acpi and /proc/keys from inside containers
- Fix handling of --registries-conf
- Fix becoming a maintainer link
- add optional CI test fo darwin
- Don't pass a nil error to errors.Wrapf()
- image filter test: use kubernetes/pause as a "since"
- Add --cidfile option to from
- vendor: update containers/storage
- Contributors need to find the CONTRIBUTOR.md file easier
- Add a --loglevel option to build-with-dockerfile
- Create Development plan
- cmd: Code improvement
- allow buildah cross compile for a darwin target
- Add unused function param lint check
- docs: Follow man-pages(7) suggestions for SYNOPSIS
- Start using github.com/seccomp/containers-golang
- umount: add all option to umount all mounted containers
- runConfigureNetwork(): remove an unused parameter
- Update github.com/opencontainers/selinux
- Fix buildah bud --layers
- Force ownership of /etc/hosts and /etc/resolv.conf to 0:0
- main: if unprivileged, reexec in a user namespace
- Vendor in latest imagebuilder
- Reduce the complexity of the buildah.Run function
- mount: output it before replacing lastError
- Vendor in latest selinux-go code
- Implement basic recognition of the "--isolation" option
- Run(): try to resolve non-absolute paths using $PATH
- Run(): don't include any default environment variables
- build without seccomp
- vendor in latest runtime-tools
- bind/mount_unsupported.go: remove import errors
- Update github.com/opencontainers/runc
- Add Capabilities lists to BuilderInfo
- Tweaks for commit tests
- commit: recognize committing to second storage locations
- Fix ARGS parsing for run commands
- Add info on registries.conf to from manpage
- Switch from using docker to podman for testing in .papr
- buildah: set the HTTP User-Agent
- ONBUILD tutorial
- Add information about the configuration files to the install docs
- Makefile: add uninstall
- Add tilde info for push to troubleshooting
- mount: support multiple inputs
- Use the right formatting when adding entries to /etc/hosts
- Vendor in latest go-selinux bindings
- Allow --userns-uid-map/--userns-gid-map to be global options
- bind: factor out UnmountMountpoints
- Run(): simplify runCopyStdio()
- Run(): handle POLLNVAL results
- Run(): tweak terminal mode handling
- Run(): rename 'copyStdio' to 'copyPipes'
- Run(): don't set a Pdeathsig for the runtime
- Run(): add options for adding and removing capabilities
- Run(): don't use a callback when a slice will do
- setupSeccomp(): refactor
- Change RunOptions.Stdin/Stdout/Stderr to just be Reader/Writers
- Escape use of '_' in .md docs
- Break out getProcIDMappings()
- Break out SetupIntermediateMountNamespace()
- Add Multi From Demo
- Use the c/image conversion code instead of converting configs manually
- Don't throw away the manifest MIME type and guess again
- Consolidate loading manifest and config in initConfig
- Pass a types.Image to Builder.initConfig
- Require an image ID in importBuilderDataFromImage
- Use c/image/manifest.GuessMIMEType instead of a custom heuristic
- Do not ignore any parsing errors in initConfig
- Explicitly handle "from scratch" images in Builder.initConfig
- Fix parsing of OCI images
- Simplify dead but dangerous-looking error handling
- Don't ignore v2s1 history if docker_version is not set
- Add --rm and --force-rm to buildah bud
- Add --all,-a flag to buildah images
- Separate stdio buffering from writing
- Remove tty check from images --format
- Add environment variable BUILDAH_RUNTIME
- Add --layers and --no-cache to buildah bud
- Touch up images man
- version.md: fix DESCRIPTION
- tests: add containers test
- tests: add images test
- images: fix usage
- fix make clean error
- Change 'registries' to 'container registries' in man
- add commit test
- Add(): learn to record hashes of what we add
- Minor update to buildah config documentation for entrypoint
- Bump to v1.2-dev
- Add registries.conf link to a few man pages

* Tue Jul 24 2018 Lokesh Mandvekar <lsm5@redhat.com> - 1.2-3
- do not depend on btrfs-progs for rhel8

* Thu Jul 19 2018 Dan Walsh <dwalsh@redhat.com> - 1.2-2
- buildah does not require ostree

* Sun Jul 15 2018 Dan Walsh <dwalsh@redhat.com> 1.2-1
- Vendor in latest containers/image
- build-using-dockerfile: let -t include transports again
- Block use of /proc/acpi and /proc/keys from inside containers
- Fix handling of --registries-conf
- Fix becoming a maintainer link
- add optional CI test fo darwin
- Don't pass a nil error to errors.Wrapf()
- image filter test: use kubernetes/pause as a "since"
- Add --cidfile option to from
- vendor: update containers/storage
- Contributors need to find the CONTRIBUTOR.md file easier
- Add a --loglevel option to build-with-dockerfile
- Create Development plan
- cmd: Code improvement
- allow buildah cross compile for a darwin target
- Add unused function param lint check
- docs: Follow man-pages(7) suggestions for SYNOPSIS
- Start using github.com/seccomp/containers-golang
- umount: add all option to umount all mounted containers
- runConfigureNetwork(): remove an unused parameter
- Update github.com/opencontainers/selinux
- Fix buildah bud --layers
- Force ownership of /etc/hosts and /etc/resolv.conf to 0:0
- main: if unprivileged, reexec in a user namespace
- Vendor in latest imagebuilder
- Reduce the complexity of the buildah.Run function
- mount: output it before replacing lastError
- Vendor in latest selinux-go code
- Implement basic recognition of the "--isolation" option
- Run(): try to resolve non-absolute paths using $PATH
- Run(): don't include any default environment variables
- build without seccomp
- vendor in latest runtime-tools
- bind/mount_unsupported.go: remove import errors
- Update github.com/opencontainers/runc
- Add Capabilities lists to BuilderInfo
- Tweaks for commit tests
- commit: recognize committing to second storage locations
- Fix ARGS parsing for run commands
- Add info on registries.conf to from manpage
- Switch from using docker to podman for testing in .papr
- buildah: set the HTTP User-Agent
- ONBUILD tutorial
- Add information about the configuration files to the install docs
- Makefile: add uninstall
- Add tilde info for push to troubleshooting
- mount: support multiple inputs
- Use the right formatting when adding entries to /etc/hosts
- Vendor in latest go-selinux bindings
- Allow --userns-uid-map/--userns-gid-map to be global options
- bind: factor out UnmountMountpoints
- Run(): simplify runCopyStdio()
- Run(): handle POLLNVAL results
- Run(): tweak terminal mode handling
- Run(): rename 'copyStdio' to 'copyPipes'
- Run(): don't set a Pdeathsig for the runtime
- Run(): add options for adding and removing capabilities
- Run(): don't use a callback when a slice will do
- setupSeccomp(): refactor
- Change RunOptions.Stdin/Stdout/Stderr to just be Reader/Writers
- Escape use of '_' in .md docs
- Break out getProcIDMappings()
- Break out SetupIntermediateMountNamespace()
- Add Multi From Demo
- Use the c/image conversion code instead of converting configs manually
- Don't throw away the manifest MIME type and guess again
- Consolidate loading manifest and config in initConfig
- Pass a types.Image to Builder.initConfig
- Require an image ID in importBuilderDataFromImage
- Use c/image/manifest.GuessMIMEType instead of a custom heuristic
- Do not ignore any parsing errors in initConfig
- Explicitly handle "from scratch" images in Builder.initConfig
- Fix parsing of OCI images
- Simplify dead but dangerous-looking error handling
- Don't ignore v2s1 history if docker_version is not set
- Add --rm and --force-rm to buildah bud
- Add --all,-a flag to buildah images
- Separate stdio buffering from writing
- Remove tty check from images --format
- Add environment variable BUILDAH_RUNTIME
- Add --layers and --no-cache to buildah bud
- Touch up images man
- version.md: fix DESCRIPTION
- tests: add containers test
- tests: add images test
- images: fix usage
- fix make clean error
- Change 'registries' to 'container registries' in man
- add commit test
- Add(): learn to record hashes of what we add
- Minor update to buildah config documentation for entrypoint
- Add registries.conf link to a few man pages

* Sun Jun 10 2018 Dan Walsh <dwalsh@redhat.com> 1.1-1
- Drop capabilities if running container processes as non root
- Print Warning message if cmd will not be used based on entrypoint
- Update 01-intro.md
- Shouldn't add insecure registries to list of search registries
- Report errors on bad transports specification when pushing images
- Move parsing code out of common for namespaces and into pkg/parse.go
- Add disable-content-trust noop flag to bud
- Change freenode chan to buildah
- runCopyStdio(): don't close stdin unless we saw POLLHUP
- Add registry errors for pull
- runCollectOutput(): just read until the pipes are closed on us
- Run(): provide redirection for stdio
- rmi, rm: add test
- add mount test
- Add parameter judgment for commands that do not require parameters
- Add context dir to bud command in baseline test
- run.bats: check that we can run with symlinks in the bundle path
- Give better messages to users when image can not be found
- use absolute path for bundlePath
- Add environment variable to buildah --format
- rm: add validation to args and all option
- Accept json array input for config entrypoint
- Run(): process RunOptions.Mounts, and its flags
- Run(): only collect error output from stdio pipes if we created some
- Add OnBuild support for Dockerfiles
- Quick fix on demo readme
- run: fix validate flags
- buildah bud should require a context directory or URL
- Touchup tutorial for run changes
- Validate common bud and from flags
- images: Error if the specified imagename does not exist
- inspect: Increase err judgments to avoid panic
- add test to inspect
- buildah bud picks up ENV from base image
- Extend the amount of time travis_wait should wait
- Add a make target for Installing CNI plugins
- Add tests for namespace control flags
- copy.bats: check ownerships in the container
- Fix SELinux test errors when SELinux is enabled
- Add example CNI configurations
- Run: set supplemental group IDs
- Run: use a temporary mount namespace
- Use CNI to configure container networks
- add/secrets/commit: Use mappings when setting permissions on added content
- Add CLI options for specifying namespace and cgroup setup
- Always set mappings when using user namespaces
- Run(): break out creation of stdio pipe descriptors
- Read UID/GID mapping information from containers and images
- Additional bud CI tests
- Run integration tests under travis_wait in Travis
- build-using-dockerfile: add --annotation
- Implement --squash for build-using-dockerfile and commit
- Vendor in latest container/storage for devicemapper support
- add test to inspect
- Vendor github.com/onsi/ginkgo and github.com/onsi/gomega
- Test with Go 1.10, too
- Add console syntax highlighting to troubleshooting page
- bud.bats: print "$output" before checking its contents
- Manage "Run" containers more closely
- Break Builder.Run()'s "run runc" bits out
- util.ResolveName(): handle completion for tagged/digested image names
- Handle /etc/hosts and /etc/resolv.conf properly in container
- Documentation fixes
- Make it easier to parse our temporary directory as an image name
- Makefile: list new pkg/ subdirectoris as dependencies for buildah
- containerImageSource: return more-correct errors
- API cleanup: PullPolicy and TerminalPolicy should be types
- Make "run --terminal" and "run -t" aliases for "run --tty"
- Vendor github.com/containernetworking/cni v0.6.0
- Update github.com/containers/storage
- Update github.com/projectatomic/libpod
- Add support for buildah bud --label
- buildah push/from can push and pull images with no reference
- Vendor in latest containers/image
- Update gometalinter to fix install.tools error
- Update troubleshooting with new run workaround
- Added a bud demo and tidied up
- Attempt to download file from url, if fails assume Dockerfile
- Add buildah bud CI tests for ENV variables
- Re-enable rpm .spec version check and new commit test
- Update buildah scratch demo to support el7
- Added Docker compatibility demo
- Update to F28 and new run format in baseline test
- Touchup man page short options across man pages
- Added demo dir and a demo. chged distrorlease
- builder-inspect: fix format option
- Add cpu-shares short flag (-c) and cpu-shares CI tests
- Minor fixes to formatting in rpm spec changelog
- Fix rpm .spec changelog formatting
- CI tests and minor fix for cache related noop flags
- buildah-from: add effective value to mount propagation

* Mon May 7 2018 Dan Walsh <dwalsh@redhat.com> 1.0-1
- Remove buildah run cmd and entrypoint execution
- Add Files section with registries.conf to pertinent man pages
- Force "localhost" as a default registry
- Add --compress, --rm, --squash flags as a noop for bud
- Add FIPS mode secret to buildah run and bud
- Add config --comment/--domainname/--history-comment/--hostname
- Add support for --iidfile to bud and commit
- Add /bin/sh -c to entrypoint in config
- buildah images and podman images are listing different sizes
- Remove tarball as an option from buildah push --help
- Update entrypoint behaviour to match docker
- Display imageId after commit
- config: add support for StopSignal
- Allow referencing stages as index and names
- Add multi-stage builds support
- Vendor in latest imagebuilder, to get mixed case AS support
- Allow umount to have multi-containers
- Update buildah push doc
- buildah bud walks symlinks
- Imagename is required for commit atm, update manpage

* Thu May 03 2018 Lokesh Mandvekar <lsm5@redhat.com> - 0.16-3.git532e267
- Resolves: #1573681
- built commit 532e267

* Tue Apr 10 2018 Lokesh Mandvekar <lsm5@redhat.com> - 0.16.0-2.git6f7d05b
- built commit 6f7d05b

* Wed Apr 4 2018 Dan Walsh <dwalsh@redhat.com> 0.16-1
-   Add support for shell
-   Vendor in latest containers/image
-    	 docker-archive generates docker legacy compatible images
-	 Do not create $DiffID subdirectories for layers with no configs
- 	 Ensure the layer IDs in legacy docker/tarfile metadata are unique
-	 docker-archive: repeated layers are symlinked in the tar file
-	 sysregistries: remove all trailing slashes
-	 Improve docker/* error messages
-	 Fix failure to make auth directory
-	 Create a new slice in Schema1.UpdateLayerInfos
-	 Drop unused storageImageDestination.{image,systemContext}
-	 Load a *storage.Image only once in storageImageSource
-	 Support gzip for docker-archive files
-	 Remove .tar extension from blob and config file names
-	 ostree, src: support copy of compressed layers
-	 ostree: re-pull layer if it misses uncompressed_digest|uncompressed_size
-	 image: fix docker schema v1 -> OCI conversion
-	 Add /etc/containers/certs.d as default certs directory
-  Change image time to locale, add troubleshooting.md, add logo to other mds
-   Allow --cmd parameter to have commands as values
-   Document the mounts.conf file
-   Fix man pages to format correctly
-   buildah from now supports pulling images using the following transports:
-   docker-archive, oci-archive, and dir.
-   If the user overrides the storage driver, the options should be dropped
-   Show Config/Manifest as JSON string in inspect when format is not set
-   Adds feature to pull compressed docker-archive files

* Tue Feb 27 2018 Dan Walsh <dwalsh@redhat.com> 0.15-1
- Fix handling of buildah run command options

* Mon Feb 26 2018 Dan Walsh <dwalsh@redhat.com> 0.14-1
- If commonOpts do not exist, we should return rather then segfault
- Display full error string instead of just status
- Implement --volume and --shm-size for bud and from
- Fix secrets patch for buildah bud
- Fixes the naming issue of blobs and config for the dir transport by removing the .tar extension

* Mon Feb 26 2018 Lokesh Mandvekar <lsm5@redhat.com> - 0.13-1.git99066e0
- use correct version

* Mon Feb 26 2018 Lokesh Mandvekar <lsm5@redhat.com> - 0.12-4.git99066e0
- enable debuginfo

* Mon Feb 26 2018 Lokesh Mandvekar <lsm5@redhat.com> - 0.12-3.git99066e0
- BR: libseccomp-devel

* Mon Feb 26 2018 Lokesh Mandvekar <lsm5@redhat.com> - 0.12-2.git99066e0
- Resolves: #1548535
- built commit 99066e0

* Mon Feb 12 2018 Dan Walsh <dwalsh@redhat.com> 0.12-1
- Added handing for simpler error message for Unknown Dockerfile instructions.
- Change default certs directory to /etc/containers/certs.dir
- Vendor in latest containers/image
- Vendor in latest containers/storage
- build-using-dockerfile: set the 'author' field for MAINTAINER
- Return exit code 1 when buildah-rmi fails
- Trim the image reference to just its name before calling getImageName
- Touch up rmi -f usage statement
- Add --format and --filter to buildah containers
- Add --prune,-p option to rmi command
- Add authfile param to commit
- Fix --runtime-flag for buildah run and bud
- format should override quiet for images
- Allow all auth params to work with bud
- Do not overwrite directory permissions on --chown
- Unescape HTML characters output into the terminal
- Fix: setting the container name to the image
- Prompt for un/pwd if not supplied with --creds
- Make bud be really quiet
- Return a better error message when failed to resolve an image
- Update auth tests and fix bud man page

* Mon Feb 05 2018 Lokesh Mandvekar <lsm5@redhat.com> - 0.11-3.git49095a8
- Resolves: #1542236 - add ostree and bump runc dep

* Thu Feb 01 2018 Frantisek Kluknavsky <fkluknav@redhat.com> - 0.11-2.git49095a8
- rebased to 49095a83f8622cf69532352d183337635562e261

* Tue Jan 16 2018 Dan Walsh <dwalsh@redhat.com> 0.11-1
- Add --all to remove containers
- Add --all functionality to rmi
- Show ctrid when doing rm -all
- Ignore sequential duplicate layers when reading v2s1
- Lots of minor bug fixes
- Vendor in latest containers/image and containers/storage

* Sat Dec 23 2017 Dan Walsh <dwalsh@redhat.com> 0.10-2
- Fix checkin

* Sat Dec 23 2017 Dan Walsh <dwalsh@redhat.com> 0.10-1
- Display Config and Manifest as strings
- Bump containers/image
- Use configured registries to resolve image names
- Update to work with newer image library
- Add --chown option to add/copy commands

* Tue Dec 12 2017 Lokesh Mandvekar <lsm5@redhat.com> - 0.9-2.git04ea079
- build for all arches

* Sat Dec 2 2017 Dan Walsh <dwalsh@redhat.com> 0.9-1
- Allow push to use the image id
- Make sure builtin volumes have the correct label

* Wed Nov 22 2017 Dan Walsh <dwalsh@redhat.com> 0.8-1
- Buildah bud was failing on SELinux machines, this fixes this
- Block access to certain kernel file systems inside of the container

* Thu Nov 16 2017 Dan Walsh <dwalsh@redhat.com> 0.7-1
- Ignore errors when trying to read containers buildah.json for loading SELinux reservations
-     Use credentials from kpod login for buildah
- Adds support for converting manifest types when using the dir transport
- Rework how we do UID resolution in images
- Bump github.com/vbatts/tar-split
- Set option.terminal appropriately in run

* Thu Nov 16 2017 Frantisek Kluknavsky <fkluknav@redhat.com> - 0.5-5.gitf7dc659
- revert building for s390x, it is intended for rhel 7.5

* Wed Nov 15 2017 Dan Walsh <dwalsh@redhat.com> 0.5-4
- Add requires for container-selinux

* Mon Nov 13 2017 Frantisek Kluknavsky <fkluknav@redhat.com> - 0.5-3.gitf7dc659
- build for s390x, https://bugzilla.redhat.com/show_bug.cgi?id=1482234

* Wed Nov 08 2017 Dan Walsh <dwalsh@redhat.com> 0.5-2
-  Bump github.com/vbatts/tar-split
-  Fixes CVE That could allow a container image to cause a DOS

* Tue Nov 07 2017 Dan Walsh <dwalsh@redhat.com> 0.5-1
-  Add secrets patch to buildah
-  Add proper SELinux labeling to buildah run
-  Add tls-verify to bud command
-  Make filtering by date use the image's date
-  images: don't list unnamed images twice
-  Fix timeout issue
-  Add further tty verbiage to buildah run
-  Make inspect try an image on failure if type not specified
-  Add support for `buildah run --hostname`
-  Tons of bug fixes and code cleanup

* Tue Nov  7 2017 Nalin Dahyabhai <nalin@redhat.com> - 0.4-2.git01db066
- bump to latest version
- set GIT_COMMIT at build-time

* Fri Sep 22 2017 Dan Walsh <dwalsh@redhat.com> 0.4-1.git9cbccf88c
-   Add default transport to push if not provided
-   Avoid trying to print a nil ImageReference
-   Add authentication to commit and push
-   Add information on buildah from man page on transports
-   Remove --transport flag
-   Run: do not complain about missing volume locations
-   Add credentials to buildah from
-   Remove export command
-   Run(): create the right working directory
-   Improve "from" behavior with unnamed references
-   Avoid parsing image metadata for dates and layers
-   Read the image's creation date from public API
-   Bump containers/storage and containers/image
-   Don't panic if an image's ID can't be parsed
-   Turn on --enable-gc when running gometalinter
-   rmi: handle truncated image IDs

* Fri Sep 22 2017 Lokesh Mandvekar <lsm5@redhat.com> - 0.4-1.git9cbccf8
- bump to v0.4

* Wed Aug 02 2017 Fedora Release Engineering <releng@fedoraproject.org> - 0.3-4.gitb9b2a8a
- Rebuilt for https://fedoraproject.org/wiki/Fedora_27_Binutils_Mass_Rebuild

* Wed Jul 26 2017 Fedora Release Engineering <releng@fedoraproject.org> - 0.3-3.gitb9b2a8a
- Rebuilt for https://fedoraproject.org/wiki/Fedora_27_Mass_Rebuild

* Thu Jul 20 2017 Dan Walsh <dwalsh@redhat.com> 0.3-2.gitb9b2a8a7e
- Bump for inclusion of OCI 1.0 Runtime and Image Spec

* Tue Jul 18 2017 Dan Walsh <dwalsh@redhat.com> 0.2.0-1.gitac2aad6
-   buildah run: Add support for -- ending options parsing
-   buildah Add/Copy support for glob syntax
-   buildah commit: Add flag to remove containers on commit
-   buildah push: Improve man page and help information
-   buildah run: add a way to disable PTY allocation
-   Buildah docs: clarify --runtime-flag of run command
-   Update to match newer storage and image-spec APIs
-   Update containers/storage and containers/image versions
-   buildah export: add support
-   buildah images: update commands
-   buildah images: Add JSON output option
-   buildah rmi: update commands
-   buildah containers: Add JSON output option
-   buildah version: add command
-   buildah run: Handle run without an explicit command correctly
-   Ensure volume points get created, and with perms
-   buildah containers: Add a -a/--all option

* Wed Jun 14 2017 Dan Walsh <dwalsh@redhat.com> 0.1.0-2.git597d2ab9
- Release Candidate 1
- All features have now been implemented.

* Fri Apr 14 2017 Dan Walsh <dwalsh@redhat.com> 0.0.1-1.git7a0a5333
- First package for Fedora
