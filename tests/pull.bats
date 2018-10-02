#!/usr/bin/env bats

load helpers

@test "pull-flags-order-verification" {
  run buildah pull image1 --tls-verify
  echo "$output"
  check_options_flag_err "--tls-verify"

  run buildah pull image1 --authfile=/tmp/somefile
  echo "$output"
  check_options_flag_err "--authfile=/tmp/somefile"

  run buildah pull image1 -q --cred bla:bla --authfile=/tmp/somefile
  echo "$output"
  check_options_flag_err "-q"
}

@test "pull-blocked" {
  run buildah --registries-conf ${TESTSDIR}/registries.conf.block pull --signature-policy ${TESTSDIR}/policy.json docker.io/alpine
  echo "$output"
  [ "$status" -ne 0 ]
  [[ "$output" =~ "is blocked by configuration" ]]
  run buildah --registries-conf ${TESTSDIR}/registries.conf       pull --signature-policy ${TESTSDIR}/policy.json docker.io/alpine
  echo "$output"
  [ "$status" -eq 0 ]
}
