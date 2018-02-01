#!/usr/bin/env bats

load helpers

@test "remove one image" {
  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  buildah rm "$cid"
  buildah rmi alpine
  run buildah --debug=false images -q
  [ "$output" == "" ]
}

@test "remove multiple images" {
  cid1=$(buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json scratch)
  cid2=$(buildah from --signature-policy ${TESTSDIR}/policy.json alpine)
  cid3=$(buildah from --signature-policy ${TESTSDIR}/policy.json busybox)
  buildah rmi alpine busybox
  run buildah --debug=false images -q
  [ "$output" != "" ]

  buildah rm $cid1 $cid2 $cid3
  buildah rmi alpine busybox
  run buildah --debug=false images -q
  [ "$output" == "" ]
}

@test "remove all images" {
  cid1=$(buildah from --signature-policy ${TESTSDIR}/policy.json scratch)
  cid2=$(buildah from --signature-policy ${TESTSDIR}/policy.json alpine)
  cid3=$(buildah from --signature-policy ${TESTSDIR}/policy.json busybox)
  buildah rmi -a -f
  run buildah --debug=false images -q
  [ "$output" == "" ]

  cid1=$(buildah from --signature-policy ${TESTSDIR}/policy.json scratch)
  cid2=$(buildah from --signature-policy ${TESTSDIR}/policy.json alpine)
  cid3=$(buildah from --signature-policy ${TESTSDIR}/policy.json busybox)
  buildah rmi --all
  run buildah --debug=false images -q
  [ "$output" != "" ]

  buildah rmi --all --force
  run buildah --debug=false images -q
  [ "$output" == "" ]
}

@test "use prune to remove dangling images" {
  createrandom ${TESTDIR}/randomfile
  createrandom ${TESTDIR}/other-randomfile

  cid=$(buildah from --signature-policy ${TESTSDIR}/policy.json busybox)

  run buildah --debug=false images -q
  [ $(wc -l <<< "$output") -eq 1 ]

  root=$(buildah mount $cid)
  cp ${TESTDIR}/randomfile $root/randomfile
  buildah unmount $cid
  buildah commit --signature-policy ${TESTSDIR}/policy.json $cid containers-storage:new-image

  run buildah --debug=false images -q
  [ $(wc -l <<< "$output") -eq 2 ]

  root=$(buildah mount $cid)
  cp ${TESTDIR}/other-randomfile $root/other-randomfile
  buildah unmount $cid
  buildah commit --signature-policy ${TESTSDIR}/policy.json $cid containers-storage:new-image

  run buildah --debug=false images -q
  [ $(wc -l <<< "$output") -eq 3 ]

  buildah rmi --prune

  run buildah --debug=false images -q
  [ $(wc -l <<< "$output") -eq 2 ]

  buildah rmi --all --force
  run buildah --debug=false images -q
  [ "$output" == "" ]
}
