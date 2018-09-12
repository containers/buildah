#!/usr/bin/env bats

load helpers

@test "mount-flags-order-verification" {
  run buildah mount cnt1 --notruncate path1
  check_options_flag_err "--notruncate"

  run buildah mount cnt1 --notruncate
  check_options_flag_err "--notruncate"

  run buildah mount cnt1 path1 --notruncate
  check_options_flag_err "--notruncate"
}

@test "mount one container" {
  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  run buildah --debug=false mount "$cid"
  [ "${status}" -eq 0 ]
  buildah rm $cid
  buildah rmi -f alpine
}

@test "mount bad container" {
  run buildah --debug=false mount badcontainer 
  [ "${status}" -ne 0 ]
}

@test "mount multi images" {
  cid1=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  cid2=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  cid3=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  run buildah mount "$cid1" "$cid2" "$cid3"
  [ "${status}" -eq 0 ] 
  buildah rm --all
  buildah rmi -f alpine
}

@test "mount multi images one bad" {
  cid1=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  cid2=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  cid3=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  run buildah mount "$cid1" badcontainer "$cid2" "$cid3"
  [ "${status}" -ne 0 ]
  buildah rm --all
  buildah rmi -f alpine
}

@test "list currently mounted containers" {
  cid1=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  buildah mount "$cid1"
  cid2=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  buildah mount "$cid2"
  cid3=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  buildah mount "$cid3"
  run buildah --debug=false mount
  [[ "${lines[0]}" =~ "/tmp" ]]
  [[ "${lines[1]}" =~ "/tmp" ]]
  [[ "${lines[2]}" =~ "/tmp" ]]
  [ $(wc -l <<< "$output") -eq 3 ]
  [ "${status}" -eq 0 ]
  buildah rm -all
  buildah rmi -f alpine
}
