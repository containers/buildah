#!/usr/bin/env bats

load helpers

@test "umount one image" {
  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  buildah mount "$cid"
  run buildah umount "$cid"
  [ "${status}" -eq 0 ]
  buildah rm --all
}

@test "umount bad image" {
  run buildah umount badcontainer 
  [ "${status}" -ne 0 ]
  buildah rm --all
}

@test "umount multi images" {
  cid1=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  buildah mount "$cid1"
  cid2=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  buildah mount "$cid2"
  cid3=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  buildah mount "$cid3"
  run buildah umount "$cid1" "$cid2" "$cid3"
  [ "${status}" -eq 0 ]
  buildah rm --all
}

@test "umount all images" {
  cid1=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  buildah mount "$cid1"
  cid2=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  buildah mount "$cid2"
  cid3=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  buildah mount "$cid3"
  run buildah umount --all
  [ "${status}" -eq 0 ]
  buildah rm --all
}

@test "umount multi images one bad" {
  cid1=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  buildah mount "$cid1"
  cid2=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  buildah mount "$cid2"
  cid3=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  buildah mount "$cid3"
  run buildah umount "$cid1" badcontainer "$cid2" "$cid3"
  [ "${status}" -ne 0 ]
  buildah rm --all
}
