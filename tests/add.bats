#!/usr/bin/env bats

load helpers

@test "add-flags-order-verification" {
  run_buildah 1 add container1 -q /tmp/container1
  check_options_flag_err "-q"

  run_buildah 1 add container1 --chown /tmp/container1 --quiet
  check_options_flag_err "--chown"

  run_buildah 1 add container1 /tmp/container1 --quiet
  check_options_flag_err "--quiet"
}

@test "add-local-plain" {
  createrandom ${TESTDIR}/randomfile
  createrandom ${TESTDIR}/other-randomfile

  cid=$(buildah from --signature-policy ${TESTSDIR}/policy.json scratch)
  root=$(buildah mount $cid)
  mkdir $root/subdir $root/other-subdir
  # Copy a file to the working directory
  run_buildah config --workingdir=/ $cid
  run_buildah add $cid ${TESTDIR}/randomfile
  # Copy a file to a specific subdirectory
  run_buildah add $cid ${TESTDIR}/randomfile /subdir
  # Copy two files to a specific subdirectory
  run_buildah add $cid ${TESTDIR}/randomfile ${TESTDIR}/other-randomfile /other-subdir
  # Copy two files to a specific location, which fails because it's not a directory.
  run_buildah 1 add ${TESTDIR}/randomfile ${TESTDIR}/other-randomfile $cid /notthereyet-subdir
  run_buildah 1 add ${TESTDIR}/randomfile $cid ${TESTDIR}/other-randomfile /randomfile
  # Copy a file to a different working directory
  run_buildah config --workingdir=/cwd $cid
  run_buildah add $cid ${TESTDIR}/randomfile
  run_buildah unmount $cid
  run_buildah commit --signature-policy ${TESTSDIR}/policy.json $cid containers-storage:new-image
  run_buildah rm $cid

  newcid=$(buildah from --signature-policy ${TESTSDIR}/policy.json new-image)
  newroot=$(buildah mount $newcid)
  test -s $newroot/randomfile
  cmp ${TESTDIR}/randomfile $newroot/randomfile
  test -s $newroot/subdir/randomfile
  cmp ${TESTDIR}/randomfile $newroot/subdir/randomfile
  test -s $newroot/other-subdir/randomfile
  cmp ${TESTDIR}/randomfile $newroot/other-subdir/randomfile
  test -s $newroot/other-subdir/other-randomfile
  cmp ${TESTDIR}/other-randomfile $newroot/other-subdir/other-randomfile
  test -d $newroot/cwd
  test -s $newroot/cwd/randomfile
  cmp ${TESTDIR}/randomfile $newroot/cwd/randomfile
  run_buildah rm $newcid
}

@test "add-local-archive" {
  createrandom ${TESTDIR}/randomfile
  createrandom ${TESTDIR}/other-randomfile

  cid=$(buildah from --signature-policy ${TESTSDIR}/policy.json scratch)
  root=$(buildah mount $cid)
  dd if=/dev/urandom bs=1024 count=4 of=${TESTDIR}/random1
  dd if=/dev/urandom bs=1024 count=4 of=${TESTDIR}/random2
  tar -c -C ${TESTDIR}    -f ${TESTDIR}/tarball1.tar random1 random2
  mkdir ${TESTDIR}/tarball2
  dd if=/dev/urandom bs=1024 count=4 of=${TESTDIR}/tarball2/tarball2.random1
  dd if=/dev/urandom bs=1024 count=4 of=${TESTDIR}/tarball2/tarball2.random2
  tar -c -C ${TESTDIR} -z -f ${TESTDIR}/tarball2.tar.gz  tarball2
  mkdir ${TESTDIR}/tarball3
  dd if=/dev/urandom bs=1024 count=4 of=${TESTDIR}/tarball3/tarball3.random1
  dd if=/dev/urandom bs=1024 count=4 of=${TESTDIR}/tarball3/tarball3.random2
  tar -c -C ${TESTDIR} -j -f ${TESTDIR}/tarball3.tar.bz2 tarball3
  mkdir ${TESTDIR}/tarball4
  dd if=/dev/urandom bs=1024 count=4 of=${TESTDIR}/tarball4/tarball4.random1
  dd if=/dev/urandom bs=1024 count=4 of=${TESTDIR}/tarball4/tarball4.random2
  tar -c -C ${TESTDIR} -j -f ${TESTDIR}/tarball4.tar.bz2 tarball4
  # Add the files to the working directory, which should extract them all.
  run_buildah config --workingdir=/ $cid
  run_buildah add $cid ${TESTDIR}/tarball1.tar
  run_buildah add $cid ${TESTDIR}/tarball2.tar.gz
  run_buildah add $cid ${TESTDIR}/tarball3.tar.bz2
  run_buildah add $cid ${TESTDIR}/tarball4.tar.bz2
  run_buildah unmount $cid
  run_buildah commit --signature-policy ${TESTSDIR}/policy.json $cid containers-storage:new-image
  run_buildah rm $cid

  newcid=$(buildah from --signature-policy ${TESTSDIR}/policy.json new-image)
  newroot=$(buildah mount $newcid)
  test -s $newroot/random1
  cmp ${TESTDIR}/random1 $newroot/random1
  test -s $newroot/random2
  cmp ${TESTDIR}/random2 $newroot/random2
  test -s $newroot/tarball2/tarball2.random1
  cmp ${TESTDIR}/tarball2/tarball2.random1 $newroot/tarball2/tarball2.random1
  test -s $newroot/tarball2/tarball2.random2
  cmp ${TESTDIR}/tarball2/tarball2.random2 $newroot/tarball2/tarball2.random2
  test -s $newroot/tarball3/tarball3.random1
  cmp ${TESTDIR}/tarball3/tarball3.random1 $newroot/tarball3/tarball3.random1
  test -s $newroot/tarball3/tarball3.random2
  cmp ${TESTDIR}/tarball3/tarball3.random2 $newroot/tarball3/tarball3.random2
  test -s $newroot/tarball4/tarball4.random1
  cmp ${TESTDIR}/tarball4/tarball4.random1 $newroot/tarball4/tarball4.random1
  test -s $newroot/tarball4/tarball4.random2
  cmp ${TESTDIR}/tarball4/tarball4.random2 $newroot/tarball4/tarball4.random2
  run_buildah rm $newcid
}

@test "add single file creates absolute path with correct permissions" {
  imgName=ubuntu-image
  createrandom ${TESTDIR}/distutils.cfg

  cid=$(buildah from --signature-policy ${TESTSDIR}/policy.json ubuntu)
  run_buildah add $cid ${TESTDIR}/distutils.cfg /usr/lib/python3.7/distutils
  run_buildah run $cid stat -c "%a" /usr/lib/python3.7/distutils
  root=$(buildah mount $cid)
  expect_output --substring "755"
  run_buildah commit --signature-policy ${TESTSDIR}/policy.json $cid containers-storage:${imgName}
  run_buildah rm $cid

  newcid=$(buildah from --signature-policy ${TESTSDIR}/policy.json ${imgName})
  run_buildah run $newcid stat -c "%a" /usr/lib/python3.7/distutils
  expect_output --substring "755"
  run_buildah rm $newcid
}

@test "add single file creates relative path with correct permissions" {
  imgName=ubuntu-image
  createrandom ${TESTDIR}/distutils.cfg

  cid=$(buildah from --signature-policy ${TESTSDIR}/policy.json ubuntu)
  run_buildah add $cid ${TESTDIR}/distutils.cfg lib/custom
  run_buildah run $cid stat -c "%a" lib/custom
  root=$(buildah mount $cid)
  expect_output --substring "755"
  run_buildah commit --signature-policy ${TESTSDIR}/policy.json $cid containers-storage:${imgName}
  run_buildah rm $cid

  newcid=$(buildah from --signature-policy ${TESTSDIR}/policy.json ${imgName})
  run_buildah run $newcid stat -c "%a" lib/custom
  expect_output --substring "755"
  run_buildah rm $newcid
}
