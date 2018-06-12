#!/usr/bin/env bats

load helpers

@test "containers" {
  cid1=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  cid2=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json busybox)
  run buildah --debug=false containers
  [ $(wc -l <<< "$output") -eq 3 ]
  [ "${status}" -eq 0 ]
  buildah rm -a
  buildah rmi -a -f
}

@test "containers filter test" {
  cid1=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  cid2=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json busybox)
  run buildah --debug=false containers --filter name=$cid1
  [ $(wc -l <<< "$output") -eq 2 ]
  [ "${status}" -eq 0 ]
  buildah rm -a
  buildah rmi -a -f
}

@test "containers format test" {
  cid1=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  cid2=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json busybox)
  run buildah --debug=false containers --format "{{.ContainerName}}"
  [ $(wc -l <<< "$output") -eq 2 ]
  [ "${status}" -eq 0 ]
  buildah rm -a
  buildah rmi -a -f
}

@test "containers noheading test" {
  cid1=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  cid2=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json busybox)
  run buildah --debug=false containers --noheading
  [ $(wc -l <<< "$output") -eq 2 ]
  [ "${status}" -eq 0 ]
  buildah rm -a
  buildah rmi -a -f
}

@test "containers quiet test" {
  cid1=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  cid2=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json busybox)
  run buildah --debug=false containers --quiet
  [ $(wc -l <<< "$output") -eq 2 ]
  [ "${status}" -eq 0 ]
  buildah rm -a
  buildah rmi -a -f
}
