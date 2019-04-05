#!/usr/bin/env bats

load helpers

@test "rename" {
  new_name=test-container
  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  old_name=$(buildah containers --format "{{.ContainerName}}")
  buildah rename ${cid} ${new_name}

  run_buildah containers --format "{{.ContainerName}}"
  expect_output --substring "test-container"

  run_buildah --debug=false containers --quiet -f name=${old_name}
  expect_output ""

  buildah rm ${new_name}
  [ "$status" -eq 0 ]
}

@test "rename same name as current name" {
  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  run_buildah 1 --debug=false rename ${cid} ${cid}
  expect_output 'renaming a container with the same name as its current name'

  buildah rm $cid
  buildah rmi -f alpine
}

@test "rename same name as other container name" {
  cid1=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  cid2=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json busybox)
  run_buildah 1 --debug=false rename ${cid1} ${cid2}
  expect_output --substring " already in use by "

  buildah rm $cid1 $cid2
  buildah rmi -f alpine busybox
}
