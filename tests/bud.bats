#!/usr/bin/env bats

load helpers

@test "bud with --dns* flags" {
  run_buildah bud --log-level=error --dns-search=example.com --dns=223.5.5.5 --dns-option=use-vc  --signature-policy ${TESTSDIR}/policy.json -f ${TESTSDIR}/bud/dns/Dockerfile  ${TESTSDIR}/bud/dns
  expect_output --substring "search example.com"
  expect_output --substring "nameserver 223.5.5.5"
  expect_output --substring "options use-vc"

  buildah rm -a
  buildah rmi -a -f
}

@test "bud with .dockerignore" {
  # Remove containers and images before bud tests
  buildah rm --all
  buildah rmi -f --all

  run_buildah bud -t testbud --signature-policy ${TESTSDIR}/policy.json -f ${TESTSDIR}/bud/dockerignore/Dockerfile ${TESTSDIR}/bud/dockerignore

  run_buildah from --name myctr testbud

  run_buildah run myctr ls -l test2.txt

  run_buildah run myctr ls -l sub1.txt

  run_buildah 1 run myctr ls -l sub2.txt

  run_buildah run myctr ls -l subdir/sub1.txt

  run_buildah 1 run myctr ls -l subdir/sub2.txt

  run_buildah bud -t testbud2 --signature-policy ${TESTSDIR}/policy.json ${TESTSDIR}/bud/dockerignore2

  run_buildah bud -t testbud3 --signature-policy ${TESTSDIR}/policy.json ${TESTSDIR}/bud/dockerignore3
  expect_output --substring "CUT HERE"

  run sed -e '/^CUT HERE/,/^CUT HERE/p' -e 'd' <<< "$output"
  run sed '/CUT HERE/d' <<< "$output"
  expect_output "$(cat ${TESTSDIR}/bud/dockerignore3/manifest)"

  buildah rmi -a -f
}

@test "bud-flags-order-verification" {
  run_buildah 1 bud /tmp/tmpdockerfile/ -t blabla
  check_options_flag_err "-t"

  run_buildah 1 bud /tmp/tmpdockerfile/ -q -t blabla
  check_options_flag_err "-q"

  run_buildah 1 bud /tmp/tmpdockerfile/ --force-rm
  check_options_flag_err "--force-rm"

  run_buildah 1 bud /tmp/tmpdockerfile/ --userns=cnt1
  check_options_flag_err "--userns=cnt1"
}

@test "bud with --layers and --no-cache flags" {
  cp -a ${TESTSDIR}/bud/use-layers ${TESTDIR}/use-layers

  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t test1 ${TESTDIR}/use-layers
  run_buildah --log-level=error images -a
  expect_line_count 8
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t test2 ${TESTDIR}/use-layers
  run_buildah --log-level=error images -a
  expect_line_count 10
  run_buildah --log-level=error inspect --format "{{index .Docker.ContainerConfig.Env 1}}" test1
  expect_output "foo=bar"
  run_buildah --log-level=error inspect --format "{{index .Docker.ContainerConfig.Env 1}}" test2
  expect_output "foo=bar"
  run_buildah --log-level=error inspect --format "{{.Docker.ContainerConfig.ExposedPorts}}" test1
  expect_output "map[8080/tcp:{}]"
  run_buildah --log-level=error inspect --format "{{.Docker.ContainerConfig.ExposedPorts}}" test2
  expect_output "map[8080/tcp:{}]"

  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t test3 -f Dockerfile.2 ${TESTDIR}/use-layers
  run_buildah --log-level=error images -a
  expect_line_count 12

  mkdir -p ${TESTDIR}/use-layers/mount/subdir
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t test4 -f Dockerfile.3 ${TESTDIR}/use-layers
  run_buildah --log-level=error images -a
  expect_line_count 14

  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t test5 -f Dockerfile.3 ${TESTDIR}/use-layers
  run_buildah --log-level=error images -a
  expect_line_count 15

  touch ${TESTDIR}/use-layers/mount/subdir/file.txt
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t test6 -f Dockerfile.3 ${TESTDIR}/use-layers
  run_buildah --log-level=error images -a
  expect_line_count 17

  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --no-cache -t test7 -f Dockerfile.2 ${TESTDIR}/use-layers
  run_buildah --log-level=error images -a
  expect_line_count 18

  buildah rmi -a -f
}

@test "bud with --layers and single and two line Dockerfiles" {
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t test -f Dockerfile.5 ${TESTSDIR}/bud/use-layers
  run_buildah --log-level=error images -a
  expect_line_count 3

  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t test1 -f Dockerfile.6 ${TESTSDIR}/bud/use-layers
  run_buildah --log-level=error images -a
  expect_line_count 4

  buildah rmi -a -f
}

@test "bud with --layers, multistage, and COPY with --from" {
  cp -a ${TESTSDIR}/bud/use-layers ${TESTDIR}/use-layers

  mkdir -p ${TESTDIR}/use-layers/uuid
  uuidgen > ${TESTDIR}/use-layers/uuid/data
  mkdir -p ${TESTDIR}/use-layers/date
  date > ${TESTDIR}/use-layers/date/data

  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t test1 -f Dockerfile.multistage-copy ${TESTDIR}/use-layers
  run_buildah --log-level=error images -a
  expect_line_count 6
  # The second time through, the layers should all get reused.
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t test1 -f Dockerfile.multistage-copy ${TESTDIR}/use-layers
  run_buildah --log-level=error images -a
  expect_line_count 6
  # The third time through, the layers should all get reused, but we'll have a new line of output for the new name.

  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t test2 -f Dockerfile.multistage-copy ${TESTDIR}/use-layers
  run_buildah --log-level=error images -a
  expect_line_count 7

  # Both interim images will be different, and all of the layers in the final image will be different.
  uuidgen > ${TESTDIR}/use-layers/uuid/data
  date > ${TESTDIR}/use-layers/date/data
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t test3 -f Dockerfile.multistage-copy ${TESTDIR}/use-layers
  run_buildah --log-level=error images -a
  expect_line_count 11
  # No leftover containers, just the header line.
  run_buildah --log-level=error containers
  expect_line_count 1

  ctr=$(buildah --log-level=error from --signature-policy ${TESTSDIR}/policy.json test3)
  mnt=$(buildah --log-level=error mount ${ctr})
  run test -e $mnt/uuid
  [ "${status}" -eq 0 ]
  run test -e $mnt/date
  [ "${status}" -eq 0 ]

  # Layers won't get reused because this build won't use caching.
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t test4 -f Dockerfile.multistage-copy ${TESTDIR}/use-layers
  run_buildah --log-level=error images -a
  expect_line_count 12

  buildah rmi -a -f
}

@test "bud-multistage-partial-cache" {
  target=foo
  # build the first stage
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -f ${TESTSDIR}/bud/cache-stages/Dockerfile.1 ${TESTSDIR}/bud/cache-stages
  # expect alpine + 1 image record for the first stage
  run_buildah --log-level=error images -a
  expect_line_count 3
  # build the second stage, itself not cached, when the first stage is found in the cache
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -f ${TESTSDIR}/bud/cache-stages/Dockerfile.2 -t ${target} ${TESTSDIR}/bud/cache-stages
  # expect alpine + 1 image record for the first stage, then two more image records for the second stage
  run_buildah --log-level=error images -a
  expect_line_count 5
}

@test "bud-multistage-copy-final-slash" {
  target=foo
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/dest-final-slash
  run_buildah --log-level=error from --signature-policy ${TESTSDIR}/policy.json ${target}
  cid="$output"
  run_buildah run ${cid} /test/ls -lR /test/ls
}

@test "bud-multistage-reused" {
  target=foo
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/multi-stage-builds/Dockerfile.reused ${TESTSDIR}/bud/multi-stage-builds
  run_buildah from --signature-policy ${TESTSDIR}/policy.json ${target}
  run_buildah rmi -f ${target}
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} --layers -f ${TESTSDIR}/bud/multi-stage-builds/Dockerfile.reused ${TESTSDIR}/bud/multi-stage-builds
  run_buildah from --signature-policy ${TESTSDIR}/policy.json ${target}
}

@test "bud-multistage-cache" {
  target=foo
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/multi-stage-builds/Dockerfile.extended ${TESTSDIR}/bud/multi-stage-builds
  run_buildah --log-level=error from --signature-policy ${TESTSDIR}/policy.json ${target}
  cid="$output"
  run_buildah --log-level=error mount "$cid"
  root="$output"
  # cache should have used this one
  test -r "$root"/tmp/preCommit
  # cache should not have used this one
  ! test -r "$root"/tmp/postCommit
}

@test "bud with --layers and symlink file" {
  cp -a ${TESTSDIR}/bud/use-layers ${TESTDIR}/use-layers
  echo 'echo "Hello World!"' > ${TESTDIR}/use-layers/hello.sh
  ln -s hello.sh ${TESTDIR}/use-layers/hello_world.sh
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t test -f Dockerfile.4 ${TESTDIR}/use-layers
  run_buildah --log-level=error images -a
  expect_line_count 4

  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t test1 -f Dockerfile.4 ${TESTDIR}/use-layers
  run_buildah --log-level=error images -a
  expect_line_count 5

  echo 'echo "Hello Cache!"' > ${TESTDIR}/use-layers/hello.sh
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t test2 -f Dockerfile.4 ${TESTDIR}/use-layers
  run_buildah --log-level=error images -a
  expect_line_count 7

  buildah rmi -a -f
}

@test "bud with --layers and dangling symlink" {
  cp -a ${TESTSDIR}/bud/use-layers ${TESTDIR}/use-layers
  mkdir ${TESTDIR}/use-layers/blah
  ln -s ${TESTSDIR}/policy.json ${TESTDIR}/use-layers/blah/policy.json

  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t test -f Dockerfile.dangling-symlink ${TESTDIR}/use-layers
  run_buildah --log-level=error images -a
  expect_line_count 3

  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t test1 -f Dockerfile.dangling-symlink ${TESTDIR}/use-layers
  run_buildah --log-level=error images -a
  expect_line_count 4

  cid=$(buildah from --signature-policy ${TESTSDIR}/policy.json test)
  run_buildah --log-level=error run $cid ls /tmp
  expect_output "policy.json"

  buildah rm -a
  buildah rmi -a -f
  rm -rf ${TESTDIR}/use-layers/blah
}

@test "bud with --layers and --build-args" {
  # base plus 3, plus the header line
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --build-arg=user=0 --layers -t test -f Dockerfile.build-args ${TESTSDIR}/bud/use-layers
  run_buildah --log-level=error images -a
  expect_line_count 5

  # two more, starting at the "echo $user" instruction
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --build-arg=user=1 --layers -t test1 -f Dockerfile.build-args ${TESTSDIR}/bud/use-layers
  run_buildah --log-level=error images -a
  expect_line_count 7

  # one more, because we added a new name to the same image
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --build-arg=user=1 --layers -t test2 -f Dockerfile.build-args ${TESTSDIR}/bud/use-layers
  run_buildah --log-level=error images -a
  expect_line_count 8

  # two more, starting at the "echo $user" instruction
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t test3 -f Dockerfile.build-args ${TESTSDIR}/bud/use-layers
  run_buildah --log-level=error images -a
  expect_line_count 10

  buildah rmi -a -f
}

@test "bud with --rm flag" {
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t test1 ${TESTSDIR}/bud/use-layers
  run_buildah --log-level=error containers
  expect_line_count 1

  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --rm=false --layers -t test2 ${TESTSDIR}/bud/use-layers
  run_buildah --log-level=error containers
  expect_line_count 7

  buildah rm -a
  buildah rmi -a -f
}

@test "bud with --force-rm flag" {
  run_buildah 1 bud --signature-policy ${TESTSDIR}/policy.json --force-rm --layers -t test1 -f Dockerfile.fail-case ${TESTSDIR}/bud/use-layers
  run_buildah --log-level=error containers
  expect_line_count 1

  run_buildah 1 bud --signature-policy ${TESTSDIR}/policy.json --layers -t test2 -f Dockerfile.fail-case ${TESTSDIR}/bud/use-layers
  run_buildah --log-level=error containers
  expect_line_count 2

  buildah rm -a
  buildah rmi -a -f
}

@test "bud --layers with non-existent/down registry" {
  run_buildah 1 bud --signature-policy ${TESTSDIR}/policy.json --force-rm --layers -t test1 -f Dockerfile.non-existent-registry ${TESTSDIR}/bud/use-layers
  expect_output --substring "no such host"
}

@test "bud from base image should have base image ENV also" {
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t test -f Dockerfile.check-env ${TESTSDIR}/bud/env
  cid=$(buildah from --signature-policy ${TESTSDIR}/policy.json test)
  run_buildah config --env random=hello,goodbye ${cid}
  run_buildah commit --signature-policy ${TESTSDIR}/policy.json ${cid} test1
  run_buildah --log-level=error inspect --format '{{index .Docker.ContainerConfig.Env 1}}' test1
  expect_output "foo=bar"
  run_buildah --log-level=error inspect --format '{{index .Docker.ContainerConfig.Env 2}}' test1
  expect_output "random=hello,goodbye"
  buildah rm ${cid}
  buildah rmi -a -f
}

@test "bud-from-scratch" {
  target=scratch-image
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/from-scratch
  cid=$(buildah from ${target})
  buildah rm ${cid}
  buildah rmi $(buildah --log-level=error images -q)
  run_buildah --log-level=error images -q
  expect_output ""
}

@test "bud-from-scratch-iid" {
  target=scratch-image
  run_buildah bud --iidfile ${TESTDIR}/output.iid --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/from-scratch
  iid=$(cat ${TESTDIR}/output.iid)
  cid=$(buildah from ${iid})
  buildah rm ${cid}
  buildah rmi $(buildah --log-level=error images -q)
  run_buildah --log-level=error images -q
  expect_output ""
}

@test "bud-from-scratch-label" {
  target=scratch-image
  run_buildah bud --label "test=label" --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/from-scratch
  run_buildah --log-level=error inspect --format '{{printf "%q" .Docker.Config.Labels}}' ${target}
  expect_output 'map["test":"label"]'

  buildah rmi ${target}
}

@test "bud-from-scratch-annotation" {
  target=scratch-image
  run_buildah bud --annotation "test=annotation1,annotation2=z" --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/from-scratch
  run_buildah --log-level=error inspect --format '{{printf "%q" .ImageAnnotations}}' ${target}
  expect_output 'map["test":"annotation1,annotation2=z"]'
  buildah rmi ${target}
}

@test "bud-from-scratch-layers" {
  target=scratch-image
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -f  ${TESTSDIR}/bud/from-scratch/Dockerfile2 -t ${target} ${TESTSDIR}/bud/from-scratch
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -f  ${TESTSDIR}/bud/from-scratch/Dockerfile2 -t ${target} ${TESTSDIR}/bud/from-scratch
  cid=$(buildah from ${target})
  run_buildah --log-level=error images
  expect_line_count 3
  run_buildah rm ${cid}
  expect_line_count 1
}

@test "bud-from-multiple-files-one-from" {
  target=scratch-image
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/from-multiple-files/Dockerfile1.scratch -f ${TESTSDIR}/bud/from-multiple-files/Dockerfile2.nofrom ${TESTSDIR}/bud/from-multiple-files
  cid=$(buildah from ${target})
  root=$(buildah mount ${cid})
  cmp $root/Dockerfile1 ${TESTSDIR}/bud/from-multiple-files/Dockerfile1.scratch
  cmp $root/Dockerfile2.nofrom ${TESTSDIR}/bud/from-multiple-files/Dockerfile2.nofrom
  test ! -s $root/etc/passwd
  buildah rm ${cid}
  buildah rmi $(buildah --log-level=error images -q)
  run_buildah --log-level=error images -q
  expect_output ""

  target=alpine-image
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f Dockerfile1.alpine -f Dockerfile2.nofrom ${TESTSDIR}/bud/from-multiple-files
  cid=$(buildah from ${target})
  root=$(buildah mount ${cid})
  cmp $root/Dockerfile1 ${TESTSDIR}/bud/from-multiple-files/Dockerfile1.alpine
  cmp $root/Dockerfile2.nofrom ${TESTSDIR}/bud/from-multiple-files/Dockerfile2.nofrom
  test -s $root/etc/passwd
  buildah rm ${cid}
  buildah rmi -a
  run_buildah --log-level=error images -q
  expect_output ""
}

@test "bud-from-multiple-files-two-froms" {
  target=scratch-image
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f Dockerfile1.scratch -f Dockerfile2.withfrom ${TESTSDIR}/bud/from-multiple-files
  cid=$(buildah from ${target})
  root=$(buildah mount ${cid})
  test ! -s $root/Dockerfile1
  cmp $root/Dockerfile2.withfrom ${TESTSDIR}/bud/from-multiple-files/Dockerfile2.withfrom
  test -s $root/etc/passwd
  buildah rm ${cid}
  buildah rmi -a
  run_buildah --log-level=error images -q
  expect_output ""

  target=alpine-image
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f Dockerfile1.alpine -f Dockerfile2.withfrom ${TESTSDIR}/bud/from-multiple-files
  cid=$(buildah from ${target})
  root=$(buildah mount ${cid})
  test ! -s $root/Dockerfile1
  cmp $root/Dockerfile2.withfrom ${TESTSDIR}/bud/from-multiple-files/Dockerfile2.withfrom
  test -s $root/etc/passwd
  buildah rm ${cid}
  buildah rmi -a
  run_buildah --log-level=error images -q
  echo "$output"
  expect_output ""
}

@test "bud-multi-stage-builds" {
  target=multi-stage-index
  buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/multi-stage-builds/Dockerfile.index ${TESTSDIR}/bud/multi-stage-builds
  cid=$(buildah from ${target})
  root=$(buildah mount ${cid})
  cmp $root/Dockerfile.index ${TESTSDIR}/bud/multi-stage-builds/Dockerfile.index
  test -s $root/etc/passwd
  buildah rm ${cid}
  buildah rmi -a
  run_buildah --log-level=error images -q
  expect_output ""

  target=multi-stage-name
  buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f Dockerfile.name ${TESTSDIR}/bud/multi-stage-builds
  cid=$(buildah from ${target})
  root=$(buildah mount ${cid})
  cmp $root/Dockerfile.name ${TESTSDIR}/bud/multi-stage-builds/Dockerfile.name
  test ! -s $root/etc/passwd
  buildah rm ${cid}
  buildah rmi $(buildah --log-level=error images -q)
  run_buildah --log-level=error images -q
  expect_output ""

  target=multi-stage-mixed
  buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/multi-stage-builds/Dockerfile.mixed ${TESTSDIR}/bud/multi-stage-builds
  cid=$(buildah from ${target})
  root=$(buildah mount ${cid})
  cmp $root/Dockerfile.name ${TESTSDIR}/bud/multi-stage-builds/Dockerfile.name
  cmp $root/Dockerfile.index ${TESTSDIR}/bud/multi-stage-builds/Dockerfile.index
  cmp $root/Dockerfile.mixed ${TESTSDIR}/bud/multi-stage-builds/Dockerfile.mixed
  buildah rm ${cid}
  buildah rmi $(buildah --log-level=error images -q)
  run_buildah --log-level=error images -q
  expect_output ""
}

@test "bud-multi-stage-builds-small-as" {
  target=multi-stage-index
  buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/multi-stage-builds-small-as/Dockerfile.index ${TESTSDIR}/bud/multi-stage-builds-small-as
  cid=$(buildah from ${target})
  root=$(buildah mount ${cid})
  cmp $root/Dockerfile.index ${TESTSDIR}/bud/multi-stage-builds-small-as/Dockerfile.index
  test -s $root/etc/passwd
  buildah rm ${cid}
  buildah rmi -a
  run_buildah --log-level=error images -q
  expect_output ""

  target=multi-stage-name
  buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f Dockerfile.name ${TESTSDIR}/bud/multi-stage-builds-small-as
  cid=$(buildah from ${target})
  root=$(buildah mount ${cid})
  cmp $root/Dockerfile.name ${TESTSDIR}/bud/multi-stage-builds-small-as/Dockerfile.name
  test ! -s $root/etc/passwd
  buildah rm ${cid}
  buildah rmi $(buildah --log-level=error images -q)
  run_buildah --log-level=error images -q
  expect_output ""

  target=multi-stage-mixed
  buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/multi-stage-builds-small-as/Dockerfile.mixed ${TESTSDIR}/bud/multi-stage-builds-small-as
  cid=$(buildah from ${target})
  root=$(buildah mount ${cid})
  cmp $root/Dockerfile.name ${TESTSDIR}/bud/multi-stage-builds-small-as/Dockerfile.name
  cmp $root/Dockerfile.index ${TESTSDIR}/bud/multi-stage-builds-small-as/Dockerfile.index
  cmp $root/Dockerfile.mixed ${TESTSDIR}/bud/multi-stage-builds-small-as/Dockerfile.mixed
  buildah rm ${cid}
  buildah rmi $(buildah --log-level=error images -q)
  run_buildah --log-level=error images -q
  expect_output ""
}

@test "bud-preserve-subvolumes" {
  # This Dockerfile needs us to be able to handle a working RUN instruction.
  skip_if_no_runtime

  target=volume-image
  buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/preserve-volumes
  cid=$(buildah from ${target})
  root=$(buildah mount ${cid})
  test -s $root/vol/subvol/subsubvol/subsubvolfile
  test ! -s $root/vol/subvol/subvolfile
  test -s $root/vol/volfile
  test -s $root/vol/Dockerfile
  test -s $root/vol/Dockerfile2
  test ! -s $root/vol/anothervolfile
  buildah rm ${cid}
  buildah rmi -a
  run_buildah --log-level=error images -q
  expect_output ""
}

@test "bud-http-Dockerfile" {
  starthttpd ${TESTSDIR}/bud/from-scratch
  target=scratch-image
  buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} http://0.0.0.0:${HTTP_SERVER_PORT}/Dockerfile
  stophttpd
  cid=$(buildah from ${target})
  buildah rm ${cid}
  buildah rmi $(buildah --log-level=error images -q)
  run_buildah --log-level=error images -q
  expect_output ""
}

@test "bud-http-context-with-Dockerfile" {
  starthttpd ${TESTSDIR}/bud/http-context
  target=scratch-image
  buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} http://0.0.0.0:${HTTP_SERVER_PORT}/context.tar
  stophttpd
  cid=$(buildah from ${target})
  buildah rm ${cid}
  buildah rmi $(buildah --log-level=error images -q)
  run_buildah --log-level=error images -q
  expect_output ""
}

@test "bud-http-context-dir-with-Dockerfile-pre" {
  starthttpd ${TESTSDIR}/bud/http-context-subdir
  target=scratch-image
  buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f context/Dockerfile http://0.0.0.0:${HTTP_SERVER_PORT}/context.tar
  stophttpd
  cid=$(buildah from ${target})
  buildah rm ${cid}
  buildah rmi $(buildah --log-level=error images -q)
  run_buildah --log-level=error images -q
  expect_output ""
}

@test "bud-http-context-dir-with-Dockerfile-post" {
  starthttpd ${TESTSDIR}/bud/http-context-subdir
  target=scratch-image
  buildah bud  --signature-policy ${TESTSDIR}/policy.json -t ${target} -f context/Dockerfile http://0.0.0.0:${HTTP_SERVER_PORT}/context.tar
  stophttpd
  cid=$(buildah from ${target})
  buildah rm ${cid}
  buildah rmi $(buildah --log-level=error images -q)
  run_buildah --log-level=error images -q
  expect_output ""
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
  buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} "${gitrepo}"
  cid=$(buildah from ${target})
  buildah rm ${cid}
  buildah rmi $(buildah --log-level=error images -q)
  run_buildah --log-level=error images -q
  expect_output ""
}

@test "bud-github-context" {
  target=github-image
  # Any repo should do, but this one is small and is FROM: scratch.
  gitrepo=github.com/projectatomic/nulecule-library
  buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} "${gitrepo}"
  cid=$(buildah from ${target})
  buildah rm ${cid}
  buildah --log-level=error images -q
  buildah rmi $(buildah --log-level=error images -q)
  run_buildah --log-level=error images -q
  expect_output ""
}

@test "bud-additional-tags" {
  target=scratch-image
  target2=another-scratch-image
  target3=so-many-scratch-images
  buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -t docker.io/${target2} -t ${target3} ${TESTSDIR}/bud/from-scratch
  run_buildah --log-level=error images
  cid=$(buildah from ${target})
  buildah rm ${cid}
  cid=$(buildah from --signature-policy ${TESTSDIR}/policy.json library/${target2})
  buildah rm ${cid}
  cid=$(buildah from --signature-policy ${TESTSDIR}/policy.json ${target3}:latest)
  buildah rm ${cid}
  buildah rmi -f $(buildah --log-level=error images -q)
  run_buildah --log-level=error images -q
  expect_output ""
}

@test "bud-additional-tags-cached" {
  target=tagged-image
  target2=another-tagged-image
  target3=yet-another-tagged-image
  target4=still-another-tagged-image
  run_buildah bud --layers --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/addtl-tags
  run_buildah bud --layers --signature-policy ${TESTSDIR}/policy.json -t ${target2} -t ${target3} -t ${target4} ${TESTSDIR}/bud/addtl-tags
  run_buildah --log-level=error inspect -f '{{.FromImageID}}' busybox
  busyboxid="$output"
  run_buildah --log-level=error inspect -f '{{.FromImageID}}' ${target}
  targetid="$output"
  [ "$targetid" != "$busyboxid" ]
  run_buildah --log-level=error inspect -f '{{.FromImageID}}' ${target2}
  targetid2="$output"
  [ "$targetid2" = "$targetid" ]
  run_buildah --log-level=error inspect -f '{{.FromImageID}}' ${target3}
  targetid3="$output"
  [ "$targetid3" = "$targetid2" ]
  run_buildah --log-level=error inspect -f '{{.FromImageID}}' ${target4}
  targetid4="$output"
  [ "$targetid4" = "$targetid3" ]
}

@test "bud-volume-perms" {
  # This Dockerfile needs us to be able to handle a working RUN instruction.
  skip_if_no_runtime

  target=volume-image
  buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/volume-perms
  cid=$(buildah from --signature-policy ${TESTSDIR}/policy.json ${target})
  root=$(buildah mount ${cid})
  test ! -s $root/vol/subvol/subvolfile
  run stat -c %f $root/vol/subvol
  echo "$output"
  [ "$status" -eq 0 ]
  [ "$output" = 41ed ]
  buildah rm ${cid}
  buildah rmi -a
  run_buildah --log-level=error images -q
  expect_output ""
}

@test "bud-from-glob" {
  target=alpine-image
  buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f Dockerfile2.glob ${TESTSDIR}/bud/from-multiple-files
  cid=$(buildah from --signature-policy ${TESTSDIR}/policy.json ${target})
  root=$(buildah mount ${cid})
  cmp $root/Dockerfile1.alpine ${TESTSDIR}/bud/from-multiple-files/Dockerfile1.alpine
  cmp $root/Dockerfile2.withfrom ${TESTSDIR}/bud/from-multiple-files/Dockerfile2.withfrom
  buildah rm ${cid}
  buildah rmi -a
  run_buildah --log-level=error images -q
  expect_output ""
}

@test "bud-maintainer" {
  target=alpine-image
  buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/maintainer
  run_buildah --log-level=error inspect --type=image --format '{{.Docker.Author}}' ${target}
  expect_output "kilroy"
  run_buildah --log-level=error inspect --type=image --format '{{.OCIv1.Author}}' ${target}
  expect_output "kilroy"
  buildah rmi -a
  run_buildah --log-level=error images -q
  expect_output ""
}

@test "bud-unrecognized-instruction" {
  target=alpine-image
  run_buildah 1 bud --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/unrecognized
  expect_output --substring "BOGUS"
  buildah rmi $(buildah --log-level=error images -q)
  run_buildah --log-level=error images -q
  expect_output ""
}

@test "bud-shell" {
  target=alpine-image
  buildah bud --format docker --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/shell
  run_buildah --log-level=error inspect --type=image --format '{{printf "%q" .Docker.Config.Shell}}' ${target}
  expect_output '["/bin/sh" "-c"]' ".Docker.Config.Shell (original)"
  ctr=$(buildah from --signature-policy ${TESTSDIR}/policy.json ${target})
  run_buildah --log-level=error config --shell "/bin/bash -c" ${ctr}
  run_buildah --log-level=error inspect --type=container --format '{{printf "%q" .Docker.Config.Shell}}' ${ctr}
  expect_output '["/bin/bash" "-c"]' ".Docker.Config.Shell (changed)"
  buildah rm ${ctr}
  buildah rmi -a
  run_buildah --log-level=error images -q
  expect_output ""
}

@test "bud-shell during build in Docker format" {
  target=alpine-image
  run_buildah bud --format docker --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/shell/Dockerfile.build-shell-default ${TESTSDIR}/bud/shell
  expect_output --substring "SHELL=/bin/sh"
  buildah rmi -a
  run_buildah --log-level=error images -q
  expect_output ""
}

@test "bud-shell during build in OCI format" {
  target=alpine-image
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/shell/Dockerfile.build-shell-default ${TESTSDIR}/bud/shell
  expect_output --substring "SHELL=/bin/sh"
  buildah rmi -a
  run_buildah --log-level=error images -q
  expect_output ""
}

@test "bud-shell changed during build in Docker format" {
  target=ubuntu-image
  run_buildah bud --format docker --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/shell/Dockerfile.build-shell-custom ${TESTSDIR}/bud/shell
  expect_output --substring "SHELL=/bin/bash"
  buildah rmi -a
  run_buildah --log-level=error images -q
  expect_output ""
}

@test "bud-shell changed during build in OCI format" {
  target=ubuntu-image
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/shell/Dockerfile.build-shell-custom ${TESTSDIR}/bud/shell
  expect_output --substring "SHELL=/bin/sh"
  buildah rmi -a
  run_buildah --log-level=error images -q
  expect_output ""
}

@test "bud with symlinks" {
  target=alpine-image
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/symlink
  cid=$(buildah from --signature-policy ${TESTSDIR}/policy.json ${target})
  root=$(buildah mount ${cid})
  run ls $root/data/log
  echo "$output"
  [ "$status" -eq 0 ]
  [[ "$output" =~ test ]]
  [[ "$output" =~ blah.txt ]]
  run ls -al $root
  echo "$output"
  [ "$status" -eq 0 ]
  [[ "$output" =~ "test-log -> /data/log" ]]
  [[ "$output" =~ "blah -> /test-log" ]]
  buildah rm ${cid}
  buildah rmi ${target}
}

@test "bud with symlinks to relative path" {
  target=alpine-image
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f Dockerfile.relative-symlink ${TESTSDIR}/bud/symlink
  cid=$(buildah from --signature-policy ${TESTSDIR}/policy.json ${target})
  root=$(buildah mount ${cid})
  run ls $root/log
  echo "$output"
  [ "$status" -eq 0 ]
  [[ "$output" =~ test ]]
  run ls -al $root
  echo "$output"
  [ "$status" -eq 0 ]
  [[ "$output" =~ "test-log -> ../log" ]]
  test -r $root/var/data/empty
  buildah rm ${cid}
  buildah rmi ${target}
}

@test "bud with multiple symlinks in a path" {
  target=alpine-image
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/symlink/Dockerfile.multiple-symlinks ${TESTSDIR}/bud/symlink
  cid=$(buildah from --signature-policy ${TESTSDIR}/policy.json ${target})
  root=$(buildah mount ${cid})
  run ls $root/data/log
  echo "$output"
  [ "$status" -eq 0 ]
  [[ "$output" =~ bin ]]
  [[ "$output" =~ blah.txt ]]
  run ls -al $root/myuser
  echo "$output"
  [ "$status" -eq 0 ]
  [[ "$output" =~ "log -> /test" ]]
  run ls -al $root/test
  echo "$output"
  [ "$status" -eq 0 ]
  [[ "$output" =~ "bar -> /test-log" ]]
  run ls -al $root/test-log
  echo "$output"
  [ "$status" -eq 0 ]
  [[ "$output" =~ "foo -> /data/log" ]]
  buildah rm ${cid}
  buildah rmi ${target}
}

@test "bud with multiple symlink pointing to itself" {
  target=alpine-image
  run_buildah 1 bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/symlink/Dockerfile.symlink-points-to-itself ${TESTSDIR}/bud/symlink
}

@test "bud multi-stage with symlink to absolute path" {
  target=ubuntu-image
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f Dockerfile.absolute-symlink ${TESTSDIR}/bud/symlink
  cid=$(buildah from --signature-policy ${TESTSDIR}/policy.json ${target})
  root=$(buildah mount ${cid})
  run ls $root/bin
  echo "$output"
  [ "$status" -eq 0 ]
  [[ "$output" =~ myexe ]]
  run cat $root/bin/myexe
  [ "$status" -eq 0 ]
  [[ "$output" == "symlink-test" ]]
  buildah rm ${cid}
  buildah rmi ${target}
}

@test "bud multi-stage with dir symlink to absolute path" {
  target=ubuntu-image
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f Dockerfile.absolute-dir-symlink ${TESTSDIR}/bud/symlink
  cid=$(buildah from --signature-policy ${TESTSDIR}/policy.json ${target})
  root=$(buildah mount ${cid})
  run ls $root/data
  echo "$output"
  [ "$status" -eq 0 ]
  [[ "$output" =~ myexe ]]
  buildah rm ${cid}
  buildah rmi ${target}
}

@test "bud with ENTRYPOINT and RUN" {
  target=alpine-image
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f Dockerfile.entrypoint-run ${TESTSDIR}/bud/run-scenarios
  expect_output --substring "unique.test.string"
  cid=$(buildah from --signature-policy ${TESTSDIR}/policy.json ${target})
  buildah rm ${cid}
  buildah rmi ${target}
}

@test "bud with ENTRYPOINT and empty RUN" {
  target=alpine-image
  run_buildah 1 bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f Dockerfile.entrypoint-empty-run ${TESTSDIR}/bud/run-scenarios
  expect_output --substring "error building at STEP"
}

@test "bud with CMD and RUN" {
  target=alpine-image
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/run-scenarios/Dockerfile.cmd-run ${TESTSDIR}/bud/run-scenarios
  expect_output --substring "unique.test.string"
  cid=$(buildah from --signature-policy ${TESTSDIR}/policy.json ${target})
  buildah rm ${cid}
  buildah rmi ${target}
}

@test "bud with CMD and empty RUN" {
  target=alpine-image
  run_buildah 1 bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f Dockerfile.cmd-empty-run ${TESTSDIR}/bud/run-scenarios
  expect_output --substring "error building at STEP"
}

@test "bud with ENTRYPOINT, CMD and RUN" {
  target=alpine-image
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/run-scenarios/Dockerfile.entrypoint-cmd-run ${TESTSDIR}/bud/run-scenarios
  expect_output --substring "unique.test.string"
  cid=$(buildah from --signature-policy ${TESTSDIR}/policy.json ${target})
  buildah rm ${cid}
  buildah rmi ${target}
}

@test "bud with ENTRYPOINT, CMD and empty RUN" {
  target=alpine-image
  run_buildah 1 bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/run-scenarios/Dockerfile.entrypoint-cmd-empty-run ${TESTSDIR}/bud/run-scenarios
  expect_output --substring "error building at STEP"
}

# Determines if a variable set with ENV is available to following commands in the Dockerfile
@test "bud access ENV variable defined in same source file" {
  target=env-image
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/env/Dockerfile.env-same-file ${TESTSDIR}/bud/env
  expect_output --substring ":unique.test.string:"
  cid=$(buildah from --signature-policy ${TESTSDIR}/policy.json ${target})
  buildah rm ${cid}
  buildah rmi ${target}
}

# Determines if a variable set with ENV in an image is available to commands in downstream Dockerfile
@test "bud access ENV variable defined in FROM image" {
  from_target=env-from-image
  target=env-image
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${from_target} -f ${TESTSDIR}/bud/env/Dockerfile.env-same-file ${TESTSDIR}/bud/env
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/env/Dockerfile.env-from-image ${TESTSDIR}/bud/env
  expect_output --substring "@unique.test.string@"
  from_cid=$(buildah from ${from_target})
  cid=$(buildah from ${target})
  buildah rm ${from_cid} ${cid}
  buildah rmi -a -f
}

@test "bud ENV preserves special characters after commit" {
  from_target=special-chars
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${from_target} -f ${TESTSDIR}/bud/env/Dockerfile.special-chars ${TESTSDIR}/bud/env
  cid=$(buildah from ${from_target})
  run_buildah run ${cid} env
  expect_output --substring "LIB=\\$\(PREFIX\)/lib"
  buildah rm ${cid}
  buildah rmi -a -f
}

@test "bud with Dockerfile from valid URL" {
  target=url-image
  url=https://raw.githubusercontent.com/containers/buildah/master/tests/bud/from-scratch/Dockerfile
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} ${url}
  cid=$(buildah from ${target})
  buildah rm ${cid}
  buildah rmi ${target}
}

@test "bud with Dockerfile from invalid URL" {
  target=url-image
  url=https://raw.githubusercontent.com/containers/buildah/master/tests/bud/from-scratch/Dockerfile.bogus
  run_buildah 1 bud --signature-policy ${TESTSDIR}/policy.json -t ${target} ${url}
}

# When provided with a -f flag and directory, buildah will look for the alternate Dockerfile name in the supplied directory
@test "bud with -f flag, alternate Dockerfile name" {
  target=fileflag-image
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f Dockerfile.noop-flags ${TESTSDIR}/bud/run-scenarios
  cid=$(buildah from ${target})
  buildah rm ${cid}
  buildah rmi ${target}
}

# Following flags are configured to result in noop but should not affect buildiah bud behavior
@test "bud with --cache-from noop flag" {
  target=noop-image
  run_buildah bud --cache-from=invalidimage --signature-policy ${TESTSDIR}/policy.json -t ${target} -f Dockerfile.noop-flags ${TESTSDIR}/bud/run-scenarios
  cid=$(buildah from ${target})
  buildah rm ${cid}
  buildah rmi ${target}
}

@test "bud with --compress noop flag" {
  target=noop-image
  run_buildah bud --compress --signature-policy ${TESTSDIR}/policy.json -t ${target} -f Dockerfile.noop-flags ${TESTSDIR}/bud/run-scenarios
  cid=$(buildah from ${target})
  buildah rm ${cid}
  buildah rmi ${target}
}

@test "bud with --cpu-shares flag, no argument" {
  target=bud-flag
  run_buildah 1 bud --cpu-shares --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/from-scratch/Dockerfile ${TESTSDIR}/bud/from-scratch
}

@test "bud with --cpu-shares flag, invalid argument" {
  target=bud-flag
  run_buildah 1 bud --cpu-shares bogus --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/from-scratch/Dockerfile ${TESTSDIR}/bud/from-scratch
  expect_output --substring "invalid argument \"bogus\" for "
}

@test "bud with --cpu-shares flag, valid argument" {
  target=bud-flag
  run_buildah bud --cpu-shares 2 --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/from-scratch/Dockerfile ${TESTSDIR}/bud/from-scratch
  cid=$(buildah from ${target})
  buildah rm ${cid}
  buildah rmi ${target}
}

@test "bud with --cpu-shares short flag (-c), no argument" {
  target=bud-flag
  run_buildah 1 bud -c --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/from-scratch/Dockerfile ${TESTSDIR}/bud/from-scratch
}

@test "bud with --cpu-shares short flag (-c), invalid argument" {
  target=bud-flag
  run_buildah 1 bud -c bogus --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/from-scratch/Dockerfile ${TESTSDIR}/bud/from-scratch
  expect_output --substring "invalid argument \"bogus\" for "
}

@test "bud with --cpu-shares short flag (-c), valid argument" {
  target=bud-flag
  run_buildah bud -c 2 --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/from-scratch
  cid=$(buildah from ${target})
  buildah rm ${cid}
  buildah rmi ${target}
}

@test "bud-onbuild" {
  target=onbuild
  buildah bud --format docker --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/onbuild
  run_buildah --log-level=error inspect --format '{{printf "%q" .Docker.Config.OnBuild}}' ${target}
  expect_output '["RUN touch /onbuild1" "RUN touch /onbuild2"]'
  cid=$(buildah from ${target})
  root=$(buildah mount ${cid})
  run ls ${root}/onbuild1 ${root}/onbuild2
  echo "$output"
  [ "$status" -eq 0 ]
  buildah umount ${cid}
  buildah rm ${cid}

  target=onbuild-image2
  buildah bud --format docker --signature-policy ${TESTSDIR}/policy.json -t ${target} -f Dockerfile1 ${TESTSDIR}/bud/onbuild
  run_buildah --log-level=error inspect --format '{{printf "%q" .Docker.Config.OnBuild}}' ${target}
  expect_output '["RUN touch /onbuild3"]'
  cid=$(buildah from ${target})
  root=$(buildah mount ${cid})
  run ls ${root}/onbuild1 ${root}/onbuild2 ${root}/onbuild3
  echo "$output"
  [ "$status" -eq 0 ]
  buildah umount ${cid}

  run_buildah --log-level=error config --onbuild "RUN touch /onbuild4" ${cid}

  target=onbuild-image3
  buildah commit --signature-policy ${TESTSDIR}/policy.json --format docker ${cid} ${target}
  run_buildah --log-level=error inspect --format '{{printf "%q" .Docker.Config.OnBuild}}' ${target}
  expect_output '["RUN touch /onbuild4"]'
  buildah rm ${cid}
  buildah rmi --all
}

@test "bud-onbuild-layers" {
  target=onbuild
  buildah bud --format docker --signature-policy ${TESTSDIR}/policy.json --layers -t ${target} -f Dockerfile2 ${TESTSDIR}/bud/onbuild
  run_buildah --log-level=error inspect --format '{{printf "%q" .Docker.Config.OnBuild}}' ${target}
  expect_output '["RUN touch /onbuild1" "RUN touch /onbuild2"]'
}

@test "bud-logfile" {
  rm -f ${TESTDIR}/logfile
  run_buildah bud --logfile ${TESTDIR}/logfile --signature-policy ${TESTSDIR}/policy.json ${TESTSDIR}/bud/preserve-volumes
  test -s ${TESTDIR}/logfile
}

@test "bud with ARGS" {
  target=alpine-image
  run_buildah --log-level=error bud -q --signature-policy ${TESTSDIR}/policy.json -t ${target} -f Dockerfile.args ${TESTSDIR}/bud/run-scenarios
  expect_output --substring "arg_value"
}

@test "bud with unused ARGS" {
  target=alpine-image
  run_buildah --log-level=error bud -q --signature-policy ${TESTSDIR}/policy.json -t ${target} -f Dockerfile.multi-args --build-arg USED_ARG=USED_VALUE ${TESTSDIR}/bud/run-scenarios
  expect_output --substring "USED_VALUE"
  [[ ! "$output" =~ "one or more build args were not consumed: [UNUSED_ARG]" ]]
  run_buildah --log-level=error bud -q --signature-policy ${TESTSDIR}/policy.json -t ${target} -f Dockerfile.multi-args --build-arg USED_ARG=USED_VALUE --build-arg UNUSED_ARG=whaaaat ${TESTSDIR}/bud/run-scenarios
  expect_output --substring "USED_VALUE"
  expect_output --substring "one or more build args were not consumed: \[UNUSED_ARG\]"
}

@test "bud with multi-value ARGS" {
  target=alpine-image
  run_buildah --log-level=error bud -q --signature-policy ${TESTSDIR}/policy.json -t ${target} -f Dockerfile.multi-args --build-arg USED_ARG=plugin1,plugin2,plugin3 ${TESTSDIR}/bud/run-scenarios
  expect_output --substring "plugin1,plugin2,plugin3"
  [[ ! "$output" =~ "one or more build args were not consumed: [UNUSED_ARG]" ]]
}

@test "bud-from-stdin" {
  target=scratch-image
  cat ${TESTSDIR}/bud/from-multiple-files/Dockerfile1.scratch | buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f - ${TESTSDIR}/bud/from-multiple-files
  cid=$(buildah from ${target})
  root=$(buildah mount ${cid})
  run test -s $root/Dockerfile1
  echo "$output"
  [ "$status" -eq 0 ]
  buildah rm ${cid}
  buildah rmi $(buildah --log-level=error images -q)
  run_buildah --log-level=error images -q
  expect_output ""
}

@test "bud with preprocessor" {
  target=alpine-image
  run_buildah --log-level=error bud -q --signature-policy ${TESTSDIR}/policy.json -t ${target} -f Decomposed.in ${TESTSDIR}/bud/preprocess
}

@test "bud with preprocessor error" {
  target=alpine-image
  run_buildah 1 --log-level=error bud -q --signature-policy ${TESTSDIR}/policy.json -t ${target} -f Error.in ${TESTSDIR}/bud/preprocess
}

@test "bud-with-rejected-name" {
  target=ThisNameShouldBeRejected
  run_buildah 1 --log-level=error bud -q --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/from-scratch
  expect_output --substring "must be lower"
}

@test "bud with chown copy" {
  imgName=alpine-image
  ctrName=alpine-chown
  run_buildah --log-level=error bud --signature-policy ${TESTSDIR}/policy.json -t ${imgName} ${TESTSDIR}/bud/copy-chown
  expect_output --substring "user:2367 group:3267"
  run_buildah --log-level=error from --name ${ctrName} ${imgName}
  run_buildah --log-level=error run alpine-chown -- stat -c '%u' /tmp/copychown.txt
  # Validate that output starts with "2367"
  expect_output --substring "2367"

  run_buildah --log-level=error run alpine-chown -- stat -c '%g' /tmp/copychown.txt
  # Validate that output starts with "3267"
  expect_output --substring "3267"
}

@test "bud with chown add" {
  imgName=alpine-image
  ctrName=alpine-chown
  run_buildah --log-level=error bud --signature-policy ${TESTSDIR}/policy.json -t ${imgName} ${TESTSDIR}/bud/add-chown
  expect_output --substring "user:2367 group:3267"
  run_buildah --log-level=error from --name ${ctrName} ${imgName}
  run_buildah --log-level=error run alpine-chown -- stat -c '%u' /tmp/addchown.txt
  # Validate that output starts with "2367"
  expect_output --substring "2367"

  run_buildah --log-level=error run alpine-chown -- stat -c '%g' /tmp/addchown.txt
  # Validate that output starts with "3267"
  expect_output --substring "3267"
}

@test "bud with ADD file construct" {
  buildah --log-level=error bud --signature-policy ${TESTSDIR}/policy.json -t test1 ${TESTSDIR}/bud/add-file
  run_buildah --log-level=error images -a
  expect_output --substring "test1"

  ctr=$(buildah --log-level=error from --signature-policy ${TESTSDIR}/policy.json test1)
  run_buildah --log-level=error containers -a
  expect_output --substring "test1"

  run_buildah --log-level=error run $ctr ls /var/file2
  expect_output --substring "/var/file2"
}

@test "bud with COPY of single file creates absolute path with correct permissions" {
  imgName=ubuntu-image
  ctrName=ubuntu-copy
  run_buildah --log-level=error bud --signature-policy ${TESTSDIR}/policy.json -t ${imgName} ${TESTSDIR}/bud/copy-create-absolute-path
  expect_output --substring "permissions=755"

  run_buildah --log-level=error from --name ${ctrName} ${imgName}
  run_buildah --log-level=error run ${ctrName} -- stat -c "%a" /usr/lib/python3.7/distutils
  expect_output --substring "755"
}

@test "bud with COPY of single file creates relative path with correct permissions" {
  imgName=ubuntu-image
  ctrName=ubuntu-copy
  run_buildah --log-level=error bud --signature-policy ${TESTSDIR}/policy.json -t ${imgName} ${TESTSDIR}/bud/copy-create-relative-path
  expect_output --substring "permissions=755"

  run_buildah --log-level=error from --name ${ctrName} ${imgName}
  run_buildah --log-level=error run ${ctrName} -- stat -c "%a" lib/custom
  expect_output --substring "755"
}

@test "bud with ADD of single file creates absolute path with correct permissions" {
  imgName=ubuntu-image
  ctrName=ubuntu-copy
  run_buildah --log-level=error bud --signature-policy ${TESTSDIR}/policy.json -t ${imgName} ${TESTSDIR}/bud/add-create-absolute-path
  expect_output --substring "permissions=755"

  run_buildah --log-level=error from --name ${ctrName} ${imgName}
  run_buildah --log-level=error run ${ctrName} -- stat -c "%a" /usr/lib/python3.7/distutils
  expect_output --substring "755"
}

@test "bud with ADD of single file creates relative path with correct permissions" {
  imgName=ubuntu-image
  ctrName=ubuntu-copy
  run_buildah --log-level=error bud --signature-policy ${TESTSDIR}/policy.json -t ${imgName} ${TESTSDIR}/bud/add-create-relative-path
  expect_output --substring "permissions=755"

  run_buildah --log-level=error from --name ${ctrName} ${imgName}
  run_buildah --log-level=error run ${ctrName} -- stat -c "%a" lib/custom
  expect_output --substring "755"
}

@test "bud multi-stage COPY creates absolute path with correct permissions" {
  imgName=ubuntu-image
  ctrName=ubuntu-copy
  run_buildah --log-level=error bud --signature-policy ${TESTSDIR}/policy.json -f ${TESTSDIR}/bud/copy-multistage-paths/Dockerfile.absolute -t ${imgName} ${TESTSDIR}/bud/copy-multistage-paths
  expect_output --substring "permissions=755"

  run_buildah --log-level=error from --name ${ctrName} ${imgName}
  run_buildah --log-level=error run ${ctrName} -- stat -c "%a" /my/bin
  expect_output --substring "755"
}

@test "bud multi-stage COPY creates relative path with correct permissions" {
  imgName=ubuntu-image
  ctrName=ubuntu-copy
  run_buildah --log-level=error bud --signature-policy ${TESTSDIR}/policy.json -f ${TESTSDIR}/bud/copy-multistage-paths/Dockerfile.relative -t ${imgName} ${TESTSDIR}/bud/copy-multistage-paths
  expect_output --substring "permissions=755"

  run_buildah --log-level=error from --name ${ctrName} ${imgName}
  run_buildah --log-level=error run ${ctrName} -- stat -c "%a" my/bin
  expect_output --substring "755"
}

@test "bud multi-stage COPY with invalid from statement" {
  imgName=ubuntu-image
  ctrName=ubuntu-copy
  run_buildah --log-level=error bud --signature-policy ${TESTSDIR}/policy.json -f ${TESTSDIR}/bud/copy-multistage-paths/Dockerfile.invalid_from -t ${imgName} ${TESTSDIR}/bud/copy-multistage-paths || true
  expect_output --substring "COPY: invalid --from flag, should be --from=<name|index>"
}

@test "bud COPY to root succeeds" {
  run_buildah --log-level=error bud --signature-policy ${TESTSDIR}/policy.json ${TESTSDIR}/bud/copy-root
}

@test "bud with FROM AS construct" {
  buildah bud --signature-policy ${TESTSDIR}/policy.json -t test1 ${TESTSDIR}/bud/from-as
  run_buildah --log-level=error images -a
  expect_output --substring "test1"

  ctr=$(buildah --log-level=error from --signature-policy ${TESTSDIR}/policy.json test1)
  run_buildah --log-level=error containers -a
  expect_output --substring "test1"

  run_buildah inspect --format "{{.Docker.ContainerConfig.Env}}" --type image test1
  expect_output --substring "LOCAL=/1"
}

@test "bud with FROM AS construct with layers" {
  buildah bud --layers --signature-policy ${TESTSDIR}/policy.json -t test1 ${TESTSDIR}/bud/from-as
  run_buildah --log-level=error images -a
  expect_output --substring "test1"

  ctr=$(buildah --log-level=error from --signature-policy ${TESTSDIR}/policy.json test1)
  run_buildah --log-level=error containers -a
  expect_output --substring "test1"

  run_buildah inspect --format "{{.Docker.ContainerConfig.Env}}" --type image test1
  expect_output --substring "LOCAL=/1"
}

@test "bud with FROM AS skip FROM construct" {
  run_buildah --log-level=error bud --signature-policy ${TESTSDIR}/policy.json -t test1 -f ${TESTSDIR}/bud/from-as/Dockerfile.skip ${TESTSDIR}/bud/from-as
  expect_output --substring "LOCAL=/1"
  expect_output --substring "LOCAL2=/2"

  run_buildah --log-level=error images -a
  expect_output --substring "test1"

  ctr=$(buildah --log-level=error from --signature-policy ${TESTSDIR}/policy.json test1)
  run_buildah --log-level=error containers -a
  expect_output --substring "test1"

  mnt=$(buildah mount $ctr)
  run test -e $mnt/1
  [ "${status}" -eq 0 ]
  run test -e $mnt/2
  [ "${status}" -ne 0 ]

  run_buildah --log-level=error inspect --format "{{.Docker.ContainerConfig.Env}}" --type image test1
  expect_output "[PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin LOCAL=/1]"
}

@test "bud with symlink Dockerfile not specified in file" {
  target=alpine-image
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/symlink ${TESTSDIR}/bud/symlink
  expect_output --substring "FROM alpine"
}

@test "bud with dir for file but no Dockerfile in dir" {
  target=alpine-image
  run_buildah 1 bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/empty-dir ${TESTSDIR}/bud/empty-dir
  expect_output --substring "no such file or directory"
}

@test "bud with bad dir Dockerfile" {
  target=alpine-image
  run_buildah 1 bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/baddirname ${TESTSDIR}/baddirname
  expect_output --substring "no such file or directory"
}

@test "bud with ARG before FROM default value" {
  target=leading-args-default
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/leading-args/Dockerfile ${TESTSDIR}/bud/leading-args
}

@test "bud with ARG before FROM" {
  target=leading-args
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} --build-arg=VERSION=musl -f ${TESTSDIR}/bud/leading-args/Dockerfile ${TESTSDIR}/bud/leading-args
}

@test "bud-with-healthcheck" {
  target=alpine-image
  buildah --log-level=error bud -q --signature-policy ${TESTSDIR}/policy.json -t ${target} --format docker ${TESTSDIR}/bud/healthcheck
  run_buildah --log-level=error inspect -f '{{printf "%q" .Docker.Config.Healthcheck.Test}} {{printf "%d" .Docker.Config.Healthcheck.StartPeriod}} {{printf "%d" .Docker.Config.Healthcheck.Interval}} {{printf "%d" .Docker.Config.Healthcheck.Timeout}} {{printf "%d" .Docker.Config.Healthcheck.Retries}}' ${target}
  second=1000000000
  threeseconds=$(( 3 * $second ))
  fiveminutes=$(( 5 * 60 * $second ))
  tenminutes=$(( 10 * 60 * $second ))
  expect_output '["CMD-SHELL" "curl -f http://localhost/ || exit 1"]'" $tenminutes $fiveminutes $threeseconds 4" "Healthcheck config"
}

@test "bud with unused build arg" {
  target=busybox-image
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} --build-arg foo=bar --build-arg foo2=bar2 -f ${TESTSDIR}/bud/build-arg ${TESTSDIR}/bud/build-arg
  expect_output --substring "one or more build args were not consumed: \[foo2\]"
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} --build-arg IMAGE=alpine -f ${TESTSDIR}/bud/build-arg/Dockerfile2 ${TESTSDIR}/bud/build-arg
  ! expect_output --substring "one or more build args were not consumed: \[IMAGE\]"
  expect_output --substring "FROM alpine"
}

@test "bud with copy-from and cache" {
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
  target=php-image
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/copy-from ${TESTSDIR}/bud/copy-from

  ctr=$(buildah --log-level=error from --signature-policy ${TESTSDIR}/policy.json ${target})
  mnt=$(buildah --log-level=error mount ${ctr})

  test -e $mnt/usr/local/bin/composer
}

@test "bud-target" {
  target=target
  run_buildah bud --log-level=error --signature-policy ${TESTSDIR}/policy.json -t ${target} --target mytarget ${TESTSDIR}/bud/target
  expect_output --substring "STEP 1: FROM ubuntu:latest"
  expect_output --substring "STEP 3: FROM alpine:latest AS mytarget"
  cid=$(buildah from ${target})
  root=$(buildah mount ${cid})
  run ls ${root}/2
  echo "$output"
  [ "$status" -eq 0 ]
  run ls ${root}/3
  [ "$status" -ne 0 ]
  buildah umount ${cid}
  buildah rm ${cid}
}

@test "bud-no-target-name" {
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json ${TESTSDIR}/bud/maintainer
}

@test "bud-multi-stage-nocache-nocommit" {
  # pull the base image directly, so that we don't record it being written to local storage in the next step
  run_buildah pull --signature-policy ${TESTSDIR}/policy.json alpine
  # okay, build an image with two stages
  run_buildah --log-level=debug bud --signature-policy ${TESTSDIR}/policy.json -f ${TESTSDIR}/bud/multi-stage-builds/Dockerfile.name ${TESTSDIR}/bud/multi-stage-builds
  # debug messages should only record us creating one new image: the one for the second stage, since we don't base anything on the first
  run grep "created new image ID" <<< "$output"
  expect_line_count 1
}

@test "bud-multi-stage-cache-nocontainer" {
  # first time through, quite normal
  run_buildah bud --layers -t base --signature-policy ${TESTSDIR}/policy.json -f ${TESTSDIR}/bud/multi-stage-builds/Dockerfile.rebase ${TESTSDIR}/bud/multi-stage-builds
  # second time through, everything should be cached, and we shouldn't create a container based on the final image
  run_buildah --log-level=debug bud --layers -t base --signature-policy ${TESTSDIR}/policy.json -f ${TESTSDIR}/bud/multi-stage-builds/Dockerfile.rebase ${TESTSDIR}/bud/multi-stage-builds
  # skip everything up through the final COMMIT step, and make sure we didn't log a "Container ID:" after it
  run sed '0,/COMMIT base/ d' <<< "$output"
  echo "$output"
  test "${#lines[@]}" -gt 1
  run grep "Container ID:" <<< "$output"
  echo "$output"
  test "${#lines[@]}" -eq 0
}

@test "bud copy to symlink" {
  target=alpine-image
  ctr=alpine-ctr
  run_buildah --log-level=error bud --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/dest-symlink
  expect_output --substring "STEP 5: RUN ln -s "

  run_buildah --log-level=error from --signature-policy ${TESTSDIR}/policy.json --name=${ctr} ${target}
  expect_output --substring ${ctr}

  run_buildah --log-level=error run ${ctr} ls -alF /etc/hbase
  expect_output --substring "/etc/hbase -> /usr/local/hbase/"

  run_buildah --log-level=error run ${ctr} ls -alF /usr/local/hbase
  expect_output --substring "Dockerfile"
}

@test "bud copy to dangling symlink" {
  target=ubuntu-image
  ctr=ubuntu-ctr
  run_buildah --log-level=error bud --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/dest-symlink-dangling
  expect_output --substring "STEP 3: RUN ln -s "

  run_buildah --log-level=error from --signature-policy ${TESTSDIR}/policy.json --name=${ctr} ${target}
  expect_output --substring ${ctr}

  run_buildah --log-level=error run ${ctr} ls -alF /src
  expect_output --substring "/src -> /symlink"

  run_buildah --log-level=error run ${ctr} ls -alF /symlink
  expect_output --substring "Dockerfile"
}

@test "bud WORKDIR isa symlink" {
  target=alpine-image
  ctr=alpine-ctr
  run_buildah --log-level=error bud --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/workdir-symlink
  expect_output --substring "STEP 3: RUN ln -sf "

  run_buildah --log-level=error from --signature-policy ${TESTSDIR}/policy.json --name=${ctr} ${target}
  expect_output --substring ${ctr}

  run_buildah --log-level=error run ${ctr} ls -alF /tempest
  expect_output --substring "/tempest -> /var/lib/tempest/"

  run_buildah --log-level=error run ${ctr} ls -alF /etc/notareal.conf
  expect_output --substring "\-rw\-rw\-r\-\-"
}

@test "bud WORKDIR isa symlink no target dir" {
  target=alpine-image
  ctr=alpine-ctr
  run_buildah --log-level=error bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f Dockerfile-2 ${TESTSDIR}/bud/workdir-symlink
  expect_output --substring "STEP 2: RUN ln -sf "

  run_buildah --log-level=error from --signature-policy ${TESTSDIR}/policy.json --name=${ctr} ${target}
  expect_output --substring ${ctr}

  run_buildah --log-level=error run ${ctr} ls -alF /tempest
  expect_output --substring "/tempest -> /var/lib/tempest/"

  run_buildah --log-level=error run ${ctr} ls /tempest
  expect_output --substring "Dockerfile-2"

  run_buildah --log-level=error run ${ctr} ls -alF /etc/notareal.conf
  expect_output --substring "\-rw\-rw\-r\-\-"
}

@test "bud WORKDIR isa symlink no target dir and follow on dir" {
  target=alpine-image
  ctr=alpine-ctr
  run_buildah --log-level=error bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f Dockerfile-3 ${TESTSDIR}/bud/workdir-symlink
  expect_output --substring "STEP 2: RUN ln -sf "

  run_buildah --log-level=error from --signature-policy ${TESTSDIR}/policy.json --name=${ctr} ${target}
  expect_output --substring ${ctr}

  run_buildah --log-level=error run ${ctr} ls -alF /tempest
  expect_output --substring "/tempest -> /var/lib/tempest/"

  run_buildah --log-level=error run ${ctr} ls /tempest
  expect_output --substring "Dockerfile-3"

  run_buildah --log-level=error run ${ctr} ls /tempest/lowerdir
  expect_output --substring "Dockerfile-3"

  run_buildah --log-level=error run ${ctr} ls -alF /etc/notareal.conf
  expect_output --substring "\-rw\-rw\-r\-\-"
}

@test "buidah bud --volume" {
  run_buildah --log-level=error bud --signature-policy ${TESTSDIR}/policy.json -v ${TESTSDIR}:/testdir ${TESTSDIR}/bud/mount
  expect_output --substring "/testdir"
}

@test "bud-copy-dot with --layers picks up changed file" {
  cp -a ${TESTSDIR}/bud/use-layers ${TESTDIR}/use-layers

  mkdir -p ${TESTDIR}/use-layers/subdir
  touch ${TESTDIR}/use-layers/subdir/file.txt
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers --iidfile ${TESTDIR}/iid1 -f Dockerfile.7 ${TESTDIR}/use-layers

  touch ${TESTDIR}/use-layers/subdir/file.txt
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers --iidfile ${TESTDIR}/iid2 -f Dockerfile.7 ${TESTDIR}/use-layers

  if [[ $(cat ${TESTDIR}/iid1) = $(cat ${TESTDIR}/iid2) ]]; then
    echo "Expected image id to change after touching a file copied into the image" >&2
    false
  fi

  buildah rmi -a -f
}

@test "buildah-bud-policy" {
  target=foo

  # A deny-all policy should prevent us from pulling the base image.
  run_buildah '?' bud --signature-policy ${TESTSDIR}/deny.json -t ${target} -v ${TESTSDIR}:/testdir ${TESTSDIR}/bud/mount
  [ "$status" -ne 0 ]
  expect_output --substring 'Source image rejected: Running image .* rejected by policy.'
  run_buildah rmi -a -f

  # A docker-only policy should allow us to pull the base image and commit.
  run_buildah bud --signature-policy ${TESTSDIR}/docker.json -t ${target} -v ${TESTSDIR}:/testdir ${TESTSDIR}/bud/mount
  # A deny-all policy shouldn't break pushing, since policy is only evaluated
  # on the source image, and we force it to allow local storage.
  run_buildah push --signature-policy ${TESTSDIR}/deny.json ${target} dir:${TESTDIR}/mount
  run_buildah rmi -a -f

  # A docker-only policy should allow us to pull the base image first...
  run_buildah pull --signature-policy ${TESTSDIR}/docker.json alpine
  # ... and since we don't need to pull the base image, a deny-all policy shouldn't break a build.
  run_buildah bud --signature-policy ${TESTSDIR}/deny.json -t ${target} -v ${TESTSDIR}:/testdir ${TESTSDIR}/bud/mount
  # A deny-all policy shouldn't break pushing, since policy is only evaluated
  # on the source image, and we force it to allow local storage.
  run_buildah push --signature-policy ${TESTSDIR}/deny.json ${target} dir:${TESTDIR}/mount
  # Similarly, a deny-all policy shouldn't break committing directly to other locations.
  run_buildah bud --signature-policy ${TESTSDIR}/deny.json -t dir:${TESTDIR}/mount -v ${TESTSDIR}:/testdir ${TESTSDIR}/bud/mount
  run_buildah rmi -a -f
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

@test "bud-copy-workdir" {
  target=testimage
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/copy-workdir
  run_buildah --log-level=error from ${target}
  cid="$output"
  run_buildah --log-level=error mount "${cid}"
  root="$output"
  test -s "${root}"/file1.txt
  test -d "${root}"/subdir
  test -s "${root}"/subdir/file2.txt
}

@test "bud-build-arg-cache" {
  target=derived-image
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t ${target} -f Dockerfile3 ${TESTSDIR}/bud/build-arg
  run_buildah --log-level=error inspect -f '{{.FromImageID}}' ${target}
  targetid="$output"

  # With build args, we should not find the previous build as a cached result.
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t ${target} -f Dockerfile3 --build-arg=UID=17122 --build-arg=CODE=/copr/coprs_frontend --build-arg=USERNAME=praiskup --build-arg=PGDATA=/pgdata ${TESTSDIR}/bud/build-arg
  run_buildah --log-level=error inspect -f '{{.FromImageID}}' ${target}
  argsid="$output"
  [[ "$argsid" != "$targetid" ]]

  # With build args, even in a different order, we should end up using the previous build as a cached result.
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t ${target} -f Dockerfile3 --build-arg=UID=17122 --build-arg=CODE=/copr/coprs_frontend --build-arg=USERNAME=praiskup --build-arg=PGDATA=/pgdata ${TESTSDIR}/bud/build-arg
  run_buildah --log-level=error inspect -f '{{.FromImageID}}' ${target}
  [[ "$output" == "$argsid" ]]

  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t ${target} -f Dockerfile3 --build-arg=CODE=/copr/coprs_frontend --build-arg=USERNAME=praiskup --build-arg=PGDATA=/pgdata --build-arg=UID=17122 ${TESTSDIR}/bud/build-arg
  run_buildah --log-level=error inspect -f '{{.FromImageID}}' ${target}
  [[ "$output" == "$argsid" ]]

  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t ${target} -f Dockerfile3 --build-arg=USERNAME=praiskup --build-arg=PGDATA=/pgdata --build-arg=UID=17122 --build-arg=CODE=/copr/coprs_frontend ${TESTSDIR}/bud/build-arg
  run_buildah --log-level=error inspect -f '{{.FromImageID}}' ${target}
  [[ "$output" == "$argsid" ]]

  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t ${target} -f Dockerfile3 --build-arg=PGDATA=/pgdata --build-arg=UID=17122 --build-arg=CODE=/copr/coprs_frontend --build-arg=USERNAME=praiskup ${TESTSDIR}/bud/build-arg
  run_buildah --log-level=error inspect -f '{{.FromImageID}}' ${target}
  [[ "$output" == "$argsid" ]]
}

@test "bud test RUN with a priv'd command" {
  target=alpinepriv
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/run-privd/Dockerfile ${TESTSDIR}/bud/run-privd
  [ "${status}" -eq 0 ]
  expect_output --substring "STEP 3: COMMIT"
  run_buildah --log-level=error images -q
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
  run_buildah --log-level=error from ${target}
  ctrid="$output"
  run_buildah --log-level=error mount "$ctrid"
  root="$output"
  id1=$(stat -c "%d:%i" ${root}/subdir/test1.txt)
  id2=$(stat -c "%d:%i" ${root}/subdir/test2.txt)
  id3=$(stat -c "%d:%i" ${root}/test3.txt)
  id4=$(stat -c "%d:%i" ${root}/test4.txt)
  [[ "$id1" == "$id2" ]]
  [[ "$id2" == "$id3" ]]
  [[ "$id3" == "$id4" ]]
}

@test "bud without any arguments should succeed" {
  cd ${TESTSDIR}/bud/from-scratch
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json
}

@test "bud without any arguments should fail when no Dockerfile exist" {
  cd $(mktemp -d)
  run_buildah 1 bud --signature-policy ${TESTSDIR}/policy.json
  [[ "$output" =~ "no such file or directory" ]]
}

@test "bud with specified context should fail if directory contains no Dockerfile" {
  DIR=$(mktemp -d)
  run_buildah 1 bud --signature-policy ${TESTSDIR}/policy.json "$DIR"
  [[ "$output" =~ "no such file or directory" ]]
}

@test "bud with specified context should fail if assumed Dockerfile is a directory" {
  DIR=$(mktemp -d)
  mkdir -p "$DIR"/Dockerfile
  run_buildah 1 bud --signature-policy ${TESTSDIR}/policy.json "$DIR"
  [[ "$output" =~ "is not a file" ]]
}

@test "bud with specified context should fail if context contains not-existing Dockerfile" {
  DIR=$(mktemp -d)
  run_buildah 1 bud --signature-policy ${TESTSDIR}/policy.json "$DIR"/Dockerfile
  [[ "$output" =~ "no such file or directory" ]]
}

@test "bud with specified context should succeed if context contains existing Dockerfile" {
  DIR=$(mktemp -d)
  touch "$DIR"/Dockerfile
  run_buildah 0 bud --signature-policy ${TESTSDIR}/policy.json "$DIR"/Dockerfile
}

@test "bud-no-change" {
  parent=alpine
  target=no-change-image
  buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/no-change
  run_buildah --log-level=error inspect --format '{{printf "%q" .FromImageDigest}}' ${parent}
  parentid="$output"
  run_buildah --log-level=error inspect --format '{{printf "%q" .FromImageDigest}}' ${target}
  [[ "$output" == "$parentid" ]]

  buildah rmi ${target}
}

@test "bud-no-change-label" {
  parent=alpine
  target=no-change-image
  buildah bud --label "test=label" --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/no-change
  run_buildah --log-level=error inspect --format '{{printf "%q" .Docker.Config.Labels}}' ${target}
  expect_output 'map["test":"label"]'

  buildah rmi ${target}
}

@test "bud-no-change-annotation" {
  target=no-change-image
  buildah bud --annotation "test=annotation" --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/no-change
  run_buildah --log-level=error inspect --format '{{printf "%q" .ImageAnnotations}}' ${target}
  expect_output 'map["test":"annotation"]'

  buildah rmi ${target}
}

@test "bud-squash-layers" {
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --squash ${TESTSDIR}/bud/layers-squash
}

@test "bud with additional directory of devices" {
  skip_if_chroot
  skip_if_rootless

  target=alpine-image
  rm -rf ${TESTSDIR}/foo
  mkdir -p ${TESTSDIR}/foo
  mknod ${TESTSDIR}/foo/null c 1 3
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --device ${TESTSDIR}/foo:/dev/fuse  -t ${target} -f ${TESTSDIR}/bud/device/Dockerfile ${TESTSDIR}/bud/device
  expect_output --substring "null"

  buildah rmi ${target}
  rm -rf ${TESTSDIR}/foo
}

@test "bud with additional device" {
  target=alpine-image
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --device /dev/fuse -t ${target} -f ${TESTSDIR}/bud/device/Dockerfile ${TESTSDIR}/bud/device
  [ "${status}" -eq 0 ]
  expect_output --substring "/dev/fuse"

  buildah rmi ${target}
}

@test "bud with Containerfile" {
  target=alpine-image
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/containerfile
  [ "${status}" -eq 0 ]
  expect_output --substring "FROM alpine"

  buildah rmi ${target}
}

@test "bud with Dockerfile" {
  target=alpine-image
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/dockerfile
  [ "${status}" -eq 0 ]
  expect_output --substring "FROM alpine"

  buildah rmi ${target}
}

@test "bud with Containerfile and Dockerfile" {
  target=alpine-image
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/containeranddockerfile
  [ "${status}" -eq 0 ]
  expect_output --substring "FROM alpine"

  buildah rmi ${target}
}

@test "bud-http-context-with-Containerfile" {
  starthttpd ${TESTSDIR}/bud/http-context-containerfile
  target=scratch-image
  buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} http://0.0.0.0:${HTTP_SERVER_PORT}/context.tar
  stophttpd
  cid=$(buildah from ${target})
  buildah rm ${cid}
  buildah rmi $(buildah --log-level=error images -q)
  run_buildah --log-level=error images -q
  expect_output ""
}

@test "bud with Dockerfile from stdin" {
  target=df-stdin
  cat ${TESTSDIR}/bud/context-from-stdin/Dockerfile | buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -
  [ "$?" -eq 0 ]
  cid=$(buildah from ${target})
  root=$(buildah mount ${cid})
  run test -s $root/etc/alpine-release
  echo "$output"
  buildah rm ${cid}
  buildah rmi $(buildah --log-level=error images -q)
  run_buildah --log-level=error images -q
  expect_output ""
}

@test "bud with Dockerfile from stdin tar" {
  target=df-stdin
  tar -c -C ${TESTSDIR}/bud/context-from-stdin . | buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -
  [ "$?" -eq 0 ]
  cid=$(buildah from ${target})
  root=$(buildah mount ${cid})
  run test -s $root/etc/alpine-release
  echo "$output"
  buildah rm ${cid}
  buildah rmi $(buildah --log-level=error images -q)
  run_buildah --log-level=error images -q
  expect_output ""
}

@test "bud containerfile with args" {
  target=use-args
  touch ${TESTSDIR}/bud/use-args/abc.txt
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} --build-arg=abc.txt ${TESTSDIR}/bud/use-args
  echo "$output"
  expect_output --substring "COMMIT use-args"
  ctrID=$(buildah from ${target})
  run_buildah run $ctrID ls abc.txt
  echo "$output"
  expect_output --substring "abc.txt"

  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f Containerfile.destination --build-arg=testArg=abc.txt --build-arg=destination=/tmp ${TESTSDIR}/bud/use-args
  echo "$output"
  expect_output --substring "COMMIT use-args"
  ctrID=$(buildah from ${target})
  run_buildah run $ctrID ls /tmp/abc.txt
  echo "$output"
  expect_output --substring "abc.txt"

  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f Containerfile.dest_nobrace --build-arg=testArg=abc.txt --build-arg=destination=/tmp ${TESTSDIR}/bud/use-args
  echo "$output"
  expect_output --substring "COMMIT use-args"
  ctrID=$(buildah from ${target})
  run_buildah run $ctrID ls /tmp/abc.txt
  echo "$output"
  expect_output --substring "abc.txt"

  rm ${TESTSDIR}/bud/use-args/abc.txt
  buildah rm --all
  buildah rmi --all --force
}

@test "bud using gitrepo and branch" {
  target=gittarget
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t ${target} git://github.com/containers/BuildSourceImage#master
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t ${target} git://github.com/containers/BuildSourceImage
}

# Fixes #1906: buildah was not detecting changed tarfile
@test "bud containerfile with tar archive in copy" {
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
  run_buildah 1 bud --signature-policy ${TESTSDIR}/policy.json -t ${target} --pull-never ${TESTSDIR}/bud/pull
  echo "$output"
  expect_output --substring "no such image"

  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} --pull ${TESTSDIR}/bud/pull
  echo "$output"
  expect_output --substring "COMMIT pull"

  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} --pull-never ${TESTSDIR}/bud/pull
  echo "$output"
  expect_output --substring "COMMIT pull"

  buildah rmi --all --force
}

@test "bud pull false no local image" {
  target=pull
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} --pull=false ${TESTSDIR}/bud/pull
  echo "$output"
  expect_output --substring "COMMIT pull"

  buildah rmi --all --force
  buildah rmi --all --force
}
