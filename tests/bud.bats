#!/usr/bin/env bats

load helpers

@test "bud-from-scratch" {
  target=scratch-image
  buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/from-scratch
  cid=$(buildah from ${target})
  buildah rm ${cid}
  buildah rmi $(buildah --debug=false images -q)
  run buildah --debug=false images -q
  [ "$status" -eq 0 ]
  [ "$output" = "" ]
}

@test "bud-from-multiple-files-one-from" {
  target=scratch-image
  buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/from-multiple-files/Dockerfile1.scratch -f ${TESTSDIR}/bud/from-multiple-files/Dockerfile2.nofrom
  cid=$(buildah from ${target})
  root=$(buildah mount ${cid})
  cmp $root/Dockerfile1 ${TESTSDIR}/bud/from-multiple-files/Dockerfile1.scratch
  cmp $root/Dockerfile2.nofrom ${TESTSDIR}/bud/from-multiple-files/Dockerfile2.nofrom
  run test -s $root/etc/passwd
  [ "$status" -ne 0 ]
  buildah rm ${cid}
  buildah rmi $(buildah --debug=false images -q)
  run buildah --debug=false images -q
  [ "$status" -eq 0 ]
  [ "$output" = "" ]

  target=alpine-image
  buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/from-multiple-files/Dockerfile1.alpine -f ${TESTSDIR}/bud/from-multiple-files/Dockerfile2.nofrom
  cid=$(buildah from ${target})
  root=$(buildah mount ${cid})
  cmp $root/Dockerfile1 ${TESTSDIR}/bud/from-multiple-files/Dockerfile1.alpine
  cmp $root/Dockerfile2.nofrom ${TESTSDIR}/bud/from-multiple-files/Dockerfile2.nofrom
  run test -s $root/etc/passwd
  [ "$status" -eq 0 ]
  buildah rm ${cid}
  buildah rmi $(buildah --debug=false images -q)
  run buildah --debug=false images -q
  [ "$status" -eq 0 ]
  [ "$output" = "" ]
}

@test "bud-from-multiple-files-two-froms" {
  target=scratch-image
  buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/from-multiple-files/Dockerfile1.scratch -f ${TESTSDIR}/bud/from-multiple-files/Dockerfile2.withfrom
  cid=$(buildah from ${target})
  root=$(buildah mount ${cid})
  cmp $root/Dockerfile1 ${TESTSDIR}/bud/from-multiple-files/Dockerfile1.scratch
  cmp $root/Dockerfile2.withfrom ${TESTSDIR}/bud/from-multiple-files/Dockerfile2.withfrom
  run test -s $root/etc/passwd
  [ "$status" -ne 0 ]
  buildah rm ${cid}
  buildah rmi $(buildah --debug=false images -q)
  run buildah --debug=false images -q
  [ "$status" -eq 0 ]
  [ "$output" = "" ]

  target=alpine-image
  buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/from-multiple-files/Dockerfile1.alpine -f ${TESTSDIR}/bud/from-multiple-files/Dockerfile2.withfrom
  cid=$(buildah from ${target})
  root=$(buildah mount ${cid})
  cmp $root/Dockerfile1 ${TESTSDIR}/bud/from-multiple-files/Dockerfile1.alpine
  cmp $root/Dockerfile2.withfrom ${TESTSDIR}/bud/from-multiple-files/Dockerfile2.withfrom
  run test -s $root/etc/passwd
  [ "$status" -eq 0 ]
  buildah rm ${cid}
  buildah rmi $(buildah --debug=false images -q)
  run buildah --debug=false images -q
  [ "$status" -eq 0 ]
  [ "$output" = "" ]
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
  buildah rmi $(buildah --debug=false images -q)
  run buildah --debug=false images -q
  [ "$status" -eq 0 ]
  [ "$output" = "" ]
}

@test "bud-http-Dockerfile" {
  starthttpd ${TESTSDIR}/bud/from-scratch
  target=scratch-image
  buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f http://0.0.0.0:${HTTP_SERVER_PORT}/Dockerfile .
  stophttpd
  cid=$(buildah from ${target})
  buildah rm ${cid}
  buildah rmi $(buildah --debug=false images -q)
  run buildah --debug=false images -q
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
  [ "$status" -eq 0 ]
  [ "$output" = "" ]
}

@test "bud-additional-tags" {
  target=scratch-image
  target2=another-scratch-image
  target3=so-many-scratch-images
  buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -t ${target2} -t ${target3} ${TESTSDIR}/bud/from-scratch
  run buildah --debug=false images
  [ "$status" -eq 0 ]
  cid=$(buildah from ${target})
  buildah rm ${cid}
  cid=$(buildah from library/${target2})
  buildah rm ${cid}
  cid=$(buildah from ${target3}:latest)
  buildah rm ${cid}
  buildah rmi -f $(buildah --debug=false images -q)
  run buildah --debug=false images -q
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
  cid=$(buildah from ${target})
  root=$(buildah mount ${cid})
  run test -s $root/vol/subvol/subvolfile
  [ "$status" -ne 0 ]
  run stat -c %f $root/vol/subvol
  [ "$status" -eq 0 ]
  [ "$output" = 41ed ]
  buildah rm ${cid}
  buildah rmi $(buildah --debug=false images -q)
  run buildah --debug=false images -q
  [ "$status" -eq 0 ]
  [ "$output" = "" ]
}

@test "bud-from-glob" {
  target=alpine-image
  buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f Dockerfile2.glob ${TESTSDIR}/bud/from-multiple-files
  cid=$(buildah from ${target})
  root=$(buildah mount ${cid})
  cmp $root/Dockerfile1.alpine ${TESTSDIR}/bud/from-multiple-files/Dockerfile1.alpine
  cmp $root/Dockerfile2.withfrom ${TESTSDIR}/bud/from-multiple-files/Dockerfile2.withfrom
  buildah rm ${cid}
  buildah rmi $(buildah --debug=false images -q)
  run buildah --debug=false images -q
  [ "$status" -eq 0 ]
  [ "$output" = "" ]
}

@test "bud-maintainer" {
  target=alpine-image
  buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/maintainer
  run buildah --debug=false inspect --type=image --format '{{.Docker.Author}}' ${target}
  [ "$status" -eq 0 ]
  [ "$output" = kilroy ]
  run buildah --debug=false inspect --type=image --format '{{.OCIv1.Author}}' ${target}
  [ "$status" -eq 0 ]
  [ "$output" = kilroy ]
  buildah rmi $(buildah --debug=false images -q)
  run buildah --debug=false images -q
  [ "$status" -eq 0 ]
  [ "$output" = "" ]
}

@test "bud-unrecognized-instruction" {
  target=alpine-image
  run buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/unrecognized
  [ "$status" -ne 0 ]
  [[ "$output" =~ BOGUS ]]
  buildah rmi $(buildah --debug=false images -q)
  run buildah --debug=false images -q
  [ "$status" -eq 0 ]
  [ "$output" = "" ]
}

@test "bud-shell" {
  target=alpine-image
  buildah bud --format docker --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/shell
  run buildah --debug=false inspect --type=image --format '{{printf "%q" .Docker.Config.Shell}}' ${target}
  echo $output
  [ "$status" -eq 0 ]
  [ "$output" = '["/bin/sh" "-c"]' ]
  ctr=$(buildah from ${target})
  run buildah --debug=false config --shell "/bin/bash -c" ${ctr}
  echo $output
  [ "$status" -eq 0 ]
  run buildah --debug=false inspect --type=container --format '{{printf "%q" .Docker.Config.Shell}}' ${ctr}
  echo $output
  [ "$status" -eq 0 ]
  [ "$output" = '["/bin/bash" "-c"]' ]
  buildah rm ${ctr}
  buildah rmi $(buildah --debug=false images -q)
  run buildah --debug=false images -q
  [ "$status" -eq 0 ]
  [ "$output" = "" ]
}

@test "bud with symlinks" {
  target=alpine-image
  run buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/symlink
  [ "$status" -eq 0 ]
  cid=$(buildah from ${target})
  root=$(buildah mount ${cid})
  run ls $root/data/log
  echo $output
  [ "$status" -eq 0 ]
  [[ "$output" =~ test ]]
  [[ "$output" =~ blah.txt ]]
  run ls -al $root
  echo $output
  [ "$status" -eq 0 ]
  [[ "$output" =~ "test-log -> /data/log" ]]
  [[ "$output" =~ "blah -> /test-log" ]]
  buildah rm ${cid}
  buildah rmi ${target}
}

@test "bud with symlinks to relative path" {
  target=alpine-image
  run buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/symlink/Dockerfile.relative-symlink
  [ "$status" -eq 0 ]
  cid=$(buildah from ${target})
  root=$(buildah mount ${cid})
  run ls $root/log
  echo $output
  [ "$status" -eq 0 ]
  [[ "$output" =~ test ]]
  run ls -al $root
  echo $output
  [ "$status" -eq 0 ]
  [[ "$output" =~ "test-log -> ../log" ]]
  buildah rm ${cid}
  buildah rmi ${target}
}

@test "bud with multiple symlinks in a path" {
  target=alpine-image
  run buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/symlink/Dockerfile.multiple-symlinks
  echo $output
  [ "$status" -eq 0 ]
  cid=$(buildah from ${target})
  root=$(buildah mount ${cid})
  run ls $root/data/log
  echo $output
  [ "$status" -eq 0 ]
  [[ "$output" =~ bin ]]
  [[ "$output" =~ blah.txt ]]
  run ls -al $root/myuser
  echo $output
  [ "$status" -eq 0 ]
  [[ "$output" =~ "log -> /test" ]]
  run ls -al $root/test
  echo $output
  [ "$status" -eq 0 ]
  [[ "$output" =~ "bar -> /test-log" ]]
  run ls -al $root/test-log
  echo $output
  [ "$status" -eq 0 ]
  [[ "$output" =~ "foo -> /data/log" ]]
  buildah rm ${cid}
  buildah rmi ${target}
}

@test "bud with multiple symlink pointing to itself" {
  target=alpine-image
  run buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} -f ${TESTSDIR}/bud/symlink/Dockerfile.symlink-points-to-itself
  [ "$status" -ne 0 ]
}
