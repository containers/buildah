#!/usr/bin/env bats

load helpers

@test "info" {
  run_buildah info
  expect_output --substring "host"

  run_buildah info --format='{{.store}}'
  # All of the following keys must be present in results. Order
  # isn't guaranteed, nor is their value, but they must all exist.
  for key in ContainerStore GraphDriverName GraphRoot RunRoot;do
    expect_output --substring "map.*$key:"
  done
}
