%if 0%{?fedora} || 0%{?rhel} == 6
%global with_bundled 1
%global with_debug 0
%global with_check 1
%else
%global with_bundled 0
%global with_debug 0
%global with_check 0
%endif

%if 0%{?with_debug}
%global _dwz_low_mem_die_limit 0
%else
%global debug_package   %{nil}
%endif

%global provider        github
%global provider_tld    com
%global project         projectatomic
%global repo            buildah
# https://github.com/projectatomic/buildah
%global provider_prefix %{provider}.%{provider_tld}/%{project}/%{repo}
%global import_path     %{provider_prefix}
%global commit         REPLACEWITHCOMMITID
%global shortcommit    %(c=%{commit}; echo ${c:0:7})

Name:           buildah
# Bump version in buildah.go too
Version:        0.15
Release:        1.git%{shortcommit}%{?dist}
Summary:        A command line tool used to creating OCI Images
License:        ASL 2.0
URL:            https://%{provider_prefix}
Source:         https://%{provider_prefix}/archive/%{commit}/%{repo}-%{shortcommit}.tar.gz

ExclusiveArch:  x86_64 aarch64 ppc64le
# If go_compiler is not set to 1, there is no virtual provide. Use golang instead.
BuildRequires:  %{?go_compiler:compiler(go-compiler)}%{!?go_compiler:golang}
BuildRequires:  git
BuildRequires:  go-md2man
BuildRequires:  gpgme-devel
BuildRequires:  device-mapper-devel
BuildRequires:  btrfs-progs-devel
BuildRequires:  libassuan-devel
BuildRequires:  libseccomp-devel
BuildRequires:  glib2-devel
BuildRequires:  ostree-devel
BuildRequires:  make
Requires:       runc >= 1.0.0-6
Requires:       container-selinux
Requires:       skopeo-containers
Provides:       %{repo} = %{version}-%{release}

%description
The buildah package provides a command line tool which can be used to
* create a working container from scratch
or
* create a working container from an image as a starting point
* mount/umount a working container's root file system for manipulation
* save container's root file system layer to create a new image
* delete a working container or an image

%prep
%autosetup -Sgit -n %{name}-%{commit}

%build
mkdir _build
pushd _build
mkdir -p src/%{provider}.%{provider_tld}/%{project}
ln -s $(dirs +1 -l) src/%{import_path}
popd

mv vendor src

export GOPATH=$(pwd)/_build:$(pwd):%{gopath}
make all GIT_COMMIT=%{shortcommit}

%install
export GOPATH=$(pwd)/_build:$(pwd):%{gopath}

make DESTDIR=%{buildroot} PREFIX=%{_prefix} install install.completions

#define license tag if not already defined
%{!?_licensedir:%global license %doc}

%files
%license LICENSE
%doc README.md
%{_bindir}/%{name}
%{_mandir}/man1/buildah*
%{_datadir}/bash-completion/completions/*

%changelog
* Tue Feb 27 2018 Dan Walsh <dwalsh@redhat.com> 0.15-1
- Fix handling of buildah run command options

* Mon Feb 26 2018 Dan Walsh <dwalsh@redhat.com> 0.14-1
- If commonOpts do not exist, we should return rather then segfault
- Display full error string instead of just status
- Implement --volume and --shm-size for bud and from
- Fix secrets patch for buildah bud
- Fixes the naming issue of blobs and config for the dir transport by removing the .tar extension

* Thu Feb 22 2018 Dan Walsh <dwalsh@redhat.com> 0.13-1
- Vendor in latest containers/storage
-    This fixes a large SELinux bug.  
- run: do not open /etc/hosts if not needed
- Add the following flags to buildah bud and from
            --add-host
            --cgroup-parent
            --cpu-period
            --cpu-quota
            --cpu-shares
            --cpuset-cpus
            --cpuset-mems
            --memory
            --memory-swap
            --security-opt
            --ulimit

* Mon Feb 12 2018 Dan Walsh <dwalsh@redhat.com> 0.12-1
- Added handing for simpler error message for Unknown Dockerfile instructions.
- Change default certs directory to /etc/containers/certs.d
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

* Tue Jan 16 2018 Dan Walsh <dwalsh@redhat.com> 0.11-1
- Add --all to remove containers
- Add --all functionality to rmi
- Show ctrid when doing rm -all
- Ignore sequential duplicate layers when reading v2s1
- Lots of minor bug fixes
- Vendor in latest containers/image and containers/storage

* Sat Dec 23 2017 Dan Walsh <dwalsh@redhat.com> 0.10-1
- Display Config and Manifest as strings
- Bump containers/image
- Use configured registries to resolve image names
- Update to work with newer image library
- Add --chown option to add/copy commands

* Sat Dec 2 2017 Dan Walsh <dwalsh@redhat.com> 0.9-1
- Allow push to use the image id
- Make sure builtin volumes have the correct label

* Thu Nov 16 2017 Dan Walsh <dwalsh@redhat.com> 0.8-1
- Buildah bud was failing on SELinux machines, this fixes this
- Block access to certain kernel file systems inside of the container

* Thu Nov 16 2017 Dan Walsh <dwalsh@redhat.com> 0.7-1
- Ignore errors when trying to read containers buildah.json for loading SELinux reservations
-     Use credentials from kpod login for buildah

* Wed Nov 15 2017 Dan Walsh <dwalsh@redhat.com> 0.6-1
- Adds support for converting manifest types when using the dir transport
- Rework how we do UID resolution in images
- Bump github.com/vbatts/tar-split
- Set option.terminal appropriately in run

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

* Tue Aug 15 2017 Josh Boyer <jwboyer@redhat.com> - 0.3-5.gitb9b2a8a
- Build for s390x as well

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
