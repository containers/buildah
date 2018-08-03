#!/usr/bin/env bats

load helpers

@test "from" {
  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  buildah rm $cid
  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json scratch)
  buildah rm $cid
  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json --name i-love-naming-things alpine)
  buildah rm i-love-naming-things
}

@test "from-defaultpull" {
  cid=$(buildah from --signature-policy ${TESTSDIR}/policy.json alpine)
  buildah rm $cid
}

@test "from-scratch" {
  cid=$(buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json scratch)
  buildah rm $cid
  cid=$(buildah from --pull=true  --signature-policy ${TESTSDIR}/policy.json scratch)
  buildah rm $cid
}

@test "from-nopull" {
  run buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
  [ "$status" -eq 1 ]
}

@test "mount" {
  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json scratch)
  root=$(buildah mount $cid)
  buildah unmount $cid
  root=$(buildah mount $cid)
  touch $root/foobar
  buildah unmount $cid
  buildah rm $cid
}

@test "by-name" {
  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json --name scratch-working-image-for-test scratch)
  root=$(buildah mount scratch-working-image-for-test)
  buildah unmount scratch-working-image-for-test
  buildah rm scratch-working-image-for-test
}

@test "commit" {
  createrandom ${TESTDIR}/randomfile
  createrandom ${TESTDIR}/other-randomfile

  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json scratch)
  root=$(buildah mount $cid)
  cp ${TESTDIR}/randomfile $root/randomfile
  buildah unmount $cid
  buildah commit --iidfile output.iid --signature-policy ${TESTSDIR}/policy.json $cid containers-storage:new-image
  iid=$(cat output.iid)
  buildah rmi $iid
  buildah commit --signature-policy ${TESTSDIR}/policy.json $cid containers-storage:new-image
  buildah rm $cid
  newcid=$(buildah from --signature-policy ${TESTSDIR}/policy.json new-image)
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
  buildah commit --signature-policy ${TESTSDIR}/policy.json $newcid containers-storage:gratuitous-new-image
  buildah unmount $newcid
  buildah rm $newcid

  othernewcid=$(buildah from --signature-policy ${TESTSDIR}/policy.json other-new-image)
  othernewroot=$(buildah mount $othernewcid)
  test -s $othernewroot/randomfile
  cmp ${TESTDIR}/randomfile $othernewroot/randomfile
  test -s $othernewroot/other-randomfile
  cmp ${TESTDIR}/other-randomfile $othernewroot/other-randomfile
  buildah rm $othernewcid

  anothernewcid=$(buildah from --signature-policy ${TESTSDIR}/policy.json another-new-image)
  anothernewroot=$(buildah mount $anothernewcid)
  test -s $anothernewroot/randomfile
  cmp ${TESTDIR}/randomfile $anothernewroot/randomfile
  test -s $anothernewroot/other-randomfile
  cmp ${TESTDIR}/other-randomfile $anothernewroot/other-randomfile
  buildah rm $anothernewcid

  yetanothernewcid=$(buildah from --signature-policy ${TESTSDIR}/policy.json yet-another-new-image)
  yetanothernewroot=$(buildah mount $yetanothernewcid)
  test -s $yetanothernewroot/randomfile
  cmp ${TESTDIR}/randomfile $yetanothernewroot/randomfile
  test -s $yetanothernewroot/other-randomfile
  cmp ${TESTDIR}/other-randomfile $yetanothernewroot/other-randomfile
  buildah delete $yetanothernewcid

  newcid=$(buildah from --signature-policy ${TESTSDIR}/policy.json new-image)
  buildah commit --rm --signature-policy ${TESTSDIR}/policy.json $newcid containers-storage:remove-container-image
  run buildah mount $newcid
  [ "$status" -ne 0 ]

  buildah rmi remove-container-image
  buildah rmi containers-storage:other-new-image
  buildah rmi another-new-image
  run buildah --debug=false images -q
  [ "$status" -eq 0 ]
  [ "$output" != "" ]
  buildah rmi -a
  run buildah --debug=false images -q
  [ "$status" -eq 0 ]
  [ "$output" == "" ]
}
