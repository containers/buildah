#!/usr/bin/env bats

load helpers

@test "bud-from-scratch" {
  target=scratch-image
  buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/from-scratch
  cid=$(buildah from ${target})
  buildah rm ${cid}
  buildah rmi $(buildah --debug=false images -q)
  run buildah --debug=false images -q
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
  run test -s $root/vol/anothervolfile
  [ "$status" -ne 0 ]
  buildah rm ${cid}
  buildah rmi $(buildah --debug=false images -q)
  run buildah --debug=false images -q
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
  [ "$output" = "" ]
}

@test "bud-http-context-dir-with-Dockerfile-post" {
  starthttpd ${TESTSDIR}/bud/http-context-subdir
  target=scratch-image
  buildah bud http://0.0.0.0:${HTTP_SERVER_PORT}/context.tar --signature-policy ${TESTSDIR}/policy.json -t ${target} -f context/Dockerfile
  stophttpd
  cid=$(buildah from ${target})
  buildah rm ${cid}
  buildah rmi $(buildah --debug=false images -q)
  run buildah --debug=false images -q
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
  [ "$output" = "" ]
}
