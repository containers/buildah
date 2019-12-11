#!/usr/bin/env bats

load helpers

@test "info" {
  run_buildah info
  expect_output --substring "host"

  run_buildah info --format='{{.store}}'
  expect_output --substring 'map.*ContainerStore.*GraphDriverName.*GraphRoot:.*RunRoot:'
}
