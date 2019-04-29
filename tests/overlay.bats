#!/usr/bin/env bats

load helpers

@test "overlay specific level" {
  if test -o "$BUILDAH_ISOLATION" = "rootless" ; then
    skip "BUILDAH_ISOLATION = $BUILDAH_ISOLATION"
  fi
  image=alpine
  mkdir ${TESTDIR}/lower
  touch ${TESTDIR}/lower/foo

cid=$(buildah --debug=false from -v ${TESTDIR}/lower:/lower:O --quiet --signature-policy ${TESTSDIR}/policy.json $image)

  # This should succeed
  run_buildah --debug=false run $cid ls /lower/foo

  # Create and remove content in the overlay directory, should succeed
  run_buildah --debug=false run $cid touch /lower/bar
  run_buildah --debug=false run $cid rm /lower/foo

  # This should fail, second runs of containers go back to original
  run_buildah 1 --debug=false run $cid ls /lower/bar

  # This should fail
  run ls ${TESTDIR}/lower/bar
  [ "$status" -ne 0 ]
}
