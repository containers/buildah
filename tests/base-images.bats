#!/usr/bin/env bats

load helpers

# Note the tests below are *not* part of buds.bats since `--base-images` is a
# buildah-only feature that is not shared with Podman; the buds.bats test are
# exercised in Podman's CI.

@test "buildah bud --base-images" {
  run_buildah build --base-images -f ${TESTSDIR}/bud/containerfile/Containerfile
  expect_output " * alpine"

  run_buildah build --base-images -f ${TESTSDIR}/bud/containerfile/Containerfile.in ${TESTSDIR}/bud/containerfile
  # --substring due to accoutn for the cpp logs
  expect_output --substring " * alpine"

  # Make sure that stage-aliases are not considered as base images
  run_buildah build --base-images -f ${TESTSDIR}/bud/multi-stage-builds/Dockerfile.extended
  expect_output " * busybox:latest"

  # Make sure that duplicates are filtered and that "scratch" is not considered a base image
  run_buildah build --base-images -f ${TESTSDIR}/bud/multi-stage-builds/Dockerfile.many
  expect_output " * alpine
 * busybox
 * fedora"
}
