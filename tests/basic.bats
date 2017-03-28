#!/usr/bin/env bats

load helpers

@test "from" {
  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  buildah delete $cid
  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json         alpine)
  buildah delete $cid
  cid=$(buildah from alpine --pull --signature-policy ${TESTSDIR}/policy.json --name i-love-naming-things)
  buildah delete i-love-naming-things
}

@test "from-defaultpull" {
  cid=$(buildah from --signature-policy ${TESTSDIR}/policy.json alpine)
  buildah delete $cid
}

@test "from-nopull" {
  run buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
  [ "$status" -eq 1 ]
}

@test "mount" {
  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  root=$(buildah mount $cid)
  buildah unmount $cid
  root=$(buildah mount $cid)
  touch $root/foobar
  buildah unmount $cid
  buildah delete $cid
}

@test "by-name" {
  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json --name alpine-working-image-for-test alpine)
  root=$(buildah mount alpine-working-image-for-test)
  buildah unmount alpine-working-image-for-test
  buildah delete alpine-working-image-for-test
}

@test "by-root" {
  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  root=$(buildah mount $cid)
  buildah unmount $cid
  buildah delete $cid
}

@test "commit" {
  createrandom ${TESTDIR}/randomfile
  createrandom ${TESTDIR}/other-randomfile

  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  root=$(buildah mount $cid)
  cp ${TESTDIR}/randomfile $root/randomfile
  buildah unmount $cid
  buildah commit --signature-policy ${TESTSDIR}/policy.json $cid containers-storage:new-image
  buildah delete $cid

  newcid=$(buildah from new-image)
  newroot=$(buildah mount $newcid)
  test -s $newroot/randomfile
  cmp ${TESTDIR}/randomfile $newroot/randomfile
  cp ${TESTDIR}/other-randomfile $newroot/other-randomfile
  buildah commit --signature-policy ${TESTSDIR}/policy.json $newcid containers-storage:other-new-image
  # Not an allowed ordering of arguments and flags.  Check that it's rejected.
  run buildah commit $newcid --signature-policy ${TESTSDIR}/policy.json containers-storage:rejected-new-image
  [ "$status" -eq 1 ]
  buildah commit --signature-policy ${TESTSDIR}/policy.json $newcid containers-storage:another-new-image
  buildah commit --signature-policy ${TESTSDIR}/policy.json $newcid yet-another-new-image
  buildah unmount $newcid
  buildah delete $newcid

  othernewcid=$(buildah from other-new-image)
  othernewroot=$(buildah mount $othernewcid)
  test -s $othernewroot/randomfile
  cmp ${TESTDIR}/randomfile $othernewroot/randomfile
  test -s $othernewroot/other-randomfile
  cmp ${TESTDIR}/other-randomfile $othernewroot/other-randomfile
  buildah delete $othernewcid

  anothernewcid=$(buildah from another-new-image)
  anothernewroot=$(buildah mount $anothernewcid)
  test -s $anothernewroot/randomfile
  cmp ${TESTDIR}/randomfile $anothernewroot/randomfile
  test -s $anothernewroot/other-randomfile
  cmp ${TESTDIR}/other-randomfile $anothernewroot/other-randomfile
  buildah delete $anothernewcid

  yetanothernewcid=$(buildah from yet-another-new-image)
  yetanothernewroot=$(buildah mount $yetanothernewcid)
  test -s $yetanothernewroot/randomfile
  cmp ${TESTDIR}/randomfile $yetanothernewroot/randomfile
  test -s $yetanothernewroot/other-randomfile
  cmp ${TESTDIR}/other-randomfile $yetanothernewroot/other-randomfile
  buildah delete $yetanothernewcid
}
