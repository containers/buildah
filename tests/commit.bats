#!/usr/bin/env bats

load helpers

@test "commit-flags-order-verification" {
  run buildah commit cnt1 --tls-verify
  check_options_flag_err "--tls-verify"

  run buildah commit cnt1 -q
  check_options_flag_err "-q"

  run buildah commit cnt1 -f=docker --quiet --creds=bla:bla
  check_options_flag_err "-f=docker"

  run buildah commit cnt1 --creds=bla:bla
  check_options_flag_err "--creds=bla:bla"
}

@test "commit" {
  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  buildah commit --signature-policy ${TESTSDIR}/policy.json $cid alpine-image
  run buildah images alpine-image
  [ "${status}" -eq 0 ]
  buildah rm $cid
  buildah rmi -a
}

@test "commit format test" {
  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  buildah commit --signature-policy ${TESTSDIR}/policy.json $cid alpine-image-oci
  buildah commit --format docker --signature-policy ${TESTSDIR}/policy.json $cid alpine-image-docker

  buildah --debug=false inspect --type=image --format '{{.Manifest}}' alpine-image-oci | grep "application/vnd.oci.image.layer.v1.tar+gzip"
  buildah --debug=false inspect --type=image --format '{{.Manifest}}' alpine-image-docker | grep "application/vnd.docker.image.rootfs.diff.tar.gzip"
  buildah rm $cid
  buildah rmi -a
}

@test "commit quiet test" {
  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  run buildah --debug=false commit --iidfile /dev/null --signature-policy ${TESTSDIR}/policy.json -q $cid alpine-image
  echo "$output"
  [ "${status}" -eq 0 ]
  [ "$output" = "" ]
  buildah rm $cid
  buildah rmi -a
}

@test "commit rm test" {
  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  buildah commit --signature-policy ${TESTSDIR}/policy.json --rm $cid alpine-image
  run buildah --debug=false rm $cid
  [ "${status}" -eq 1 ]
  [ "${lines[0]}" == "error removing container \"alpine-working-container\": error reading build container: container not known" ]
  [ $(wc -l <<< "$output") -eq 1 ]
  buildah rmi -a
}

@test "commit-alternate-storage" {
  echo FROM
  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json openshift/hello-openshift)
  echo COMMIT
  buildah commit --signature-policy ${TESTSDIR}/policy.json $cid "containers-storage:[vfs@${TESTDIR}/root2+${TESTDIR}/runroot2]newimage"
  echo FROM
  buildah --storage-driver vfs --root ${TESTDIR}/root2 --runroot ${TESTDIR}/runroot2 from --signature-policy ${TESTSDIR}/policy.json newimage
}

@test "commit-rejected-name" {
  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  run buildah --debug=false commit --signature-policy ${TESTSDIR}/policy.json $cid ThisNameShouldBeRejected
  echo "$output"
  [ "${status}" -ne 0 ]
  [[ "${output}" =~ "must be lower" ]]
}
