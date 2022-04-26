#!/usr/bin/env bats

load helpers

@test "umount-flags-order-verification" {
  run_buildah 125 umount cnt1 -a
  check_options_flag_err "-a"

  run_buildah 125 umount cnt1 --all cnt2
  check_options_flag_err "--all"

  run_buildah 125 umount cnt1 cnt2 --all
  check_options_flag_err "--all"
}

@test "umount one image" {
  _prefetch alpine
  run_buildah from --quiet --pull=false $WITH_POLICY_JSON alpine
  cid=$output
  run_buildah mount "$cid"
  run_buildah umount "$cid"
}

@test "umount bad image" {
  run_buildah 125 umount badcontainer
}

@test "umount multi images" {
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
  run_buildah umount "$cid1" "$cid2" "$cid3"
}

@test "umount all images" {
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
  run_buildah umount --all
}

@test "umount multi images one bad" {
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
  run_buildah 125 umount "$cid1" badcontainer "$cid2" "$cid3"
}
