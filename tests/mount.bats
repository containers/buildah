#!/usr/bin/env bats

load helpers

@test "mount-flags-order-verification" {
  run_buildah 1 mount cnt1 --notruncate path1
  check_options_flag_err "--notruncate"

  run_buildah 1 mount cnt1 --notruncate
  check_options_flag_err "--notruncate"

  run_buildah 1 mount cnt1 path1 --notruncate
  check_options_flag_err "--notruncate"
}

@test "mount one container" {
  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  run_buildah --debug=false mount "$cid"
  buildah rm $cid
  buildah rmi -f alpine
}

@test "mount bad container" {
  run_buildah 1 --debug=false mount badcontainer
}

@test "mount multi images" {
  cid1=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  cid2=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  cid3=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  run_buildah mount "$cid1" "$cid2" "$cid3"
  buildah rm --all
  buildah rmi -f alpine
}

@test "mount multi images one bad" {
  cid1=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  cid2=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  cid3=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  run_buildah 1 mount "$cid1" badcontainer "$cid2" "$cid3"
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
  run_buildah --debug=false mount
  expect_output --from="${lines[0]}" --substring "/tmp" "mount line 1 of 3"
  expect_output --from="${lines[1]}" --substring "/tmp" "mount line 2 of 3"
  expect_output --from="${lines[2]}" --substring "/tmp" "mount line 3 of 3"
  expect_line_count 3

  buildah rm --all
  buildah rmi -f alpine
}
