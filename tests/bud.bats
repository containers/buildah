#!/usr/bin/env bats

load helpers

@test "bud with a path to a Dockerfile (-f) containing a non-directory entry" {
  run_buildah 125 bud -f ${TESTSDIR}/bud/non-directory-in-path/non-directory/Dockerfile
  expect_output --substring "non-directory/Dockerfile: not a directory"
}

@test "bud with --dns* flags" {
  _prefetch alpine
  run_buildah bud --dns-search=example.com --dns=223.5.5.5 --dns-option=use-vc  --signature-policy ${TESTSDIR}/policy.json -f ${TESTSDIR}/bud/dns/Dockerfile  ${TESTSDIR}/bud/dns
  expect_output --substring "search example.com"
  expect_output --substring "nameserver 223.5.5.5"
  expect_output --substring "options use-vc"
}

@test "bud with .dockerignore" {
  _prefetch alpine busybox
  run_buildah 125 bud -t testbud --signature-policy ${TESTSDIR}/policy.json -f ${TESTSDIR}/bud/dockerignore/Dockerfile ${TESTSDIR}/bud/dockerignore
  expect_output --substring 'error building.*"COPY subdir \./".*no such file or directory'

  run_buildah bud -t testbud --signature-policy ${TESTSDIR}/policy.json -f ${TESTSDIR}/bud/dockerignore/Dockerfile.succeed ${TESTSDIR}/bud/dockerignore

  run_buildah from --name myctr testbud

  run_buildah 1 run myctr ls -l test1.txt

  run_buildah run myctr ls -l test2.txt

  run_buildah 1 run myctr ls -l sub1.txt

  run_buildah 1 run myctr ls -l sub2.txt

  run_buildah run myctr ls -l subdir/sub1.txt

  run_buildah 1 run myctr ls -l subdir/sub2.txt
}

@test "bud with .dockerignore - unmatched" {
  # Here .dockerignore contains 'unmatched', which will not match anything.
  # Therefore everything in the subdirectory should be copied into the image.
  #
  # We need to do this from a tmpdir, not the original or distributed
  # bud subdir, because of rpm: as of 2020-04-01 rpmbuild 4.16 alpha
  # on rawhide no longer packages circular symlinks (rpm issue #1159).
  # We used to include these symlinks in git and the rpm; now we need to
  # set them up manually as part of test setup.
  cp -a ${TESTSDIR}/bud/dockerignore2 ${TESTDIR}/dockerignore2

  # Create symlinks, including bad ones
  ln -sf subdir        ${TESTDIR}/dockerignore2/symlink
  ln -sf circular-link ${TESTDIR}/dockerignore2/subdir/circular-link
  ln -sf no-such-file  ${TESTDIR}/dockerignore2/subdir/dangling-link

  # Build, create a container, mount it, and list all files therein
  run_buildah bud -t testbud2 --signature-policy ${TESTSDIR}/policy.json ${TESTDIR}/dockerignore2

  run_buildah from testbud2
  cid=$output

  run_buildah mount $cid
  mnt=$output
  run find $mnt -printf "%P(%l)\n"
  filelist=$(LC_ALL=C sort <<<"$output")
  run_buildah umount $cid

  # Format is: filename, and, in parentheses, symlink target (usually empty)
  # The list below has been painstakingly crafted; please be careful if
  # you need to touch it (e.g. if you add new files/symlinks)
  expect="()
.dockerignore()
Dockerfile()
subdir()
subdir/circular-link(circular-link)
subdir/dangling-link(no-such-file)
subdir/sub1.txt()
subdir/subsubdir()
subdir/subsubdir/subsub1.txt()
symlink(subdir)"

  # If this test ever fails, the 'expect' message will be almost impossible
  # for humans to read -- sorry, I never implemented multi-line comparisons.
  # Should this ever happen, uncomment these two lines and run tests in
  # your own vm; then diff the two files.
  #echo "$filelist" >${TMPDIR}/filelist.actual
  #echo "$expect"   >${TMPDIR}/filelist.expect

  expect_output --from="$filelist" "$expect" "container file list"
}

@test "bud with .dockerignore - 3" {
  run_buildah 125 bud -t testbud3 --signature-policy ${TESTSDIR}/policy.json ${TESTSDIR}/bud/dockerignore3
  expect_output --substring 'error building.*"COPY test1.txt /upload/test1.txt".*no such file or directory'
}

@test "bud-flags-order-verification" {
  run_buildah 125 bud /tmp/tmpdockerfile/ -t blabla
  check_options_flag_err "-t"

  run_buildah 125 bud /tmp/tmpdockerfile/ -q -t blabla
  check_options_flag_err "-q"

  run_buildah 125 bud /tmp/tmpdockerfile/ --force-rm
  check_options_flag_err "--force-rm"

  run_buildah 125 bud /tmp/tmpdockerfile/ --userns=cnt1
  check_options_flag_err "--userns=cnt1"
}

@test "bud with --layers and --no-cache flags" {
  _prefetch alpine
  cp -a ${TESTSDIR}/bud/use-layers ${TESTDIR}/use-layers

  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t test1 ${TESTDIR}/use-layers
  run_buildah images -a
  expect_line_count 8
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t test2 ${TESTDIR}/use-layers
  run_buildah images -a
  expect_line_count 10
  run_buildah inspect --format "{{index .Docker.ContainerConfig.Env 1}}" test1
  expect_output "foo=bar"
  run_buildah inspect --format "{{index .Docker.ContainerConfig.Env 1}}" test2
  expect_output "foo=bar"
  run_buildah inspect --format "{{.Docker.ContainerConfig.ExposedPorts}}" test1
  expect_output "map[8080/tcp:{}]"
  run_buildah inspect --format "{{.Docker.ContainerConfig.ExposedPorts}}" test2
  expect_output "map[8080/tcp:{}]"

  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t test3 -f Dockerfile.2 ${TESTDIR}/use-layers
  run_buildah images -a
  expect_line_count 12

  mkdir -p ${TESTDIR}/use-layers/mount/subdir
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t test4 -f Dockerfile.3 ${TESTDIR}/use-layers
  run_buildah images -a
  expect_line_count 14

  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t test5 -f Dockerfile.3 ${TESTDIR}/use-layers
  run_buildah images -a
  expect_line_count 15

  touch ${TESTDIR}/use-layers/mount/subdir/file.txt
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t test6 -f Dockerfile.3 ${TESTDIR}/use-layers
  run_buildah images -a
  expect_line_count 17

  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --no-cache -t test7 -f Dockerfile.2 ${TESTDIR}/use-layers
  run_buildah images -a
  expect_line_count 18
}

@test "bud with --layers and single and two line Dockerfiles" {
  _prefetch alpine
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t test -f Dockerfile.5 ${TESTSDIR}/bud/use-layers
  run_buildah images -a
  expect_line_count 3

  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t test1 -f Dockerfile.6 ${TESTSDIR}/bud/use-layers
  run_buildah images -a
  expect_line_count 4
}

@test "bud with --layers, multistage, and COPY with --from" {
  _prefetch alpine
  cp -a ${TESTSDIR}/bud/use-layers ${TESTDIR}/use-layers

  mkdir -p ${TESTDIR}/use-layers/uuid
  uuidgen > ${TESTDIR}/use-layers/uuid/data
  mkdir -p ${TESTDIR}/use-layers/date
  date > ${TESTDIR}/use-layers/date/data

  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t test1 -f Dockerfile.multistage-copy ${TESTDIR}/use-layers
  run_buildah images -a
  expect_line_count 6
  # The second time through, the layers should all get reused.
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t test1 -f Dockerfile.multistage-copy ${TESTDIR}/use-layers
  run_buildah images -a
  expect_line_count 6
  # The third time through, the layers should all get reused, but we'll have a new line of output for the new name.

  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t test2 -f Dockerfile.multistage-copy ${TESTDIR}/use-layers
  run_buildah images -a
  expect_line_count 7

  # Both interim images will be different, and all of the layers in the final image will be different.
  uuidgen > ${TESTDIR}/use-layers/uuid/data
  date > ${TESTDIR}/use-layers/date/data
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t test3 -f Dockerfile.multistage-copy ${TESTDIR}/use-layers
  run_buildah images -a
  expect_line_count 11
  # No leftover containers, just the header line.
  run_buildah containers
  expect_line_count 1

  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json test3
  ctr=$output
  run_buildah mount ${ctr}
  mnt=$output
  test -e $mnt/uuid
  test -e $mnt/date

  # Layers won't get reused because this build won't use caching.
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t test4 -f Dockerfile.multistage-copy ${TESTDIR}/use-layers
  run_buildah images -a
  expect_line_count 12
}

@test "bud-multistage-partial-cache" {
  _prefetch alpine
  target=foo
  # build the first stage
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -f ${TESTSDIR}/bud/cache-stages/Dockerfile.1 ${TESTSDIR}/bud/cache-stages
  # expect alpine + 1 image record for the first stage
  run_buildah images -a
  expect_line_count 3
  # build the second stage, itself not cached, when the first stage is found in the cache
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -f ${TESTSDIR}/bud/cache-stages/Dockerfile.2 -t ${target} ${TESTSDIR}/bud/cache-stages
  # expect alpine + 1 image record for the first stage, then two more image records for the second stage
  run_buildah images -a
  expect_line_count 5
}

@test "bud-multistage-copy-final-slash" {
  _prefetch busybox
  target=foo
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/dest-final-slash
  run_buildah from --signature-policy ${TESTSDIR}/policy.json ${target}
  cid="$output"
  run_buildah run ${cid} /test/ls -lR /test/ls
}

@test "bud-multistage-reused" {
  _prefetch alpine busybox
  target=foo
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/multi-stage-builds/Dockerfile.reused ${TESTSDIR}/bud/multi-stage-builds
  run_buildah from --signature-policy ${TESTSDIR}/policy.json ${target}
  run_buildah rmi -f ${target}
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} --layers -f ${TESTSDIR}/bud/multi-stage-builds/Dockerfile.reused ${TESTSDIR}/bud/multi-stage-builds
  run_buildah from --signature-policy ${TESTSDIR}/policy.json ${target}
}

@test "bud-multistage-cache" {
  _prefetch alpine busybox
  target=foo
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/multi-stage-builds/Dockerfile.extended ${TESTSDIR}/bud/multi-stage-builds
  run_buildah from --signature-policy ${TESTSDIR}/policy.json ${target}
  cid="$output"
  run_buildah mount "$cid"
  root="$output"
  # cache should have used this one
  test -r "$root"/tmp/preCommit
  # cache should not have used this one
  ! test -r "$root"/tmp/postCommit
}

@test "bud with --layers and symlink file" {
  _prefetch alpine
  cp -a ${TESTSDIR}/bud/use-layers ${TESTDIR}/use-layers
  echo 'echo "Hello World!"' > ${TESTDIR}/use-layers/hello.sh
  ln -s hello.sh ${TESTDIR}/use-layers/hello_world.sh
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t test -f Dockerfile.4 ${TESTDIR}/use-layers
  run_buildah images -a
  expect_line_count 4

  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t test1 -f Dockerfile.4 ${TESTDIR}/use-layers
  run_buildah images -a
  expect_line_count 5

  echo 'echo "Hello Cache!"' > ${TESTDIR}/use-layers/hello.sh
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t test2 -f Dockerfile.4 ${TESTDIR}/use-layers
  run_buildah images -a
  expect_line_count 7
}

@test "bud with --layers and dangling symlink" {
  _prefetch alpine
  cp -a ${TESTSDIR}/bud/use-layers ${TESTDIR}/use-layers
  mkdir ${TESTDIR}/use-layers/blah
  ln -s ${TESTSDIR}/policy.json ${TESTDIR}/use-layers/blah/policy.json

  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t test -f Dockerfile.dangling-symlink ${TESTDIR}/use-layers
  run_buildah images -a
  expect_line_count 3

  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t test1 -f Dockerfile.dangling-symlink ${TESTDIR}/use-layers
  run_buildah images -a
  expect_line_count 4

  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json test
  cid=$output
  run_buildah run $cid ls /tmp
  expect_output "policy.json"
}

@test "bud with --layers and --build-args" {
  _prefetch alpine
  # base plus 3, plus the header line
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --build-arg=user=0 --layers -t test -f Dockerfile.build-args ${TESTSDIR}/bud/use-layers
  run_buildah images -a
  expect_line_count 5

  # two more, starting at the "echo $user" instruction
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --build-arg=user=1 --layers -t test1 -f Dockerfile.build-args ${TESTSDIR}/bud/use-layers
  run_buildah images -a
  expect_line_count 7

  # one more, because we added a new name to the same image
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --build-arg=user=1 --layers -t test2 -f Dockerfile.build-args ${TESTSDIR}/bud/use-layers
  run_buildah images -a
  expect_line_count 8

  # two more, starting at the "echo $user" instruction
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t test3 -f Dockerfile.build-args ${TESTSDIR}/bud/use-layers
  run_buildah images -a
  expect_line_count 10
}

@test "bud with --rm flag" {
  _prefetch alpine
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t test1 ${TESTSDIR}/bud/use-layers
  run_buildah containers
  expect_line_count 1

  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --rm=false --layers -t test2 ${TESTSDIR}/bud/use-layers
  run_buildah containers
  expect_line_count 7
}

@test "bud with --force-rm flag" {
  _prefetch alpine
  run_buildah 125 bud --signature-policy ${TESTSDIR}/policy.json --force-rm --layers -t test1 -f Dockerfile.fail-case ${TESTSDIR}/bud/use-layers
  run_buildah containers
  expect_line_count 1

  run_buildah 125 bud --signature-policy ${TESTSDIR}/policy.json --layers -t test2 -f Dockerfile.fail-case ${TESTSDIR}/bud/use-layers
  run_buildah containers
  expect_line_count 2
}

@test "bud --layers with non-existent/down registry" {
  _prefetch alpine
  run_buildah 125 bud --signature-policy ${TESTSDIR}/policy.json --force-rm --layers -t test1 -f Dockerfile.non-existent-registry ${TESTSDIR}/bud/use-layers
  expect_output --substring "no such host"
}

@test "bud from base image should have base image ENV also" {
  _prefetch alpine
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t test -f Dockerfile.check-env ${TESTSDIR}/bud/env
  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json test
  cid=$output
  run_buildah config --env random=hello,goodbye ${cid}
  run_buildah commit --signature-policy ${TESTSDIR}/policy.json ${cid} test1
  run_buildah inspect --format '{{index .Docker.ContainerConfig.Env 1}}' test1
  expect_output "foo=bar"
  run_buildah inspect --format '{{index .Docker.ContainerConfig.Env 2}}' test1
  expect_output "random=hello,goodbye"
}

@test "bud-from-scratch" {
  target=scratch-image
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/from-scratch
  run_buildah from ${target}
  expect_output "${target}-working-container"
}

@test "bud-from-scratch-iid" {
  target=scratch-image
  run_buildah bud --iidfile ${TESTDIR}/output.iid --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/from-scratch
  iid=$(cat ${TESTDIR}/output.iid)
  run_buildah from ${iid}
  expect_output "${target}-working-container"
}

@test "bud-from-scratch-label" {
  run_buildah --version
  local -a output_fields=($output)
  buildah_version=${output_fields[2]}
  want_output='map["io.buildah.version":"'$buildah_version'" "test":"label"]'

  target=scratch-image
  run_buildah bud --label "test=label" --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/from-scratch
  run_buildah inspect --format '{{printf "%q" .Docker.Config.Labels}}' ${target}
  expect_output "$want_output"
}

@test "bud-from-scratch-annotation" {
  target=scratch-image
  run_buildah bud --annotation "test=annotation1,annotation2=z" --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/from-scratch
  run_buildah inspect --format '{{printf "%q" .ImageAnnotations}}' ${target}
  expect_output 'map["test":"annotation1,annotation2=z"]'
}

@test "bud-from-scratch-layers" {
  target=scratch-image
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -f  ${TESTSDIR}/bud/from-scratch/Dockerfile2 -t ${target} ${TESTSDIR}/bud/from-scratch
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -f  ${TESTSDIR}/bud/from-scratch/Dockerfile2 -t ${target} ${TESTSDIR}/bud/from-scratch
  run_buildah from --quiet ${target}
  cid=$output
  run_buildah images
  expect_line_count 3
  run_buildah rm ${cid}
  expect_line_count 1
}

@test "bud-from-multiple-files-one-from" {
  target=scratch-image
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/from-multiple-files/Dockerfile1.scratch -f ${TESTSDIR}/bud/from-multiple-files/Dockerfile2.nofrom ${TESTSDIR}/bud/from-multiple-files
  run_buildah from --quiet ${target}
  cid=$output
  run_buildah mount ${cid}
  root=$output
  cmp $root/Dockerfile1 ${TESTSDIR}/bud/from-multiple-files/Dockerfile1.scratch
  cmp $root/Dockerfile2.nofrom ${TESTSDIR}/bud/from-multiple-files/Dockerfile2.nofrom
  test ! -s $root/etc/passwd
  run_buildah rm ${cid}
  run_buildah rmi -a

  _prefetch alpine
  target=alpine-image
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f Dockerfile1.alpine -f Dockerfile2.nofrom ${TESTSDIR}/bud/from-multiple-files
  run_buildah from --quiet ${target}
  cid=$output
  run_buildah mount ${cid}
  root=$output
  cmp $root/Dockerfile1 ${TESTSDIR}/bud/from-multiple-files/Dockerfile1.alpine
  cmp $root/Dockerfile2.nofrom ${TESTSDIR}/bud/from-multiple-files/Dockerfile2.nofrom
  test -s $root/etc/passwd
}

@test "bud-from-multiple-files-two-froms" {
  _prefetch alpine
  target=scratch-image
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f Dockerfile1.scratch -f Dockerfile2.withfrom ${TESTSDIR}/bud/from-multiple-files
  run_buildah from --quiet ${target}
  cid=$output
  run_buildah mount ${cid}
  root=$output
  test ! -s $root/Dockerfile1
  cmp $root/Dockerfile2.withfrom ${TESTSDIR}/bud/from-multiple-files/Dockerfile2.withfrom
  test -s $root/etc/passwd
  run_buildah rm ${cid}
  run_buildah rmi -a

  _prefetch alpine
  target=alpine-image
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f Dockerfile1.alpine -f Dockerfile2.withfrom ${TESTSDIR}/bud/from-multiple-files
  run_buildah from --quiet ${target}
  cid=$output
  run_buildah mount ${cid}
  root=$output
  test ! -s $root/Dockerfile1
  cmp $root/Dockerfile2.withfrom ${TESTSDIR}/bud/from-multiple-files/Dockerfile2.withfrom
  test -s $root/etc/passwd
}

@test "bud-multi-stage-builds" {
  _prefetch alpine
  target=multi-stage-index
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/multi-stage-builds/Dockerfile.index ${TESTSDIR}/bud/multi-stage-builds
  run_buildah from --quiet ${target}
  cid=$output
  run_buildah mount ${cid}
  root=$output
  cmp $root/Dockerfile.index ${TESTSDIR}/bud/multi-stage-builds/Dockerfile.index
  test -s $root/etc/passwd
  run_buildah rm ${cid}
  run_buildah rmi -a

  _prefetch alpine
  target=multi-stage-name
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f Dockerfile.name ${TESTSDIR}/bud/multi-stage-builds
  run_buildah from --quiet ${target}
  cid=$output
  run_buildah mount ${cid}
  root=$output
  cmp $root/Dockerfile.name ${TESTSDIR}/bud/multi-stage-builds/Dockerfile.name
  test ! -s $root/etc/passwd
  run_buildah rm ${cid}
  run_buildah rmi -a

  target=multi-stage-mixed
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/multi-stage-builds/Dockerfile.mixed ${TESTSDIR}/bud/multi-stage-builds
  run_buildah from --quiet ${target}
  cid=$output
  run_buildah mount ${cid}
  root=$output
  cmp $root/Dockerfile.name ${TESTSDIR}/bud/multi-stage-builds/Dockerfile.name
  cmp $root/Dockerfile.index ${TESTSDIR}/bud/multi-stage-builds/Dockerfile.index
  cmp $root/Dockerfile.mixed ${TESTSDIR}/bud/multi-stage-builds/Dockerfile.mixed
}

@test "bud-multi-stage-builds-small-as" {
  _prefetch alpine
  target=multi-stage-index
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/multi-stage-builds-small-as/Dockerfile.index ${TESTSDIR}/bud/multi-stage-builds-small-as
  run_buildah from --quiet ${target}
  cid=$output
  run_buildah mount ${cid}
  root=$output
  cmp $root/Dockerfile.index ${TESTSDIR}/bud/multi-stage-builds-small-as/Dockerfile.index
  test -s $root/etc/passwd
  run_buildah rm ${cid}
  run_buildah rmi -a

  _prefetch alpine
  target=multi-stage-name
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f Dockerfile.name ${TESTSDIR}/bud/multi-stage-builds-small-as
  run_buildah from --quiet ${target}
  cid=$output
  run_buildah mount ${cid}
  root=$output
  cmp $root/Dockerfile.name ${TESTSDIR}/bud/multi-stage-builds-small-as/Dockerfile.name
  test ! -s $root/etc/passwd
  run_buildah rm ${cid}
  run_buildah rmi -a

  target=multi-stage-mixed
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/multi-stage-builds-small-as/Dockerfile.mixed ${TESTSDIR}/bud/multi-stage-builds-small-as
  run_buildah from --quiet ${target}
  cid=$output
  run_buildah mount ${cid}
  root=$output
  cmp $root/Dockerfile.name ${TESTSDIR}/bud/multi-stage-builds-small-as/Dockerfile.name
  cmp $root/Dockerfile.index ${TESTSDIR}/bud/multi-stage-builds-small-as/Dockerfile.index
  cmp $root/Dockerfile.mixed ${TESTSDIR}/bud/multi-stage-builds-small-as/Dockerfile.mixed
}

@test "bud-preserve-subvolumes" {
  # This Dockerfile needs us to be able to handle a working RUN instruction.
  skip_if_no_runtime

  _prefetch alpine
  target=volume-image
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/preserve-volumes
  run_buildah from --quiet ${target}
  cid=$output
  run_buildah mount ${cid}
  root=$output
  test -s $root/vol/subvol/subsubvol/subsubvolfile
  test ! -s $root/vol/subvol/subvolfile
  test -s $root/vol/volfile
  test -s $root/vol/Dockerfile
  test -s $root/vol/Dockerfile2
  test ! -s $root/vol/anothervolfile
}

# Helper function for several of the tests which pull from http.
#
#  Usage:  _test_http  SUBDIRECTORY  URL_PATH  [EXTRA ARGS]
#
#     SUBDIRECTORY   is a subdirectory path under the 'buds' subdirectory.
#                    This will be the argument to starthttpd(), i.e. where
#                    the httpd will serve files.
#
#     URL_PATH       is the path requested by buildah from the http server,
#                    probably 'Dockerfile' or 'context.tar'
#
#     [EXTRA ARGS]   if present, will be passed to buildah on the 'bud'
#                    command line; it is intended for '-f subdir/Dockerfile'.
#
function _test_http() {
  local testdir=$1; shift;        # in: subdirectory under bud/
  local urlpath=$1; shift;        # in: path to request from localhost

  starthttpd "${TESTSDIR}/bud/$testdir"
  target=scratch-image
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json \
              -t ${target} \
              "$@"         \
              http://0.0.0.0:${HTTP_SERVER_PORT}/$urlpath
  stophttpd
  run_buildah from ${target}
}

@test "bud-http-Dockerfile" {
  _test_http from-scratch Dockerfile
}

@test "bud-http-context-with-Dockerfile" {
  _test_http http-context context.tar
}

@test "bud-http-context-dir-with-Dockerfile" {
  _test_http http-context-subdir context.tar -f context/Dockerfile
}

@test "bud-git-context" {
  # We need git and ssh to be around to handle cloning a repository.
  if ! which git ; then
    skip "no git in PATH"
  fi
  if ! which ssh ; then
    skip "no ssh in PATH"
  fi
  target=giturl-image
  # Any repo should do, but this one is small and is FROM: scratch.
  gitrepo=git://github.com/projectatomic/nulecule-library
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} "${gitrepo}"
  run_buildah from ${target}
}

@test "bud-github-context" {
  target=github-image
  # Any repo should do, but this one is small and is FROM: scratch.
  gitrepo=github.com/projectatomic/nulecule-library
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} "${gitrepo}"
  run_buildah from ${target}
}

@test "bud-additional-tags" {
  target=scratch-image
  target2=another-scratch-image
  target3=so-many-scratch-images
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -t docker.io/${target2} -t ${target3} ${TESTSDIR}/bud/from-scratch
  run_buildah images
  run_buildah from --quiet ${target}
  cid=$output
  run_buildah rm ${cid}
  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json library/${target2}
  cid=$output
  run_buildah rm ${cid}
  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json ${target3}:latest
  run_buildah rm $output

  run_buildah rmi $target3 $target2 $target
  expect_line_count 4
  for i in 0 1 2;do
      expect_output --substring --from="${lines[$i]}" "untagged: "
  done
  expect_output --substring --from="${lines[3]}" '^[0-9a-f]{64}$'
}

@test "bud-additional-tags-cached" {
  _prefetch busybox
  target=tagged-image
  target2=another-tagged-image
  target3=yet-another-tagged-image
  target4=still-another-tagged-image
  run_buildah bud --layers --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/addtl-tags
  run_buildah bud --layers --signature-policy ${TESTSDIR}/policy.json -t ${target2} -t ${target3} -t ${target4} ${TESTSDIR}/bud/addtl-tags
  run_buildah inspect -f '{{.FromImageID}}' busybox
  busyboxid="$output"
  run_buildah inspect -f '{{.FromImageID}}' ${target}
  targetid="$output"
  [ "$targetid" != "$busyboxid" ]
  run_buildah inspect -f '{{.FromImageID}}' ${target2}
  expect_output "$targetid" "target2 -> .FromImageID"
  run_buildah inspect -f '{{.FromImageID}}' ${target3}
  expect_output "$targetid" "target3 -> .FromImageID"
  run_buildah inspect -f '{{.FromImageID}}' ${target4}
  expect_output "$targetid" "target4 -> .FromImageID"
}

@test "bud-volume-perms" {
  # This Dockerfile needs us to be able to handle a working RUN instruction.
  skip_if_no_runtime

  _prefetch alpine
  target=volume-image
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/volume-perms
  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json ${target}
  cid=$output
  run_buildah mount ${cid}
  root=$output
  test ! -s $root/vol/subvol/subvolfile
  run stat -c %f $root/vol/subvol
  [ "$status" -eq 0 ]
  expect_output "41ed" "stat($root/vol/subvol) [0x41ed = 040755]"
}

@test "bud-volume-ownership" {
  # This Dockerfile needs us to be able to handle a working RUN instruction.
  skip_if_no_runtime

  _prefetch alpine
  target=volume-image
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/volume-ownership
  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json ${target}
  cid=$output
  run_buildah run $cid stat -c "%U %G" /vol/subvol
  expect_output "testuser testgroup"
}

@test "bud-from-glob" {
  _prefetch alpine
  target=alpine-image
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f Dockerfile2.glob ${TESTSDIR}/bud/from-multiple-files
  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json ${target}
  cid=$output
  run_buildah mount ${cid}
  root=$output
  cmp $root/Dockerfile1.alpine ${TESTSDIR}/bud/from-multiple-files/Dockerfile1.alpine
  cmp $root/Dockerfile2.withfrom ${TESTSDIR}/bud/from-multiple-files/Dockerfile2.withfrom
}

@test "bud-maintainer" {
  _prefetch alpine
  target=alpine-image
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/maintainer
  run_buildah inspect --type=image --format '{{.Docker.Author}}' ${target}
  expect_output "kilroy"
  run_buildah inspect --type=image --format '{{.OCIv1.Author}}' ${target}
  expect_output "kilroy"
}

@test "bud-unrecognized-instruction" {
  _prefetch alpine
  target=alpine-image
  run_buildah 125 bud --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/unrecognized
  expect_output --substring "BOGUS"
}

@test "bud-shell" {
  _prefetch alpine
  target=alpine-image
  run_buildah bud --format docker --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/shell
  run_buildah inspect --type=image --format '{{printf "%q" .Docker.Config.Shell}}' ${target}
  expect_output '["/bin/sh" "-c"]' ".Docker.Config.Shell (original)"
  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json ${target}
  ctr=$output
  run_buildah config --shell "/bin/bash -c" ${ctr}
  run_buildah inspect --type=container --format '{{printf "%q" .Docker.Config.Shell}}' ${ctr}
  expect_output '["/bin/bash" "-c"]' ".Docker.Config.Shell (changed)"
}

@test "bud-shell during build in Docker format" {
  _prefetch alpine
  target=alpine-image
  run_buildah bud --format docker --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/shell/Dockerfile.build-shell-default ${TESTSDIR}/bud/shell
  expect_output --substring "SHELL=/bin/sh"
}

@test "bud-shell during build in OCI format" {
  _prefetch alpine
  target=alpine-image
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/shell/Dockerfile.build-shell-default ${TESTSDIR}/bud/shell
  expect_output --substring "SHELL=/bin/sh"
}

@test "bud-shell changed during build in Docker format" {
  _prefetch ubuntu
  target=ubuntu-image
  run_buildah bud --format docker --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/shell/Dockerfile.build-shell-custom ${TESTSDIR}/bud/shell
  expect_output --substring "SHELL=/bin/bash"
}

@test "bud-shell changed during build in OCI format" {
  _prefetch ubuntu
  target=ubuntu-image
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/shell/Dockerfile.build-shell-custom ${TESTSDIR}/bud/shell
  expect_output --substring "SHELL=/bin/sh"
}

@test "bud with symlinks" {
  _prefetch alpine
  target=alpine-image
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/symlink
  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json ${target}
  cid=$output
  run_buildah mount ${cid}
  root=$output
  run ls $root/data/log
  [ "$status" -eq 0 ]
  expect_output --substring "test"     "ls \$root/data/log"
  expect_output --substring "blah.txt" "ls \$root/data/log"

  run ls -al $root
  [ "$status" -eq 0 ]
  expect_output --substring "test-log -> /data/log" "ls -l \$root/data/log"
  expect_output --substring "blah -> /test-log"     "ls -l \$root/data/log"
}

@test "bud with symlinks to relative path" {
  _prefetch alpine
  target=alpine-image
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f Dockerfile.relative-symlink ${TESTSDIR}/bud/symlink
  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json ${target}
  cid=$output
  run_buildah mount ${cid}
  root=$output
  run ls $root/log
  [ "$status" -eq 0 ]
  expect_output --substring "test" "ls \$root/log"

  run ls -al $root
  [ "$status" -eq 0 ]
  expect_output --substring "test-log -> ../log" "ls -l \$root/log"
  test -r $root/var/data/empty
}

@test "bud with multiple symlinks in a path" {
  _prefetch alpine
  target=alpine-image
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/symlink/Dockerfile.multiple-symlinks ${TESTSDIR}/bud/symlink
  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json ${target}
  cid=$output
  run_buildah mount ${cid}
  root=$output
  run ls $root/data/log
  [ "$status" -eq 0 ]
  expect_output --substring "bin"      "ls \$root/data/log"
  expect_output --substring "blah.txt" "ls \$root/data/log"

  run ls -al $root/myuser
  [ "$status" -eq 0 ]
  expect_output --substring "log -> /test" "ls -al \$root/myuser"

  run ls -al $root/test
  [ "$status" -eq 0 ]
  expect_output --substring "bar -> /test-log" "ls -al \$root/test"

  run ls -al $root/test-log
  [ "$status" -eq 0 ]
  expect_output --substring "foo -> /data/log" "ls -al \$root/test-log"
}

@test "bud with multiple symlink pointing to itself" {
  _prefetch alpine
  target=alpine-image
  run_buildah 1 bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/symlink/Dockerfile.symlink-points-to-itself ${TESTSDIR}/bud/symlink
}

@test "bud multi-stage with symlink to absolute path" {
  _prefetch ubuntu
  target=ubuntu-image
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f Dockerfile.absolute-symlink ${TESTSDIR}/bud/symlink
  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json ${target}
  cid=$output
  run_buildah mount ${cid}
  root=$output
  run ls $root/bin
  [ "$status" -eq 0 ]
  expect_output --substring "myexe" "ls \$root/bin"

  run cat $root/bin/myexe
  [ "$status" -eq 0 ]
  expect_output "symlink-test" "cat \$root/bin/myexe"
}

@test "bud multi-stage with dir symlink to absolute path" {
  _prefetch ubuntu
  target=ubuntu-image
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f Dockerfile.absolute-dir-symlink ${TESTSDIR}/bud/symlink
  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json ${target}
  cid=$output
  run_buildah mount ${cid}
  root=$output
  run ls $root/data
  [ "$status" -eq 0 ]
  expect_output --substring "myexe" "ls \$root/data"
}

@test "bud with ENTRYPOINT and RUN" {
  _prefetch alpine
  target=alpine-image
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f Dockerfile.entrypoint-run ${TESTSDIR}/bud/run-scenarios
  expect_output --substring "unique.test.string"
  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json ${target}
}

@test "bud with ENTRYPOINT and empty RUN" {
  _prefetch alpine
  target=alpine-image
  run_buildah 2 bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f Dockerfile.entrypoint-empty-run ${TESTSDIR}/bud/run-scenarios
  expect_output --substring "error building at STEP"
}

@test "bud with CMD and RUN" {
  _prefetch alpine
  target=alpine-image
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/run-scenarios/Dockerfile.cmd-run ${TESTSDIR}/bud/run-scenarios
  expect_output --substring "unique.test.string"
  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json ${target}
}

@test "bud with CMD and empty RUN" {
  _prefetch alpine
  target=alpine-image
  run_buildah 2 bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f Dockerfile.cmd-empty-run ${TESTSDIR}/bud/run-scenarios
  expect_output --substring "error building at STEP"
}

@test "bud with ENTRYPOINT, CMD and RUN" {
  _prefetch alpine
  target=alpine-image
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/run-scenarios/Dockerfile.entrypoint-cmd-run ${TESTSDIR}/bud/run-scenarios
  expect_output --substring "unique.test.string"
  run_buildah from --signature-policy ${TESTSDIR}/policy.json ${target}
}

@test "bud with ENTRYPOINT, CMD and empty RUN" {
  _prefetch alpine
  target=alpine-image
  run_buildah 2 bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/run-scenarios/Dockerfile.entrypoint-cmd-empty-run ${TESTSDIR}/bud/run-scenarios
  expect_output --substring "error building at STEP"
}

# Determines if a variable set with ENV is available to following commands in the Dockerfile
@test "bud access ENV variable defined in same source file" {
  _prefetch alpine
  target=env-image
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/env/Dockerfile.env-same-file ${TESTSDIR}/bud/env
  expect_output --substring ":unique.test.string:"
  run_buildah from --signature-policy ${TESTSDIR}/policy.json ${target}
}

# Determines if a variable set with ENV in an image is available to commands in downstream Dockerfile
@test "bud access ENV variable defined in FROM image" {
  _prefetch alpine
  from_target=env-from-image
  target=env-image
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${from_target} -f ${TESTSDIR}/bud/env/Dockerfile.env-same-file ${TESTSDIR}/bud/env
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/env/Dockerfile.env-from-image ${TESTSDIR}/bud/env
  expect_output --substring "@unique.test.string@"
  run_buildah from --quiet ${from_target}
  from_cid=$output
  run_buildah from ${target}
}

@test "bud ENV preserves special characters after commit" {
  _prefetch ubuntu
  from_target=special-chars
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${from_target} -f ${TESTSDIR}/bud/env/Dockerfile.special-chars ${TESTSDIR}/bud/env
  run_buildah from --quiet ${from_target}
  cid=$output
  run_buildah run ${cid} env
  expect_output --substring "LIB=\\$\(PREFIX\)/lib"
}

@test "bud with Dockerfile from valid URL" {
  target=url-image
  url=https://raw.githubusercontent.com/containers/buildah/master/tests/bud/from-scratch/Dockerfile
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} ${url}
  run_buildah from ${target}
}

@test "bud with Dockerfile from invalid URL" {
  target=url-image
  url=https://raw.githubusercontent.com/containers/buildah/master/tests/bud/from-scratch/Dockerfile.bogus
  run_buildah 125 bud --signature-policy ${TESTSDIR}/policy.json -t ${target} ${url}
}

# When provided with a -f flag and directory, buildah will look for the alternate Dockerfile name in the supplied directory
@test "bud with -f flag, alternate Dockerfile name" {
  target=fileflag-image
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f Dockerfile.noop-flags ${TESTSDIR}/bud/run-scenarios
  run_buildah from ${target}
}

# Following flags are configured to result in noop but should not affect buildiah bud behavior
@test "bud with --cache-from noop flag" {
  target=noop-image
  run_buildah bud --cache-from=invalidimage --signature-policy ${TESTSDIR}/policy.json -t ${target} -f Dockerfile.noop-flags ${TESTSDIR}/bud/run-scenarios
  run_buildah from ${target}
}

@test "bud with --compress noop flag" {
  target=noop-image
  run_buildah bud --compress --signature-policy ${TESTSDIR}/policy.json -t ${target} -f Dockerfile.noop-flags ${TESTSDIR}/bud/run-scenarios
  run_buildah from ${target}
}

@test "bud with --cpu-shares flag, no argument" {
  target=bud-flag
  run_buildah 125 bud --cpu-shares --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/from-scratch/Dockerfile ${TESTSDIR}/bud/from-scratch
}

@test "bud with --cpu-shares flag, invalid argument" {
  target=bud-flag
  run_buildah 125 bud --cpu-shares bogus --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/from-scratch/Dockerfile ${TESTSDIR}/bud/from-scratch
  expect_output --substring "invalid argument \"bogus\" for "
}

@test "bud with --cpu-shares flag, valid argument" {
  target=bud-flag
  run_buildah bud --cpu-shares 2 --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/from-scratch/Dockerfile ${TESTSDIR}/bud/from-scratch
  run_buildah from ${target}
}

@test "bud with --cpu-shares short flag (-c), no argument" {
  target=bud-flag
  run_buildah 125 bud -c --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/from-scratch/Dockerfile ${TESTSDIR}/bud/from-scratch
}

@test "bud with --cpu-shares short flag (-c), invalid argument" {
  target=bud-flag
  run_buildah 125 bud -c bogus --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/from-scratch/Dockerfile ${TESTSDIR}/bud/from-scratch
  expect_output --substring "invalid argument \"bogus\" for "
}

@test "bud with --cpu-shares short flag (-c), valid argument" {
  target=bud-flag
  run_buildah bud -c 2 --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/from-scratch
  run_buildah from ${target}
}

@test "bud-onbuild" {
  _prefetch alpine
  target=onbuild
  run_buildah bud --format docker --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/onbuild
  run_buildah inspect --format '{{printf "%q" .Docker.Config.OnBuild}}' ${target}
  expect_output '["RUN touch /onbuild1" "RUN touch /onbuild2"]'
  run_buildah from --quiet ${target}
  cid=${lines[0]}
  run_buildah mount ${cid}
  root=$output

  test -e ${root}/onbuild1
  test -e ${root}/onbuild2

  run_buildah umount ${cid}
  run_buildah rm ${cid}

  target=onbuild-image2
  run_buildah bud --format docker --signature-policy ${TESTSDIR}/policy.json -t ${target} -f Dockerfile1 ${TESTSDIR}/bud/onbuild
  run_buildah inspect --format '{{printf "%q" .Docker.Config.OnBuild}}' ${target}
  expect_output '["RUN touch /onbuild3"]'
  run_buildah from --quiet ${target}
  cid=${lines[0]}
  run_buildah mount ${cid}
  root=$output

  test -e ${root}/onbuild1
  test -e ${root}/onbuild2
  test -e ${root}/onbuild3
  run_buildah umount ${cid}

  run_buildah config --onbuild "RUN touch /onbuild4" ${cid}

  target=onbuild-image3
  run_buildah commit --signature-policy ${TESTSDIR}/policy.json --format docker ${cid} ${target}
  run_buildah inspect --format '{{printf "%q" .Docker.Config.OnBuild}}' ${target}
  expect_output '["RUN touch /onbuild4"]'
}

@test "bud-onbuild-layers" {
  _prefetch alpine
  target=onbuild
  run_buildah bud --format docker --signature-policy ${TESTSDIR}/policy.json --layers -t ${target} -f Dockerfile2 ${TESTSDIR}/bud/onbuild
  run_buildah inspect --format '{{printf "%q" .Docker.Config.OnBuild}}' ${target}
  expect_output '["RUN touch /onbuild1" "RUN touch /onbuild2"]'
}

@test "bud-logfile" {
  _prefetch alpine
  rm -f ${TESTDIR}/logfile
  run_buildah bud --logfile ${TESTDIR}/logfile --signature-policy ${TESTSDIR}/policy.json ${TESTSDIR}/bud/preserve-volumes
  test -s ${TESTDIR}/logfile
}

@test "bud with ARGS" {
  _prefetch alpine
  target=alpine-image
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f Dockerfile.args ${TESTSDIR}/bud/run-scenarios
  expect_output --substring "arg_value"
}

@test "bud with unused ARGS" {
  _prefetch alpine
  target=alpine-image
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f Dockerfile.multi-args --build-arg USED_ARG=USED_VALUE ${TESTSDIR}/bud/run-scenarios
  expect_output --substring "USED_VALUE"
  [[ ! "$output" =~ "one or more build args were not consumed: [UNUSED_ARG]" ]]
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f Dockerfile.multi-args --build-arg USED_ARG=USED_VALUE --build-arg UNUSED_ARG=whaaaat ${TESTSDIR}/bud/run-scenarios
  expect_output --substring "USED_VALUE"
  expect_output --substring "one or more build args were not consumed: \[UNUSED_ARG\]"
}

@test "bud with multi-value ARGS" {
  _prefetch alpine
  target=alpine-image
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f Dockerfile.multi-args --build-arg USED_ARG=plugin1,plugin2,plugin3 ${TESTSDIR}/bud/run-scenarios
  expect_output --substring "plugin1,plugin2,plugin3"
   if [[ "$output" =~ "one or more build args were not consumed" ]]; then
      expect_output "[not expecting to see 'one or more build args were not consumed']"
  fi
}

@test "bud-from-stdin" {
  target=scratch-image
  cat ${TESTSDIR}/bud/from-multiple-files/Dockerfile1.scratch | run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f - ${TESTSDIR}/bud/from-multiple-files
  run_buildah from --quiet ${target}
  cid=$output
  run_buildah mount ${cid}
  root=$output
  test -s $root/Dockerfile1
}

@test "bud with preprocessor" {
  _prefetch alpine
  target=alpine-image
  run_buildah bud -q --signature-policy ${TESTSDIR}/policy.json -t ${target} -f Decomposed.in ${TESTSDIR}/bud/preprocess
}

@test "bud with preprocessor error" {
  target=alpine-image
  run_buildah 1 bud -q --signature-policy ${TESTSDIR}/policy.json -t ${target} -f Error.in ${TESTSDIR}/bud/preprocess
}

@test "bud-with-rejected-name" {
  target=ThisNameShouldBeRejected
  run_buildah 125 bud -q --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/from-scratch
  expect_output --substring "must be lower"
}

@test "bud with chown copy" {
  _prefetch alpine
  imgName=alpine-image
  ctrName=alpine-chown
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${imgName} ${TESTSDIR}/bud/copy-chown
  expect_output --substring "user:2367 group:3267"
  run_buildah from --name ${ctrName} ${imgName}
  run_buildah run alpine-chown -- stat -c '%u' /tmp/copychown.txt
  # Validate that output starts with "2367"
  expect_output --substring "2367"

  run_buildah run alpine-chown -- stat -c '%g' /tmp/copychown.txt
  # Validate that output starts with "3267"
  expect_output --substring "3267"
}

@test "bud with chown copy with bad chown flag in Dockerfile with --layers" {
  _prefetch alpine
  imgName=alpine-image
  ctrName=alpine-chown
  run_buildah 125 bud --signature-policy ${TESTSDIR}/policy.json --layers -t ${imgName} -f ${TESTSDIR}/bud/copy-chown/Dockerfile.bad ${TESTSDIR}/bud/copy-chown
  expect_output --substring "COPY only supports the --chown=<uid:gid> and the --from=<image|stage> flags"
}

@test "bud with chown add" {
  _prefetch alpine
  imgName=alpine-image
  ctrName=alpine-chown
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${imgName} ${TESTSDIR}/bud/add-chown
  expect_output --substring "user:2367 group:3267"
  run_buildah from --name ${ctrName} ${imgName}
  run_buildah run alpine-chown -- stat -c '%u' /tmp/addchown.txt
  # Validate that output starts with "2367"
  expect_output --substring "2367"

  run_buildah run alpine-chown -- stat -c '%g' /tmp/addchown.txt
  # Validate that output starts with "3267"
  expect_output --substring "3267"
}

@test "bud with chown add with bad chown flag in Dockerfile with --layers" {
  _prefetch alpine
  imgName=alpine-image
  ctrName=alpine-chown
  run_buildah 125 bud --signature-policy ${TESTSDIR}/policy.json --layers -t ${imgName} -f ${TESTSDIR}/bud/add-chown/Dockerfile.bad ${TESTSDIR}/bud/add-chown
  expect_output --substring "ADD only supports the --chown=<uid:gid> flag"
}

@test "bud with ADD file construct" {
  _prefetch busybox
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t test1 ${TESTSDIR}/bud/add-file
  run_buildah images -a
  expect_output --substring "test1"

  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json test1
  ctr=$output
  run_buildah containers -a
  expect_output --substring "test1"

  run_buildah run $ctr ls /var/file2
  expect_output --substring "/var/file2"
}

@test "bud with COPY of single file creates absolute path with correct permissions" {
  _prefetch ubuntu
  imgName=ubuntu-image
  ctrName=ubuntu-copy
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${imgName} ${TESTSDIR}/bud/copy-create-absolute-path
  expect_output --substring "permissions=755"

  run_buildah from --name ${ctrName} ${imgName}
  run_buildah run ${ctrName} -- stat -c "%a" /usr/lib/python3.7/distutils
  expect_output "755"
}

@test "bud with COPY of single file creates relative path with correct permissions" {
  _prefetch ubuntu
  imgName=ubuntu-image
  ctrName=ubuntu-copy
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${imgName} ${TESTSDIR}/bud/copy-create-relative-path
  expect_output --substring "permissions=755"

  run_buildah from --name ${ctrName} ${imgName}
  run_buildah run ${ctrName} -- stat -c "%a" lib/custom
  expect_output "755"
}

@test "bud with ADD of single file creates absolute path with correct permissions" {
  _prefetch ubuntu
  imgName=ubuntu-image
  ctrName=ubuntu-copy
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${imgName} ${TESTSDIR}/bud/add-create-absolute-path
  expect_output --substring "permissions=755"

  run_buildah from --name ${ctrName} ${imgName}
  run_buildah run ${ctrName} -- stat -c "%a" /usr/lib/python3.7/distutils
  expect_output "755"
}

@test "bud with ADD of single file creates relative path with correct permissions" {
  _prefetch ubuntu
  imgName=ubuntu-image
  ctrName=ubuntu-copy
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${imgName} ${TESTSDIR}/bud/add-create-relative-path
  expect_output --substring "permissions=755"

  run_buildah from --name ${ctrName} ${imgName}
  run_buildah run ${ctrName} -- stat -c "%a" lib/custom
  expect_output "755"
}

@test "bud multi-stage COPY creates absolute path with correct permissions" {
  _prefetch ubuntu
  imgName=ubuntu-image
  ctrName=ubuntu-copy
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -f ${TESTSDIR}/bud/copy-multistage-paths/Dockerfile.absolute -t ${imgName} ${TESTSDIR}/bud/copy-multistage-paths
  expect_output --substring "permissions=755"

  run_buildah from --name ${ctrName} ${imgName}
  run_buildah run ${ctrName} -- stat -c "%a" /my/bin
  expect_output "755"
}

@test "bud multi-stage COPY creates relative path with correct permissions" {
  _prefetch ubuntu
  imgName=ubuntu-image
  ctrName=ubuntu-copy
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -f ${TESTSDIR}/bud/copy-multistage-paths/Dockerfile.relative -t ${imgName} ${TESTSDIR}/bud/copy-multistage-paths
  expect_output --substring "permissions=755"

  run_buildah from --name ${ctrName} ${imgName}
  run_buildah run ${ctrName} -- stat -c "%a" my/bin
  expect_output "755"
}

@test "bud multi-stage COPY with invalid from statement" {
  _prefetch ubuntu
  imgName=ubuntu-image
  ctrName=ubuntu-copy
  run_buildah 125 bud --signature-policy ${TESTSDIR}/policy.json -f ${TESTSDIR}/bud/copy-multistage-paths/Dockerfile.invalid_from -t ${imgName} ${TESTSDIR}/bud/copy-multistage-paths
  expect_output --substring "COPY only supports the --chown=<uid:gid> and the --from=<image|stage> flags"
}

@test "bud COPY to root succeeds" {
  _prefetch ubuntu
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json ${TESTSDIR}/bud/copy-root
}

@test "bud with FROM AS construct" {
  _prefetch alpine
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t test1 ${TESTSDIR}/bud/from-as
  run_buildah images -a
  expect_output --substring "test1"

  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json test1
  ctr=$output
  run_buildah containers -a
  expect_output --substring "test1"

  run_buildah inspect --format "{{.Docker.ContainerConfig.Env}}" --type image test1
  expect_output --substring "LOCAL=/1"
}

@test "bud with FROM AS construct with layers" {
  _prefetch alpine
  run_buildah bud --layers --signature-policy ${TESTSDIR}/policy.json -t test1 ${TESTSDIR}/bud/from-as
  run_buildah images -a
  expect_output --substring "test1"

  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json test1
  ctr=$output
  run_buildah containers -a
  expect_output --substring "test1"

  run_buildah inspect --format "{{.Docker.ContainerConfig.Env}}" --type image test1
  expect_output --substring "LOCAL=/1"
}

@test "bud with FROM AS skip FROM construct" {
  _prefetch alpine
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t test1 -f ${TESTSDIR}/bud/from-as/Dockerfile.skip ${TESTSDIR}/bud/from-as
  expect_output --substring "LOCAL=/1"
  expect_output --substring "LOCAL2=/2"

  run_buildah images -a
  expect_output --substring "test1"

  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json test1
  ctr=$output
  run_buildah containers -a
  expect_output --substring "test1"

  run_buildah mount $ctr
  mnt=$output
  test   -e $mnt/1
  test ! -e $mnt/2

  run_buildah inspect --format "{{.Docker.ContainerConfig.Env}}" --type image test1
  expect_output "[PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin LOCAL=/1]"
}

@test "bud with symlink Dockerfile not specified in file" {
  _prefetch alpine
  target=alpine-image
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/symlink ${TESTSDIR}/bud/symlink
  expect_output --substring "FROM alpine"
}

@test "bud with dir for file but no Dockerfile in dir" {
  target=alpine-image
  run_buildah 125 bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/empty-dir ${TESTSDIR}/bud/empty-dir
  expect_output --substring "no such file or directory"
}

@test "bud with bad dir Dockerfile" {
  target=alpine-image
  run_buildah 125 bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/baddirname ${TESTSDIR}/baddirname
  expect_output --substring "no such file or directory"
}

@test "bud with ARG before FROM default value" {
  _prefetch busybox
  target=leading-args-default
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/leading-args/Dockerfile ${TESTSDIR}/bud/leading-args
}

@test "bud with ARG before FROM" {
  _prefetch busybox:musl
  target=leading-args
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} --build-arg=VERSION=musl -f ${TESTSDIR}/bud/leading-args/Dockerfile ${TESTSDIR}/bud/leading-args
}

@test "bud-with-healthcheck" {
  _prefetch alpine
  target=alpine-image
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} --format docker ${TESTSDIR}/bud/healthcheck
  run_buildah inspect -f '{{printf "%q" .Docker.Config.Healthcheck.Test}} {{printf "%d" .Docker.Config.Healthcheck.StartPeriod}} {{printf "%d" .Docker.Config.Healthcheck.Interval}} {{printf "%d" .Docker.Config.Healthcheck.Timeout}} {{printf "%d" .Docker.Config.Healthcheck.Retries}}' ${target}
  second=1000000000
  threeseconds=$(( 3 * $second ))
  fiveminutes=$(( 5 * 60 * $second ))
  tenminutes=$(( 10 * 60 * $second ))
  expect_output '["CMD-SHELL" "curl -f http://localhost/ || exit 1"]'" $tenminutes $fiveminutes $threeseconds 4" "Healthcheck config"
}

@test "bud with unused build arg" {
  _prefetch alpine busybox
  target=busybox-image
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} --build-arg foo=bar --build-arg foo2=bar2 -f ${TESTSDIR}/bud/build-arg ${TESTSDIR}/bud/build-arg
  expect_output --substring "one or more build args were not consumed: \[foo2\]"
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} --build-arg IMAGE=alpine -f ${TESTSDIR}/bud/build-arg/Dockerfile2 ${TESTSDIR}/bud/build-arg
  ! expect_output --substring "one or more build args were not consumed: \[IMAGE\]"
  expect_output --substring "FROM alpine"
}

@test "bud with copy-from and cache" {
  _prefetch busybox
  target=busybox-image
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers --iidfile ${TESTDIR}/iid1 -f ${TESTSDIR}/bud/copy-from/Dockerfile2 ${TESTSDIR}/bud/copy-from
  cat ${TESTDIR}/iid1
  test -s ${TESTDIR}/iid1
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers --iidfile ${TESTDIR}/iid2 -f ${TESTSDIR}/bud/copy-from/Dockerfile2 ${TESTSDIR}/bud/copy-from
  cat ${TESTDIR}/iid2
  test -s ${TESTDIR}/iid2
  cmp ${TESTDIR}/iid1 ${TESTDIR}/iid2
}

@test "bud with copy-from in Dockerfile no prior FROM" {
  _prefetch php:7.2
  target=php-image
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/copy-from ${TESTSDIR}/bud/copy-from

  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json ${target}
  ctr=$output
  run_buildah mount ${ctr}
  mnt=$output

  test -e $mnt/usr/local/bin/composer
}

@test "bud with copy-from with bad from flag in Dockerfile with --layers" {
  _prefetch php:7.2
  target=php-image
  run_buildah 125 bud --signature-policy ${TESTSDIR}/policy.json --layers -t ${target} -f ${TESTSDIR}/bud/copy-from/Dockerfile.bad ${TESTSDIR}/bud/copy-from
  expect_output --substring "COPY only supports the --chown=<uid:gid> and the --from=<image|stage> flags"
}

@test "bud with copy-from referencing the base image" {
  _prefetch busybox
  target=busybox-derived
  target_mt=busybox-mt-derived
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/copy-from/Dockerfile3 ${TESTSDIR}/bud/copy-from
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --jobs 4 -t ${target} -f ${TESTSDIR}/bud/copy-from/Dockerfile3 ${TESTSDIR}/bud/copy-from

  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/copy-from/Dockerfile4 ${TESTSDIR}/bud/copy-from
  run_buildah bud --no-cache --signature-policy ${TESTSDIR}/policy.json --jobs 4 -t ${target_mt} -f ${TESTSDIR}/bud/copy-from/Dockerfile4 ${TESTSDIR}/bud/copy-from

  run_buildah from  --quiet ${target}
  cid=$output
  run_buildah mount ${cid}
  root_single_job=$output

  run_buildah from --quiet ${target_mt}
  cid=$output
  run_buildah mount ${cid}
  root_multi_job=$output

  # Check that both the version with --jobs 1 and --jobs=N have the same number of files
  test $(find $root_single_job -type f | wc -l) = $(find $root_multi_job -type f | wc -l)
}

@test "bud with copy-from referencing the current stage" {
  _prefetch busybox
  target=busybox-derived
  run_buildah 125 bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/copy-from/Dockerfile2.bad ${TESTSDIR}/bud/copy-from
  expect_output --substring "COPY --from=build: no stage or image found with that name"
}

@test "bud-target" {
  _prefetch alpine ubuntu
  target=target
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} --target mytarget ${TESTSDIR}/bud/target
  expect_output --substring "STEP 1: FROM ubuntu:latest"
  expect_output --substring "STEP 3: FROM alpine:latest AS mytarget"
  run_buildah from --quiet ${target}
  cid=$output
  run_buildah mount ${cid}
  root=$output
  test   -e ${root}/2
  test ! -e ${root}/3
}

@test "bud-no-target-name" {
  _prefetch alpine
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json ${TESTSDIR}/bud/maintainer
}

@test "bud-multi-stage-nocache-nocommit" {
  _prefetch alpine
  # pull the base image directly, so that we don't record it being written to local storage in the next step
  run_buildah pull --signature-policy ${TESTSDIR}/policy.json alpine
  # okay, build an image with two stages
  run_buildah --log-level=debug bud --signature-policy ${TESTSDIR}/policy.json -f ${TESTSDIR}/bud/multi-stage-builds/Dockerfile.name ${TESTSDIR}/bud/multi-stage-builds
  # debug messages should only record us creating one new image: the one for the second stage, since we don't base anything on the first
  run grep "created new image ID" <<< "$output"
  expect_line_count 1
}

@test "bud-multi-stage-cache-nocontainer" {
  skip "FIXME: Broken in CI right now"
  _prefetch alpine
  # first time through, quite normal
  run_buildah bud --layers -t base --signature-policy ${TESTSDIR}/policy.json -f ${TESTSDIR}/bud/multi-stage-builds/Dockerfile.rebase ${TESTSDIR}/bud/multi-stage-builds
  # second time through, everything should be cached, and we shouldn't create a container based on the final image
  run_buildah --log-level=debug bud --layers -t base --signature-policy ${TESTSDIR}/policy.json -f ${TESTSDIR}/bud/multi-stage-builds/Dockerfile.rebase ${TESTSDIR}/bud/multi-stage-builds
  # skip everything up through the final COMMIT step, and make sure we didn't log a "Container ID:" after it
  run sed '0,/COMMIT base/ d' <<< "$output"
  echo "$output" >&2
  test "${#lines[@]}" -gt 1
  run grep "Container ID:" <<< "$output"
  expect_output ""
}

@test "bud copy to symlink" {
  _prefetch alpine
  target=alpine-image
  ctr=alpine-ctr
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/dest-symlink
  expect_output --substring "STEP 5: RUN ln -s "

  run_buildah from --signature-policy ${TESTSDIR}/policy.json --name=${ctr} ${target}
  expect_output --substring ${ctr}

  run_buildah run ${ctr} ls -alF /etc/hbase
  expect_output --substring "/etc/hbase -> /usr/local/hbase/"

  run_buildah run ${ctr} ls -alF /usr/local/hbase
  expect_output --substring "Dockerfile"
}

@test "bud copy to dangling symlink" {
  _prefetch ubuntu
  target=ubuntu-image
  ctr=ubuntu-ctr
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/dest-symlink-dangling
  expect_output --substring "STEP 3: RUN ln -s "

  run_buildah from --signature-policy ${TESTSDIR}/policy.json --name=${ctr} ${target}
  expect_output --substring ${ctr}

  run_buildah run ${ctr} ls -alF /src
  expect_output --substring "/src -> /symlink"

  run_buildah run ${ctr} ls -alF /symlink
  expect_output --substring "Dockerfile"
}

@test "bud WORKDIR isa symlink" {
  _prefetch alpine
  target=alpine-image
  ctr=alpine-ctr
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/workdir-symlink
  expect_output --substring "STEP 3: RUN ln -sf "

  run_buildah from --signature-policy ${TESTSDIR}/policy.json --name=${ctr} ${target}
  expect_output --substring ${ctr}

  run_buildah run ${ctr} ls -alF /tempest
  expect_output --substring "/tempest -> /var/lib/tempest/"

  run_buildah run ${ctr} ls -alF /etc/notareal.conf
  expect_output --substring "\-rw\-rw\-r\-\-"
}

@test "bud WORKDIR isa symlink no target dir" {
  _prefetch alpine
  target=alpine-image
  ctr=alpine-ctr
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f Dockerfile-2 ${TESTSDIR}/bud/workdir-symlink
  expect_output --substring "STEP 2: RUN ln -sf "

  run_buildah from --signature-policy ${TESTSDIR}/policy.json --name=${ctr} ${target}
  expect_output --substring ${ctr}

  run_buildah run ${ctr} ls -alF /tempest
  expect_output --substring "/tempest -> /var/lib/tempest/"

  run_buildah run ${ctr} ls /tempest
  expect_output --substring "Dockerfile-2"

  run_buildah run ${ctr} ls -alF /etc/notareal.conf
  expect_output --substring "\-rw\-rw\-r\-\-"
}

@test "bud WORKDIR isa symlink no target dir and follow on dir" {
  _prefetch alpine
  target=alpine-image
  ctr=alpine-ctr
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f Dockerfile-3 ${TESTSDIR}/bud/workdir-symlink
  expect_output --substring "STEP 2: RUN ln -sf "

  run_buildah from --signature-policy ${TESTSDIR}/policy.json --name=${ctr} ${target}
  expect_output --substring ${ctr}

  run_buildah run ${ctr} ls -alF /tempest
  expect_output --substring "/tempest -> /var/lib/tempest/"

  run_buildah run ${ctr} ls /tempest
  expect_output --substring "Dockerfile-3"

  run_buildah run ${ctr} ls /tempest/lowerdir
  expect_output --substring "Dockerfile-3"

  run_buildah run ${ctr} ls -alF /etc/notareal.conf
  expect_output --substring "\-rw\-rw\-r\-\-"
}

@test "buidah bud --volume" {
  voldir=${TESTDIR}/bud-volume
  mkdir -p ${voldir}

  _prefetch alpine
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -v ${voldir}:/testdir ${TESTSDIR}/bud/mount
  expect_output --substring "/testdir"
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -v ${voldir}:/testdir:rw ${TESTSDIR}/bud/mount
  expect_output --substring "/testdir"
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -v ${voldir}:/testdir:rw,z ${TESTSDIR}/bud/mount
  expect_output --substring "/testdir"
}

@test "bud-copy-dot with --layers picks up changed file" {
  _prefetch alpine
  cp -a ${TESTSDIR}/bud/use-layers ${TESTDIR}/use-layers

  mkdir -p ${TESTDIR}/use-layers/subdir
  touch ${TESTDIR}/use-layers/subdir/file.txt
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers --iidfile ${TESTDIR}/iid1 -f Dockerfile.7 ${TESTDIR}/use-layers

  touch ${TESTDIR}/use-layers/subdir/file.txt
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers --iidfile ${TESTDIR}/iid2 -f Dockerfile.7 ${TESTDIR}/use-layers

  if [[ $(cat ${TESTDIR}/iid1) != $(cat ${TESTDIR}/iid2) ]]; then
    echo "Expected image id to not change after touching a file copied into the image" >&2
    false
  fi
}

@test "buildah-bud-policy" {
  target=foo

  # A deny-all policy should prevent us from pulling the base image.
  run_buildah 125 bud --signature-policy ${TESTSDIR}/deny.json -t ${target} -v ${TESTSDIR}:/testdir ${TESTSDIR}/bud/mount
  expect_output --substring 'Source image rejected: Running image .* rejected by policy.'

  # A docker-only policy should allow us to pull the base image and commit.
  run_buildah bud --signature-policy ${TESTSDIR}/docker.json -t ${target} -v ${TESTSDIR}:/testdir ${TESTSDIR}/bud/mount
  # A deny-all policy shouldn't break pushing, since policy is only evaluated
  # on the source image, and we force it to allow local storage.
  run_buildah push --signature-policy ${TESTSDIR}/deny.json ${target} dir:${TESTDIR}/mount
  run_buildah rmi ${target}

  # A docker-only policy should allow us to pull the base image first...
  run_buildah pull --signature-policy ${TESTSDIR}/docker.json alpine
  # ... and since we don't need to pull the base image, a deny-all policy shouldn't break a build.
  run_buildah bud --signature-policy ${TESTSDIR}/deny.json -t ${target} -v ${TESTSDIR}:/testdir ${TESTSDIR}/bud/mount
  # A deny-all policy shouldn't break pushing, since policy is only evaluated
  # on the source image, and we force it to allow local storage.
  run_buildah push --signature-policy ${TESTSDIR}/deny.json ${target} dir:${TESTDIR}/mount
  # Similarly, a deny-all policy shouldn't break committing directly to other locations.
  run_buildah bud --signature-policy ${TESTSDIR}/deny.json -t dir:${TESTDIR}/mount -v ${TESTSDIR}:/testdir ${TESTSDIR}/bud/mount
}

@test "bud-copy-replace-symlink" {
  mkdir -p ${TESTDIR}/top
  cp ${TESTSDIR}/bud/symlink/Dockerfile.replace-symlink ${TESTDIR}/top/
  ln -s Dockerfile.replace-symlink ${TESTDIR}/top/symlink
  echo foo > ${TESTDIR}/top/.dockerignore
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -f ${TESTDIR}/top/Dockerfile.replace-symlink ${TESTDIR}/top
}

@test "bud-copy-recurse" {
  mkdir -p ${TESTDIR}/recurse
  cp ${TESTSDIR}/bud/recurse/Dockerfile ${TESTDIR}/recurse
  echo foo > ${TESTDIR}/recurse/.dockerignore
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json ${TESTDIR}/recurse
}

@test "bud copy with .dockerignore #1" {
  _prefetch alpine
  mytmpdir=${TESTDIR}/my-dir
  mkdir -p $mytmpdir/stuff/huge/usr/bin/
  (cd $mytmpdir/stuff/huge/usr/bin/; touch file1 file2)
  (cd $mytmpdir/stuff/huge/usr/; touch file3)

  cat > $mytmpdir/.dockerignore << _EOF
stuff/huge/*
!stuff/huge/usr/bin/*
_EOF

  cat > $mytmpdir/Containerfile << _EOF
FROM alpine
COPY stuff /tmp/stuff
RUN find /tmp/stuff -type f
_EOF

  run_buildah bud -t testbud --signature-policy ${TESTSDIR}/policy.json ${mytmpdir}
  expect_output --substring "file1"
  expect_output --substring "file2"
  ! expect_output --substring "file3"
}

@test "bud copy with .dockerignore #2" {
  _prefetch alpine
  mytmpdir=${TESTDIR}/my-dir1
  mkdir -p $mytmpdir/stuff/huge/usr/bin/
  (cd $mytmpdir/stuff/huge/usr/bin/; touch file1 file2)

  cat > $mytmpdir/.dockerignore << _EOF
stuff/huge/*
_EOF

  cat > $mytmpdir/Containerfile << _EOF
FROM alpine
COPY stuff /tmp/stuff
RUN find /tmp/stuff -type f
_EOF

  run_buildah bud -t testbud --signature-policy ${TESTSDIR}/policy.json ${mytmpdir}
  ! expect_output --substring "file1"
  ! expect_output --substring "file2"
}

@test "bud-copy-workdir" {
  target=testimage
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/copy-workdir
  run_buildah from ${target}
  cid="$output"
  run_buildah mount "${cid}"
  root="$output"
  test -s "${root}"/file1.txt
  test -d "${root}"/subdir
  test -s "${root}"/subdir/file2.txt
}

@test "bud-build-arg-cache" {
  _prefetch busybox
  target=derived-image
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t ${target} -f Dockerfile3 ${TESTSDIR}/bud/build-arg
  run_buildah inspect -f '{{.FromImageID}}' ${target}
  targetid="$output"

  # With build args, we should not find the previous build as a cached result.
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t ${target} -f Dockerfile3 --build-arg=UID=17122 --build-arg=CODE=/copr/coprs_frontend --build-arg=USERNAME=praiskup --build-arg=PGDATA=/pgdata ${TESTSDIR}/bud/build-arg
  run_buildah inspect -f '{{.FromImageID}}' ${target}
  argsid="$output"
  [[ "$argsid" != "$targetid" ]]

  # With build args, even in a different order, we should end up using the previous build as a cached result.
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t ${target} -f Dockerfile3 --build-arg=UID=17122 --build-arg=CODE=/copr/coprs_frontend --build-arg=USERNAME=praiskup --build-arg=PGDATA=/pgdata ${TESTSDIR}/bud/build-arg
  run_buildah inspect -f '{{.FromImageID}}' ${target}
  expect_output "$argsid"

  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t ${target} -f Dockerfile3 --build-arg=CODE=/copr/coprs_frontend --build-arg=USERNAME=praiskup --build-arg=PGDATA=/pgdata --build-arg=UID=17122 ${TESTSDIR}/bud/build-arg
  run_buildah inspect -f '{{.FromImageID}}' ${target}
  expect_output "$argsid"

  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t ${target} -f Dockerfile3 --build-arg=USERNAME=praiskup --build-arg=PGDATA=/pgdata --build-arg=UID=17122 --build-arg=CODE=/copr/coprs_frontend ${TESTSDIR}/bud/build-arg
  run_buildah inspect -f '{{.FromImageID}}' ${target}
  expect_output "$argsid"

  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t ${target} -f Dockerfile3 --build-arg=PGDATA=/pgdata --build-arg=UID=17122 --build-arg=CODE=/copr/coprs_frontend --build-arg=USERNAME=praiskup ${TESTSDIR}/bud/build-arg
  run_buildah inspect -f '{{.FromImageID}}' ${target}
  expect_output "$argsid"
}

@test "bud test RUN with a priv'd command" {
  _prefetch alpine
  target=alpinepriv
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/run-privd/Dockerfile ${TESTSDIR}/bud/run-privd
  expect_output --substring "STEP 3: COMMIT"
  run_buildah images -q
  expect_line_count 2
}

@test "bud-copy-dockerignore-hardlinks" {
  target=image
  mkdir -p ${TESTDIR}/hardlinks/subdir
  cp ${TESTSDIR}/bud/recurse/Dockerfile ${TESTDIR}/hardlinks
  echo foo > ${TESTDIR}/hardlinks/.dockerignore
  echo test1 > ${TESTDIR}/hardlinks/subdir/test1.txt
  ln ${TESTDIR}/hardlinks/subdir/test1.txt ${TESTDIR}/hardlinks/subdir/test2.txt
  ln ${TESTDIR}/hardlinks/subdir/test2.txt ${TESTDIR}/hardlinks/test3.txt
  ln ${TESTDIR}/hardlinks/test3.txt ${TESTDIR}/hardlinks/test4.txt
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTDIR}/hardlinks
  run_buildah from ${target}
  ctrid="$output"
  run_buildah mount "$ctrid"
  root="$output"

  run stat -c "%d:%i" ${root}/subdir/test1.txt
  id1=$output
  run stat -c "%h" ${root}/subdir/test1.txt
  expect_output 4
  run stat -c "%d:%i" ${root}/subdir/test2.txt
  expect_output $id1 "stat(test2) == stat(test1)"
  run stat -c "%h" ${root}/subdir/test2.txt
  expect_output 4
  run stat -c "%d:%i" ${root}/test3.txt
  expect_output $id1 "stat(test3) == stat(test1)"
  run stat -c "%h" ${root}/test3.txt
  expect_output 4
  run stat -c "%d:%i" ${root}/test4.txt
  expect_output $id1 "stat(test4) == stat(test1)"
  run stat -c "%h" ${root}/test4.txt
  expect_output 4
}

@test "bud without any arguments should succeed" {
  cd ${TESTSDIR}/bud/from-scratch
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json
}

@test "bud without any arguments should fail when no Dockerfile exist" {
  cd $(mktemp -d)
  run_buildah 125 bud --signature-policy ${TESTSDIR}/policy.json
  expect_output --substring "no such file or directory"
}

@test "bud with specified context should fail if directory contains no Dockerfile" {
  DIR=$(mktemp -d)
  run_buildah 125 bud --signature-policy ${TESTSDIR}/policy.json "$DIR"
  expect_output --substring "no such file or directory"
}

@test "bud with specified context should fail if assumed Dockerfile is a directory" {
  DIR=$(mktemp -d)
  mkdir -p "$DIR"/Dockerfile
  run_buildah 125 bud --signature-policy ${TESTSDIR}/policy.json "$DIR"
  expect_output --substring "is not a file"
}

@test "bud with specified context should fail if context contains not-existing Dockerfile" {
  DIR=$(mktemp -d)
  run_buildah 125 bud --signature-policy ${TESTSDIR}/policy.json "$DIR"/Dockerfile
  expect_output --substring "no such file or directory"
}

@test "bud with specified context should succeed if context contains existing Dockerfile" {
  DIR=$(mktemp -d)
  echo "FROM alpine" > "$DIR"/Dockerfile
  run_buildah 0 bud --signature-policy ${TESTSDIR}/policy.json "$DIR"/Dockerfile
}

@test "bud with specified context should fail if context contains empty Dockerfile" {
  DIR=$(mktemp -d)
  touch "$DIR"/Dockerfile
  run_buildah 125 bud --signature-policy ${TESTSDIR}/policy.json "$DIR"/Dockerfile
}

@test "bud-no-change" {
  _prefetch alpine
  parent=alpine
  target=no-change-image
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/no-change
  run_buildah inspect --format '{{printf "%q" .FromImageDigest}}' ${parent}
  parentid="$output"
  run_buildah inspect --format '{{printf "%q" .FromImageDigest}}' ${target}
  expect_output "$parentid"
}

@test "bud-no-change-label" {
  run_buildah --version
  local -a output_fields=($output)
  buildah_version=${output_fields[2]}
  want_output='map["io.buildah.version":"'$buildah_version'" "test":"label"]'

  _prefetch alpine
  parent=alpine
  target=no-change-image
  run_buildah bud --label "test=label" --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/no-change
  run_buildah inspect --format '{{printf "%q" .Docker.Config.Labels}}' ${target}
  expect_output "$want_output"
}

@test "bud-no-change-annotation" {
  _prefetch alpine
  target=no-change-image
  run_buildah bud --annotation "test=annotation" --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/no-change
  run_buildah inspect --format '{{printf "%q" .ImageAnnotations}}' ${target}
  expect_output 'map["test":"annotation"]'
}

@test "bud-squash-layers" {
  _prefetch alpine
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --squash ${TESTSDIR}/bud/layers-squash
}

@test "bud-squash-hardlinks" {
  _prefetch busybox
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --squash ${TESTSDIR}/bud/layers-squash/Dockerfile.hardlinks
}

@test "bud with additional directory of devices" {
  skip_if_chroot
  skip_if_rootless

  _prefetch alpine
  target=alpine-image
  rm -rf ${TESTSDIR}/foo
  mkdir -p ${TESTSDIR}/foo
  mknod ${TESTSDIR}/foo/null c 1 3
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --device ${TESTSDIR}/foo:/dev/fuse  -t ${target} -f ${TESTSDIR}/bud/device/Dockerfile ${TESTSDIR}/bud/device
  expect_output --substring "null"
}

@test "bud with additional device" {
  _prefetch alpine
  target=alpine-image
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --device /dev/fuse -t ${target} -f ${TESTSDIR}/bud/device/Dockerfile ${TESTSDIR}/bud/device
  [ "${status}" -eq 0 ]
  expect_output --substring "/dev/fuse"
}

@test "bud with Containerfile" {
  _prefetch alpine
  target=alpine-image
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/containerfile
  [ "${status}" -eq 0 ]
  expect_output --substring "FROM alpine"
}

@test "bud with Dockerfile" {
  _prefetch alpine
  target=alpine-image
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/dockerfile
  [ "${status}" -eq 0 ]
  expect_output --substring "FROM alpine"
}

@test "bud with Containerfile and Dockerfile" {
  _prefetch alpine
  target=alpine-image
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/containeranddockerfile
  [ "${status}" -eq 0 ]
  expect_output --substring "FROM alpine"
}

@test "bud-http-context-with-Containerfile" {
  _test_http http-context-containerfile context.tar
}

@test "bud with Dockerfile from stdin" {
  _prefetch alpine
  target=df-stdin
  cat ${TESTSDIR}/bud/context-from-stdin/Dockerfile | buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -
  [ "$?" -eq 0 ]
  run_buildah from --quiet ${target}
  cid=$output
  run_buildah mount ${cid}
  root=$output

  test -s $root/scratchfile
  run cat $root/scratchfile
  expect_output "stdin-context" "contents of \$root/scratchfile"

  # FROM scratch overrides FROM alpine
  test ! -s $root/etc/alpine-release
}

@test "bud with Dockerfile from stdin tar" {
  _prefetch alpine
  target=df-stdin
  tar -c -C ${TESTSDIR}/bud/context-from-stdin . | buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -
  [ "$?" -eq 0 ]
  run_buildah from --quiet ${target}
  cid=$output
  run_buildah mount ${cid}
  root=$output

  test -s $root/scratchfile
  run cat $root/scratchfile
  expect_output "stdin-context" "contents of \$root/scratchfile"

  # FROM scratch overrides FROM alpine
  test ! -s $root/etc/alpine-release
}

@test "bud containerfile with args" {
  _prefetch alpine
  target=use-args
  touch ${TESTSDIR}/bud/use-args/abc.txt
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} --build-arg=abc.txt ${TESTSDIR}/bud/use-args
  expect_output --substring "COMMIT use-args"
  run_buildah from --quiet ${target}
  ctrID=$output
  run_buildah run $ctrID ls abc.txt
  expect_output --substring "abc.txt"

  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f Containerfile.destination --build-arg=testArg=abc.txt --build-arg=destination=/tmp ${TESTSDIR}/bud/use-args
  expect_output --substring "COMMIT use-args"
  run_buildah from --quiet ${target}
  ctrID=$output
  run_buildah run $ctrID ls /tmp/abc.txt
  expect_output --substring "abc.txt"

  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f Containerfile.dest_nobrace --build-arg=testArg=abc.txt --build-arg=destination=/tmp ${TESTSDIR}/bud/use-args
  expect_output --substring "COMMIT use-args"
  run_buildah from --quiet ${target}
  ctrID=$output
  run_buildah run $ctrID ls /tmp/abc.txt
  expect_output --substring "abc.txt"

  rm ${TESTSDIR}/bud/use-args/abc.txt
}

@test "bud using gitrepo and branch" {
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t gittarget -f tests/bud/shell/Dockerfile git://github.com/containers/buildah#release-1.11-rhel
}

# Fixes #1906: buildah was not detecting changed tarfile
@test "bud containerfile with tar archive in copy" {
  _prefetch busybox
  # First check to verify cache is used if the tar file does not change
  target=copy-archive
  date > ${TESTSDIR}/bud/${target}/test
  tar -C $TESTSDIR -cJf ${TESTSDIR}/bud/${target}/test.tar.xz bud/${target}/test
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t ${target} ${TESTSDIR}/bud/${target}
  expect_output --substring "COMMIT copy-archive"

  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t ${target} ${TESTSDIR}/bud/${target}
  expect_output --substring " Using cache"
  expect_output --substring "COMMIT copy-archive"

  # Now test that we do NOT use cache if the tar file changes
  echo This is a change >> ${TESTSDIR}/bud/${target}/test
  tar -C $TESTSDIR -cJf ${TESTSDIR}/bud/${target}/test.tar.xz bud/${target}/test
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t ${target} ${TESTSDIR}/bud/${target}
  if [[ "$output" =~ " Using cache" ]]; then
      expect_output "[no instance of 'Using cache']" "no cache used"
  fi
  expect_output --substring "COMMIT copy-archive"

  rm -f ${TESTSDIR}/bud/${target}/test*
}

@test "bud pull never" {
  target=pull
  run_buildah 125 bud --signature-policy ${TESTSDIR}/policy.json -t ${target} --pull-never ${TESTSDIR}/bud/pull
  expect_output --substring "pull policy is \"never\" but \""
  expect_output --substring "\" could not be found locally"

  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} --pull ${TESTSDIR}/bud/pull
  expect_output --substring "COMMIT pull"

  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} --pull-never ${TESTSDIR}/bud/pull
  expect_output --substring "COMMIT pull"
}

@test "bud pull false no local image" {
  target=pull
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} --pull=false ${TESTSDIR}/bud/pull
  expect_output --substring "COMMIT pull"
}

@test "bud with Containerfile should fail with nonexist authfile" {
  target=alpine-image
  run_buildah 125 bud --authfile /tmp/nonexist --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/containerfile
}

@test "bud COPY with URL should fail" {
  mkdir ${TESTSDIR}/bud/copy
  FILE=${TESTSDIR}/bud/copy/Dockerfile.url
  /bin/cat <<EOM >$FILE
FROM alpine:latest
COPY https://getfedora.org/index.html .
EOM

  run_buildah 125 bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/copy/Dockerfile.url ${TESTSDIR}/bud/copy
  rm -r ${TESTSDIR}/bud/copy
}

@test "bud quiet" {
  _prefetch alpine
  run_buildah bud --format docker -t quiet-test --signature-policy ${TESTSDIR}/policy.json -q ${TESTSDIR}/bud/shell
  expect_line_count 1
  expect_output --substring '^[0-9a-f]{64}$'
}

@test "bud COPY with Env Var in Containerfile" {
  _prefetch alpine
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t testctr ${TESTSDIR}/bud/copy-envvar
  run_buildah from testctr
  run_buildah run testctr-working-container ls /file-0.0.1.txt
  run_buildah rm -a

  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t testctr ${TESTSDIR}/bud/copy-envvar
  run_buildah from testctr
  run_buildah run testctr-working-container ls /file-0.0.1.txt
  run_buildah rm -a
}

@test "bud with custom arch" {
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json \
    -f ${TESTSDIR}/bud/from-scratch/Dockerfile \
    -t arch-test \
    --arch=arm

  run_buildah inspect --format "{{ .Docker.Architecture }}" arch-test
  expect_output arm

  run_buildah inspect --format "{{ .OCIv1.Architecture }}" arch-test
  expect_output arm
}

@test "bud with custom os" {
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json \
    -f ${TESTSDIR}/bud/from-scratch/Dockerfile \
    -t os-test \
    --os=windows

  run_buildah inspect --format "{{ .Docker.OS }}" os-test
  expect_output windows

  run_buildah inspect --format "{{ .OCIv1.OS }}" os-test
  expect_output windows
}

@test "bud with custom platform" {
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json \
    -f ${TESTSDIR}/bud/from-scratch/Dockerfile \
    -t platform-test \
    --platform=windows/arm

  run_buildah inspect --format "{{ .Docker.OS }}" platform-test
  expect_output windows

  run_buildah inspect --format "{{ .OCIv1.OS }}" platform-test
  expect_output windows

  run_buildah inspect --format "{{ .Docker.Architecture }}" platform-test
  expect_output arm

  run_buildah inspect --format "{{ .OCIv1.Architecture }}" platform-test
  expect_output arm
}

@test "bud Add with linked tarball" {
  _prefetch alpine
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -f ${TESTSDIR}/bud/symlink/Containerfile.add-tar-with-link -t testctr ${TESTSDIR}/bud/symlink
  run_buildah from testctr
  run_buildah run testctr-working-container ls /tmp/testdir/testfile.txt
  run_buildah rm -a
  run_buildah rmi -a -f

  _prefetch alpine
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -f ${TESTSDIR}/bud/symlink/Containerfile.add-tar-gz-with-link -t testctr ${TESTSDIR}/bud/symlink
  run_buildah from testctr
  run_buildah run testctr-working-container ls /tmp/testdir/testfile.txt
  run_buildah rm -a
  run_buildah rmi -a -f
}

@test "bud file above context directory" {
  run_buildah 125 bud --signature-policy ${TESTSDIR}/policy.json -t testctr ${TESTSDIR}/bud/context-escape-dir/testdir
  expect_output --substring "escaping context directory error"
}

@test "bud-multi-stage-args-scope" {
  _prefetch alpine
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t multi-stage-args --build-arg SECRET=secretthings -f Dockerfile.arg ${TESTSDIR}/bud/multi-stage-builds
  run_buildah from --name test-container multi-stage-args
  run_buildah run test-container -- cat test_file
  expect_output ""
}

@test "bud-multi-stage-args-history" {
  _prefetch alpine
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t multi-stage-args --build-arg SECRET=secretthings -f Dockerfile.arg ${TESTSDIR}/bud/multi-stage-builds
  run_buildah inspect --format '{{range .History}}{{println .CreatedBy}}{{end}}' multi-stage-args
  run grep "secretthings" <<< "$output"
  expect_output ""

  run_buildah inspect --format '{{range .OCIv1.History}}{{println .CreatedBy}}{{end}}' multi-stage-args
  run grep "secretthings" <<< "$output"
  expect_output ""

  run_buildah inspect --format '{{range .Docker.History}}{{println .CreatedBy}}{{end}}' multi-stage-args
  run grep "secretthings" <<< "$output"
  expect_output ""
}

@test "bud with encrypted FROM image" {
  _prefetch busybox
  mkdir ${TESTDIR}/tmp
  openssl genrsa -out ${TESTDIR}/tmp/mykey.pem 1024
  openssl genrsa -out ${TESTDIR}/tmp/mykey2.pem 1024
  openssl rsa -in ${TESTDIR}/tmp/mykey.pem -pubout > ${TESTDIR}/tmp/mykey.pub
  run_buildah push --signature-policy ${TESTSDIR}/policy.json --tls-verify=false --creds testuser:testpassword --encryption-key jwe:${TESTDIR}/tmp/mykey.pub busybox docker://localhost:5000/buildah/busybox_encrypted:latest

  target=busybox-image
  # Try to build from encrypted image without key
  run_buildah 125 bud --signature-policy ${TESTSDIR}/policy.json --tls-verify=false  --creds testuser:testpassword -t ${target} -f ${TESTSDIR}/bud/from-encrypted-image/Dockerfile
  # Try to build from encrypted image with wrong key
  run_buildah 125 bud --signature-policy ${TESTSDIR}/policy.json --tls-verify=false  --creds testuser:testpassword --decryption-key ${TESTDIR}/tmp/mykey2.pem -t ${target} -f ${TESTSDIR}/bud/from-encrypted-image/Dockerfile
  # Try to build with the correct key
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --tls-verify=false  --creds testuser:testpassword --decryption-key ${TESTDIR}/tmp/mykey.pem -t ${target} -f ${TESTSDIR}/bud/from-encrypted-image/Dockerfile

  rm -rf ${TESTDIR}/tmp
}

@test "bud with --build-arg" {
  _prefetch alpine
  run_buildah --log-level "warn" bud --signature-policy ${TESTSDIR}/policy.json -t test ${TESTSDIR}/bud/build-arg
  expect_output --substring 'missing .+ build argument'
}

@test "bud arg and env var with same name" {
  # Regression test for https://github.com/containers/buildah/issues/2345
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t testctr ${TESTSDIR}/bud/dupe-arg-env-name
  expect_output --substring "https://example.org/bar"
}

@test "bud copy chown with newuser" {
  # Regression test for https://github.com/containers/buildah/issues/2192
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t testctr -f ${TESTSDIR}/bud/copy-chown/Containerfile.chown_user ${TESTSDIR}/bud/copy-chown
  expect_output --substring "myuser myuser"
}

@test "bud-builder-identity" {
  _prefetch alpine
  parent=alpine
  target=no-change-image
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/from-scratch
  run_buildah --version
  local -a output_fields=($output)
  buildah_version=${output_fields[2]}

  run_buildah inspect --format '{{ index .Docker.Config.Labels "io.buildah.version"}}' $target
  expect_output "$buildah_version"
}

@test "run check --from with arg" {
  skip_if_no_runtime

  ${OCI} --version
  _prefetch alpine
  _prefetch debian

  run_buildah bud --build-arg base=alpine --build-arg toolchainname=busybox --build-arg destinationpath=/tmp --pull=false --signature-policy ${TESTSDIR}/policy.json -f ${TESTSDIR}/bud/from-with-arg/Containerfile .
  expect_output --substring "FROM alpine"
  expect_output --substring 'STEP 4: COPY --from=\$\{toolchainname\} \/ \$\{destinationpath\}'
  run_buildah rm -a
}

@test "bud timestamp" {
  _prefetch alpine
  run_buildah bud --timestamp=0 --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json -t timestamp -f Dockerfile.1 ${TESTSDIR}/bud/cache-stages
  cid=$output
  run_buildah inspect --format '{{ .Docker.Created }}' timestamp
  expect_output --substring "1970-01-01"
  run_buildah inspect --format '{{ .OCIv1.Created }}' timestamp
  expect_output --substring "1970-01-01"
  run_buildah inspect --format '{{ .History }}' timestamp
  expect_output --substring '1970-01-01 00:00:00'

  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json timestamp
  cid=$output
  run_buildah run $cid ls -l /tmpfile
  expect_output --substring "1970"

  rm -rf ${TESTDIR}/tmp
}

@test "bud timestamp compare" {
  _prefetch alpine
  TIMESTAMP=$(date '+%s')
  run_buildah bud --timestamp=${TIMESTAMP} --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json -t timestamp -f Dockerfile.1 ${TESTSDIR}/bud/cache-stages
  cid=$output

  run_buildah bud --timestamp=${TIMESTAMP} --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json -t timestamp -f Dockerfile.1 ${TESTSDIR}/bud/cache-stages
  expect_output "$cid"

  rm -rf ${TESTDIR}/tmp
}

@test "bud with-rusage" {
  _prefetch alpine
  run_buildah bud --log-rusage --layers --pull=false --format docker --signature-policy ${TESTSDIR}/policy.json ${TESTSDIR}/bud/shell
  cid=$output
  # expect something that looks like it was formatted using pkg/rusage.FormatDiff()
  expect_output --substring ".*\(system\).*\(user\).*\(elapsed\).*input.*output"
}

@test "bud-caching-from-scratch" {
  _prefetch alpine
  # run the build once
  run_buildah bud --quiet --layers --pull=false --format docker --signature-policy ${TESTSDIR}/policy.json ${TESTSDIR}/bud/cache-scratch
  iid="$output"
  # now run it again - the cache should give us the same final image ID
  run_buildah bud --quiet --layers --pull=false --format docker --signature-policy ${TESTSDIR}/policy.json ${TESTSDIR}/bud/cache-scratch
  iid2="$output"
  expect_output --substring "$iid"
  # now run it *again*, except with more content added at an intermediate step, which should invalidate the cache
  run_buildah bud --quiet --layers --pull=false --format docker --signature-policy ${TESTSDIR}/policy.json -f Dockerfile.different1 ${TESTSDIR}/bud/cache-scratch
  test "$output" != "$iid"
  # now run it *again* again, except with more content added at an intermediate step, which should invalidate the cache
  run_buildah bud --quiet --layers --pull=false --format docker --signature-policy ${TESTSDIR}/policy.json -f Dockerfile.different2 ${TESTSDIR}/bud/cache-scratch
  test "$output" != "$iid"
  test "$output" != "$iid2"
}

@test "bud-caching-from-scratch-config" {
  _prefetch alpine
  # run the build once
  run_buildah bud --quiet --layers --pull=false --format docker --signature-policy ${TESTSDIR}/policy.json -f Dockerfile.config ${TESTSDIR}/bud/cache-scratch
  iid="$output"
  # now run it again - the cache should give us the same final image ID
  run_buildah bud --quiet --layers --pull=false --format docker --signature-policy ${TESTSDIR}/policy.json -f Dockerfile.config ${TESTSDIR}/bud/cache-scratch
  iid2="$output"
  expect_output --substring "$iid"
  # now run it *again*, except with more content added at an intermediate step, which should invalidate the cache
  run_buildah bud --quiet --layers --pull=false --format docker --signature-policy ${TESTSDIR}/policy.json -f Dockerfile.different1 ${TESTSDIR}/bud/cache-scratch
  test "$output" != "$iid"
  # now run it *again* again, except with more content added at an intermediate step, which should invalidate the cache
  run_buildah bud --quiet --layers --pull=false --format docker --signature-policy ${TESTSDIR}/policy.json -f Dockerfile.different2 ${TESTSDIR}/bud/cache-scratch
  test "$output" != "$iid"
  test "$output" != "$iid2"
}

@test "bud capabilities test" {
  _prefetch busybox
  run_buildah bud -t testcap --signature-policy ${TESTSDIR}/policy.json -f ${TESTSDIR}/bud/capabilities/Dockerfile
  expect_output --substring "uid=3267"
  expect_output --substring "CapBnd:	00000000a80425fb"
  expect_output --substring "CapEff:	0000000000000000"
}

@test "bud does not gobble stdin" {
  _prefetch alpine

  ctxdir=${TESTDIR}/bud
  mkdir -p $ctxdir
  cat >$ctxdir/Dockerfile <<EOF
FROM alpine
RUN true
EOF

  random_msg=$(head -10 /dev/urandom | tr -dc a-zA-Z0-9 | head -c12)

  # Prior to #2708, buildah bud would gobble up its stdin even if it
  # didn't actually use it. This prevented the use of 'cmdlist | bash';
  # if 'buildah bud' was in cmdlist, everything past it would be lost.
  #
  # This is ugly but effective: it checks that buildah passes stdin untouched.
  passthru=$(echo "$random_msg" | (run_buildah bud --quiet --signature-policy ${TESTSDIR}/policy.json -t stdin-test ${ctxdir} >/dev/null; cat))

  expect_output --from="$passthru" "$random_msg" "stdin was passed through"
}

@test "bud cache by format" {
  # Build first in Docker format.  Whether we do OCI or Docker first shouldn't matter, so we picked one.
  run_buildah bud --iidfile first-docker  --format docker --layers --quiet --signature-policy ${TESTSDIR}/policy.json ${TESTSDIR}/bud/cache-format
  # Build in OCI format.  Cache should not re-use the same images, so we should get a different image ID.
  run_buildah bud --iidfile first-oci     --format oci    --layers --quiet --signature-policy ${TESTSDIR}/policy.json ${TESTSDIR}/bud/cache-format
  # Build in Docker format again.  Cache traversal should 100% hit the Docker image, so we should get its image ID.
  run_buildah bud --iidfile second-docker --format docker --layers --quiet --signature-policy ${TESTSDIR}/policy.json ${TESTSDIR}/bud/cache-format
  # Build in OCI format again.  Cache traversal should 100% hit the OCI image, so we should get its image ID.
  run_buildah bud --iidfile second-oci    --format oci    --layers --quiet --signature-policy ${TESTSDIR}/policy.json ${TESTSDIR}/bud/cache-format
  # Compare them.  The two images we built in Docker format should be the same, the two we built in OCI format
  # should be the same, but the OCI and Docker format images should be different.
  cmp first-docker second-docker
  cmp first-oci    second-oci
  run cmp first-docker first-oci
  [[ "$status" -ne 0 ]]
}

@test "bud cache add-copy-chown" {
  # Build each variation of COPY (from context, from previous stage) and ADD (from context, not overriding an archive, URL) twice.
  # Each second build should produce an image with the same ID as the first build, because the cache matches, but they should
  # otherwise all be different.
  run_buildah bud --iidfile copy1 --layers --quiet --signature-policy ${TESTSDIR}/policy.json -f Dockerfile.copy1 ${TESTSDIR}/bud/cache-chown
  run_buildah bud --iidfile prev1 --layers --quiet --signature-policy ${TESTSDIR}/policy.json -f Dockerfile.prev1 ${TESTSDIR}/bud/cache-chown
  run_buildah bud --iidfile add1  --layers --quiet --signature-policy ${TESTSDIR}/policy.json -f Dockerfile.add1  ${TESTSDIR}/bud/cache-chown
  run_buildah bud --iidfile tar1  --layers --quiet --signature-policy ${TESTSDIR}/policy.json -f Dockerfile.tar1  ${TESTSDIR}/bud/cache-chown
  run_buildah bud --iidfile url1  --layers --quiet --signature-policy ${TESTSDIR}/policy.json -f Dockerfile.url1  ${TESTSDIR}/bud/cache-chown
  run_buildah bud --iidfile copy2 --layers --quiet --signature-policy ${TESTSDIR}/policy.json -f Dockerfile.copy2 ${TESTSDIR}/bud/cache-chown
  run_buildah bud --iidfile prev2 --layers --quiet --signature-policy ${TESTSDIR}/policy.json -f Dockerfile.prev2 ${TESTSDIR}/bud/cache-chown
  run_buildah bud --iidfile add2  --layers --quiet --signature-policy ${TESTSDIR}/policy.json -f Dockerfile.add2  ${TESTSDIR}/bud/cache-chown
  run_buildah bud --iidfile tar2  --layers --quiet --signature-policy ${TESTSDIR}/policy.json -f Dockerfile.tar2  ${TESTSDIR}/bud/cache-chown
  run_buildah bud --iidfile url2  --layers --quiet --signature-policy ${TESTSDIR}/policy.json -f Dockerfile.url2  ${TESTSDIR}/bud/cache-chown
  run_buildah bud --iidfile copy3 --layers --quiet --signature-policy ${TESTSDIR}/policy.json -f Dockerfile.copy1 ${TESTSDIR}/bud/cache-chown
  run_buildah bud --iidfile prev3 --layers --quiet --signature-policy ${TESTSDIR}/policy.json -f Dockerfile.prev1 ${TESTSDIR}/bud/cache-chown
  run_buildah bud --iidfile add3  --layers --quiet --signature-policy ${TESTSDIR}/policy.json -f Dockerfile.add1  ${TESTSDIR}/bud/cache-chown
  run_buildah bud --iidfile tar3  --layers --quiet --signature-policy ${TESTSDIR}/policy.json -f Dockerfile.tar1  ${TESTSDIR}/bud/cache-chown
  run_buildah bud --iidfile url3  --layers --quiet --signature-policy ${TESTSDIR}/policy.json -f Dockerfile.url1  ${TESTSDIR}/bud/cache-chown

  # The third round of builds should match all of the first rounds by way of caching.
  cmp copy1 copy3
  cmp prev1 prev3
  cmp add1  add3
  cmp tar1  tar3
  cmp url1  url3

  # The second round of builds should not match the first rounds, since the different ownership
  # makes the changes look different to the cache, except for cases where we extract an archive,
  # where --chown is ignored.
  run cmp copy1 copy2
  [[ "$status" -ne 0 ]]
  run cmp prev1 prev2
  [[ "$status" -ne 0 ]]
  run cmp add1  add2
  [[ "$status" -ne 0 ]]
  cmp tar1 tar2
  run cmp url1  url2
  [[ "$status" -ne 0 ]]

  # The first rounds of builds should all be different from each other, as a sanith thing.
  run cmp copy1 prev1
  [[ "$status" -ne 0 ]]
  run cmp copy1 add1
  [[ "$status" -ne 0 ]]
  run cmp copy1 tar1
  [[ "$status" -ne 0 ]]
  run cmp copy1 url1
  [[ "$status" -ne 0 ]]

  run cmp prev1 add1
  [[ "$status" -ne 0 ]]
  run cmp prev1 tar1
  [[ "$status" -ne 0 ]]
  run cmp prev1 url1
  [[ "$status" -ne 0 ]]

  run cmp add1 tar1
  [[ "$status" -ne 0 ]]
  run cmp add1 url1
  [[ "$status" -ne 0 ]]

  run cmp tar1 url1
  [[ "$status" -ne 0 ]]
}

@test "bud-terminal" {
  run_buildah bud ${TESTSDIR}/bud/terminal
}

@test "bud --ignore containerignore" {
  _prefetch alpine busybox

  CONTEXTDIR=${TESTDIR}/dockerignore
  cp -r ${TESTSDIR}/bud/dockerignore ${CONTEXTDIR}
  mv ${CONTEXTDIR}/.dockerignore ${TESTDIR}/containerignore

  run_buildah bud -t testbud --signature-policy ${TESTSDIR}/policy.json -f ${CONTEXTDIR}/Dockerfile.succeed --ignorefile  ${TESTDIR}/containerignore  ${CONTEXTDIR}

  run_buildah from --name myctr testbud

  run_buildah 1 run myctr ls -l test1.txt

  run_buildah run myctr ls -l test2.txt

  run_buildah 1 run myctr ls -l sub1.txt

  run_buildah 1 run myctr ls -l sub2.txt

  run_buildah run myctr ls -l subdir/sub1.txt

  run_buildah 1 run myctr ls -l subdir/sub2.txt
}

@test "bud with network options" {
  _prefetch alpine
  target=alpine-image

  run_buildah bud --network=none --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/containerfile
  [ "${status}" -eq 0 ]
  expect_output --substring "FROM alpine"

  run_buildah bud --network=private --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/containerfile
  [ "${status}" -eq 0 ]
  expect_output --substring "FROM alpine"

  run_buildah bud --network=container --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/containerfile
  [ "${status}" -eq 0 ]
  expect_output --substring "FROM alpine"

  run_buildah 125 bud --network=bogus --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/containerfile
  expect_output "error checking for network namespace: stat bogus: no such file or directory"
}
