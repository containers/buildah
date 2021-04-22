#!/usr/bin/env bats

load helpers

@test "bud overlay storage leaked mount" {
  if test \! -e /usr/bin/fuse-overlayfs -a "$BUILDAH_ISOLATION" = "rootless"; then
    skip "BUILDAH_ISOLATION = $BUILDAH_ISOLATION" and no /usr/bin/fuse-overlayfs present
  fi

  target=pull
  run_buildah 125 --storage-driver=overlay bud --signature-policy ${TESTSDIR}/policy.json -t ${target} --pull-never ${TESTSDIR}/bud/pull
  expect_output --substring "image not known"

  leftover=$(mount | grep $TESTDIR | cat)
  if [ -n "$leftover" ]; then
    die "buildah leaked a mount on error: $leftover"
  fi
}
