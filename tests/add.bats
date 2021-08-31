#!/usr/bin/env bats

load helpers

@test "add-flags-order-verification" {
  run_buildah 125 add container1 -q /tmp/container1
  check_options_flag_err "-q"

  run_buildah 125 add container1 --chown /tmp/container1 --quiet
  check_options_flag_err "--chown"

  run_buildah 125 add container1 /tmp/container1 --quiet
  check_options_flag_err "--quiet"
}

@test "add-local-plain" {
  createrandom ${TESTDIR}/randomfile
  createrandom ${TESTDIR}/other-randomfile

  run_buildah from --signature-policy ${TESTSDIR}/policy.json scratch
  cid=$output
  run_buildah mount $cid
  root=$output
  mkdir $root/subdir $root/other-subdir
  # Copy a file to the working directory
  run_buildah config --workingdir=/ $cid
  run_buildah add $cid ${TESTDIR}/randomfile
  # Copy a file to a specific subdirectory
  run_buildah add $cid ${TESTDIR}/randomfile /subdir
  # Copy two files to a specific subdirectory
  run_buildah add $cid ${TESTDIR}/randomfile ${TESTDIR}/other-randomfile /other-subdir
  # Copy two files to a specific location, which succeeds because we can create it as a directory.
  run_buildah add $cid ${TESTDIR}/randomfile ${TESTDIR}/other-randomfile /notthereyet-subdir
  # Copy two files to a specific location, which fails because it's not a directory.
  run_buildah 125 add $cid ${TESTDIR}/randomfile ${TESTDIR}/other-randomfile /randomfile
  # Copy a file to a different working directory
  run_buildah config --workingdir=/cwd $cid
  run_buildah add $cid ${TESTDIR}/randomfile
  run_buildah unmount $cid
  run_buildah commit --signature-policy ${TESTSDIR}/policy.json $cid containers-storage:new-image
  run_buildah rm $cid

  run_buildah from --signature-policy ${TESTSDIR}/policy.json new-image
  newcid=$output
  run_buildah mount $newcid
  newroot=$output
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

  run_buildah from --signature-policy ${TESTSDIR}/policy.json scratch
  cid=$output

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
  run_buildah commit --signature-policy ${TESTSDIR}/policy.json $cid containers-storage:new-image
  run_buildah rm $cid

  run_buildah from --signature-policy ${TESTSDIR}/policy.json new-image
  newcid=$output
  run_buildah mount $newcid
  newroot=$output
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
}

@test "add single file creates absolute path with correct permissions" {
  _prefetch ubuntu
  imgName=ubuntu-image
  createrandom ${TESTDIR}/distutils.cfg
  permission=$(stat -c "%a" ${TESTDIR}/distutils.cfg)

  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json ubuntu
  cid=$output
  run_buildah add $cid ${TESTDIR}/distutils.cfg /usr/lib/python3.7/distutils
  run_buildah run $cid stat -c "%a" /usr/lib/python3.7/distutils
  expect_output $permission
  run_buildah commit --signature-policy ${TESTSDIR}/policy.json $cid containers-storage:${imgName}
  run_buildah rm $cid

  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json ${imgName}
  newcid=$output
  run_buildah run $newcid stat -c "%a" /usr/lib/python3.7/distutils
  expect_output $permission
}

@test "add single file creates relative path with correct permissions" {
  _prefetch ubuntu
  imgName=ubuntu-image
  createrandom ${TESTDIR}/distutils.cfg
  permission=$(stat -c "%a" ${TESTDIR}/distutils.cfg)

  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json ubuntu
  cid=$output
  run_buildah add $cid ${TESTDIR}/distutils.cfg lib/custom
  run_buildah run $cid stat -c "%a" lib/custom
  expect_output $permission
  run_buildah commit --signature-policy ${TESTSDIR}/policy.json $cid containers-storage:${imgName}
  run_buildah rm $cid

  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json ${imgName}
  newcid=$output
  run_buildah run $newcid stat -c "%a" lib/custom
  expect_output $permission
}

@test "add with chown" {
  _prefetch busybox
  createrandom ${TESTDIR}/randomfile
  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json busybox
  cid=$output
  run_buildah add --chown bin:bin $cid ${TESTDIR}/randomfile /tmp/random
  run_buildah run $cid ls -l /tmp/random

  expect_output --substring bin.*bin
}

@test "add with chmod" {
  _prefetch busybox
  createrandom ${TESTDIR}/randomfile
  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json busybox
  cid=$output
  run_buildah add --chmod 777 $cid ${TESTDIR}/randomfile /tmp/random
  run_buildah run $cid ls -l /tmp/random

  expect_output --substring rwxrwxrwx
}

@test "add url" {
  _prefetch busybox
  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json busybox
  cid=$output
  run_buildah add $cid https://github.com/containers/buildah/raw/main/README.md
  run_buildah run $cid ls /README.md

  run_buildah add $cid https://github.com/containers/buildah/raw/main/README.md /home
  run_buildah run $cid ls /home/README.md
}

@test "add relative" {
  # make sure we don't get thrown by relative source locations
  _prefetch busybox
  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json busybox
  cid=$output

  run_buildah add $cid deny.json /
  run_buildah run $cid ls /deny.json

  run_buildah add $cid ./docker.json /
  run_buildah run $cid ls /docker.json

  run_buildah add $cid tools/Makefile /
  run_buildah run $cid ls /Makefile
}

@test "add --ignorefile" {
  mytest=${TESTDIR}/mytest
  mkdir -p ${mytest}
  touch ${mytest}/mystuff
  touch ${mytest}/source.go
  mkdir -p ${mytest}/notmystuff
  touch ${mytest}/notmystuff/notmystuff
  cat > ${mytest}/.ignore << _EOF
*.go
.ignore
notmystuff
_EOF

expect="
stuff
stuff/mystuff"

  run_buildah from --signature-policy ${TESTSDIR}/policy.json scratch
  cid=$output

  run_buildah 125 copy --ignorefile ${mytest}/.ignore $cid ${mytest} /stuff
  expect_output -- "--ignorefile option requires that you specify a context dir using --contextdir" "container file list"

  run_buildah add --contextdir=${mytest} --ignorefile ${mytest}/.ignore $cid ${mytest} /stuff

  run_buildah mount $cid
  mnt=$output
  run find $mnt -printf "%P\n"
  filelist=$(LC_ALL=C sort <<<"$output")
  run_buildah umount $cid
  expect_output --from="$filelist" "$expect" "container file list"
}

@test "add quietly" {
  _prefetch busybox
  createrandom ${TESTDIR}/randomfile
  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json busybox
  cid=$output
  run_buildah add --quiet $cid ${TESTDIR}/randomfile /tmp/random
  expect_output ""
  run_buildah mount $cid
  croot=$output
  cmp ${TESTDIR}/randomfile ${croot}/tmp/random
}

@test "add from container" {
  _prefetch busybox
  createrandom ${TESTDIR}/randomfile
  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json busybox
  from=$output
  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json busybox
  cid=$output
  run_buildah add --quiet $from ${TESTDIR}/randomfile /tmp/random
  expect_output ""
  run_buildah add --quiet --signature-policy ${TESTSDIR}/policy.json --from $from $cid /tmp/random /tmp/random # absolute path
  expect_output ""
  run_buildah add --quiet --signature-policy ${TESTSDIR}/policy.json --from $from $cid  tmp/random /tmp/random2 # relative path
  expect_output ""
  run_buildah mount $cid
  croot=$output
  cmp ${TESTDIR}/randomfile ${croot}/tmp/random
  cmp ${TESTDIR}/randomfile ${croot}/tmp/random2
}

@test "add from image" {
  _prefetch busybox
  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json busybox
  cid=$output
  run_buildah add --quiet --signature-policy ${TESTSDIR}/policy.json --from ubuntu $cid /etc/passwd /tmp/passwd # should pull the image, absolute path
  expect_output ""
  run_buildah add --quiet --signature-policy ${TESTSDIR}/policy.json --from ubuntu $cid  etc/passwd /tmp/passwd2 # relative path
  expect_output ""
  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json ubuntu
  ubuntu=$output
  run_buildah mount $cid
  croot=$output
  run_buildah mount $ubuntu
  ubuntu=$output
  cmp $ubuntu/etc/passwd ${croot}/tmp/passwd
  cmp $ubuntu/etc/passwd ${croot}/tmp/passwd2
}
