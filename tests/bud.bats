#!/usr/bin/env bats

load helpers

@test "bud-flags-order-verification" {
  run buildah bud /tmp/tmpdockerfile/ -t blabla
  check_options_flag_err "-t"

  run buildah bud /tmp/tmpdockerfile/ -q -t blabla
  check_options_flag_err "-q"

  run buildah bud /tmp/tmpdockerfile/ --force-rm
  check_options_flag_err "--force-rm"

  run buildah bud /tmp/tmpdockerfile/ --userns=cnt1
  check_options_flag_err "--userns=cnt1"
}

@test "bud with --layers and --no-cache flags" {
  buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t test1 ${TESTSDIR}/bud/use-layers
  run buildah --debug=false images -a
  [ $(wc -l <<< "$output") -eq 8 ]
  [ "${status}" -eq 0 ]
  buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t test2 ${TESTSDIR}/bud/use-layers
  run buildah --debug=false images -a
  [ $(wc -l <<< "$output") -eq 10 ]
  [ "${status}" -eq 0 ]
  run buildah inspect --format "{{.Docker.ContainerConfig.Env}}" test2
  echo "$output"
  [ "$status" -eq 0 ]
  [[ $output =~ "foo=bar" ]]
  run buildah inspect --format "{{.Docker.ContainerConfig.ExposedPorts}}" test2
  echo "$output"
  [ "$status" -eq 0 ]
  [[ $output =~ "8080" ]]

  buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t test3 -f Dockerfile.2 ${TESTSDIR}/bud/use-layers
  run buildah --debug=false images -a
  [ $(wc -l <<< "$output") -eq 12 ]
  [ "${status}" -eq 0 ]

  mkdir -p ${TESTSDIR}/bud/use-layers/mount/subdir
  buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t test4 -f Dockerfile.3 ${TESTSDIR}/bud/use-layers
  run buildah --debug=false images -a
  [ $(wc -l <<< "$output") -eq 14 ]
  [ "${status}" -eq 0 ]
  touch ${TESTSDIR}/bud/use-layers/mount/subdir/file.txt
  buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t test5 -f Dockerfile.3 ${TESTSDIR}/bud/use-layers
  run buildah --debug=false images -a
  [ $(wc -l <<< "$output") -eq 16 ]
  [ "${status}" -eq 0 ]

  buildah bud --signature-policy ${TESTSDIR}/policy.json --no-cache -t test6 -f Dockerfile.2 ${TESTSDIR}/bud/use-layers
  run buildah --debug=false images -a
  [ $(wc -l <<< "$output") -eq 19 ]
  [ "${status}" -eq 0 ]

  buildah rmi -a -f
  rm -rf ${TESTSDIR}/bud/use-layers/mount
}

@test "bud with --layers, multistage, and COPY with --from" {
  mkdir -p ${TESTSDIR}/bud/use-layers/uuid
  uuidgen > ${TESTSDIR}/bud/use-layers/uuid/data
  mkdir -p ${TESTSDIR}/bud/use-layers/date
  date > ${TESTSDIR}/bud/use-layers/date/data

  buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t test1 -f Dockerfile.multistage-copy ${TESTSDIR}/bud/use-layers
  run buildah --debug=false images -a
  [ $(wc -l <<< "$output") -eq 6 ]
  [ "${status}" -eq 0 ]
  buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t test2 -f Dockerfile.multistage-copy ${TESTSDIR}/bud/use-layers
  run buildah --debug=false images -a
  [ $(wc -l <<< "$output") -eq 7 ]
  [ "${status}" -eq 0 ]

  uuidgen > ${TESTSDIR}/bud/use-layers/uuid/data
  date > ${TESTSDIR}/bud/use-layers/date/data
  buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t test3 -f Dockerfile.multistage-copy ${TESTSDIR}/bud/use-layers
  run buildah --debug=false images -a
  [ $(wc -l <<< "$output") -eq 11 ]
  [ "${status}" -eq 0 ]
  run buildah --debug=false containers
  [ $(wc -l <<< "$output") -eq 1 ]
  [ "${status}" -eq 0 ]
  
  ctr=$(buildah --debug=false from --signature-policy ${TESTSDIR}/policy.json test3)
  mnt=$(buildah --debug=false mount ${ctr})
  run test -e $mnt/uuid
  [ "${status}" -eq 0 ]
  run test -e $mnt/date
  [ "${status}" -eq 0 ]

  buildah bud --signature-policy ${TESTSDIR}/policy.json -t test4 -f Dockerfile.multistage-copy ${TESTSDIR}/bud/use-layers
  run buildah --debug=false images -a
  [ $(wc -l <<< "$output") -eq 12 ]
  [ "${status}" -eq 0 ]

  buildah rmi -a -f
  rm -rf ${TESTSDIR}/bud/use-layers/uuid
  rm -rf ${TESTSDIR}/bud/use-layers/date
}

@test "bud with --layers and symlink file" {
  echo 'echo "Hello World!"' > ${TESTSDIR}/bud/use-layers/hello.sh
  cd ${TESTSDIR}/bud/use-layers && ln -s hello.sh hello_world.sh
  buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t test -f Dockerfile.4 ${TESTSDIR}/bud/use-layers
  run buildah --debug=false images -a
  [ $(wc -l <<< "$output") -eq 4 ]
  [ "${status}" -eq 0 ]
  buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t test1 -f Dockerfile.4 ${TESTSDIR}/bud/use-layers
  run buildah --debug=false images -a
  [ $(wc -l <<< "$output") -eq 5 ]
  [ "${status}" -eq 0 ]

  echo 'echo "Hello Cache!"' > ${TESTSDIR}/bud/use-layers/hello.sh
  buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t test2 -f Dockerfile.4 ${TESTSDIR}/bud/use-layers
  run buildah --debug=false images -a
  [ $(wc -l <<< "$output") -eq 7 ]
  [ "${status}" -eq 0 ]

  rm ${TESTSDIR}/bud/use-layers/hello.sh ${TESTSDIR}/bud/use-layers/hello_world.sh
  buildah rmi -a -f
}

@test "bud with --rm flag" {
  buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t test1 ${TESTSDIR}/bud/use-layers
  run buildah --debug=false containers
  [ $(wc -l <<< "$output") -eq 1 ]
  [ "${status}" -eq 0 ]

  buildah bud --signature-policy ${TESTSDIR}/policy.json --rm=false --layers -t test2 ${TESTSDIR}/bud/use-layers
  run buildah --debug=false containers
  [ $(wc -l <<< "$output") -eq 7 ]
  [ "${status}" -eq 0 ]

  buildah rm -a
  buildah rmi -a -f
}

@test "bud with --force-rm flag" {
  run buildah bud --signature-policy ${TESTSDIR}/policy.json --force-rm --layers -t test1 -f Dockerfile.fail-case ${TESTSDIR}/bud/use-layers
  echo "$output"
  [ "$status" -ne 0 ]
  run buildah --debug=false containers
  [ $(wc -l <<< "$output") -eq 1 ]
  [ "${status}" -eq 0 ]

  run buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t test2 -f Dockerfile.fail-case ${TESTSDIR}/bud/use-layers
  echo "$output"
  [ "$status" -ne 0 ]
  run buildah --debug=false containers
  [ $(wc -l <<< "$output") -eq 2 ]
  [ "${status}" -eq 0 ]

  buildah rm -a
  buildah rmi -a -f
}

@test "bud --layers with non-existent/down registry" {
  run buildah bud --signature-policy ${TESTSDIR}/policy.json --force-rm --layers -t test1 -f Dockerfile.non-existent-registry ${TESTSDIR}/bud/use-layers
  echo "$output"
  [ "$status" -ne 0 ]
  [[ "$output" =~ "no such host" ]]
}

@test "bud from base image should have base image ENV also" {
  buildah bud --signature-policy ${TESTSDIR}/policy.json -t test -f Dockerfile.check-env ${TESTSDIR}/bud/env
  cid=$(buildah from --signature-policy ${TESTSDIR}/policy.json test)
  buildah config --env random=hello ${cid}
  buildah commit --signature-policy ${TESTSDIR}/policy.json ${cid} test1
  buildah --debug inspect test1 | grep foo=bar
  buildah --debug  inspect test1 | grep random=hello
  buildah rm ${cid}
  buildah rmi -a -f
}

@test "bud-from-scratch" {
  target=scratch-image
  buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/from-scratch
  cid=$(buildah from ${target})
  buildah rm ${cid}
  buildah rmi $(buildah --debug=false images -q)
  run buildah --debug=false images -q
  echo "$output"
  [ "$status" -eq 0 ]
  [ "$output" = "" ]
}

@test "bud-from-scratch-iid" {
  target=scratch-image
  buildah bud --iidfile output.iid --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/from-scratch
  iid=$(cat output.iid)
  cid=$(buildah from ${iid})
  buildah rm ${cid}
  buildah rmi $(buildah --debug=false images -q)
  run buildah --debug=false images -q
  echo "$output"
  [ "$status" -eq 0 ]
  [ "$output" = "" ]
}

@test "bud-from-scratch-label" {
  target=scratch-image
  buildah bud --label "test=label" --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/from-scratch
  run buildah --debug=false inspect --format '{{printf "%q" .Docker.Config.Labels}}' ${target}
  echo "$output"
  [ "$status" -eq 0 ]
  [ "$output" = 'map["test":"label"]' ]
  buildah rmi ${target}
}

@test "bud-from-scratch-annotation" {
  target=scratch-image
  buildah bud --annotation "test=annotation" --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/from-scratch
  run buildah --debug=false inspect --format '{{printf "%q" .ImageAnnotations}}' ${target}
  echo "$output"
  [ "$status" -eq 0 ]
  [ "$output" = 'map["test":"annotation"]' ]
  buildah rmi ${target}
}

@test "bud-from-multiple-files-one-from" {
  target=scratch-image
  buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/from-multiple-files/Dockerfile1.scratch -f ${TESTSDIR}/bud/from-multiple-files/Dockerfile2.nofrom ${TESTSDIR}/bud/from-multiple-files
  cid=$(buildah from ${target})
  root=$(buildah mount ${cid})
  cmp $root/Dockerfile1 ${TESTSDIR}/bud/from-multiple-files/Dockerfile1.scratch
  cmp $root/Dockerfile2.nofrom ${TESTSDIR}/bud/from-multiple-files/Dockerfile2.nofrom
  run test -s $root/etc/passwd
  echo "$output"
  [ "$status" -ne 0 ]
  buildah rm ${cid}
  buildah rmi $(buildah --debug=false images -q)
  run buildah --debug=false images -q
  echo "$output"
  [ "$status" -eq 0 ]
  [ "$output" = "" ]

  target=alpine-image
  buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f Dockerfile1.alpine -f Dockerfile2.nofrom ${TESTSDIR}/bud/from-multiple-files
  cid=$(buildah from ${target})
  root=$(buildah mount ${cid})
  cmp $root/Dockerfile1 ${TESTSDIR}/bud/from-multiple-files/Dockerfile1.alpine
  cmp $root/Dockerfile2.nofrom ${TESTSDIR}/bud/from-multiple-files/Dockerfile2.nofrom
  run test -s $root/etc/passwd
  echo "$output"
  [ "$status" -eq 0 ]
  buildah rm ${cid}
  buildah rmi -a
  run buildah --debug=false images -q
  echo "$output"
  [ "$status" -eq 0 ]
  [ "$output" = "" ]
}

@test "bud-from-multiple-files-two-froms" {
  target=scratch-image
  buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f Dockerfile1.scratch -f Dockerfile2.withfrom ${TESTSDIR}/bud/from-multiple-files
  cid=$(buildah from ${target})
  root=$(buildah mount ${cid})
  run test -s $root/Dockerfile1
  echo "$output"
  [ "$status" -ne 0 ]
  cmp $root/Dockerfile2.withfrom ${TESTSDIR}/bud/from-multiple-files/Dockerfile2.withfrom
  run test -s $root/etc/passwd
  echo "$output"
  [ "$status" -eq 0 ]
  buildah rm ${cid}
  buildah rmi -a
  run buildah --debug=false images -q
  echo "$output"
  [ "$status" -eq 0 ]
  [ "$output" = "" ]

  target=alpine-image
  buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f Dockerfile1.alpine -f Dockerfile2.withfrom ${TESTSDIR}/bud/from-multiple-files
  cid=$(buildah from ${target})
  root=$(buildah mount ${cid})
  run test -s $root/Dockerfile1
  echo "$output"
  [ "$status" -ne 0 ]
  cmp $root/Dockerfile2.withfrom ${TESTSDIR}/bud/from-multiple-files/Dockerfile2.withfrom
  run test -s $root/etc/passwd
  echo "$output"
  [ "$status" -eq 0 ]
  buildah rm ${cid}
  buildah rmi -a
  run buildah --debug=false images -q
  echo "$output"
  [ "$status" -eq 0 ]
  [ "$output" = "" ]
}

@test "bud-multi-stage-builds" {
  target=multi-stage-index
  buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/multi-stage-builds/Dockerfile.index ${TESTSDIR}/bud/multi-stage-builds
  cid=$(buildah from ${target})
  root=$(buildah mount ${cid})
  cmp $root/Dockerfile.index ${TESTSDIR}/bud/multi-stage-builds/Dockerfile.index
  run test -s $root/etc/passwd
  [ "$status" -eq 0 ]
  buildah rm ${cid}
  buildah rmi -a
  run buildah --debug=false images -q
  echo "$output"
  [ "$status" -eq 0 ]
  [ "$output" = "" ]

  target=multi-stage-name
  buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f Dockerfile.name ${TESTSDIR}/bud/multi-stage-builds
  cid=$(buildah from ${target})
  root=$(buildah mount ${cid})
  cmp $root/Dockerfile.name ${TESTSDIR}/bud/multi-stage-builds/Dockerfile.name
  run test -s $root/etc/passwd
  [ "$status" -ne 0 ]
  buildah rm ${cid}
  buildah rmi $(buildah --debug=false images -q)
  run buildah --debug=false images -q
  echo "$output"
  [ "$output" = "" ]
  [ "$status" -eq 0 ]

  target=multi-stage-mixed
  buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/multi-stage-builds/Dockerfile.mixed ${TESTSDIR}/bud/multi-stage-builds
  cid=$(buildah from ${target})
  root=$(buildah mount ${cid})
  cmp $root/Dockerfile.name ${TESTSDIR}/bud/multi-stage-builds/Dockerfile.name
  cmp $root/Dockerfile.index ${TESTSDIR}/bud/multi-stage-builds/Dockerfile.index
  cmp $root/Dockerfile.mixed ${TESTSDIR}/bud/multi-stage-builds/Dockerfile.mixed
  buildah rm ${cid}
  buildah rmi $(buildah --debug=false images -q)
  run buildah --debug=false images -q
  echo "$output"
  [ "$output" = "" ]
  [ "$status" -eq 0 ]
}

@test "bud-preserve-subvolumes" {
  # This Dockerfile needs us to be able to handle a working RUN instruction.
  if ! which runc ; then
    skip
  fi
  target=volume-image
  buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/preserve-volumes
  cid=$(buildah from ${target})
  root=$(buildah mount ${cid})
  test -s $root/vol/subvol/subsubvol/subsubvolfile
  run test -s $root/vol/subvol/subvolfile
  [ "$status" -ne 0 ]
  test -s $root/vol/volfile
  test -s $root/vol/Dockerfile
  test -s $root/vol/Dockerfile2
  run test -s $root/vol/anothervolfile
  [ "$status" -ne 0 ]
  buildah rm ${cid}
  buildah rmi -a
  run buildah --debug=false images -q
  echo "$output"
  [ "$status" -eq 0 ]
  [ "$output" = "" ]
}

@test "bud-http-Dockerfile" {
  starthttpd ${TESTSDIR}/bud/from-scratch
  target=scratch-image
  buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} http://0.0.0.0:${HTTP_SERVER_PORT}/Dockerfile
  stophttpd
  cid=$(buildah from ${target})
  buildah rm ${cid}
  buildah rmi $(buildah --debug=false images -q)
  run buildah --debug=false images -q
  echo "$output"
  [ "$status" -eq 0 ]
  [ "$output" = "" ]
}

@test "bud-http-context-with-Dockerfile" {
  starthttpd ${TESTSDIR}/bud/http-context
  target=scratch-image
  buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} http://0.0.0.0:${HTTP_SERVER_PORT}/context.tar
  stophttpd
  cid=$(buildah from ${target})
  buildah rm ${cid}
  buildah rmi $(buildah --debug=false images -q)
  run buildah --debug=false images -q
  echo "$output"
  [ "$status" -eq 0 ]
  [ "$output" = "" ]
}

@test "bud-http-context-dir-with-Dockerfile-pre" {
  starthttpd ${TESTSDIR}/bud/http-context-subdir
  target=scratch-image
  buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f context/Dockerfile http://0.0.0.0:${HTTP_SERVER_PORT}/context.tar
  stophttpd
  cid=$(buildah from ${target})
  buildah rm ${cid}
  buildah rmi $(buildah --debug=false images -q)
  run buildah --debug=false images -q
  echo "$output"
  [ "$status" -eq 0 ]
  [ "$output" = "" ]
}

@test "bud-http-context-dir-with-Dockerfile-post" {
  starthttpd ${TESTSDIR}/bud/http-context-subdir
  target=scratch-image
  buildah bud  --signature-policy ${TESTSDIR}/policy.json -t ${target} -f context/Dockerfile http://0.0.0.0:${HTTP_SERVER_PORT}/context.tar
  stophttpd
  cid=$(buildah from ${target})
  buildah rm ${cid}
  buildah rmi $(buildah --debug=false images -q)
  run buildah --debug=false images -q
  echo "$output"
  [ "$status" -eq 0 ]
  [ "$output" = "" ]
}

@test "bud-git-context" {
  # We need git and ssh to be around to handle cloning a repository.
  if ! which git ; then
    skip
  fi
  if ! which ssh ; then
    skip
  fi
  target=giturl-image
  # Any repo should do, but this one is small and is FROM: scratch.
  gitrepo=git://github.com/projectatomic/nulecule-library
  buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} "${gitrepo}"
  cid=$(buildah from ${target})
  buildah rm ${cid}
  buildah rmi $(buildah --debug=false images -q)
  run buildah --debug=false images -q
  echo "$output"
  [ "$status" -eq 0 ]
  [ "$output" = "" ]
}

@test "bud-github-context" {
  target=github-image
  # Any repo should do, but this one is small and is FROM: scratch.
  gitrepo=github.com/projectatomic/nulecule-library
  buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} "${gitrepo}"
  cid=$(buildah from ${target})
  buildah rm ${cid}
  buildah --debug=false images -q
  buildah rmi $(buildah --debug=false images -q)
  run buildah --debug=false images -q
  echo "$output"
  [ "$status" -eq 0 ]
  [ "$output" = "" ]
}

@test "bud-additional-tags" {
  target=scratch-image
  target2=another-scratch-image
  target3=so-many-scratch-images
  buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -t docker.io/${target2} -t ${target3} ${TESTSDIR}/bud/from-scratch
  run buildah --debug=false images
  echo "$output"
  [ "$status" -eq 0 ]
  cid=$(buildah from ${target})
  buildah rm ${cid}
  cid=$(buildah from --signature-policy ${TESTSDIR}/policy.json library/${target2})
  buildah rm ${cid}
  cid=$(buildah from --signature-policy ${TESTSDIR}/policy.json ${target3}:latest)
  buildah rm ${cid}
  buildah rmi -f $(buildah --debug=false images -q)
  run buildah --debug=false images -q
  echo "$output"
  [ "$status" -eq 0 ]
  [ "$output" = "" ]
}

@test "bud-volume-perms" {
  # This Dockerfile needs us to be able to handle a working RUN instruction.
  if ! which runc ; then
    skip
  fi
  target=volume-image
  buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/volume-perms
  cid=$(buildah from --signature-policy ${TESTSDIR}/policy.json ${target})
  root=$(buildah mount ${cid})
  run test -s $root/vol/subvol/subvolfile
  [ "$status" -ne 0 ]
  run stat -c %f $root/vol/subvol
  echo "$output"
  [ "$status" -eq 0 ]
  [ "$output" = 41ed ]
  buildah rm ${cid}
  buildah rmi -a
  run buildah --debug=false images -q
  echo "$output"
  [ "$status" -eq 0 ]
  [ "$output" = "" ]
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
  run buildah --debug=false images -q
  echo "$output"
  [ "$status" -eq 0 ]
  [ "$output" = "" ]
}

@test "bud-maintainer" {
  target=alpine-image
  buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/maintainer
  run buildah --debug=false inspect --type=image --format '{{.Docker.Author}}' ${target}
  echo "$output"
  [ "$status" -eq 0 ]
  [ "$output" = kilroy ]
  run buildah --debug=false inspect --type=image --format '{{.OCIv1.Author}}' ${target}
  echo "$output"
  [ "$status" -eq 0 ]
  [ "$output" = kilroy ]
  buildah rmi -a
  run buildah --debug=false images -q
  echo "$output"
  [ "$status" -eq 0 ]
  [ "$output" = "" ]
}

@test "bud-unrecognized-instruction" {
  target=alpine-image
  run buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/unrecognized
  echo "$output"
  [ "$status" -ne 0 ]
  [[ "$output" =~ BOGUS ]]
  buildah rmi $(buildah --debug=false images -q)
  run buildah --debug=false images -q
  echo "$output"
  [ "$status" -eq 0 ]
  [ "$output" = "" ]
}

@test "bud-shell" {
  target=alpine-image
  buildah bud --format docker --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/shell
  run buildah --debug=false inspect --type=image --format '{{printf "%q" .Docker.Config.Shell}}' ${target}
  echo "$output"
  [ "$status" -eq 0 ]
  [ "$output" = '["/bin/sh" "-c"]' ]
  ctr=$(buildah from --signature-policy ${TESTSDIR}/policy.json ${target})
  run buildah --debug=false config --shell "/bin/bash -c" ${ctr}
  echo "$output"
  [ "$status" -eq 0 ]
  run buildah --debug=false inspect --type=container --format '{{printf "%q" .Docker.Config.Shell}}' ${ctr}
  echo "$output"
  [ "$status" -eq 0 ]
  [ "$output" = '["/bin/bash" "-c"]' ]
  buildah rm ${ctr}
  buildah rmi -a
  run buildah --debug=false images -q
  echo "$output"
  [ "$status" -eq 0 ]
  [ "$output" = "" ]
}

@test "bud with symlinks" {
  target=alpine-image
  run buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/symlink
  echo "$output"
  [ "$status" -eq 0 ]
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
  run buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f Dockerfile.relative-symlink ${TESTSDIR}/bud/symlink 
  echo "$output"
  [ "$status" -eq 0 ]
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
  buildah rm ${cid}
  buildah rmi ${target}
}

@test "bud with multiple symlinks in a path" {
  target=alpine-image
  run buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/symlink/Dockerfile.multiple-symlinks ${TESTSDIR}/bud/symlink
  echo "$output"
  [ "$status" -eq 0 ]
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
  run buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/symlink/Dockerfile.symlink-points-to-itself ${TESTSDIR}/bud/symlink
  echo "$output"
  [ "$status" -ne 0 ]
}

@test "bud with ENTRYPOINT and RUN" {
  target=alpine-image
  run buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f Dockerfile.entrypoint-run ${TESTSDIR}/bud/run-scenarios
  echo "$output"
  [ "$status" -eq 0 ]
  [[ "$output" =~ "unique.test.string" ]]
  cid=$(buildah from --signature-policy ${TESTSDIR}/policy.json ${target})
  buildah rm ${cid}
  buildah rmi ${target}
}

@test "bud with ENTRYPOINT and empty RUN" {
  target=alpine-image
  run buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f Dockerfile.entrypoint-empty-run ${TESTSDIR}/bud/run-scenarios
  echo "$output"
  [[ "$output" =~ "error building at step" ]]
  [ "$status" -eq 1 ]
}

@test "bud with CMD and RUN" {
  target=alpine-image
  run buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/run-scenarios/Dockerfile.cmd-run ${TESTSDIR}/bud/run-scenarios
  echo "$output"
  [[ "$output" =~ "unique.test.string" ]]
  [ "$status" -eq 0 ]
  cid=$(buildah from --signature-policy ${TESTSDIR}/policy.json ${target})
  buildah rm ${cid}
  buildah rmi ${target}
}

@test "bud with CMD and empty RUN" {
  target=alpine-image
  run buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f Dockerfile.cmd-empty-run ${TESTSDIR}/bud/run-scenarios
  echo "$output"
  [[ "$output" =~ "error building at step" ]]
  [ "$status" -eq 1 ]
}

@test "bud with ENTRYPOINT, CMD and RUN" {
  target=alpine-image
  run buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/run-scenarios/Dockerfile.entrypoint-cmd-run ${TESTSDIR}/bud/run-scenarios
  echo "$output"
  [[ "$output" =~ "unique.test.string" ]]
  [ "$status" -eq 0 ]
  cid=$(buildah from --signature-policy ${TESTSDIR}/policy.json ${target})
  buildah rm ${cid}
  buildah rmi ${target}
}

@test "bud with ENTRYPOINT, CMD and empty RUN" {
  target=alpine-image
  run buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/run-scenarios/Dockerfile.entrypoint-cmd-empty-run ${TESTSDIR}/bud/run-scenarios
  echo "$output"
  [[ "$output" =~ "error building at step" ]]
  [ "$status" -eq 1 ]
}

# Determines if a variable set with ENV is available to following commands in the Dockerfile
@test "bud access ENV variable defined in same source file" {
  target=env-image
  run buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/env/Dockerfile.env-same-file ${TESTSDIR}/bud/env
  echo "$output"
  [[ "$output" =~ ":unique.test.string:" ]]
  [ "$status" -eq 0 ]
  cid=$(buildah from --signature-policy ${TESTSDIR}/policy.json ${target})
  buildah rm ${cid}
  buildah rmi ${target}
}

# Determines if a variable set with ENV in an image is available to commands in downstream Dockerfile
@test "bud access ENV variable defined in FROM image" {
  from_target=env-from-image
  target=env-image
  run buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${from_target} -f ${TESTSDIR}/bud/env/Dockerfile.env-same-file ${TESTSDIR}/bud/env
  echo "$output"
  [ "$status" -eq 0 ]
  run buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/env/Dockerfile.env-from-image ${TESTSDIR}/bud/env
  echo "$output"
  [[ "$output" =~ "@unique.test.string@" ]]
  [ "$status" -eq 0 ]
  from_cid=$(buildah from ${from_target})
  cid=$(buildah from ${target})
  buildah rm ${from_cid} ${cid}
  buildah rmi -a -f
}

@test "bud with Dockerfile from valid URL" {
  target=url-image
  url=https://raw.githubusercontent.com/containers/buildah/master/tests/bud/from-scratch/Dockerfile
  run buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} ${url}
  [ "$status" -eq 0 ]
  cid=$(buildah from ${target})
  buildah rm ${cid}
  buildah rmi ${target}
}

@test "bud with Dockerfile from invalid URL" {
  target=url-image
  url=https://raw.githubusercontent.com/containers/buildah/master/tests/bud/from-scratch/Dockerfile.bogus
  run buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} ${url}
  [ "$status" -eq 1 ]
}

# When provided with a -f flag and directory, buildah will look for the alternate Dockerfile name in the supplied directory
@test "bud with -f flag, alternate Dockerfile name" {
  target=fileflag-image
  run buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f Dockerfile.noop-flags ${TESTSDIR}/bud/run-scenarios
  [ "$status" -eq 0 ]
  cid=$(buildah from ${target})
  buildah rm ${cid}
  buildah rmi ${target}
}

# Following flags are configured to result in noop but should not affect buildiah bud behavior
@test "bud with --cache-from noop flag" {
  target=noop-image
  run buildah bud --cache-from=invalidimage --signature-policy ${TESTSDIR}/policy.json -t ${target} -f Dockerfile.noop-flags ${TESTSDIR}/bud/run-scenarios
  echo "$output"
  [ "$status" -eq 0 ]
  cid=$(buildah from ${target})
  buildah rm ${cid}
  buildah rmi ${target}
}

@test "bud with --compress noop flag" {
  target=noop-image
  run buildah bud --compress --signature-policy ${TESTSDIR}/policy.json -t ${target} -f Dockerfile.noop-flags ${TESTSDIR}/bud/run-scenarios
  echo "$output"
  [ "$status" -eq 0 ]
  cid=$(buildah from ${target})
  buildah rm ${cid}
  buildah rmi ${target}
}

@test "bud with --cpu-shares flag, no argument" {
  target=bud-flag
  run buildah bud --cpu-shares --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/from-scratch/Dockerfile ${TESTSDIR}/bud/from-scratch
  echo "$output"
  [ "$status" -ne 0 ]
}

@test "bud with --cpu-shares flag, invalid argument" {
  target=bud-flag
  run buildah bud --cpu-shares bogus --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/from-scratch/Dockerfile ${TESTSDIR}/bud/from-scratch
  echo "$output"
  [ "$status" -ne 0 ]
  [[ "$output" =~ "invalid value \"bogus\" for flag" ]]
}

@test "bud with --cpu-shares flag, valid argument" {
  target=bud-flag
  run buildah bud --cpu-shares 2 --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/from-scratch/Dockerfile ${TESTSDIR}/bud/from-scratch
  echo "$output"
  [ "$status" -eq 0 ]
  cid=$(buildah from ${target})
  buildah rm ${cid}
  buildah rmi ${target}
}

@test "bud with --cpu-shares short flag (-c), no argument" {
  target=bud-flag
  run buildah bud -c --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/from-scratch/Dockerfile ${TESTSDIR}/bud/from-scratch
  echo "$output"
  [ "$status" -ne 0 ]
}

@test "bud with --cpu-shares short flag (-c), invalid argument" {
  target=bud-flag
  run buildah bud -c bogus --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/from-scratch/Dockerfile ${TESTSDIR}/bud/from-scratch
  echo "$output"
  [ "$status" -ne 0 ]
  [[ "$output" =~ "invalid value \"bogus\" for flag" ]]
}

@test "bud with --cpu-shares short flag (-c), valid argument" {
  target=bud-flag
  run buildah bud -c 2 --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/from-scratch
  echo "$output"
  [ "$status" -eq 0 ]
  cid=$(buildah from ${target})
  buildah rm ${cid}
  buildah rmi ${target}
}

@test "bud-onbuild" {
  target=onbuild
  buildah bud --format docker --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/onbuild
  run buildah --debug=false inspect --format '{{printf "%q" .Docker.Config.OnBuild}}' ${target}
  echo "$output"
  [ "$status" -eq 0 ]
  [ "$output" = '["RUN touch /onbuild1" "RUN touch /onbuild2"]' ]
  cid=$(buildah from ${target})
  root=$(buildah mount ${cid})
  run ls ${root}/onbuild1 ${root}/onbuild2
  echo "$output"
  [ "$status" -eq 0 ]
  buildah umount ${cid}
  buildah rm ${cid}

  target=onbuild-image2
  buildah bud --format docker --signature-policy ${TESTSDIR}/policy.json -t ${target} -f Dockerfile1 ${TESTSDIR}/bud/onbuild
  run buildah --debug=false inspect --format '{{printf "%q" .Docker.Config.OnBuild}}' ${target}
  echo "$output"
  [ "$status" -eq 0 ]
  [ "$output" = '["RUN touch /onbuild3"]' ]
  cid=$(buildah from ${target})
  root=$(buildah mount ${cid})
  run ls ${root}/onbuild1 ${root}/onbuild2 ${root}/onbuild3
  echo "$output"
  [ "$status" -eq 0 ]
  buildah umount ${cid}

  run buildah --debug=false config --onbuild "RUN touch /onbuild4" ${cid}
  echo "$output"
  [ "$status" -eq 0 ]

  target=onbuild-image3
  buildah commit --signature-policy ${TESTSDIR}/policy.json --format docker ${cid} ${target}
  run buildah --debug=false inspect --format '{{printf "%q" .Docker.Config.OnBuild}}' ${target}
  echo "$output"
  [ "$status" -eq 0 ]
  [ "$output" = '["RUN touch /onbuild4"]' ]
  buildah rm ${cid}
  buildah rmi --all
}

@test "bud-logfile" {
  rm -f ${TESTDIR}/logfile
  run buildah bud --logfile ${TESTDIR}/logfile --signature-policy ${TESTSDIR}/policy.json ${TESTSDIR}/bud/preserve-volumes
  echo "$output"
  [ "$status" -eq 0 ]
  [ "$output" = "" ]
  test -s ${TESTDIR}/logfile
}

@test "bud with ARGS" {
  target=alpine-image
  run buildah --debug=false bud -q --signature-policy ${TESTSDIR}/policy.json -t ${target} -f Dockerfile.args ${TESTSDIR}/bud/run-scenarios
  echo "$output"
  [[ "$output" =~ "arg_value" ]]
  [ "$status" -eq 0 ]
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
  buildah rmi $(buildah --debug=false images -q)
  run buildah --debug=false images -q
  echo "$output"
  [ "$status" -eq 0 ]
  [ "$output" = "" ]
}

@test "bud with preprocessor" {
  target=alpine-image
  run buildah --debug=false bud -q --signature-policy ${TESTSDIR}/policy.json -t ${target} -f Decomposed.in ${TESTSDIR}/bud/preprocess
  echo "$output"
  [ "$status" -eq 0 ]
  echo "$output"
}

@test "bud with preprocessor error" {
  target=alpine-image
  run buildah --debug=false bud -q --signature-policy ${TESTSDIR}/policy.json -t ${target} -f Error.in ${TESTSDIR}/bud/preprocess
  echo "$output"
  [ "$status" -eq 1 ]
}

@test "bud-with-rejected-name" {
  target=ThisNameShouldBeRejected
  run buildah --debug=false bud -q --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/from-scratch
  echo "$output"
  [ "$status" -eq 1 ]
  [[ "${output}" =~ "must be lower" ]]
}

@test "bud with chown copy" {
  imgName=alpine-image
  ctrName=alpine-chown
  run buildah --debug=false bud --signature-policy ${TESTSDIR}/policy.json -t ${imgName} ${TESTSDIR}/bud/copy-chown
  echo "$output"
  [ "$status" -eq 0 ]
  [[ "$output" =~ user:2367[[:space:]]group:3267 ]]
  run buildah --debug=false from --name ${ctrName} ${imgName}
  echo "$output"
  [ "$status" -eq 0 ] 
  run buildah --debug=false run alpine-chown -- stat -c '%u' /tmp/copychown.txt
  echo "$output"
  # Validate that output starts with "2367"
  [ $(expr "$output" : "2367") -ne 0 ]
  [ "$status" -eq 0 ]
  
  run buildah --debug=false run alpine-chown -- stat -c '%g' /tmp/copychown.txt
  echo "$output"
  # Validate that output starts with "3267"
  [ $(expr "$output" : "3267") -ne 0 ]
  [ "$status" -eq 0 ]
}

@test "bud with ADD file construct" {
  buildah --debug=false bud --signature-policy ${TESTSDIR}/policy.json -t test1 ${TESTSDIR}/bud/add-file
  run buildah --debug=false images -a
  [[ $output =~ "test1" ]]
  [ "${status}" -eq 0 ]

  ctr=$(buildah --debug=false from --signature-policy ${TESTSDIR}/policy.json test1)
  run buildah --debug=false containers -a
  [[ $output =~ "test1" ]]
  [ "${status}" -eq 0 ]

  run buildah --debug=false run $ctr ls /var/file2
  echo "$output"
  [ "$status" -eq 0 ]
  [[ $output =~ "/var/file2" ]]
}

@test "bud with FROM AS construct" {
  buildah bud --signature-policy ${TESTSDIR}/policy.json -t test1 ${TESTSDIR}/bud/from-as
  run buildah --debug=false images -a
  [[ $output =~ "test1" ]]
  [ "${status}" -eq 0 ]

  ctr=$(buildah --debug=false from --signature-policy ${TESTSDIR}/policy.json test1)
  run buildah --debug=false containers -a
  [[ $output =~ "test1" ]]
  [ "${status}" -eq 0 ]
  
  run buildah inspect --format "{{.Docker.ContainerConfig.Env}}" --type image test1
  echo "$output"
  [ "$status" -eq 0 ]
  [[ $output =~ "LOCAL=/1" ]]
}

@test "bud with FROM AS construct with layers" {
  buildah bud --layers --signature-policy ${TESTSDIR}/policy.json -t test1 ${TESTSDIR}/bud/from-as
  run buildah --debug=false images -a
  [[ $output =~ "test1" ]]
  [ "${status}" -eq 0 ]

  ctr=$(buildah --debug=false from --signature-policy ${TESTSDIR}/policy.json test1)
  run buildah --debug=false containers -a
  [[ $output =~ "test1" ]]
  [ "${status}" -eq 0 ]
  
  run buildah inspect --format "{{.Docker.ContainerConfig.Env}}" --type image test1
  echo "$output"
  [ "$status" -eq 0 ]
  [[ $output =~ "LOCAL=/1" ]]
}

@test "bud with FROM AS skip FROM construct" {
  run buildah --debug=false bud --signature-policy ${TESTSDIR}/policy.json -t test1 -f ${TESTSDIR}/bud/from-as/Dockerfile.skip ${TESTSDIR}/bud/from-as
  [[ $output =~ "LOCAL=/1" ]]
  [[ $output =~ "LOCAL2=/2" ]]
  [ "${status}" -eq 0 ]

  run buildah --debug=false images -a
  [[ $output =~ "test1" ]]
  [ "${status}" -eq 0 ]

  ctr=$(buildah --debug=false from --signature-policy ${TESTSDIR}/policy.json test1)
  run buildah --debug=false containers -a
  [[ $output =~ "test1" ]]
  [ "${status}" -eq 0 ]

  mnt=$(buildah mount $ctr)
  run test -e $mnt/1
  [ "${status}" -eq 0 ]
  run test -e $mnt/2
  [ "${status}" -ne 0 ]
 
  run buildah --debug=false inspect --format "{{.Docker.ContainerConfig.Env}}" --type image test1
  echo "$output"
  [ "$status" -eq 0 ]
  [[ $output =~ "[PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin LOCAL=/1]" ]]
}

@test "bud with symlink Dockerfile not specified in file" {
  target=alpine-image
  run buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/symlink ${TESTSDIR}/bud/symlink
  echo "$output"
  [[ $output =~ "FROM alpine" ]]
  [ "$status" -eq 0 ]
}

@test "bud with dir for file but no Dockerfile in dir" {
  target=alpine-image
  run buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/empty-dir ${TESTSDIR}/bud/empty-dir
  echo "$output"
  [[ $output =~ "no such file or directory" ]]
  [ "$status" -ne 0 ]
}

@test "bud with bad dir Dockerfile" {
  target=alpine-image
  run buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/baddirname ${TESTSDIR}/baddirname
  echo "$output"
  [[ $output =~ "no such file or directory" ]]
  [ "$status" -ne 0 ]
}

@test "bud with ARG before FROM default value" {
  target=leading-args-default
  run buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/leading-args/Dockerfile ${TESTSDIR}/bud/leading-args
  [ "$status" -eq 0 ]
}

@test "bud with ARG before FROM" {
  target=leading-args
  run buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} --build-arg=VERSION=musl -f ${TESTSDIR}/bud/leading-args/Dockerfile ${TESTSDIR}/bud/leading-args
  [ "$status" -eq 0 ]
}

@test "bud-with-healthcheck" {
  target=alpine-image
  buildah --debug=false bud -q --signature-policy ${TESTSDIR}/policy.json -t ${target} --format docker ${TESTSDIR}/bud/healthcheck
  run buildah --debug=false inspect -f '{{printf "%q" .Docker.Config.Healthcheck.Test}} {{printf "%d" .Docker.Config.Healthcheck.StartPeriod}} {{printf "%d" .Docker.Config.Healthcheck.Interval}} {{printf "%d" .Docker.Config.Healthcheck.Timeout}} {{printf "%d" .Docker.Config.Healthcheck.Retries}}' ${target}
  echo "$output"
  [ "$status" -eq 0 ]
  second=1000000000
  threeseconds=$(( 3 * $second ))
  fiveminutes=$(( 5 * 60 * $second ))
  tenminutes=$(( 10 * 60 * $second ))
  [ "$output" = '["CMD-SHELL" "curl -f http://localhost/ || exit 1"]'" $tenminutes $fiveminutes $threeseconds 4" ]
}
