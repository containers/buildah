#!/usr/bin/env bats

load helpers

@test "containers" {
  cid1=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  cid2=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json busybox)
  run_buildah --debug=false containers
  expect_line_count 3

  buildah rm -a
  buildah rmi -a -f
}

@test "containers filter test" {
  cid1=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  cid2=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json busybox)
  run_buildah --debug=false containers --filter name=$cid1
  expect_line_count 2

  buildah rm -a
  buildah rmi -a -f
}

@test "containers format test" {
  cid1=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  cid2=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json busybox)
  run_buildah --debug=false containers --format "{{.ContainerName}}"
  expect_line_count 2

  buildah rm -a
  buildah rmi -a -f
}

@test "containers json test" {
  cid1=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  out=$(buildah --debug=false containers --json | grep "{" | wc -l)
  [ "$out" -ne "0" ]
  buildah rm -a
  buildah rmi -a -f
}

@test "containers noheading test" {
  cid1=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  cid2=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json busybox)
  run_buildah --debug=false containers --noheading
  expect_line_count 2

  buildah rm -a
  buildah rmi -a -f
}

@test "containers quiet test" {
  cid1=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  cid2=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json busybox)
  run_buildah --debug=false containers --quiet
  expect_line_count 2

  buildah rm -a
  buildah rmi -a -f
}
