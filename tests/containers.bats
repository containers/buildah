#!/usr/bin/env bats

load helpers

@test "containers" {
  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json busybox
  run_buildah containers
  expect_line_count 3
}

@test "containers filter test" {
  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
  cid1=$output
  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json busybox
  cid2=$output
  run_buildah containers --filter name=$cid1
  expect_line_count 2
}

@test "containers format test" {
  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json busybox
  run_buildah containers --format "{{.ContainerName}}"
  expect_line_count 2
  expect_output --from="${lines[0]}" "alpine-working-container"
  expect_output --from="${lines[1]}" "busybox-working-container"
}

@test "containers json test" {
  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
  run_buildah containers --json
  expect_output --substring '\{'
}

@test "containers noheading test" {
  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json busybox
  run_buildah containers --noheading
  expect_line_count 2
  if [[ $output =~ "NAME" ]]; then
      expect_output "[no instance of 'NAME']" "'NAME' header should be absent"
  fi
}

@test "containers quiet test" {
  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json busybox
  run_buildah containers --quiet
  expect_line_count 2

  # Both lines should be CIDs and nothing else.
  expect_output --substring --from="${lines[0]}" '^[0-9a-f]{64}$'
  expect_output --substring --from="${lines[1]}" '^[0-9a-f]{64}$'
}
