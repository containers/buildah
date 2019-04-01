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
  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  buildah mount "$cid"
  run_buildah umount "$cid"
  buildah rm --all
}

@test "umount bad image" {
  run_buildah 1 umount badcontainer
  buildah rm --all
}

@test "umount multi images" {
  cid1=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  buildah mount "$cid1"
  cid2=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  buildah mount "$cid2"
  cid3=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  buildah mount "$cid3"
  run_buildah umount "$cid1" "$cid2" "$cid3"
  buildah rm --all
}

@test "umount all images" {
  cid1=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  buildah mount "$cid1"
  cid2=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  buildah mount "$cid2"
  cid3=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  buildah mount "$cid3"
  run_buildah umount --all
  buildah rm --all
}

@test "umount multi images one bad" {
  cid1=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  buildah mount "$cid1"
  cid2=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  buildah mount "$cid2"
  cid3=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  buildah mount "$cid3"
  run_buildah 1 umount "$cid1" badcontainer "$cid2" "$cid3"
  buildah rm --all
}
