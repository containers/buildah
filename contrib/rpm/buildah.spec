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
Version:        0.6
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
BuildRequires:  glib2-devel
BuildRequires:  ostree-devel
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
* Wed Nov 15 2017 Dan Walsh <dwalsh@redhat.com> 0.6-1
- Adds support for converting manifest types when using the dir transport
- Rework how we do UID resolution in images
- Bump github.com/vbatts/tar-split
- Set option.terminal appropriately in run

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

* Thu Jul 20 2017 Dan Walsh <dwalsh@redhat.com> 0.3.0-1
- Bump for inclusion of OCI 1.0 Runtime and Image Spec

* Tue Jul 18 2017 Dan Walsh <dwalsh@redhat.com> 0.2.0-1
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

* Fri Apr 14 2017 Dan Walsh <dwalsh@redhat.com> 0.0.1-1.git7a0a5333
- First package for Fedora
