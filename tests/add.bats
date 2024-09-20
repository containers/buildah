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
  createrandom ${TEST_SCRATCH_DIR}/randomfile
  createrandom ${TEST_SCRATCH_DIR}/other-randomfile

  run_buildah from $WITH_POLICY_JSON scratch
  cid=$output
  run_buildah mount $cid
  root=$output
  mkdir $root/subdir $root/other-subdir
  # Copy a file to the working directory
  run_buildah config --workingdir=/ $cid
  run_buildah add --retry 4 --retry-delay 4s $cid ${TEST_SCRATCH_DIR}/randomfile
  # Copy a file to a specific subdirectory
  run_buildah add $cid ${TEST_SCRATCH_DIR}/randomfile /subdir
  # Copy two files to a specific subdirectory
  run_buildah add $cid ${TEST_SCRATCH_DIR}/randomfile ${TEST_SCRATCH_DIR}/other-randomfile /other-subdir
  # Copy two files to a specific location, which succeeds because we can create it as a directory.
  run_buildah add $cid ${TEST_SCRATCH_DIR}/randomfile ${TEST_SCRATCH_DIR}/other-randomfile /notthereyet-subdir
  # Copy two files to a specific location, which fails because it's not a directory.
  run_buildah 125 add $cid ${TEST_SCRATCH_DIR}/randomfile ${TEST_SCRATCH_DIR}/other-randomfile /randomfile
  # Copy a file to a different working directory
  run_buildah config --workingdir=/cwd $cid
  run_buildah add $cid ${TEST_SCRATCH_DIR}/randomfile
  run_buildah unmount $cid
  run_buildah commit $WITH_POLICY_JSON $cid containers-storage:new-image
  run_buildah rm $cid

  run_buildah from $WITH_POLICY_JSON new-image
  newcid=$output
  run_buildah mount $newcid
  newroot=$output
  test -s $newroot/randomfile
  cmp ${TEST_SCRATCH_DIR}/randomfile $newroot/randomfile
  test -s $newroot/subdir/randomfile
  cmp ${TEST_SCRATCH_DIR}/randomfile $newroot/subdir/randomfile
  test -s $newroot/other-subdir/randomfile
  cmp ${TEST_SCRATCH_DIR}/randomfile $newroot/other-subdir/randomfile
  test -s $newroot/other-subdir/other-randomfile
  cmp ${TEST_SCRATCH_DIR}/other-randomfile $newroot/other-subdir/other-randomfile
  test -d $newroot/cwd
  test -s $newroot/cwd/randomfile
  cmp ${TEST_SCRATCH_DIR}/randomfile $newroot/cwd/randomfile
  run_buildah rm $newcid
}

@test "add-local-archive" {
  createrandom ${TEST_SCRATCH_DIR}/randomfile
  createrandom ${TEST_SCRATCH_DIR}/other-randomfile

  run_buildah from $WITH_POLICY_JSON scratch
  cid=$output

  dd if=/dev/urandom bs=1024 count=4 of=${TEST_SCRATCH_DIR}/random1
  dd if=/dev/urandom bs=1024 count=4 of=${TEST_SCRATCH_DIR}/random2
  tar -c -C ${TEST_SCRATCH_DIR}    -f ${TEST_SCRATCH_DIR}/tarball1.tar random1 random2
  mkdir ${TEST_SCRATCH_DIR}/tarball2
  dd if=/dev/urandom bs=1024 count=4 of=${TEST_SCRATCH_DIR}/tarball2/tarball2.random1
  dd if=/dev/urandom bs=1024 count=4 of=${TEST_SCRATCH_DIR}/tarball2/tarball2.random2
  tar -c -C ${TEST_SCRATCH_DIR} -z -f ${TEST_SCRATCH_DIR}/tarball2.tar.gz  tarball2
  mkdir ${TEST_SCRATCH_DIR}/tarball3
  dd if=/dev/urandom bs=1024 count=4 of=${TEST_SCRATCH_DIR}/tarball3/tarball3.random1
  dd if=/dev/urandom bs=1024 count=4 of=${TEST_SCRATCH_DIR}/tarball3/tarball3.random2
  tar -c -C ${TEST_SCRATCH_DIR} -j -f ${TEST_SCRATCH_DIR}/tarball3.tar.bz2 tarball3
  mkdir ${TEST_SCRATCH_DIR}/tarball4
  dd if=/dev/urandom bs=1024 count=4 of=${TEST_SCRATCH_DIR}/tarball4/tarball4.random1
  dd if=/dev/urandom bs=1024 count=4 of=${TEST_SCRATCH_DIR}/tarball4/tarball4.random2
  tar -c -C ${TEST_SCRATCH_DIR} -j -f ${TEST_SCRATCH_DIR}/tarball4.tar.bz2 tarball4
  # Add the files to the working directory, which should extract them all.
  run_buildah config --workingdir=/ $cid
  run_buildah add $cid ${TEST_SCRATCH_DIR}/tarball1.tar
  run_buildah add $cid ${TEST_SCRATCH_DIR}/tarball2.tar.gz
  run_buildah add $cid ${TEST_SCRATCH_DIR}/tarball3.tar.bz2
  run_buildah add $cid ${TEST_SCRATCH_DIR}/tarball4.tar.bz2
  run_buildah commit $WITH_POLICY_JSON $cid containers-storage:new-image
  run_buildah rm $cid

  run_buildah from $WITH_POLICY_JSON new-image
  newcid=$output
  run_buildah mount $newcid
  newroot=$output
  test -s $newroot/random1
  cmp ${TEST_SCRATCH_DIR}/random1 $newroot/random1
  test -s $newroot/random2
  cmp ${TEST_SCRATCH_DIR}/random2 $newroot/random2
  test -s $newroot/tarball2/tarball2.random1
  cmp ${TEST_SCRATCH_DIR}/tarball2/tarball2.random1 $newroot/tarball2/tarball2.random1
  test -s $newroot/tarball2/tarball2.random2
  cmp ${TEST_SCRATCH_DIR}/tarball2/tarball2.random2 $newroot/tarball2/tarball2.random2
  test -s $newroot/tarball3/tarball3.random1
  cmp ${TEST_SCRATCH_DIR}/tarball3/tarball3.random1 $newroot/tarball3/tarball3.random1
  test -s $newroot/tarball3/tarball3.random2
  cmp ${TEST_SCRATCH_DIR}/tarball3/tarball3.random2 $newroot/tarball3/tarball3.random2
  test -s $newroot/tarball4/tarball4.random1
  cmp ${TEST_SCRATCH_DIR}/tarball4/tarball4.random1 $newroot/tarball4/tarball4.random1
  test -s $newroot/tarball4/tarball4.random2
  cmp ${TEST_SCRATCH_DIR}/tarball4/tarball4.random2 $newroot/tarball4/tarball4.random2
}

@test "add single file creates absolute path with correct permissions" {
  _prefetch ubuntu
  imgName=ubuntu-image
  createrandom ${TEST_SCRATCH_DIR}/distutils.cfg
  permission=$(stat -c "%a" ${TEST_SCRATCH_DIR}/distutils.cfg)

  run_buildah from --quiet $WITH_POLICY_JSON ubuntu
  cid=$output
  run_buildah add $cid ${TEST_SCRATCH_DIR}/distutils.cfg /usr/lib/python3.7/distutils
  run_buildah run $cid stat -c "%a" /usr/lib/python3.7/distutils
  expect_output $permission
  run_buildah commit $WITH_POLICY_JSON $cid containers-storage:${imgName}
  run_buildah rm $cid

  run_buildah from --quiet $WITH_POLICY_JSON ${imgName}
  newcid=$output
  run_buildah run $newcid stat -c "%a" /usr/lib/python3.7/distutils
  expect_output $permission
}

@test "add single file creates relative path with correct permissions" {
  _prefetch ubuntu
  imgName=ubuntu-image
  createrandom ${TEST_SCRATCH_DIR}/distutils.cfg
  permission=$(stat -c "%a" ${TEST_SCRATCH_DIR}/distutils.cfg)

  run_buildah from --quiet $WITH_POLICY_JSON ubuntu
  cid=$output
  run_buildah add $cid ${TEST_SCRATCH_DIR}/distutils.cfg lib/custom
  run_buildah run $cid stat -c "%a" lib/custom
  expect_output $permission
  run_buildah commit $WITH_POLICY_JSON $cid containers-storage:${imgName}
  run_buildah rm $cid

  run_buildah from --quiet $WITH_POLICY_JSON ${imgName}
  newcid=$output
  run_buildah run $newcid stat -c "%a" lib/custom
  expect_output $permission
}

@test "add with chown" {
  _prefetch busybox
  createrandom ${TEST_SCRATCH_DIR}/randomfile
  run_buildah from --quiet $WITH_POLICY_JSON busybox
  cid=$output
  run_buildah add --chown bin:bin $cid ${TEST_SCRATCH_DIR}/randomfile /tmp/random
  run_buildah run $cid ls -l /tmp/random

  expect_output --substring bin.*bin
}

@test "add with chmod" {
  _prefetch busybox
  createrandom ${TEST_SCRATCH_DIR}/randomfile
  run_buildah from --quiet $WITH_POLICY_JSON busybox
  cid=$output
  run_buildah add --chmod 777 $cid ${TEST_SCRATCH_DIR}/randomfile /tmp/random
  run_buildah run $cid ls -l /tmp/random

  expect_output --substring rwxrwxrwx
}

@test "add url" {
  _prefetch busybox
  run_buildah from --quiet $WITH_POLICY_JSON busybox
  cid=$output
  run_buildah add $cid https://github.com/containers/buildah/raw/main/README.md
  run_buildah run $cid ls /README.md

  run_buildah add $cid https://github.com/containers/buildah/raw/main/README.md /home
  run_buildah run $cid ls /home/README.md
}

@test "add relative" {
  # make sure we don't get thrown by relative source locations
  _prefetch busybox
  run_buildah from --quiet $WITH_POLICY_JSON busybox
  cid=$output

  run_buildah add $cid deny.json /
  run_buildah run $cid ls /deny.json

  run_buildah add $cid ./docker.json /
  run_buildah run $cid ls /docker.json

  run_buildah add $cid tools/Makefile /
  run_buildah run $cid ls /Makefile
}

@test "add --ignorefile" {
  mytest=${TEST_SCRATCH_DIR}/mytest
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

  run_buildah from $WITH_POLICY_JSON scratch
  cid=$output

  run_buildah 125 copy --ignorefile ${mytest}/.ignore $cid ${mytest} /stuff
  expect_output -- "Error: --ignorefile option requires that you specify a context dir using --contextdir" "container file list"

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
  createrandom ${TEST_SCRATCH_DIR}/randomfile
  run_buildah from --quiet $WITH_POLICY_JSON busybox
  cid=$output
  run_buildah add --quiet $cid ${TEST_SCRATCH_DIR}/randomfile /tmp/random
  expect_output ""
  run_buildah mount $cid
  croot=$output
  cmp ${TEST_SCRATCH_DIR}/randomfile ${croot}/tmp/random
}

@test "add from container" {
  _prefetch busybox
  createrandom ${TEST_SCRATCH_DIR}/randomfile
  run_buildah from --quiet $WITH_POLICY_JSON busybox
  from=$output
  run_buildah from --quiet $WITH_POLICY_JSON busybox
  cid=$output
  run_buildah add --quiet $from ${TEST_SCRATCH_DIR}/randomfile /tmp/random
  expect_output ""
  run_buildah add --quiet $WITH_POLICY_JSON --from $from $cid /tmp/random /tmp/random # absolute path
  expect_output ""
  run_buildah add --quiet $WITH_POLICY_JSON --from $from $cid  tmp/random /tmp/random2 # relative path
  expect_output ""
  run_buildah mount $cid
  croot=$output
  cmp ${TEST_SCRATCH_DIR}/randomfile ${croot}/tmp/random
  cmp ${TEST_SCRATCH_DIR}/randomfile ${croot}/tmp/random2
}

@test "add from image" {
  _prefetch busybox ubuntu
  run_buildah from --quiet $WITH_POLICY_JSON busybox
  cid=$output
  run_buildah add --quiet $WITH_POLICY_JSON --from ubuntu $cid /etc/passwd /tmp/passwd # should pull the image, absolute path
  expect_output ""
  run_buildah add --quiet $WITH_POLICY_JSON --from ubuntu $cid  etc/passwd /tmp/passwd2 # relative path
  expect_output ""
  run_buildah from --quiet $WITH_POLICY_JSON ubuntu
  ubuntu=$output
  run_buildah mount $cid
  croot=$output
  run_buildah mount $ubuntu
  ubuntu=$output
  cmp $ubuntu/etc/passwd ${croot}/tmp/passwd
  cmp $ubuntu/etc/passwd ${croot}/tmp/passwd2
}

@test "add url with checksum flag" {
  _prefetch busybox
  run_buildah from --quiet $WITH_POLICY_JSON busybox
  cid=$output
  run_buildah add --checksum=sha256:4fd3aed66b5488b45fe83dd11842c2324fadcc38e1217bb45fbd28d660afdd39 $cid https://raw.githubusercontent.com/containers/buildah/bf3b55ba74102cc2503eccbaeffe011728d46b20/README.md /
  run_buildah run $cid ls /README.md
}

@test "add url with bad checksum" {
  _prefetch busybox
  run_buildah from --quiet $WITH_POLICY_JSON busybox
  cid=$output
  run_buildah 125 add --checksum=sha256:0000000000000000000000000000000000000000000000000000000000000000 $cid https://raw.githubusercontent.com/containers/buildah/bf3b55ba74102cc2503eccbaeffe011728d46b20/README.md /
  expect_output --substring "unexpected response digest for \"https://raw.githubusercontent.com/containers/buildah/bf3b55ba74102cc2503eccbaeffe011728d46b20/README.md\": sha256:4fd3aed66b5488b45fe83dd11842c2324fadcc38e1217bb45fbd28d660afdd39, want sha256:0000000000000000000000000000000000000000000000000000000000000000"
}

@test "add path with checksum flag" {
  _prefetch busybox
  createrandom ${TEST_SCRATCH_DIR}/randomfile
  run_buildah from --quiet $WITH_POLICY_JSON busybox
  cid=$output
  run_buildah 125 add --checksum=sha256:0000000000000000000000000000000000000000000000000000000000000000 $cid ${TEST_SCRATCH_DIR}/randomfile /
  expect_output --substring "checksum flag is not supported for local sources"
}

@test "add file with IMA xattr" {
    if ! getfattr -d -n 'security.ima' /usr/libexec/catatonit/catatonit | grep -q ima; then
	skip "catatonit does not have IMA xattr, cannot perform test"
    fi

    run_buildah from --quiet scratch
    cid=$output

    # We do not care if the attribute was actually added, as rootless is allowed to discard it.
    # Only that the add was actually successful.
    run_buildah add $cid /usr/libexec/catatonit/catatonit /catatonit
}
