#!/usr/bin/env bats

load helpers

@test "copy-flags-order-verification" {
  run_buildah 1 copy container1 -q /tmp/container1
  check_options_flag_err "-q"

  run_buildah 1 copy container1 --chown /tmp/container1 --quiet
  check_options_flag_err "--chown"

  run_buildah 1 copy container1 /tmp/container1 --quiet
  check_options_flag_err "--quiet"
}

@test "copy-local-multiple" {
  createrandom ${TESTDIR}/randomfile
  createrandom ${TESTDIR}/other-randomfile
  createrandom ${TESTDIR}/third-randomfile

  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json scratch)
  root=$(buildah mount $cid)
  buildah config --workingdir / $cid
  buildah copy $cid ${TESTDIR}/randomfile
  run_buildah 1 copy $cid ${TESTDIR}/other-randomfile ${TESTDIR}/third-randomfile ${TESTDIR}/randomfile
  buildah rm $cid

  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  root=$(buildah mount $cid)
  buildah config --workingdir / $cid
  buildah copy $cid ${TESTDIR}/randomfile
  buildah copy $cid ${TESTDIR}/other-randomfile ${TESTDIR}/third-randomfile ${TESTDIR}/randomfile /etc
  buildah rm $cid

  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  root=$(buildah mount $cid)
  buildah config --workingdir / $cid
  buildah copy $cid "${TESTDIR}/*randomfile" /etc
  (cd ${TESTDIR}; for i in *randomfile; do cmp $i ${root}/etc/$i; done)
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

  newcid=$(buildah from --signature-policy ${TESTSDIR}/policy.json new-image)
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

@test "copy --chown" {
  mkdir -p ${TESTDIR}/subdir
  mkdir -p ${TESTDIR}/other-subdir
  createrandom ${TESTDIR}/subdir/randomfile
  createrandom ${TESTDIR}/subdir/other-randomfile
  createrandom ${TESTDIR}/randomfile
  createrandom ${TESTDIR}/other-subdir/randomfile
  createrandom ${TESTDIR}/other-subdir/other-randomfile

  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  buildah config --workingdir / $cid
  buildah copy --chown 1:1 $cid ${TESTDIR}/randomfile
  buildah copy --chown root:1 $cid ${TESTDIR}/randomfile /randomfile2
  buildah copy --chown nobody $cid ${TESTDIR}/randomfile /randomfile3
  buildah copy --chown nobody:root $cid ${TESTDIR}/subdir /subdir
  buildah run $cid stat -c "%u:%g" /randomfile
  test $(buildah run $cid stat -c "%u:%g" /randomfile) = "1:1"
  buildah run $cid stat -c "%U:%g" /randomfile2
  test $(buildah run $cid stat -c "%U:%g" /randomfile2) = "root:1"
  buildah run $cid stat -c "%U" /randomfile3
  test $(buildah run $cid stat -c "%U" /randomfile3) = "nobody"
  (for i in randomfile other-randomfile ; do test $(buildah run $cid stat -c "%U:%G" /subdir/$i) = "nobody:root"; done)
  buildah copy --chown root:root $cid ${TESTDIR}/other-subdir /subdir
  (for i in randomfile other-randomfile ; do test $(buildah run $cid stat -c "%U:%G" /subdir/$i) = "root:root"; done)
  buildah run $cid stat -c "%U:%G" /subdir
  test $(buildah run $cid stat -c "%U:%G" /subdir) = "nobody:root"
}

@test "copy-symlink" {
  createrandom ${TESTDIR}/randomfile
  ln -s ${TESTDIR}/randomfile ${TESTDIR}/link-randomfile

  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json scratch)
  root=$(buildah mount $cid)
  buildah config --workingdir / $cid
  buildah copy $cid ${TESTDIR}/link-randomfile
  buildah unmount $cid
  buildah commit --signature-policy ${TESTSDIR}/policy.json $cid containers-storage:new-image
  buildah rm $cid

  newcid=$(buildah from --signature-policy ${TESTSDIR}/policy.json new-image)
  newroot=$(buildah mount $newcid)
  test -s $newroot/link-randomfile
  test -f $newroot/link-randomfile
  cmp ${TESTDIR}/randomfile $newroot/link-randomfile
  buildah rm $newcid
}
