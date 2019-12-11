#!/usr/bin/env bats

load helpers

@test "containers" {
  cid1=$(buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json alpine)
  cid2=$(buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json busybox)
  run_buildah containers
  expect_line_count 3

  buildah rm -a
  buildah rmi -a -f
}

@test "containers filter test" {
  cid1=$(buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json alpine)
  cid2=$(buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json busybox)
  run_buildah containers --filter name=$cid1
  expect_line_count 2

  buildah rm -a
  buildah rmi -a -f
}

@test "containers format test" {
  cid1=$(buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json alpine)
  cid2=$(buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json busybox)
  run_buildah containers --format "{{.ContainerName}}"
  expect_line_count 2
  expect_output --from="${lines[0]}" "alpine-working-container"
  expect_output --from="${lines[1]}" "busybox-working-container"

  buildah rm -a
  buildah rmi -a -f
}

@test "containers json test" {
  cid1=$(buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json alpine)
  out=$(buildah containers --json | grep "{" | wc -l)
  [ "$out" -ne "0" ]
  buildah rm -a
  buildah rmi -a -f
}

@test "containers noheading test" {
  cid1=$(buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json alpine)
  cid2=$(buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json busybox)
  run_buildah containers --noheading
  expect_line_count 2
  if [[ $output =~ "NAME" ]]; then
      expect_output "[no instance of 'NAME']" "'NAME' header should be absent"
  fi

  buildah rm -a
  buildah rmi -a -f
}

@test "containers quiet test" {
  cid1=$(buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json alpine)
  cid2=$(buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json busybox)
  run_buildah containers --quiet
  expect_line_count 2

  # Both lines should be CIDs and nothing else.
  expect_output --substring --from="${lines[0]}" '^[0-9a-f]{64}$'
  expect_output --substring --from="${lines[1]}" '^[0-9a-f]{64}$'

  buildah rm -a
  buildah rmi -a -f
}
