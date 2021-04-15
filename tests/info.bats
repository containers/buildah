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

@test "logging levels" {
  # check that these logging levels are recognized
  run_buildah --log-level=trace info
  run_buildah --log-level=debug info
  run_buildah --log-level=warn  info
  run_buildah --log-level=info  info
  run_buildah --log-level=error info
  run_buildah --log-level=fatal info
  run_buildah --log-level=panic info
  # check that we reject bogus logging levels
  run_buildah 125 --log-level=telepathic info
  expect_output --substring "unable to parse log level: not a valid logrus Level"
}
