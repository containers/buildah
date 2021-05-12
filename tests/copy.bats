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
  createrandom ${TESTDIR}/randomfile
  createrandom ${TESTDIR}/other-randomfile
  createrandom ${TESTDIR}/third-randomfile

  run_buildah from --signature-policy ${TESTSDIR}/policy.json scratch
  cid=$output
  run_buildah mount $cid
  root=$output
  run_buildah config --workingdir / $cid
  # copy ${TESTDIR}/randomfile to a file of the same name in the container's working directory
  run_buildah copy $cid ${TESTDIR}/randomfile
  # copy ${TESTDIR}/other-randomfile and ${TESTDIR}/third-randomfile to a new directory named ${TESTDIR}/randomfile in the container
  run_buildah copy $cid ${TESTDIR}/other-randomfile ${TESTDIR}/third-randomfile ${TESTDIR}/randomfile
  # try to copy ${TESTDIR}/other-randomfile and ${TESTDIR}/third-randomfile to a /randomfile, which already exists and is a file
  run_buildah 125 copy $cid ${TESTDIR}/other-randomfile ${TESTDIR}/third-randomfile /randomfile
  # copy ${TESTDIR}/other-randomfile and ${TESTDIR}/third-randomfile to previously-created directory named ${TESTDIR}/randomfile in the container
  run_buildah copy $cid ${TESTDIR}/other-randomfile ${TESTDIR}/third-randomfile ${TESTDIR}/randomfile
  run_buildah rm $cid

  _prefetch alpine
  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
  cid=$output
  run_buildah mount $cid
  root=$output
  run_buildah config --workingdir / $cid
  run_buildah copy $cid ${TESTDIR}/randomfile
  run_buildah copy $cid ${TESTDIR}/other-randomfile ${TESTDIR}/third-randomfile ${TESTDIR}/randomfile /etc
  run_buildah rm $cid

  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
  cid=$output
  run_buildah mount $cid
  root=$output
  run_buildah config --workingdir / $cid
  run_buildah copy $cid "${TESTDIR}/*randomfile" /etc
  (cd ${TESTDIR}; for i in *randomfile; do cmp $i ${root}/etc/$i; done)
}

@test "copy-local-plain" {
  createrandom ${TESTDIR}/randomfile
  createrandom ${TESTDIR}/other-randomfile
  createrandom ${TESTDIR}/third-randomfile

  run_buildah from --signature-policy ${TESTSDIR}/policy.json scratch
  cid=$output
  run_buildah mount $cid
  root=$output
  run_buildah config --workingdir / $cid
  run_buildah copy $cid ${TESTDIR}/randomfile
  run_buildah copy $cid ${TESTDIR}/other-randomfile
  run_buildah unmount $cid
  run_buildah commit --signature-policy ${TESTSDIR}/policy.json $cid containers-storage:new-image
  run_buildah rm $cid

  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json new-image
  newcid=$output
  run_buildah mount $newcid
  newroot=$output
  test -s $newroot/randomfile
  cmp ${TESTDIR}/randomfile $newroot/randomfile
  test -s $newroot/other-randomfile
  cmp ${TESTDIR}/other-randomfile $newroot/other-randomfile
}

@test "copy-local-subdirectory" {
  mkdir -p ${TESTDIR}/subdir
  createrandom ${TESTDIR}/subdir/randomfile
  createrandom ${TESTDIR}/subdir/other-randomfile

  run_buildah from --signature-policy ${TESTSDIR}/policy.json scratch
  cid=$output
  run_buildah config --workingdir /container-subdir $cid
  run_buildah copy $cid ${TESTDIR}/subdir
  run_buildah mount $cid
  root=$output
  test -s $root/container-subdir/randomfile
  cmp ${TESTDIR}/subdir/randomfile $root/container-subdir/randomfile
  test -s $root/container-subdir/other-randomfile
  cmp ${TESTDIR}/subdir/other-randomfile $root/container-subdir/other-randomfile
  run_buildah copy $cid ${TESTDIR}/subdir /other-subdir
  test -s $root/other-subdir/randomfile
  cmp ${TESTDIR}/subdir/randomfile $root/other-subdir/randomfile
  test -s $root/other-subdir/other-randomfile
  cmp ${TESTDIR}/subdir/other-randomfile $root/other-subdir/other-randomfile
}

@test "copy-local-force-directory" {
  createrandom ${TESTDIR}/randomfile

  run_buildah from --signature-policy ${TESTSDIR}/policy.json scratch
  cid=$output
  run_buildah config --workingdir / $cid
  run_buildah copy $cid ${TESTDIR}/randomfile /randomfile
  run_buildah mount $cid
  root=$output
  test -s $root/randomfile
  cmp ${TESTDIR}/randomfile $root/randomfile
  run_buildah rm $cid

  run_buildah from --signature-policy ${TESTSDIR}/policy.json scratch
  cid=$output
  run_buildah config --workingdir / $cid
  run_buildah copy $cid ${TESTDIR}/randomfile /randomsubdir/
  run_buildah mount $cid
  root=$output
  test -s $root/randomsubdir/randomfile
  cmp ${TESTDIR}/randomfile $root/randomsubdir/randomfile
}

@test "copy-url-mtime" {
  # Create a file with random content and a non-now timestamp (so we can
  # can trust that buildah correctly set mtime on copy)
  createrandom ${TESTDIR}/randomfile
  touch -t 201910310123.45 ${TESTDIR}/randomfile

  run_buildah from --signature-policy ${TESTSDIR}/policy.json scratch
  cid=$output
  run_buildah config --workingdir / $cid
  starthttpd ${TESTDIR}
  run_buildah copy $cid http://0.0.0.0:${HTTP_SERVER_PORT}/randomfile /urlfile
  stophttpd
  run_buildah mount $cid
  root=$output
  test -s $root/urlfile
  cmp ${TESTDIR}/randomfile $root/urlfile

  # Compare timestamps. Display them in human-readable form, so if there's
  # a mismatch it will be shown in the test log.
  mtime_randomfile=$(stat --format %y ${TESTDIR}/randomfile)
  mtime_urlfile=$(stat --format %y $root/urlfile)

  expect_output --from="$mtime_randomfile" "$mtime_urlfile" "mtime[randomfile] == mtime[urlfile]"
}

@test "copy --chown" {
  mkdir -p ${TESTDIR}/subdir
  mkdir -p ${TESTDIR}/other-subdir
  createrandom ${TESTDIR}/subdir/randomfile
  createrandom ${TESTDIR}/subdir/other-randomfile
  createrandom ${TESTDIR}/randomfile
  createrandom ${TESTDIR}/other-subdir/randomfile
  createrandom ${TESTDIR}/other-subdir/other-randomfile

  _prefetch alpine
  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json alpine
  cid=$output
  run_buildah config --workingdir / $cid
  run_buildah copy --chown 1:1 $cid ${TESTDIR}/randomfile
  run_buildah copy --chown root:1 $cid ${TESTDIR}/randomfile /randomfile2
  run_buildah copy --chown nobody $cid ${TESTDIR}/randomfile /randomfile3
  run_buildah copy --chown nobody:root $cid ${TESTDIR}/subdir /subdir
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

  run_buildah copy --chown root:root $cid ${TESTDIR}/other-subdir /subdir
  for i in randomfile other-randomfile ; do
      run_buildah run $cid stat -c "%U:%G" /subdir/$i
      expect_output "root:root" "stat UG /subdir/$i (after chown)"
  done

  # subdir itself will have not been copied (the destination directory was created implicitly), so its permissions should not have changed
  run_buildah run $cid stat -c "%U:%G" /subdir
  expect_output "nobody:root" "stat UG /subdir"
}

@test "copy --chmod" {
  mkdir -p ${TESTDIR}/subdir
  mkdir -p ${TESTDIR}/other-subdir
  createrandom ${TESTDIR}/subdir/randomfile
  createrandom ${TESTDIR}/subdir/other-randomfile
  createrandom ${TESTDIR}/randomfile
  createrandom ${TESTDIR}/other-subdir/randomfile
  createrandom ${TESTDIR}/other-subdir/other-randomfile

  _prefetch alpine
  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json alpine
  cid=$output
  run_buildah config --workingdir / $cid
  run_buildah copy --chmod 777 $cid ${TESTDIR}/randomfile
  run_buildah copy --chmod 700 $cid ${TESTDIR}/randomfile /randomfile2
  run_buildah copy --chmod 755 $cid ${TESTDIR}/randomfile /randomfile3
  run_buildah copy --chmod 660 $cid ${TESTDIR}/subdir /subdir

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

  run_buildah copy --chmod 600 $cid ${TESTDIR}/other-subdir /subdir
  for i in randomfile other-randomfile ; do
      run_buildah run $cid ls -l /subdir/$i
      expect_output --substring rw-------
  done
}

@test "copy-symlink" {
  createrandom ${TESTDIR}/randomfile
  ln -s ${TESTDIR}/randomfile ${TESTDIR}/link-randomfile

  run_buildah from --signature-policy ${TESTSDIR}/policy.json scratch
  cid=$output
  run_buildah mount $cid
  root=$output
  run_buildah config --workingdir / $cid
  run_buildah copy $cid ${TESTDIR}/link-randomfile
  run_buildah unmount $cid
  run_buildah commit --signature-policy ${TESTSDIR}/policy.json $cid containers-storage:new-image
  run_buildah rm $cid

  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json new-image
  newcid=$output
  run_buildah mount $newcid
  newroot=$output
  test -s $newroot/link-randomfile
  test -f $newroot/link-randomfile
  cmp ${TESTDIR}/randomfile $newroot/link-randomfile
}

@test "ignore-socket" {
  createrandom ${TESTDIR}/randomfile
  # This seems to be the least-worst way to create a socket: run and kill nc
  nc -lkU ${TESTDIR}/test.socket &
  nc_pid=$!
  # This should succeed fairly quickly. We test with a timeout in case of
  # failure (likely reason: 'nc' not installed.)
  retries=50
  while ! test -e ${TESTDIR}/test.socket; do
      sleep 0.1
      retries=$((retries - 1))
      if [[ $retries -eq 0 ]]; then
          die "Timed out waiting for ${TESTDIR}/test.socket (is nc installed?)"
      fi
  done
  kill $nc_pid

  run_buildah from --signature-policy ${TESTSDIR}/policy.json scratch
  cid=$output
  run_buildah mount $cid
  root=$output
  run_buildah config --workingdir / $cid
  run_buildah unmount $cid
  run_buildah commit --signature-policy ${TESTSDIR}/policy.json $cid containers-storage:new-image
  run_buildah rm $cid

  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json new-image
  newcid=$output
  run_buildah mount $newcid
  newroot=$output
  test \! -e $newroot/test.socket
}

@test "copy-symlink-archive-suffix" {
  createrandom ${TESTDIR}/randomfile.tar.gz
  ln -s ${TESTDIR}/randomfile.tar.gz ${TESTDIR}/link-randomfile.tar.gz

  run_buildah from --signature-policy ${TESTSDIR}/policy.json scratch
  cid=$output
  run_buildah mount $cid
  root=$output
  run_buildah config --workingdir / $cid
  run_buildah copy $cid ${TESTDIR}/link-randomfile.tar.gz
  run_buildah unmount $cid
  run_buildah commit --signature-policy ${TESTSDIR}/policy.json $cid containers-storage:new-image
  run_buildah rm $cid

  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json new-image
  newcid=$output
  run_buildah mount $newcid
  newroot=$output
  test -s $newroot/link-randomfile.tar.gz
  test -f $newroot/link-randomfile.tar.gz
  cmp ${TESTDIR}/randomfile.tar.gz $newroot/link-randomfile.tar.gz
}

@test "copy-detect-missing-data" {
  _prefetch busybox

  : > ${TESTDIR}/Dockerfile
  echo FROM busybox AS builder                                >> ${TESTDIR}/Dockerfile
  echo FROM scratch                                           >> ${TESTDIR}/Dockerfile
  echo COPY --from=builder /bin/-no-such-file-error- /usr/bin >> ${TESTDIR}/Dockerfile
  run_buildah 125 build-using-dockerfile --signature-policy ${TESTSDIR}/policy.json ${TESTDIR}
  expect_output --substring "no such file or directory"
}

@test "copy --ignore" {
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
  expect_output -- "--ignore options requires that you specify a context dir using --contextdir" "container file list"

  run_buildah copy --contextdir=${mytest} --ignorefile ${mytest}/.ignore $cid ${mytest} /stuff

  run_buildah mount $cid
  mnt=$output
  run find $mnt -printf "%P\n"
  filelist=$(LC_ALL=C sort <<<"$output")
  run_buildah umount $cid
  expect_output --from="$filelist" "$expect" "container file list"
}

@test "copy-quiet" {
  createrandom ${TESTDIR}/randomfile
  _prefetch alpine
  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
  cid=$output
  run_buildah mount $cid
  root=$output
  run_buildah copy --quiet $cid ${TESTDIR}/randomfile /
  expect_output ""
  cmp ${TESTDIR}/randomfile $root/randomfile
  run_buildah umount $cid
  run_buildah rm $cid
}

@test "copy-from-container" {
  _prefetch busybox
  createrandom ${TESTDIR}/randomfile
  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json busybox
  from=$output
  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json busybox
  cid=$output
  run_buildah copy --quiet $from ${TESTDIR}/randomfile /tmp/random
  expect_output ""
  run_buildah copy --quiet --signature-policy ${TESTSDIR}/policy.json --from $from $cid /tmp/random /tmp/random # absolute path
  expect_output ""
  run_buildah copy --quiet --signature-policy ${TESTSDIR}/policy.json --from $from $cid  tmp/random /tmp/random2 # relative path
  expect_output ""
  run_buildah mount $cid
  croot=$output
  cmp ${TESTDIR}/randomfile ${croot}/tmp/random
  cmp ${TESTDIR}/randomfile ${croot}/tmp/random2
}

@test "copy-container-root" {
  _prefetch busybox
  createrandom ${TESTDIR}/randomfile
  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json busybox
  from=$output
  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json busybox
  cid=$output
  run_buildah copy --quiet $from ${TESTDIR}/randomfile /tmp/random
  expect_output ""
  run_buildah copy --quiet --signature-policy ${TESTSDIR}/policy.json --from $from $cid / /tmp/
  expect_output "" || \
    expect_output --substring "copier: file disappeared while reading"
  run_buildah mount $cid
  croot=$output
  cmp ${TESTDIR}/randomfile ${croot}/tmp/tmp/random
}

@test "copy-from-image" {
  _prefetch busybox
  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json busybox
  cid=$output
  run_buildah add --signature-policy ${TESTSDIR}/policy.json --quiet --from ubuntu $cid /etc/passwd /tmp/passwd # should pull the image, absolute path
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
