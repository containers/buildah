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
  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
  cid=$output
  run_buildah rm "$cid"
  run_buildah rmi alpine
  run_buildah images -q
  expect_output ""
}

@test "remove multiple images" {
  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json alpine
  cid2=$output
  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json busybox
  cid3=$output
  run_buildah 1 rmi alpine busybox
  run_buildah images -q
  [ "$output" != "" ]

  run_buildah rmi -f alpine busybox
  run_buildah images -q
  expect_output ""
}

@test "remove multiple non-existent images errors" {
  run_buildah 1 rmi image1 image2 image3
  expect_output --from="${lines[0]}" "could not get image \"image1\": identifier is not an image" "output line 1"
  expect_output --from="${lines[1]}" "could not get image \"image2\": identifier is not an image" "output line 2"
  expect_output --from="${lines[2]}" "could not get image \"image3\": identifier is not an image" "output line 3"
  [ $(wc -l <<< "$output") -gt 2 ]
}

@test "remove all images" {
  run_buildah from --signature-policy ${TESTSDIR}/policy.json scratch
  cid1=$output
  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json alpine
  cid2=$output
  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json busybox
  cid3=$output
  run_buildah rmi -a -f
  run_buildah images -q
  expect_output ""

  run_buildah from --signature-policy ${TESTSDIR}/policy.json scratch
  cid1=$output
  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json alpine
  cid2=$output
  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json busybox
  cid3=$output
  run_buildah 1 rmi --all
  run_buildah images -q
  [ "$output" != "" ]

  run_buildah rmi --all --force
  run_buildah images -q
  expect_output ""
}

@test "use prune to remove dangling images" {
  createrandom ${TESTDIR}/randomfile
  createrandom ${TESTDIR}/other-randomfile

  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json busybox
  cid=$output

  run_buildah images -q
  expect_line_count 1

  run_buildah mount $cid
  root=$output
  cp ${TESTDIR}/randomfile $root/randomfile
  run_buildah unmount $cid
  run_buildah commit --signature-policy ${TESTSDIR}/policy.json $cid containers-storage:new-image

  run_buildah images -q
  expect_line_count 2

  run_buildah mount $cid
  root=$output
  cp ${TESTDIR}/other-randomfile $root/other-randomfile
  run_buildah unmount $cid
  run_buildah commit --signature-policy ${TESTSDIR}/policy.json $cid containers-storage:new-image

  run_buildah images -q
  expect_line_count 3

  run_buildah rmi --prune

  run_buildah images -q
  expect_line_count 2

  run_buildah rmi --all --force
  run_buildah images -q
  expect_output ""
}

@test "use conflicting commands to remove images" {
  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
  cid=$output
  run_buildah rm "$cid"
  run_buildah 1 rmi -a alpine
  expect_output --substring "when using the --all switch, you may not pass any images names or IDs"

  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
  cid=$output
  run_buildah rm "$cid"
  run_buildah 1 rmi -p alpine
  expect_output --substring "when using the --prune switch, you may not pass any images names or IDs"

  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
  cid=$output
  run_buildah rm "$cid"
  run_buildah 1 rmi -a -p
  expect_output --substring "when using the --all switch, you may not use --prune switch"
  run_buildah rmi --all
}

@test "remove image that is a parent of another image" {
  run_buildah rmi -a -f
  run_buildah from --quiet --pull=true --signature-policy ${TESTSDIR}/policy.json alpine
  cid=$output
  run_buildah config --entrypoint '[ "/ENTRYPOINT" ]' $cid
  run_buildah commit --signature-policy ${TESTSDIR}/policy.json $cid new-image
  run_buildah rm -a
  run_buildah 1 rmi alpine
  expect_line_count 2
  run_buildah images -q
  expect_line_count 1
  run_buildah images -q -a
  expect_line_count 2
  my_images=( $(buildah images -a -q) )
  run_buildah 1 rmi ${my_images[2]}
  run_buildah rmi new-image
}

@test "rmi with cached images" {
  run_buildah rmi -a -f
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t test1 ${TESTSDIR}/bud/use-layers
  run_buildah images -a -q
  expect_line_count 7
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t test2 -f Dockerfile.2 ${TESTSDIR}/bud/use-layers
  run_buildah images -a -q
  expect_line_count 9
  run_buildah rmi test2
  run_buildah images -a -q
  expect_line_count 7
  run_buildah rmi test1
  run_buildah images -a -q
  expect_line_count 1
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t test3 -f Dockerfile.2 ${TESTSDIR}/bud/use-layers
  run_buildah 1 rmi alpine
  expect_line_count 2
  run_buildah rmi test3
  run_buildah images -a -q
  expect_output ""
}

@test "rmi image that is created from another named image" {
  run_buildah rmi -a -f
  run_buildah from --quiet --pull=true --signature-policy ${TESTSDIR}/policy.json alpine
  cid=$output
  run_buildah config --entrypoint '[ "/ENTRYPOINT" ]' $cid
  run_buildah commit --signature-policy ${TESTSDIR}/policy.json $cid new-image
  run_buildah from --quiet --pull=true --signature-policy ${TESTSDIR}/policy.json new-image
  cid=$output
  run_buildah config --env 'foo=bar' $cid
  run_buildah commit --signature-policy ${TESTSDIR}/policy.json $cid new-image-2
  run_buildah rm -a
  run_buildah rmi new-image-2
  run_buildah images -q
  expect_line_count 2
}
