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
%global project         containers
%global repo            buildah
# https://github.com/containers/buildah
%global provider_prefix %{provider}.%{provider_tld}/%{project}/%{repo}
%global import_path     %{provider_prefix}
%global commit         REPLACEWITHCOMMITID
%global shortcommit    %(c=%{commit}; echo ${c:0:7})

Name:           buildah
# Bump version in buildah.go too
Version:        1.6-dev
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
%if 0%{?rhel} == 7
Requires:       container-selinux
Requires:       skopeo-containers
%else
Recommends:     container-selinux
Requires:       containers-common
%endif
Requires:       shadow-utils
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
* Wed Nov 21 2018 Tom Sweeney <tsweeney@redhat.com> 1.6-dev-1

* Wed Nov 21 2018 Tom Sweeney <tsweeney@redhat.com> 1.5-1
- Bump min go to 1.10 in install.md
- vendor: update ostree-go
- Update docker build command line in conformance test
- Print command in SystemExec as debug information
- Add some skip word for inspect check in conformance test
- Update regex for multi stage base test
- Sort CLI flags
- vendor: update containers/storage
- Add note to install about non-root on RHEL/CentOS
- Update imagebuild depdency to support heading ARGs in Dockerfile
- rootless: do not specify --rootless to the OCI runtime
- Export resolvesymlink function
- Exclude --force-rm from common bud cli flags
- run: bind mount /etc/hosts and /etc/resolv.conf if not in a volume
- rootless: use slirp4netns to setup the network namespace
- Instructions for completing the pull command
- Fix travis to not run environment variable patch
- rootless: only discard network configuration names
- run: only set up /etc/hosts or /etc/resolv.conf with network
- common: getFormat: match entire string not only the prefix
- vendor: update libpod
- Change validation EPOCH
- Fixing broken link for container-registries.conf
- Restore rootless isolation test for from volume ro test
- ostree: fix tag for build constraint
- Handle directories better in bud -f
- vndr in latest containers/storage
- Fix unshare gofmt issue
- runSetupBuiltinVolumes(): break up volume setup
- common: support a per-user registries conf file
- unshare: do not override the configuration
- common: honor the rootless configuration file
- unshare: create a new mount namespace
- unshare: support libpod rootless pkg
- Use libpod GetDefaultStorage to report proper storage config
- Allow container storage to manage the SELinux labels
- Resolve image names with default transport in from command
- run: When the value of isolation is set, use the set value instead of the default value.
- Vendor in latest containers/storage and opencontainers/selinux
- Remove no longer valid todo
- Check for empty buildTime in version
- Change gofmt so it runs on all but 1.10
- Run gofmt only on Go 1.11
- Walk symlinks when checking cached images for copied/added files
- ReserveSELinuxLabels(): handle wrapped errors from OpenBuilder
- Set WorkingDir to empty, not / for conformance
- Update calls in e2e to addres 1101
- imagebuilder.BuildDockerfiles: return the image ID
- Update for changes in the containers/image API
- bump(github.com/containers/image)
- Allow setting --no-pivot default with an env var
- Add man page and bash completion, for --no-pivot
- Add the --no-pivot flag to the run command
- Improve reporting about individual pull failures
- Move the "short name but no search registries" error handling to resolveImage
- Return a "search registries were needed but empty" indication in util.ResolveName
- Simplify handling of the "tried to pull an image but found nothing" case in newBuilder
- Don't even invoke the pull loop if options.FromImage == ""
- Eliminate the long-running ref and img variables in resolveImage
- In resolveImage, return immediately on success
- Fix From As in Dockerfile
- Vendor latest containers/image
- Vendor in latest libpod
- Sort CLI flags of buildah bud
- Change from testing with golang 1.9 to 1.11.
- unshare: detect when unprivileged userns are disabled
- Optimize redundant code
- fix missing format param
- chroot: fix the args check
- imagebuildah: make ResolveSymLink public
- Update copy chown test
- buildah: use the same logic for XDG_RUNTIME_DIR as podman
- V1.4 Release Announcement
- Podman  --privileged selinux is broken
- papr: mount source at gopath
- parse: Modify the return value
- parse: modify the verification of the isolation value
- Make sure we log or return every error
- pullImage(): when completing an image name, try docker://
- Fix up Tutorial 3 to account for format
- Vendor in latest containers/storage and containers/image
- docs/tutorials/01-intro.md: enhanced installation instructions
- Enforce "blocked" for registries for the "docker" transport
- Correctly set DockerInsecureSkipTLSVerify when pulling images
- chroot: set up seccomp and capabilities after supplemental groups
- chroot: fix capabilities list setup and application
- .papr.yml: log the podman version
- namespaces.bats: fix handling of uidmap/gidmap options in pairs
- chroot: only create user namespaces when we know we need them
- Check /proc/sys/user/max_user_namespaces on unshare(NEWUSERNS)
- bash/buildah: add isolation option to the from command

* Tue Oct 2 2018 Dan Walsh <dwalsh@redhat.com> 1.4-1
- from: fix isolation option
- Touchup pull manpage
- Export buildah ReserveSELinuxLables so podman can use it
- Add buildah.io to README.md and doc fixes
- Update rmi man for prune changes
- Ignore file not found removal error in bud
- bump(github.com/containers/{storage,image})
- NewImageSource(): only create one Diff() at a time
- Copy ExposedPorts from base image into the config
- tests: run conformance test suite in Travis
- Change rmi --prune to not accept an imageID
- Clear intermediate container IDs after each stage
- Request podman version for build issues
- unshare: keep the additional groups of the user
- Builtin volumes should be owned by the UID/GID of the container
- Get rid of dangling whitespace in markdown files
- Move buildah from projecatatomic/buildah to containers/buildah
- nitpick: parse.validateFlags loop in bud cli
- bash: Completion options
- Add signature policy to push tests
- vendor in latest containers/image
- Fix grammar in Container Tools Guide
- Don't build btrfs if it is not installed
- new: Return image-pulling errors from resolveImage
- pull: Return image-pulling errors from pullImage
- Add more volume mount tests
- chroot: create missing parent directories for volume mounts
- Push: Allow an empty destination
- Add Podman relationship to readme, create container tools guide
- Fix arg usage in buildah-tag
- Add flags/arguments order verification to other commands
- Handle ErrDuplicateName errors from store.CreateContainer()
- Evaluate symbolic links on Add/Copy Commands
- Vendor in latest containers/image and containers/storage
- Retain bounding set when running containers as non root
- run container-diff tests in Travis
- buildah-images.md: Fix option contents
- push: show image digest after push succeed
- Vendor in latest containers/storage,image,libpod and runc
- Change references to cri-o to point at new repository
- Exclude --layers from the common bug cli flags
- demos: Increase the executable permissions
- run: clear default seccomp filter if not enabled
- Bump maximum cyclomatic complexity to 45
- stdin: on HUP, read everything
- nitpick: use tabs in tests/helpers.bash
- Add flags/arguments order verification to one arg commands
- nitpick: decrease cognitive complexity in buildah-bud
- rename: Avoid renaming the same name as other containers
- chroot isolation: chroot() before setting up seccomp
- Small nitpick at the "if" condition in tag.go
- cmd/images: Modify json option
- cmd/images: Disallow the input of image when using the -a option
- Fix examples to include context directory
- Update containers/image to fix commit layer issue
- cmd/containers: End loop early when using the json option
- Make buildah-from error message clear when flags are after arg
- Touch up README.md for conformance tests
- Update container/storage for lock fix
- cmd/rm: restore the correct containerID display
- Remove debug lines
- Remove docker build image after each test
- Add README for conformance test
- Update the MakeOptions to accept all command options for buildah
- Update regrex to fit the docker output in test "run with JSON"
- cmd/buildah: Remove redundant variable declarations
- Warn about using Commands in Dockerfile that are not supported by OCI.
- Add buildah bud conformance test
- Fix rename to also change container name in builder
- Makefile: use $(GO) env-var everywhere
- Cleanup code to more closely match Docker Build images
- Document BUILDAH_* environment variables in buildah bud --help output
- Return error immediately if error occurs in Prepare step
- Fix --layers ADD from url issue
- Add "Sign your PRs" TOC item to contributing.md.
- Display the correct ID after deleting image
- rmi: Modify the handling of errors
- Let util.ResolveName() return parsing errors
- Explain Open Container Initiative (OCI) acronym, add link
- Update vendor for urfave/cli back to master
- Handle COPY --chown in Dockerfile
- Switch to Recommends container-selinux
- Update vendor for containernetworking, imagebuildah and podman
- Document STORAGE_DRIVER and STORAGE_OPTS environment variable
- Change references to projectatomic/libpod to containers/libpod
- Add container PATH retrieval example
- Expand variables names for --env
- imagebuildah: provide a way to provide stdin for RUN
- Remove an unused srcRef.NewImageSource in pullImage
- chroot: correct a comment
- chroot: bind mount an empty directory for masking
- Don't bother with --no-pivot for rootless isolation
- CentOS need EPEL repo
- Export a Pull() function
- Remove stream options, since docker build does not have it
- release v1.3: mention openSUSE
- Add Release Announcements directory

* Sat Aug 4 2018 Dan Walsh <dwalsh@redhat.com> 1.3-1
- Revert pull error handling from 881
- bud should not search context directory for Dockerfile
- Set BUILDAH_ISOLATION=rootless when running unprivileged
- .papr.sh: Also test with BUILDAH_ISOLATION=rootless
- Skip certain tests when we're using "rootless" isolation
- .travis.yml: run integration tests with BUILDAH_ISOLATION=chroot
- Add and implement IsolationOCIRootless
- Add a value for IsolationOCIRootless
- Fix rmi to remove intermediate images associated with an image
- Return policy error on pull
- Update containers/image to 216acb1bcd2c1abef736ee322e17147ee2b7d76c
- Switch to github.com/containers/image/pkg/sysregistriesv2
- unshare: make adjusting the OOM score optional
- Add flags validation
- chroot: handle raising process limits
- chroot: make the resource limits name map module-global
- Remove rpm.bats, we need to run this manually
- Set the default ulimits to match Docker
- buildah: no args is out of bounds
- unshare: error message missed the pid
- preprocess ".in" suffixed Dockerfiles
- Fix the the in buildah-config man page
- Only test rpmbuild on latest fedora
- Add support for multiple Short options
- Update to latest urvave/cli
- Add additional SELinux tests
- Vendor in latest github.com/containers/{image;storage}
- Stop testing with golang 1.8
- Fix volume cache issue with buildah bud --layers
- Create buildah pull command
- Increase the deadline for gometalinter during 'make validate'
- .papr.sh: Also test with BUILDAH_ISOLATION=chroot
- .travis.yml: run integration tests with BUILDAH_ISOLATION=chroot
- Add a Dockerfile
- Set BUILDAH_ISOLATION=chroot when running unprivileged
- Add and implement IsolationChroot
- Update github.com/opencontainers/runc
- maybeReexecUsingUserNamespace: add a default for root
- Allow ping command without NET_RAW Capabilities
- rmi.storageImageID: fix Wrapf format warning
- Allow Dockerfile content to come from stdin
- Vendor latest container/storage to fix overlay mountopt
- userns: assign additional IDs sequentially
- Remove default dev/pts
- Add OnBuild test to baseline test
- tests/run.bats(volumes): use :z when SELinux is enabled
- Avoid a stall in runCollectOutput()
- Use manifest from container/image
- Vendor in latest containers/image and containers/storage
- add rename command
- Completion command
- Update CHANGELOG.md
- Update vendor for runc to fix 32 bit builds
- bash completion: remove shebang
- Update vendor for runc to fix 32 bit builds

* Sat Jul 14 2018 Dan Walsh <dwalsh@redhat.com> 1.2-1
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

* Sat Jun 9 2018 Dan Walsh <dwalsh@redhat.com> 1.1-1
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
- Update github.com/containers/libpod
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

* Wed Apr 4 2018 Dan Walsh <dwalsh@redhat.com> 0.16-1
- Add support for shell
- Vendor in latest containers/image
- docker-archive generates docker legacy compatible images
- Do not create $DiffID subdirectories for layers with no configs
- Ensure the layer IDs in legacy docker/tarfile metadata are unique
- docker-archive: repeated layers are symlinked in the tar file
- sysregistries: remove all trailing slashes
- Improve docker/* error messages
- Fix failure to make auth directory
- Create a new slice in Schema1.UpdateLayerInfos
- Drop unused storageImageDestination.{image,systemContext}
- Load a *storage.Image only once in storageImageSource
- Support gzip for docker-archive files
- Remove .tar extension from blob and config file names
- ostree, src: support copy of compressed layers
- ostree: re-pull layer if it misses uncompressed_digest|uncompressed_size
- image: fix docker schema v1 -> OCI conversion
- Add /etc/containers/certs.d as default certs directory
- Change image time to locale, add troubleshooting.md, add logo to other mds
- Allow --cmd parameter to have commands as values
- Document the mounts.conf file
- Fix man pages to format correctly
- buildah from now supports pulling images using the following transports:
- docker-archive, oci-archive, and dir.
- If the user overrides the storage driver, the options should be dropped
- Show Config/Manifest as JSON string in inspect when format is not set
- Adds feature to pull compressed docker-archive files

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
- This fixes a large SELinux bug.  
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
- Use credentials from kpod login for buildah

* Wed Nov 15 2017 Dan Walsh <dwalsh@redhat.com> 0.6-1
- Adds support for converting manifest types when using the dir transport
- Rework how we do UID resolution in images
- Bump github.com/vbatts/tar-split
- Set option.terminal appropriately in run

* Wed Nov 08 2017 Dan Walsh <dwalsh@redhat.com> 0.5-2
- Bump github.com/vbatts/tar-split
- Fixes CVE That could allow a container image to cause a DOS

* Tue Nov 07 2017 Dan Walsh <dwalsh@redhat.com> 0.5-1
- Add secrets patch to buildah
- Add proper SELinux labeling to buildah run
- Add tls-verify to bud command
- Make filtering by date use the image's date
- images: don't list unnamed images twice
- Fix timeout issue
- Add further tty verbiage to buildah run
- Make inspect try an image on failure if type not specified
- Add support for `buildah run --hostname`
- Tons of bug fixes and code cleanup

* Fri Sep 22 2017 Dan Walsh <dwalsh@redhat.com> 0.4-1.git9cbccf88c
- Add default transport to push if not provided
- Avoid trying to print a nil ImageReference
- Add authentication to commit and push
- Add information on buildah from man page on transports
- Remove --transport flag
- Run: do not complain about missing volume locations
- Add credentials to buildah from
- Remove export command
- Run(): create the right working directory
- Improve "from" behavior with unnamed references
- Avoid parsing image metadata for dates and layers
- Read the image's creation date from public API
- Bump containers/storage and containers/image
- Don't panic if an image's ID can't be parsed
- Turn on --enable-gc when running gometalinter
- rmi: handle truncated image IDs

* Tue Aug 15 2017 Josh Boyer <jwboyer@redhat.com> 0.3-5.gitb9b2a8a
- Build for s390x as well

* Wed Aug 02 2017 Fedora Release Engineering <releng@fedoraproject.org> 0.3-4.gitb9b2a8a
- Rebuilt for https://fedoraproject.org/wiki/Fedora_27_Binutils_Mass_Rebuild

* Wed Jul 26 2017 Fedora Release Engineering <releng@fedoraproject.org> 0.3-3.gitb9b2a8a
- Rebuilt for https://fedoraproject.org/wiki/Fedora_27_Mass_Rebuild

* Thu Jul 20 2017 Dan Walsh <dwalsh@redhat.com> 0.3-2.gitb9b2a8a7e
- Bump for inclusion of OCI 1.0 Runtime and Image Spec

* Tue Jul 18 2017 Dan Walsh <dwalsh@redhat.com> 0.2.0-1.gitac2aad6
- buildah run: Add support for -- ending options parsing 
- buildah Add/Copy support for glob syntax
- buildah commit: Add flag to remove containers on commit
- buildah push: Improve man page and help information
- buildah run: add a way to disable PTY allocation
- Buildah docs: clarify --runtime-flag of run command
- Update to match newer storage and image-spec APIs
- Update containers/storage and containers/image versions
- buildah export: add support
- buildah images: update commands
- buildah images: Add JSON output option
- buildah rmi: update commands
- buildah containers: Add JSON output option
- buildah version: add command
- buildah run: Handle run without an explicit command correctly
- Ensure volume points get created, and with perms
- buildah containers: Add a -a/--all option

* Wed Jun 14 2017 Dan Walsh <dwalsh@redhat.com> 0.1.0-2.git597d2ab9
- Release Candidate 1
- All features have now been implemented.

* Fri Apr 14 2017 Dan Walsh <dwalsh@redhat.com> 0.0.1-1.git7a0a5333
- First package for Fedora
