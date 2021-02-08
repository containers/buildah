#!/usr/bin/env bats

load helpers

@test "from" {
  _prefetch alpine
  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
  cid=$output
  run_buildah rm $cid
  run_buildah from --signature-policy ${TESTSDIR}/policy.json scratch
  cid=$output
  run_buildah rm $cid
  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json --name i-love-naming-things alpine
  cid=$output
  run_buildah rm i-love-naming-things
}

@test "from-defaultpull" {
  _prefetch alpine
  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json alpine
  cid=$output
  run_buildah rm $cid
}

@test "from-scratch" {
  run_buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json scratch
  cid=$output
  run_buildah rm $cid
  run_buildah from --pull=true  --signature-policy ${TESTSDIR}/policy.json scratch
  cid=$output
  run_buildah rm $cid
}

@test "from-nopull" {
  run_buildah 125 from --pull-never --signature-policy ${TESTSDIR}/policy.json alpine
}

@test "mount" {
  run_buildah from --signature-policy ${TESTSDIR}/policy.json scratch
  cid=$output
  run_buildah mount $cid
  root=$output
  run_buildah unmount $cid
  run_buildah mount $cid
  root=$output
  touch $root/foobar
  run_buildah unmount $cid
  run_buildah rm $cid
}

@test "by-name" {
  run_buildah from --signature-policy ${TESTSDIR}/policy.json --name scratch-working-image-for-test scratch
  cid=$output
  run_buildah mount scratch-working-image-for-test
  root=$output
  run_buildah unmount scratch-working-image-for-test
  run_buildah rm scratch-working-image-for-test
}

@test "commit" {
  createrandom ${TESTDIR}/randomfile
  createrandom ${TESTDIR}/other-randomfile

  run_buildah from --signature-policy ${TESTSDIR}/policy.json scratch
  cid=$output
  run_buildah mount $cid
  root=$output
  cp ${TESTDIR}/randomfile $root/randomfile
  run_buildah unmount $cid
  run_buildah commit --iidfile output.iid --signature-policy ${TESTSDIR}/policy.json $cid containers-storage:new-image
  iid=$(cat output.iid)
  [[ "$iid" == "sha256:"* ]]
  run_buildah rmi $iid
  run_buildah commit --signature-policy ${TESTSDIR}/policy.json $cid containers-storage:new-image
  run_buildah rm $cid
  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json new-image
  newcid=$output
  run_buildah mount $newcid
  newroot=$output
  test -s $newroot/randomfile
  cmp ${TESTDIR}/randomfile $newroot/randomfile
  cp ${TESTDIR}/other-randomfile $newroot/other-randomfile
  run_buildah commit --signature-policy ${TESTSDIR}/policy.json $newcid containers-storage:other-new-image
  # Not an allowed ordering of arguments and flags.  Check that it's rejected.
  run_buildah 125 commit $newcid --signature-policy ${TESTSDIR}/policy.json containers-storage:rejected-new-image
  run_buildah commit --signature-policy ${TESTSDIR}/policy.json $newcid containers-storage:another-new-image
  run_buildah commit --signature-policy ${TESTSDIR}/policy.json $newcid yet-another-new-image
  run_buildah commit --signature-policy ${TESTSDIR}/policy.json $newcid containers-storage:gratuitous-new-image
  run_buildah unmount $newcid
  run_buildah rm $newcid

  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json other-new-image
  othernewcid=$output
  run_buildah mount $othernewcid
  othernewroot=$output
  test -s $othernewroot/randomfile
  cmp ${TESTDIR}/randomfile $othernewroot/randomfile
  test -s $othernewroot/other-randomfile
  cmp ${TESTDIR}/other-randomfile $othernewroot/other-randomfile
  run_buildah rm $othernewcid

  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json another-new-image
  anothernewcid=$output
  run_buildah mount $anothernewcid
  anothernewroot=$output
  test -s $anothernewroot/randomfile
  cmp ${TESTDIR}/randomfile $anothernewroot/randomfile
  test -s $anothernewroot/other-randomfile
  cmp ${TESTDIR}/other-randomfile $anothernewroot/other-randomfile
  run_buildah rm $anothernewcid

  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json yet-another-new-image
  yetanothernewcid=$output
  run_buildah mount $yetanothernewcid
  yetanothernewroot=$output
  test -s $yetanothernewroot/randomfile
  cmp ${TESTDIR}/randomfile $yetanothernewroot/randomfile
  test -s $yetanothernewroot/other-randomfile
  cmp ${TESTDIR}/other-randomfile $yetanothernewroot/other-randomfile
  run_buildah delete $yetanothernewcid

  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json new-image
  newcid=$output
  run_buildah commit --rm --signature-policy ${TESTSDIR}/policy.json $newcid containers-storage:remove-container-image
  run_buildah 125 mount $newcid

  run_buildah rmi remove-container-image
  run_buildah rmi containers-storage:other-new-image
  run_buildah rmi another-new-image
  run_buildah images -q
  [ "$output" != "" ]
  run_buildah rmi -a
  run_buildah images -q
  expect_output ""
}
