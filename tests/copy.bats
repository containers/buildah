#!/usr/bin/env bats

load helpers

@test "copy-local-multiple" {
  createrandom ${TESTDIR}/randomfile
  createrandom ${TESTDIR}/other-randomfile
  createrandom ${TESTDIR}/third-randomfile

  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  root=$(buildah mount $cid)
  buildah config --workingdir / $cid
  buildah copy $cid ${TESTDIR}/randomfile
  run buildah copy $cid ${TESTDIR}/other-randomfile ${TESTDIR}/third-randomfile ${TESTDIR}/randomfile
  [ "$status" -eq 1 ]
  buildah rm $cid

  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  root=$(buildah mount $cid)
  buildah config --workingdir / $cid
  buildah copy $cid ${TESTDIR}/randomfile
  buildah copy $cid ${TESTDIR}/other-randomfile ${TESTDIR}/third-randomfile ${TESTDIR}/randomfile /etc
  buildah rm $cid
}

@test "copy-local-plain" {
  createrandom ${TESTDIR}/randomfile
  createrandom ${TESTDIR}/other-randomfile
  createrandom ${TESTDIR}/third-randomfile

  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  root=$(buildah mount $cid)
  buildah config --workingdir / $cid
  buildah copy $cid ${TESTDIR}/randomfile
  buildah copy $cid ${TESTDIR}/other-randomfile
  buildah unmount $cid
  buildah commit --signature-policy ${TESTSDIR}/policy.json $cid containers-storage:new-image
  buildah rm $cid

  newcid=$(buildah from new-image)
  newroot=$(buildah mount $newcid)
  test -s $newroot/randomfile
  cmp ${TESTDIR}/randomfile $newroot/randomfile
  test -s $newroot/other-randomfile
  cmp ${TESTDIR}/other-randomfile $newroot/other-randomfile
  buildah rm $newcid
}
