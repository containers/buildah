#!/usr/bin/env bats

load helpers

@test "remove multiple containers errors" {
  run buildah --debug=false rm mycontainer1 mycontainer2 mycontainer3
  [ "${lines[0]}" == "error removing container \"mycontainer1\": error reading build container: container not known" ]
  [ "${lines[1]}" == "error removing container \"mycontainer2\": error reading build container: container not known" ]
  [ "${lines[2]}" == "error removing container \"mycontainer3\": error reading build container: container not known" ]
  [ $(wc -l <<< "$output") -eq 3 ]
  [ "${status}" -eq 1 ]
}

@test "remove one container" {
  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  run buildah --debug=false rm "$cid"
  echo "$output"
  [ "$status" -eq 0 ]
  buildah rmi alpine
  echo "$output"
  [ "$status" -eq 0 ]
}

@test "remove multiple containers" {
  cid2=$(buildah from --signature-policy ${TESTSDIR}/policy.json alpine)
  cid3=$(buildah from --signature-policy ${TESTSDIR}/policy.json busybox)
  run buildah --debug=false rm "$cid2" "$cid3"
  echo "$output"
  [ "$status" -eq 0 ]
  buildah rmi alpine busybox
  echo "$output"
  [ "$status" -eq 0 ]
}

@test "remove all containers" {
  cid1=$(buildah from --signature-policy ${TESTSDIR}/policy.json scratch)
  cid2=$(buildah from --signature-policy ${TESTSDIR}/policy.json alpine)
  cid3=$(buildah from --signature-policy ${TESTSDIR}/policy.json busybox)
  run buildah --debug=false rm -a
  echo "$output"
  [ "$status" -eq 0 ]
  run buildah rmi --all
  echo "$output"
  [ "$status" -eq 0 ]
}
