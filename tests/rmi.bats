#!/usr/bin/env bats

load helpers

@test "rmi-flags-order-verification" {
  run_buildah 1 rmi img1 -f
  check_options_flag_err "-f"

  run_buildah 1 rmi img1 --all img2
  check_options_flag_err "--all"

  run_buildah 1 rmi img1 img2 --force
  check_options_flag_err "--force"
}

@test "remove one image" {
  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  buildah rm "$cid"
  buildah rmi alpine
  run_buildah --debug=false images -q
  expect_output ""
}

@test "remove multiple images" {
  cid2=$(buildah from --signature-policy ${TESTSDIR}/policy.json alpine)
  cid3=$(buildah from --signature-policy ${TESTSDIR}/policy.json busybox)
  run_buildah 1 rmi alpine busybox
  run_buildah --debug=false images -q
  [ "$output" != "" ]

  buildah rmi -f alpine busybox
  run_buildah --debug=false images -q
  expect_output ""
}

@test "remove multiple non-existent images errors" {
  run_buildah 1 --debug=false rmi image1 image2 image3
  expect_output --from="${lines[0]}" "could not get image \"image1\": identifier is not an image" "output line 1"
  expect_output --from="${lines[1]}" "could not get image \"image2\": identifier is not an image" "output line 2"
  expect_output --from="${lines[2]}" "could not get image \"image3\": identifier is not an image" "output line 3"
  [ $(wc -l <<< "$output") -gt 2 ]
}

@test "remove all images" {
  cid1=$(buildah from --signature-policy ${TESTSDIR}/policy.json scratch)
  cid2=$(buildah from --signature-policy ${TESTSDIR}/policy.json alpine)
  cid3=$(buildah from --signature-policy ${TESTSDIR}/policy.json busybox)
  buildah rmi -a -f
  run_buildah --debug=false images -q
  expect_output ""

  cid1=$(buildah from --signature-policy ${TESTSDIR}/policy.json scratch)
  cid2=$(buildah from --signature-policy ${TESTSDIR}/policy.json alpine)
  cid3=$(buildah from --signature-policy ${TESTSDIR}/policy.json busybox)
  run_buildah 1 rmi --all
  run_buildah --debug=false images -q
  [ "$output" != "" ]

  buildah rmi --all --force
  run_buildah --debug=false images -q
  expect_output ""
}

@test "use prune to remove dangling images" {
  createrandom ${TESTDIR}/randomfile
  createrandom ${TESTDIR}/other-randomfile

  cid=$(buildah from --signature-policy ${TESTSDIR}/policy.json busybox)

  run_buildah --debug=false images -q
  expect_line_count 1

  root=$(buildah mount $cid)
  cp ${TESTDIR}/randomfile $root/randomfile
  buildah unmount $cid
  buildah commit --signature-policy ${TESTSDIR}/policy.json $cid containers-storage:new-image

  run_buildah --debug=false images -q
  expect_line_count 2

  root=$(buildah mount $cid)
  cp ${TESTDIR}/other-randomfile $root/other-randomfile
  buildah unmount $cid
  buildah commit --signature-policy ${TESTSDIR}/policy.json $cid containers-storage:new-image

  run_buildah --debug=false images -q
  expect_line_count 3

  buildah rmi --prune

  run_buildah --debug=false images -q
  expect_line_count 2

  buildah rmi --all --force
  run_buildah --debug=false images -q
  expect_output ""
}

@test "use conflicting commands to remove images" {
  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  buildah rm "$cid"
  run_buildah 1 --debug=false rmi -a alpine
  expect_output --substring "when using the --all switch, you may not pass any images names or IDs"

  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  buildah rm "$cid"
  run_buildah 1 --debug=false rmi -p alpine
  expect_output --substring "when using the --prune switch, you may not pass any images names or IDs"

  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  buildah rm "$cid"
  run_buildah 1 --debug=false rmi -a -p
  expect_output --substring "when using the --all switch, you may not use --prune switch"
  buildah rmi --all
}

@test "remove image that is a parent of another image" {
  buildah rmi -a -f
  cid=$(buildah from --pull=true --signature-policy ${TESTSDIR}/policy.json alpine)
  buildah config --entrypoint '[ "/ENTRYPOINT" ]' $cid
  buildah commit --signature-policy ${TESTSDIR}/policy.json $cid new-image
  buildah rm -a
  run_buildah 1 --debug=false rmi alpine
  expect_line_count 2
  run_buildah --debug=false images -q
  expect_line_count 1
  run_buildah --debug=false images -q -a
  expect_line_count 2
  my_images=( $(buildah --debug=false images -a -q) )
  run_buildah 1 --debug=false rmi ${my_images[2]}
  buildah rmi new-image
}

@test "rmi with cached images" {
  buildah rmi -a -f
  buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t test1 ${TESTSDIR}/bud/use-layers
  run_buildah --debug=false images -a -q
  expect_line_count 7
  buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t test2 -f Dockerfile.2 ${TESTSDIR}/bud/use-layers
  run_buildah --debug=false images -a -q
  expect_line_count 9
  run_buildah --debug=false rmi test2
  run_buildah --debug=false images -a -q
  expect_line_count 7
  run_buildah --debug=false rmi test1
  run_buildah --debug=false images -a -q
  expect_line_count 1
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t test3 -f Dockerfile.2 ${TESTSDIR}/bud/use-layers
  run_buildah 1 --debug=false rmi alpine
  expect_line_count 2
  run_buildah --debug=false rmi test3
  run_buildah --debug=false images -a -q
  expect_output ""
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
  run_buildah --debug=false rmi new-image-2
  run_buildah --debug=false images -q
  expect_line_count 2
}
