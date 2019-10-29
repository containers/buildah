#!/usr/bin/env bats

load helpers

@test "tag by name" {
  cid=$(buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json scratch)
  run_buildah commit --signature-policy ${TESTSDIR}/policy.json "$cid" scratch-image
  run_buildah 1 inspect --type image tagged-image
  run_buildah tag scratch-image tagged-image tagged-also-image named-image
  run_buildah inspect --type image tagged-image
  run_buildah inspect --type image tagged-also-image
  run_buildah inspect --type image named-image
}

@test "tag by id" {
  buildah pull --signature-policy ${TESTSDIR}/policy.json busybox
  id=$(buildah images -q busybox)
  run_buildah tag $id busybox1
  run_buildah from busybox1
  run_buildah mount busybox1-working-container
  run_buildah unmount busybox1-working-container
  run_buildah rm busybox1-working-container
  run_buildah rmi busybox1
  run_buildah inspect --type image busybox
}
