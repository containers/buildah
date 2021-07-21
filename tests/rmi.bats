#!/usr/bin/env bats

load helpers

@test "rmi-flags-order-verification" {
  run_buildah 125 rmi img1 -f
  check_options_flag_err "-f"

  run_buildah 125 rmi img1 --all img2
  check_options_flag_err "--all"

  run_buildah 125 rmi img1 img2 --force
  check_options_flag_err "--force"
}

@test "remove one image" {
  _prefetch alpine
  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
  cid=$output
  run_buildah rm "$cid"
  run_buildah rmi alpine
  run_buildah images -q
  expect_output ""
}

@test "remove multiple images" {
  _prefetch alpine busybox
  run_buildah from --pull=false --quiet --signature-policy ${TESTSDIR}/policy.json alpine
  cid2=$output
  run_buildah from --pull=false --quiet --signature-policy ${TESTSDIR}/policy.json busybox
  cid3=$output
  run_buildah 125 rmi alpine busybox
  run_buildah images -q
  [ "$output" != "" ]

  run_buildah rmi -f alpine busybox
  run_buildah images -q
  expect_output ""
}

@test "remove multiple non-existent images errors" {
  run_buildah 125 rmi image1 image2 image3
  expect_output --from="${lines[1]}" --substring " image1: image not known"
  expect_output --from="${lines[2]}" --substring " image2: image not known"
  expect_output --from="${lines[3]}" --substring " image3: image not known"
}

@test "remove all images" {
  _prefetch alpine busybox
  run_buildah from --signature-policy ${TESTSDIR}/policy.json scratch
  cid1=$output
  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json alpine
  cid2=$output
  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json busybox
  cid3=$output
  run_buildah rmi -a -f
  run_buildah images -q
  expect_output ""

  _prefetch alpine busybox
  run_buildah from --signature-policy ${TESTSDIR}/policy.json scratch
  cid1=$output
  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json alpine
  cid2=$output
  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json busybox
  cid3=$output
  run_buildah 125 rmi --all
  run_buildah images -q
  [ "$output" != "" ]

  run_buildah rmi --all --force
  run_buildah images -q
  expect_output ""
}

@test "use prune to remove dangling images" {
  _prefetch busybox

  createrandom ${TESTDIR}/randomfile
  createrandom ${TESTDIR}/other-randomfile

  run_buildah from --pull=false --quiet --signature-policy ${TESTSDIR}/policy.json busybox
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

@test "use prune to remove dangling images with parent" {
  createrandom ${TESTDIR}/randomfile
  createrandom ${TESTDIR}/other-randomfile

  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json scratch
  cid=$output

  run_buildah images -q -a
  expect_line_count 0

  run_buildah mount $cid
  root=$output
  cp ${TESTDIR}/randomfile $root/randomfile
  run_buildah unmount $cid
  run_buildah commit --quiet --signature-policy ${TESTSDIR}/policy.json $cid
  image=$output
  run_buildah rm $cid

  run_buildah images -q -a
  expect_line_count 1

  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json $image
  cid=$output
  run_buildah mount $cid
  root=$output
  cp ${TESTDIR}/other-randomfile $root/other-randomfile
  run_buildah unmount $cid
  run_buildah commit --signature-policy ${TESTSDIR}/policy.json $cid
  run_buildah rm $cid

  run_buildah images -q -a
  expect_line_count 2

  run_buildah rmi --prune

  run_buildah images -q -a
  expect_line_count 0

  run_buildah images -q -a
  expect_output ""
}

@test "attempt to prune non-dangling empty images" {
  # Regression test for containers/podman/issues/10832
  ctxdir=${TESTDIR}/bud
  mkdir -p $ctxdir
  cat >$ctxdir/Dockerfile <<EOF
FROM scratch
ENV test1=test1
ENV test2=test2
EOF

  run_buildah bud -t test $ctxdir
  run_buildah rmi --prune
  expect_output "" "no image gets pruned"
}

@test "use conflicting commands to remove images" {
  _prefetch alpine
  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
  cid=$output
  run_buildah rm "$cid"
  run_buildah 125 rmi -a alpine
  expect_output --substring "when using the --all switch, you may not pass any images names or IDs"

  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
  cid=$output
  run_buildah rm "$cid"
  run_buildah 125 rmi -p alpine
  expect_output --substring "when using the --prune switch, you may not pass any images names or IDs"

  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
  cid=$output
  run_buildah rm "$cid"
  run_buildah 125 rmi -a -p
  expect_output --substring "when using the --all switch, you may not use --prune switch"
  run_buildah rmi --all
}

@test "remove image that is a parent of another image" {
  _prefetch alpine
  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
  cid=$output
  run_buildah config --entrypoint '[ "/ENTRYPOINT" ]' $cid
  run_buildah commit --signature-policy ${TESTSDIR}/policy.json $cid new-image
  run_buildah rm -a

  # Since it has children, alpine will only be untagged (Podman compat) but not
  # marked as removed.  However, it won't show up in the image list anymore.
  run_buildah rmi alpine
  expect_output --substring "untagged: "
  run_buildah images -q
  expect_line_count 1
  run_buildah images -q -a
  expect_line_count 1
}

@test "rmi with cached images" {
  _prefetch alpine
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
  run_buildah rmi alpine
  run_buildah rmi test3
  run_buildah images -a -q
  expect_output ""
}

@test "rmi image that is created from another named image" {
  _prefetch alpine
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
