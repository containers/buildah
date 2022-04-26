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
  expect_output --substring "error removing container \"alpine-working-container\": container not known"
}

@test "commit-alternate-storage" {
  _prefetch alpine
  echo FROM
  run_buildah from --quiet --pull=false $WITH_POLICY_JSON alpine
  cid=$output
  echo COMMIT
  run_buildah commit $WITH_POLICY_JSON $cid "containers-storage:[vfs@${TESTDIR}/root2+${TESTDIR}/runroot2]newimage"
  echo FROM
  run_buildah --storage-driver vfs --root ${TESTDIR}/root2 --runroot ${TESTDIR}/runroot2 from $WITH_POLICY_JSON newimage
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
  [ "${status}" -eq 0 ]
  expect_output "untracked actions"

  run_buildah config --created-by "" $cid
  run_buildah commit $WITH_POLICY_JSON $cid ${target}
  run_buildah inspect --format '{{.Config}}' ${target}
  config="$output"
  run python3 -c 'import json, sys; config = json.load(sys.stdin); print(config["history"][len(config["history"])-1]["created_by"])' <<< "$config"
  echo "$output"
  [ "${status}" -eq 0 ]
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

@test "commit-parent-id" {
  _prefetch alpine
  run_buildah from --quiet --pull $WITH_POLICY_JSON alpine
  cid=$output
  run_buildah inspect --format '{{.FromImageID}}' $cid
  iid=$output

  run_buildah commit $WITH_POLICY_JSON --format docker $cid alpine-image
  run_buildah inspect --format '{{.Docker.Parent}}' alpine-image
  expect_output "sha256:$iid" "alpine-image -> .Docker.Parent"
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
  mkdir ${TESTDIR}/tmp
  openssl genrsa -out ${TESTDIR}/tmp/mykey.pem 1024
  openssl rsa -in ${TESTDIR}/tmp/mykey.pem -pubout > ${TESTDIR}/tmp/mykey.pub
  run_buildah from --quiet --pull=false $WITH_POLICY_JSON busybox
  cid=$output
  run_buildah commit --iidfile /dev/null $WITH_POLICY_JSON --encryption-key jwe:${TESTDIR}/tmp/mykey.pub -q $cid oci:${TESTDIR}/tmp/busybox_enc
  imgtype  -show-manifest oci:${TESTDIR}/tmp/busybox_enc | grep "+encrypted"
  rm -rf ${TESTDIR}/tmp
}

@test "commit oci encrypt to registry" {
  _prefetch busybox
  mkdir ${TESTDIR}/tmp
  openssl genrsa -out ${TESTDIR}/tmp/mykey.pem 1024
  openssl rsa -in ${TESTDIR}/tmp/mykey.pem -pubout > ${TESTDIR}/tmp/mykey.pub
  start_registry
  run_buildah from --quiet --pull=false $WITH_POLICY_JSON busybox
  cid=$output
  run_buildah commit --iidfile /dev/null --tls-verify=false --creds testuser:testpassword $WITH_POLICY_JSON --encryption-key jwe:${TESTDIR}/tmp/mykey.pub -q $cid docker://localhost:${REGISTRY_PORT}/buildah/busybox_encrypted:latest
  # this test, just checks the ability to commit an image to a registry
  # there is no good way to test the details of the image unless with ./buildah pull, test will be in pull.bats
  rm -rf ${TESTDIR}/tmp
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

  rm -rf ${TESTDIR}/tmp
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

  rm -rf ${TESTDIR}/tmp
}

@test "commit with authfile" {
  _prefetch busybox
  run_buildah from --quiet --pull=false $WITH_POLICY_JSON busybox
  cid=$output
  run_buildah run $cid touch /test

  start_registry
  run_buildah login --authfile ${TESTDIR}/test.auth --username testuser --password testpassword --tls-verify=false localhost:${REGISTRY_PORT}
  run_buildah commit --authfile ${TESTDIR}/test.auth $WITH_POLICY_JSON --tls-verify=false $cid docker://localhost:${REGISTRY_PORT}/buildah/my-busybox
  expect_output --substring "Writing manifest to image destination"
}
