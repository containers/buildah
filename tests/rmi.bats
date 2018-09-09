#!/usr/bin/env bats

load helpers

@test "rmi-flags-order-verification" {
  run buildah rmi img1 -f
  check_options_flag_err "-f"

  run buildah rm img1 --all img2 
  check_options_flag_err "--all"

  run buildah rm img1 img2 --force
  check_options_flag_err "--force"
}

@test "remove one image" {
  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  buildah rm "$cid"
  buildah rmi alpine
  run buildah --debug=false images -q
  echo "$output"
  [ "$status" -eq 0 ]
  [ "$output" == "" ]
}

@test "remove multiple images" {
  cid2=$(buildah from --signature-policy ${TESTSDIR}/policy.json alpine)
  cid3=$(buildah from --signature-policy ${TESTSDIR}/policy.json busybox)
  run buildah rmi alpine busybox
  [ "$status" -eq 1 ]
  run buildah --debug=false images -q
  [ "$output" != "" ]

  buildah rmi -f alpine busybox
  run buildah --debug=false images -q
  echo "$output"
  [ "$status" -eq 0 ]
  [ "$output" == "" ]
}

@test "remove multiple non-existent images errors" {
  run buildah --debug=false rmi image1 image2 image3
  [ "${lines[0]}" == "could not get image \"image1\": identifier is not an image" ]
  [ "${lines[1]}" == "could not get image \"image2\": identifier is not an image" ]
  [ "${lines[2]}" == "could not get image \"image3\": identifier is not an image" ]
  [ $(wc -l <<< "$output") -eq 3 ]
  [ "${status}" -eq 1 ]
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
  run buildah rmi --all
  [ "$status" -eq 1 ]
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

@test "use conflicting commands to remove images" {
  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  buildah rm "$cid"
  run buildah --debug=false rmi -a alpine
  [ "$status" -eq 1 ]
  [ "$output" == "when using the --all switch, you may not pass any images names or IDs" ]

  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  buildah rm "$cid"
  run buildah --debug=false rmi -a -p
  [ "$status" -eq 1 ]
  [ "$output" == "when using the --all switch, you may not use --prune switch" ]
  buildah rmi --all
}

@test "remove image that is a parent of another image" {
  buildah rmi -a -f
  cid=$(buildah from --pull=true --signature-policy ${TESTSDIR}/policy.json alpine)
  buildah config --entrypoint '[ "/ENTRYPOINT" ]' $cid
  buildah commit --signature-policy ${TESTSDIR}/policy.json $cid new-image
  buildah rm -a
  run buildah --debug=false rmi alpine
  echo "$output"
  [ "${status}" -eq 1 ]
  [ $(wc -l <<< "$output") -eq 2 ]
  run buildah --debug=false images -q
  echo "$output"
  [ $(wc -l <<< "$output") -eq 1 ]
  [ "${status}" -eq 0 ]
  run buildah --debug=false images -q -a
  echo "$output"
  [ $(wc -l <<< "$output") -eq 2 ]
  [ "${status}" -eq 0 ]
  my_images=( $(buildah --debug=false images -a -q) )
  run buildah --debug=false rmi ${my_images[2]}
  echo "$output"
  [ "${status}" -ne 0 ]
  buildah rmi new-image
}

@test "rmi with cached images" {
  buildah rmi -a -f
  buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t test1 ${TESTSDIR}/bud/use-layers
  run buildah --debug=false images -a -q
  echo "$output"
  [ $(wc -l <<< "$output") -eq 6 ]
  [ "${status}" -eq 0 ]
  buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t test2 -f Dockerfile.2 ${TESTSDIR}/bud/use-layers
  run buildah --debug=false images -a -q
  echo "$output"
  [ $(wc -l <<< "$output") -eq 8 ]
  [ "${status}" -eq 0 ]
  run buildah --debug=false rmi test2
  echo "$output"
  [ "${status}" -eq 0 ]
  run buildah --debug=false images -a -q
  echo "$output"
  [ $(wc -l <<< "$output") -eq 6 ]
  [ "${status}" -eq 0 ]
  run buildah --debug=false rmi test1
  echo "$output"
  [ "${status}" -eq 0 ]
  run buildah --debug=false images -a -q
  echo "$output"
  [ $(wc -l <<< "$output") -eq 1 ]
  [ "${status}" -eq 0 ]
  buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t test3 -f Dockerfile.2 ${TESTSDIR}/bud/use-layers
  run buildah --debug=false rmi alpine
  echo "$output"
  [ "${status}" -eq 1 ]
  [ $(wc -l <<< "$output") -eq 2 ]
  run buildah --debug=false rmi test3
  echo "$output"
  [ "${status}" -eq 0 ]
  run buildah --debug=false images -a -q
  echo "$output"
  [ "${status}" -eq 0 ]
  [ "$output" == "" ]
}

@test "rmi image that is created from another named image" {
  buildah rmi -a -f
  cid=$(buildah from --pull=true --signature-policy ${TESTSDIR}/policy.json alpine)
  buildah config --entrypoint '[ "/ENTRYPOINT" ]' $cid
  buildah commit --signature-policy ${TESTSDIR}/policy.json $cid new-image
  cid=$(buildah from --pull=true --signature-policy ${TESTSDIR}/policy.json new-image)
  buildah config --env 'foo=bar' $cid
  buildah commit --signature-policy ${TESTSDIR}/policy.json $cid new-image-2
  buildah rm -a
  run buildah --debug=false rmi new-image-2
  echo "$output"
  [ "${status}" -eq 0 ]
  run buildah --debug=false images -q
  echo "$output"
  [ $(wc -l <<< "$output") -eq 2 ]
  [ "${status}" -eq 0 ]
}
