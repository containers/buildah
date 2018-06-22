#!/usr/bin/env bats

load helpers

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
  run buildah commit --signature-policy ${TESTSDIR}/policy.json -q $cid alpine-image
  [ "${status}" -eq 0 ]
  [ $(wc -l <<< "$output") -eq 34 ]
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
  buildah --storage-driver vfs --root ${TESTDIR}/root2 --runroot ${TESTDIR}/runroot2 from newimage
}
