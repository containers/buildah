#!/usr/bin/env bats

load helpers

@test "from-by-id" {
  image=busybox

  # Pull down the image, if we have to.
  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json $image
  expect_output "$image-working-container"
  cid=$output
  run_buildah rm $cid

  # Get the image's ID.
  run_buildah images -q $image
  expect_line_count 1
  iid="$output"

  # Use the image's ID to create a container.
  run_buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json ${iid}
  expect_line_count 1
  cid="$output"
  run_buildah rm $cid

  # Use a truncated form of the image's ID to create a container.
  run_buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json ${iid:0:6}
  expect_line_count 1
  cid="$output"
  run_buildah rm $cid

  run_buildah rmi $iid
}

@test "inspect-by-id" {
  image=busybox

  # Pull down the image, if we have to.
  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json $image
  expect_output "$image-working-container"
  cid=$output
  run_buildah rm $cid

  # Get the image's ID.
  run_buildah images -q $image
  expect_line_count 1
  iid="$output"

  # Use the image's ID to inspect it.
  run_buildah inspect --type=image ${iid}

  # Use a truncated copy of the image's ID to inspect it.
  run_buildah inspect --type=image ${iid:0:6}

  run_buildah rmi $iid
}

@test "push-by-id" {
  for image in busybox k8s.gcr.io/pause ; do
    echo pulling/pushing image $image

    TARGET=${TESTDIR}/subdir-$(basename $image)
    mkdir -p $TARGET $TARGET-truncated

    # Pull down the image, if we have to.
    run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json $image
    expect_output "${image##*/}-working-container"  # image, w/o registry prefix
    run_buildah rm $output

    # Get the image's ID.
    run_buildah images -q $image
    expect_output --substring '^[0-9a-f]{12,64}$'
    iid="$output"

    # Use the image's ID to push it.
    run_buildah push --signature-policy ${TESTSDIR}/policy.json $iid dir:$TARGET

    # Use a truncated form of the image's ID to push it.
    run_buildah push --signature-policy ${TESTSDIR}/policy.json ${iid:0:6} dir:$TARGET-truncated

    # Use the image's complete ID to remove it.
    run_buildah rmi $iid
  done
}

@test "rmi-by-id" {
  image=busybox

  # Pull down the image, if we have to.
  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json $image
  expect_output "$image-working-container"
  run_buildah rm $output

  # Get the image's ID.
  run_buildah images -q $image
  expect_output --substring '^[0-9a-f]{12,64}$'
  iid="$output"

  # Use a truncated copy of the image's ID to remove it.
  run_buildah rmi ${iid:0:6}
}
