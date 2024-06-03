#!/usr/bin/env bats

load helpers

@test "commit-flags-order-verification" {
  run_buildah 125 commit cnt1 --tls-verify
  check_options_flag_err "--tls-verify"

  run_buildah 125 commit cnt1 -q
  check_options_flag_err "-q"

  run_buildah 125 commit cnt1 -f=docker --quiet --creds=bla:bla
  check_options_flag_err "-f=docker"

  run_buildah 125 commit cnt1 --creds=bla:bla
  check_options_flag_err "--creds=bla:bla"
}

@test "commit" {
  _prefetch alpine
  run_buildah from --quiet --pull=false $WITH_POLICY_JSON alpine
  cid=$output
  run_buildah commit $WITH_POLICY_JSON $cid alpine-image
  run_buildah images alpine-image
}

# Mainly this test is added for rootless setups where XDG_RUNTIME_DIR
# is not set and we end up setting incorrect runroot at various steps
# Use case is typically seen on environments where current session
# is invalid login session.
@test "commit image on rootless setup with mount" {
  unset XDG_RUNTIME_DIR
  run dd if=/dev/zero of=${TEST_SCRATCH_DIR}/file count=1 bs=10M
  run_buildah from scratch
  CONT=$output
  unset XDG_RUNTIME_DIR
  run_buildah mount $CONT
  MNT=$output
  run cp ${TEST_SCRATCH_DIR}/file $MNT/file
  run_buildah umount $CONT
  run_buildah commit $CONT foo
  run_buildah images foo
  expect_output --substring "10.5 MB"
}

@test "commit-with-remove-identity-label" {
  _prefetch alpine
  run_buildah from --quiet --pull=false $WITH_POLICY_JSON alpine
  cid=$output
  run_buildah commit --identity-label=false $WITH_POLICY_JSON $cid alpine-image
  run_buildah images alpine-image
  run_buildah inspect --format '{{printf "%q" .Docker.Config.Labels}}' alpine-image
  expect_output "map[]"
}

@test "commit format test" {
  _prefetch alpine
  run_buildah from --quiet --pull=false $WITH_POLICY_JSON alpine
  cid=$output
  run_buildah commit $WITH_POLICY_JSON $cid alpine-image-oci
  run_buildah commit --format docker --disable-compression=false $WITH_POLICY_JSON $cid alpine-image-docker

  run_buildah inspect --type=image --format '{{.Manifest}}' alpine-image-oci
  mediatype=$(jq -r '.layers[0].mediaType' <<<"$output")
  expect_output --from="$mediatype" "application/vnd.oci.image.layer.v1.tar"
  run_buildah inspect --type=image --format '{{.Manifest}}' alpine-image-docker
  mediatype=$(jq -r '.layers[1].mediaType' <<<"$output")
  expect_output --from="$mediatype" "application/vnd.docker.image.rootfs.diff.tar.gzip"
}

@test "commit --unsetenv PATH" {
  _prefetch alpine
  run_buildah from --quiet --pull=false $WITH_POLICY_JSON alpine
  cid=$output
  run_buildah commit --unsetenv PATH $WITH_POLICY_JSON $cid alpine-image-oci
  run_buildah commit --unsetenv PATH --format docker --disable-compression=false $WITH_POLICY_JSON $cid alpine-image-docker

  run_buildah inspect --type=image --format '{{.OCIv1.Config.Env}}' alpine-image-oci
  expect_output "[]" "No Path should be defined"
  run_buildah inspect --type=image --format '{{.Docker.Config.Env}}' alpine-image-docker
  expect_output "[]" "No Path should be defined"
}

@test "commit quiet test" {
  _prefetch alpine
  run_buildah from --quiet --pull=false $WITH_POLICY_JSON alpine
  cid=$output
  run_buildah commit --iidfile /dev/null $WITH_POLICY_JSON -q $cid alpine-image
  expect_output ""
}

@test "commit rm test" {
  _prefetch alpine
  run_buildah from --quiet --pull=false $WITH_POLICY_JSON alpine
  cid=$output
  run_buildah commit $WITH_POLICY_JSON --rm $cid alpine-image
  run_buildah 125 rm $cid
  expect_output --substring "removing container \"alpine-working-container\": container not known"
}

@test "commit-alternate-storage" {
  _prefetch alpine
  echo FROM
  run_buildah from --quiet --pull=false $WITH_POLICY_JSON alpine
  cid=$output
  echo COMMIT
  run_buildah commit $WITH_POLICY_JSON $cid "containers-storage:[vfs@${TEST_SCRATCH_DIR}/root2+${TEST_SCRATCH_DIR}/runroot2]newimage"
  echo FROM
  run_buildah --storage-driver vfs --root ${TEST_SCRATCH_DIR}/root2 --runroot ${TEST_SCRATCH_DIR}/runroot2 from $WITH_POLICY_JSON newimage
}

@test "commit-rejected-name" {
  _prefetch alpine
  run_buildah from --quiet --pull=false $WITH_POLICY_JSON alpine
  cid=$output
  run_buildah 125 commit $WITH_POLICY_JSON $cid ThisNameShouldBeRejected
  expect_output --substring "must be lower"
}

@test "commit-no-empty-created-by" {
  if ! python3 -c 'import json, sys' 2> /dev/null ; then
    skip "python interpreter with json module not found"
  fi
  target=new-image
  _prefetch alpine
  run_buildah from --quiet --pull=false $WITH_POLICY_JSON alpine
  cid=$output

  run_buildah config --created-by "untracked actions" $cid
  run_buildah commit $WITH_POLICY_JSON $cid ${target}
  run_buildah inspect --format '{{.Config}}' ${target}
  config="$output"
  run python3 -c 'import json, sys; config = json.load(sys.stdin); print(config["history"][len(config["history"])-1]["created_by"])' <<< "$config"
  echo "$output"
  assert "$status" -eq 0 "status from python command 1"
  expect_output "untracked actions"

  run_buildah config --created-by "" $cid
  run_buildah commit $WITH_POLICY_JSON $cid ${target}
  run_buildah inspect --format '{{.Config}}' ${target}
  config="$output"
  run python3 -c 'import json, sys; config = json.load(sys.stdin); print(config["history"][len(config["history"])-1]["created_by"])' <<< "$config"
  echo "$output"
  assert "$status" -eq 0 "status from python command 2"
  expect_output "/bin/sh"
}

@test "commit-no-name" {
  _prefetch alpine
  run_buildah from --quiet --pull=false $WITH_POLICY_JSON alpine
  cid=$output
  run_buildah commit $WITH_POLICY_JSON $cid
}

@test "commit should fail with nonexistent authfile" {
  _prefetch alpine
  run_buildah from --quiet --pull $WITH_POLICY_JSON alpine
  cid=$output
  run_buildah 125 commit --authfile /tmp/nonexistent $WITH_POLICY_JSON $cid alpine-image
}

@test "commit-builder-identity" {
	_prefetch alpine
	run_buildah from --quiet --pull $WITH_POLICY_JSON alpine
	cid=$output
	run_buildah commit $WITH_POLICY_JSON $cid alpine-image

	run_buildah --version
        local -a output_fields=($output)
	buildah_version=${output_fields[2]}

	run_buildah inspect --format '{{ index .Docker.Config.Labels "io.buildah.version"}}' alpine-image
        expect_output "$buildah_version"
}

@test "commit-container-id" {
  _prefetch alpine
  run_buildah from --quiet --pull $WITH_POLICY_JSON alpine

  # There is exactly one container. Get its ID.
  run_buildah containers --format '{{.ContainerID}}'
  cid=$output

  run_buildah commit $WITH_POLICY_JSON --format docker $cid alpine-image
  run_buildah inspect --format '{{.Docker.Container}}' alpine-image
  expect_output "$cid" "alpine-image -> .Docker.Container"
}

@test "commit with name" {
  _prefetch busybox
  run_buildah from --quiet $WITH_POLICY_JSON --name busyboxc busybox
  expect_output "busyboxc"

  # Commit with a new name
  newname="commitbyname/busyboxname"
  run_buildah commit $WITH_POLICY_JSON busyboxc $newname

  run_buildah from $WITH_POLICY_JSON localhost/$newname
  expect_output "busyboxname-working-container"

  cname=$output
  run_buildah inspect --format '{{.FromImage}}' $cname
  expect_output "localhost/$newname:latest"
}

@test "commit to docker-distribution" {
  _prefetch busybox
  run_buildah from $WITH_POLICY_JSON --name busyboxc busybox
  start_registry
  run_buildah commit $WITH_POLICY_JSON --tls-verify=false --creds testuser:testpassword busyboxc docker://localhost:${REGISTRY_PORT}/commit/busybox
  run_buildah from $WITH_POLICY_JSON --name fromdocker --tls-verify=false --creds testuser:testpassword docker://localhost:${REGISTRY_PORT}/commit/busybox
}

@test "commit encrypted local oci image" {
  skip_if_rootless_environment
  _prefetch busybox
  mkdir ${TEST_SCRATCH_DIR}/tmp
  openssl genrsa -out ${TEST_SCRATCH_DIR}/tmp/mykey.pem 1024
  openssl rsa -in ${TEST_SCRATCH_DIR}/tmp/mykey.pem -pubout > ${TEST_SCRATCH_DIR}/tmp/mykey.pub
  run_buildah from --quiet --pull=false $WITH_POLICY_JSON busybox
  cid=$output
  run_buildah commit --iidfile /dev/null $WITH_POLICY_JSON --encryption-key jwe:${TEST_SCRATCH_DIR}/tmp/mykey.pub -q $cid oci:${TEST_SCRATCH_DIR}/tmp/busybox_enc
  imgtype  -show-manifest oci:${TEST_SCRATCH_DIR}/tmp/busybox_enc | grep "+encrypted"
  rm -rf ${TEST_SCRATCH_DIR}/tmp
}

@test "commit oci encrypt to registry" {
  _prefetch busybox
  mkdir ${TEST_SCRATCH_DIR}/tmp
  openssl genrsa -out ${TEST_SCRATCH_DIR}/tmp/mykey.pem 1024
  openssl rsa -in ${TEST_SCRATCH_DIR}/tmp/mykey.pem -pubout > ${TEST_SCRATCH_DIR}/tmp/mykey.pub
  start_registry
  run_buildah from --quiet --pull=false $WITH_POLICY_JSON busybox
  cid=$output
  run_buildah commit --iidfile /dev/null --tls-verify=false --creds testuser:testpassword $WITH_POLICY_JSON --encryption-key jwe:${TEST_SCRATCH_DIR}/tmp/mykey.pub -q $cid docker://localhost:${REGISTRY_PORT}/buildah/busybox_encrypted:latest
  # this test, just checks the ability to commit an image to a registry
  # there is no good way to test the details of the image unless with ./buildah pull, test will be in pull.bats
  rm -rf ${TEST_SCRATCH_DIR}/tmp

  # verify that encrypted layers are not cached or reused for an non-encrypted image (See containers/image#1533)
  run_buildah commit --iidfile /dev/null --tls-verify=false --creds testuser:testpassword $WITH_POLICY_JSON -q $cid docker://localhost:${REGISTRY_PORT}/buildah/busybox_not_encrypted:latest
  run_buildah from $WITH_POLICY_JSON --tls-verify=false --creds testuser:testpassword docker://localhost:${REGISTRY_PORT}/buildah/busybox_not_encrypted:latest
}

@test "commit omit-timestamp" {
  _prefetch busybox
  run_buildah from --quiet --pull=false $WITH_POLICY_JSON busybox
  cid=$output
  run_buildah run $cid touch /test
  run_buildah commit $WITH_POLICY_JSON --omit-timestamp -q $cid omit
  run_buildah inspect --format '{{ .Docker.Created }}' omit
  expect_output --substring "1970-01-01"
  run_buildah inspect --format '{{ .OCIv1.Created }}' omit
  expect_output --substring "1970-01-01"


  run_buildah from --quiet --pull=false $WITH_POLICY_JSON omit
  cid=$output
  run_buildah run $cid ls -l /test
  expect_output --substring "1970"

  rm -rf ${TEST_SCRATCH_DIR}/tmp
}

@test "commit timestamp" {
  _prefetch busybox
  run_buildah from --quiet --pull=false $WITH_POLICY_JSON busybox
  cid=$output
  run_buildah run $cid touch /test
  run_buildah commit $WITH_POLICY_JSON --timestamp 0 -q $cid omit
  run_buildah inspect --format '{{ .Docker.Created }}' omit
  expect_output --substring "1970-01-01"
  run_buildah inspect --format '{{ .OCIv1.Created }}' omit
  expect_output --substring "1970-01-01"


  run_buildah from --quiet --pull=false $WITH_POLICY_JSON omit
  cid=$output
  run_buildah run $cid ls -l /test
  expect_output --substring "1970"

  rm -rf ${TEST_SCRATCH_DIR}/tmp
}

@test "commit with authfile" {
  _prefetch busybox
  run_buildah from --quiet --pull=false $WITH_POLICY_JSON busybox
  cid=$output
  run_buildah run $cid touch /test

  start_registry
  run_buildah login --authfile ${TEST_SCRATCH_DIR}/test.auth --username testuser --password testpassword --tls-verify=false localhost:${REGISTRY_PORT}
  run_buildah commit --authfile ${TEST_SCRATCH_DIR}/test.auth $WITH_POLICY_JSON --tls-verify=false $cid docker://localhost:${REGISTRY_PORT}/buildah/my-busybox
  expect_output --substring "Writing manifest to image destination"
}

@test "commit-without-names" {
  _prefetch busybox
  run_buildah from --quiet --pull=false $WITH_POLICY_JSON busybox
  cid=$output
  run_buildah run $cid touch /testfile
  run_buildah run $cid chown $(id -u):$(id -g) /testfile
  run_buildah commit $cid dir:${TEST_SCRATCH_DIR}/new-image
  config=$(jq -r .config.digest ${TEST_SCRATCH_DIR}/new-image/manifest.json)
  echo "config blob is $config"
  diffid=$(jq -r '.rootfs.diff_ids[-1]' ${TEST_SCRATCH_DIR}/new-image/${config##*:})
  echo "new layer is $diffid"
  run_buildah copy $cid ${TEST_SCRATCH_DIR}/new-image/${diffid##*:} /testdiff.tar
  # use in-container version of tar to avoid worrying about differences in
  # output formats between tar implementations
  run_buildah run $cid tar tvf /testdiff.tar testfile
  echo "new file looks like [$output]"
  # ownership information should be forced to be in number/number format
  # instead of name/name because the names are gone
  assert "$output" =~ $(id -u)/$(id -g)
}

@test "commit-with-extra-files" {
  _prefetch busybox
  run_buildah from --quiet --pull=false $WITH_POLICY_JSON busybox
  cid=$output
  createrandom ${BATS_TMPDIR}/randomfile1
  createrandom ${BATS_TMPDIR}/randomfile2

  for method in --squash=false --squash=true ; do
    run_buildah commit $method --add-file ${BATS_TMPDIR}/randomfile1:/randomfile1 $cid with-random-1
    run_buildah commit $method --add-file ${BATS_TMPDIR}/randomfile2:/in-a-subdir/randomfile2 $cid with-random-2
    run_buildah commit $method --add-file ${BATS_TMPDIR}/randomfile1:/randomfile1 --add-file ${BATS_TMPDIR}/randomfile2:/in-a-subdir/randomfile2 $cid with-random-both

    # first one should have the first file and not the second, and the shell should be there
    run_buildah from --quiet --pull=false $WITH_POLICY_JSON with-random-1
    cid=$output
    run_buildah mount $cid
    mountpoint=$output
    test -s $mountpoint/bin/sh || test -L $mountpoint/bin/sh
    cmp ${BATS_TMPDIR}/randomfile1 $mountpoint/randomfile1
    run stat -c %u:%g $mountpoint
    [ $status -eq 0 ]
    rootowner=$output
    run stat -c %u:%g:%A $mountpoint/randomfile1
    [ $status -eq 0 ]
    assert ${rootowner}:-rw-r--r--
    ! test -f $mountpoint/randomfile2

    # second one should have the second file and not the first, and the shell should be there
    run_buildah from --quiet --pull=false $WITH_POLICY_JSON with-random-2
    cid=$output
    run_buildah mount $cid
    mountpoint=$output
    test -s $mountpoint/bin/sh || test -L $mountpoint/bin/sh
    cmp ${BATS_TMPDIR}/randomfile2 $mountpoint/in-a-subdir/randomfile2
    run stat -c %u:%g $mountpoint
    [ $status -eq 0 ]
    rootowner=$output
    run stat -c %u:%g:%A $mountpoint/in-a-subdir/randomfile2
    [ $status -eq 0 ]
    assert ${rootowner}:-rw-r--r--
    ! test -f $mountpoint/randomfile1

    # third one should have both files, and the shell should be there
    run_buildah from --quiet --pull=false $WITH_POLICY_JSON with-random-both
    cid=$output
    run_buildah mount $cid
    mountpoint=$output
    test -s $mountpoint/bin/sh || test -L $mountpoint/bin/sh
    cmp ${BATS_TMPDIR}/randomfile1 $mountpoint/randomfile1
    run stat -c %u:%g $mountpoint
    [ $status -eq 0 ]
    rootowner=$output
    run stat -c %u:%g:%A $mountpoint/randomfile1
    [ $status -eq 0 ]
    assert ${rootowner}:-rw-r--r--
    cmp ${BATS_TMPDIR}/randomfile2 $mountpoint/in-a-subdir/randomfile2
    run stat -c %u:%g:%A $mountpoint/in-a-subdir/randomfile2
    [ $status -eq 0 ]
    assert ${rootowner}:-rw-r--r--
  done
}

@test "commit with insufficient disk space" {
  skip_if_rootless_environment
  _prefetch busybox
  local tmp=$TEST_SCRATCH_DIR/buildah-test
  mkdir -p $tmp
  mount -t tmpfs -o size=4M tmpfs $tmp
  # Create a temporary file which should not be easy to compress,
  # which we'll add to our container for committing, but which is
  # larger than the filesystem where the layer blob that would
  # contain it, compressed or not, would be written during commit.
  run dd if=/dev/urandom of=$TEST_SCRATCH_DIR/8M bs=1M count=8
  # Create a working container.
  run_buildah from --pull=never $WITH_POLICY_JSON busybox
  ctrID="$output"
  # Copy the file into the working container.
  run_buildah copy $ctrID $TEST_SCRATCH_DIR/8M /8M
  # Try to commit the image.  The temporary copy of the layer diff should
  # require more space than is available where we're telling it to store
  # temporary things.
  TMPDIR=$tmp run_buildah '?' commit $ctrID
  umount $tmp
  expect_output --substring "no space left on device"
}
