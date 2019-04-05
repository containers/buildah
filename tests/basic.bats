#!/usr/bin/env bats

load helpers

@test "from" {
  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  run_buildah rm $cid
  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json scratch)
  run_buildah rm $cid
  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json --name i-love-naming-things alpine)
  run_buildah rm i-love-naming-things
}

@test "from-defaultpull" {
  cid=$(buildah from --signature-policy ${TESTSDIR}/policy.json alpine)
  run_buildah rm $cid
}

@test "from-scratch" {
  cid=$(buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json scratch)
  run_buildah rm $cid
  cid=$(buildah from --pull=true  --signature-policy ${TESTSDIR}/policy.json scratch)
  run_buildah rm $cid
}

@test "from-nopull" {
  run_buildah 1 from --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
}

@test "mount" {
  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json scratch)
  root=$(buildah mount $cid)
  run_buildah unmount $cid
  root=$(buildah mount $cid)
  touch $root/foobar
  run_buildah unmount $cid
  run_buildah rm $cid
}

@test "by-name" {
  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json --name scratch-working-image-for-test scratch)
  root=$(buildah mount scratch-working-image-for-test)
  run_buildah unmount scratch-working-image-for-test
  run_buildah rm scratch-working-image-for-test
}

@test "commit" {
  createrandom ${TESTDIR}/randomfile
  createrandom ${TESTDIR}/other-randomfile

  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json scratch)
  root=$(buildah mount $cid)
  cp ${TESTDIR}/randomfile $root/randomfile
  run_buildah unmount $cid
  run_buildah commit --iidfile output.iid --signature-policy ${TESTSDIR}/policy.json $cid containers-storage:new-image
  iid=$(cat output.iid)
  run_buildah rmi $iid
  run_buildah commit --signature-policy ${TESTSDIR}/policy.json $cid containers-storage:new-image
  run_buildah rm $cid
  newcid=$(buildah from --signature-policy ${TESTSDIR}/policy.json new-image)
  newroot=$(buildah mount $newcid)
  test -s $newroot/randomfile
  cmp ${TESTDIR}/randomfile $newroot/randomfile
  cp ${TESTDIR}/other-randomfile $newroot/other-randomfile
  run_buildah commit --signature-policy ${TESTSDIR}/policy.json $newcid containers-storage:other-new-image
  # Not an allowed ordering of arguments and flags.  Check that it's rejected.
  run_buildah 1 commit $newcid --signature-policy ${TESTSDIR}/policy.json containers-storage:rejected-new-image
  run_buildah commit --signature-policy ${TESTSDIR}/policy.json $newcid containers-storage:another-new-image
  run_buildah commit --signature-policy ${TESTSDIR}/policy.json $newcid yet-another-new-image
  run_buildah commit --signature-policy ${TESTSDIR}/policy.json $newcid containers-storage:gratuitous-new-image
  run_buildah unmount $newcid
  run_buildah rm $newcid

  othernewcid=$(buildah from --signature-policy ${TESTSDIR}/policy.json other-new-image)
  othernewroot=$(buildah mount $othernewcid)
  test -s $othernewroot/randomfile
  cmp ${TESTDIR}/randomfile $othernewroot/randomfile
  test -s $othernewroot/other-randomfile
  cmp ${TESTDIR}/other-randomfile $othernewroot/other-randomfile
  run_buildah rm $othernewcid

  anothernewcid=$(buildah from --signature-policy ${TESTSDIR}/policy.json another-new-image)
  anothernewroot=$(buildah mount $anothernewcid)
  test -s $anothernewroot/randomfile
  cmp ${TESTDIR}/randomfile $anothernewroot/randomfile
  test -s $anothernewroot/other-randomfile
  cmp ${TESTDIR}/other-randomfile $anothernewroot/other-randomfile
  run_buildah rm $anothernewcid

  yetanothernewcid=$(buildah from --signature-policy ${TESTSDIR}/policy.json yet-another-new-image)
  yetanothernewroot=$(buildah mount $yetanothernewcid)
  test -s $yetanothernewroot/randomfile
  cmp ${TESTDIR}/randomfile $yetanothernewroot/randomfile
  test -s $yetanothernewroot/other-randomfile
  cmp ${TESTDIR}/other-randomfile $yetanothernewroot/other-randomfile
  run_buildah delete $yetanothernewcid

  newcid=$(buildah from --signature-policy ${TESTSDIR}/policy.json new-image)
  buildah commit --rm --signature-policy ${TESTSDIR}/policy.json $newcid containers-storage:remove-container-image
  run_buildah 1 mount $newcid

  run_buildah rmi remove-container-image
  run_buildah rmi containers-storage:other-new-image
  run_buildah rmi another-new-image
  run_buildah --debug=false images -q
  [ "$output" != "" ]
  run_buildah rmi -a
  run_buildah --debug=false images -q
  expect_output ""
}
