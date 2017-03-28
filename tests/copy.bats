#!/usr/bin/env bats

load helpers

@test "copy-local-multiple" {
  createrandom ${TESTDIR}/randomfile
  createrandom ${TESTDIR}/other-randomfile
  createrandom ${TESTDIR}/third-randomfile

  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json scratch)
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

  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json scratch)
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

@test "copy-local-subdirectory" {
  mkdir -p ${TESTDIR}/subdir
  createrandom ${TESTDIR}/subdir/randomfile
  createrandom ${TESTDIR}/subdir/other-randomfile

  cid=$(buildah from --signature-policy ${TESTSDIR}/policy.json scratch)
  buildah config --workingdir /container-subdir $cid
  buildah copy $cid ${TESTDIR}/subdir
  root=$(buildah mount $cid)
  test -s $root/container-subdir/randomfile
  cmp ${TESTDIR}/subdir/randomfile $root/container-subdir/randomfile
  test -s $root/container-subdir/other-randomfile
  cmp ${TESTDIR}/subdir/other-randomfile $root/container-subdir/other-randomfile
  buildah copy $cid ${TESTDIR}/subdir /other-subdir
  test -s $root/other-subdir/randomfile
  cmp ${TESTDIR}/subdir/randomfile $root/other-subdir/randomfile
  test -s $root/other-subdir/other-randomfile
  cmp ${TESTDIR}/subdir/other-randomfile $root/other-subdir/other-randomfile
  buildah rm $cid
}

@test "copy-local-force-directory" {
  createrandom ${TESTDIR}/randomfile

  cid=$(buildah from --signature-policy ${TESTSDIR}/policy.json scratch)
  buildah config --workingdir / $cid
  buildah copy $cid ${TESTDIR}/randomfile /randomfile
  root=$(buildah mount $cid)
  test -s $root/randomfile
  cmp ${TESTDIR}/randomfile $root/randomfile
  buildah rm $cid

  cid=$(buildah from --signature-policy ${TESTSDIR}/policy.json scratch)
  buildah config --workingdir / $cid
  buildah copy $cid ${TESTDIR}/randomfile /randomsubdir/
  root=$(buildah mount $cid)
  test -s $root/randomsubdir/randomfile
  cmp ${TESTDIR}/randomfile $root/randomsubdir/randomfile
  buildah rm $cid
}

@test "copy-url-mtime" {
  createrandom ${TESTDIR}/randomfile

  cid=$(buildah from --signature-policy ${TESTSDIR}/policy.json scratch)
  buildah config --workingdir / $cid
  starthttpd ${TESTDIR}
  buildah copy $cid http://0.0.0.0:${HTTP_SERVER_PORT}/randomfile /urlfile
  stophttpd
  root=$(buildah mount $cid)
  test -s $root/urlfile
  cmp ${TESTDIR}/randomfile $root/urlfile
  run test -nt ${TESTDIR}/randomfile $root/urlfile
  [ "$status" -ne 0 ]
  run test -ot ${TESTDIR}/randomfile $root/urlfile
  [ "$status" -ne 0 ]
  buildah rm $cid
}
