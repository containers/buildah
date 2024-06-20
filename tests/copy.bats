#!/usr/bin/env bats

load helpers

@test "copy-flags-order-verification" {
  run_buildah 125 copy container1 -q /tmp/container1
  check_options_flag_err "-q"

  run_buildah 125 copy container1 --chown /tmp/container1 --quiet
  check_options_flag_err "--chown"

  run_buildah 125 copy container1 /tmp/container1 --quiet
  check_options_flag_err "--quiet"
}

@test "copy-local-multiple" {
  createrandom ${TEST_SCRATCH_DIR}/randomfile
  createrandom ${TEST_SCRATCH_DIR}/other-randomfile
  createrandom ${TEST_SCRATCH_DIR}/third-randomfile

  run_buildah from $WITH_POLICY_JSON scratch
  cid=$output
  run_buildah mount $cid
  root=$output
  run_buildah config --workingdir / $cid
  # copy ${TEST_SCRATCH_DIR}/randomfile to a file of the same name in the container's working directory
  run_buildah copy --retry 4 --retry-delay 4s $cid ${TEST_SCRATCH_DIR}/randomfile
  # copy ${TEST_SCRATCH_DIR}/other-randomfile and ${TEST_SCRATCH_DIR}/third-randomfile to a new directory named ${TEST_SCRATCH_DIR}/randomfile in the container
  run_buildah copy $cid ${TEST_SCRATCH_DIR}/other-randomfile ${TEST_SCRATCH_DIR}/third-randomfile ${TEST_SCRATCH_DIR}/randomfile
  # try to copy ${TEST_SCRATCH_DIR}/other-randomfile and ${TEST_SCRATCH_DIR}/third-randomfile to a /randomfile, which already exists and is a file
  run_buildah 125 copy $cid ${TEST_SCRATCH_DIR}/other-randomfile ${TEST_SCRATCH_DIR}/third-randomfile /randomfile
  # copy ${TEST_SCRATCH_DIR}/other-randomfile and ${TEST_SCRATCH_DIR}/third-randomfile to previously-created directory named ${TEST_SCRATCH_DIR}/randomfile in the container
  run_buildah copy $cid ${TEST_SCRATCH_DIR}/other-randomfile ${TEST_SCRATCH_DIR}/third-randomfile ${TEST_SCRATCH_DIR}/randomfile
  run_buildah rm $cid

  _prefetch alpine
  run_buildah from --quiet --pull=false $WITH_POLICY_JSON alpine
  cid=$output
  run_buildah mount $cid
  root=$output
  run_buildah config --workingdir / $cid
  run_buildah copy $cid ${TEST_SCRATCH_DIR}/randomfile
  run_buildah copy $cid ${TEST_SCRATCH_DIR}/other-randomfile ${TEST_SCRATCH_DIR}/third-randomfile ${TEST_SCRATCH_DIR}/randomfile /etc
  run_buildah rm $cid

  run_buildah from --quiet --pull=false $WITH_POLICY_JSON alpine
  cid=$output
  run_buildah mount $cid
  root=$output
  run_buildah config --workingdir / $cid
  run_buildah copy $cid "${TEST_SCRATCH_DIR}/*randomfile" /etc
  (cd ${TEST_SCRATCH_DIR}; for i in *randomfile; do cmp $i ${root}/etc/$i; done)
}

@test "copy-local-plain" {
  createrandom ${TEST_SCRATCH_DIR}/randomfile
  createrandom ${TEST_SCRATCH_DIR}/other-randomfile
  createrandom ${TEST_SCRATCH_DIR}/third-randomfile

  run_buildah from $WITH_POLICY_JSON scratch
  cid=$output
  run_buildah mount $cid
  root=$output
  run_buildah config --workingdir / $cid
  run_buildah copy $cid ${TEST_SCRATCH_DIR}/randomfile
  run_buildah copy $cid ${TEST_SCRATCH_DIR}/other-randomfile
  run_buildah unmount $cid
  run_buildah commit $WITH_POLICY_JSON $cid containers-storage:new-image
  run_buildah rm $cid

  run_buildah from --quiet $WITH_POLICY_JSON new-image
  newcid=$output
  run_buildah mount $newcid
  newroot=$output
  test -s $newroot/randomfile
  cmp ${TEST_SCRATCH_DIR}/randomfile $newroot/randomfile
  test -s $newroot/other-randomfile
  cmp ${TEST_SCRATCH_DIR}/other-randomfile $newroot/other-randomfile
}

@test "copy-local-subdirectory" {
  mkdir -p ${TEST_SCRATCH_DIR}/subdir
  createrandom ${TEST_SCRATCH_DIR}/subdir/randomfile
  createrandom ${TEST_SCRATCH_DIR}/subdir/other-randomfile

  run_buildah from $WITH_POLICY_JSON scratch
  cid=$output
  run_buildah config --workingdir /container-subdir $cid
  run_buildah copy $cid ${TEST_SCRATCH_DIR}/subdir
  run_buildah mount $cid
  root=$output
  test -s $root/container-subdir/randomfile
  cmp ${TEST_SCRATCH_DIR}/subdir/randomfile $root/container-subdir/randomfile
  test -s $root/container-subdir/other-randomfile
  cmp ${TEST_SCRATCH_DIR}/subdir/other-randomfile $root/container-subdir/other-randomfile
  run_buildah copy $cid ${TEST_SCRATCH_DIR}/subdir /other-subdir
  test -s $root/other-subdir/randomfile
  cmp ${TEST_SCRATCH_DIR}/subdir/randomfile $root/other-subdir/randomfile
  test -s $root/other-subdir/other-randomfile
  cmp ${TEST_SCRATCH_DIR}/subdir/other-randomfile $root/other-subdir/other-randomfile
}

@test "copy-local-force-directory" {
  createrandom ${TEST_SCRATCH_DIR}/randomfile

  run_buildah from $WITH_POLICY_JSON scratch
  cid=$output
  run_buildah config --workingdir / $cid
  run_buildah copy $cid ${TEST_SCRATCH_DIR}/randomfile /randomfile
  run_buildah mount $cid
  root=$output
  test -s $root/randomfile
  cmp ${TEST_SCRATCH_DIR}/randomfile $root/randomfile
  run_buildah rm $cid

  run_buildah from $WITH_POLICY_JSON scratch
  cid=$output
  run_buildah config --workingdir / $cid
  run_buildah copy $cid ${TEST_SCRATCH_DIR}/randomfile /randomsubdir/
  run_buildah mount $cid
  root=$output
  test -s $root/randomsubdir/randomfile
  cmp ${TEST_SCRATCH_DIR}/randomfile $root/randomsubdir/randomfile
}

@test "copy-url-mtime" {
  # Create a file with random content and a non-now timestamp (so we can
  # can trust that buildah correctly set mtime on copy)
  createrandom ${TEST_SCRATCH_DIR}/randomfile
  touch -t 201910310123.45 ${TEST_SCRATCH_DIR}/randomfile

  run_buildah from $WITH_POLICY_JSON scratch
  cid=$output
  run_buildah config --workingdir / $cid
  starthttpd ${TEST_SCRATCH_DIR}
  run_buildah copy $cid http://0.0.0.0:${HTTP_SERVER_PORT}/randomfile /urlfile
  stophttpd
  run_buildah mount $cid
  root=$output
  test -s $root/urlfile
  cmp ${TEST_SCRATCH_DIR}/randomfile $root/urlfile

  # Compare timestamps. Display them in human-readable form, so if there's
  # a mismatch it will be shown in the test log.
  mtime_randomfile=$(stat --format %y ${TEST_SCRATCH_DIR}/randomfile)
  mtime_urlfile=$(stat --format %y $root/urlfile)

  expect_output --from="$mtime_randomfile" "$mtime_urlfile" "mtime[randomfile] == mtime[urlfile]"
}

@test "copy --chown" {
  mkdir -p ${TEST_SCRATCH_DIR}/subdir
  mkdir -p ${TEST_SCRATCH_DIR}/other-subdir
  createrandom ${TEST_SCRATCH_DIR}/subdir/randomfile
  createrandom ${TEST_SCRATCH_DIR}/subdir/other-randomfile
  createrandom ${TEST_SCRATCH_DIR}/randomfile
  createrandom ${TEST_SCRATCH_DIR}/other-subdir/randomfile
  createrandom ${TEST_SCRATCH_DIR}/other-subdir/other-randomfile

  _prefetch alpine
  run_buildah from --quiet $WITH_POLICY_JSON alpine
  cid=$output
  run_buildah config --workingdir / $cid
  run_buildah copy --chown 1:1 $cid ${TEST_SCRATCH_DIR}/randomfile
  run_buildah copy --chown root:1 $cid ${TEST_SCRATCH_DIR}/randomfile /randomfile2
  run_buildah copy --chown nobody $cid ${TEST_SCRATCH_DIR}/randomfile /randomfile3
  run_buildah copy --chown nobody:root $cid ${TEST_SCRATCH_DIR}/subdir /subdir
  run_buildah run $cid stat -c "%u:%g" /randomfile
  expect_output "1:1" "stat ug /randomfile"

  run_buildah run $cid stat -c "%U:%g" /randomfile2
  expect_output "root:1" "stat Ug /randomfile2"

  run_buildah run $cid stat -c "%U" /randomfile3
  expect_output "nobody" "stat U /randomfile3"

  for i in randomfile other-randomfile ; do
      run_buildah run $cid stat -c "%U:%G" /subdir/$i
      expect_output "nobody:root" "stat UG /subdir/$i"
  done

  # subdir will have been implicitly created, and the --chown should have had an effect
  run_buildah run $cid stat -c "%U:%G" /subdir
  expect_output "nobody:root" "stat UG /subdir"

  run_buildah copy --chown root:root $cid ${TEST_SCRATCH_DIR}/other-subdir /subdir
  for i in randomfile other-randomfile ; do
      run_buildah run $cid stat -c "%U:%G" /subdir/$i
      expect_output "root:root" "stat UG /subdir/$i (after chown)"
  done

  # subdir itself will have not been copied (the destination directory was created implicitly), so its permissions should not have changed
  run_buildah run $cid stat -c "%U:%G" /subdir
  expect_output "nobody:root" "stat UG /subdir"
}

@test "copy --chmod" {
  mkdir -p ${TEST_SCRATCH_DIR}/subdir
  mkdir -p ${TEST_SCRATCH_DIR}/other-subdir
  createrandom ${TEST_SCRATCH_DIR}/subdir/randomfile
  createrandom ${TEST_SCRATCH_DIR}/subdir/other-randomfile
  createrandom ${TEST_SCRATCH_DIR}/randomfile
  createrandom ${TEST_SCRATCH_DIR}/other-subdir/randomfile
  createrandom ${TEST_SCRATCH_DIR}/other-subdir/other-randomfile

  _prefetch alpine
  run_buildah from --quiet $WITH_POLICY_JSON alpine
  cid=$output
  run_buildah config --workingdir / $cid
  run_buildah copy --chmod 777 $cid ${TEST_SCRATCH_DIR}/randomfile
  run_buildah copy --chmod 700 $cid ${TEST_SCRATCH_DIR}/randomfile /randomfile2
  run_buildah copy --chmod 755 $cid ${TEST_SCRATCH_DIR}/randomfile /randomfile3
  run_buildah copy --chmod 660 $cid ${TEST_SCRATCH_DIR}/subdir /subdir

  run_buildah run $cid ls -l /randomfile
  expect_output --substring rwxrwxrwx

  run_buildah run $cid ls -l /randomfile2
  expect_output --substring rwx------

  run_buildah run $cid ls -l /randomfile3
  expect_output --substring rwxr-xr-x

  for i in randomfile other-randomfile ; do
      run_buildah run $cid ls -l /subdir/$i
      expect_output --substring rw-rw----
  done

  run_buildah run $cid ls -l /subdir
  expect_output --substring rw-rw----

  run_buildah copy --chmod 600 $cid ${TEST_SCRATCH_DIR}/other-subdir /subdir
  for i in randomfile other-randomfile ; do
      run_buildah run $cid ls -l /subdir/$i
      expect_output --substring rw-------
  done
}

@test "copy-symlink" {
  createrandom ${TEST_SCRATCH_DIR}/randomfile
  ln -s ${TEST_SCRATCH_DIR}/randomfile ${TEST_SCRATCH_DIR}/link-randomfile

  run_buildah from $WITH_POLICY_JSON scratch
  cid=$output
  run_buildah mount $cid
  root=$output
  run_buildah config --workingdir / $cid
  run_buildah copy $cid ${TEST_SCRATCH_DIR}/link-randomfile
  run_buildah unmount $cid
  run_buildah commit $WITH_POLICY_JSON $cid containers-storage:new-image
  run_buildah rm $cid

  run_buildah from --quiet $WITH_POLICY_JSON new-image
  newcid=$output
  run_buildah mount $newcid
  newroot=$output
  test -s $newroot/link-randomfile
  test -f $newroot/link-randomfile
  cmp ${TEST_SCRATCH_DIR}/randomfile $newroot/link-randomfile
}

@test "ignore-socket" {
  createrandom ${TEST_SCRATCH_DIR}/randomfile
  # This seems to be the least-worst way to create a socket: run and kill nc
  nc -lkU ${TEST_SCRATCH_DIR}/test.socket &
  nc_pid=$!
  # This should succeed fairly quickly. We test with a timeout in case of
  # failure (likely reason: 'nc' not installed.)
  retries=50
  while ! test -e ${TEST_SCRATCH_DIR}/test.socket; do
      sleep 0.1
      retries=$((retries - 1))
      if [[ $retries -eq 0 ]]; then
          die "Timed out waiting for ${TEST_SCRATCH_DIR}/test.socket (is nc installed?)"
      fi
  done
  kill $nc_pid

  run_buildah from $WITH_POLICY_JSON scratch
  cid=$output
  run_buildah mount $cid
  root=$output
  run_buildah config --workingdir / $cid
  run_buildah unmount $cid
  run_buildah commit $WITH_POLICY_JSON $cid containers-storage:new-image
  run_buildah rm $cid

  run_buildah from --quiet $WITH_POLICY_JSON new-image
  newcid=$output
  run_buildah mount $newcid
  newroot=$output
  test \! -e $newroot/test.socket
}

@test "copy-symlink-archive-suffix" {
  createrandom ${TEST_SCRATCH_DIR}/randomfile.tar.gz
  ln -s ${TEST_SCRATCH_DIR}/randomfile.tar.gz ${TEST_SCRATCH_DIR}/link-randomfile.tar.gz

  run_buildah from $WITH_POLICY_JSON scratch
  cid=$output
  run_buildah mount $cid
  root=$output
  run_buildah config --workingdir / $cid
  run_buildah copy $cid ${TEST_SCRATCH_DIR}/link-randomfile.tar.gz
  run_buildah unmount $cid
  run_buildah commit $WITH_POLICY_JSON $cid containers-storage:new-image
  run_buildah rm $cid

  run_buildah from --quiet $WITH_POLICY_JSON new-image
  newcid=$output
  run_buildah mount $newcid
  newroot=$output
  test -s $newroot/link-randomfile.tar.gz
  test -f $newroot/link-randomfile.tar.gz
  cmp ${TEST_SCRATCH_DIR}/randomfile.tar.gz $newroot/link-randomfile.tar.gz
}

@test "copy-detect-missing-data" {
  _prefetch busybox

  : > ${TEST_SCRATCH_DIR}/Dockerfile
  echo FROM busybox AS builder                                >> ${TEST_SCRATCH_DIR}/Dockerfile
  echo FROM scratch                                           >> ${TEST_SCRATCH_DIR}/Dockerfile
  echo COPY --from=builder /bin/-no-such-file-error- /usr/bin >> ${TEST_SCRATCH_DIR}/Dockerfile
  run_buildah 125 build-using-dockerfile $WITH_POLICY_JSON ${TEST_SCRATCH_DIR}
  expect_output --substring "no such file or directory"
}

@test "copy --ignorefile" {
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

  run_buildah copy --contextdir=${mytest} --ignorefile ${mytest}/.ignore $cid ${mytest} /stuff

  run_buildah mount $cid
  mnt=$output
  run find $mnt -printf "%P\n"
  filelist=$(LC_ALL=C sort <<<"$output")
  run_buildah umount $cid
  expect_output --from="$filelist" "$expect" "container file list"
}

@test "copy-quiet" {
  createrandom ${TEST_SCRATCH_DIR}/randomfile
  _prefetch alpine
  run_buildah from --quiet --pull=false $WITH_POLICY_JSON alpine
  cid=$output
  run_buildah mount $cid
  root=$output
  run_buildah copy --quiet $cid ${TEST_SCRATCH_DIR}/randomfile /
  expect_output ""
  cmp ${TEST_SCRATCH_DIR}/randomfile $root/randomfile
  run_buildah umount $cid
  run_buildah rm $cid
}

@test "copy-from-container" {
  _prefetch busybox
  createrandom ${TEST_SCRATCH_DIR}/randomfile
  run_buildah from --quiet $WITH_POLICY_JSON busybox
  from=$output
  run_buildah from --quiet $WITH_POLICY_JSON busybox
  cid=$output
  run_buildah copy --quiet $from ${TEST_SCRATCH_DIR}/randomfile /tmp/random
  expect_output ""
  run_buildah copy --quiet $WITH_POLICY_JSON --from $from $cid /tmp/random /tmp/random # absolute path
  expect_output ""
  run_buildah copy --quiet $WITH_POLICY_JSON --from $from $cid  tmp/random /tmp/random2 # relative path
  expect_output ""
  run_buildah mount $cid
  croot=$output
  cmp ${TEST_SCRATCH_DIR}/randomfile ${croot}/tmp/random
  cmp ${TEST_SCRATCH_DIR}/randomfile ${croot}/tmp/random2
}

@test "copy-container-root" {
  _prefetch busybox
  createrandom ${TEST_SCRATCH_DIR}/randomfile
  run_buildah from --quiet $WITH_POLICY_JSON busybox
  from=$output
  run_buildah from --quiet $WITH_POLICY_JSON busybox
  cid=$output
  run_buildah copy --quiet $from ${TEST_SCRATCH_DIR}/randomfile /tmp/random
  expect_output ""
  run_buildah copy --quiet $WITH_POLICY_JSON --from $from $cid / /tmp/
  expect_output "" || \
    expect_output --substring "copier: file disappeared while reading"
  run_buildah mount $cid
  croot=$output
  cmp ${TEST_SCRATCH_DIR}/randomfile ${croot}/tmp/tmp/random
}

@test "add-from-image" {
  _prefetch busybox
  run_buildah from --quiet $WITH_POLICY_JSON busybox
  cid=$output
  run_buildah add $WITH_POLICY_JSON --quiet --from ubuntu $cid /etc/passwd /tmp/passwd # should pull the image, absolute path
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

@test "copy with .dockerignore" {
  _prefetch alpine busybox
  run_buildah from --quiet $WITH_POLICY_JSON alpine
  from=$output
  run_buildah copy --contextdir=$BUDFILES/dockerignore $from $BUDFILES/dockerignore ./

  run_buildah 1 run $from ls -l test1.txt

  run_buildah run $from ls -l test2.txt

  run_buildah 1 run $from ls -l sub1.txt

  run_buildah 1 run $from ls -l sub2.txt

  run_buildah 1 run $from ls -l subdir/
}

@test "copy-preserving-extended-attributes" {
  createrandom ${TEST_SCRATCH_DIR}/randomfile
  # if we need to change which image we use, any image that can provide a working setattr/setcap/getfattr will do
  image="quay.io/libpod/fedora-minimal:34"
  if ! which setfattr > /dev/null 2> /dev/null; then
    skip "setfattr not available, unable to check if it'll work in filesystem at ${TEST_SCRATCH_DIR}"
  fi
  run setfattr -n user.yeah -v butno ${TEST_SCRATCH_DIR}/root
  if [ "$status" -ne 0 ] ; then
    if [[ "$output" =~ "not supported" ]] ; then
      skip "setfattr not supported in filesystem at ${TEST_SCRATCH_DIR}"
    fi
    skip "$output"
  fi
  _prefetch $image
  run_buildah from --quiet $WITH_POLICY_JSON $image
  first="$output"
  run_buildah run $first microdnf -y install /usr/bin/setfattr /usr/sbin/setcap
  run_buildah copy $first ${TEST_SCRATCH_DIR}/randomfile /
  # set security.capability
  run_buildah run $first setcap cap_setuid=ep /randomfile
  # set user.something
  run_buildah run $first setfattr -n user.yeah -v butno /randomfile
  # copy the file to a second container
  run_buildah from --quiet $WITH_POLICY_JSON $image
  second="$output"
  run_buildah run $second microdnf -y install /usr/bin/getfattr
  run_buildah copy --from $first $second /randomfile /
  # compare what the extended attributes look like. if we're on a system with SELinux, there's a label in here, too
  run_buildah run $first sh -c "getfattr -d -m . --absolute-names /randomfile | grep -v ^security.selinux | sort"
  expected="$output"
  run_buildah run $second sh -c "getfattr -d -m . --absolute-names /randomfile | grep -v ^security.selinux | sort"
  expect_output "$expected"
}

@test "copy-relative-context-dir" {
  image=busybox
  _prefetch $image
  mkdir -p ${TEST_SCRATCH_DIR}/context
  createrandom ${TEST_SCRATCH_DIR}/context/excluded_test_file
  createrandom ${TEST_SCRATCH_DIR}/context/test_file
  echo excluded_test_file | tee ${TEST_SCRATCH_DIR}/context/.containerignore | tee ${TEST_SCRATCH_DIR}/context/.dockerignore
  run_buildah from --quiet $WITH_POLICY_JSON $image
  ctr="$output"
  cd ${TEST_SCRATCH_DIR}/context
  run_buildah copy --contextdir . $ctr / /opt/
  run_buildah run $ctr ls -1 /opt/
  expect_line_count 1
  assert "$output" = "test_file" "only contents of copied directory"
}

@test "copy-file-relative-context-dir" {
  image=busybox
  _prefetch $image
  mkdir -p ${TEST_SCRATCH_DIR}/context
  createrandom ${TEST_SCRATCH_DIR}/context/test_file
  run_buildah from --quiet $WITH_POLICY_JSON $image
  ctr="$output"
  run_buildah copy --contextdir ${TEST_SCRATCH_DIR}/context $ctr test_file /opt/
  run_buildah run $ctr ls -1 /opt/
  expect_line_count 1
  assert "$output" = "test_file" "only the one file"
}

@test "copy-file-absolute-context-dir" {
  image=busybox
  _prefetch $image
  mkdir -p ${TEST_SCRATCH_DIR}/context/subdir
  createrandom ${TEST_SCRATCH_DIR}/context/subdir/test_file
  run_buildah from --quiet $WITH_POLICY_JSON $image
  ctr="$output"
  run_buildah copy --contextdir ${TEST_SCRATCH_DIR}/context $ctr /subdir/test_file /opt/
  run_buildah run $ctr ls -1 /opt/
  expect_line_count 1
  assert "$output" = "test_file" "only the one file"
}

@test "copy-file-relative-no-context-dir" {
  image=busybox
  _prefetch $image
  mkdir -p ${TEST_SCRATCH_DIR}/context
  createrandom ${TEST_SCRATCH_DIR}/context/test_file
  run_buildah from --quiet $WITH_POLICY_JSON $image
  ctr="$output"
  # we're not in that directory currently
  run_buildah 125 copy $ctr test_file /opt/
  # now we are
  cd ${TEST_SCRATCH_DIR}/context
  run_buildah copy $ctr test_file /opt/
  run_buildah run $ctr ls -1 /opt/
  expect_line_count 1
  assert "$output" = "test_file" "only the one file"
}

@test "copy-from-ownership" {
  # Build both a container and an image that have contents owned by a
  # non-default user.
  truncate -s 256 ${TEST_SCRATCH_DIR}/random-file-1
  truncate -s 256 ${TEST_SCRATCH_DIR}/random-file-2
  truncate -s 256 ${TEST_SCRATCH_DIR}/random-file-3
  truncate -s 256 ${TEST_SCRATCH_DIR}/random-file-4
  truncate -s 256 ${TEST_SCRATCH_DIR}/random-file-5
  truncate -s 256 ${TEST_SCRATCH_DIR}/random-file-6
  run_buildah from scratch
  sourcectr="$output"
  run_buildah copy --chown 123:123 $sourcectr ${TEST_SCRATCH_DIR}/random-file-1
  run_buildah copy --chown 123:123 $sourcectr ${TEST_SCRATCH_DIR}/random-file-2
  run_buildah copy --chown 456:456 $sourcectr ${TEST_SCRATCH_DIR}/random-file-4
  run_buildah copy --chown 456:456 $sourcectr ${TEST_SCRATCH_DIR}/random-file-5
  sourceimg=testimage
  run_buildah commit $sourcectr $sourceimg
  _prefetch busybox
  run_buildah from --pull=never $WITH_POLICY_JSON busybox
  ctr="$output"
  run_buildah copy $ctr ${TEST_SCRATCH_DIR}/random-file-3
  run_buildah copy --from=$sourceimg $ctr /random-file-1 # should be preserved as 123:123
  run_buildah copy --from=$sourceimg --chown=456:456 $ctr /random-file-2
  run_buildah copy --from=$sourcectr $ctr /random-file-4 # should be preserved as 456:456
  run_buildah copy --from=$sourcectr --chown=123:123 $ctr /random-file-5
  run_buildah copy $ctr ${TEST_SCRATCH_DIR}/random-file-3
  run_buildah copy --chown=789:789 $ctr ${TEST_SCRATCH_DIR}/random-file-6
  run_buildah run $ctr stat -c %u:%g /random-file-1
  assert 123:123
  run_buildah run $ctr stat -c %u:%g /random-file-2
  assert 456:456
  run_buildah run $ctr stat -c %u:%g /random-file-3
  assert 0:0
  run_buildah run $ctr stat -c %u:%g /random-file-4
  assert 456:456
  run_buildah run $ctr stat -c %u:%g /random-file-5
  assert 123:123
  run_buildah run $ctr stat -c %u:%g /random-file-6
  assert 789:789
}
