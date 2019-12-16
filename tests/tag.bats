#!/usr/bin/env bats

load helpers

@test "tag by name" {
  run_buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json scratch
  cid=$output
  run_buildah commit --signature-policy ${TESTSDIR}/policy.json "$cid" scratch-image
  run_buildah 1 inspect --type image tagged-image
  run_buildah tag scratch-image tagged-image tagged-also-image named-image
  run_buildah inspect --type image tagged-image
  run_buildah inspect --type image tagged-also-image
  run_buildah inspect --type image named-image
}

@test "tag by id" {
  run_buildah pull --quiet --signature-policy ${TESTSDIR}/policy.json busybox
  id=$output

  # Tag by ID, then make a container from that tag
  run_buildah tag $id busybox1
  run_buildah from busybox1            # gives us busybox1-working-container

  # The from-name should be busybox1, but ID should be same as pulled image
  run_buildah inspect --format '{{ .FromImage }}' busybox1-working-container
  expect_output "localhost/busybox1:latest"
  run_buildah inspect --format '{{ .FromImageID }}' busybox1-working-container
  expect_output $id

  # Clean up
  run_buildah rm busybox1-working-container
  run_buildah rmi busybox1
}
