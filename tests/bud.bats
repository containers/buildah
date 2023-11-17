#!/usr/bin/env bats

load helpers

@test "bud with a path to a Dockerfile (-f) containing a non-directory entry" {
  run_buildah 125 build -f $BUDFILES/non-directory-in-path/non-directory/Dockerfile
  expect_output --substring "non-directory/Dockerfile: not a directory"
}

@test "bud stdio is usable pipes" {
  run_buildah build $BUDFILES/stdio
}

@test "bud: build manifest list and --add-compression zstd" {
  local contextdir=${TEST_SCRATCH_DIR}/bud/platform
  mkdir -p $contextdir

  cat > $contextdir/Dockerfile1 << _EOF
FROM alpine
_EOF

  start_registry
  run_buildah login --tls-verify=false --authfile ${TEST_SCRATCH_DIR}/test.auth --username testuser --password testpassword localhost:${REGISTRY_PORT}
  run_buildah build $WITH_POLICY_JSON -t image1 --platform linux/amd64 -f $contextdir/Dockerfile1
  run_buildah build $WITH_POLICY_JSON -t image2 --platform linux/arm64 -f $contextdir/Dockerfile1

  run_buildah manifest create foo
  run_buildah manifest add foo image1
  run_buildah manifest add foo image2

  run_buildah manifest push $WITH_POLICY_JSON --authfile ${TEST_SCRATCH_DIR}/test.auth --all --add-compression zstd --tls-verify=false foo docker://localhost:${REGISTRY_PORT}/list

  run_buildah manifest inspect --authfile ${TEST_SCRATCH_DIR}/test.auth --tls-verify=false localhost:${REGISTRY_PORT}/list
  list="$output"

  validate_instance_compression "0" "$list" "amd64" "gzip"
  validate_instance_compression "1" "$list" "arm64" "gzip"
  validate_instance_compression "2" "$list" "amd64" "zstd"
  validate_instance_compression "3" "$list" "arm64" "zstd"
}

@test "bud: build manifest list with --add-compression zstd, --compression and --force-compression" {
  local contextdir=${TEST_SCRATCH_DIR}/bud/platform
  mkdir -p $contextdir

  cat > $contextdir/Dockerfile1 << _EOF
FROM alpine
_EOF

  start_registry
  run_buildah login --tls-verify=false --authfile ${TEST_SCRATCH_DIR}/test.auth --username testuser --password testpassword localhost:${REGISTRY_PORT}
  run_buildah build $WITH_POLICY_JSON -t image1 --platform linux/amd64 -f $contextdir/Dockerfile1
  run_buildah build $WITH_POLICY_JSON -t image2 --platform linux/arm64 -f $contextdir/Dockerfile1

  run_buildah manifest create foo
  run_buildah manifest add foo image1
  run_buildah manifest add foo image2

  run_buildah manifest push $WITH_POLICY_JSON --authfile ${TEST_SCRATCH_DIR}/test.auth --all --add-compression zstd --tls-verify=false foo docker://localhost:${REGISTRY_PORT}/list

  run_buildah manifest inspect --authfile ${TEST_SCRATCH_DIR}/test.auth --tls-verify=false localhost:${REGISTRY_PORT}/list
  list="$output"

  validate_instance_compression "0" "$list" "amd64" "gzip"
  validate_instance_compression "1" "$list" "arm64" "gzip"
  validate_instance_compression "2" "$list" "amd64" "zstd"
  validate_instance_compression "3" "$list" "arm64" "zstd"

  # Pushing again should keep every thing intact if original compression is `gzip` and `--force-compression` is specified
  run_buildah manifest push $WITH_POLICY_JSON --authfile ${TEST_SCRATCH_DIR}/test.auth --all --add-compression zstd --compression-format gzip --force-compression --tls-verify=false foo docker://localhost:${REGISTRY_PORT}/list

  run_buildah manifest inspect --authfile ${TEST_SCRATCH_DIR}/test.auth --tls-verify=false localhost:${REGISTRY_PORT}/list
  list="$output"

  validate_instance_compression "0" "$list" "amd64" "gzip"
  validate_instance_compression "1" "$list" "arm64" "gzip"
  validate_instance_compression "2" "$list" "amd64" "zstd"
  validate_instance_compression "3" "$list" "arm64" "zstd"

  # Pushing again without --force-compression but with --compression-format should do the same thing
  run_buildah manifest push $WITH_POLICY_JSON --authfile ${TEST_SCRATCH_DIR}/test.auth --all --add-compression zstd --compression-format gzip --tls-verify=false foo docker://localhost:${REGISTRY_PORT}/list

  run_buildah manifest inspect --authfile ${TEST_SCRATCH_DIR}/test.auth --tls-verify=false localhost:${REGISTRY_PORT}/list
  list="$output"

  validate_instance_compression "0" "$list" "amd64" "gzip"
  validate_instance_compression "1" "$list" "arm64" "gzip"
  validate_instance_compression "2" "$list" "amd64" "zstd"
  validate_instance_compression "3" "$list" "arm64" "zstd"
}

@test "Multi-stage should not remove used base-image without --layers" {
  run_buildah build -t parent-one -f $BUDFILES/multi-stage-only-base/Containerfile1
  run_buildah build -t parent-two -f $BUDFILES/multi-stage-only-base/Containerfile2
  run_buildah build -t multi-stage -f $BUDFILES/multi-stage-only-base/Containerfile3
  run_buildah images -a
  expect_output --substring "parent-one" "parent one must not be removed"
}

@test "no layer should be created on scratch" {
  run_buildah build --layers --label "label1=value1" -t test -f $BUDFILES/from-scratch/Containerfile
  run_buildah inspect -f '{{len .Docker.RootFS.DiffIDs}}' test
  expect_output "0" "layer should not exist"
  run_buildah build --layers -t test -f $BUDFILES/from-scratch/Containerfile
  run_buildah inspect -f '{{len .Docker.RootFS.DiffIDs}}' test
  expect_output "0" "layer should not exist"
}

@test "bud: build push with --force-compression" {
  skip_if_no_podman
  local contextdir=${TEST_SCRATCH_DIR}/bud/platform
  mkdir -p $contextdir

  cat > $contextdir/Dockerfile1 << _EOF
FROM alpine
_EOF

  start_registry
  run_buildah login --tls-verify=false --authfile ${TEST_SCRATCH_DIR}/test.auth --username testuser --password testpassword localhost:${REGISTRY_PORT}
  run_buildah build $WITH_POLICY_JSON -t image1 --platform linux/amd64 -f $contextdir/Dockerfile1

  run_buildah push $WITH_POLICY_JSON --authfile ${TEST_SCRATCH_DIR}/test.auth --tls-verify=false --compression-format gzip image1 docker://localhost:${REGISTRY_PORT}/image
  run podman run --rm --mount type=bind,src=${TEST_SCRATCH_DIR}/test.auth,target=/test.auth,Z --net host quay.io/skopeo/stable inspect --authfile=/test.auth --tls-verify=false --raw docker://localhost:${REGISTRY_PORT}/image
  # layers should have no trace of zstd since push was with --compression-format gzip
  assert "$output" !~ "zstd" "zstd found in layers where push was with --compression-format gzip"
  run_buildah push $WITH_POLICY_JSON --authfile ${TEST_SCRATCH_DIR}/test.auth --tls-verify=false --compression-format zstd --force-compression=false image1 docker://localhost:${REGISTRY_PORT}/image
  run podman run --rm --mount type=bind,src=${TEST_SCRATCH_DIR}/test.auth,target=/test.auth,Z --net host quay.io/skopeo/stable inspect --authfile=/test.auth --tls-verify=false --raw docker://localhost:${REGISTRY_PORT}/image
  # layers should have no trace of zstd since push is --force-compression=false
  assert "$output" !~ "zstd" "zstd found even though push was without --force-compression"
  run_buildah push $WITH_POLICY_JSON --authfile ${TEST_SCRATCH_DIR}/test.auth --tls-verify=false --compression-format zstd image1 docker://localhost:${REGISTRY_PORT}/image
  run podman run --rm --mount type=bind,src=${TEST_SCRATCH_DIR}/test.auth,target=/test.auth,Z --net host quay.io/skopeo/stable inspect --authfile=/test.auth --tls-verify=false --raw docker://localhost:${REGISTRY_PORT}/image
  # layers should container `zstd`
  expect_output --substring "zstd" "layers must contain zstd compression"
  run_buildah push $WITH_POLICY_JSON --authfile ${TEST_SCRATCH_DIR}/test.auth --tls-verify=false --compression-format zstd --force-compression image1 docker://localhost:${REGISTRY_PORT}/image
  run podman run --rm --mount type=bind,src=${TEST_SCRATCH_DIR}/test.auth,target=/test.auth,Z --net host quay.io/skopeo/stable inspect --authfile=/test.auth --tls-verify=false --raw docker://localhost:${REGISTRY_PORT}/image
  # layers should container `zstd`
  expect_output --substring "zstd" "layers must contain zstd compression"
}

@test "bud with --dns* flags" {
  _prefetch alpine

  for dnsopt in --dns --dns-option --dns-search; do
    run_buildah 125 build $dnsopt=example.com --network=none $WITH_POLICY_JSON -f $BUDFILES/dns/Dockerfile  $BUDFILES/dns
    expect_output "Error: the $dnsopt option cannot be used with --network=none" "dns options should not be allowed with --network=none"
  done

  run_buildah build --dns-search=example.com --dns=223.5.5.5 --dns-option=use-vc  $WITH_POLICY_JSON -f $BUDFILES/dns/Dockerfile  $BUDFILES/dns
  expect_output --substring "search example.com"
  expect_output --substring "nameserver 223.5.5.5"
  expect_output --substring "options use-vc"
}

@test "build with inline RUN --network=host" {
  #hostns=$(readlink /proc/self/ns/net)
  run readlink /proc/self/ns/net
  hostns="$output"
  run_buildah build $WITH_POLICY_JSON -t source -f $BUDFILES/inline-network/Dockerfile1
  expect_output --from="${lines[8]}" "${hostns}"
}

@test "build with inline RUN --network=none" {
  run_buildah 1 build $WITH_POLICY_JSON -t source -f $BUDFILES/inline-network/Dockerfile2
  expect_output --substring "wget: bad address"
}

@test "build with inline RUN --network=fake" {
  run_buildah 125 build $WITH_POLICY_JSON -t source -f $BUDFILES/inline-network/Dockerfile3
  expect_output --substring "unsupported value"
}

@test "build with inline default RUN --network=default" {
  skip_if_chroot
  _prefetch alpine
  run readlink /proc/self/ns/net
  hostns=$output
  run_buildah build --network=host $WITH_POLICY_JSON -t source -f $BUDFILES/inline-network/Dockerfile4
  firstns=${lines[2]}
  assert "${hostns}" == "$firstns"
  run_buildah build --network=private $WITH_POLICY_JSON -t source -f $BUDFILES/inline-network/Dockerfile4
  secondns=${lines[2]}
  assert "$secondns" != "$firstns"
}


@test "bud with ignoresymlink on default file" {
	  cat > /tmp/private_file << _EOF
hello
_EOF

run_buildah build $WITH_POLICY_JSON -t test -f Dockerfile $BUDFILES/container-ignoresymlink
# Default file must not point to symlink so hello should not be ignored from build context
expect_output --substring "hello"

}

#Verify https://github.com/containers/buildah/issues/4342
@test "buildkit-mount type=cache should not hang if cache is wiped in between" {
  containerfile=$BUDFILES/cache-mount-locked/Containerfile
  run_buildah build $WITH_POLICY_JSON --build-arg WIPE_CACHE=1 -t source -f $containerfile $BUDFILES/cache-mount-locked
  # build should be success and must contain `hello` from `file` in last step
  expect_output --substring "hello"
}

# Test for https://github.com/containers/buildah/pull/4295
@test "build test warning for preconfigured TARGETARCH, TARGETOS, TARGETPLATFORM or TARGETVARIANT" {
  _prefetch alpine
  containerfile=$BUDFILES/platform-sets-args/Containerfile

  # Containerfile must contain one or more (four, as of 2022-10) lines
  # of the form 'ARG TARGETxxx' for each of the variables of interest.
  local -a checkvars=($(sed -ne 's/^ARG //p' <$containerfile))
  assert "${checkvars[*]}" != "" \
         "INTERNAL ERROR! No 'ARG xxx' lines in $containerfile!"

  # With explicit and full --platform, buildah should not warn.
  run_buildah build $WITH_POLICY_JSON --platform linux/amd64/v2 \
              -t source -f $containerfile
  assert "$output" !~ "missing .* build argument" \
         "With explicit --platform, buildah should not warn"

  # Likewise with individual args
  run_buildah build $WITH_POLICY_JSON --os linux --arch amd64 --variant v2 \
              -t source -f $containerfile
  assert "$output" !~ "missing .* build argument" \
         "With explicit --os + --arch + --variant, buildah should not warn"

  # FIXME FIXME FIXME: #4319: with --os only, buildah should not warn about OS
  if false; then
      run_buildah build $WITH_POLICY_JSON --os linux \
                  -t source -f $containerfile
      assert "$output" !~ "missing.*TARGETOS" \
             "With explicit --os (but no arch/variant), buildah should not warn about TARGETOS"
      # FIXME: add --arch test too, and maybe make this cleaner
  fi

}

@test "build-conflicting-isolation-chroot-and-network" {
  _prefetch alpine
  cat > ${TEST_SCRATCH_DIR}/Containerfile << _EOF
FROM alpine
RUN ping -c 1 4.2.2.2
_EOF

  run_buildah 125 build --network=none --isolation=chroot $WITH_POLICY_JSON ${TEST_SCRATCH_DIR}
  expect_output --substring "cannot set --network other than host with --isolation chroot"
}

@test "bud with .dockerignore #1" {
  _prefetch alpine busybox
  run_buildah 125 build -t testbud $WITH_POLICY_JSON -f $BUDFILES/dockerignore/Dockerfile $BUDFILES/dockerignore
  expect_output --substring 'building.*"COPY subdir \./".*no such file or directory'

  run_buildah build -t testbud $WITH_POLICY_JSON -f $BUDFILES/dockerignore/Dockerfile.succeed $BUDFILES/dockerignore

  run_buildah from --name myctr testbud

  run_buildah 1 run myctr ls -l test1.txt

  run_buildah run myctr ls -l test2.txt

  run_buildah 1 run myctr ls -l sub1.txt

  run_buildah 1 run myctr ls -l sub2.txt

  run_buildah 1 run myctr ls -l subdir/
}

@test "bud with .containerignore" {
  _prefetch alpine busybox
  run_buildah 125 build -t testbud $WITH_POLICY_JSON -f $BUDFILES/containerignore/Dockerfile $BUDFILES/containerignore
  expect_output --substring 'building.*"COPY subdir \./".*no such file or directory'

  run_buildah build -t testbud $WITH_POLICY_JSON -f $BUDFILES/containerignore/Dockerfile.succeed $BUDFILES/containerignore

  run_buildah from --name myctr testbud

  run_buildah 1 run myctr ls -l test1.txt

  run_buildah run myctr ls -l test2.txt

  run_buildah 1 run myctr ls -l sub1.txt

  run_buildah 1 run myctr ls -l sub2.txt

  run_buildah 1 run myctr ls -l subdir/
}

@test "bud with .dockerignore - unmatched" {
  # Here .dockerignore contains 'unmatched', which will not match anything.
  # Therefore everything in the subdirectory should be copied into the image.
  #
  # We need to do this from a tmpdir, not the original or distributed
  # bud subdir, because of rpm: as of 2020-04-01 rpmbuild 4.16 alpha
  # on rawhide no longer packages circular symlinks (rpm issue #1159).
  # We used to include these symlinks in git and the rpm; now we need to
  # set them up manually as part of test setup to be able to package tests.
  local contextdir=${TEST_SCRATCH_DIR}/dockerignore2
  cp -a $BUDFILES/dockerignore2 $contextdir

  # Create symlinks, including bad ones
  ln -sf subdir        $contextdir/symlink
  ln -sf circular-link $contextdir/subdir/circular-link
  ln -sf no-such-file  $contextdir/subdir/dangling-link

  # Build, create a container, mount it, and list all files therein
  run_buildah build -t testbud2 $WITH_POLICY_JSON $contextdir

  run_buildah from --pull=false testbud2
  cid=$output

  run_buildah mount $cid
  mnt=$output
  run find $mnt -printf "%P(%l)\n"
  filelist=$(LC_ALL=C sort <<<"$output")
  run_buildah umount $cid

  # Format is: filename, and, in parentheses, symlink target (usually empty)
  # The list below has been painstakingly crafted; please be careful if
  # you need to touch it (e.g. if you add new files/symlinks)
  expect="()
.dockerignore()
Dockerfile()
subdir()
subdir/circular-link(circular-link)
subdir/dangling-link(no-such-file)
subdir/sub1.txt()
subdir/subsubdir()
subdir/subsubdir/subsub1.txt()
symlink(subdir)"

  # If this test ever fails, the 'expect' message will be almost impossible
  # for humans to read -- sorry, I never implemented multi-line comparisons.
  # Should this ever happen, uncomment these two lines and run tests in
  # your own vm; then diff the two files.
  #echo "$filelist" >${TMPDIR}/filelist.actual
  #echo "$expect"   >${TMPDIR}/filelist.expect

  expect_output --from="$filelist" "$expect" "container file list"
}

@test "bud with .dockerignore #2" {
  run_buildah 125 build -t testbud3 $WITH_POLICY_JSON $BUDFILES/dockerignore3
  expect_output --substring 'building.*"COPY test1.txt /upload/test1.txt".*no such file or directory'
  expect_output --substring $(realpath "$BUDFILES/dockerignore3/.dockerignore")
}

@test "bud with .dockerignore #4" {
  run_buildah 125 build -t testbud3 $WITH_POLICY_JSON -f Dockerfile.test $BUDFILES/dockerignore4
  expect_output --substring 'building.*"COPY test1.txt /upload/test1.txt".*no such file or directory'
  expect_output --substring '1 filtered out using /[^ ]*/Dockerfile.test.dockerignore'
}

@test "bud with .dockerignore #6" {
  _prefetch alpine busybox
  run_buildah 125 build -t testbud $WITH_POLICY_JSON -f $BUDFILES/dockerignore6/Dockerfile $BUDFILES/dockerignore6
  expect_output --substring 'building.*"COPY subdir \./".*no such file or directory'

  run_buildah build -t testbud $WITH_POLICY_JSON -f $BUDFILES/dockerignore6/Dockerfile.succeed $BUDFILES/dockerignore6

  run_buildah from --name myctr testbud

  run_buildah 1 run myctr ls -l test1.txt

  run_buildah run myctr ls -l test2.txt

  run_buildah 1 run myctr ls -l sub1.txt

  run_buildah 1 run myctr ls -l sub2.txt

  run_buildah 1 run myctr ls -l subdir/
}

@test "build with --platform without OS" {
  run_buildah info --format '{{.host.arch}}'
  myarch="$output"

  run_buildah build --platform $myarch $WITH_POLICY_JSON -t test -f $BUDFILES/base-with-arg/Containerfile
  expect_output --substring "This is built for $myarch"

  ## podman-remote binding has a bug where is sends `--platform as /`
  run_buildah build --platform "/" $WITH_POLICY_JSON -t test -f $BUDFILES/base-with-arg/Containerfile
  expect_output --substring "This is built for $myarch"
}

@test "build with basename resolving default arg" {
  run_buildah info --format '{{.host.arch}}'
  myarch="$output"
  run_buildah info --format '{{.host.variant}}'
  myvariant="$output"

  run_buildah build --platform linux/$myarch/$myvariant $WITH_POLICY_JSON -t test -f $BUDFILES/base-with-arg/Containerfile
  expect_output --substring "This is built for $myarch"
}

@test "build with basename resolving user arg" {
  run_buildah build --build-arg CUSTOM_TARGET=first $WITH_POLICY_JSON -t test -f $BUDFILES/base-with-arg/Containerfile2
  expect_output --substring "This is built for first"
  run_buildah build --build-arg CUSTOM_TARGET=second $WITH_POLICY_JSON -t test -f $BUDFILES/base-with-arg/Containerfile2
  expect_output --substring "This is built for second"
}

@test "build with basename resolving user arg from file" {
  run_buildah build \
	--build-arg-file $BUDFILES/base-with-arg/first.args \
	$WITH_POLICY_JSON -t test -f $BUDFILES/base-with-arg/Containerfile2
  expect_output --substring "This is built for first"

  run_buildah build \
	--build-arg-file $BUDFILES/base-with-arg/second.args \
	$WITH_POLICY_JSON -t test -f $BUDFILES/base-with-arg/Containerfile2
  expect_output --substring "This is built for second"
}

@test "build with basename resolving user arg from latest file in arg list" {
  run_buildah build \
	--build-arg-file $BUDFILES/base-with-arg/second.args \
	--build-arg-file $BUDFILES/base-with-arg/first.args \
	$WITH_POLICY_JSON -t test -f $BUDFILES/base-with-arg/Containerfile2
  expect_output --substring "This is built for first"
}

@test "build with basename resolving user arg from in arg list" {
  run_buildah build \
	--build-arg-file $BUDFILES/base-with-arg/second.args \
	--build-arg CUSTOM_TARGET=first \
	$WITH_POLICY_JSON -t test -f $BUDFILES/base-with-arg/Containerfile2
  expect_output --substring "This is built for first"
}

# Following test should fail since we are trying to use build-arg which
# was not declared. Honors discussion here: https://github.com/containers/buildah/pull/4061/commits/1237c04d6ae0ee1f027a1f02bf3ab5c57ac7d9b6#r906188374
@test "build with basename resolving user arg - should fail" {
  run_buildah 125 build --build-arg CUSTOM_TARGET=first $WITH_POLICY_JSON -t test -f $BUDFILES/base-with-arg/Containerfilebad
  expect_output --substring "invalid reference format"
}

# Try building with arch and variant
# Issue: https://github.com/containers/buildah/issues/4276
@test "build-with-inline-platform-and-variant" {
  local contextdir=${TEST_SCRATCH_DIR}/bud/platform
  mkdir -p $contextdir
  cat > $contextdir/Dockerfile << _EOF
FROM --platform=freebsd/arm64/v8 scratch
COPY . .
_EOF

  run_buildah build $WITH_POLICY_JSON -t test $contextdir
  run_buildah inspect --format '{{ .OCIv1.Architecture }}' test
  expect_output --substring "arm64"
  run_buildah inspect --format '{{ .OCIv1.Variant }}' test
  expect_output --substring "v8"
}

# Following test must fail since we are trying to run linux/arm64 on linux/amd64
# Issue: https://github.com/containers/buildah/issues/3712
@test "build-with-inline-platform" {
  # Host arch
  local contextdir=${TEST_SCRATCH_DIR}/bud/platform
  mkdir -p $contextdir
  run_buildah info --format '{{.host.arch}}'
  myarch="$output"
  otherarch="arm64"

  # just make sure that other arch is not equivalent to host arch
  if [[ "$otherarch" == "$myarch" ]]; then
    otherarch="amd64"
  fi
  # ...create a Containerfile with --platform=linux/$otherarch
  cat > $contextdir/Dockerfile << _EOF
FROM --platform=linux/${otherarch} alpine
RUN uname -m
_EOF

  run_buildah '?' build $WITH_POLICY_JSON -t test $contextdir
  if [[ $status -eq 0 ]]; then
    run_buildah inspect --format '{{ .OCIv1.Architecture }}' test
    expect_output --substring "$otherarch"
  else
    # Build failed: we DO NOT have qemu-user-static installed.
    expect_output --substring "format error"
  fi
}

@test "build-with-inline-platform-and-rely-on-defaultbuiltinargs" {
  # Get host arch
  run_buildah info --format '{{.host.arch}}'
  myarch="$output"
  otherarch="arm64"
  # just make sure that other arch is not equivalent to host arch
  if [[ "$otherarch" == "$myarch" ]]; then
    otherarch="amd64"
  fi

  run_buildah build --platform linux/$otherarch $WITH_POLICY_JSON -t test -f $BUDFILES/multiarch/Dockerfile.built-in-args
  expect_output --substring "I'm compiling for linux/$otherarch"
  expect_output --substring "and tagging for linux/$otherarch"
  expect_output --substring "and OS linux"
  expect_output --substring "and ARCH $otherarch"
  run_buildah inspect --format '{{ .OCIv1.Architecture }}' test
  expect_output --substring "$otherarch"
}

# Buildkit parity: this verifies if we honor custom overrides of TARGETOS, TARGETVARIANT, TARGETARCH and TARGETPLATFORM if user wants
@test "build-with-inline-platform-and-rely-on-defaultbuiltinargs-check-custom-override" {
  run_buildah build --platform linux/arm64 $WITH_POLICY_JSON --build-arg TARGETOS=android -t test -f $BUDFILES/multiarch/Dockerfile.built-in-args
  expect_output --substring "I'm compiling for linux/arm64"
  expect_output --substring "and tagging for linux/arm64"
  ## Note since we used --build-arg and overrode OS, OS must be android
  expect_output --substring "and OS android"
  expect_output --substring "and ARCH $otherarch"
  run_buildah inspect --format '{{ .OCIv1.Architecture }}' test
  expect_output --substring "$otherarch"
}

# Following test must pass since we want to tag image as host arch
# Test for use-case described here: https://github.com/containers/buildah/issues/3261
@test "build-with-inline-platform-amd-but-tag-as-arm" {
  # Host arch
  local contextdir=${TEST_SCRATCH_DIR}/bud/platform
  mkdir -p $contextdir
  run_buildah info --format '{{.host.arch}}'
  myarch="$output"
  targetarch="arm64"

  if [[ "$targetArch" == "$myarch" ]]; then
    targetarch="amd64"
  fi

  cat > $contextdir/Dockerfile << _EOF
FROM --platform=linux/${myarch} alpine
RUN uname -m
_EOF

  # Tries building image where baseImage has --platform=linux/HostArch
  run_buildah build --platform linux/${targetarch} $WITH_POLICY_JSON -t test $contextdir
  run_buildah inspect --format '{{ .OCIv1.Architecture }}' test
  # base image is pulled as HostArch but tagged as non host arch
  expect_output --substring $targetarch
}

# Test build with --add-history=false
@test "build-with-omit-history-to-true should not add history" {
  local contextdir=${TEST_SCRATCH_DIR}/bud/platform
  mkdir -p $contextdir

  cat > $contextdir/Dockerfile1 << _EOF
FROM alpine
RUN echo hello
RUN echo world
_EOF

  # Built image must not contain history for the layers which we have just built.
  run_buildah build $WITH_POLICY_JSON --omit-history -t source -f $contextdir/Dockerfile1
  run_buildah inspect --format "{{index .Docker.History}}" source
  expect_output "[]"
  run_buildah inspect --format "{{index .OCIv1.History}}" source
  expect_output "[]"
  run_buildah inspect --format "{{index .History}}" source
  expect_output "[]"
}

# Test building with --userns=auto
@test "build with --userns=auto also with size" {
  local contextdir=${TEST_SCRATCH_DIR}/bud/platform
  mkdir -p $contextdir
  user=$USER

  if [[ "$user" == "root" ]]; then
    user="containers"
  fi

  if ! grep -q $user "/etc/subuid"; then
    skip "cannot find mappings for the current user"
  fi

  cat > $contextdir/Dockerfile << _EOF
FROM alpine
RUN cat /proc/self/uid_map
RUN echo hello

FROM alpine
COPY --from=0 /tmp /tmp
RUN cat /proc/self/uid_map
RUN ls -a
_EOF

  run_buildah build --userns=auto $WITH_POLICY_JSON -t source -f $contextdir/Dockerfile
  expect_output --substring "1024"
  run_buildah build --userns=auto:size=500 $WITH_POLICY_JSON -t source -f $contextdir/Dockerfile
  expect_output --substring "500"
}

# Test building with --userns=auto with uidmapping
@test "build with --userns=auto with uidmapping" {
  local contextdir=${TEST_SCRATCH_DIR}/bud/platform
  mkdir -p $contextdir
  user=$USER

  if [[ "$user" == "root" ]]; then
    user="containers"
  fi

  if ! grep -q $user "/etc/subuid"; then
    skip "cannot find mappings for the current user"
  fi

  cat > $contextdir/Dockerfile << _EOF
FROM alpine
RUN cat /proc/self/uid_map
_EOF

  run_buildah build --userns=auto:size=8192,uidmapping=0:0:1 $WITH_POLICY_JSON -t source -f $contextdir/Dockerfile
  expect_output --substring "8191"
  run_buildah build --userns=auto:uidmapping=0:0:1 $WITH_POLICY_JSON -t source -f $contextdir/Dockerfile
  expect_output --substring "         0          0          1"
}

# Test building with --userns=auto with gidmapping
@test "build with --userns=auto with gidmapping" {
  local contextdir=${TEST_SCRATCH_DIR}/bud/platform
  mkdir -p $contextdir
  user=$USER

  if [[ "$user" == "root" ]]; then
    user="containers"
  fi

  if ! grep -q $user "/etc/subuid"; then
    skip "cannot find mappings for the current user"
  fi

  cat > $contextdir/Dockerfile << _EOF
FROM alpine
RUN cat /proc/self/gid_map
_EOF

  run_buildah build --userns=auto:size=8192,gidmapping=0:0:1 $WITH_POLICY_JSON -t source -f $contextdir/Dockerfile
  expect_output --substring "8191"
  run_buildah build --userns=auto:gidmapping=0:0:1 $WITH_POLICY_JSON -t source -f $contextdir/Dockerfile
  expect_output --substring "         0          0          1"
}

# Test bud with prestart hook
@test "build-test with OCI prestart hook" {
  skip_if_in_container # This works in privileged container setup but does not works in CI setup
  local contextdir=${TEST_SCRATCH_DIR}/bud/platform
  mkdir -p $contextdir/hooks

  cat > $contextdir/Dockerfile << _EOF
FROM alpine
RUN echo hello
_EOF

  cat > $contextdir/hooks/test.json << _EOF
{
  "version": "1.0.0",
  "hook": {
    "path": "$contextdir/hooks/test"
  },
  "when": {
    "always": true
  },
  "stages": ["prestart"]
}
_EOF

  cat > $contextdir/hooks/test << _EOF
#!/bin/sh
echo from-hook > $contextdir/hooks/hook-output
_EOF

  # make actual hook executable
  chmod +x $contextdir/hooks/test
  run_buildah build $WITH_POLICY_JSON -t source --hooks-dir=$contextdir/hooks -f $contextdir/Dockerfile
  run cat $contextdir/hooks/hook-output
  expect_output --substring "from-hook"
}

@test "build with add resolving to invalid HTTP status code" {
  local contextdir=${TEST_SCRATCH_DIR}/bud/platform
  mkdir -p $contextdir

  cat > $contextdir/Dockerfile << _EOF
FROM alpine
ADD https://google.com/test /
_EOF

  run_buildah 125 build $WITH_POLICY_JSON -t source -f $contextdir/Dockerfile
  expect_output --substring "invalid response status"
}

@test "build test has gid in supplemental groups" {
  _prefetch alpine
  run_buildah build $WITH_POLICY_JSON -t source -f $BUDFILES/supplemental-groups/Dockerfile
  # gid 1000 must be in supplemental groups
  expect_output --substring "Groups:	1000"
}

@test "build test if supplemental groups has gid with --isolation chroot" {
  test -z "${BUILDAH_ISOLATION}" || skip "BUILDAH_ISOLATION=${BUILDAH_ISOLATION} overrides --isolation"

  _prefetch alpine
  run_buildah build --isolation chroot $WITH_POLICY_JSON -t source -f $BUDFILES/supplemental-groups/Dockerfile
  # gid 1000 must be in supplemental groups
  expect_output --substring "Groups:	1000"
}

@test "build-test --mount=type=secret test relative to workdir mount" {
  local contextdir=$BUDFILES/secret-relative
  run_buildah build $WITH_POLICY_JSON --no-cache --secret id=secret-foo,src=$contextdir/secret1.txt --secret id=secret-bar,src=$contextdir/secret2.txt -t test -f $contextdir/Dockerfile
  expect_output --substring "secret:foo"
  expect_output --substring "secret:bar"
}

@test "build-test --mount=type=cache test relative to workdir mount" {
  local contextdir=${TEST_SCRATCH_DIR}/bud/platform
  mkdir -p $contextdir
  ## write-cache
  cat > $contextdir/Dockerfile << _EOF
FROM alpine
RUN mkdir test
WORKDIR test
RUN --mount=type=cache,id=YfHI60aApFM-target,target=target echo world > /test/target/hello
_EOF

  run_buildah build $WITH_POLICY_JSON -t source -f $contextdir/Dockerfile

  cat > $contextdir/Dockerfile << _EOF
FROM alpine
RUN mkdir test
WORKDIR test
RUN --mount=type=cache,id=YfHI60aApFM-target,target=target cat /test/target/hello
_EOF

  run_buildah build $WITH_POLICY_JSON -t source -f $contextdir/Dockerfile
  expect_output --substring "world"
}

@test "build-test do not use mount stage from cache if it was rebuilt" {
  local contextdir=${TEST_SCRATCH_DIR}/bud/platform
  mkdir -p $contextdir

  cat > $contextdir/Dockerfile << _EOF
FROM alpine as dependencies

RUN mkdir /build && echo v1 > /build/version

FROM alpine

RUN --mount=type=bind,source=/build,target=/build,from=dependencies \
    cp /build/version /version

RUN cat /version
_EOF

  run_buildah build $WITH_POLICY_JSON --layers -t source -f $contextdir/Dockerfile
  run_buildah build $WITH_POLICY_JSON --layers -t source2 -f $contextdir/Dockerfile
  expect_output --substring "Using cache"

  # First stage i.e dependencies is changed so it should not use the steps in second stage from
  # cache
  cat > $contextdir/Dockerfile << _EOF
FROM alpine as dependencies

RUN mkdir /build && echo v2 > /build/version

FROM alpine

RUN --mount=type=bind,source=/build,target=/build,from=dependencies \
    cp /build/version /version

RUN cat /version
_EOF

  run_buildah build $WITH_POLICY_JSON --layers -t source3 -f $contextdir/Dockerfile
  assert "$output" !~ "Using cache"

}

# Verify: https://github.com/containers/buildah/issues/4572
@test "build-test verify no dangling containers are left" {
  local contextdir=${TEST_SCRATCH_DIR}/bud/platform
  mkdir -p $contextdir

  cat > $contextdir/Dockerfile << _EOF
FROM alpine AS alpine_builder
FROM busybox AS busybox_builder
FROM scratch
COPY --from=alpine_builder /etc/alpine* .
COPY --from=busybox_builder /bin/busybox /bin/busybox
_EOF

  run_buildah build $WITH_POLICY_JSON -t source -f $contextdir/Dockerfile
  # No leftover containers, just the header line.
  run_buildah containers
  expect_line_count 1
}

# Verify: https://github.com/containers/buildah/issues/4485
# Verify: https://github.com/containers/buildah/issues/4319
@test "No default warning for TARGETARCH, TARGETOS, TARGETPLATFORM " {
  local contextdir=$BUDFILES/targetarch

  run_buildah build $WITH_POLICY_JSON --platform=linux/amd64,linux/arm64 -f $contextdir/Dockerfile
  assert "$output" !~ "one or more build args were not consumed" \
	 "No warning for default args should be there"

  run_buildah build $WITH_POLICY_JSON --os linux -f $contextdir/Dockerfile
  assert "$output" !~ "Try adding" \
	"No Warning for default args should be there"
}


@test "build-test skipping unwanted stages with --skip-unused-stages=false and --skip-unused-stages=true" {
  local contextdir=${TEST_SCRATCH_DIR}/bud/platform
  mkdir -p $contextdir

  cat > $contextdir/Dockerfile << _EOF
FROM alpine
RUN echo "first unwanted stage"

FROM alpine as one
RUN echo "needed stage"

FROM alpine
RUN echo "another unwanted stage"

FROM one
RUN echo "target stage"
_EOF

  # with --skip-unused-stages=false
  run_buildah build $WITH_POLICY_JSON --skip-unused-stages=false -t source -f $contextdir/Dockerfile
  expect_output --substring "needed stage"
  expect_output --substring "target stage"
  # this is expected since user specified `--skip-unused-stages=false`
  expect_output --substring "first unwanted stage"
  expect_output --substring "another unwanted stage"

  # with --skip-unused-stages=true
  run_buildah build $WITH_POLICY_JSON --skip-unused-stages=true -t source -f $contextdir/Dockerfile
  expect_output --substring "needed stage"
  expect_output --substring "target stage"
  assert "$output" !~ "unwanted stage"
}

@test "build-test: do not warn for instructions declared in unused stages" {
  local contextdir=${TEST_SCRATCH_DIR}/bud/platform
  mkdir -p $contextdir

  cat > $contextdir/Dockerfile << _EOF
FROM alpine
RUN echo "first unwanted stage"

FROM alpine as one
RUN echo "needed stage"

FROM alpine
ARG FOO_BAR
RUN echo "another unwanted stage"

FROM one
RUN echo "target stage"
_EOF

  # with --skip-unused-stages=true no warning should be printed since ARG is decalred in stage which is not used
  run_buildah build $WITH_POLICY_JSON --skip-unused-stages=true -t source -f $contextdir/Dockerfile
  expect_output --substring "needed stage"
  expect_output --substring "target stage"
  assert "$output" !~ "unwanted stage"
  # must not contain warning "missing FOO_BAR"
  assert "$output" !~ "missing"

  # with --skip-unused-stages=false should print unwanted stage as well as warning for unused arg
  run_buildah build $WITH_POLICY_JSON --skip-unused-stages=false -t source -f $contextdir/Dockerfile
  expect_output --substring "needed stage"
  expect_output --substring "target stage"
  expect_output --substring "unwanted stage"
  expect_output --substring "missing"
}

# Test skipping images with FROM
@test "build-test skipping unwanted stages with FROM" {
  local contextdir=${TEST_SCRATCH_DIR}/bud/platform
  mkdir -p $contextdir

  cat > $contextdir/Dockerfile << _EOF
FROM alpine
RUN echo "unwanted stage"

FROM alpine as one
RUN echo "needed stage"

FROM alpine
RUN echo "another unwanted stage"

FROM one
RUN echo "target stage"
_EOF

  run_buildah build $WITH_POLICY_JSON -t source -f $contextdir/Dockerfile
  expect_output --substring "needed stage"
  expect_output --substring "target stage"
  assert "$output" !~ "unwanted stage"
}

# Note: Please skip this tests in case of podman-remote build
@test "build: test race in updating image name while performing parallel commits" {
  _prefetch alpine
  # Run 25 parallel builds using the same Containerfile
  local count=25
  for i in $(seq --format '%02g' 1 $count); do
      timeout --foreground -v --kill=10 300 \
              ${BUILDAH_BINARY} ${BUILDAH_REGISTRY_OPTS} ${ROOTDIR_OPTS} $WITH_POLICY_JSON build --quiet --squash --iidfile ${TEST_SCRATCH_DIR}/id.$i --timestamp 0 -f $BUDFILES/check-race/Containerfile >/dev/null &
  done
  # Wait for all background builds to complete. Note that this succeeds
  # even if some of the individual builds fail! Our actual test is below.
  wait
  # Number of output bytes must be always same, which confirms that there is no race.
  assert "$(cat ${TEST_SCRATCH_DIR}/id.* | wc -c)" = 1775 "Total chars in all id.* files"
  # clean all images built for this test
  run_buildah rmi --all -f
}

# Test skipping images with FROM but stage name also conflicts with additional build context
# so selected stage should be still skipped since it is not being actually used by additional build
# context is being used.
@test "build-test skipping unwanted stages with FROM and conflict with additional build context" {
  local contextdir=${TEST_SCRATCH_DIR}/bud/platform
  mkdir -p $contextdir
  # add file on original context
  echo something > $contextdir/somefile

  cat > $contextdir/Dockerfile << _EOF
FROM alpine
RUN echo "unwanted stage"

FROM alpine as one
RUN echo "unwanted stage"
RUN echo "from stage unwanted stage"

FROM alpine
RUN echo "another unwanted stage"

FROM alpine
COPY --from=one somefile .
RUN cat somefile
_EOF

  run_buildah build $WITH_POLICY_JSON --build-context one=$contextdir -t source -f $contextdir/Dockerfile
  expect_output --substring "something"
  assert "$output" !~ "unwanted stage"
}

# Test skipping unwanted stage with COPY from stage name
@test "build-test skipping unwanted stages with COPY from stage name" {
  local contextdir=${TEST_SCRATCH_DIR}/bud/platform
  mkdir -p $contextdir

  echo something > $contextdir/somefile
  cat > $contextdir/Dockerfile << _EOF
FROM alpine
RUN echo "unwanted stage"

FROM alpine as one
RUN echo "needed stage"
COPY somefile file

FROM alpine
COPY --from=one file .
RUN cat file
RUN echo "target stage"
_EOF

  run_buildah build $WITH_POLICY_JSON -t source -f $contextdir/Dockerfile $contextdir
  expect_output --substring "needed stage"
  expect_output --substring "something"
  expect_output --substring "target stage"
  assert "$output" !~ "unwanted stage"
}

@test "build test --retry and --retry-delay" {
  local contextdir=${TEST_SCRATCH_DIR}/bud/platform
  mkdir -p $contextdir

  echo something > $contextdir/somefile
  cat > $contextdir/Dockerfile << _EOF
FROM alpine
RUN echo hello
_EOF

  run_buildah --log-level debug build --retry 4 --retry-delay 5s $WITH_POLICY_JSON --layers -t source -f $contextdir/Dockerfile $contextdir
  expect_output --substring "Setting MaxPullPushRetries to 4 and PullPushRetryDelay to 5s"
}

# Test skipping unwanted stage with COPY from stage index
@test "build-test skipping unwanted stages with COPY from stage index" {
  local contextdir=${TEST_SCRATCH_DIR}/bud/platform
  mkdir -p $contextdir

  echo something > $contextdir/somefile
  cat > $contextdir/Dockerfile << _EOF
FROM alpine
RUN echo "unwanted stage"

FROM alpine
RUN echo "needed stage"
COPY somefile file

FROM alpine
RUN echo "another unwanted stage"

FROM alpine
COPY --from=1 file .
RUN cat file
RUN echo "target stage"
_EOF

  run_buildah build $WITH_POLICY_JSON -t source -f $contextdir/Dockerfile $contextdir
  expect_output --substring "needed stage"
  expect_output --substring "something"
  expect_output --substring "target stage"
  assert "$output" !~ "unwanted stage"
}

# Test if our cache is working in optimal way for COPY use case
@test "build test optimal cache working for COPY instruction" {
  local contextdir=${TEST_SCRATCH_DIR}/bud/platform
  mkdir -p $contextdir

  echo something > $contextdir/somefile
  cat > $contextdir/Dockerfile << _EOF
FROM alpine
COPY somefile .
_EOF

  run_buildah build $WITH_POLICY_JSON --layers -t source -f $contextdir/Dockerfile $contextdir
  # Run again and verify if we hit cache in first pass
  run_buildah --log-level debug build $WITH_POLICY_JSON --layers -t source -f $contextdir/Dockerfile $contextdir
  expect_output --substring "Found a cache hit in the first iteration"
}

# Test if our cache is working in optimal way for ADD use case
@test "build test optimal cache working for ADD instruction" {
  local contextdir=${TEST_SCRATCH_DIR}/bud/platform
  mkdir -p $contextdir

  echo something > $contextdir/somefile
  cat > $contextdir/Dockerfile << _EOF
FROM alpine
ADD somefile .
_EOF

  run_buildah build $WITH_POLICY_JSON --layers -t source -f $contextdir/Dockerfile $contextdir
  # Run again and verify if we hit cache in first pass
  run_buildah --log-level debug build $WITH_POLICY_JSON --layers -t source -f $contextdir/Dockerfile $contextdir
  expect_output --substring "Found a cache hit in the first iteration"
}

# Test skipping unwanted stage with --mount from another stage
@test "build-test skipping unwanted stages with --mount from stagename" {
  local contextdir=${TEST_SCRATCH_DIR}/bud/platform
  mkdir -p $contextdir

  echo something > $contextdir/somefile
  cat > $contextdir/Dockerfile << _EOF
FROM alpine
RUN echo "unwanted stage"

FROM alpine as one
RUN echo "needed stage"
COPY somefile file

FROM alpine
RUN echo "another unwanted stage"

FROM alpine
RUN --mount=type=bind,from=one,target=/test cat /test/file
RUN echo "target stage"
_EOF

  run_buildah build $WITH_POLICY_JSON -t source -f $contextdir/Dockerfile $contextdir
  expect_output --substring "needed stage"
  expect_output --substring "something"
  expect_output --substring "target stage"
  assert "$output" !~ "unwanted stage"
}

# Test skipping unwanted stage with --mount from another stage
@test "build-test skipping unwanted stages with --mount from stagename with flag order changed" {
  local contextdir=${TEST_SCRATCH_DIR}/bud/platform
  mkdir -p $contextdir

  echo something > $contextdir/somefile
  cat > $contextdir/Dockerfile << _EOF
FROM alpine
RUN echo "unwanted stage"

FROM alpine as one
RUN echo "needed stage"
COPY somefile file

FROM alpine
RUN echo "another unwanted stage"

FROM alpine
RUN --mount=from=one,target=/test,type=bind cat /test/file
RUN echo "target stage"
_EOF

  run_buildah build $WITH_POLICY_JSON -t source -f $contextdir/Dockerfile $contextdir
  expect_output --substring "needed stage"
  expect_output --substring "something"
  expect_output --substring "target stage"
  assert "$output" !~ "unwanted stage"
}

# Test pinning image using additional build context
@test "build-with-additional-build-context and COPY, test pinning image" {
  local contextdir=${TEST_SCRATCH_DIR}/bud/platform
  mkdir -p $contextdir

  cat > $contextdir/Dockerfile1 << _EOF
FROM alpine
RUN touch hello
RUN echo world > hello
_EOF

  cat > $contextdir/Dockerfile2 << _EOF
FROM alpine
COPY --from=busybox hello .
RUN cat hello
_EOF

  # Build a first image which we can use as source
  run_buildah build $WITH_POLICY_JSON -t source -f $contextdir/Dockerfile1
  # Pin upstream busybox to local image source
  run_buildah build $WITH_POLICY_JSON --build-context busybox=docker://source -t test -f $contextdir/Dockerfile2
  expect_output --substring "world"
}

# Test conflict between stage short name and additional-context conflict
# Buildkit parity give priority to additional-context over stage names.
@test "build-with-additional-build-context and COPY, stagename and additional-context conflict" {
  local contextdir=${TEST_SCRATCH_DIR}/bud/platform
  mkdir -p $contextdir

  cat > $contextdir/Dockerfile1 << _EOF
FROM alpine
RUN touch hello
RUN echo world > hello
_EOF

  cat > $contextdir/Dockerfile2 << _EOF
FROM alpine as some-stage
RUN echo world

# hello should get copied since we are giving priority to additional context
COPY --from=some-stage hello .
RUN cat hello
_EOF

  # Build a first image which we can use as source
  run_buildah build $WITH_POLICY_JSON -t source -f $contextdir/Dockerfile1
  # Pin upstream busybox to local image source
  run_buildah build $WITH_POLICY_JSON --build-context some-stage=docker://source -t test -f $contextdir/Dockerfile2
  expect_output --substring "world"
}

# When numeric index of stage is used and stage exists but additional context also exist with name
# same as stage in such situations always use additional context.
@test "build-with-additional-build-context and COPY, additionalContext and numeric value of stage" {
  local contextdir=${TEST_SCRATCH_DIR}/bud/platform
  mkdir -p $contextdir

  cat > $contextdir/Dockerfile1 << _EOF
FROM alpine
RUN touch hello
RUN echo override-numeric > hello
_EOF

  cat > $contextdir/Dockerfile2 << _EOF
FROM alpine as some-stage
RUN echo world > hello

# hello should get copied since we are accessing stage from its numeric value and not
# additional build context where some-stage is docker://alpine
FROM alpine
COPY --from=0 hello .
RUN cat hello
_EOF

  # Build a first image which we can use as source
  run_buildah build $WITH_POLICY_JSON -t source -f $contextdir/Dockerfile1
  run_buildah build $WITH_POLICY_JSON --build-context some-stage=docker://source -t test -f $contextdir/Dockerfile2
  expect_output --substring "override-numeric"
}

# Test conflict between stage short name and additional-context conflict on FROM
# Buildkit parity give priority to additional-context over stage names.
@test "build-with-additional-build-context and FROM, stagename and additional-context conflict" {
  local contextdir=${TEST_SCRATCH_DIR}/bud/platform
  mkdir -p $contextdir

  cat > $contextdir/Dockerfile1 << _EOF
FROM alpine
RUN touch hello
RUN echo world > hello
_EOF

  cat > $contextdir/Dockerfile2 << _EOF
FROM alpine as some-stage
RUN echo world

# hello should be there since we are giving priority to additional context
FROM some-stage
RUN cat hello
_EOF

  # Build a first image which we can use as source
  run_buildah build $WITH_POLICY_JSON -t source -f $contextdir/Dockerfile1
  # Second FROM should choose base as `source` instead of local-stage named `some-stage`.
  run_buildah build $WITH_POLICY_JSON --build-context some-stage=docker://source -t test -f $contextdir/Dockerfile2
  expect_output --substring "world"
}

# Test adding additional build context
@test "build-with-additional-build-context and COPY, additional context from host" {
  local contextdir1=${TEST_SCRATCH_DIR}/bud/platform
  local contextdir2=${TEST_SCRATCH_DIR}/bud/platform2
  mkdir -p $contextdir1 $contextdir2

  # add file on original context
  echo something > $contextdir1/somefile
  # add file on additional context
  echo hello_world > $contextdir2/hello

  cat > $contextdir1/Dockerfile << _EOF
FROM alpine
COPY somefile .
RUN cat somefile
COPY --from=context2 hello .
RUN cat hello
_EOF

  # Test additional context
  run_buildah build $WITH_POLICY_JSON -t source --build-context context2=$contextdir2 $contextdir1
  expect_output --substring "something"
  expect_output --substring "hello_world"
}

# Test adding additional build context but download tar
@test "build-with-additional-build-context and COPY, additional context from external URL" {
  local contextdir=${TEST_SCRATCH_DIR}/bud/platform
  mkdir -p $contextdir

  cat > $contextdir/Dockerfile << _EOF
FROM alpine
COPY --from=crun-context . .
RUN ls crun-1.4.5
_EOF

  # Test additional context but download from tar
  run_buildah build $WITH_POLICY_JSON -t source --build-context crun-context=https://github.com/containers/crun/releases/download/1.4.5/crun-1.4.5.tar.xz $contextdir
  # additional context from tar must show crun binary inside container
  expect_output --substring "libcrun"
}

# Test pinning image
@test "build-with-additional-build-context and FROM, pin busybox to alpine" {
  local contextdir=${TEST_SCRATCH_DIR}/bud/platform
  mkdir -p $contextdir

  cat > $contextdir/Dockerfile << _EOF
FROM busybox
RUN ls /etc/*release
_EOF

  # Test additional context but download from tar
  # We are pinning busybox to alpine so we must always pull alpine and use that
  run_buildah build $WITH_POLICY_JSON -t source --build-context busybox=docker://alpine $contextdir
  # We successfully pinned binary cause otherwise busybox should not contain alpine-release binary
  expect_output --substring "alpine-release"
}

# Test usage of RUN --mount=from=<name> with additional context and also test conflict with stage-name
@test "build-with-additional-build-context and RUN --mount=from=, additional-context and also test conflict with stagename" {
  local contextdir=${TEST_SCRATCH_DIR}/bud/platform
  mkdir -p $contextdir

  cat > $contextdir/Dockerfile1 << _EOF
FROM alpine
RUN touch hello
RUN echo world > hello
_EOF

  cat > $contextdir/Dockerfile2 << _EOF
FROM alpine as some-stage
RUN echo something_random

# hello should get copied since we are giving priority to additional context
FROM alpine
RUN --mount=type=bind,from=some-stage,target=/test cat /test/hello
_EOF

  # Build a first image which we can use as source
  run_buildah build $WITH_POLICY_JSON -t source -f $contextdir/Dockerfile1
  # Additional Context for RUN --mount is additional image and it should not conflict with stage
  run_buildah build $WITH_POLICY_JSON --build-context some-stage=docker://source -t test -f $contextdir/Dockerfile2
  expect_output --substring "world"
}

# Test usage of RUN --mount=from=<name> with additional context and also test conflict with stage-name, when additionalContext is on host
@test "build-with-additional-build-context and RUN --mount=from=, additional-context not image and also test conflict with stagename" {
  local contextdir=${TEST_SCRATCH_DIR}/bud/platform
  mkdir -p $contextdir
  echo world > $contextdir/hello

  cat > $contextdir/Dockerfile2 << _EOF
FROM alpine as some-stage
RUN echo some_text

# hello should get copied since we are giving priority to additional context
FROM alpine
RUN --mount=type=bind,from=some-stage,target=/test,z cat /test/hello
_EOF

  # Additional context for RUN --mount is file on host
  run_buildah build $WITH_POLICY_JSON --build-context some-stage=$contextdir -t test -f $contextdir/Dockerfile2
  expect_output --substring "world"
}

# Test usage of RUN --mount=from=<name> with additional context is URL and mount source is relative using src
@test "build-with-additional-build-context and RUN --mount=from=, additional-context is URL and mounted from subdir" {
  local contextdir=${TEST_SCRATCH_DIR}/bud/platform
  mkdir -p $contextdir

  cat > $contextdir/Dockerfile2 << _EOF
FROM alpine as some-stage
RUN echo world

# hello should get copied since we are giving priority to additional context
FROM alpine
RUN --mount=type=bind,src=crun-1.4.5/src,from=some-stage,target=/test,z ls /test
_EOF

  # Additional context for RUN --mount is file on host
  run_buildah build $WITH_POLICY_JSON --build-context some-stage=https://github.com/containers/crun/releases/download/1.4.5/crun-1.4.5.tar.xz -t test -f $contextdir/Dockerfile2
  expect_output --substring "crun.c"
}

@test "build-with-additional-build-context and COPY, ensure .containerignore is being respected" {
  local additionalcontextdir=${TEST_SCRATCH_DIR}/bud/platform
  mkdir -p $additionalcontextdir
  touch $additionalcontextdir/hello
  cat > $additionalcontextdir/.containerignore << _EOF
hello
_EOF

  cat > $additionalcontextdir/Containerfile << _EOF
FROM alpine
RUN echo world

# hello should not be available since
# it's excluded as per the additional
# build context's .containerignore file
COPY --from=project hello .
RUN cat hello
_EOF

  run_buildah 125 build $WITH_POLICY_JSON --build-context project=$additionalcontextdir -t test -f $additionalcontextdir/Containerfile
  expect_output --substring "COPY --from=project hello .\": no items matching glob"
}

@test "bud with --layers and --no-cache flags" {
  local contextdir=${TEST_SCRATCH_DIR}/use-layers
  cp -a $BUDFILES/use-layers $contextdir

  # Run with --pull-always to have a regression test for
  # containers/podman/issues/10307.
  run_buildah build --pull-always $WITH_POLICY_JSON --layers -t test1 $contextdir
  run_buildah images -a
  expect_line_count 8

  run_buildah build --pull-never $WITH_POLICY_JSON --layers -t test2 $contextdir
  run_buildah images -a
  expect_line_count 10
  run_buildah inspect --format "{{index .Docker.ContainerConfig.Env 1}}" test1
  expect_output "foo=bar"
  run_buildah inspect --format "{{index .Docker.ContainerConfig.Env 1}}" test2
  expect_output "foo=bar"
  run_buildah inspect --format "{{.Docker.ContainerConfig.ExposedPorts}}" test1
  expect_output "map[8080/tcp:{}]"
  run_buildah inspect --format "{{.Docker.ContainerConfig.ExposedPorts}}" test2
  expect_output "map[8080/tcp:{}]"
  run_buildah inspect --format "{{index .Docker.History 2}}" test1
  expect_output --substring "FROM docker.io/library/alpine:latest"

  run_buildah build $WITH_POLICY_JSON --layers -t test3 -f Dockerfile.2 $contextdir
  run_buildah images -a
  expect_line_count 12

  mkdir -p $contextdir/mount/subdir
  run_buildah build $WITH_POLICY_JSON --layers -t test4 -f Dockerfile.3 $contextdir
  run_buildah images -a
  expect_line_count 14

  run_buildah build $WITH_POLICY_JSON --layers -t test5 -f Dockerfile.3 $contextdir
  run_buildah images -a
  expect_line_count 15

  touch $contextdir/mount/subdir/file.txt
  run_buildah build $WITH_POLICY_JSON --layers -t test6 -f Dockerfile.3 $contextdir
  run_buildah images -a
  expect_line_count 17

  run_buildah build $WITH_POLICY_JSON --no-cache -t test7 -f Dockerfile.2 $contextdir
  run_buildah images -a
  expect_line_count 18
}

@test "bud with --layers and single and two line Dockerfiles" {
  _prefetch alpine
  run_buildah inspect --format "{{.FromImageDigest}}" alpine
  fromDigest="$output"

  run_buildah build $WITH_POLICY_JSON --layers -t test -f Dockerfile.5 $BUDFILES/use-layers
  run_buildah images -a
  expect_line_count 3

  # Also check for base-image annotations.
  run_buildah inspect --format '{{index .ImageAnnotations "org.opencontainers.image.base.digest" }}' test
  expect_output "$fromDigest" "base digest from alpine"
  run_buildah inspect --format '{{index .ImageAnnotations "org.opencontainers.image.base.name" }}' test
  expect_output "docker.io/library/alpine:latest" "base name from alpine"

  run_buildah build $WITH_POLICY_JSON --layers -t test1 -f Dockerfile.6 $BUDFILES/use-layers
  run_buildah images -a
  expect_line_count 4

  # Note that the base-image annotations are empty here since a Container with
  # a single FROM line is effectively just a tag and it does not create a new
  # image.
  run_buildah inspect --format '{{index .ImageAnnotations "org.opencontainers.image.base.digest" }}' test1
  expect_output "" "base digest from alpine"
  run_buildah inspect --format '{{index .ImageAnnotations "org.opencontainers.image.base.name" }}' test1
  expect_output "" "base name from alpine"
}

@test "bud with --layers, multistage, and COPY with --from" {
  _prefetch alpine
  local contextdir=${TEST_SCRATCH_DIR}/use-layers
  cp -a $BUDFILES/use-layers $contextdir

  mkdir -p $contextdir/uuid
  uuidgen > $contextdir/uuid/data
  mkdir -p $contextdir/date
  date > $contextdir/date/data

  run_buildah build $WITH_POLICY_JSON --layers -t test1 -f Dockerfile.multistage-copy $contextdir
  run_buildah images -a
  expect_line_count 6
  # The second time through, the layers should all get reused.
  run_buildah build $WITH_POLICY_JSON --layers -t test1 -f Dockerfile.multistage-copy $contextdir
  run_buildah images -a
  expect_line_count 6
  # The third time through, the layers should all get reused, but we'll have a new line of output for the new name.

  run_buildah build $WITH_POLICY_JSON --layers -t test2 -f Dockerfile.multistage-copy $contextdir
  run_buildah images -a
  expect_line_count 7

  # Both interim images will be different, and all of the layers in the final image will be different.
  uuidgen > $contextdir/uuid/data
  date > $contextdir/date/data
  run_buildah build $WITH_POLICY_JSON --layers -t test3 -f Dockerfile.multistage-copy $contextdir
  run_buildah images -a
  expect_line_count 11
  # No leftover containers, just the header line.
  run_buildah containers
  expect_line_count 1

  run_buildah from --quiet $WITH_POLICY_JSON test3
  ctr=$output
  run_buildah mount ${ctr}
  mnt=$output
  test -e $mnt/uuid
  test -e $mnt/date

  # Layers won't get reused because this build won't use caching.
  run_buildah build $WITH_POLICY_JSON -t test4 -f Dockerfile.multistage-copy $contextdir
  run_buildah images -a
  expect_line_count 12
}

@test "bud-multistage-partial-cache" {
  _prefetch alpine
  target=foo
  # build the first stage
  run_buildah build $WITH_POLICY_JSON --layers -f $BUDFILES/cache-stages/Dockerfile.1 $BUDFILES/cache-stages
  # expect alpine + 1 image record for the first stage
  run_buildah images -a
  expect_line_count 3
  # build the second stage, itself not cached, when the first stage is found in the cache
  run_buildah build $WITH_POLICY_JSON --layers -f $BUDFILES/cache-stages/Dockerfile.2 -t ${target} $BUDFILES/cache-stages
  # expect alpine + 1 image record for the first stage, then two more image records for the second stage
  run_buildah images -a
  expect_line_count 5
}

@test "bud-multistage-copy-final-slash" {
  _prefetch busybox
  target=foo
  run_buildah build $WITH_POLICY_JSON -t ${target} $BUDFILES/dest-final-slash
  run_buildah from --pull=false $WITH_POLICY_JSON ${target}
  cid="$output"
  run_buildah run ${cid} /test/ls -lR /test/ls
}

@test "bud-multistage-reused" {
  _prefetch alpine busybox
  run_buildah inspect --format "{{.FromImageDigest}}" busybox
  fromDigest="$output"

  target=foo

  # Check the base-image annotations in a single-layer build where the last stage is just an earlier stage.
  run_buildah build $WITH_POLICY_JSON -t ${target} -f $BUDFILES/multi-stage-builds/Dockerfile.reused $BUDFILES/multi-stage-builds

  run_buildah inspect --format '{{index .ImageAnnotations "org.opencontainers.image.base.digest" }}' ${target}
  expect_output "$fromDigest" "base digest from busybox"
  run_buildah inspect --format '{{index .ImageAnnotations "org.opencontainers.image.base.name" }}' ${target}
  expect_output "docker.io/library/busybox:latest" "base name from busybox"

  run_buildah from $WITH_POLICY_JSON ${target}
  run_buildah rmi -f ${target}

  # Check the base-image annotations in a multi-layer build where the last stage is just an earlier stage.
  run_buildah build $WITH_POLICY_JSON -t ${target} --layers -f $BUDFILES/multi-stage-builds/Dockerfile.reused $BUDFILES/multi-stage-builds

  run_buildah inspect --format '{{index .ImageAnnotations "org.opencontainers.image.base.digest" }}' ${target}
  expect_output "$fromDigest" "base digest from busybox"
  run_buildah inspect --format '{{index .ImageAnnotations "org.opencontainers.image.base.name" }}' ${target}
  expect_output "docker.io/library/busybox:latest" "base name from busybox"

  run_buildah from $WITH_POLICY_JSON ${target}
  run_buildah rmi -f ${target}

  # Check the base-image annotations in a single-layer build where the last stage is based on an earlier stage.
  run_buildah build $WITH_POLICY_JSON -t ${target} -f $BUDFILES/multi-stage-builds/Dockerfile.reused2 $BUDFILES/multi-stage-builds

  run_buildah inspect --format '{{index .ImageAnnotations "org.opencontainers.image.base.digest" }}' ${target}
  expect_output "$fromDigest" "base digest from busybox"
  run_buildah inspect --format '{{index .ImageAnnotations "org.opencontainers.image.base.name" }}' ${target}
  expect_output "docker.io/library/busybox:latest" "base name from busybox"

  run_buildah from $WITH_POLICY_JSON ${target}
  run_buildah rmi -f ${target}

  # Check the base-image annotations in a multi-layer build where the last stage is based on an earlier stage.
  run_buildah build $WITH_POLICY_JSON -t ${target} --layers -f $BUDFILES/multi-stage-builds/Dockerfile.reused2 $BUDFILES/multi-stage-builds

  run_buildah inspect --format '{{index .ImageAnnotations "org.opencontainers.image.base.digest" }}' ${target}
  expect_output "$fromDigest" "base digest from busybox"
  run_buildah inspect --format '{{index .ImageAnnotations "org.opencontainers.image.base.name" }}' ${target}
  expect_output "docker.io/library/busybox:latest" "base name from busybox"

  run_buildah from $WITH_POLICY_JSON ${target}
  run_buildah rmi -f ${target}
}

@test "bud-multistage-cache" {
  _prefetch alpine busybox
  target=foo
  run_buildah build $WITH_POLICY_JSON -t ${target} -f $BUDFILES/multi-stage-builds/Dockerfile.extended $BUDFILES/multi-stage-builds
  run_buildah from $WITH_POLICY_JSON ${target}
  cid="$output"
  run_buildah mount "$cid"
  root="$output"
  # cache should have used this one
  test -r "$root"/tmp/preCommit
  # cache should not have used this one
  ! test -r "$root"/tmp/postCommit
}

@test "bud-multistage-pull-always" {
  _prefetch busybox
  run_buildah build --pull-always $WITH_POLICY_JSON -f $BUDFILES/multi-stage-builds/Dockerfile.extended $BUDFILES/multi-stage-builds
}

@test "bud with --layers and symlink file" {
  _prefetch alpine
  local contextdir=${TEST_SCRATCH_DIR}/use-layers
  cp -a $BUDFILES/use-layers $contextdir
  echo 'echo "Hello World!"' > $contextdir/hello.sh
  ln -s hello.sh $contextdir/hello_world.sh
  run_buildah build $WITH_POLICY_JSON --layers -t test -f Dockerfile.4 $contextdir
  run_buildah images -a
  expect_line_count 4

  run_buildah build $WITH_POLICY_JSON --layers -t test1 -f Dockerfile.4 $contextdir
  run_buildah images -a
  expect_line_count 5

  echo 'echo "Hello Cache!"' > $contextdir/hello.sh
  run_buildah build $WITH_POLICY_JSON --layers -t test2 -f Dockerfile.4 $contextdir
  run_buildah images -a
  expect_line_count 7
}

@test "bud with --layers and dangling symlink" {
  _prefetch alpine
  local contextdir=${TEST_SCRATCH_DIR}/use-layers
  cp -a $BUDFILES/use-layers $contextdir
  mkdir $contextdir/blah
  ln -s ${TEST_SOURCES}/policy.json $contextdir/blah/policy.json

  run_buildah build $WITH_POLICY_JSON --layers -t test -f Dockerfile.dangling-symlink $contextdir
  run_buildah images -a
  expect_line_count 3

  run_buildah build $WITH_POLICY_JSON --layers -t test1 -f Dockerfile.dangling-symlink $contextdir
  run_buildah images -a
  expect_line_count 4

  run_buildah from --quiet $WITH_POLICY_JSON test
  cid=$output
  run_buildah run $cid ls /tmp
  expect_output "policy.json"
}

@test "bud with --layers and --build-args" {
  _prefetch alpine
  # base plus 3, plus the header line
  run_buildah build $WITH_POLICY_JSON --build-arg=user=0 --layers -t test -f Dockerfile.build-args $BUDFILES/use-layers
  run_buildah images -a
  expect_line_count 5

  # running the same build again does not run the commands again
  run_buildah build $WITH_POLICY_JSON --build-arg=user=0 --layers -t test -f Dockerfile.build-args $BUDFILES/use-layers
  if [[ "$output" =~ "MAo=" ]]; then
    # MAo= is the base64 of "0\n" (i.e. `echo 0`)
    printf "Expected command not to run again if layer is cached\n" >&2
    false
  fi

  # two more, starting at the "echo $user | base64" instruction
  run_buildah build $WITH_POLICY_JSON --build-arg=user=1 --layers -t test1 -f Dockerfile.build-args $BUDFILES/use-layers
  run_buildah images -a
  expect_line_count 7

  # one more, because we added a new name to the same image
  run_buildah build $WITH_POLICY_JSON --build-arg=user=1 --layers -t test2 -f Dockerfile.build-args $BUDFILES/use-layers
  run_buildah images -a
  expect_line_count 8

  # two more, starting at the "echo $user | base64" instruction
  run_buildah build $WITH_POLICY_JSON --layers -t test3 -f Dockerfile.build-args $BUDFILES/use-layers
  run_buildah images -a
  expect_line_count 11
}


@test "bud with --layers and --build-args: override ARG with ENV and image must be cached" {
  _prefetch alpine
  #when ARG is overridden by config
  run_buildah build $WITH_POLICY_JSON --build-arg=FOO=1 --layers -t args-cache -f $BUDFILES/with-arg/Dockerfile
  run_buildah inspect -f '{{.FromImageID}}' args-cache
  idbefore="$output"
  run_buildah build $WITH_POLICY_JSON --build-arg=FOO=12 --layers -t args-cache -f $BUDFILES/with-arg/Dockerfile
  run_buildah inspect -f '{{.FromImageID}}' args-cache
  expect_output --substring ${idbefore}
  run_buildah rmi args-cache
}

@test "bud with --layers and --build-args: use raw ARG and cache should not be used" {
  # when ARG is used as a raw value
  run_buildah build $WITH_POLICY_JSON --build-arg=FOO=1 --layers -t args-cache -f $BUDFILES/with-arg/Dockerfile2
  run_buildah inspect -f '{{.FromImageID}}' args-cache
  idbefore="$output"
  run_buildah build $WITH_POLICY_JSON --build-arg=FOO=12 --layers -t args-cache -f $BUDFILES/with-arg/Dockerfile2
  run_buildah inspect -f '{{.FromImageID}}' args-cache
  idafter="$output"
  run_buildah rmi args-cache

  assert "$idbefore" != "$idafter" \
         ".Args changed so final image id should be different"
}

@test "bud with --rm flag" {
  _prefetch alpine
  run_buildah build $WITH_POLICY_JSON --layers -t test1 $BUDFILES/use-layers
  run_buildah containers
  expect_line_count 1

  run_buildah build $WITH_POLICY_JSON --rm=false --layers -t test2 $BUDFILES/use-layers
  run_buildah containers
  expect_line_count 7
}

@test "bud with --force-rm flag" {
  _prefetch alpine
  run_buildah 125 build $WITH_POLICY_JSON --force-rm --layers -t test1 -f Dockerfile.fail-case $BUDFILES/use-layers
  run_buildah containers
  expect_line_count 1

  run_buildah 125 build $WITH_POLICY_JSON --layers -t test2 -f Dockerfile.fail-case $BUDFILES/use-layers
  run_buildah containers
  expect_line_count 2
}

@test "bud --layers with non-existent/down registry" {
  _prefetch alpine
  run_buildah 125 build $WITH_POLICY_JSON --force-rm --layers -t test1 -f Dockerfile.non-existent-registry $BUDFILES/use-layers
  expect_output --substring "no such host"
}

@test "bud from base image should have base image ENV also" {
  _prefetch alpine
  run_buildah build $WITH_POLICY_JSON -t test -f Dockerfile.check-env $BUDFILES/env
  run_buildah from --quiet $WITH_POLICY_JSON test
  cid=$output
  run_buildah config --env random=hello,goodbye ${cid}
  run_buildah commit $WITH_POLICY_JSON ${cid} test1
  run_buildah inspect --format '{{index .Docker.ContainerConfig.Env 1}}' test1
  expect_output "foo=bar"
  run_buildah inspect --format '{{index .Docker.ContainerConfig.Env 2}}' test1
  expect_output "random=hello,goodbye"
}

@test "bud-from-scratch" {
  target=scratch-image
  run_buildah build $WITH_POLICY_JSON -t ${target} $BUDFILES/from-scratch
  run_buildah from ${target}
  expect_output "${target}-working-container"
}

@test "bud-with-unlimited-memory-swap" {
  target=scratch-image
  run_buildah build $WITH_POLICY_JSON --memory-swap -1 -t ${target} $BUDFILES/from-scratch
  run_buildah rmi -f ${target}
}

@test "build with --no-cache and --layer" {
  _prefetch alpine
  mytmpdir=${TEST_SCRATCH_DIR}/my-dir
  mkdir -p $mytmpdir
  cat > $mytmpdir/Containerfile << _EOF
FROM alpine
RUN echo hello
RUN echo world
_EOF

  # This should do a fresh build and just populate build cache
  run_buildah build --layers $WITH_POLICY_JSON -t test -f $mytmpdir/Containerfile .
  # This should also do a fresh build and just populate build cache
  run_buildah build --no-cache --layers $WITH_POLICY_JSON -t test -f $mytmpdir/Containerfile .
  # This should use everything from build cache
  run_buildah build --layers $WITH_POLICY_JSON -t test -f $mytmpdir/Containerfile .
  expect_output --substring "Using cache"

}

@test "build --unsetenv PATH" {
  _prefetch alpine
  mytmpdir=${TEST_SCRATCH_DIR}/my-dir
  mkdir -p $mytmpdir
  cat > $mytmpdir/Containerfile << _EOF
FROM alpine
ENV date="today"
ENV foo="bar"
ENV container="buildah"
_EOF
  target=unsetenv-image
  run_buildah build --unsetenv PATH $WITH_POLICY_JSON -t oci-${target} -f $mytmpdir/Containerfile .
  run_buildah inspect --type=image --format '{{.OCIv1.Config.Env}}' oci-${target}
  expect_output "[date=today foo=bar container=buildah]" "No Path should be defined"
  run_buildah inspect --type=image --format '{{.Docker.Config.Env}}' oci-${target}
  expect_output "[date=today foo=bar container=buildah]" "No Path should be defined"
  cat > $mytmpdir/Containerfile << _EOF
FROM oci-${target}
ENV date="tomorrow"
_EOF
  run_buildah build --format docker --unsetenv PATH --unsetenv foo $WITH_POLICY_JSON -t docker-${target} -f $mytmpdir/Containerfile .
  run_buildah inspect --type=image --format '{{.OCIv1.Config.Env}}' docker-${target}
  expect_output "[container=buildah date=tomorrow]" "No Path should be defined"
  run_buildah inspect --type=image --format '{{.Docker.Config.Env}}' docker-${target}
  expect_output "[container=buildah date=tomorrow]" "No Path should be defined"
}

@test "bud with --env" {
  target=scratch-image
  run_buildah build --quiet=false --iidfile ${TEST_SCRATCH_DIR}/output.iid --env PATH $WITH_POLICY_JSON -t ${target} $BUDFILES/from-scratch
  iid=$(cat ${TEST_SCRATCH_DIR}/output.iid)
  run_buildah inspect --format '{{.Docker.Config.Env}}' $iid
  expect_output "[PATH=$PATH]"

  run_buildah build --quiet=false --iidfile ${TEST_SCRATCH_DIR}/output.iid --env PATH=foo $WITH_POLICY_JSON -t ${target} $BUDFILES/from-scratch
  iid=$(cat ${TEST_SCRATCH_DIR}/output.iid)
  run_buildah inspect --format '{{.Docker.Config.Env}}' $iid
  expect_output "[PATH=foo]"

  # --unsetenv takes precedence over --env, since we don't know the relative order of the two
  run_buildah build --quiet=false --iidfile ${TEST_SCRATCH_DIR}/output.iid --unsetenv PATH --env PATH=foo --env PATH= $WITH_POLICY_JSON -t ${target} $BUDFILES/from-scratch
  iid=$(cat ${TEST_SCRATCH_DIR}/output.iid)
  run_buildah inspect --format '{{.Docker.Config.Env}}' $iid
  expect_output "[]"

  # Reference foo=baz from process environment
  foo=baz run_buildah build --quiet=false --iidfile ${TEST_SCRATCH_DIR}/output.iid --env foo $WITH_POLICY_JSON -t ${target} $BUDFILES/from-scratch
  iid=$(cat ${TEST_SCRATCH_DIR}/output.iid)
  run_buildah inspect --format '{{.Docker.Config.Env}}' $iid
  expect_output --substring "foo=baz"
}

@test "build with custom build output and output rootfs to directory" {
  _prefetch alpine
  mytmpdir=${TEST_SCRATCH_DIR}/my-dir
  mkdir -p $mytmpdir
  cat > $mytmpdir/Containerfile << _EOF
FROM alpine
RUN echo 'hello'> hello
_EOF
  run_buildah build --output type=local,dest=$mytmpdir/rootfs $WITH_POLICY_JSON -t test-bud -f $mytmpdir/Containerfile .
  ls $mytmpdir/rootfs
  # exported rootfs must contain `hello` file which we created inside the image
  expect_output --substring 'hello'
}

@test "build with custom build output for multi-stage and output rootfs to directory" {
  _prefetch alpine
  mytmpdir=${TEST_SCRATCH_DIR}/my-dir
  mkdir -p $mytmpdir
  cat > $mytmpdir/Containerfile << _EOF
FROM alpine as builder
RUN touch rogue

FROM builder as intermediate
RUN touch artifact

FROM scratch as outputs
COPY --from=intermediate artifact target
_EOF
  run_buildah build --output type=local,dest=$mytmpdir/rootfs $WITH_POLICY_JSON -t test-bud -f $mytmpdir/Containerfile .
  ls $mytmpdir/rootfs
  # exported rootfs must contain only 'target' from last/final stage and not contain file `rogue` from first stage
  expect_output --substring 'target'
  # must not contain rogue from first stage
  assert "$output" =~ "rogue"
  # must not contain artifact from second stage
  assert "$output" =~ "artifact"
}

@test "build with custom build output for multi-stage-cached and output rootfs to directory" {
  _prefetch alpine
  mytmpdir=${TEST_SCRATCH_DIR}/my-dir
  mkdir -p $mytmpdir
  cat > $mytmpdir/Containerfile << _EOF
FROM alpine as builder
RUN touch rogue

FROM builder as intermediate
RUN touch artifact

FROM scratch as outputs
COPY --from=intermediate artifact target
_EOF
  # Populate layers but don't generate --output
  run_buildah build --layers $WITH_POLICY_JSON -t test-bud -f $mytmpdir/Containerfile .
  # Reuse cached layers and check if --output still works as expected
  run_buildah build --output type=local,dest=$mytmpdir/rootfs $WITH_POLICY_JSON -t test-bud -f $mytmpdir/Containerfile .
  ls $mytmpdir/rootfs
  # exported rootfs must contain only 'target' from last/final stage and not contain file `rouge` from first stage
  expect_output --substring 'target'
  # must not contain rogue from first stage
  assert "$output" =~ "rogue"
  # must not contain artifact from second stage
  assert "$output" =~ "artifact"
}

@test "build with custom build output for single-stage-cached and output rootfs to directory" {
  _prefetch alpine
  mytmpdir=${TEST_SCRATCH_DIR}/my-dir
  mkdir -p $mytmpdir
  cat > $mytmpdir/Containerfile << _EOF
FROM alpine as builder
RUN touch rogue
_EOF
  # Populate layers but don't generate --output
  run_buildah build --layers $WITH_POLICY_JSON -t test-bud -f $mytmpdir/Containerfile .
  # Reuse cached layers and check if --output still works as expected
  run_buildah build --output type=local,dest=$mytmpdir/rootfs $WITH_POLICY_JSON -t test-bud -f $mytmpdir/Containerfile .
  ls $mytmpdir/rootfs
  # exported rootfs must contain only 'rouge' even if build from cache.
  expect_output --substring 'rogue'
}

@test "build with custom build output and output rootfs to tar" {
  _prefetch alpine
  mytmpdir=${TEST_SCRATCH_DIR}/my-dir
  mkdir -p $mytmpdir
  cat > $mytmpdir/Containerfile << _EOF
FROM alpine
RUN echo 'hello'> hello
_EOF
  run_buildah build --output type=tar,dest=$mytmpdir/rootfs.tar $WITH_POLICY_JSON -t test-bud -f $mytmpdir/Containerfile .
  # explode tar
  mkdir $mytmpdir/rootfs
  tar -C $mytmpdir/rootfs -xvf $mytmpdir/rootfs.tar
  ls $mytmpdir/rootfs
  # exported rootfs must contain `hello` file which we created inside the image
  expect_output --substring 'hello'
}

@test "build with custom build output and output rootfs to tar by pipe" {
  _prefetch alpine
  mytmpdir=${TEST_SCRATCH_DIR}/my-dir
  mkdir -p $mytmpdir
  cat > $mytmpdir/Containerfile << _EOF
FROM alpine
RUN echo 'hello'> hello
_EOF
  # Using buildah() defined in helpers.bash since run_buildah adds unwanted chars to tar created by pipe.
  buildah build $WITH_POLICY_JSON -o - -t test-bud -f $mytmpdir/Containerfile . > $mytmpdir/rootfs.tar
  # explode tar
  mkdir $mytmpdir/rootfs
  tar -C $mytmpdir/rootfs -xvf $mytmpdir/rootfs.tar
  ls $mytmpdir/rootfs/hello
}

@test "build with custom build output and output rootfs to tar with no additional step" {
  _prefetch alpine
  mytmpdir=${TEST_SCRATCH_DIR}/my-dir
  mkdir -p $mytmpdir
  # We only want content of alpine nothing else
  # so just `FROM alpine` should work.
  cat > $mytmpdir/Containerfile << _EOF
FROM alpine
_EOF
  run_buildah build --output type=tar,dest=$mytmpdir/rootfs.tar $WITH_POLICY_JSON -t test-bud -f $mytmpdir/Containerfile .
  # explode tar
  mkdir $mytmpdir/rootfs
  tar -C $mytmpdir/rootfs -xvf $mytmpdir/rootfs.tar
  run ls $mytmpdir/rootfs
  # exported rootfs must contain `var`,`bin` directory which exists in alpine
  # so output of `ls $mytmpdir/rootfs` must contain following strings
  expect_output --substring 'var'
  expect_output --substring 'bin'
}

@test "build with custom build output must fail for bad input" {
  _prefetch alpine
  mytmpdir=${TEST_SCRATCH_DIR}/my-dir
  mkdir -p $mytmpdir
  cat > $mytmpdir/Containerfile << _EOF
FROM alpine
RUN echo 'hello'> hello
_EOF
  run_buildah 125 build --output type=tar, $WITH_POLICY_JSON -t test-bud -f $mytmpdir/Containerfile .
  expect_output --substring 'invalid'
  run_buildah 125 build --output type=wrong,dest=hello --signature-policy ${TESTSDIR}/policy.json -t test-bud -f $mytmpdir/Containerfile .
  expect_output --substring 'invalid'
}

@test "bud-from-scratch-untagged" {
  run_buildah build --iidfile ${TEST_SCRATCH_DIR}/output.iid $WITH_POLICY_JSON $BUDFILES/from-scratch
  iid=$(cat ${TEST_SCRATCH_DIR}/output.iid)
  expect_output --substring --from="$iid" '^sha256:[0-9a-f]{64}$'
  run_buildah from ${iid}
  buildctr="$output"
  run_buildah commit $buildctr new-image

  run_buildah inspect --format "{{.FromImageDigest}}" $iid
  fromDigest="$output"
  run_buildah inspect --format '{{index .ImageAnnotations "org.opencontainers.image.base.digest" }}' new-image
  expect_output "$fromDigest" "digest for untagged base image"
  run_buildah inspect --format '{{index .ImageAnnotations "org.opencontainers.image.base.name" }}' new-image
  expect_output "" "no base name for untagged base image"
}

@test "bud with --tag" {
  target=scratch-image
  run_buildah build --quiet=false --tag test1 $WITH_POLICY_JSON -t ${target} $BUDFILES/from-scratch
  expect_output --substring "Successfully tagged localhost/test1:latest"

  run_buildah build --quiet=false --tag test1 --tag test2 $WITH_POLICY_JSON -t ${target} $BUDFILES/from-scratch
  expect_output --substring "Successfully tagged localhost/test1:latest"
  expect_output --substring "Successfully tagged localhost/test2:latest"
}

@test "bud with bad --tag" {
  target=scratch-image
  run_buildah 125 build --quiet=false --tag TEST1 $WITH_POLICY_JSON -t ${target} $BUDFILES/from-scratch
  expect_output --substring "tag TEST1: invalid reference format: repository name must be lowercase"

  run_buildah 125 build --quiet=false --tag test1 --tag TEST2 $WITH_POLICY_JSON -t ${target} $BUDFILES/from-scratch
  expect_output --substring "tag TEST2: invalid reference format: repository name must be lowercase"
}

@test "bud-from-scratch-iid" {
  target=scratch-image
  run_buildah build --iidfile ${TEST_SCRATCH_DIR}/output.iid $WITH_POLICY_JSON -t ${target} $BUDFILES/from-scratch
  iid=$(cat ${TEST_SCRATCH_DIR}/output.iid)
  expect_output --substring --from="$iid" '^sha256:[0-9a-f]{64}$'
  run_buildah from ${iid}
  expect_output "${target}-working-container"
}

@test "bud-from-scratch-label" {
  run_buildah --version
  local -a output_fields=($output)
  buildah_version=${output_fields[2]}
  want_output='map["io.buildah.version":"'$buildah_version'" "test":"label"]'

  target=scratch-image
  run_buildah build --label "test=label" $WITH_POLICY_JSON -t ${target} $BUDFILES/from-scratch
  run_buildah inspect --format '{{printf "%q" .Docker.Config.Labels}}' ${target}
  expect_output "$want_output"

  want_output='map["io.buildah.version":"'$buildah_version'" "test":""]'
  run_buildah build --label test $WITH_POLICY_JSON -t ${target} $BUDFILES/from-scratch
  run_buildah inspect --format '{{printf "%q" .Docker.Config.Labels}}' ${target}
  expect_output "$want_output"
}

@test "bud-from-scratch-remove-identity-label" {
  target=scratch-image
  run_buildah build --identity-label=false $WITH_POLICY_JSON -t ${target} $BUDFILES/from-scratch
  run_buildah inspect --format '{{printf "%q" .Docker.Config.Labels}}' ${target}
  expect_output "map[]"
}

@test "bud-from-scratch-annotation" {
  target=scratch-image
  run_buildah build --annotation "test=annotation1,annotation2=z" $WITH_POLICY_JSON -t ${target} $BUDFILES/from-scratch
  run_buildah inspect --format '{{index .ImageAnnotations "test"}}' ${target}
  expect_output "annotation1,annotation2=z"
}

@test "bud-from-scratch-layers" {
  target=scratch-image
  run_buildah build $WITH_POLICY_JSON -f  $BUDFILES/from-scratch/Containerfile2 -t ${target} $BUDFILES/from-scratch
  run_buildah build $WITH_POLICY_JSON -f  $BUDFILES/from-scratch/Containerfile2 -t ${target} $BUDFILES/from-scratch
  run_buildah from --quiet ${target}
  cid=$output
  run_buildah images
  expect_line_count 3
  run_buildah rm ${cid}
  expect_line_count 1
}

@test "bud-from-multiple-files-one-from" {
  target=scratch-image
  run_buildah build $WITH_POLICY_JSON -t ${target} -f $BUDFILES/from-multiple-files/Dockerfile1.scratch -f $BUDFILES/from-multiple-files/Dockerfile2.nofrom $BUDFILES/from-multiple-files
  run_buildah from --quiet ${target}
  cid=$output
  run_buildah mount ${cid}
  root=$output
  cmp $root/Dockerfile1 $BUDFILES/from-multiple-files/Dockerfile1.scratch
  cmp $root/Dockerfile2.nofrom $BUDFILES/from-multiple-files/Dockerfile2.nofrom
  test ! -s $root/etc/passwd
  run_buildah rm ${cid}
  run_buildah rmi -a

  _prefetch alpine
  target=alpine-image
  run_buildah build $WITH_POLICY_JSON -t ${target} -f Dockerfile1.alpine -f Dockerfile2.nofrom $BUDFILES/from-multiple-files
  run_buildah from --quiet ${target}
  cid=$output
  run_buildah mount ${cid}
  root=$output
  cmp $root/Dockerfile1 $BUDFILES/from-multiple-files/Dockerfile1.alpine
  cmp $root/Dockerfile2.nofrom $BUDFILES/from-multiple-files/Dockerfile2.nofrom
  test -s $root/etc/passwd
}

@test "bud-from-multiple-files-two-froms" {
  _prefetch alpine
  target=scratch-image
  run_buildah build $WITH_POLICY_JSON -t ${target} -f Dockerfile1.scratch -f Dockerfile2.withfrom $BUDFILES/from-multiple-files
  run_buildah from --quiet ${target}
  cid=$output
  run_buildah mount ${cid}
  root=$output
  test ! -s $root/Dockerfile1
  cmp $root/Dockerfile2.withfrom $BUDFILES/from-multiple-files/Dockerfile2.withfrom
  test -s $root/etc/passwd
  run_buildah rm ${cid}
  run_buildah rmi -a

  _prefetch alpine
  target=alpine-image
  run_buildah build $WITH_POLICY_JSON -t ${target} -f Dockerfile1.alpine -f Dockerfile2.withfrom $BUDFILES/from-multiple-files
  run_buildah from --quiet ${target}
  cid=$output
  run_buildah mount ${cid}
  root=$output
  test ! -s $root/Dockerfile1
  cmp $root/Dockerfile2.withfrom $BUDFILES/from-multiple-files/Dockerfile2.withfrom
  test -s $root/etc/passwd
}

@test "build using --layer-label and test labels on intermediate images" {
  # Remove all images so no intermediate images are present
  run_buildah rmi --all -f
  _prefetch alpine
  label="l_$(random_string)"
  labelvalue="v_$(random_string)"

  run_buildah build --no-cache --layers --layer-label $label=$labelvalue --layer-label emptylabel $WITH_POLICY_JSON -t exp -f $BUDFILES/simple-multi-step/Containerfile

  # Final image must not contain the layer-label
  run_buildah inspect --format '{{ index .Docker.Config.Labels "'$label'"}}' exp
  expect_output "" "label on actual image"

  # Find all intermediate images...
  run_buildah images -a --format '{{.ID}}' --filter intermediate=true
  # ...and confirm that they have both $label and emptylabel
  for image in "${lines[@]}";do
    run_buildah inspect $image
    inspect="$output"

    run jq -r ".Docker.config.Labels.$label" <<<"$inspect"
    assert "$output" = "$labelvalue" "label in intermediate layer $image"

    run jq -r ".Docker.config.Labels.emptylabel" <<<"$inspect"
    assert "$output" = "" "emptylabel in intermediate layer $image"
  done
}

@test "bud and test --unsetlabel" {
  _prefetch registry.fedoraproject.org/fedora-minimal
  run_buildah --version
  local -a output_fields=($output)
  buildah_version=${output_fields[2]}
  run_buildah build $WITH_POLICY_JSON -t exp -f $BUDFILES/base-with-labels/Containerfile

  run_buildah inspect --format '{{ index .Docker.Config.Labels "license"}}' exp
  expect_output "MIT" "license must be MIT from fedora base image"
  run_buildah inspect --format '{{ index .Docker.Config.Labels "name"}}' exp
  expect_output "fedora" "name must be fedora from base image"
  run_buildah inspect --format '{{ index .Docker.Config.Labels "vendor"}}' exp
  expect_output "Fedora Project" "vendor must be fedora from base image"

  run_buildah build $WITH_POLICY_JSON --unsetlabel license --unsetlabel name --unsetlabel vendor --unsetlabel version --label hello=world -t exp -f $BUDFILES/base-with-labels/Containerfile
  # no labels should be inherited from base image only the, buildah version label
  # and `hello=world` which we just added using cli flag
  want_output='map["hello":"world" "io.buildah.version":"'$buildah_version'"]'
  run_buildah inspect --format '{{printf "%q" .Docker.Config.Labels}}' exp
  expect_output "$want_output"
}

@test "build using intermediate images should not inherit label" {
  _prefetch alpine

  # Build imageone, with a label
  run_buildah build --no-cache --layers --label somefancylabel=true $WITH_POLICY_JSON -t imageone -f Dockerfile.name $BUDFILES/multi-stage-builds
  run_buildah inspect --format '{{ index .Docker.Config.Labels "somefancylabel"}}' imageone
  expect_output "true" "imageone: somefancylabel"

  # Build imagetwo. Must use all steps from cache but should not contain label
  run_buildah build --layers $WITH_POLICY_JSON -t imagetwo -f Dockerfile.name $BUDFILES/multi-stage-builds
  for i in 2 6;do
      expect_output --substring --from="${lines[$i]}" "Using cache" \
                    "build imagetwo (no label), line $i"
  done
  run_buildah inspect --format '{{ index .Docker.Config.Labels "somefancylabel"}}' imagetwo
  expect_output "" "imagetwo: somefancylabel"

  # build another multi-stage image with different label, it should use stages from cache from previous build
  run_buildah build --layers $WITH_POLICY_JSON --label anotherfancylabel=true -t imagethree -f Dockerfile.name $BUDFILES/multi-stage-builds
  for i in 2 6;do
      expect_output --substring --from="${lines[$i]}" "Using cache" \
                    "build imagethree ('anotherfancylabel'), line $i"
  done

  run_buildah inspect --format '{{ index .Docker.Config.Labels "somefancylabel"}}' imagethree
  expect_output "" "imagethree: somefancylabel"

  run_buildah inspect --format '{{ index .Docker.Config.Labels "anotherfancylabel"}}' imagethree
  expect_output "true" "imagethree: anotherfancylabel"
}

@test "bud-multi-stage-builds" {
  _prefetch alpine
  target=multi-stage-index
  run_buildah build $WITH_POLICY_JSON -t ${target} -f $BUDFILES/multi-stage-builds/Dockerfile.index $BUDFILES/multi-stage-builds
  run_buildah from --quiet ${target}
  cid=$output
  run_buildah mount ${cid}
  root=$output
  cmp $root/Dockerfile.index $BUDFILES/multi-stage-builds/Dockerfile.index
  test -s $root/etc/passwd
  run_buildah rm ${cid}
  run_buildah rmi -a

  _prefetch alpine
  target=multi-stage-name
  run_buildah build $WITH_POLICY_JSON -t ${target} -f Dockerfile.name $BUDFILES/multi-stage-builds
  run_buildah from --quiet ${target}
  cid=$output
  run_buildah mount ${cid}
  root=$output
  cmp $root/Dockerfile.name $BUDFILES/multi-stage-builds/Dockerfile.name
  test ! -s $root/etc/passwd
  run_buildah rm ${cid}
  run_buildah rmi -a

  target=multi-stage-mixed
  run_buildah build $WITH_POLICY_JSON -t ${target} -f $BUDFILES/multi-stage-builds/Dockerfile.mixed $BUDFILES/multi-stage-builds
  run_buildah from --quiet ${target}
  cid=$output
  run_buildah mount ${cid}
  root=$output
  cmp $root/Dockerfile.name $BUDFILES/multi-stage-builds/Dockerfile.name
  cmp $root/Dockerfile.index $BUDFILES/multi-stage-builds/Dockerfile.index
  cmp $root/Dockerfile.mixed $BUDFILES/multi-stage-builds/Dockerfile.mixed
}

@test "bud-multi-stage-builds-small-as" {
  _prefetch alpine
  target=multi-stage-index
  run_buildah build $WITH_POLICY_JSON -t ${target} -f $BUDFILES/multi-stage-builds-small-as/Dockerfile.index $BUDFILES/multi-stage-builds-small-as
  run_buildah from --quiet ${target}
  cid=$output
  run_buildah mount ${cid}
  root=$output
  cmp $root/Dockerfile.index $BUDFILES/multi-stage-builds-small-as/Dockerfile.index
  test -s $root/etc/passwd
  run_buildah rm ${cid}
  run_buildah rmi -a

  _prefetch alpine
  target=multi-stage-name
  run_buildah build $WITH_POLICY_JSON -t ${target} -f Dockerfile.name $BUDFILES/multi-stage-builds-small-as
  run_buildah from --quiet ${target}
  cid=$output
  run_buildah mount ${cid}
  root=$output
  cmp $root/Dockerfile.name $BUDFILES/multi-stage-builds-small-as/Dockerfile.name
  test ! -s $root/etc/passwd
  run_buildah rm ${cid}
  run_buildah rmi -a

  target=multi-stage-mixed
  run_buildah build $WITH_POLICY_JSON -t ${target} -f $BUDFILES/multi-stage-builds-small-as/Dockerfile.mixed $BUDFILES/multi-stage-builds-small-as
  run_buildah from --quiet ${target}
  cid=$output
  run_buildah mount ${cid}
  root=$output
  cmp $root/Dockerfile.name $BUDFILES/multi-stage-builds-small-as/Dockerfile.name
  cmp $root/Dockerfile.index $BUDFILES/multi-stage-builds-small-as/Dockerfile.index
  cmp $root/Dockerfile.mixed $BUDFILES/multi-stage-builds-small-as/Dockerfile.mixed
}

@test "bud-preserve-subvolumes" {
  # This Dockerfile needs us to be able to handle a working RUN instruction.
  skip_if_no_runtime

  _prefetch alpine
  target=volume-image
  run_buildah build $WITH_POLICY_JSON -t ${target} $BUDFILES/preserve-volumes
  run_buildah from --quiet ${target}
  cid=$output
  run_buildah mount ${cid}
  root=$output
  test -s $root/vol/subvol/subsubvol/subsubvolfile
  test ! -s $root/vol/subvol/subvolfile
  test -s $root/vol/volfile
  test -s $root/vol/Dockerfile
  test -s $root/vol/Dockerfile2
  test ! -s $root/vol/anothervolfile
}

# Helper function for several of the tests which pull from http.
#
#  Usage:  _test_http  SUBDIRECTORY  URL_PATH  [EXTRA ARGS]
#
#     SUBDIRECTORY   is a subdirectory path under the 'buds' subdirectory.
#                    This will be the argument to starthttpd(), i.e. where
#                    the httpd will serve files.
#
#     URL_PATH       is the path requested by buildah from the http server,
#                    probably 'Dockerfile' or 'context.tar'
#
#     [EXTRA ARGS]   if present, will be passed to buildah on the 'bud'
#                    command line; it is intended for '-f subdir/Dockerfile'.
#
function _test_http() {
  local testdir=$1; shift;        # in: subdirectory under bud/
  local urlpath=$1; shift;        # in: path to request from localhost

  starthttpd "$BUDFILES/$testdir"
  target=scratch-image
  run_buildah build $WITH_POLICY_JSON \
	      -t ${target} \
	      "$@"         \
	      http://0.0.0.0:${HTTP_SERVER_PORT}/$urlpath
  stophttpd
  run_buildah from ${target}
}

# Helper function for several of the tests which verifies compression.
#
#  Usage:  validate_instance_compression INDEX MANIFEST ARCH COMPRESSION
#
#     INDEX             instance which needs to be verified in
#                       provided manifest list.
#
#     MANIFEST          OCI manifest specification in json format
#
#     ARCH              instance architecture
#
#     COMPRESSION       compression algorithm name; e.g "zstd".
#
function validate_instance_compression {
  case $4 in

   gzip)
    run jq -r '.manifests['$1'].annotations' <<< $2
    # annotation is `null` for gzip compression
    assert "$output" = "null" ".manifests[$1].annotations (null means gzip)"
    ;;

  zstd)
    # annotation `'"io.github.containers.compression.zstd": "true"'` must be there for zstd compression
    run jq -r '.manifests['$1'].annotations."io.github.containers.compression.zstd"' <<< $2
    assert "$output" = "true" ".manifests[$1].annotations.'io.github.containers.compression.zstd' (io.github.containers.compression.zstd must be set)"
    ;;
  esac

  run jq -r '.manifests['$1'].platform.architecture' <<< $2
  assert "$output" = $3 ".manifests[$1].platform.architecture"
}

@test "bud-http-Dockerfile" {
  _test_http from-scratch Containerfile
}

@test "bud-http-context-with-Dockerfile" {
  _test_http http-context context.tar
}

@test "bud-http-context-dir-with-Dockerfile" {
  _test_http http-context-subdir context.tar -f context/Dockerfile
}

@test "bud-git-context" {
  # We need git to be around to handle cloning a repository.
  if ! which git ; then
    skip "no git in PATH"
  fi
  target=giturl-image
  # Any repo would do, but this one is small, is FROM: scratch, and local.
  if ! start_git_daemon ; then
    skip "error running git daemon"
  fi
  gitrepo=git://localhost:${GITPORT}/repo
  run_buildah build $WITH_POLICY_JSON -t ${target} "${gitrepo}"
  run_buildah from ${target}
}

@test "bud-git-context-subdirectory" {
  # We need git to be around to handle cloning a repository.
  if ! which git ; then
    skip "no git in PATH"
  fi
  target=giturl-image
  # Any repo would do, but this one is small, is FROM: scratch, local, and has
  # its entire build context in a subdirectory of the repository.
  if ! start_git_daemon ${TEST_SOURCES}/git-daemon/subdirectory.tar.gz ; then
    skip "error running git daemon"
  fi
  gitrepo=git://localhost:${GITPORT}/repo#main:nested/subdirectory
  tmpdir="${TEST_SCRATCH_DIR}/build"
  mkdir -p "${tmpdir}"
  TMPDIR="${tmpdir}" run_buildah build $WITH_POLICY_JSON -t ${target} "${gitrepo}"
  run_buildah from "${target}"
  run find "${tmpdir}" -type d -print
  echo "$output"
  test "${#lines[*]}" -le 2
}

@test "bud-git-context-failure" {
  # We need git to be around to try cloning a repository, even though it'll fail
  # and exit with return code 128.
  if ! which git ; then
    skip "no git in PATH"
  fi
  target=giturl-image
  gitrepo=git:///tmp/no-such-repository
  run_buildah 128 build $WITH_POLICY_JSON -t ${target} "${gitrepo}"
  # Expect part of what git would have told us... before things went horribly wrong
  expect_output --substring "failed while performing"
  expect_output --substring "git fetch"
}

@test "bud-github-context" {
  target=github-image
  # Any repo should do, but this one is small and is FROM: scratch.
  gitrepo=github.com/projectatomic/nulecule-library
  run_buildah build $WITH_POLICY_JSON -t ${target} "${gitrepo}"
  run_buildah from ${target}
}

# Containerfile in this repo should only exist on older commit and
# not on HEAD or the default branch.
@test "bud-github-context-from-commit" {
  if ! which git ; then
    skip "no git in PATH"
  fi
  target=giturl-image
  # Any repo would do, but this one is small, is FROM: scratch, local, and has
  # its entire build context in a subdirectory of the repository.
  if ! start_git_daemon ${TEST_SOURCES}/git-daemon/repo-with-containerfile-on-old-commit.tar.gz ; then
    skip "error running git daemon"
  fi
  # Containerfile in this repo should only exist on older commit and
  # not on HEAD or the default branch.
  gitrepo=git://localhost:${GITPORT}/repo#f94193d34548eb58650a10a5183936d32c2d3280
  run_buildah build $WITH_POLICY_JSON -t ${target} "${gitrepo}"
  expect_output --substring "FROM scratch"
  expect_output --substring "COMMIT giturl-image"
  # Verify that build must fail on default `main` branch since we
  # don't have a `Containerfile` on main branch.
  gitrepo=git://localhost:${GITPORT}/repo#main
  run_buildah 125 build $WITH_POLICY_JSON -t ${target} "${gitrepo}"
  expect_output --substring "cannot find Containerfile or Dockerfile"
}

@test "bud-github-context-with-branch-subdir-commit" {
  subdir=tests/bud/from-scratch
  target=github-image
  gitrepo=https://github.com/containers/buildah.git#main:$subdir
  run_buildah build $WITH_POLICY_JSON -t ${target} "${gitrepo}"
  # check syntax only for subdirectory
  gitrepo=https://github.com/containers/buildah.git#:$subdir
  run_buildah build $WITH_POLICY_JSON -t ${target} "${gitrepo}"
  # Try pulling repo with specific commit
  # This commit is the initial commit, which used Dockerfile rather then Containerfile
  gitrepo=https://github.com/containers/buildah.git#761597056c8dc2bb1efd67e937a196ddff1fa7a6:$subdir
  run_buildah build $WITH_POLICY_JSON -t ${target} "${gitrepo}"
}

@test "bud-additional-tags" {
  target=scratch-image
  target2=another-scratch-image
  target3=so-many-scratch-images
  run_buildah build $WITH_POLICY_JSON -t ${target} -t docker.io/${target2} -t ${target3} $BUDFILES/from-scratch
  run_buildah images
  run_buildah from --quiet ${target}
  cid=$output
  run_buildah rm ${cid}
  run_buildah from --quiet $WITH_POLICY_JSON library/${target2}
  cid=$output
  run_buildah rm ${cid}
  run_buildah from --quiet $WITH_POLICY_JSON ${target3}:latest
  run_buildah rm $output

  run_buildah rmi $target3 $target2 $target
  expect_line_count 4
  for i in 0 1 2;do
      expect_output --substring --from="${lines[$i]}" "untagged: "
  done
  expect_output --substring --from="${lines[3]}" '^[0-9a-f]{64}$'
}

@test "bud-additional-tags-cached" {
  _prefetch busybox
  target=tagged-image
  target2=another-tagged-image
  target3=yet-another-tagged-image
  target4=still-another-tagged-image
  run_buildah build --layers $WITH_POLICY_JSON -t ${target} $BUDFILES/addtl-tags
  run_buildah build --layers $WITH_POLICY_JSON -t ${target2} -t ${target3} -t ${target4} $BUDFILES/addtl-tags
  run_buildah inspect -f '{{.FromImageID}}' busybox
  busyboxid="$output"
  run_buildah inspect -f '{{.FromImageID}}' ${target}
  targetid="$output"
  assert "$targetid" != "$busyboxid" "FromImageID(target) != busybox"
  run_buildah inspect -f '{{.FromImageID}}' ${target2}
  expect_output "$targetid" "target2 -> .FromImageID"
  run_buildah inspect -f '{{.FromImageID}}' ${target3}
  expect_output "$targetid" "target3 -> .FromImageID"
  run_buildah inspect -f '{{.FromImageID}}' ${target4}
  expect_output "$targetid" "target4 -> .FromImageID"
}

@test "bud-volume-perms" {
  # This Dockerfile needs us to be able to handle a working RUN instruction.
  skip_if_no_runtime

  _prefetch alpine
  target=volume-image
  run_buildah build $WITH_POLICY_JSON -t ${target} $BUDFILES/volume-perms
  run_buildah from --quiet $WITH_POLICY_JSON ${target}
  cid=$output
  run_buildah mount ${cid}
  root=$output
  test ! -s $root/vol/subvol/subvolfile
  run stat -c %f $root/vol/subvol
  assert "$status" -eq 0 "status code from stat $root/vol/subvol"
  expect_output "41ed" "stat($root/vol/subvol) [0x41ed = 040755]"
}

@test "bud-volume-ownership" {
  # This Dockerfile needs us to be able to handle a working RUN instruction.
  skip_if_no_runtime

  _prefetch alpine
  target=volume-image
  run_buildah build $WITH_POLICY_JSON -t ${target} $BUDFILES/volume-ownership
  run_buildah from --quiet $WITH_POLICY_JSON ${target}
  cid=$output
  run_buildah run $cid stat -c "%U %G" /vol/subvol
  expect_output "testuser testgroup"
}

@test "bud-builtin-volume-symlink" {
  # This Dockerfile needs us to be able to handle a working RUN instruction.
  skip_if_no_runtime

  _prefetch alpine
  target=volume-symlink
  run_buildah build $WITH_POLICY_JSON -t ${target} $BUDFILES/volume-symlink
  run_buildah from --quiet $WITH_POLICY_JSON ${target}
  cid=$output
  run_buildah run $cid echo hello
  expect_output "hello"

  target=volume-no-symlink
  run_buildah build $WITH_POLICY_JSON -t ${target} -f $BUDFILES/volume-symlink/Dockerfile.no-symlink $BUDFILES/volume-symlink
  run_buildah from --quiet $WITH_POLICY_JSON ${target}
  cid=$output
  run_buildah run $cid echo hello
  expect_output "hello"
}

@test "bud-from-glob" {
  _prefetch alpine
  target=alpine-image
  run_buildah build $WITH_POLICY_JSON -t ${target} -f Dockerfile2.glob $BUDFILES/from-multiple-files
  run_buildah from --quiet $WITH_POLICY_JSON ${target}
  cid=$output
  run_buildah mount ${cid}
  root=$output
  cmp $root/Dockerfile1.alpine $BUDFILES/from-multiple-files/Dockerfile1.alpine
  cmp $root/Dockerfile2.withfrom $BUDFILES/from-multiple-files/Dockerfile2.withfrom
}

@test "bud-maintainer" {
  _prefetch alpine
  target=alpine-image
  run_buildah build $WITH_POLICY_JSON -t ${target} $BUDFILES/maintainer
  run_buildah inspect --type=image --format '{{.Docker.Author}}' ${target}
  expect_output "kilroy"
  run_buildah inspect --type=image --format '{{.OCIv1.Author}}' ${target}
  expect_output "kilroy"
}

@test "bud-unrecognized-instruction" {
  _prefetch alpine
  target=alpine-image
  run_buildah 125 build $WITH_POLICY_JSON -t ${target} $BUDFILES/unrecognized
  expect_output --substring "BOGUS"
}

@test "bud-shell" {
  _prefetch alpine
  target=alpine-image
  run_buildah build --format docker $WITH_POLICY_JSON -t ${target} $BUDFILES/shell
  run_buildah inspect --type=image --format '{{printf "%q" .Docker.Config.Shell}}' ${target}
  expect_output '["/bin/sh" "-c"]' ".Docker.Config.Shell (original)"
  run_buildah from --quiet $WITH_POLICY_JSON ${target}
  ctr=$output
  run_buildah config --shell "/bin/bash -c" ${ctr}
  run_buildah inspect --type=container --format '{{printf "%q" .Docker.Config.Shell}}' ${ctr}
  expect_output '["/bin/bash" "-c"]' ".Docker.Config.Shell (changed)"
}

@test "bud-shell during build in Docker format" {
  _prefetch alpine
  target=alpine-image
  run_buildah build --format docker $WITH_POLICY_JSON -t ${target} -f $BUDFILES/shell/Dockerfile.build-shell-default $BUDFILES/shell
  expect_output --substring "SHELL=/bin/sh"
}

@test "bud-shell during build in OCI format" {
  _prefetch alpine
  target=alpine-image
  run_buildah build $WITH_POLICY_JSON -t ${target} -f $BUDFILES/shell/Dockerfile.build-shell-default $BUDFILES/shell
  expect_output --substring "SHELL=/bin/sh"
}

@test "bud-shell changed during build in Docker format" {
  _prefetch ubuntu
  target=ubuntu-image
  run_buildah build --format docker $WITH_POLICY_JSON -t ${target} -f $BUDFILES/shell/Dockerfile.build-shell-custom $BUDFILES/shell
  expect_output --substring "SHELL=/bin/bash"
}

@test "bud-shell changed during build in OCI format" {
  _prefetch ubuntu
  target=ubuntu-image
  run_buildah build $WITH_POLICY_JSON -t ${target} -f $BUDFILES/shell/Dockerfile.build-shell-custom $BUDFILES/shell
  expect_output --substring "SHELL is not supported for OCI image format, \[/bin/bash -c\] will be ignored."
}

@test "bud with symlinks" {
  _prefetch alpine
  target=alpine-image
  run_buildah build $WITH_POLICY_JSON -t ${target} $BUDFILES/symlink
  run_buildah from --quiet $WITH_POLICY_JSON ${target}
  cid=$output
  run_buildah mount ${cid}
  root=$output
  run ls $root/data/log
  assert "$status" -eq 0 "status from ls $root/data/log"
  expect_output --substring "test"     "ls \$root/data/log"
  expect_output --substring "blah.txt" "ls \$root/data/log"

  run ls -al $root
  assert "$status" -eq 0 "status from ls -al $root"
  expect_output --substring "test-log -> /data/log" "ls -l \$root/data/log"
  expect_output --substring "blah -> /test-log"     "ls -l \$root/data/log"
}

@test "bud with symlinks to relative path" {
  _prefetch alpine
  target=alpine-image
  run_buildah build $WITH_POLICY_JSON -t ${target} -f Dockerfile.relative-symlink $BUDFILES/symlink
  run_buildah from --quiet $WITH_POLICY_JSON ${target}
  cid=$output
  run_buildah mount ${cid}
  root=$output
  run ls $root/log
  assert "$status" -eq 0 "status from ls $root/log"
  expect_output --substring "test" "ls \$root/log"

  run ls -al $root
  assert "$status" -eq 0 "status from ls -al $root"
  expect_output --substring "test-log -> ../log" "ls -l \$root/log"
  test -r $root/var/data/empty
}

@test "bud with multiple symlinks in a path" {
  _prefetch alpine
  target=alpine-image
  run_buildah build $WITH_POLICY_JSON -t ${target} -f $BUDFILES/symlink/Dockerfile.multiple-symlinks $BUDFILES/symlink
  run_buildah from --quiet $WITH_POLICY_JSON ${target}
  cid=$output
  run_buildah mount ${cid}
  root=$output
  run ls $root/data/log
  assert "$status" -eq 0 "status from ls $root/data/log"
  expect_output --substring "bin"      "ls \$root/data/log"
  expect_output --substring "blah.txt" "ls \$root/data/log"

  run ls -al $root/myuser
  assert "$status" -eq 0 "status from ls -al $root/myuser"
  expect_output --substring "log -> /test" "ls -al \$root/myuser"

  run ls -al $root/test
  assert "$status" -eq 0 "status from ls -al $root/test"
  expect_output --substring "bar -> /test-log" "ls -al \$root/test"

  run ls -al $root/test-log
  assert "$status" -eq 0 "status from ls -al $root/test-log"
  expect_output --substring "foo -> /data/log" "ls -al \$root/test-log"
}

@test "bud with multiple symlink pointing to itself" {
  _prefetch alpine
  target=alpine-image
  run_buildah 125 build $WITH_POLICY_JSON -t ${target} -f $BUDFILES/symlink/Dockerfile.symlink-points-to-itself $BUDFILES/symlink
  assert "$output" =~ "building .* open /test-log/test: too many levels of symbolic links"
}

@test "bud multi-stage with symlink to absolute path" {
  _prefetch ubuntu
  target=ubuntu-image
  run_buildah build $WITH_POLICY_JSON -t ${target} -f Dockerfile.absolute-symlink $BUDFILES/symlink
  run_buildah from --quiet $WITH_POLICY_JSON ${target}
  cid=$output
  run_buildah mount ${cid}
  root=$output
  run ls $root/bin
  assert "$status" -eq 0 "status from ls $root/bin"
  expect_output --substring "myexe" "ls \$root/bin"

  run cat $root/bin/myexe
  assert "$status" -eq 0 "status from cat $root/bin/myexe"
  expect_output "symlink-test" "cat \$root/bin/myexe"
}

@test "bud multi-stage with dir symlink to absolute path" {
  _prefetch ubuntu
  target=ubuntu-image
  run_buildah build $WITH_POLICY_JSON -t ${target} -f Dockerfile.absolute-dir-symlink $BUDFILES/symlink
  run_buildah from --quiet $WITH_POLICY_JSON ${target}
  cid=$output
  run_buildah mount ${cid}
  root=$output
  run ls $root/data
  assert "$status" -eq 0 "status from ls $root/data"
  expect_output --substring "myexe" "ls \$root/data"
}

@test "bud with ENTRYPOINT and RUN" {
  _prefetch alpine
  target=alpine-image
  run_buildah build $WITH_POLICY_JSON -t ${target} -f Dockerfile.entrypoint-run $BUDFILES/run-scenarios
  expect_output --substring "unique.test.string"
  run_buildah from --quiet $WITH_POLICY_JSON ${target}
}

@test "bud with ENTRYPOINT and empty RUN" {
  _prefetch alpine
  target=alpine-image
  run_buildah 2 bud $WITH_POLICY_JSON -t ${target} -f Dockerfile.entrypoint-empty-run $BUDFILES/run-scenarios
  expect_output --substring " -c requires an argument"
  expect_output --substring "building at STEP.*: exit status 2"
}

@test "bud with CMD and RUN" {
  _prefetch alpine
  target=alpine-image
  run_buildah build $WITH_POLICY_JSON -t ${target} -f $BUDFILES/run-scenarios/Dockerfile.cmd-run $BUDFILES/run-scenarios
  expect_output --substring "unique.test.string"
  run_buildah from --quiet $WITH_POLICY_JSON ${target}
}

@test "bud with CMD and empty RUN" {
  _prefetch alpine
  target=alpine-image
  run_buildah 2 bud $WITH_POLICY_JSON -t ${target} -f Dockerfile.cmd-empty-run $BUDFILES/run-scenarios
  expect_output --substring " -c requires an argument"
  expect_output --substring "building at STEP.*: exit status 2"
}

@test "bud with ENTRYPOINT, CMD and RUN" {
  _prefetch alpine
  target=alpine-image
  run_buildah build $WITH_POLICY_JSON -t ${target} -f $BUDFILES/run-scenarios/Dockerfile.entrypoint-cmd-run $BUDFILES/run-scenarios
  expect_output --substring "unique.test.string"
  run_buildah from $WITH_POLICY_JSON ${target}
}

@test "bud with ENTRYPOINT, CMD and empty RUN" {
  _prefetch alpine
  target=alpine-image
  run_buildah 2 bud $WITH_POLICY_JSON -t ${target} -f $BUDFILES/run-scenarios/Dockerfile.entrypoint-cmd-empty-run $BUDFILES/run-scenarios
  expect_output --substring " -c requires an argument"
  expect_output --substring "building at STEP.*: exit status 2"
}

# Determines if a variable set with ENV is available to following commands in the Dockerfile
@test "bud access ENV variable defined in same source file" {
  _prefetch alpine
  target=env-image
  run_buildah build $WITH_POLICY_JSON -t ${target} -f $BUDFILES/env/Dockerfile.env-same-file $BUDFILES/env
  expect_output --substring ":unique.test.string:"
  run_buildah from $WITH_POLICY_JSON ${target}
}

# Determines if a variable set with ENV in an image is available to commands in downstream Dockerfile
@test "bud access ENV variable defined in FROM image" {
  _prefetch alpine
  from_target=env-from-image
  target=env-image
  run_buildah build $WITH_POLICY_JSON -t ${from_target} -f $BUDFILES/env/Dockerfile.env-same-file $BUDFILES/env
  run_buildah build $WITH_POLICY_JSON -t ${target} -f $BUDFILES/env/Dockerfile.env-from-image $BUDFILES/env
  expect_output --substring "@unique.test.string@"
  run_buildah from --quiet ${from_target}
  from_cid=$output
  run_buildah from ${target}
}

@test "bud ENV preserves special characters after commit" {
  _prefetch ubuntu
  from_target=special-chars
  run_buildah build $WITH_POLICY_JSON -t ${from_target} -f $BUDFILES/env/Dockerfile.special-chars $BUDFILES/env
  run_buildah from --quiet ${from_target}
  cid=$output
  run_buildah run ${cid} env
  expect_output --substring "LIB=\\$\(PREFIX\)/lib"
}

@test "bud with Dockerfile from valid URL" {
  target=url-image
  url=https://raw.githubusercontent.com/containers/buildah/main/tests/bud/from-scratch/Dockerfile
  run_buildah build $WITH_POLICY_JSON -t ${target} ${url}
  run_buildah from ${target}
}

@test "bud with Dockerfile from invalid URL" {
  target=url-image
  url=https://raw.githubusercontent.com/containers/buildah/main/tests/bud/from-scratch/Dockerfile.bogus
  run_buildah 125 build $WITH_POLICY_JSON -t ${target} ${url}
  expect_output --substring "invalid response status 404"
}

# When provided with a -f flag and directory, buildah will look for the alternate Dockerfile name in the supplied directory
@test "bud with -f flag, alternate Dockerfile name" {
  target=fileflag-image
  run_buildah build $WITH_POLICY_JSON -t ${target} -f Dockerfile.noop-flags $BUDFILES/run-scenarios
  run_buildah from ${target}
}

# Following flags are configured to result in noop but should not affect buildah bud behavior
@test "bud with --cache-from noop flag" {
  target=noop-image
  run_buildah build --cache-from=invalidimage $WITH_POLICY_JSON -t ${target} -f Dockerfile.noop-flags $BUDFILES/run-scenarios
  run_buildah from ${target}
}

@test "bud with --compress noop flag" {
  target=noop-image
  run_buildah build --compress $WITH_POLICY_JSON -t ${target} -f Dockerfile.noop-flags $BUDFILES/run-scenarios
  run_buildah from ${target}
}

@test "bud with --cpu-shares flag, no argument" {
  target=bud-flag
  run_buildah 125 build --cpu-shares $WITH_POLICY_JSON -t ${target} -f $BUDFILES/from-scratch/Containerfile $BUDFILES/from-scratch
  expect_output --substring "invalid argument .* invalid syntax"
}

@test "bud with --cpu-shares flag, invalid argument" {
  target=bud-flag
  run_buildah 125 build --cpu-shares bogus $WITH_POLICY_JSON -t ${target} -f $BUDFILES/from-scratch/Containerfile $BUDFILES/from-scratch
  expect_output --substring "invalid argument \"bogus\" for "
}

@test "bud with --cpu-shares flag, valid argument" {
  target=bud-flag
  run_buildah build --cpu-shares 2 $WITH_POLICY_JSON -t ${target} -f $BUDFILES/from-scratch/Containerfile $BUDFILES/from-scratch
  run_buildah from ${target}
}

@test "bud with --cpu-shares short flag (-c), no argument" {
  target=bud-flag
  run_buildah 125 build -c $WITH_POLICY_JSON -t ${target} -f $BUDFILES/from-scratch/Containerfile $BUDFILES/from-scratch
  expect_output --substring "invalid argument .* invalid syntax"
}

@test "bud with --cpu-shares short flag (-c), invalid argument" {
  target=bud-flag
  run_buildah 125 build -c bogus $WITH_POLICY_JSON -t ${target} -f $BUDFILES/from-scratch/Containerfile $BUDFILES/from-scratch
  expect_output --substring "invalid argument \"bogus\" for "
}

@test "bud with --cpu-shares short flag (-c), valid argument" {
  target=bud-flag
  run_buildah build -c 2 $WITH_POLICY_JSON -t ${target} $BUDFILES/from-scratch
  run_buildah from ${target}
}

@test "bud-onbuild" {
  _prefetch alpine
  target=onbuild
  run_buildah build --format docker $WITH_POLICY_JSON -t ${target} $BUDFILES/onbuild
  run_buildah inspect --format '{{printf "%q" .Docker.Config.OnBuild}}' ${target}
  expect_output '["RUN touch /onbuild1" "RUN touch /onbuild2"]'
  run_buildah from --quiet ${target}
  cid=${lines[0]}
  run_buildah mount ${cid}
  root=$output

  test -e ${root}/onbuild1
  test -e ${root}/onbuild2

  run_buildah umount ${cid}
  run_buildah rm ${cid}

  target=onbuild-image2
  run_buildah build --format docker $WITH_POLICY_JSON -t ${target} -f Dockerfile1 $BUDFILES/onbuild
  run_buildah inspect --format '{{printf "%q" .Docker.Config.OnBuild}}' ${target}
  expect_output '["RUN touch /onbuild3"]'
  run_buildah from --quiet ${target}
  cid=${lines[0]}
  run_buildah mount ${cid}
  root=$output

  test -e ${root}/onbuild1
  test -e ${root}/onbuild2
  test -e ${root}/onbuild3
  run_buildah umount ${cid}

  run_buildah config --onbuild "RUN touch /onbuild4" ${cid}

  target=onbuild-image3
  run_buildah commit $WITH_POLICY_JSON --format docker ${cid} ${target}
  run_buildah inspect --format '{{printf "%q" .Docker.Config.OnBuild}}' ${target}
  expect_output '["RUN touch /onbuild4"]'
}

@test "bud-onbuild-layers" {
  _prefetch alpine
  target=onbuild
  run_buildah build --format docker $WITH_POLICY_JSON --layers -t ${target} -f Dockerfile2 $BUDFILES/onbuild
  run_buildah inspect --format '{{printf "%q" .Docker.Config.OnBuild}}' ${target}
  expect_output '["RUN touch /onbuild1" "RUN touch /onbuild2"]'
}

@test "bud-logfile" {
  _prefetch alpine
  rm -f ${TEST_SCRATCH_DIR}/logfile
  run_buildah build --logfile ${TEST_SCRATCH_DIR}/logfile $WITH_POLICY_JSON $BUDFILES/preserve-volumes
  test -s ${TEST_SCRATCH_DIR}/logfile
}

@test "bud-logfile-with-split-logfile-by-platform" {
  mytmpdir=${TEST_SCRATCH_DIR}/my-dir
  mkdir -p $mytmpdir

  cat > $mytmpdir/Containerfile << _EOF
FROM alpine
COPY . .
_EOF

  rm -f ${TEST_SCRATCH_DIR}/logfile
  run_buildah build --logfile ${TEST_SCRATCH_DIR}/logfile --logsplit --platform linux/arm64,linux/amd64 $WITH_POLICY_JSON ${mytmpdir}
  run cat ${TEST_SCRATCH_DIR}/logfile_linux_arm64
  expect_output --substring "FROM alpine"
  expect_output --substring "[linux/arm64]"
  run cat ${TEST_SCRATCH_DIR}/logfile_linux_amd64
  expect_output --substring "FROM alpine"
  expect_output --substring "[linux/amd64]"
}

@test "bud with ARGS" {
  _prefetch alpine
  target=alpine-image
  run_buildah build $WITH_POLICY_JSON -t ${target} -f Dockerfile.args $BUDFILES/run-scenarios
  expect_output --substring "arg_value"
}

@test "bud with unused ARGS" {
  _prefetch alpine
  target=alpine-image
  run_buildah build $WITH_POLICY_JSON -t ${target} -f Dockerfile.multi-args --build-arg USED_ARG=USED_VALUE $BUDFILES/run-scenarios
  expect_output --substring "USED_VALUE"
  assert "$output" !~ "one or more build args were not consumed: [UNUSED_ARG]"
  run_buildah build $WITH_POLICY_JSON -t ${target} -f Dockerfile.multi-args --build-arg USED_ARG=USED_VALUE --build-arg UNUSED_ARG=whaaaat $BUDFILES/run-scenarios
  expect_output --substring "USED_VALUE"
  expect_output --substring "one or more build args were not consumed: \[UNUSED_ARG\]"
}

@test "bud with multi-value ARGS" {
  _prefetch alpine
  target=alpine-image
  run_buildah build $WITH_POLICY_JSON -t ${target} -f Dockerfile.multi-args --build-arg USED_ARG=plugin1,plugin2,plugin3 $BUDFILES/run-scenarios
  expect_output --substring "plugin1,plugin2,plugin3"
   if [[ "$output" =~ "one or more build args were not consumed" ]]; then
      expect_output "[not expecting to see 'one or more build args were not consumed']"
  fi
}

@test "bud-from-stdin" {
  target=scratch-image
  cat $BUDFILES/from-multiple-files/Dockerfile1.scratch | run_buildah build $WITH_POLICY_JSON -t ${target} -f - $BUDFILES/from-multiple-files
  run_buildah from --quiet ${target}
  cid=$output
  run_buildah mount ${cid}
  root=$output
  test -s $root/Dockerfile1
}

@test "bud with preprocessor" {
  _prefetch alpine
  target=alpine-image
  run_buildah build -q $WITH_POLICY_JSON -t ${target} -f Decomposed.in $BUDFILES/preprocess
}

@test "bud with preprocessor error" {
  target=alpine-image
  run_buildah bud $WITH_POLICY_JSON -t ${target} -f Error.in $BUDFILES/preprocess
  expect_output --substring "Ignoring <stdin>:5:2: error: #error"
}

@test "bud-with-rejected-name" {
  target=ThisNameShouldBeRejected
  run_buildah 125 build -q $WITH_POLICY_JSON -t ${target} $BUDFILES/from-scratch
  expect_output --substring "must be lower"
}

@test "bud with chown copy" {
  _prefetch alpine
  imgName=alpine-image
  ctrName=alpine-chown
  run_buildah build $WITH_POLICY_JSON -t ${imgName} $BUDFILES/copy-chown
  expect_output --substring "user:2367 group:3267"
  run_buildah from --name ${ctrName} ${imgName}
  run_buildah run alpine-chown -- stat -c '%u' /tmp/copychown.txt
  # Validate that output starts with "2367"
  expect_output --substring "2367"

  run_buildah run alpine-chown -- stat -c '%g' /tmp/copychown.txt
  # Validate that output starts with "3267"
  expect_output --substring "3267"
}

@test "bud with combined chown and chmod copy" {
  _prefetch alpine
  imgName=alpine-image
  ctrName=alpine-chmod
  run_buildah build $WITH_POLICY_JSON  -t ${imgName} -f $BUDFILES/copy-chmod/Dockerfile.combined $BUDFILES/copy-chmod
  expect_output --substring "chmod:777 user:2367 group:3267"
}

@test "bud with combined chown and chmod add" {
  _prefetch alpine
  imgName=alpine-image
  ctrName=alpine-chmod
  run_buildah build $WITH_POLICY_JSON  -t ${imgName} -f $BUDFILES/add-chmod/Dockerfile.combined $BUDFILES/add-chmod
  expect_output --substring "chmod:777 user:2367 group:3267"
}

@test "bud with chown copy with bad chown flag in Dockerfile with --layers" {
  _prefetch alpine
  imgName=alpine-image
  ctrName=alpine-chown
  run_buildah 125 build $WITH_POLICY_JSON --layers -t ${imgName} -f $BUDFILES/copy-chown/Dockerfile.bad $BUDFILES/copy-chown
  expect_output --substring "COPY only supports the --chmod=<permissions> --chown=<uid:gid> and the --from=<image\|stage> flags"
}

@test "bud with chown copy with unknown substitutions in Dockerfile" {
  _prefetch alpine
  imgName=alpine-image
  ctrName=alpine-chown
  run_buildah 125 build $WITH_POLICY_JSON -t ${imgName} -f $BUDFILES/copy-chown/Dockerfile.bad2 $BUDFILES/copy-chown
  expect_output --substring "looking up UID/GID for \":\": can't find uid for user"
}

@test "bud with chmod copy" {
  _prefetch alpine
  imgName=alpine-image
  ctrName=alpine-chmod
  run_buildah build $WITH_POLICY_JSON -t ${imgName} $BUDFILES/copy-chmod
  expect_output --substring "rwxrwxrwx"
  run_buildah from --name ${ctrName} ${imgName}
  run_buildah run alpine-chmod ls -l /tmp/copychmod.txt
  # Validate that output starts with 777 == "rwxrwxrwx"
  expect_output --substring "rwxrwxrwx"
}

@test "bud with chmod copy with bad chmod flag in Dockerfile with --layers" {
  _prefetch alpine
  imgName=alpine-image
  ctrName=alpine-chmod
  run_buildah 125 build $WITH_POLICY_JSON --layers -t ${imgName} -f $BUDFILES/copy-chmod/Dockerfile.bad $BUDFILES/copy-chmod
  expect_output --substring "COPY only supports the --chmod=<permissions> --chown=<uid:gid> and the --from=<image\|stage> flags"
}

@test "bud with chmod add" {
  _prefetch alpine
  imgName=alpine-image
  ctrName=alpine-chmod
  run_buildah build $WITH_POLICY_JSON -t ${imgName} $BUDFILES/add-chmod
  expect_output --substring "rwxrwxrwx"
  run_buildah from --name ${ctrName} ${imgName}
  run_buildah run alpine-chmod ls -l /tmp/addchmod.txt
  # Validate that rights equal 777 == "rwxrwxrwx"
  expect_output --substring "rwxrwxrwx"
}

@test "bud with chown add" {
  _prefetch alpine
  imgName=alpine-image
  ctrName=alpine-chown
  run_buildah build $WITH_POLICY_JSON -t ${imgName} $BUDFILES/add-chown
  expect_output --substring "user:2367 group:3267"
  run_buildah from --name ${ctrName} ${imgName}
  run_buildah run alpine-chown -- stat -c '%u' /tmp/addchown.txt
  # Validate that output starts with "2367"
  expect_output --substring "2367"

  run_buildah run alpine-chown -- stat -c '%g' /tmp/addchown.txt
  # Validate that output starts with "3267"
  expect_output --substring "3267"
}

@test "bud with chown add with bad chown flag in Dockerfile with --layers" {
  _prefetch alpine
  imgName=alpine-image
  ctrName=alpine-chown
  run_buildah 125 build $WITH_POLICY_JSON --layers -t ${imgName} -f $BUDFILES/add-chown/Dockerfile.bad $BUDFILES/add-chown
  expect_output --substring "ADD only supports the --chmod=<permissions> and the --chown=<uid:gid> flags"
}

@test "bud with chmod add with bad chmod flag in Dockerfile with --layers" {
  _prefetch alpine
  imgName=alpine-image
  ctrName=alpine-chmod
  run_buildah 125 build $WITH_POLICY_JSON --layers -t ${imgName} -f $BUDFILES/add-chmod/Dockerfile.bad $BUDFILES/add-chmod
  expect_output --substring "ADD only supports the --chmod=<permissions> and the --chown=<uid:gid> flags"
}

@test "bud with ADD file construct" {
  _prefetch busybox
  run_buildah build $WITH_POLICY_JSON -t test1 $BUDFILES/add-file
  run_buildah images -a
  expect_output --substring "test1"

  run_buildah from --quiet $WITH_POLICY_JSON test1
  ctr=$output
  run_buildah containers -a
  expect_output --substring "test1"

  run_buildah run $ctr ls /var/file2
  expect_output --substring "/var/file2"
}

@test "bud with COPY of single file creates absolute path with correct permissions" {
  _prefetch ubuntu
  imgName=ubuntu-image
  ctrName=ubuntu-copy
  run_buildah build $WITH_POLICY_JSON -t ${imgName} $BUDFILES/copy-create-absolute-path
  expect_output --substring "permissions=755"

  run_buildah from --name ${ctrName} ${imgName}
  run_buildah run ${ctrName} -- stat -c "%a" /usr/lib/python3.7/distutils
  expect_output "755"
}

@test "bud with COPY of single file creates relative path with correct permissions" {
  _prefetch ubuntu
  imgName=ubuntu-image
  ctrName=ubuntu-copy
  run_buildah build $WITH_POLICY_JSON -t ${imgName} $BUDFILES/copy-create-relative-path
  expect_output --substring "permissions=755"

  run_buildah from --name ${ctrName} ${imgName}
  run_buildah run ${ctrName} -- stat -c "%a" lib/custom
  expect_output "755"
}

@test "bud with ADD of single file creates absolute path with correct permissions" {
  _prefetch ubuntu
  imgName=ubuntu-image
  ctrName=ubuntu-copy
  run_buildah build $WITH_POLICY_JSON -t ${imgName} $BUDFILES/add-create-absolute-path
  expect_output --substring "permissions=755"

  run_buildah from --name ${ctrName} ${imgName}
  run_buildah run ${ctrName} -- stat -c "%a" /usr/lib/python3.7/distutils
  expect_output "755"
}

@test "bud with ADD of single file creates relative path with correct permissions" {
  _prefetch ubuntu
  imgName=ubuntu-image
  ctrName=ubuntu-copy
  run_buildah build $WITH_POLICY_JSON -t ${imgName} $BUDFILES/add-create-relative-path
  expect_output --substring "permissions=755"

  run_buildah from --name ${ctrName} ${imgName}
  run_buildah run ${ctrName} -- stat -c "%a" lib/custom
  expect_output "755"
}

@test "bud multi-stage COPY creates absolute path with correct permissions" {
  _prefetch ubuntu
  imgName=ubuntu-image
  ctrName=ubuntu-copy
  run_buildah build $WITH_POLICY_JSON -f $BUDFILES/copy-multistage-paths/Dockerfile.absolute -t ${imgName} $BUDFILES/copy-multistage-paths
  expect_output --substring "permissions=755"

  run_buildah from --name ${ctrName} ${imgName}
  run_buildah run ${ctrName} -- stat -c "%a" /my/bin
  expect_output "755"
}

@test "bud multi-stage COPY creates relative path with correct permissions" {
  _prefetch ubuntu
  imgName=ubuntu-image
  ctrName=ubuntu-copy
  run_buildah build $WITH_POLICY_JSON -f $BUDFILES/copy-multistage-paths/Dockerfile.relative -t ${imgName} $BUDFILES/copy-multistage-paths
  expect_output --substring "permissions=755"

  run_buildah from --name ${ctrName} ${imgName}
  run_buildah run ${ctrName} -- stat -c "%a" my/bin
  expect_output "755"
}

@test "bud multi-stage COPY with invalid from statement" {
  _prefetch ubuntu
  imgName=ubuntu-image
  ctrName=ubuntu-copy
  run_buildah 125 build $WITH_POLICY_JSON -f $BUDFILES/copy-multistage-paths/Dockerfile.invalid_from -t ${imgName} $BUDFILES/copy-multistage-paths
  expect_output --substring "COPY only supports the --chmod=<permissions> --chown=<uid:gid> and the --from=<image\|stage> flags"
}

@test "bud COPY to root succeeds" {
  _prefetch ubuntu
  run_buildah build $WITH_POLICY_JSON $BUDFILES/copy-root
}

@test "bud with FROM AS construct" {
  _prefetch alpine
  run_buildah build $WITH_POLICY_JSON -t test1 $BUDFILES/from-as
  run_buildah images -a
  expect_output --substring "test1"

  run_buildah from --quiet $WITH_POLICY_JSON test1
  ctr=$output
  run_buildah containers -a
  expect_output --substring "test1"

  run_buildah inspect --format "{{.Docker.ContainerConfig.Env}}" --type image test1
  expect_output --substring "LOCAL=/1"
}

@test "bud with FROM AS construct with layers" {
  _prefetch alpine
  run_buildah build --layers $WITH_POLICY_JSON -t test1 $BUDFILES/from-as
  run_buildah images -a
  expect_output --substring "test1"

  run_buildah from --quiet $WITH_POLICY_JSON test1
  ctr=$output
  run_buildah containers -a
  expect_output --substring "test1"

  run_buildah inspect --format "{{.Docker.ContainerConfig.Env}}" --type image test1
  expect_output --substring "LOCAL=/1"
}

@test "bud with FROM AS skip FROM construct" {
  _prefetch alpine
  run_buildah build $WITH_POLICY_JSON -t test1 -f $BUDFILES/from-as/Dockerfile.skip $BUDFILES/from-as
  expect_output --substring "LOCAL=/1"
  expect_output --substring "LOCAL2=/2"

  run_buildah images -a
  expect_output --substring "test1"

  run_buildah from --quiet $WITH_POLICY_JSON test1
  ctr=$output
  run_buildah containers -a
  expect_output --substring "test1"

  run_buildah mount $ctr
  mnt=$output
  test   -e $mnt/1
  test ! -e $mnt/2

  run_buildah inspect --format "{{.Docker.ContainerConfig.Env}}" --type image test1
  expect_output "[PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin LOCAL=/1]"
}

@test "build with -f pointing to not a file should fail" {
  _prefetch alpine
  target=alpine-image
  run_buildah 125 build $WITH_POLICY_JSON -t ${target} -f $BUDFILES/dockerfile/
  expect_output --substring "cannot be path to a directory"
}

@test "bud with symlink Dockerfile not specified in file" {
  _prefetch alpine
  target=alpine-image
  run_buildah build $WITH_POLICY_JSON -t ${target} -f $BUDFILES/symlink/Dockerfile $BUDFILES/symlink
  expect_output --substring "FROM alpine"
}

@test "bud with dir for file but no Dockerfile in dir" {
  target=alpine-image
  run_buildah 125 build $WITH_POLICY_JSON -t ${target} -f $BUDFILES/empty-dir $BUDFILES/empty-dir
  expect_output --substring "no such file or directory"
}

@test "bud with bad dir Dockerfile" {
  target=alpine-image
  run_buildah 125 build $WITH_POLICY_JSON -t ${target} -f ${TEST_SOURCES}/baddirname ${TEST_SOURCES}/baddirname
  expect_output --substring "no such file or directory"
}

@test "bud with ARG before FROM default value" {
  _prefetch busybox
  target=leading-args-default
  run_buildah build $WITH_POLICY_JSON -t ${target} -f $BUDFILES/leading-args/Dockerfile $BUDFILES/leading-args
}

@test "bud with ARG before FROM" {
  _prefetch busybox:musl
  target=leading-args
  run_buildah build $WITH_POLICY_JSON -t ${target} --build-arg=VERSION=musl -f $BUDFILES/leading-args/Dockerfile $BUDFILES/leading-args

  #Verify https://github.com/containers/buildah/issues/4312
  # stage `FROM stage_${my_env}` must be resolved with default arg value and build should be successful.
  run_buildah build $WITH_POLICY_JSON -t source -f $BUDFILES/multi-stage-builds/Dockerfile.arg_in_stage

  #Verify https://github.com/containers/buildah/issues/4573
  # stage `COPY --from=stage_${my_env}` must be resolved with default arg value and build should be successful.
  run_buildah build $WITH_POLICY_JSON -t source -f $BUDFILES/multi-stage-builds/Dockerfile.arg_in_copy

}

@test "bud-with-healthcheck" {
  _prefetch alpine
  target=alpine-image
  run_buildah build $WITH_POLICY_JSON -t ${target} --format docker $BUDFILES/healthcheck
  run_buildah inspect -f '{{printf "%q" .Docker.Config.Healthcheck.Test}} {{printf "%d" .Docker.Config.Healthcheck.StartPeriod}} {{printf "%d" .Docker.Config.Healthcheck.Interval}} {{printf "%d" .Docker.Config.Healthcheck.Timeout}} {{printf "%d" .Docker.Config.Healthcheck.Retries}}' ${target}
  second=1000000000
  threeseconds=$(( 3 * $second ))
  fiveminutes=$(( 5 * 60 * $second ))
  tenminutes=$(( 10 * 60 * $second ))
  expect_output '["CMD-SHELL" "curl -f http://localhost/ || exit 1"]'" $tenminutes $fiveminutes $threeseconds 4" "Healthcheck config"
}

@test "bud with unused build arg" {
  _prefetch alpine busybox
  target=busybox-image
  run_buildah build $WITH_POLICY_JSON -t ${target} --build-arg foo=bar --build-arg foo2=bar2 -f $BUDFILES/build-arg/Dockerfile $BUDFILES/build-arg
  expect_output --substring "one or more build args were not consumed: \[foo2\]"
  run_buildah build $WITH_POLICY_JSON -t ${target} --build-arg IMAGE=alpine -f $BUDFILES/build-arg/Dockerfile2 $BUDFILES/build-arg
  assert "$output" !~ "one or more build args were not consumed: \[IMAGE\]"
  expect_output --substring "FROM alpine"
}

@test "bud with copy-from and cache" {
  _prefetch busybox
  target=busybox-image
  run_buildah build $WITH_POLICY_JSON --layers --iidfile ${TEST_SCRATCH_DIR}/iid1 -f $BUDFILES/copy-from/Dockerfile2 $BUDFILES/copy-from
  cat ${TEST_SCRATCH_DIR}/iid1
  test -s ${TEST_SCRATCH_DIR}/iid1
  run_buildah build $WITH_POLICY_JSON --layers --iidfile ${TEST_SCRATCH_DIR}/iid2 -f $BUDFILES/copy-from/Dockerfile2 $BUDFILES/copy-from
  cat ${TEST_SCRATCH_DIR}/iid2
  test -s ${TEST_SCRATCH_DIR}/iid2
  cmp ${TEST_SCRATCH_DIR}/iid1 ${TEST_SCRATCH_DIR}/iid2
}

@test "bud with copy-from in Dockerfile no prior FROM" {
  want_tag=20221018
  _prefetch busybox quay.io/libpod/testimage:$want_tag
  target=no-prior-from
  run_buildah build $WITH_POLICY_JSON -t ${target} -f $BUDFILES/copy-from/Dockerfile $BUDFILES/copy-from

  run_buildah from --quiet $WITH_POLICY_JSON ${target}
  ctr=$output
  run_buildah mount ${ctr}
  mnt=$output

  newfile="/home/busyboxpodman/copied-testimage-id"
  test -e $mnt/$newfile
  expect_output --from="$(< $mnt/$newfile)" "$want_tag" "Contents of $newfile"
}

@test "bud with copy-from with bad from flag in Dockerfile with --layers" {
  _prefetch busybox
  target=bad-from-flag
  run_buildah 125 build $WITH_POLICY_JSON --layers -t ${target} -f $BUDFILES/copy-from/Dockerfile.bad $BUDFILES/copy-from
  expect_output --substring "COPY only supports the --chmod=<permissions> --chown=<uid:gid> and the --from=<image\|stage> flags"
}

@test "bud with copy-from referencing the base image" {
  _prefetch busybox
  target=busybox-derived
  target_mt=busybox-mt-derived
  run_buildah build $WITH_POLICY_JSON -t ${target} -f $BUDFILES/copy-from/Dockerfile3 $BUDFILES/copy-from
  run_buildah build $WITH_POLICY_JSON --jobs 4 -t ${target} -f $BUDFILES/copy-from/Dockerfile3 $BUDFILES/copy-from

  run_buildah build $WITH_POLICY_JSON -t ${target} -f $BUDFILES/copy-from/Dockerfile4 $BUDFILES/copy-from
  run_buildah build --no-cache $WITH_POLICY_JSON --jobs 4 -t ${target_mt} -f $BUDFILES/copy-from/Dockerfile4 $BUDFILES/copy-from

  run_buildah from  --quiet ${target}
  cid=$output
  run_buildah mount ${cid}
  root_single_job=$output

  run_buildah from --quiet ${target_mt}
  cid=$output
  run_buildah mount ${cid}
  root_multi_job=$output

  # Check that both the version with --jobs 1 and --jobs=N have the same number of files
  test $(find $root_single_job -type f | wc -l) = $(find $root_multi_job -type f | wc -l)
}

@test "bud with copy-from referencing the current stage" {
  _prefetch busybox
  target=busybox-derived
  run_buildah 125 build $WITH_POLICY_JSON -t ${target} -f $BUDFILES/copy-from/Dockerfile2.bad $BUDFILES/copy-from
  expect_output --substring "COPY --from=build: no stage or image found with that name"
}

@test "bud-target" {
  _prefetch alpine ubuntu
  target=target
  run_buildah build $WITH_POLICY_JSON -t ${target} --target mytarget $BUDFILES/target
  expect_output --substring "\[1/2] STEP 1/3: FROM ubuntu:latest"
  expect_output --substring "\[2/2] STEP 1/3: FROM alpine:latest AS mytarget"
  run_buildah from --quiet ${target}
  cid=$output
  run_buildah mount ${cid}
  root=$output
  test   -e ${root}/2
  test ! -e ${root}/3
}

@test "bud-no-target-name" {
  _prefetch alpine
  run_buildah build $WITH_POLICY_JSON $BUDFILES/maintainer
}

@test "bud-multi-stage-nocache-nocommit" {
  _prefetch alpine
  # pull the base image directly, so that we don't record it being written to local storage in the next step
  run_buildah pull $WITH_POLICY_JSON alpine
  # okay, build an image with two stages
  run_buildah --log-level=debug bud $WITH_POLICY_JSON -f $BUDFILES/multi-stage-builds/Dockerfile.name $BUDFILES/multi-stage-builds
  # debug messages should only record us creating one new image: the one for the second stage, since we don't base anything on the first
  run grep "created new image ID" <<< "$output"
  expect_line_count 1
}

@test "bud-multi-stage-cache-nocontainer" {
  skip "FIXME: Broken in CI right now"
  _prefetch alpine
  # first time through, quite normal
  run_buildah build --layers -t base $WITH_POLICY_JSON -f $BUDFILES/multi-stage-builds/Dockerfile.rebase $BUDFILES/multi-stage-builds
  # second time through, everything should be cached, and we shouldn't create a container based on the final image
  run_buildah --log-level=debug bud --layers -t base $WITH_POLICY_JSON -f $BUDFILES/multi-stage-builds/Dockerfile.rebase $BUDFILES/multi-stage-builds
  # skip everything up through the final COMMIT step, and make sure we didn't log a "Container ID:" after it
  run sed '0,/COMMIT base/ d' <<< "$output"
  echo "$output" >&2
  test "${#lines[@]}" -gt 1
  run grep "Container ID:" <<< "$output"
  expect_output ""
}

@test "bud copy to symlink" {
  _prefetch alpine
  target=alpine-image
  ctr=alpine-ctr
  run_buildah build $WITH_POLICY_JSON -t ${target} $BUDFILES/dest-symlink
  expect_output --substring "STEP 5/6: RUN ln -s "

  run_buildah from $WITH_POLICY_JSON --name=${ctr} ${target}
  expect_output --substring ${ctr}

  run_buildah run ${ctr} ls -alF /etc/hbase
  expect_output --substring "/etc/hbase -> /usr/local/hbase/"

  run_buildah run ${ctr} ls -alF /usr/local/hbase
  expect_output --substring "Dockerfile"
}

@test "bud copy to dangling symlink" {
  _prefetch ubuntu
  target=ubuntu-image
  ctr=ubuntu-ctr
  run_buildah build $WITH_POLICY_JSON -t ${target} $BUDFILES/dest-symlink-dangling
  expect_output --substring "STEP 3/5: RUN ln -s "

  run_buildah from $WITH_POLICY_JSON --name=${ctr} ${target}
  expect_output --substring ${ctr}

  run_buildah run ${ctr} ls -alF /src
  expect_output --substring "/src -> /symlink"

  run_buildah run ${ctr} ls -alF /symlink
  expect_output --substring "Dockerfile"
}

@test "bud WORKDIR isa symlink" {
  _prefetch alpine
  target=alpine-image
  ctr=alpine-ctr
  run_buildah build $WITH_POLICY_JSON -t ${target} $BUDFILES/workdir-symlink
  expect_output --substring "STEP 3/6: RUN ln -sf "

  run_buildah from $WITH_POLICY_JSON --name=${ctr} ${target}
  expect_output --substring ${ctr}

  run_buildah run ${ctr} ls -alF /tempest
  expect_output --substring "/tempest -> /var/lib/tempest/"

  run_buildah run ${ctr} ls -alF /etc/notareal.conf
  expect_output --substring "\-rw\-rw\-r\-\-"
}

@test "bud WORKDIR isa symlink no target dir" {
  _prefetch alpine
  target=alpine-image
  ctr=alpine-ctr
  run_buildah build $WITH_POLICY_JSON -t ${target} -f Dockerfile-2 $BUDFILES/workdir-symlink
  expect_output --substring "STEP 2/6: RUN ln -sf "

  run_buildah from $WITH_POLICY_JSON --name=${ctr} ${target}
  expect_output --substring ${ctr}

  run_buildah run ${ctr} ls -alF /tempest
  expect_output --substring "/tempest -> /var/lib/tempest/"

  run_buildah run ${ctr} ls /tempest
  expect_output --substring "Dockerfile-2"

  run_buildah run ${ctr} ls -alF /etc/notareal.conf
  expect_output --substring "\-rw\-rw\-r\-\-"
}

@test "bud WORKDIR isa symlink no target dir and follow on dir" {
  _prefetch alpine
  target=alpine-image
  ctr=alpine-ctr
  run_buildah build $WITH_POLICY_JSON -t ${target} -f Dockerfile-3 $BUDFILES/workdir-symlink
  expect_output --substring "STEP 2/9: RUN ln -sf "

  run_buildah from $WITH_POLICY_JSON --name=${ctr} ${target}
  expect_output --substring ${ctr}

  run_buildah run ${ctr} ls -alF /tempest
  expect_output --substring "/tempest -> /var/lib/tempest/"

  run_buildah run ${ctr} ls /tempest
  expect_output --substring "Dockerfile-3"

  run_buildah run ${ctr} ls /tempest/lowerdir
  expect_output --substring "Dockerfile-3"

  run_buildah run ${ctr} ls -alF /etc/notareal.conf
  expect_output --substring "\-rw\-rw\-r\-\-"
}

@test "buildah bud --volume" {
  voldir=${TEST_SCRATCH_DIR}/bud-volume
  mkdir -p ${voldir}

  _prefetch alpine
  run_buildah build $WITH_POLICY_JSON -v ${voldir}:/testdir $BUDFILES/mount
  expect_output --substring "/testdir"
  run_buildah build $WITH_POLICY_JSON -v ${voldir}:/testdir:rw $BUDFILES/mount
  expect_output --substring "/testdir"
  run_buildah build $WITH_POLICY_JSON -v ${voldir}:/testdir:rw,z $BUDFILES/mount
  expect_output --substring "/testdir"
}

@test "bud-copy-dot with --layers picks up changed file" {
  _prefetch alpine
  local contextdir=${TEST_SCRATCH_DIR}/use-layers
  cp -a $BUDFILES/use-layers $contextdir

  mkdir -p $contextdir/subdir
  touch $contextdir/file.txt
  run_buildah build $WITH_POLICY_JSON --layers --iidfile ${TEST_SCRATCH_DIR}/iid1 -f Dockerfile.7 $contextdir

  touch $contextdir/file.txt
  run_buildah build $WITH_POLICY_JSON --layers --iidfile ${TEST_SCRATCH_DIR}/iid2 -f Dockerfile.7 $contextdir

  if [[ $(cat ${TEST_SCRATCH_DIR}/iid1) != $(cat ${TEST_SCRATCH_DIR}/iid2) ]]; then
    echo "Expected image id to not change after touching a file copied into the image" >&2
    false
  fi
}

@test "buildah-bud-policy" {
  target=foo

  # A deny-all policy should prevent us from pulling the base image.
  run_buildah 125 build --signature-policy ${TEST_SOURCES}/deny.json -t ${target} -v ${TEST_SOURCES}:/testdir $BUDFILES/mount
  expect_output --substring 'Source image rejected: Running image .* rejected by policy.'

  # A docker-only policy should allow us to pull the base image and commit.
  run_buildah build --signature-policy ${TEST_SOURCES}/docker.json -t ${target} -v ${TEST_SOURCES}:/testdir $BUDFILES/mount
  # A deny-all policy shouldn't break pushing, since policy is only evaluated
  # on the source image, and we force it to allow local storage.
  run_buildah push --signature-policy ${TEST_SOURCES}/deny.json ${target} dir:${TEST_SCRATCH_DIR}/mount
  run_buildah rmi ${target}

  # A docker-only policy should allow us to pull the base image first...
  run_buildah pull --signature-policy ${TEST_SOURCES}/docker.json alpine
  # ... and since we don't need to pull the base image, a deny-all policy shouldn't break a build.
  run_buildah build --signature-policy ${TEST_SOURCES}/deny.json -t ${target} -v ${TEST_SOURCES}:/testdir $BUDFILES/mount
  # A deny-all policy shouldn't break pushing, since policy is only evaluated
  # on the source image, and we force it to allow local storage.
  run_buildah push --signature-policy ${TEST_SOURCES}/deny.json ${target} dir:${TEST_SCRATCH_DIR}/mount
  # Similarly, a deny-all policy shouldn't break committing directly to other locations.
  run_buildah build --signature-policy ${TEST_SOURCES}/deny.json -t dir:${TEST_SCRATCH_DIR}/mount -v ${TEST_SOURCES}:/testdir $BUDFILES/mount
}

@test "bud-copy-replace-symlink" {
  local contextdir=${TEST_SCRATCH_DIR}/top
  mkdir -p $contextdir
  cp $BUDFILES/symlink/Dockerfile.replace-symlink $contextdir/
  ln -s Dockerfile.replace-symlink $contextdir/symlink
  echo foo > $contextdir/.dockerignore
  run_buildah build $WITH_POLICY_JSON -f $contextdir/Dockerfile.replace-symlink $contextdir
}

@test "bud-copy-recurse" {
  local contextdir=${TEST_SCRATCH_DIR}/recurse
  mkdir -p $contextdir
  cp $BUDFILES/recurse/Dockerfile $contextdir
  echo foo > $contextdir/.dockerignore
  run_buildah build $WITH_POLICY_JSON $contextdir
}

@test "bud copy with .dockerignore #1" {
  _prefetch alpine
  mytmpdir=${TEST_SCRATCH_DIR}/my-dir
  mkdir -p $mytmpdir/stuff/huge/usr/bin/
  touch $mytmpdir/stuff/huge/usr/bin/{file1,file2}
  touch $mytmpdir/stuff/huge/usr/file3

  cat > $mytmpdir/.dockerignore << _EOF
stuff/huge/*
!stuff/huge/usr/bin/*
_EOF

  cat > $mytmpdir/Containerfile << _EOF
FROM alpine
COPY stuff /tmp/stuff
RUN find /tmp/stuff -type f
_EOF

  run_buildah build -t testbud $WITH_POLICY_JSON ${mytmpdir}
  expect_output --substring "file1"
  expect_output --substring "file2"
  assert "$output" !~ "file3"
}

@test "bud copy with .dockerignore #2" {
  _prefetch alpine
  mytmpdir=${TEST_SCRATCH_DIR}/my-dir1
  mkdir -p $mytmpdir/stuff/huge/usr/bin/
  touch $mytmpdir/stuff/huge/usr/bin/{file1,file2}

  cat > $mytmpdir/.dockerignore << _EOF
stuff/huge/*
_EOF

  cat > $mytmpdir/Containerfile << _EOF
FROM alpine
COPY stuff /tmp/stuff
RUN find /tmp/stuff -type f
_EOF

  run_buildah build -t testbud $WITH_POLICY_JSON ${mytmpdir}
  assert "$output" !~ file1
  assert "$output" !~ file2
}

@test "bud-copy-workdir" {
  target=testimage
  run_buildah build $WITH_POLICY_JSON -t ${target} $BUDFILES/copy-workdir
  run_buildah from ${target}
  cid="$output"
  run_buildah mount "${cid}"
  root="$output"
  test -s "${root}"/file1.txt
  test -d "${root}"/subdir
  test -s "${root}"/subdir/file2.txt
}

# regression test for https://github.com/containers/podman/issues/10671
@test "bud-copy-workdir --layers" {
  _prefetch alpine

  target=testimage
  run_buildah build $WITH_POLICY_JSON --layers -t ${target} -f Dockerfile.2 $BUDFILES/copy-workdir
  run_buildah from ${target}
  cid="$output"
  run_buildah mount "${cid}"
  root="$output"
  test -d "${root}"/subdir
  test -s "${root}"/subdir/file1.txt
}

@test "bud-build-arg-cache" {
  _prefetch busybox alpine
  target=derived-image
  run_buildah build $WITH_POLICY_JSON --layers -t ${target} -f Dockerfile3 $BUDFILES/build-arg
  run_buildah inspect -f '{{.FromImageID}}' ${target}
  targetid="$output"

  # With build args, we should not find the previous build as a cached result. This will be true because there is a RUN command after all the ARG
  # commands in the containerfile, so this does not truly test if the ARG commands were using cache or not. There is a test for that case below.
  run_buildah build $WITH_POLICY_JSON --layers -t ${target} -f Dockerfile3 --build-arg=UID=17122 --build-arg=CODE=/copr/coprs_frontend --build-arg=USERNAME=praiskup --build-arg=PGDATA=/pgdata $BUDFILES/build-arg
  run_buildah inspect -f '{{.FromImageID}}' ${target}
  argsid="$output"
  assert "$argsid" != "$initialid" \
         ".FromImageID of test-img-2 ($argsid) == same as test-img, it should be different"

  # With build args, even in a different order, we should end up using the previous build as a cached result.
  run_buildah build $WITH_POLICY_JSON --layers -t ${target} -f Dockerfile3 --build-arg=UID=17122 --build-arg=CODE=/copr/coprs_frontend --build-arg=USERNAME=praiskup --build-arg=PGDATA=/pgdata $BUDFILES/build-arg
  run_buildah inspect -f '{{.FromImageID}}' ${target}
  expect_output "$argsid" "FromImageID of build 3"

  run_buildah build $WITH_POLICY_JSON --layers -t ${target} -f Dockerfile3 --build-arg=CODE=/copr/coprs_frontend --build-arg=USERNAME=praiskup --build-arg=PGDATA=/pgdata --build-arg=UID=17122 $BUDFILES/build-arg
  run_buildah inspect -f '{{.FromImageID}}' ${target}
  expect_output "$argsid" "FromImageID of build 4"

  run_buildah build $WITH_POLICY_JSON --layers -t ${target} -f Dockerfile3 --build-arg=USERNAME=praiskup --build-arg=PGDATA=/pgdata --build-arg=UID=17122 --build-arg=CODE=/copr/coprs_frontend $BUDFILES/build-arg
  run_buildah inspect -f '{{.FromImageID}}' ${target}
  expect_output "$argsid" "FromImageID of build 5"

  run_buildah build $WITH_POLICY_JSON --layers -t ${target} -f Dockerfile3 --build-arg=PGDATA=/pgdata --build-arg=UID=17122 --build-arg=CODE=/copr/coprs_frontend --build-arg=USERNAME=praiskup $BUDFILES/build-arg
  run_buildah inspect -f '{{.FromImageID}}' ${target}
  expect_output "$argsid" "FromImageID of build 6"

  # If build-arg is specified via the command line and is different from the previous cached build, it should not use the cached layers.
  # Note, this containerfile does not have any RUN commands and we verify that the ARG steps are being rebuilt when a change is detected.
  run_buildah build $WITH_POLICY_JSON --layers -t test-img -f Dockerfile4 $BUDFILES/build-arg
  run_buildah inspect -f '{{.FromImageID}}' test-img
  initialid="$output"

  # Build the same containerfile again and verify that the cached layers were used
  run_buildah build $WITH_POLICY_JSON --layers -t test-img-1 -f Dockerfile4 $BUDFILES/build-arg
  run_buildah inspect -f '{{.FromImageID}}' test-img-1
  expect_output "$initialid" "FromImageID of test-img-1 should match test-img"

  # Set the build-arg flag and verify that the cached layers are not used
  run_buildah build $WITH_POLICY_JSON --layers -t test-img-2 --build-arg TEST=foo -f Dockerfile4 $BUDFILES/build-arg
  run_buildah inspect -f '{{.FromImageID}}' test-img-2
  argsid="$output"
  assert "$argsid" != "$initialid" \
         ".FromImageID of test-img-2 ($argsid) == same as test-img, it should be different"

  # Set the build-arg via an ENV in the local environment and verify that the cached layers are not used
  export TEST=bar
  run_buildah build $WITH_POLICY_JSON --layers -t test-img-3 --build-arg TEST -f Dockerfile4 $BUDFILES/build-arg
  run_buildah inspect -f '{{.FromImageID}}' test-img-3
  argsid="$output"
  assert "$argsid" != "$initialid" \
         ".FromImageID of test-img-3 ($argsid) == same as test-img, it should be different"
}

@test "bud test RUN with a privileged command" {
  _prefetch alpine
  target=alpinepriv
  run_buildah build $WITH_POLICY_JSON -t ${target} -f $BUDFILES/run-privd/Dockerfile $BUDFILES/run-privd
  expect_output --substring "[^:][^[:graph:]]COMMIT ${target}"
  run_buildah images -q
  expect_line_count 2
}

@test "bud-copy-dockerignore-hardlinks" {
  target=image
  local contextdir=${TEST_SCRATCH_DIR}/hardlinks
  mkdir -p $contextdir/subdir
  cp $BUDFILES/recurse/Dockerfile $contextdir
  echo foo > $contextdir/.dockerignore
  echo test1 > $contextdir/subdir/test1.txt
  ln $contextdir/subdir/test1.txt $contextdir/subdir/test2.txt
  ln $contextdir/subdir/test2.txt $contextdir/test3.txt
  ln $contextdir/test3.txt $contextdir/test4.txt
  run_buildah build $WITH_POLICY_JSON -t ${target} $contextdir
  run_buildah from ${target}
  ctrid="$output"
  run_buildah mount "$ctrid"
  root="$output"

  run stat -c "%d:%i" ${root}/subdir/test1.txt
  id1=$output
  run stat -c "%h" ${root}/subdir/test1.txt
  expect_output 4 "test1: number of hardlinks"
  run stat -c "%d:%i" ${root}/subdir/test2.txt
  expect_output $id1 "stat(test2) == stat(test1)"
  run stat -c "%h" ${root}/subdir/test2.txt
  expect_output 4 "test2: number of hardlinks"
  run stat -c "%d:%i" ${root}/test3.txt
  expect_output $id1 "stat(test3) == stat(test1)"
  run stat -c "%h" ${root}/test3.txt
  expect_output 4 "test3: number of hardlinks"
  run stat -c "%d:%i" ${root}/test4.txt
  expect_output $id1 "stat(test4) == stat(test1)"
  run stat -c "%h" ${root}/test4.txt
  expect_output 4 "test4: number of hardlinks"
}

@test "bud without any arguments should succeed" {
  cd $BUDFILES/from-scratch
  run_buildah build --signature-policy ${TEST_SOURCES}/policy.json
}

@test "bud without any arguments should fail when no Dockerfile exists" {
  cd $TEST_SCRATCH_DIR
  run_buildah 125 build --signature-policy ${TEST_SOURCES}/policy.json
  expect_output --substring "no such file or directory"
}

@test "bud with specified context should fail if directory contains no Dockerfile" {
  run_buildah 125 build $WITH_POLICY_JSON "$TEST_SCRATCH_DIR"
  expect_output --substring "no such file or directory"
}

@test "bud with specified context should fail if assumed Dockerfile is a directory" {
  mkdir -p "$TEST_SCRATCH_DIR"/Dockerfile
  run_buildah 125 build $WITH_POLICY_JSON "$TEST_SCRATCH_DIR"
  expect_output --substring "is not a file"
}

@test "bud with specified context should fail if context contains not-existing Dockerfile" {
  run_buildah 125 build $WITH_POLICY_JSON "$TEST_SCRATCH_DIR"/Dockerfile
  expect_output --substring "no such file or directory"
}

@test "bud with specified context should succeed if context contains existing Dockerfile" {
  echo "FROM alpine" > $TEST_SCRATCH_DIR/Dockerfile
  run_buildah bud $WITH_POLICY_JSON $TEST_SCRATCH_DIR/Dockerfile
}

@test "bud with specified context should fail if context contains empty Dockerfile" {
  touch $TEST_SCRATCH_DIR/Dockerfile
  run_buildah 125 build $WITH_POLICY_JSON $TEST_SCRATCH_DIR/Dockerfile
  expect_output --substring "no contents in \"$TEST_SCRATCH_DIR/Dockerfile\""
}

@test "bud-no-change" {
  _prefetch alpine
  parent=alpine
  target=no-change-image
  run_buildah build $WITH_POLICY_JSON -t ${target} $BUDFILES/no-change
  run_buildah inspect --format '{{printf "%q" .FromImageDigest}}' ${parent}
  parentid="$output"
  run_buildah inspect --format '{{printf "%q" .FromImageDigest}}' ${target}
  expect_output "$parentid"
}

@test "bud-no-change-label" {
  run_buildah --version
  local -a output_fields=($output)
  buildah_version=${output_fields[2]}
  want_output='map["io.buildah.version":"'$buildah_version'" "test":"label"]'

  _prefetch alpine
  parent=alpine
  target=no-change-image
  run_buildah build --label "test=label" $WITH_POLICY_JSON -t ${target} $BUDFILES/no-change
  run_buildah inspect --format '{{printf "%q" .Docker.Config.Labels}}' ${target}
  expect_output "$want_output"
}

@test "bud-no-change-annotation" {
  _prefetch alpine
  target=no-change-image
  run_buildah build --annotation "test=annotation" $WITH_POLICY_JSON -t ${target} $BUDFILES/no-change
  run_buildah inspect --format '{{index .ImageAnnotations "test"}}' ${target}
  expect_output "annotation"
}

@test "bud-squash-layers" {
  _prefetch alpine
  run_buildah build $WITH_POLICY_JSON --squash $BUDFILES/layers-squash
}

@test "bud-squash-hardlinks" {
  _prefetch busybox
  run_buildah build $WITH_POLICY_JSON --squash $BUDFILES/layers-squash/Dockerfile.hardlinks
}

# Following test must pass for both rootless and rootfull
@test "rootless: support --device and renaming device using bind-mount" {
  skip_if_in_container # unable to perform mount of /dev/null for test in CI container setup
  local contextdir=${TEST_SCRATCH_DIR}/bud/platform
  mkdir -p $contextdir

  cat > $contextdir/Dockerfile << _EOF
FROM alpine
RUN ls /test/dev
_EOF
  run_buildah build $WITH_POLICY_JSON --device /dev/null:/test/dev/null  -t test -f $contextdir/Dockerfile
  expect_output --substring "null"
}

@test "bud with additional directory of devices" {
  skip_if_rootless_environment
  skip_if_chroot
  skip_if_rootless

  _prefetch alpine
  target=alpine-image
  local contextdir=${TEST_SCRATCH_DIR}/foo
  mkdir -p $contextdir
  mknod $contextdir/null c 1 3
  run_buildah build $WITH_POLICY_JSON --device $contextdir:/dev/fuse  -t ${target} -f $BUDFILES/device/Dockerfile $BUDFILES/device
  expect_output --substring "null"
}

@test "bud with additional device" {
  _prefetch alpine
  target=alpine-image
  run_buildah build $WITH_POLICY_JSON --device /dev/fuse -t ${target} -f $BUDFILES/device/Dockerfile $BUDFILES/device
  expect_output --substring "/dev/fuse"
}

@test "bud with Containerfile" {
  _prefetch alpine
  target=alpine-image
  run_buildah build $WITH_POLICY_JSON -t ${target} $BUDFILES/containerfile
  expect_output --substring "FROM alpine"
}

@test "bud with Containerfile.in, --cpp-flag" {
  _prefetch alpine
  target=alpine-image
  run_buildah build $WITH_POLICY_JSON -t ${target} -f $BUDFILES/containerfile/Containerfile.in $BUDFILES/containerfile
  expect_output --substring "Ignoring In file included .* invalid preprocessing directive #This"
  expect_output --substring "FROM alpine"
  expect_output --substring "success"
  expect_output --substring "debug=no" "with no cpp-flag or BUILDAH_CPPFLAGS"

  run_buildah build $WITH_POLICY_JSON -t ${target} --cpp-flag "-DTESTCPPDEBUG" -f $BUDFILES/containerfile/Containerfile.in $BUDFILES/containerfile
  expect_output --substring "Ignoring In file included .* invalid preprocessing directive #This"
  expect_output --substring "FROM alpine"
  expect_output --substring "success"
  expect_output --substring "debug=yes" "with --cpp-flag -DTESTCPPDEBUG"
}

@test "bud with Containerfile.in, via envariable" {
  _prefetch alpine
  target=alpine-image

  BUILDAH_CPPFLAGS="-DTESTCPPDEBUG" run_buildah build $WITH_POLICY_JSON -t ${target} -f $BUDFILES/containerfile/Containerfile.in $BUDFILES/containerfile
  expect_output --substring "Ignoring In file included .* invalid preprocessing directive #This"
  expect_output --substring "FROM alpine"
  expect_output --substring "success"
  expect_output --substring "debug=yes" "with BUILDAH_CPPFLAGS=-DTESTCPPDEBUG"
}

@test "bud with Dockerfile" {
  _prefetch alpine
  target=alpine-image
  run_buildah build $WITH_POLICY_JSON -t ${target} $BUDFILES/dockerfile
  expect_output --substring "FROM alpine"
}

@test "bud with Containerfile and Dockerfile" {
  _prefetch alpine
  target=alpine-image
  run_buildah build $WITH_POLICY_JSON -t ${target} $BUDFILES/containeranddockerfile
  expect_output --substring "FROM alpine"
}

@test "bud-http-context-with-Containerfile" {
  _test_http http-context-containerfile context.tar
}

@test "bud with Dockerfile from stdin" {
  _prefetch alpine
  target=df-stdin
  run_buildah build $WITH_POLICY_JSON -t ${target} - < $BUDFILES/context-from-stdin/Dockerfile
  run_buildah from --quiet ${target}
  cid=$output
  run_buildah mount ${cid}
  root=$output

  test -s $root/scratchfile
  run cat $root/scratchfile
  expect_output "stdin-context" "contents of \$root/scratchfile"

  # FROM scratch overrides FROM alpine
  test ! -s $root/etc/alpine-release
}

@test "bud with Dockerfile from stdin tar" {
  _prefetch alpine
  target=df-stdin
  # 'cmd1 < <(cmd2)' == 'cmd2 | cmd1' but runs cmd1 in this shell, not sub.
  run_buildah build $WITH_POLICY_JSON -t ${target} - < <(tar -c -C $BUDFILES/context-from-stdin .)
  run_buildah from --quiet ${target}
  cid=$output
  run_buildah mount ${cid}
  root=$output

  test -s $root/scratchfile
  run cat $root/scratchfile
  expect_output "stdin-context" "contents of \$root/scratchfile"

  # FROM scratch overrides FROM alpine
  test ! -s $root/etc/alpine-release
}

@test "bud containerfile with args" {
  _prefetch alpine
  target=use-args
  touch $BUDFILES/use-args/abc.txt
  run_buildah build $WITH_POLICY_JSON -t ${target} --build-arg=abc.txt $BUDFILES/use-args
  expect_output --substring "COMMIT use-args"
  run_buildah from --quiet ${target}
  ctrID=$output
  run_buildah run $ctrID ls abc.txt
  expect_output --substring "abc.txt"

  run_buildah build $WITH_POLICY_JSON -t ${target} -f Containerfile.destination --build-arg=testArg=abc.txt --build-arg=destination=/tmp $BUDFILES/use-args
  expect_output --substring "COMMIT use-args"
  run_buildah from --quiet ${target}
  ctrID=$output
  run_buildah run $ctrID ls /tmp/abc.txt
  expect_output --substring "abc.txt"

  run_buildah build $WITH_POLICY_JSON -t ${target} -f Containerfile.dest_nobrace --build-arg=testArg=abc.txt --build-arg=destination=/tmp $BUDFILES/use-args
  expect_output --substring "COMMIT use-args"
  run_buildah from --quiet ${target}
  ctrID=$output
  run_buildah run $ctrID ls /tmp/abc.txt
  expect_output --substring "abc.txt"

  rm $BUDFILES/use-args/abc.txt
}

@test "bud using gitrepo and branch" {
  if ! start_git_daemon ${TEST_SOURCES}/git-daemon/release-1.11-rhel.tar.gz ; then
    skip "error running git daemon"
  fi
  run_buildah build $WITH_POLICY_JSON --layers -t gittarget -f $BUDFILES/shell/Dockerfile git://localhost:${GITPORT}/repo#release-1.11-rhel
}

@test "bud using gitrepo with .git and branch" {
  run_buildah build $WITH_POLICY_JSON --layers -t gittarget -f $BUDFILES/shell/Dockerfile https://github.com/containers/buildah.git#release-1.11-rhel
}

# Fixes #1906: buildah was not detecting changed tarfile
@test "bud containerfile with tar archive in copy" {
  _prefetch busybox
  # First check to verify cache is used if the tar file does not change
  target=copy-archive
  date > $BUDFILES/${target}/test
  tar -C $TEST_SOURCES -cJf $BUDFILES/${target}/test.tar.xz bud/${target}/test
  run_buildah build $WITH_POLICY_JSON --layers -t ${target} $BUDFILES/${target}
  expect_output --substring "COMMIT copy-archive"

  run_buildah build $WITH_POLICY_JSON --layers -t ${target} $BUDFILES/${target}
  expect_output --substring " Using cache"
  expect_output --substring "COMMIT copy-archive"

  # Now test that we do NOT use cache if the tar file changes
  echo This is a change >> $BUDFILES/${target}/test
  tar -C $TEST_SOURCES -cJf $BUDFILES/${target}/test.tar.xz bud/${target}/test
  run_buildah build $WITH_POLICY_JSON --layers -t ${target} $BUDFILES/${target}
  if [[ "$output" =~ " Using cache" ]]; then
      expect_output "[no instance of 'Using cache']" "no cache used"
  fi
  expect_output --substring "COMMIT copy-archive"

  rm -f $BUDFILES/${target}/test*
}

@test "bud pull never" {
  target=pull
  run_buildah 125 build $WITH_POLICY_JSON -t ${target} --pull-never $BUDFILES/pull
  expect_output --substring "busybox: image not known"

  run_buildah build $WITH_POLICY_JSON -t ${target} --pull $BUDFILES/pull
  expect_output --substring "COMMIT pull"

  run_buildah build $WITH_POLICY_JSON -t ${target} --pull=never $BUDFILES/pull
  expect_output --substring "COMMIT pull"
}

@test "bud pull false no local image" {
  target=pull
  run_buildah build $WITH_POLICY_JSON -t ${target} --pull=false $BUDFILES/pull
  expect_output --substring "COMMIT pull"
}

@test "bud with Containerfile should fail with nonexistent authfile" {
  target=alpine-image
  run_buildah 125 build --authfile /tmp/nonexistent $WITH_POLICY_JSON -t ${target} $BUDFILES/containerfile
  expect_output "Error: credential file is not accessible: stat /tmp/nonexistent: no such file or directory"
}


@test "bud for multi-stage Containerfile with invalid registry and --authfile as a fd, should fail with no such host" {
  target=alpine-multi-stage-image
  run_buildah 125 build --authfile=<(echo "{ \"auths\": { \"myrepository.example\": { \"auth\": \"$(echo 'username:password' | base64 --wrap=0)\" } } }") -t ${target} --file $BUDFILES/from-invalid-registry/Containerfile
  # Should fail with `no such host` instead of: error reading JSON file "/dev/fd/x"
  expect_output --substring "no such host"
}

@test "bud COPY with URL should fail" {
  local contextdir=${TEST_SCRATCH_DIR}/budcopy
  mkdir $contextdir
  FILE=$contextdir/Dockerfile.url
  /bin/cat <<EOM >$FILE
FROM alpine:latest
COPY https://getfedora.org/index.html .
EOM

  run_buildah 125 build $WITH_POLICY_JSON -t foo -f $contextdir/Dockerfile.url
  expect_output --substring "building .* source can.t be a URL for COPY"
}

@test "bud quiet" {
  _prefetch alpine
  run_buildah build --format docker -t quiet-test $WITH_POLICY_JSON -q $BUDFILES/shell
  expect_line_count 1
  expect_output --substring '^[0-9a-f]{64}$'
}

@test "bud COPY with Env Var in Containerfile" {
  _prefetch alpine
  run_buildah build $WITH_POLICY_JSON -t testctr $BUDFILES/copy-envvar
  run_buildah from testctr
  run_buildah run testctr-working-container ls /file-0.0.1.txt
  run_buildah rm -a

  run_buildah build $WITH_POLICY_JSON --layers -t testctr $BUDFILES/copy-envvar
  run_buildah from testctr
  run_buildah run testctr-working-container ls /file-0.0.1.txt
  run_buildah rm -a
}

@test "bud with custom arch" {
  run_buildah build $WITH_POLICY_JSON \
    -f $BUDFILES/from-scratch/Containerfile \
    -t arch-test \
    --arch=arm

  run_buildah inspect --format "{{ .Docker.Architecture }}" arch-test
  expect_output arm

  run_buildah inspect --format "{{ .OCIv1.Architecture }}" arch-test
  expect_output arm
}

@test "bud with custom os" {
  run_buildah build $WITH_POLICY_JSON \
    -f $BUDFILES/from-scratch/Containerfile \
    -t os-test \
    --os=windows

  run_buildah inspect --format "{{ .Docker.OS }}" os-test
  expect_output windows

  run_buildah inspect --format "{{ .OCIv1.OS }}" os-test
  expect_output windows
}

@test "bud with custom os-version" {
  run_buildah build $WITH_POLICY_JSON \
    -f $BUDFILES/from-scratch/Containerfile \
    -t os-version-test \
    --os-version=1.0

  run_buildah inspect --format "{{ .Docker.OSVersion }}" os-version-test
  expect_output 1.0

  run_buildah inspect --format "{{ .OCIv1.OSVersion }}" os-version-test
  expect_output 1.0
}

@test "bud with custom os-features" {
  run_buildah build $WITH_POLICY_JSON \
    -f $BUDFILES/from-scratch/Containerfile \
    -t os-features-test \
    --os-feature removed --os-feature removed- --os-feature win32k

  run_buildah inspect --format "{{ .Docker.OSFeatures }}" os-features-test
  expect_output '[win32k]'

  run_buildah inspect --format "{{ .OCIv1.OSFeatures }}" os-features-test
  expect_output '[win32k]'
}

@test "bud with custom platform" {
  run_buildah build $WITH_POLICY_JSON \
    -f $BUDFILES/from-scratch/Containerfile \
    -t platform-test \
    --platform=windows/arm

  run_buildah inspect --format "{{ .Docker.OS }}" platform-test
  expect_output windows

  run_buildah inspect --format "{{ .OCIv1.OS }}" platform-test
  expect_output windows

  run_buildah inspect --format "{{ .Docker.Architecture }}" platform-test
  expect_output arm

  run_buildah inspect --format "{{ .OCIv1.Architecture }}" platform-test
  expect_output arm
}

@test "bud with custom platform and empty os or arch" {
  run_buildah build $WITH_POLICY_JSON \
    -f $BUDFILES/from-scratch/Containerfile \
    -t platform-test \
    --platform=windows/

  run_buildah inspect --format "{{ .Docker.OS }}" platform-test
  expect_output windows

  run_buildah inspect --format "{{ .OCIv1.OS }}" platform-test
  expect_output windows

  run_buildah build $WITH_POLICY_JSON \
    -f $BUDFILES/from-scratch/Containerfile \
    -t platform-test2 \
    --platform=/arm

  run_buildah inspect --format "{{ .Docker.Architecture }}" platform-test2
  expect_output arm

  run_buildah inspect --format "{{ .OCIv1.Architecture }}" platform-test2
  expect_output arm
}

@test "bud Add with linked tarball" {
  _prefetch alpine
  run_buildah build $WITH_POLICY_JSON -f $BUDFILES/symlink/Containerfile.add-tar-with-link -t testctr $BUDFILES/symlink
  run_buildah from testctr
  run_buildah run testctr-working-container ls /tmp/testdir/testfile.txt
  run_buildah rm -a
  run_buildah rmi -a -f

  _prefetch alpine
  run_buildah build $WITH_POLICY_JSON -f $BUDFILES/symlink/Containerfile.add-tar-gz-with-link -t testctr $BUDFILES/symlink
  run_buildah from testctr
  run_buildah run testctr-working-container ls /tmp/testdir/testfile.txt
  run_buildah rm -a
  run_buildah rmi -a -f
}

@test "bud file above context directory" {
  run_buildah 125 build $WITH_POLICY_JSON -t testctr $BUDFILES/context-escape-dir/testdir
  expect_output --substring "escaping context directory error"
}

@test "bud-multi-stage-args-scope" {
  _prefetch alpine
  run_buildah build $WITH_POLICY_JSON --layers -t multi-stage-args --build-arg SECRET=secretthings -f Dockerfile.arg $BUDFILES/multi-stage-builds
  run_buildah from --name test-container multi-stage-args
  run_buildah run test-container -- cat test_file
  expect_output ""
}

@test "bud-multi-stage-args-history" {
  _prefetch alpine
  run_buildah build $WITH_POLICY_JSON --layers -t multi-stage-args --build-arg SECRET=secretthings -f Dockerfile.arg $BUDFILES/multi-stage-builds
  run_buildah inspect --format '{{range .History}}{{println .CreatedBy}}{{end}}' multi-stage-args
  run grep "secretthings" <<< "$output"
  expect_output ""

  run_buildah inspect --format '{{range .OCIv1.History}}{{println .CreatedBy}}{{end}}' multi-stage-args
  run grep "secretthings" <<< "$output"
  expect_output ""

  run_buildah inspect --format '{{range .Docker.History}}{{println .CreatedBy}}{{end}}' multi-stage-args
  run grep "secretthings" <<< "$output"
  expect_output ""
}

@test "bud with encrypted FROM image" {
  _prefetch busybox
  local contextdir=${TEST_SCRATCH_DIR}/tmp
  mkdir $contextdir
  openssl genrsa -out $contextdir/mykey.pem 1024
  openssl genrsa -out $contextdir/mykey2.pem 1024
  openssl rsa -in $contextdir/mykey.pem -pubout > $contextdir/mykey.pub
  start_registry
  run_buildah push $WITH_POLICY_JSON --tls-verify=false --creds testuser:testpassword --encryption-key jwe:$contextdir/mykey.pub busybox docker://localhost:${REGISTRY_PORT}/buildah/busybox_encrypted:latest

  target=busybox-image
  echo FROM localhost:${REGISTRY_PORT}/buildah/busybox_encrypted:latest > $contextdir/Dockerfile

  # Try to build from encrypted image without key
  run_buildah 1 build $WITH_POLICY_JSON --tls-verify=false  --creds testuser:testpassword -t ${target} -f $contextdir/Dockerfile
  assert "$output" =~ "archive/tar: invalid tar header"

  # Try to build from encrypted image with wrong key
  run_buildah 125 build $WITH_POLICY_JSON --tls-verify=false  --creds testuser:testpassword --decryption-key $contextdir/mykey2.pem -t ${target} -f $contextdir/Dockerfile
  assert "$output" =~ "no suitable key found for decrypting layer key"
  assert "$output" =~ "- JWE: No suitable private key found for decryption"

  # Try to build with the correct key
  run_buildah build $WITH_POLICY_JSON --tls-verify=false  --creds testuser:testpassword --decryption-key $contextdir/mykey.pem -t ${target} -f $contextdir/Dockerfile
  assert "$output" =~ "Successfully tagged localhost:$REGISTRY_PORT/"

  rm -rf $contextdir
}

@test "bud with --build-arg" {
  _prefetch alpine busybox
  target=busybox-image

  # Envariable not present at all
  run_buildah --log-level "warn" bud $WITH_POLICY_JSON -t ${target} $BUDFILES/build-arg
  expect_output --substring 'missing \\"foo\\" build argument. Try adding'

  # Envariable explicitly set on command line
  run_buildah build $WITH_POLICY_JSON -t ${target} --build-arg foo=bar $BUDFILES/build-arg
  assert "${lines[3]}" = "bar"

  # Envariable from environment
  export foo=$(random_string 20)
  run_buildah build $WITH_POLICY_JSON -t ${target} --build-arg foo $BUDFILES/build-arg
  assert "${lines[3]}" = "$foo"
}

@test "bud arg and env var with same name" {
  # Regression test for https://github.com/containers/buildah/issues/2345
  run_buildah build $WITH_POLICY_JSON -t testctr $BUDFILES/dupe-arg-env-name
  expect_output --substring "https://example.org/bar"
}

@test "bud copy chown with newuser" {
  # Regression test for https://github.com/containers/buildah/issues/2192
  run_buildah build $WITH_POLICY_JSON -t testctr -f $BUDFILES/copy-chown/Containerfile.chown_user $BUDFILES/copy-chown
  expect_output --substring "myuser myuser"
}

@test "bud-builder-identity" {
  _prefetch alpine
  parent=alpine
  target=no-change-image
  run_buildah build $WITH_POLICY_JSON -t ${target} $BUDFILES/from-scratch
  run_buildah --version
  local -a output_fields=($output)
  buildah_version=${output_fields[2]}

  run_buildah inspect --format '{{ index .Docker.Config.Labels "io.buildah.version"}}' $target
  expect_output "$buildah_version"
}

@test "run check --from with arg" {
  skip_if_no_runtime

  ${OCI} --version
  _prefetch alpine
  _prefetch debian

  run_buildah build --build-arg base=alpine --build-arg toolchainname=busybox --build-arg destinationpath=/tmp --pull=false $WITH_POLICY_JSON -f $BUDFILES/from-with-arg/Containerfile .
  expect_output --substring "FROM alpine"
  expect_output --substring 'STEP 4/4: COPY --from=\$\{toolchainname\} \/ \$\{destinationpath\}'
  run_buildah rm -a
}

@test "bud preserve rootfs for --mount=type=bind,from=" {
  _prefetch alpine
  run_buildah build --build-arg NONCE="$(date)" --layers --pull=false $WITH_POLICY_JSON -f Dockerfile.3 $BUDFILES/cache-stages
  expect_output --substring "Worked"
}

@test "bud timestamp" {
  _prefetch alpine
  timestamp=40
  run_buildah build --timestamp=${timestamp} --quiet --pull=false $WITH_POLICY_JSON -t timestamp -f Dockerfile.1 $BUDFILES/cache-stages
  cid=$output
  run_buildah inspect --format '{{ .Docker.Created }}' timestamp
  expect_output --substring "1970-01-01"
  run_buildah inspect --format '{{ .OCIv1.Created }}' timestamp
  expect_output --substring "1970-01-01"
  run_buildah inspect --format '{{ .History }}' timestamp
  expect_output --substring "1970-01-01 00:00:${timestamp}"

  run_buildah from --quiet --pull=false $WITH_POLICY_JSON timestamp
  cid=$output
  run_buildah run $cid ls -l /tmpfile
  expect_output --substring "1970"

  run_buildah images --format "{{.Created}}" timestamp
  expect_output ${timestamp}

  rm -rf ${TEST_SCRATCH_DIR}/tmp
}

@test "bud timestamp compare" {
  _prefetch alpine
  TIMESTAMP=$(date '+%s')
  run_buildah build --timestamp=${TIMESTAMP} --quiet --pull=false $WITH_POLICY_JSON -t timestamp -f Dockerfile.1 $BUDFILES/cache-stages
  cid=$output

  run_buildah images --format "{{.Created}}" timestamp
  expect_output ${timestamp}

  run_buildah build --timestamp=${TIMESTAMP} --quiet --pull=false $WITH_POLICY_JSON -t timestamp -f Dockerfile.1 $BUDFILES/cache-stages
  expect_output "$cid"

  rm -rf ${TEST_SCRATCH_DIR}/tmp
}

@test "bud with-rusage" {
  _prefetch alpine
  run_buildah build --log-rusage --layers --pull=false --format docker $WITH_POLICY_JSON $BUDFILES/shell
  cid=$output
  # expect something that looks like it was formatted using pkg/rusage.FormatDiff()
  expect_output --substring ".*\(system\).*\(user\).*\(elapsed\).*input.*output"
}

@test "bud with-rusage-logfile" {
  _prefetch alpine
  run_buildah build --log-rusage --rusage-logfile ${TEST_SCRATCH_DIR}/foo.log --layers --pull=false --format docker $WITH_POLICY_JSON $BUDFILES/shell
  # the logfile should exist
  if [ ! -e ${TEST_SCRATCH_DIR}/foo.log ]; then die "rusage-logfile foo.log did not get created!"; fi
  # expect that foo.log only contains lines that were formatted using pkg/rusage.FormatDiff()
  formatted_lines=$(grep ".*\(system\).*\(user\).*\(elapsed\).*input.*output" ${TEST_SCRATCH_DIR}/foo.log | wc -l)
  line_count=$(wc -l <${TEST_SCRATCH_DIR}/foo.log)
  if [[ "$formatted_lines" -ne "$line_count" ]]; then
      die "Got ${formatted_lines} lines formatted with pkg/rusage.FormatDiff() but rusage-logfile has ${line_count} lines"
  fi
}

@test "bud-caching-from-scratch" {
  _prefetch alpine
  # run the build once
  run_buildah build --quiet --layers --pull=false --format docker $WITH_POLICY_JSON $BUDFILES/cache-scratch
  iid="$output"

  # now run it again - the cache should give us the same final image ID
  run_buildah build --quiet --layers --pull=false --format docker $WITH_POLICY_JSON $BUDFILES/cache-scratch
  assert "$output" = "$iid"

  # now run it *again*, except with more content added at an intermediate step, which should invalidate the cache
  run_buildah build --quiet --layers --pull=false --format docker $WITH_POLICY_JSON -f Dockerfile.different1 $BUDFILES/cache-scratch
  assert "$output" !~ "$iid"

  # now run it *again* again, except with more content added at an intermediate step, which should invalidate the cache
  run_buildah build --quiet --layers --pull=false --format docker $WITH_POLICY_JSON -f Dockerfile.different2 $BUDFILES/cache-scratch
  assert "$output" !~ "$iid"
}

@test "bud-caching-from-scratch-config" {
  _prefetch alpine
  # run the build once
  run_buildah build --quiet --layers --pull=false --format docker $WITH_POLICY_JSON -f Dockerfile.config $BUDFILES/cache-scratch
  iid="$output"

  # now run it again - the cache should give us the same final image ID
  run_buildah build --quiet --layers --pull=false --format docker $WITH_POLICY_JSON -f Dockerfile.config $BUDFILES/cache-scratch
  assert "$output" = "$iid"

  # now run it *again*, except with more content added at an intermediate step, which should invalidate the cache
  run_buildah build --quiet --layers --pull=false --format docker $WITH_POLICY_JSON -f Dockerfile.different1 $BUDFILES/cache-scratch
  assert "$output" !~ "$iid"

  # now run it *again* again, except with more content added at an intermediate step, which should invalidate the cache
  run_buildah build --quiet --layers --pull=false --format docker $WITH_POLICY_JSON -f Dockerfile.different2 $BUDFILES/cache-scratch
  assert "$output" !~ "$iid"
}

@test "bud capabilities test" {
  _prefetch busybox
  # something not enabled by default in containers.conf
  run_buildah build --cap-add cap_sys_ptrace -t testcap $WITH_POLICY_JSON -f $BUDFILES/capabilities/Dockerfile
  expect_output --substring "uid=3267"
  expect_output --substring "CapBnd:	00000000a80c25fb"
  expect_output --substring "CapEff:	0000000000000000"

  # some things enabled by default in containers.conf
  run_buildah build --cap-drop cap_chown,cap_dac_override,cap_fowner -t testcapd $WITH_POLICY_JSON -f $BUDFILES/capabilities/Dockerfile
  expect_output --substring "uid=3267"
  expect_output --substring "CapBnd:	00000000a80425f0"
  expect_output --substring "CapEff:	0000000000000000"
}

@test "bud does not gobble stdin" {
  _prefetch alpine

  ctxdir=${TEST_SCRATCH_DIR}/bud
  mkdir -p $ctxdir
  cat >$ctxdir/Dockerfile <<EOF
FROM alpine
RUN true
EOF

  random_msg=$(head -10 /dev/urandom | tr -dc a-zA-Z0-9 | head -c12)

  # Prior to #2708, buildah bud would gobble up its stdin even if it
  # didn't actually use it. This prevented the use of 'cmdlist | bash';
  # if 'buildah bud' was in cmdlist, everything past it would be lost.
  #
  # This is ugly but effective: it checks that buildah passes stdin untouched.
  passthru=$(echo "$random_msg" | (run_buildah build --quiet $WITH_POLICY_JSON -t stdin-test ${ctxdir} >/dev/null; cat))

  expect_output --from="$passthru" "$random_msg" "stdin was passed through"
}

@test "bud cache by format" {
  # Build first in Docker format.  Whether we do OCI or Docker first shouldn't matter, so we picked one.
  run_buildah build --iidfile ${TEST_SCRATCH_DIR}/first-docker  --format docker --layers --quiet $WITH_POLICY_JSON $BUDFILES/cache-format

  # Build in OCI format.  Cache should not re-use the same images, so we should get a different image ID.
  run_buildah build --iidfile ${TEST_SCRATCH_DIR}/first-oci     --format oci    --layers --quiet $WITH_POLICY_JSON $BUDFILES/cache-format

  # Build in Docker format again.  Cache traversal should 100% hit the Docker image, so we should get its image ID.
  run_buildah build --iidfile ${TEST_SCRATCH_DIR}/second-docker --format docker --layers --quiet $WITH_POLICY_JSON $BUDFILES/cache-format

  # Build in OCI format again.  Cache traversal should 100% hit the OCI image, so we should get its image ID.
  run_buildah build --iidfile ${TEST_SCRATCH_DIR}/second-oci    --format oci    --layers --quiet $WITH_POLICY_JSON $BUDFILES/cache-format

  # Compare them.  The two images we built in Docker format should be the same, the two we built in OCI format
  # should be the same, but the OCI and Docker format images should be different.
  assert "$(< ${TEST_SCRATCH_DIR}/first-docker)" = "$(< ${TEST_SCRATCH_DIR}/second-docker)" \
         "iidfile(first docker) == iidfile(second docker)"
  assert "$(< ${TEST_SCRATCH_DIR}/first-oci)" = "$(< ${TEST_SCRATCH_DIR}/second-oci)" \
         "iidfile(first oci) == iidfile(second oci)"

  assert "$(< ${TEST_SCRATCH_DIR}/first-docker)" != "$(< ${TEST_SCRATCH_DIR}/first-oci)" \
         "iidfile(first docker) != iidfile(first oci)"
}

@test "bud cache add-copy-chown" {
  # Build each variation of COPY (from context, from previous stage) and ADD (from context, not overriding an archive, URL) twice.
  # Each second build should produce an image with the same ID as the first build, because the cache matches, but they should
  # otherwise all be different.
  local actions="copy prev add tar url";
  for i in 1 2 3; do
    for action in $actions; do
      # iidfiles are 1 2 3, but dockerfiles are only 1 2 then back to 1
      iidfile=${TEST_SCRATCH_DIR}/${action}${i}
      containerfile=Dockerfile.${action}$(((i-1) % 2 + 1))

      run_buildah build --iidfile $iidfile --layers --quiet $WITH_POLICY_JSON -f $containerfile $BUDFILES/cache-chown
    done
  done

  for action in $actions; do
    # The third round of builds should match all of the first rounds by way
    # of caching.
    assert "$(< ${TEST_SCRATCH_DIR}/${action}1)" = "$(< ${TEST_SCRATCH_DIR}/${action}3)" \
           "iidfile(${action}1) = iidfile(${action}3)"

    # The second round of builds should not match the first rounds, since
    # the different ownership makes the changes look different to the cache,
    # except for cases where we extract an archive, where --chown is ignored.
    local op="!="
    if [[ $action = "tar" ]]; then
      op="=";
    fi
    assert "$(< ${TEST_SCRATCH_DIR}/${action}1)" $op "$(< ${TEST_SCRATCH_DIR}/${action}2)" \
           "iidfile(${action}1) $op iidfile(${action}2)"

    # The first rounds of builds should all be different from each other,
    # as a sanity thing.
    for other in $actions; do
      if [[ $other != $action ]]; then
        assert "$(< ${TEST_SCRATCH_DIR}/${action}1)" != "$(< ${TEST_SCRATCH_DIR}/${other}1)" \
               "iidfile(${action}1) != iidfile(${other}1)"
      fi
    done
  done
}

@test "bud-terminal" {
  run_buildah build $BUDFILES/terminal
}

@test "bud --ignorefile containerignore" {
  _prefetch alpine busybox

  CONTEXTDIR=${TEST_SCRATCH_DIR}/dockerignore
  cp -r $BUDFILES/dockerignore ${CONTEXTDIR}
  mv ${CONTEXTDIR}/.dockerignore ${TEST_SCRATCH_DIR}/containerignore

  run_buildah build -t testbud $WITH_POLICY_JSON -f ${CONTEXTDIR}/Dockerfile.succeed --ignorefile  ${TEST_SCRATCH_DIR}/containerignore  ${CONTEXTDIR}

  run_buildah from --name myctr testbud

  run_buildah 1 run myctr ls -l test1.txt
  expect_output --substring "ls: test1.txt: No such file or directory"

  run_buildah run myctr ls -l test2.txt

  run_buildah 1 run myctr ls -l sub1.txt
  expect_output --substring "ls: sub1.txt: No such file or directory"

  run_buildah 1 run myctr ls -l sub2.txt
  expect_output --substring "ls: sub2.txt: No such file or directory"

  run_buildah 1 run myctr ls -l subdir/
  expect_output --substring "ls: subdir/: No such file or directory"
}

@test "bud with network options" {
  _prefetch alpine
  target=alpine-image

  run_buildah build --network=none $WITH_POLICY_JSON -t ${target} $BUDFILES/containerfile
  expect_output --substring "FROM alpine"

  run_buildah build --network=private $WITH_POLICY_JSON -t ${target} $BUDFILES/containerfile
  expect_output --substring "FROM alpine"

  run_buildah build --network=container $WITH_POLICY_JSON -t ${target} $BUDFILES/containerfile
  expect_output --substring "FROM alpine"
}

@test "bud-replace-from-in-containerfile" {
  _prefetch alpine busybox
  # override the first FROM (fedora) image in the Containerfile
  # with alpine, leave the second (busybox) alone.
  run_buildah build $WITH_POLICY_JSON --from=alpine $BUDFILES/build-with-from
  expect_output --substring "\[1/2] STEP 1/1: FROM alpine AS builder"
  expect_output --substring "\[2/2] STEP 1/2: FROM busybox"
}

@test "bud test no --stdin" {
  _prefetch alpine
  mytmpdir=${TEST_SCRATCH_DIR}/my-dir
  mkdir -p ${mytmpdir}
cat > $mytmpdir/Containerfile << _EOF
FROM alpine
RUN read -t 1 x && echo test got \<\$x\>
RUN touch /tmp/done
_EOF

  # fail without --stdin
  run_buildah 1 bud -t testbud $WITH_POLICY_JSON ${mytmpdir} <<< input
  expect_output --substring "building .*: exit status 1"

  run_buildah build --stdin -t testbud $WITH_POLICY_JSON ${mytmpdir} <<< input
  expect_output --substring "test got <input>"
}

@test "bud with --arch flag" {
  _prefetch alpine
  mytmpdir=${TEST_SCRATCH_DIR}/my-dir
  mkdir -p ${mytmpdir}
cat > $mytmpdir/Containerfile << _EOF
FROM alpine
#RUN arch
_EOF

  run_buildah build --arch=arm64 -t arch-test $WITH_POLICY_JSON ${mytmpdir} <<< input
# expect_output --substring "aarch64"

#  run_buildah from --quiet --pull=false $WITH_POLICY_JSON arch-test
#  cid=$output
#  run_buildah run $cid arch
#  expect_output --substring "aarch64"
}

@test "bud with --manifest flag new manifest" {
  _prefetch alpine
  mytmpdir=${TEST_SCRATCH_DIR}/my-dir
  mkdir -p ${mytmpdir}
cat > $mytmpdir/Containerfile << _EOF
from alpine
run echo hello
_EOF

  run_buildah build -q --manifest=testlist -t arch-test $WITH_POLICY_JSON ${mytmpdir} <<< input
  cid=$output
  run_buildah images
  expect_output --substring testlist

  run_buildah inspect --format '{{ .FromImageDigest }}' $cid
  digest=$output

  run_buildah manifest inspect testlist
  expect_output --substring $digest
}

@test "bud with --manifest flag existing manifest" {
  _prefetch alpine
  mytmpdir=${TEST_SCRATCH_DIR}/my-dir
  mkdir -p ${mytmpdir}
cat > $mytmpdir/Containerfile << _EOF
from alpine
run echo hello
_EOF

  run_buildah manifest create testlist

  run_buildah build -q --manifest=testlist -t arch-test $WITH_POLICY_JSON ${mytmpdir} <<< input
  cid=$output
  run_buildah images
  expect_output --substring testlist

  run_buildah inspect --format '{{ .FromImageDigest }}' $cid
  digest=$output

  run_buildah manifest inspect testlist
  expect_output --substring $digest
}

@test "bud test empty newdir" {
  _prefetch alpine
  mytmpdir=${TEST_SCRATCH_DIR}/my-dir
  mkdir -p ${mytmpdir}
cat > $mytmpdir/Containerfile << _EOF
FROM alpine as galaxy

RUN mkdir -p /usr/share/ansible/roles /usr/share/ansible/collections
RUN echo "bar"
RUN echo "foo" > /usr/share/ansible/collections/file.txt

FROM galaxy

RUN mkdir -p /usr/share/ansible/roles /usr/share/ansible/collections
COPY --from=galaxy /usr/share/ansible/roles /usr/share/ansible/roles
COPY --from=galaxy /usr/share/ansible/collections /usr/share/ansible/collections
_EOF

  run_buildah build --layers $WITH_POLICY_JSON -t testbud $mytmpdir
  expect_output --substring "COPY --from=galaxy /usr/share/ansible/collections /usr/share/ansible/collections"
}

@test "bud retain intermediary image" {
  _prefetch alpine
  mytmpdir=${TEST_SCRATCH_DIR}/my-dir
  mkdir -p ${mytmpdir}
cat > $mytmpdir/Containerfile.a << _EOF
FROM alpine
LABEL image=a
RUN echo foo
_EOF

cat > $mytmpdir/Containerfile.b << _EOF
FROM image-a
FROM scratch
_EOF

  run_buildah build -f Containerfile.a -q --manifest=testlist -t image-a $WITH_POLICY_JSON ${mytmpdir} <<< input
  cid=$output
  run_buildah images -f "label=image=a"
  expect_output --substring image-a

  run_buildah build -f Containerfile.b -q --manifest=testlist -t image-b $WITH_POLICY_JSON ${mytmpdir} <<< input
  cid=$output
  run_buildah images
  expect_output --substring image-a
}

@test "bud --pull=false --arch test" {
  mytmpdir=${TEST_SCRATCH_DIR}/my-dir
  mkdir -p ${mytmpdir}
cat > $mytmpdir/Containerfile << _EOF
FROM $SAFEIMAGE
_EOF
  run_buildah build --pull=false -q --arch=amd64 -t image-amd $WITH_POLICY_JSON ${mytmpdir}
  run_buildah inspect --format '{{ .OCIv1.Architecture }}' image-amd
  expect_output amd64

  # Tag the image to localhost/safeimage to make sure that the image gets
  # pulled since the local one does not match the requested architecture.
  run_buildah tag image-amd localhost/${SAFEIMAGE_NAME}:${SAFEIMAGE_TAG}
  run_buildah build --pull=false -q --arch=arm64 -t image-arm $WITH_POLICY_JSON ${mytmpdir}
  run_buildah inspect --format '{{ .OCIv1.Architecture }}' image-arm
  expect_output arm64

  run_buildah inspect --format '{{ .FromImageID }}' image-arm
  fromiid=$output

  run_buildah inspect --format '{{ .OCIv1.Architecture  }}'  $fromiid
  expect_output arm64
}

@test "bud --file with directory" {
  _prefetch alpine
  mytmpdir=${TEST_SCRATCH_DIR}/my-dir1
  mkdir -p ${mytmpdir}
  cat > $mytmpdir/Containerfile << _EOF
FROM alpine
_EOF

  run_buildah 125 build -t testbud $WITH_POLICY_JSON --file ${mytmpdir} .
}

@test "bud --authfile" {
  _prefetch alpine
  start_registry
  run_buildah login --tls-verify=false --authfile ${TEST_SCRATCH_DIR}/test.auth --username testuser --password testpassword localhost:${REGISTRY_PORT}
  run_buildah push $WITH_POLICY_JSON --tls-verify=false --authfile ${TEST_SCRATCH_DIR}/test.auth alpine docker://localhost:${REGISTRY_PORT}/buildah/alpine

  mytmpdir=${TEST_SCRATCH_DIR}/my-dir
  mkdir -p ${mytmpdir}
  cat > $mytmpdir/Containerfile << _EOF
FROM localhost:${REGISTRY_PORT}/buildah/alpine
RUN touch /test
_EOF
  run_buildah build -t myalpine --authfile ${TEST_SCRATCH_DIR}/test.auth --tls-verify=false $WITH_POLICY_JSON --file ${mytmpdir}/Containerfile .
  run_buildah rmi localhost:${REGISTRY_PORT}/buildah/alpine
  run_buildah rmi myalpine
}

@test "build verify cache behaviour with --cache-ttl" {
  _prefetch alpine
  local contextdir=${TEST_SCRATCH_DIR}/bud/platform
  mkdir -p $contextdir

  cat > $contextdir/Dockerfile1 << _EOF
FROM alpine
RUN touch hello
RUN echo world
_EOF

  # Build with --timestamp somewhere in the past
  run_buildah build $WITH_POLICY_JSON --timestamp 1628099045 --layers -t source -f $contextdir/Dockerfile1
  # Specify --cache-ttl 0.5s and cache should
  # not be used since cached image is created
  # with timestamp somewhere in past ( in ~2021 )
  run_buildah build $WITH_POLICY_JSON --cache-ttl=0.5s --layers -t source -f $contextdir/Dockerfile1
  # Should not contain `Using cache` since all
  # cached layers are 1s old.
  assert "$output" !~ "Using cache"
  # clean all images and cache
  run_buildah rmi --all -f
  _prefetch alpine
  run_buildah build $WITH_POLICY_JSON --layers -t source -f $contextdir/Dockerfile1
  # Cache should be used since our ttl is 1h but
  # cache layers are just built so they should be
  # few seconds old.
  run_buildah build $WITH_POLICY_JSON --cache-ttl=1h --layers -t source -f $contextdir/Dockerfile1
  # must use already cached images.
  expect_output --substring "Using cache"
}

@test "build verify cache behaviour with --cache-ttl=0s" {
  _prefetch alpine
  local contextdir=${TEST_SCRATCH_DIR}/bud/platform
  mkdir -p $contextdir

  cat > $contextdir/Dockerfile1 << _EOF
FROM alpine
RUN touch hello
RUN echo world
_EOF

  # Build with --timestamp somewhere in the past
  run_buildah build $WITH_POLICY_JSON --timestamp 1628099045 --layers -t source -f $contextdir/Dockerfile1
  # Specify --cache-ttl 0.5s and cache should
  # not be used since cached image is created
  # with timestamp somewhere in past ( in ~2021 )
  run_buildah --log-level debug build $WITH_POLICY_JSON --cache-ttl=0 --layers -t source -f $contextdir/Dockerfile1
  # Should not contain `Using cache` since all
  # cached layers are 1s old.
  assert "$output" !~ "Using cache"
  expect_output --substring "Setting --no-cache=true"
}

@test "build test pushing and pulling from multiple remote cache sources" {
  _prefetch alpine
  mytmpdir=${TEST_SCRATCH_DIR}/my-dir
  mkdir -p ${mytmpdir}
  echo something > ${mytmpdir}/somefile
  cat > $mytmpdir/Containerfile << _EOF
FROM alpine
RUN echo hello
RUN echo world
RUN touch hello
ADD somefile somefile

FROM alpine
RUN echo hello
COPY --from=0 hello hello
_EOF

  start_registry
  run_buildah login --tls-verify=false --authfile ${TEST_SCRATCH_DIR}/test.auth --username testuser --password testpassword localhost:${REGISTRY_PORT}

  # ------ Test case ------ #
  # prepare expected output beforehand
  # must push cache twice i.e for first step and second step
  run printf "STEP 2/5: RUN echo hello\nhello\n--> Pushing cache"
  step1=$output
  run printf "STEP 3/5: RUN echo world\nworld\n--> Pushing cache"
  step2=$output
  run printf "STEP 4/5: RUN touch hello\n--> Pushing cache"
  step3=$output
  run printf "STEP 5/5: ADD somefile somefile\n--> Pushing cache"
  step4=$output
  # First run step in second stage should not be pushed since its already pushed
  run printf "STEP 2/3: RUN echo hello\n--> Using cache"
  step5=$output
  # Last step is `COPY --from=0 hello hello' so it must be committed and pushed
  # actual output is `[2/2] STEP 3/3: COPY --from=0 hello hello\n[2/2] COMMIT test\n-->Pushing cache`
  # but lets match smaller suffix
  run printf "COMMIT test\n--> Pushing cache"
  step6=$output

  # actually run build
  run_buildah build $WITH_POLICY_JSON --tls-verify=false --authfile ${TEST_SCRATCH_DIR}/test.auth --layers --cache-to localhost:${REGISTRY_PORT}/temp2 --cache-to localhost:${REGISTRY_PORT}/temp -t test -f ${mytmpdir}/Containerfile ${mytmpdir}
  expect_output --substring "$step1"
  expect_output --substring "$step2"
  expect_output --substring "$step3"
  expect_output --substring "$step4"
  expect_output --substring "$step5"
  expect_output --substring "$step6"

  # clean all cache and intermediate images
  # to make sure that we are only using cache
  # from remote repo and not the local storage.
  run_buildah rmi --all -f

  # ------ Test case ------ #
  # expect cache to be pushed on remote stream
  # now a build on clean slate must pull cache
  # from remote instead of actually computing the
  # run steps
  run printf "STEP 2/5: RUN echo hello\n--> Cache pulled from remote"
  step1=$output
  run printf "STEP 3/5: RUN echo world\n--> Cache pulled from remote"
  step2=$output
  run printf "STEP 4/5: RUN touch hello\n--> Cache pulled from remote"
  step3=$output
  run printf "STEP 5/5: ADD somefile somefile\n--> Cache pulled from remote"
  step4=$output
  # First run step in second stage should not be pulled since its already pulled
  run printf "STEP 2/3: RUN echo hello\n--> Using cache"
  step5=$output
  run printf "COPY --from=0 hello hello\n--> Cache pulled from remote"
  step6=$output
  run_buildah build $WITH_POLICY_JSON --tls-verify=false --authfile ${TEST_SCRATCH_DIR}/test.auth --layers --cache-from localhost:${REGISTRY_PORT}/temp -t test -f ${mytmpdir}/Containerfile ${mytmpdir}
  expect_output --substring "$step1"
  expect_output --substring "$step2"
  expect_output --substring "$step3"
  expect_output --substring "$step4"
  expect_output --substring "$step5"
  expect_output --substring "$step6"

  ##### Test when cache source is: localhost:${REGISTRY_PORT}/temp2

  # clean all cache and intermediate images
  # to make sure that we are only using cache
  # from remote repo and not the local storage.
  run_buildah rmi --all -f

  # ------ Test case ------ #
  # expect cache to be pushed on remote stream
  # now a build on clean slate must pull cache
  # from remote instead of actually computing the
  # run steps
  run printf "STEP 2/5: RUN echo hello\n--> Cache pulled from remote"
  step1=$output
  run printf "STEP 3/5: RUN echo world\n--> Cache pulled from remote"
  step2=$output
  run printf "STEP 4/5: RUN touch hello\n--> Cache pulled from remote"
  step3=$output
  run printf "STEP 5/5: ADD somefile somefile\n--> Cache pulled from remote"
  step4=$output
  # First run step in second stage should not be pulled since its already pulled
  run printf "STEP 2/3: RUN echo hello\n--> Using cache"
  step5=$output
  run printf "COPY --from=0 hello hello\n--> Cache pulled from remote"
  step6=$output
  run_buildah build $WITH_POLICY_JSON --tls-verify=false --authfile ${TEST_SCRATCH_DIR}/test.auth --layers --cache-from localhost:${REGISTRY_PORT}/temp2 -t test -f ${mytmpdir}/Containerfile ${mytmpdir}
  expect_output --substring "$step1"
  expect_output --substring "$step2"
  expect_output --substring "$step3"
  expect_output --substring "$step4"
  expect_output --substring "$step5"
  expect_output --substring "$step6"
}

@test "build test pushing and pulling from remote cache sources" {
  _prefetch alpine
  mytmpdir=${TEST_SCRATCH_DIR}/my-dir
  mkdir -p ${mytmpdir}
  echo something > ${mytmpdir}/somefile
  cat > $mytmpdir/Containerfile << _EOF
FROM alpine
RUN echo hello
RUN echo world
RUN touch hello
ADD somefile somefile

FROM alpine
RUN echo hello
COPY --from=0 hello hello
RUN --mount=type=cache,id=YfHI60aApFM-target,target=/target echo world > /target/hello
_EOF

  start_registry
  run_buildah login --tls-verify=false --authfile ${TEST_SCRATCH_DIR}/test.auth --username testuser --password testpassword localhost:${REGISTRY_PORT}

  # ------ Test case ------ #
  # prepare expected output beforehand
  # must push cache twice i.e for first step and second step
  run printf "STEP 2/5: RUN echo hello\nhello\n--> Pushing cache"
  step1=$output
  run printf "STEP 3/5: RUN echo world\nworld\n--> Pushing cache"
  step2=$output
  run printf "STEP 4/5: RUN touch hello\n--> Pushing cache"
  step3=$output
  run printf "STEP 5/5: ADD somefile somefile\n--> Pushing cache"
  step4=$output
  # First run step in second stage should not be pushed since its already pushed
  run printf "STEP 2/4: RUN echo hello\n--> Using cache"
  step5=$output
  # Last step is `COPY --from=0 hello hello' so it must be committed and pushed
  # actual output is `[2/2] STEP 3/3: COPY --from=0 hello hello\n[2/2] COMMIT test\n-->Pushing cache`
  # but lets match smaller suffix
  run printf "COMMIT test\n--> Pushing cache"
  step6=$output

  # actually run build
  run_buildah build $WITH_POLICY_JSON --tls-verify=false --authfile ${TEST_SCRATCH_DIR}/test.auth --layers --cache-to localhost:${REGISTRY_PORT}/temp -t test -f ${mytmpdir}/Containerfile ${mytmpdir}
  expect_output --substring "$step1"
  expect_output --substring "$step2"
  expect_output --substring "$step3"
  expect_output --substring "$step4"
  expect_output --substring "$step5"
  expect_output --substring "$step6"

  # clean all cache and intermediate images
  # to make sure that we are only using cache
  # from remote repo and not the local storage.
  run_buildah rmi --all -f

  # ------ Test case ------ #
  # expect cache to be pushed on remote stream
  # now a build on clean slate must pull cache
  # from remote instead of actually computing the
  # run steps
  run printf "STEP 2/5: RUN echo hello\n--> Cache pulled from remote"
  step1=$output
  run printf "STEP 3/5: RUN echo world\n--> Cache pulled from remote"
  step2=$output
  run printf "STEP 4/5: RUN touch hello\n--> Cache pulled from remote"
  step3=$output
  run printf "STEP 5/5: ADD somefile somefile\n--> Cache pulled from remote"
  step4=$output
  # First run step in second stage should not be pulled since its already pulled
  run printf "STEP 2/4: RUN echo hello\n--> Using cache"
  step5=$output
  run printf "COPY --from=0 hello hello\n--> Cache pulled from remote"
  step6=$output
  run_buildah build $WITH_POLICY_JSON --tls-verify=false --authfile ${TEST_SCRATCH_DIR}/test.auth --layers --cache-from localhost:${REGISTRY_PORT}/temp --cache-to localhost:${REGISTRY_PORT}/temp -t test -f ${mytmpdir}/Containerfile ${mytmpdir}
  expect_output --substring "$step1"
  expect_output --substring "$step2"
  expect_output --substring "$step3"
  expect_output --substring "$step4"
  expect_output --substring "$step5"
  expect_output --substring "$step6"

  # ------ Test case ------ #
  # Try building again with --cache-from to make sure
  # we don't pull image if we already have it in our
  # local storage
  run_buildah build $WITH_POLICY_JSON --tls-verify=false --authfile ${TEST_SCRATCH_DIR}/test.auth --layers --cache-from localhost:${REGISTRY_PORT}/temp -t test -f ${mytmpdir}/Containerfile ${mytmpdir}
  # must use cache since we have cache in local storage
  expect_output --substring "Using cache"
  # should not pull cache if its already in local storage
  assert "$output" !~ "Cache pulled"

  # ------ Test case ------ #
  # Build again with --cache-to and --cache-from
  # Since intermediate images are already present
  # on local storage so nothing must be pulled but
  # intermediate must be pushed since buildah is not
  # aware if they on remote repo or not.
  run_buildah build $WITH_POLICY_JSON --tls-verify=false --authfile ${TEST_SCRATCH_DIR}/test.auth --layers --cache-from localhost:${REGISTRY_PORT}/temp --cache-to localhost:${REGISTRY_PORT}/temp -t test -f ${mytmpdir}/Containerfile ${mytmpdir}
  # must use cache since we have cache in local storage
  expect_output --substring "Using cache"
  # must also push cache since nothing was pulled from remote repo
  expect_output --substring "Pushing cache"
  # should not pull cache if its already in local storage
  assert "$output" !~ "Cache pulled"

}

@test "build test pushing and pulling from remote cache sources - after adding content summary" {
  _prefetch alpine

  start_registry
  run_buildah login --tls-verify=false --authfile ${TEST_SCRATCH_DIR}/test.auth --username testuser --password testpassword localhost:${REGISTRY_PORT}

  # ------ Test case ------ #
  # prepare expected output beforehand
  # must push cache twice i.e for first step and second step
  run printf "STEP 2/3: ARG VAR=hello\n--> Pushing cache"
  step1=$output
  run printf "STEP 3/3: RUN echo \"Hello \$VAR\""
  step2=$output
  run printf "Hello hello"
  step3=$output
  run printf "COMMIT test\n--> Pushing cache"
  step6=$output

  # actually run build
  run_buildah build $WITH_POLICY_JSON --tls-verify=false --authfile ${TEST_SCRATCH_DIR}/test.auth --layers --cache-to localhost:${REGISTRY_PORT}/temp -t test -f $BUDFILES/cache-from/Containerfile
  expect_output --substring "$step1"
  #expect_output --substring "$step2"
  expect_output --substring "$step3"
  expect_output --substring "$step6"

  # clean all cache and intermediate images
  # to make sure that we are only using cache
  # from remote repo and not the local storage.

  # Important side-note: don't use `run_buildah rmi --all -f`
  # since on podman-remote test this will remove prefetched alpine
  # and it will try to pull alpine from docker.io with
  # completely different digest (ruining our cache logic).
  run_buildah rmi test

  # ------ Test case ------ #
  # expect cache to be pushed on remote stream
  # now a build on clean slate must pull cache
  # from remote instead of actually computing the
  # run steps
  run printf "STEP 2/3: ARG VAR=hello\n--> Cache pulled from remote"
  step1=$output
  run printf "VAR\"\n--> Cache pulled from remote"
  step2=$output
  run_buildah build $WITH_POLICY_JSON --tls-verify=false --authfile ${TEST_SCRATCH_DIR}/test.auth --layers --cache-from localhost:${REGISTRY_PORT}/temp --cache-to localhost:${REGISTRY_PORT}/temp -t test -f $BUDFILES/cache-from/Containerfile
  expect_output --substring "$step1"
  expect_output --substring "$step2"

  # ------ Test case ------ #
  # Try building again with --cache-from to make sure
  # we don't pull image if we already have it in our
  # local storage
  run_buildah build $WITH_POLICY_JSON --tls-verify=false --authfile ${TEST_SCRATCH_DIR}/test.auth --layers --cache-from localhost:${REGISTRY_PORT}/temp -t test -f $BUDFILES/cache-from/Containerfile
  # must use cache since we have cache in local storage
  expect_output --substring "Using cache"
  # should not pull cache if its already in local storage
  assert "$output" !~ "Cache pulled"

  # ------ Test case ------ #
  # Build again with --cache-to and --cache-from
  # Since intermediate images are already present
  # on local storage so nothing must be pulled but
  # intermediate must be pushed since buildah is not
  # aware if they on remote repo or not.
  run_buildah build $WITH_POLICY_JSON --tls-verify=false --authfile ${TEST_SCRATCH_DIR}/test.auth --layers --cache-from localhost:${REGISTRY_PORT}/temp --cache-to localhost:${REGISTRY_PORT}/temp -t test -f $BUDFILES/cache-from/Containerfile
  # must use cache since we have cache in local storage
  expect_output --substring "Using cache"
  # must also push cache since nothing was pulled from remote repo
  expect_output --substring "Pushing cache"
  # should not pull cache if its already in local storage
  assert "$output" !~ "Cache pulled"

}

@test "bud with undefined build arg directory" {
  _prefetch alpine
  mytmpdir=${TEST_SCRATCH_DIR}/my-dir1
  mkdir -p ${mytmpdir}
  cat > $mytmpdir/Containerfile << _EOF
FROM alpine
ARG SECRET="Itismysecret"
ARG NEWSECRET
RUN echo $SECRET
RUN touch hello
FROM alpine
COPY --from=0 hello .
RUN echo "$SECRET"
_EOF

  run_buildah build -t testbud $WITH_POLICY_JSON --file ${mytmpdir}/Containerfile .
  assert "$output" !~ '--build-arg SECRET=<VALUE>'
  expect_output --substring '\-\-build-arg NEWSECRET=<VALUE>'

  run_buildah build -t testbud $WITH_POLICY_JSON --build-arg NEWSECRET="VerySecret" --file ${mytmpdir}/Containerfile .
  assert "$output" !~ '--build-arg SECRET=<VALUE>'
  assert "$output" !~ '--build-arg NEWSECRET=<VALUE>'

# case should similarly honor globally declared args
  cat > $mytmpdir/Containerfile << _EOF
ARG SECRET="Itismysecret"
FROM alpine
ARG SECRET
ARG NEWSECRET
RUN echo $SECRET
RUN touch hello
FROM alpine
COPY --from=0 hello .
RUN echo "$SECRET"
_EOF

  run_buildah build -t testbud $WITH_POLICY_JSON --file ${mytmpdir}/Containerfile .
  assert "$output" !~ '--build-arg SECRET=<VALUE>'
  expect_output --substring '\-\-build-arg NEWSECRET=<VALUE>'

  run_buildah build -t testbud $WITH_POLICY_JSON --build-arg NEWSECRET="VerySecret" --file ${mytmpdir}/Containerfile .
  assert "$output" !~ '--build-arg SECRET=<VALUE>'
  assert "$output" !~ '--build-arg NEWSECRET=<VALUE>'
}

@test "bud with arg in from statement" {
  _prefetch alpine
  run_buildah build -t testbud $WITH_POLICY_JSON --build-arg app_type=x --build-arg another_app_type=m --file $BUDFILES/with-arg/Dockerfilefromarg .
  expect_output --substring 'world'
}

@test "bud with --runtime and --runtime-flag" {
  # This Containerfile needs us to be able to handle a working RUN instruction.
  skip_if_no_runtime
  skip_if_chroot

  _prefetch alpine

  mytmpdir=${TEST_SCRATCH_DIR}/my-dir
  mkdir -p ${mytmpdir}
  cat > $mytmpdir/Containerfile << _EOF
from alpine
run echo hello
_EOF

  local found_runtime=

  # runc-1.0.0-70.rc92 and 1.0.1-3 have completely different
  # debug messages. This is the only string common to both.
  local flag_accepted_rx="level=debug.*msg=.child process in init"
  if [ -n "$(command -v runc)" ]; then
    found_runtime=y
    if is_cgroupsv2; then
      # The result with cgroup v2 depends on the version of runc.
      run_buildah '?' bud --runtime=runc --runtime-flag=debug \
                        -q -t alpine-bud-runc $WITH_POLICY_JSON --file ${mytmpdir}/Containerfile .
      if [ "$status" -eq 0 ]; then
        expect_output --substring "$flag_accepted_rx"
      else
        # If it fails, this is because this version of runc doesn't support cgroup v2.
        expect_output --substring "this version of runc doesn't work on cgroups v2" "should fail by unsupportability for cgroupv2"
      fi
    else
      run_buildah build --runtime=runc --runtime-flag=debug \
                      -q -t alpine-bud-runc $WITH_POLICY_JSON --file ${mytmpdir}/Containerfile .
      expect_output --substring "$flag_accepted_rx"
    fi

  fi

  if [ -n "$(command -v crun)" ]; then
    found_runtime=y

    # Use seccomp to make crun output a warning message because crun writes few logs.
    cat > ${TEST_SCRATCH_DIR}/seccomp.json << _EOF
{
    "defaultAction": "SCMP_ACT_ALLOW",
    "syscalls": [
      {
        "name": "unknown",
        "action": "SCMP_ACT_KILL"
	    }
    ]
}
_EOF

    run_buildah build --runtime=crun --runtime-flag=debug --security-opt seccomp=${TEST_SCRATCH_DIR}/seccomp.json \
                    -q -t alpine-bud-crun $WITH_POLICY_JSON --file ${mytmpdir}/Containerfile .
    expect_output --substring "unknown seccomp syscall"
  fi

  if [ -z "${found_runtime}" ]; then
    die "Did not find 'runc' nor 'crun' in \$PATH - could not run this test!"
  fi

}

@test "bud - invalid runtime flags test" {
  skip_if_no_runtime
  skip_if_chroot

  _prefetch alpine

  mytmpdir=${TEST_SCRATCH_DIR}/my-dir
  mkdir -p ${mytmpdir}
  cat > $mytmpdir/Containerfile << _EOF
from alpine
run echo hello
_EOF

    run_buildah 1 build $WITH_POLICY_JSON --runtime-flag invalidflag -t build_test $mytmpdir
    assert "$output" =~ ".*invalidflag" "failed when passing undefined flags to the runtime"
}

@test "bud - accept at most one arg" {
    run_buildah 125 build $WITH_POLICY_JSON $BUDFILES/dns extraarg
    assert "$output" =~ ".*accepts at most 1 arg\(s\), received 2" "Should fail when passed extra arg after context directory"
}

@test "bud with --no-hostname" {
  skip_if_no_runtime

  _prefetch alpine

  run_buildah build --no-cache -t testbud \
                  $WITH_POLICY_JSON $BUDFILES/no-hostname
  assert "${lines[2]}" != "localhost" "Should be set to something other then localhost"

  run_buildah build --no-hostname --no-cache -t testbud \
                  $WITH_POLICY_JSON \
		  $BUDFILES/no-hostname
  assert "${lines[2]}" == "localhost" "Should be set to localhost"

  run_buildah 1 build --network=none --no-hostname --no-cache -t testbud \
                  $WITH_POLICY_JSON \
		  -f $BUDFILES/no-hostname/Containerfile.noetc \
		  $BUDFILES/no-hostname
  assert "$output" =~ ".*ls: /etc: No such file or directory" "/etc/ directory should be gone"
}

@test "bud with --add-host" {
  skip_if_no_runtime

  _prefetch alpine

  mytmpdir=${TEST_SCRATCH_DIR}/my-dir
  mkdir -p ${mytmpdir}
  cat > $mytmpdir/Containerfile << _EOF
from alpine
run grep "myhostname" /etc/hosts
_EOF

  ip=123.45.67.$(( $RANDOM % 256 ))
  run_buildah build --add-host=myhostname:$ip -t testbud \
                  $WITH_POLICY_JSON --file ${mytmpdir}/Containerfile .
  expect_output --from="${lines[2]}" --substring "^$ip\s+myhostname"

  run_buildah 125 build --no-cache --add-host=myhostname:$ip \
                  --no-hosts \
                  $WITH_POLICY_JSON --file ${mytmpdir}/Containerfile .
  expect_output --substring "\-\-no-hosts and \-\-add-host conflict, can not be used together"

  run_buildah 1 build --no-cache --no-hosts \
                  $WITH_POLICY_JSON --file ${mytmpdir}/Containerfile .
  expect_output --substring 'building at STEP "RUN grep "myhostname" /etc/hosts'
}

@test "bud with --cgroup-parent" {
  skip_if_rootless_environment
  skip_if_no_runtime
  skip_if_chroot

  _prefetch alpine

  mytmpdir=${TEST_SCRATCH_DIR}/my-dir
  mkdir -p ${mytmpdir}
  cat > $mytmpdir/Containerfile << _EOF
from alpine
run cat /proc/self/cgroup
_EOF

  # with cgroup-parent
  run_buildah --cgroup-manager cgroupfs build --cgroupns=host --cgroup-parent test-cgroup -t with-flag \
                  $WITH_POLICY_JSON --file ${mytmpdir}/Containerfile .
  if is_cgroupsv2; then
    expect_output --from="${lines[2]}" "0::/test-cgroup"
  else
    expect_output --substring "/test-cgroup"
  fi
  # without cgroup-parent
  run_buildah --cgroup-manager cgroupfs build -t without-flag \
                  $WITH_POLICY_JSON --file ${mytmpdir}/Containerfile .
  if [ -n "$(grep "test-cgroup" <<< "$output")" ]; then
    die "Unexpected cgroup."
  fi
}

@test "bud with --cpu-period and --cpu-quota" {
  skip_if_chroot
  skip_if_rootless_and_cgroupv1
  skip_if_rootless_environment
  skip_if_no_runtime

  _prefetch alpine

  mytmpdir=${TEST_SCRATCH_DIR}/my-dir
  mkdir -p ${mytmpdir}

  if is_cgroupsv2; then
    cat > $mytmpdir/Containerfile << _EOF
from alpine
run cat /sys/fs/cgroup/\$(awk -F: '{print \$NF}' /proc/self/cgroup)/cpu.max
_EOF
  else
    cat > $mytmpdir/Containerfile << _EOF
from alpine
run echo "\$(cat /sys/fs/cgroup/cpu/cpu.cfs_quota_us) \$(cat /sys/fs/cgroup/cpu/cpu.cfs_period_us)"
_EOF
  fi

  run_buildah build --cpu-period=1234 --cpu-quota=5678 -t testcpu \
                  $WITH_POLICY_JSON --file ${mytmpdir}/Containerfile .
  expect_output --from="${lines[2]}" "5678 1234"
}

@test "bud check mount /sys/fs/cgroup" {
  skip_if_rootless_and_cgroupv1
  mytmpdir=${TEST_SCRATCH_DIR}/my-dir
  mkdir -p ${mytmpdir}

  cat > $mytmpdir/Containerfile << _EOF
from alpine
run ls /sys/fs/cgroup
_EOF
  run_buildah build $WITH_POLICY_JSON --file ${mytmpdir}/Containerfile .
  expect_output --substring "cpu"
  expect_output --substring "memory"
}

@test "bud with --cpu-shares" {
  skip_if_chroot
  skip_if_rootless_environment
  skip_if_rootless_and_cgroupv1
  skip_if_no_runtime

  _prefetch alpine

  local shares=12345
  local expect=

  mytmpdir=${TEST_SCRATCH_DIR}/my-dir
  mkdir -p ${mytmpdir}

  if is_cgroupsv2; then
    cat > $mytmpdir/Containerfile << _EOF
from alpine
run printf "weight " && cat /sys/fs/cgroup/\$(awk -F : '{print \$NF}' /proc/self/cgroup)/cpu.weight
_EOF
    expect="weight $((1 + ((${shares} - 2) * 9999) / 262142))"
  else
    cat > $mytmpdir/Containerfile << _EOF
from alpine
run printf "weight " && cat /sys/fs/cgroup/cpu/cpu.shares
_EOF
    expect="weight ${shares}"
  fi

  run_buildah build --cpu-shares=${shares} -t testcpu \
                  $WITH_POLICY_JSON --file ${mytmpdir}/Containerfile .
  expect_output --from="${lines[2]}" "${expect}"
}

@test "bud with --cpuset-cpus" {
  skip_if_chroot
  skip_if_rootless_and_cgroupv1
  skip_if_rootless_environment
  skip_if_no_runtime

  _prefetch alpine

  mytmpdir=${TEST_SCRATCH_DIR}/my-dir
  mkdir -p ${mytmpdir}

  if is_cgroupsv2; then
    cat > $mytmpdir/Containerfile << _EOF
from alpine
run printf "cpuset-cpus " && cat /sys/fs/cgroup/\$(awk -F : '{print \$NF}' /proc/self/cgroup)/cpuset.cpus
_EOF
  else
    cat > $mytmpdir/Containerfile << _EOF
from alpine
run printf "cpuset-cpus " && cat /sys/fs/cgroup/cpuset/cpuset.cpus
_EOF
  fi

  run_buildah build --cpuset-cpus=0 -t testcpuset \
                  $WITH_POLICY_JSON --file ${mytmpdir}/Containerfile .
  expect_output --from="${lines[2]}" "cpuset-cpus 0"
}

@test "bud with --cpuset-mems" {
  skip_if_chroot
  skip_if_rootless_and_cgroupv1
  skip_if_rootless_environment
  skip_if_no_runtime

  _prefetch alpine

  mytmpdir=${TEST_SCRATCH_DIR}/my-dir
  mkdir -p ${mytmpdir}

  if is_cgroupsv2; then
    cat > $mytmpdir/Containerfile << _EOF
from alpine
run printf "cpuset-mems " && cat /sys/fs/cgroup/\$(awk -F : '{print \$NF}' /proc/self/cgroup)/cpuset.mems
_EOF
  else
    cat > $mytmpdir/Containerfile << _EOF
from alpine
run printf "cpuset-mems " && cat /sys/fs/cgroup/cpuset/cpuset.mems
_EOF
  fi

  run_buildah build --cpuset-mems=0 -t testcpuset \
                  $WITH_POLICY_JSON --file ${mytmpdir}/Containerfile .
  expect_output --from="${lines[2]}" "cpuset-mems 0"
}

@test "bud with --isolation" {
  skip_if_rootless_environment
  skip_if_no_runtime
  test -z "${BUILDAH_ISOLATION}" || skip "BUILDAH_ISOLATION=${BUILDAH_ISOLATION} overrides --isolation"

  _prefetch alpine

  mytmpdir=${TEST_SCRATCH_DIR}/my-dir
  mkdir -p ${mytmpdir}
  cat > $mytmpdir/Containerfile << _EOF
from alpine
run readlink /proc/self/ns/pid
_EOF

  run readlink /proc/self/ns/pid
  host_pidns=$output
  run_buildah build --isolation chroot -t testisolation --pid private \
                  $WITH_POLICY_JSON --file ${mytmpdir}/Containerfile .
  # chroot isolation doesn't make a new PID namespace.
  expect_output --from="${lines[2]}" "${host_pidns}"
}

@test "bud with --pull-always" {
  _prefetch docker.io/library/alpine
  run_buildah build --pull-always $WITH_POLICY_JSON -t testpull $BUDFILES/containerfile
  expect_output --from="${lines[1]}" "Trying to pull docker.io/library/alpine:latest..."
  run_buildah build --pull=always $WITH_POLICY_JSON -t testpull $BUDFILES/containerfile
  expect_output --from="${lines[1]}" "Trying to pull docker.io/library/alpine:latest..."
}

@test "bud with --memory and --memory-swap" {
  skip_if_chroot
  skip_if_no_runtime
  skip_if_rootless_and_cgroupv1
  skip_if_rootless_environment

  _prefetch alpine

  mytmpdir=${TEST_SCRATCH_DIR}/my-dir
  mkdir -p ${mytmpdir}

  local expect_swap=
  if is_cgroupsv2; then
    cat > $mytmpdir/Containerfile << _EOF
from alpine
run printf "memory-max=" && cat /sys/fs/cgroup/\$(awk -F : '{print \$NF}' /proc/self/cgroup)/memory.max
run printf "memory-swap-result=" && cat /sys/fs/cgroup/\$(awk -F : '{print \$NF}' /proc/self/cgroup)/memory.swap.max
_EOF
    expect_swap=31457280
  else
    cat > $mytmpdir/Containerfile << _EOF
from alpine
run printf "memory-max=" && cat /sys/fs/cgroup/memory/memory.limit_in_bytes
run printf "memory-swap-result=" && cat /sys/fs/cgroup/memory/memory.memsw.limit_in_bytes
_EOF
    expect_swap=73400320
  fi

  run_buildah build --memory=40m --memory-swap=70m -t testmemory \
                  $WITH_POLICY_JSON --file ${mytmpdir}/Containerfile .
  expect_output --from="${lines[2]}" "memory-max=41943040"
  expect_output --from="${lines[4]}" "memory-swap-result=${expect_swap}"
}

@test "bud with --shm-size" {
  skip_if_chroot
  skip_if_no_runtime

  _prefetch alpine

  mytmpdir=${TEST_SCRATCH_DIR}/my-dir
  mkdir -p ${mytmpdir}
  cat > $mytmpdir/Containerfile << _EOF
from alpine
run df -h /dev/shm
_EOF

  run_buildah build --shm-size=80m -t testshm \
                  $WITH_POLICY_JSON --file ${mytmpdir}/Containerfile .
  expect_output --from="${lines[3]}" --substring "shm\s+80.0M"
}

@test "bud with --ulimit" {
  _prefetch alpine

  mytmpdir=${TEST_SCRATCH_DIR}/my-dir
  mkdir -p ${mytmpdir}
  cat > $mytmpdir/Containerfile << _EOF
from alpine
run printf "ulimit=" && ulimit -t
_EOF

  run_buildah build --ulimit cpu=300 -t testulimit \
                  $WITH_POLICY_JSON --file ${mytmpdir}/Containerfile .
  expect_output --from="${lines[2]}" "ulimit=300"
}

@test "bud with .dockerignore #3" {
  run_buildah build -t test $WITH_POLICY_JSON $BUDFILES/copy-globs
  run_buildah build -t test2 -f Containerfile.missing $WITH_POLICY_JSON $BUDFILES/copy-globs

  run_buildah 125 build -t test3 -f Containerfile.bad $WITH_POLICY_JSON $BUDFILES/copy-globs
  expect_output --substring 'building.*"COPY \*foo /testdir".*no such file or directory'
}

@test "bud with containerfile secret" {
  _prefetch alpine
  mytmpdir=${TEST_SCRATCH_DIR}/my-dir1
  mkdir -p ${mytmpdir}
  cat > $mytmpdir/mysecret << _EOF
SOMESECRETDATA
_EOF

  run_buildah build --secret=id=mysecret,src=${mytmpdir}/mysecret $WITH_POLICY_JSON  -t secretimg -f $BUDFILES/run-mounts/Dockerfile.secret $BUDFILES/run-mounts
  expect_output --substring "SOMESECRETDATA"

  run_buildah from secretimg
  run_buildah 1 run secretimg-working-container cat /run/secrets/mysecret
  expect_output --substring "cat: can't open '/run/secrets/mysecret': No such file or directory"
  run_buildah rm -a
}

@test "bud with containerfile secret and secret is accessed twice and build should be successful" {
  _prefetch alpine
  mytmpdir=${TEST_SCRATCH_DIR}/my-dir1
  mkdir -p ${mytmpdir}
  cat > $mytmpdir/mysecret << _EOF
SOMESECRETDATA
_EOF

  cat > $mytmpdir/Dockerfile << _EOF
FROM alpine

RUN --mount=type=secret,id=mysecret,dst=/home/root/mysecret cat /home/root/mysecret

RUN --mount=type=secret,id=mysecret,dst=/home/root/mysecret2 echo hello && cat /home/root/mysecret2
_EOF

  run_buildah build --secret=id=mysecret,src=${mytmpdir}/mysecret $WITH_POLICY_JSON  -t secretimg -f ${mytmpdir}/Dockerfile
  expect_output --substring "hello"
  expect_output --substring "SOMESECRETDATA"
}

@test "bud with containerfile secret accessed on second RUN" {
  _prefetch alpine
  mytmpdir=${TEST_SCRATCH_DIR}/my-dir1
  mkdir -p ${mytmpdir}
  cat > $mytmpdir/mysecret << _EOF
SOMESECRETDATA
_EOF

  run_buildah 1 bud --secret=id=mysecret,src=${mytmpdir}/mysecret $WITH_POLICY_JSON  -t secretimg -f $BUDFILES/run-mounts/Dockerfile.secret-access $BUDFILES/run-mounts
  expect_output --substring "SOMESECRETDATA"
  expect_output --substring "cat: can't open '/mysecret': No such file or directory"
}

@test "bud with default mode perms" {
  _prefetch alpine
  mytmpdir=${TEST_SCRATCH_DIR}/my-dir1
  mkdir -p ${mytmpdir}
  cat > $mytmpdir/mysecret << _EOF
SOMESECRETDATA
_EOF

  run_buildah bud --secret=id=mysecret,src=${mytmpdir}/mysecret $WITH_POLICY_JSON  -t secretmode -f $BUDFILES/run-mounts/Dockerfile.secret-mode $BUDFILES/run-mounts
  expect_output --substring "400"
}

@test "bud with containerfile secret options" {
  _prefetch alpine
  mytmpdir=${TEST_SCRATCH_DIR}/my-dir1
  mkdir -p ${mytmpdir}
  cat > $mytmpdir/mysecret << _EOF
SOMESECRETDATA
_EOF

  run_buildah build --secret=id=mysecret,src=${mytmpdir}/mysecret $WITH_POLICY_JSON  -t secretopts -f $BUDFILES/run-mounts/Dockerfile.secret-options $BUDFILES/run-mounts
  expect_output --substring "444"
  expect_output --substring "1000"
  expect_output --substring "1001"
}

@test "bud with containerfile secret not required" {
  _prefetch alpine

  run_buildah build $WITH_POLICY_JSON  -t secretnotreq -f $BUDFILES/run-mounts/Dockerfile.secret-not-required $BUDFILES/run-mounts
  run_buildah 1 build $WITH_POLICY_JSON  -t secretnotreq -f $BUDFILES/run-mounts/Dockerfile.secret-required-false $BUDFILES/run-mounts
  expect_output --substring "No such file or directory"
  assert "$output" !~ "secret required but no secret with id mysecret found"
}

@test "bud with containerfile secret required" {
  _prefetch alpine

  run_buildah 125 build $WITH_POLICY_JSON  -t secretreq -f $BUDFILES/run-mounts/Dockerfile.secret-required $BUDFILES/run-mounts
  expect_output --substring "secret required but no secret with id mysecret found"

  # Also test secret required without value
  run_buildah 125 build $WITH_POLICY_JSON  -t secretreq -f $BUDFILES/run-mounts/Dockerfile.secret-required-wo-value $BUDFILES/run-mounts
  expect_output --substring "secret required but no secret with id mysecret found"
}

@test "bud with containerfile env secret" {
  export MYSECRET=SOMESECRETDATA
  run_buildah build --secret=id=mysecret,src=MYSECRET,type=env $WITH_POLICY_JSON  -t secretimg -f $BUDFILES/run-mounts/Dockerfile.secret $BUDFILES/run-mounts
  expect_output --substring "SOMESECRETDATA"

  run_buildah from secretimg
  run_buildah 1 run secretimg-working-container cat /run/secrets/mysecret
  expect_output --substring "cat: can't open '/run/secrets/mysecret': No such file or directory"
  run_buildah rm -a

  run_buildah build --secret=id=mysecret,env=MYSECRET $WITH_POLICY_JSON  -t secretimg -f $BUDFILES/run-mounts/Dockerfile.secret $BUDFILES/run-mounts
  expect_output --substring "SOMESECRETDATA"

  run_buildah from secretimg
  run_buildah 1 run secretimg-working-container cat /run/secrets/mysecret
  expect_output --substring "cat: can't open '/run/secrets/mysecret': No such file or directory"
  run_buildah rm -a
}

@test "bud with containerfile env secret priority" {
  _prefetch alpine
  mytmpdir=${TEST_SCRATCH_DIR}/my-dir1
  mkdir -p ${mytmpdir}
  cat > $mytmpdir/mysecret << _EOF
SOMESECRETDATA
_EOF

  export mysecret=ENVDATA
  run_buildah build --secret=id=mysecret $WITH_POLICY_JSON  -t secretimg -f $BUDFILES/run-mounts/Dockerfile.secret $BUDFILES/run-mounts
  expect_output --substring "ENVDATA"
}

@test "bud-multiple-platform-values" {
  skip "FIXME: #4396 - this test is broken, and is failing gating tests"
  outputlist=testlist
  # check if we can run a couple of 32-bit versions of an image, and if we can,
  # assume that emulation for other architectures is in place.
  os=`go env GOOS`
  run_buildah from $WITH_POLICY_JSON --name try-386 --platform=$os/386 alpine
  run_buildah '?' run try-386 true
  if test $status -ne 0 ; then
    skip "unable to run 386 container, assuming emulation is not available"
  fi
  run_buildah from $WITH_POLICY_JSON --name try-arm --platform=$os/arm alpine
  run_buildah '?' run try-arm true
  if test $status -ne 0 ; then
    skip "unable to run arm container, assuming emulation is not available"
  fi

  # build for those architectures - RUN gets exercised
  run_buildah build $WITH_POLICY_JSON --jobs=0 --platform=$os/arm,$os/386 --manifest $outputlist $BUDFILES/multiarch
  run_buildah manifest inspect $outputlist
  list="$output"
  run jq -r '.manifests[0].digest' <<< "$list"
  d1="$output"
  run jq -r '.manifests[1].digest' <<< "$list"
  d2="$output"
  assert "$d1" =~ "^sha256:[0-9a-f]{64}\$"
  assert "$d2" =~ "^sha256:[0-9a-f]{64}\$"
  assert "$d1" != "$d2" "digest(arm) should != digest(386)"
}

@test "bud-multiple-platform-no-partial-manifest-list" {
  outputlist=localhost/testlist
  run_buildah 1 bud $WITH_POLICY_JSON --platform=linux/arm,linux/amd64 --manifest $outputlist -f $BUDFILES/multiarch/Dockerfile.fail $BUDFILES/multiarch
  expect_output --substring "building at STEP \"RUN test .arch. = x86_64"
  run_buildah 125 manifest inspect $outputlist
  expect_output --substring "reading image .* pinging container registry"
}

@test "bud-multiple-platform-failure" {
  # check if we can run a couple of 32-bit versions of an image, and if we can,
  # assume that emulation for other architectures is in place.
  os=$(go env GOOS)
  if [[ "$os" != linux ]]; then
    skip "GOOS is '$os'; this test can only run on linux"
  fi
  run_buildah from $WITH_POLICY_JSON --name try-386 --platform=$os/386 alpine
  run_buildah '?' run try-386 true
  if test $status -ne 0 ; then
    skip "unable to run 386 container, assuming emulation is not available"
  fi
  run_buildah from $WITH_POLICY_JSON --name try-arm --platform=$os/arm alpine
  run_buildah '?' run try-arm true
  if test $status -ne 0 ; then
    skip "unable to run arm container, assuming emulation is not available"
  fi
  outputlist=localhost/testlist
  run_buildah 1 build $WITH_POLICY_JSON \
              --jobs=0 \
              --platform=linux/arm64,linux/amd64 \
              --manifest $outputlist \
              --build-arg SAFEIMAGE=$SAFEIMAGE \
              -f $BUDFILES/multiarch/Dockerfile.fail-multistage \
              $BUDFILES/multiarch
  expect_output --substring 'building at STEP "RUN false"'
}

@test "bud-multiple-platform-no-run" {
  outputlist=localhost/testlist
  run_buildah build $WITH_POLICY_JSON \
              --jobs=0 \
              --all-platforms \
              --manifest $outputlist \
              --build-arg SAFEIMAGE=$SAFEIMAGE \
              -f $BUDFILES/multiarch/Dockerfile.no-run \
              $BUDFILES/multiarch

  run_buildah manifest inspect $outputlist
  manifests=$(jq -r '.manifests[].platform.architecture' <<<"$output" |sort|fmt)
  assert "$manifests" = "amd64 arm64 ppc64le s390x" "arch list in manifest"
}

# attempts to resolve heading arg as base-image with --all-platforms
@test "bud-multiple-platform-with-base-as-default-arg" {
  outputlist=localhost/testlist
  run_buildah build $WITH_POLICY_JSON \
              --jobs=1 \
              --all-platforms \
              --manifest $outputlist \
              -f $BUDFILES/all-platform/Containerfile.default-arg \
              $BUDFILES/all-platform

  run_buildah manifest inspect $outputlist
  manifests=$(jq -r '.manifests[].platform.architecture' <<<"$output" |sort|fmt)
  assert "$manifests" = "386 amd64 arm arm arm64 ppc64le s390x" "arch list in manifest"
}

@test "bud-multiple-platform for --all-platform with additional-build-context" {
  outputlist=localhost/testlist
  local contextdir=${TEST_SCRATCH_DIR}/bud/platform
  mkdir -p $contextdir

cat > $contextdir/Dockerfile1 << _EOF
FROM busybox
_EOF

  # Pulled images must be $SAFEIMAGE since we configured --build-context
  run_buildah build $WITH_POLICY_JSON --all-platforms --build-context busybox=docker://$SAFEIMAGE --manifest $outputlist -f $contextdir/Dockerfile1
  # must contain pulling logs for $SAFEIMAGE instead of busybox
  expect_output --substring "STEP 1/1: FROM $SAFEIMAGE"
  assert "$output" =~ "\[linux/s390x\] COMMIT"
  assert "$output" =~ "\[linux/ppc64le\] COMMIT"
  assert "$output" !~ "busybox"

  # Confirm the manifests and their architectures. It is not possible for
  # this to change, unless we bump $SAFEIMAGE to a new versioned tag.
  run_buildah manifest inspect $outputlist
  manifests=$(jq -r '.manifests[].platform.architecture' <<<"$output" |sort|fmt)
  assert "$manifests" = "amd64 arm64 ppc64le s390x" "arch list in manifest"
}

# * Performs multi-stage build with label1=value1 and verifies
# * Relabels build with label1=value2 and verifies
# * Rebuild with label1=value1 and makes sure everything is used from cache
@test "bud-multistage-relabel" {
  _prefetch alpine busybox
  run_buildah inspect --format "{{.FromImageDigest}}" busybox
  fromDigest="$output"

  target=relabel
  run_buildah build --layers --label "label1=value1" $WITH_POLICY_JSON -t ${target} -f $BUDFILES/multi-stage-builds/Dockerfile.reused $BUDFILES/multi-stage-builds

  # Store base digest of first image
  run_buildah inspect --format '{{index .ImageAnnotations "org.opencontainers.image.base.digest" }}' ${target}
  firstDigest="$output"

  # Store image id of first build
  run_buildah inspect --format '{{ .FromImageID }}' ${target}
  firstImageID="$output"

  # Label of first build must contain label1:value1
  run_buildah inspect --format '{{ .Docker.ContainerConfig.Labels }}' ${target}
  expect_output --substring "label1:value1"

  # Rebuild with new label
  run_buildah build --layers --label "label1=value2" $WITH_POLICY_JSON -t ${target} -f $BUDFILES/multi-stage-builds/Dockerfile.reused $BUDFILES/multi-stage-builds

  # Base digest should match with first build
  run_buildah inspect --format '{{index .ImageAnnotations "org.opencontainers.image.base.digest" }}' ${target}
  expect_output "$firstDigest" "base digest from busybox"

  # Label of second build must contain label1:value2
  run_buildah inspect --format '{{ .Docker.ContainerConfig.Labels }}' ${target}
  expect_output --substring "label1:value2"

  # Rebuild everything with label1=value1 and everything should be cached from first image
  run_buildah build --layers --label "label1=value1" $WITH_POLICY_JSON -t ${target} -f $BUDFILES/multi-stage-builds/Dockerfile.reused $BUDFILES/multi-stage-builds

  # Entire image must be picked from cache
  run_buildah inspect --format '{{ .FromImageID }}' ${target}
  expect_output "$firstImageID" "Image ID cached from first build"

  run_buildah rmi -f ${target}
}


@test "bud-from-relabel" {
  _prefetch alpine busybox

  run_buildah inspect --format "{{.FromImageDigest}}" alpine
  alpineDigest="$output"

  run_buildah inspect --format "{{.FromImageDigest}}" busybox
  busyboxDigest="$output"

  target=relabel2
  run_buildah build --layers --label "label1=value1" --from=alpine -t ${target} $BUDFILES/from-scratch

  run_buildah inspect --format '{{index .ImageAnnotations "org.opencontainers.image.base.digest" }}' ${target}
  expect_output "$alpineDigest" "base digest from alpine"

  # Label of second build must contain label1:value1
  run_buildah inspect --format '{{ .Docker.ContainerConfig.Labels }}' ${target}
  expect_output --substring "label1:value1"


  run_buildah build --layers --label "label1=value2" --from=busybox -t ${target} $BUDFILES/from-scratch

  run_buildah inspect --format '{{index .ImageAnnotations "org.opencontainers.image.base.digest" }}' ${target}
  expect_output "$busyboxDigest" "base digest from busybox"

  # Label of second build must contain label1:value2
  run_buildah inspect --format '{{ .Docker.ContainerConfig.Labels }}' ${target}
  expect_output --substring "label1:value2"

  run_buildah rmi -f ${target}
}

@test "bud with run should not leave mounts behind cleanup test" {
  skip_if_in_container
  skip_if_no_podman

  # Create target dir where we will export tar
  target=cleanable
  local contextdir=${TEST_SCRATCH_DIR}/${target}
  mkdir $contextdir

  # Build and export container to tar
  run_buildah build --no-cache $WITH_POLICY_JSON -t ${target} -f $BUDFILES/containerfile/Containerfile.in $BUDFILES/containerfile
  podman export $(podman create --name ${target} --net=host ${target}) --output=$contextdir.tar

  # We are done exporting so remove images and containers which are not needed
  podman rm -f ${target}
  run_buildah rmi ${target}

  # Explode tar
  tar -xf $contextdir.tar -C $contextdir
  count=$(ls -A $contextdir/run | wc -l)
  ## exported /run should be empty
  assert "$count" == "0"
}

@test "bud with custom files in /run/ should persist cleanup test" {
  skip_if_in_container
  skip_if_no_podman

  # Create target dir where we will export tar
  target=cleanable
  local contextdir=${TEST_SCRATCH_DIR}/${target}
  mkdir $contextdir

  # Build and export container to tar
  run_buildah build --no-cache $WITH_POLICY_JSON -t ${target} -f $BUDFILES/add-run-dir/Dockerfile
  podman export $(podman create --name ${target} --net=host ${target}) --output=$contextdir.tar

  # We are done exporting so remove images and containers which are not needed
  podman rm -f ${target}
  run_buildah rmi ${target}

  # Explode tar
  tar -xf $contextdir.tar -C $contextdir
  count=$(ls -A $contextdir/run | wc -l)
  ## exported /run should not be empty
  assert "$count" == "1"
}

@test "bud-with-mount-like-buildkit" {
  skip_if_no_runtime
  skip_if_in_container
  local contextdir=$BUDFILES/buildkit-mount
  run_buildah build -t testbud $WITH_POLICY_JSON -f $contextdir/Dockerfile $contextdir/
  expect_output --substring "hello"
  run_buildah rmi -f testbud
}

@test "bud-with-mount-no-source-like-buildkit" {
  skip_if_no_runtime
  skip_if_in_container
  local contextdir=${TEST_SCRATCH_DIR}/buildkit-mount
  cp -R $BUDFILES/buildkit-mount $contextdir
  run_buildah build -t testbud $WITH_POLICY_JSON -f $contextdir/Dockerfile2 $contextdir/
  expect_output --substring "hello"
  run_buildah rmi -f testbud
}

@test "bud-with-mount-with-only-target-like-buildkit" {
  skip_if_no_runtime
  skip_if_in_container
  local contextdir=${TEST_SCRATCH_DIR}/buildkit-mount
  cp -R $BUDFILES/buildkit-mount $contextdir
  run_buildah build -t testbud $WITH_POLICY_JSON -f $contextdir/Dockerfile6 $contextdir/
  expect_output --substring "hello"
}

@test "bud-with-mount-no-subdir-like-buildkit" {
  skip_if_no_runtime
  skip_if_in_container
  local contextdir=${TEST_SCRATCH_DIR}/buildkit-mount
  cp -R $BUDFILES/buildkit-mount $contextdir
  run_buildah build -t testbud $WITH_POLICY_JSON -f $contextdir/Dockerfile $contextdir/subdir/
  expect_output --substring "hello"
  run_buildah rmi -f testbud
}

@test "bud-with-mount-relative-path-like-buildkit" {
  skip_if_no_runtime
  skip_if_in_container
  local contextdir=${TEST_SCRATCH_DIR}/buildkit-mount
  cp -R $BUDFILES/buildkit-mount $contextdir
  run_buildah build -t testbud $WITH_POLICY_JSON -f $contextdir/Dockerfile4 $contextdir/
  expect_output --substring "hello"
  run_buildah rmi -f testbud
}

@test "bud-with-mount-with-rw-like-buildkit" {
  skip_if_no_runtime
  skip_if_in_container
  local contextdir=${TEST_SCRATCH_DIR}/buildkit-mount
  cp -R $BUDFILES/buildkit-mount $contextdir
  run_buildah build --isolation chroot -t testbud $WITH_POLICY_JSON -f $contextdir/Dockerfile3 $contextdir/subdir/
  expect_output --substring "world"
  run_buildah rmi -f testbud
}

@test "bud-verify-if-we-dont-clean-prexisting-path" {
  skip_if_no_runtime
  skip_if_in_container
  run_buildah 1 build -t testbud $WITH_POLICY_JSON --secret id=secret-foo,src=$BUDFILES/verify-cleanup/secret1.txt -f $BUDFILES/verify-cleanup/Dockerfile $BUDFILES/verify-cleanup/
  expect_output --substring "hello"
  expect_output --substring "secrettext"
  expect_output --substring "Directory /tmp exists."
  expect_output --substring "Directory /var/tmp exists."
  expect_output --substring "Directory /testdir DOES NOT exists."
  expect_output --substring "Cache Directory /cachedir DOES NOT exists."
  expect_output --substring "Secret File secret1.txt DOES NOT exists."
  expect_output --substring "/tmp/hey: No such file or directory"
}

@test "bud-with-mount-with-tmpfs-like-buildkit" {
  skip_if_no_runtime
  skip_if_in_container
  local contextdir=${TEST_SCRATCH_DIR}/buildkit-mount
  cp -R $BUDFILES/buildkit-mount $contextdir
  # tmpfs mount: target should be available on container without creating any special directory on container
  run_buildah build -t testbud $WITH_POLICY_JSON -f $contextdir/Dockerfiletmpfs
  run_buildah rmi -f testbud
}

@test "bud-with-mount-with-tmpfs-with-copyup-like-buildkit" {
  skip_if_no_runtime
  skip_if_in_container
  local contextdir=${TEST_SCRATCH_DIR}/buildkit-mount
  cp -R $BUDFILES/buildkit-mount $contextdir
  run_buildah build -t testbud $WITH_POLICY_JSON -f $contextdir/Dockerfiletmpfscopyup
  expect_output --substring "certs"
  run_buildah rmi -f testbud
}

@test "bud-with-mount-cache-like-buildkit" {
  skip_if_no_runtime
  skip_if_in_container
  local contextdir=${TEST_SCRATCH_DIR}/buildkit-mount
  cp -R $BUDFILES/buildkit-mount $contextdir
  # try writing something to persistent cache
  run_buildah build -t testbud $WITH_POLICY_JSON -f $contextdir/Dockerfilecachewrite
  # try reading something from persistent cache in a different build
  run_buildah build -t testbud2 $WITH_POLICY_JSON -f $contextdir/Dockerfilecacheread
  expect_output --substring "hello"
  run_buildah rmi -f testbud
  run_buildah rmi -f testbud2
}

@test "bud-with-mount-cache-like-buildkit with buildah prune should clear the cache" {
  skip_if_no_runtime
  skip_if_in_container
  local contextdir=${TEST_SCRATCH_DIR}/buildkit-mount
  cp -R $BUDFILES/buildkit-mount $contextdir
  # try writing something to persistent cache
  run_buildah build -t testbud $WITH_POLICY_JSON -f $contextdir/Dockerfilecachewrite
  # prune the mount cache
  run_buildah prune
  # try reading something from persistent cache in a different build
  run_buildah 1 build -t testbud2 $WITH_POLICY_JSON -f $contextdir/Dockerfilecacheread
  expect_output --substring "No such file or directory"
  run_buildah rmi -f testbud
}

@test "bud-with-mount-cache-like-buildkit-verify-default-selinux-option" {
  skip_if_no_runtime
  skip_if_in_container
  _prefetch alpine
  # try writing something to persistent cache
  TMPDIR=${TEST_SCRATCH_DIR} run_buildah build -t testbud $WITH_POLICY_JSON -f $BUDFILES/buildkit-mount/Dockerfilecachewritewithoutz
  # try reading something from persistent cache in a different build
  TMPDIR=${TEST_SCRATCH_DIR} run_buildah build -t testbud2 $WITH_POLICY_JSON -f $BUDFILES/buildkit-mount/Dockerfilecachereadwithoutz
  buildah_cache_dir="${TEST_SCRATCH_DIR}/buildah-cache-$UID"
  # buildah cache parent must have been created for our uid specific to this test
  test -d "$buildah_cache_dir"
  expect_output --substring "hello"
}

@test "bud-with-mount-cache-like-buildkit-locked-across-steps" {
  # Note: this test is just testing syntax for sharing, actual behaviour test needs parallel build in order to test locking.
  skip_if_no_runtime
  skip_if_in_container
  local contextdir=${TEST_SCRATCH_DIR}/buildkit-mount
  cp -R $BUDFILES/buildkit-mount $contextdir
  # try writing something to persistent cache
  run_buildah build -t testbud $WITH_POLICY_JSON -f $contextdir/Dockerfilecachewritesharing
  expect_output --substring "world"
  run_buildah rmi -f testbud
}

@test "bud-with-multiple-mount-keeps-default-bind-mount" {
  skip_if_no_runtime
  skip_if_in_container
  local contextdir=${TEST_SCRATCH_DIR}/buildkit-mount
  cp -R $BUDFILES/buildkit-mount $contextdir
  run_buildah build -t testbud $WITH_POLICY_JSON -f $contextdir/Dockerfilemultiplemounts $contextdir/
  expect_output --substring "hello"
  run_buildah rmi -f testbud
}

@test "bud with user in groups" {
  target=bud-group
  run_buildah build $WITH_POLICY_JSON -t ${target} $BUDFILES/group
}

@test "build proxy" {
  _prefetch alpine
  mytmpdir=${TEST_SCRATCH_DIR}/my-dir
  mkdir -p $mytmpdir
  cat > $mytmpdir/Containerfile << _EOF
FROM alpine
run printenv
_EOF
  target=env-image
  check="FTP_PROXY="FTP" ftp_proxy=ftp http_proxy=http HTTPS_PROXY=HTTPS"
  bogus="BOGUS_PROXY=BOGUS"
  eval $check $bogus run_buildah build --unsetenv PATH $WITH_POLICY_JSON -t oci-${target} -f $mytmpdir/Containerfile .
  for i in $check; do
    expect_output --substring "$i" "Environment variables available within build"
  done
  if [ -n "$(grep "$bogus" <<< "$output")" ]; then
    die "Unexpected bogus environment."
  fi
}

@test "bud-with-mount-bind-from-like-buildkit" {
  skip_if_no_runtime
  skip_if_in_container
  local contextdir=${TEST_SCRATCH_DIR}/buildkit-mount-from
  cp -R $BUDFILES/buildkit-mount-from $contextdir
  # build base image which we will use as our `from`
  run_buildah build -t buildkitbase $WITH_POLICY_JSON -f $contextdir/Dockerfilebuildkitbase $contextdir/
  # try reading something from another image in a different build
  run_buildah build -t testbud $WITH_POLICY_JSON -f $contextdir/Dockerfilebindfrom
  expect_output --substring "hello"
  run_buildah rmi -f buildkitbase
  run_buildah rmi -f testbud
}

@test "bud-with-writeable-mount-bind-from-like-buildkit" {
  skip_if_no_runtime
  skip_if_in_container
  local contextdir=${TEST_SCRATCH_DIR}/buildkit-mount-from
  cp -R $BUDFILES/buildkit-mount-from $contextdir
  # build base image which we will use as our `from`
  run_buildah build -t buildkitbase $WITH_POLICY_JSON -f $contextdir/Dockerfilebuildkitbase $contextdir/
  # try reading something from another image in a different build
  run_buildah build -t testbud $WITH_POLICY_JSON -f $contextdir/Dockerfilebindfromwriteable
  expect_output --substring "world"
  run_buildah rmi -f buildkitbase
  run_buildah rmi -f testbud
}

@test "bud-with-mount-bind-from-without-source-like-buildkit" {
  skip_if_no_runtime
  skip_if_in_container
  local contextdir=${TEST_SCRATCH_DIR}/buildkit-mount-from
  cp -R $BUDFILES/buildkit-mount-from $contextdir
  # build base image which we will use as our `from`
  run_buildah build -t buildkitbase $WITH_POLICY_JSON -f $contextdir/Dockerfilebuildkitbase $contextdir/
  # try reading something from another image in a different build
  run_buildah build -t testbud $WITH_POLICY_JSON -f $contextdir/Dockerfilebindfromwithoutsource
  expect_output --substring "hello"
  run_buildah rmi -f buildkitbase
  run_buildah rmi -f testbud
}

@test "bud-with-mount-bind-from-with-empty-from-like-buildkit" {
  skip_if_no_runtime
  skip_if_in_container
  local contextdir=${TEST_SCRATCH_DIR}/buildkit-mount-from
  cp -R $BUDFILES/buildkit-mount-from $contextdir
  # build base image which we will use as our `from`
  run_buildah build -t buildkitbase $WITH_POLICY_JSON -f $contextdir/Dockerfilebuildkitbase $contextdir/
  # try reading something from image in a different build
  run_buildah 125 build -t testbud $WITH_POLICY_JSON -f $contextdir/Dockerfilebindfromwithemptyfrom
  expect_output --substring "points to an empty value"
  run_buildah rmi -f buildkitbase
}

@test "bud-with-mount-cache-from-like-buildkit" {
  skip_if_no_runtime
  skip_if_in_container
  local contextdir=${TEST_SCRATCH_DIR}/buildkit-mount-from
  cp -R $BUDFILES/buildkit-mount-from $contextdir
  # try reading something from persistent cache in a different build
  run_buildah build -t testbud $WITH_POLICY_JSON -f $contextdir/Dockerfilecachefrom $contextdir/
  expect_output --substring "hello"
  run_buildah rmi -f testbud
}

# following test must fail
@test "bud-with-mount-cache-image-from-like-buildkit" {
  skip_if_no_runtime
  skip_if_in_container
  local contextdir=${TEST_SCRATCH_DIR}/buildkit-mount-from
  cp -R $BUDFILES/buildkit-mount-from $contextdir

  # build base image which we will use as our `from`
  run_buildah build -t buildkitbase $WITH_POLICY_JSON -f $contextdir/Dockerfilebuildkitbase $contextdir/

  # try reading something from persistent cache in a different build
  run_buildah 125 build -t testbud $WITH_POLICY_JSON -f $contextdir/Dockerfilecachefromimage
  expect_output --substring "no stage found with name buildkitbase"
  run_buildah rmi -f buildkitbase
}

@test "bud-with-mount-cache-multiple-from-like-buildkit" {
  skip_if_no_runtime
  skip_if_in_container
  local contextdir=${TEST_SCRATCH_DIR}/buildkit-mount-from
  cp -R $BUDFILES/buildkit-mount-from $contextdir
  # try reading something from persistent cache in a different build
  run_buildah build -t testbud $WITH_POLICY_JSON -f $contextdir/Dockerfilecachemultiplefrom $contextdir/
  expect_output --substring "hello"
  expect_output --substring "hello2"
  run_buildah rmi -f testbud
}

@test "bud-with-mount-bind-from-relative-like-buildkit" {
  skip_if_no_runtime
  skip_if_in_container
  local contextdir=${TEST_SCRATCH_DIR}/buildkit-mount-from
  cp -R $BUDFILES/buildkit-mount-from $contextdir
  # build base image which we will use as our `from`
  run_buildah build -t buildkitbaserelative $WITH_POLICY_JSON -f $contextdir/Dockerfilebuildkitbaserelative $contextdir/
  # try reading something from image in a different build
  run_buildah build -t testbud $WITH_POLICY_JSON -f $contextdir/Dockerfilebindfromrelative
  expect_output --substring "hello"
  run_buildah rmi -f buildkitbaserelative
  run_buildah rmi -f testbud
}

@test "bud-with-mount-bind-from-multistage-relative-like-buildkit" {
  local contextdir=${TEST_SCRATCH_DIR}/buildkit-mount-from
  cp -R $BUDFILES/buildkit-mount-from $contextdir
  skip_if_no_runtime
  skip_if_in_container
  # build base image which we will use as our `from`
  run_buildah build -t testbud $WITH_POLICY_JSON -f $contextdir/Dockerfilemultistagefrom $contextdir/
  expect_output --substring "hello"
  run_buildah rmi -f testbud
}

@test "bud-with-mount-bind-from-cache-multistage-relative-like-buildkit" {
  skip_if_no_runtime
  skip_if_in_container
  local contextdir=${TEST_SCRATCH_DIR}/buildkit-mount-from
  cp -R $BUDFILES/buildkit-mount-from $contextdir
  # build base image which we will use as our `from`
  run_buildah build -t testbud $WITH_POLICY_JSON -f $contextdir/Dockerfilemultistagefromcache $contextdir/
  expect_output --substring "hello"
  expect_output --substring "hello2"
  run_buildah rmi -f testbud
}

@test "bud with network names" {
  skip_if_no_runtime
  skip_if_in_container
  skip_if_rootless_environment

  _prefetch alpine

  run_buildah 125 bud $WITH_POLICY_JSON --network notexists $BUDFILES/network
  expect_output --substring "network not found"

  if test "$BUILDAH_ISOLATION" = "oci"; then
    run_buildah bud $WITH_POLICY_JSON --network podman $BUDFILES/network
    # default subnet is 10.88.0.0/16
    expect_output --substring "10.88."
  fi
}

@test "bud with --network slirp4netns" {
  skip_if_no_runtime
  skip_if_in_container
  skip_if_chroot

  _prefetch alpine

  run_buildah bud $WITH_POLICY_JSON --network slirp4netns $BUDFILES/network
  # default subnet is 10.0.2.100/24
  assert "$output" =~ "10.0.2.100/24" "ip addr shows default subnet"

  run_buildah bud $WITH_POLICY_JSON --network slirp4netns:cidr=192.168.255.0/24,mtu=2000 $BUDFILES/network
  assert "$output" =~ "192.168.255.100/24" "ip addr shows custom subnet"
  assert "$output" =~ "mtu 2000" "ip addr shows mtu 2000"
}

@test "bud with --network pasta" {
  skip_if_no_runtime
  skip_if_chroot
  skip_if_root_environment "pasta only works rootless"

  # FIXME: unskip when we have a new pasta version with:
  # https://archives.passt.top/passt-dev/20230623082531.25947-2-pholzing@redhat.com/
  skip "pasta bug prevents this from working"

  _prefetch alpine

  # pasta by default copies the host ip
  ip=$(hostname -I | cut -f 1 -d " ")

  run_buildah bud $WITH_POLICY_JSON --network pasta $BUDFILES/network
  assert "$output" =~ "$ip" "ip addr shows default subnet"

  # check some entwork options, it accepts raw pasta(1) areguments
  mac="9a:dd:31:ea:92:98"
  run_buildah bud $WITH_POLICY_JSON --network pasta:--mtu,2000,--ns-mac-addr,"$mac" $BUDFILES/network
  assert "$output" =~ "$mac" "ip addr shows custom mac address"
  assert "$output" =~ "mtu 2000" "ip addr shows mtu 2000"
}

@test "bud WORKDIR owned by USER" {
  _prefetch alpine
  target=alpine-image
  ctr=alpine-ctr
  run_buildah build $WITH_POLICY_JSON -t ${target} $BUDFILES/workdir-user
  expect_output --substring "1000:1000 /home/http/public"
}

function build_signalled {
  skip_if_no_runtime

  _prefetch alpine

  mkfifo ${TEST_SCRATCH_DIR}/pipe
  # start the build running in the background - don't use the function wrapper because that sets '$!' to a value that's not what we want
  ${BUILDAH_BINARY} ${BUILDAH_REGISTRY_OPTS} ${ROOTDIR_OPTS} $WITH_POLICY_JSON build $BUDFILES/long-sleep > ${TEST_SCRATCH_DIR}/pipe 2>&1 &
  buildah_pid="${!}"
  echo buildah is pid ${buildah_pid}
  # save what's written to the fifo to a plain file
  coproc cat ${TEST_SCRATCH_DIR}/pipe > ${TEST_SCRATCH_DIR}/log
  cat_pid="${COPROC_PID}"
  echo cat is pid ${cat_pid}
  # kill the buildah process early
  sleep 30
  kill -s ${1} "${buildah_pid}"
  # wait for output to stop getting written from anywhere
  wait "${buildah_pid}" "${cat_pid}"
  echo log:
  cat ${TEST_SCRATCH_DIR}/log
  echo checking:
  ! grep 'not fully killed' ${TEST_SCRATCH_DIR}/log
}

@test "build interrupted" {
  build_signalled SIGINT
}

@test "build terminated" {
  build_signalled SIGTERM
}

@test "build killed" {
  build_signalled SIGKILL
}

@test "build-multiple-parse" {
  _prefetch alpine
  echo 'FROM alpine' | tee ${TEST_SCRATCH_DIR}/Dockerfile1
  echo '# escape=|\nFROM alpine' | tee ${TEST_SCRATCH_DIR}/Dockerfile2
  run_buildah 125 build -f ${TEST_SCRATCH_DIR}/Dockerfile1 -f ${TEST_SCRATCH_DIR}/Dockerfile2 ${TEST_SCRATCH_DIR}
  assert "$output" =~ "parsing additional Dockerfile .*Dockerfile2: invalid ESCAPE"
}

@test "build-with-network-test" {
  skip_if_in_container # Test only works in OCI isolation, which doesn't work in CI/CD systems. Buildah defaults to chroot isolation

  image="quay.io/libpod/alpine_nginx:latest"
  _prefetch $image
  cat > ${TEST_SCRATCH_DIR}/Containerfile << _EOF
FROM $image
RUN curl -k -o /dev/null http://www.redhat.com:80
_EOF

  # curl results show success
  run_buildah build ${WITH_POLICY_JSON} ${TEST_SCRATCH_DIR}

  # A proper test would use ping or nc, and check for ENETUNREACH.
  # But in a tightly firewalled environment, even the expected-success
  # test will fail. A not-quite-equivalent workaround is to use curl
  # and hope that $http_proxy is set; we then rely on curl to fail
  # in a slightly different way
  expect_rc=6
  expect_err="Could not resolve host: www.redhat.com"
  if [[ $http_proxy != "" ]]; then
    expect_rc=5
    expect_err="Could not resolve proxy:"
  fi
  run_buildah $expect_rc build --network=none ${WITH_POLICY_JSON} ${TEST_SCRATCH_DIR}
  expect_output --substring "$expect_err"
}

@test "build-with-no-new-privileges-test" {
  _prefetch alpine
  cat > ${TEST_SCRATCH_DIR}/Containerfile << _EOF
FROM alpine
RUN grep NoNewPrivs /proc/self/status
_EOF

  run_buildah build --security-opt no-new-privileges $WITH_POLICY_JSON ${TEST_SCRATCH_DIR}
  expect_output --substring "NoNewPrivs:.*1"
}

@test "build --group-add" {
  skip_if_no_runtime
  id=$RANDOM

  _prefetch alpine
  cat > ${TEST_SCRATCH_DIR}/Containerfile << _EOF
FROM alpine
RUN id -G
_EOF

  run_buildah build --group-add $id $WITH_POLICY_JSON ${TEST_SCRATCH_DIR}
  expect_output --substring "$id"

  if is_rootless && has_supplemental_groups; then
     run_buildah build --group-add keep-groups $WITH_POLICY_JSON ${TEST_SCRATCH_DIR}
     expect_output --substring "65534"
  fi
}

@test "build-env-precedence" {
  skip_if_no_runtime

  _prefetch alpine

  run_buildah build --no-cache --env E=F --env G=H --env I=J --env K=L -f ${BUDFILES}/env/Dockerfile.env-precedence ${BUDFILES}/env
  expect_output --substring "a=b c=d E=F G=H"
  expect_output --substring "a=b c=d E=E G=G"
  expect_output --substring "w=x y=z I=J K=L"
  expect_output --substring "w=x y=z I=I K=K"

  run_buildah build --no-cache --layers --env E=F --env G=H --env I=J --env K=L -f ${BUDFILES}/env/Dockerfile.env-precedence ${BUDFILES}/env
  expect_output --substring "a=b c=d E=F G=H"
  expect_output --substring "a=b c=d E=E G=G"
  expect_output --substring "w=x y=z I=J K=L"
  expect_output --substring "w=x y=z I=I K=K"
}

@test "build prints 12-digit hash" {
  run_buildah build -t test -f $BUDFILES/containerfile/Containerfile .
  regex='--> [0-9a-zA-Z]{12}'
  if ! [[ $output =~ $regex ]]; then
    false
  fi
}

@test "build with name path changes" {
  _prefetch busybox
  run_buildah build --no-cache --quiet --pull=false $WITH_POLICY_JSON -t foo/bar $BUDFILES/commit/name-path-changes/
  run_buildah build --no-cache --quiet --pull=false $WITH_POLICY_JSON -t bar $BUDFILES/commit/name-path-changes/
  run_buildah images
  expect_output --substring "localhost/foo/bar"
  expect_output --substring "localhost/bar"
}
