#!/usr/bin/env bats

load helpers

@test "images" {
  cid1=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  cid2=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json busybox)
  run buildah --debug=false images
  [ $(wc -l <<< "$output") -eq 3 ]
  [ "${status}" -eq 0 ]
  buildah rm -a
  buildah rmi -a -f
}

@test "images filter test" {
  cid1=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  cid2=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json busybox)
  run buildah --debug=false images --filter since=alpine
  [ $(wc -l <<< "$output") -eq 2 ]
  [ "${status}" -eq 0 ]
  buildah rm -a
  buildah rmi -a -f
}

@test "images format test" {
  cid1=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  cid2=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json busybox)
  run buildah --debug=false images --format "{{.Name}}"
  [ $(wc -l <<< "$output") -eq 1 ]
  [ "${status}" -eq 0 ]
  buildah rm -a
  buildah rmi -a -f
}

@test "images noheading test" {
  cid1=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  cid2=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json busybox)
  run buildah --debug=false images --noheading
  [ $(wc -l <<< "$output") -eq 2 ]
  [ "${status}" -eq 0 ]
  buildah rm -a
  buildah rmi -a -f
}

@test "images quiet test" {
  cid1=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  cid2=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json busybox)
  run buildah --debug=false images --quiet
  [ $(wc -l <<< "$output") -eq 2 ]
  [ "${status}" -eq 0 ]
  buildah rm -a
  buildah rmi -a -f
}

@test "specify an existing image" {
  cid1=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  cid2=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json busybox)
  run buildah --debug=false images alpine
  [ $(wc -l <<< "$output") -eq 2 ]
  [ "${status}" -eq 0 ]
  buildah rm -a
  buildah rmi -a -f
}

@test "specify a nonexistent image" {
  run buildah --debug=false images alpine
  [ "${lines[0]}" == "No such image alpine" ]
  [ $(wc -l <<< "$output") -eq 1 ]
  [ "${status}" -eq 1 ]
}
