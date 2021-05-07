#!/usr/bin/env bats

load helpers

@test "overlay specific level" {
  if test \! -e /usr/bin/fuse-overlayfs -a "$BUILDAH_ISOLATION" = "rootless"; then
    skip "BUILDAH_ISOLATION = $BUILDAH_ISOLATION" and no /usr/bin/fuse-overlayfs present
  elif test "$STORAGE_DRIVER" = "vfs"; then
    skip "skipping overlay test because \$STORAGE_DRIVER = $STORAGE_DRIVER"
  fi
  image=alpine
  mkdir ${TESTDIR}/lower
  touch ${TESTDIR}/lower/foo

  run_buildah from --quiet -v ${TESTDIR}/lower:/lower:O --quiet --signature-policy ${TESTSDIR}/policy.json $image
  cid=$output

  # This should succeed
  run_buildah run $cid ls /lower/foo

  # Create and remove content in the overlay directory, should succeed,
  # resetting the contents between each run.
  run_buildah run $cid touch /lower/bar
  run_buildah run $cid rm /lower/foo

  # This should fail, second runs of containers go back to original
  run_buildah 1 run $cid ls /lower/bar

  # This should fail
  run ls ${TESTDIR}/lower/bar
  [ "$status" -ne 0 ]
}

@test "overlay source permissions and owners" {
  if test \! -e /usr/bin/fuse-overlayfs -a "$BUILDAH_ISOLATION" = "rootless"; then
    skip "BUILDAH_ISOLATION = $BUILDAH_ISOLATION" and no /usr/bin/fuse-overlayfs present
  elif test "$STORAGE_DRIVER" = "vfs"; then
    skip "skipping overlay test because \$STORAGE_DRIVER = $STORAGE_DRIVER"
  fi
  image=alpine
  mkdir -m 770 ${TESTDIR}/lower
  chown 1:1 ${TESTDIR}/lower
  permission=$(stat -c "%a %u %g" ${TESTDIR}/lower)
  run_buildah from --quiet -v ${TESTDIR}/lower:/tmp/test:O --quiet --signature-policy ${TESTSDIR}/policy.json $image
  cid=$output

  # This should succeed
  run_buildah run $cid sh -c 'stat -c "%a %u %g" /tmp/test'
  expect_output "$permission"

  # Create and remove content in the overlay directory, should succeed
  touch ${TESTDIR}/lower/foo
  run_buildah run $cid touch /tmp/test/bar
  run_buildah run $cid rm /tmp/test/foo

  # This should fail, second runs of containers go back to original
  run_buildah 1 run $cid ls /tmp/test/bar

  # This should fail since /tmp/test was an overlay, not a bind mount
  run ls ${TESTDIR}/lower/bar
  [ "$status" -ne 0 ]
}
