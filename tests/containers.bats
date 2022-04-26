#!/usr/bin/env bats

load helpers

@test "containers" {
  _prefetch alpine busybox
  run_buildah from --quiet --pull=false $WITH_POLICY_JSON alpine
  run_buildah from --quiet --pull=false $WITH_POLICY_JSON busybox
  run_buildah containers
  expect_line_count 3
}

@test "containers filter test" {
  _prefetch alpine busybox
  run_buildah from --quiet --pull=false $WITH_POLICY_JSON alpine
  cid1=$output
  run_buildah from --quiet --pull=false $WITH_POLICY_JSON busybox
  cid2=$output
  run_buildah containers --filter name=$cid1
  expect_line_count 2
}

@test "containers format test" {
  _prefetch alpine busybox
  run_buildah from --quiet --pull=false $WITH_POLICY_JSON alpine
  run_buildah from --quiet --pull=false $WITH_POLICY_JSON busybox
  run_buildah containers --format "{{.ContainerName}}"
  expect_line_count 2
  expect_output --from="${lines[0]}" "alpine-working-container"
  expect_output --from="${lines[1]}" "busybox-working-container"
}

@test "containers json test" {
  _prefetch alpine
  run_buildah from --quiet --pull=false $WITH_POLICY_JSON alpine
  run_buildah containers --json
  expect_output --substring '\{'
}

@test "containers noheading test" {
  _prefetch alpine busybox
  run_buildah from --quiet --pull=false $WITH_POLICY_JSON alpine
  run_buildah from --quiet --pull=false $WITH_POLICY_JSON busybox
  run_buildah containers --noheading
  expect_line_count 2
  if [[ $output =~ "NAME" ]]; then
      expect_output "[no instance of 'NAME']" "'NAME' header should be absent"
  fi
}

@test "containers quiet test" {
  _prefetch alpine busybox
  run_buildah from --quiet --pull=false $WITH_POLICY_JSON alpine
  run_buildah from --quiet --pull=false $WITH_POLICY_JSON busybox
  run_buildah containers --quiet
  expect_line_count 2

  # Both lines should be CIDs and nothing else.
  expect_output --substring --from="${lines[0]}" '^[0-9a-f]{64}$'
  expect_output --substring --from="${lines[1]}" '^[0-9a-f]{64}$'
}

@test "containers notruncate test" {
  _prefetch alpine
  run_buildah from --quiet --pull=false $WITH_POLICY_JSON alpine
  run_buildah containers --notruncate
  expect_line_count 2
  expect_output --substring --from="${lines[1]}" '^[0-9a-f]{64}'
}

@test "containers all test" {
  skip_if_in_container
  skip_if_no_podman

  _prefetch alpine busybox
  run_buildah from --quiet --pull=false $WITH_POLICY_JSON alpine
  podman create --root ${TEST_SCRATCH_DIR}/root --storage-driver ${STORAGE_DRIVER} --net=host busybox ls
  run_buildah containers
  expect_line_count 2
  run_buildah containers -a
  expect_line_count 3
}
