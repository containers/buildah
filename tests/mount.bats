#!/usr/bin/env bats

load helpers

@test "mount-flags-order-verification" {
  run_buildah 125 mount cnt1 --notruncate path1
  check_options_flag_err "--notruncate"

  run_buildah 125 mount cnt1 --notruncate
  check_options_flag_err "--notruncate"

  run_buildah 125 mount cnt1 path1 --notruncate
  check_options_flag_err "--notruncate"
}

@test "mount one container" {
  _prefetch alpine
  run_buildah from --quiet --pull=false $WITH_POLICY_JSON alpine
  cid=$output
  run_buildah mount "$cid"
}

@test "mount bad container" {
  run_buildah 125 mount badcontainer
}

@test "mount multi images" {
  _prefetch alpine
  run_buildah from --quiet --pull=false $WITH_POLICY_JSON alpine
  cid1=$output
  run_buildah from --quiet --pull-never $WITH_POLICY_JSON alpine
  cid2=$output
  run_buildah from --quiet --pull-never $WITH_POLICY_JSON alpine
  cid3=$output
  run_buildah mount "$cid1" "$cid2" "$cid3"
}

@test "mount multi images one bad" {
  _prefetch alpine
  run_buildah from --quiet --pull=false $WITH_POLICY_JSON alpine
  cid1=$output
  run_buildah from --quiet --pull-never $WITH_POLICY_JSON alpine
  cid2=$output
  run_buildah from --quiet --pull-never $WITH_POLICY_JSON alpine
  cid3=$output
  run_buildah 125 mount "$cid1" badcontainer "$cid2" "$cid3"
}

@test "list currently mounted containers" {
  _prefetch alpine
  run_buildah from --quiet --pull=false $WITH_POLICY_JSON alpine
  cid1=$output
  run_buildah mount "$cid1"
  run_buildah from --quiet --pull-never $WITH_POLICY_JSON alpine
  cid2=$output
  run_buildah mount "$cid2"
  run_buildah from --quiet --pull-never $WITH_POLICY_JSON alpine
  cid3=$output
  run_buildah mount "$cid3"
  run_buildah mount
  expect_line_count 3
  expect_output --from="${lines[0]}" --substring "/tmp" "mount line 1 of 3"
  expect_output --from="${lines[1]}" --substring "/tmp" "mount line 2 of 3"
  expect_output --from="${lines[2]}" --substring "/tmp" "mount line 3 of 3"
}
