#!/usr/bin/env bats

load helpers

@test "log-level set to debug" {
  run_buildah --log-level=error --log-level=debug images -q
  expect_output --substring "level=debug "
}

@test "log-level set to info" {
  run_buildah --log-level=error --log-level=info images -q
  expect_output ""
}

@test "log-level set to warn" {
  run_buildah --log-level=error --log-level=warn images -q
  expect_output ""
}

@test "log-level set to error" {
  run_buildah --log-level=error --log-level=error images -q
  expect_output ""
}

@test "log-level set to invalid" {
  run_buildah 1 --log-level=error --log-level=invalid images -q
  expect_output --substring "unable to parse log level"
}
