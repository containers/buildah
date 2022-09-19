#!/usr/bin/env bats

load helpers

@test "rename" {
  _prefetch alpine
  new_name=test-container
  run_buildah from --quiet --pull=false $WITH_POLICY_JSON alpine
  cid=$output
  run_buildah containers --format "{{.ContainerName}}"
  old_name=$output
  run_buildah rename ${cid} ${new_name}

  run_buildah containers --format "{{.ContainerName}}"
  expect_output --substring "test-container"

  run_buildah containers --quiet -f name=${old_name}
  expect_output ""
}

@test "rename same name as current name" {
  _prefetch alpine
  run_buildah from --quiet --pull=false $WITH_POLICY_JSON alpine
  cid=$output
  run_buildah 125 rename ${cid} ${cid}
  expect_output 'Error: renaming a container with the same name as its current name'
}

@test "rename same name as other container name" {
  _prefetch alpine busybox
  run_buildah from --quiet --pull=false $WITH_POLICY_JSON alpine
  cid1=$output
  run_buildah from --quiet --pull=false $WITH_POLICY_JSON busybox
  cid2=$output
  run_buildah 125 rename ${cid1} ${cid2}
  expect_output --substring " already in use by "
}
