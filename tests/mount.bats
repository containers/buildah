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
  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
  cid=$output
  run_buildah mount "$cid"
  run_buildah rm $cid
  run_buildah rmi -f alpine
}

@test "mount bad container" {
  run_buildah 1 mount badcontainer
}

@test "mount multi images" {
  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
  cid1=$output
  run_buildah from --quiet --pull-never --signature-policy ${TESTSDIR}/policy.json alpine
  cid2=$output
  run_buildah from --quiet --pull-never --signature-policy ${TESTSDIR}/policy.json alpine
  cid3=$output
  run_buildah mount "$cid1" "$cid2" "$cid3"
  run_buildah rm --all
  run_buildah rmi -f alpine
}

@test "mount multi images one bad" {
  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
  cid1=$output
  run_buildah from --quiet --pull-never --signature-policy ${TESTSDIR}/policy.json alpine
  cid2=$output
  run_buildah from --quiet --pull-never --signature-policy ${TESTSDIR}/policy.json alpine
  cid3=$output
  run_buildah 1 mount "$cid1" badcontainer "$cid2" "$cid3"
  run_buildah rm --all
  run_buildah rmi -f alpine
}

@test "list currently mounted containers" {
  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
  cid1=$output
  run_buildah mount "$cid1"
  run_buildah from --quiet --pull-never --signature-policy ${TESTSDIR}/policy.json alpine
  cid2=$output
  run_buildah mount "$cid2"
  run_buildah from --quiet --pull-never --signature-policy ${TESTSDIR}/policy.json alpine
  cid3=$output
  run_buildah mount "$cid3"
  run_buildah mount
  expect_output --from="${lines[0]}" --substring "/tmp" "mount line 1 of 3"
  expect_output --from="${lines[1]}" --substring "/tmp" "mount line 2 of 3"
  expect_output --from="${lines[2]}" --substring "/tmp" "mount line 3 of 3"
  expect_line_count 3

  run_buildah rm --all
  run_buildah rmi -f alpine
}
