#!/usr/bin/env bats

load helpers

@test "umount-flags-order-verification" {
  run_buildah 1 umount cnt1 -a
  check_options_flag_err "-a"

  run_buildah 1 umount cnt1 --all cnt2
  check_options_flag_err "--all"

  run_buildah 1 umount cnt1 cnt2 --all
  check_options_flag_err "--all"
}

@test "umount one image" {
  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
  cid=$output
  run_buildah mount "$cid"
  run_buildah umount "$cid"
  run_buildah rm --all
}

@test "umount bad image" {
  run_buildah 1 umount badcontainer
  run_buildah rm --all
}

@test "umount multi images" {
  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
  cid1=$output
  run_buildah mount "$cid1"
  run_buildah from --quiet --pull-never --signature-policy ${TESTSDIR}/policy.json alpine
  cid2=$output
  run_buildah mount "$cid2"
  run_buildah from --quiet --pull-never --signature-policy ${TESTSDIR}/policy.json alpine
  cid3=$output
  run_buildah mount "$cid3"
  run_buildah umount "$cid1" "$cid2" "$cid3"
  run_buildah rm --all
}

@test "umount all images" {
  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
  cid1=$output
  run_buildah mount "$cid1"
  run_buildah from --quiet --pull-never --signature-policy ${TESTSDIR}/policy.json alpine
  cid2=$output
  run_buildah mount "$cid2"
  run_buildah from --quiet --pull-never --signature-policy ${TESTSDIR}/policy.json alpine
  cid3=$output
  run_buildah mount "$cid3"
  run_buildah umount --all
  run_buildah rm --all
}

@test "umount multi images one bad" {
  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
  cid1=$output
  run_buildah mount "$cid1"
  run_buildah from --quiet --pull-never --signature-policy ${TESTSDIR}/policy.json alpine
  cid2=$output
  run_buildah mount "$cid2"
  run_buildah from --quiet --pull-never --signature-policy ${TESTSDIR}/policy.json alpine
  cid3=$output
  run_buildah mount "$cid3"
  run_buildah 1 umount "$cid1" badcontainer "$cid2" "$cid3"
  run_buildah rm --all
}
