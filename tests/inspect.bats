#!/usr/bin/env bats

load helpers

@test "inspect-json" {
  cid=$(buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json scratch)
  run buildah --debug=false inspect "$cid"
  [ "$status" -eq 0 ]
  [ "$output" != "" ]
}

@test "inspect-format" {
  cid=$(buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json scratch)
  run buildah --debug=false inspect --format '{{.}}' "$cid"
  [ "$status" -eq 0 ]
  [ "$output" != "" ]
}

@test "inspect-image" {
  cid=$(buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json scratch)
  buildah commit --signature-policy ${TESTSDIR}/policy.json $cid scratchy-image
  run buildah --debug=false inspect --type image scratchy-image
  [ "$status" -eq 0 ]
  [ "$output" != "" ]
  run buildah --debug=false inspect --type image scratchy-image:latest
  [ "$status" -eq 0 ]
  [ "$output" != "" ]
}

@test "HTML escaped" {
  cid=$(buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json scratch)
  buildah config --label maintainer="Darth Vader <dvader@darkside.io>" ${cid}
  buildah commit --signature-policy ${TESTSDIR}/policy.json $cid darkside-image
  buildah rm ${cid}
  output=$(buildah inspect --type image darkside-image)
  [ $(output | grep "u003" | wc -l) -eq 0 ]
  output=$(buildah inspect --type image darkside-image | grep "u003" | wc -l)
  [ "$output" -ne 0 ]
  buildah rmi darkside-image
}
