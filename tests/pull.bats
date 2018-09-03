#!/usr/bin/env bats

load helpers

@test "pull-flags-order-verification" {
  run buildah pull image1 --tls-verify
  check_options_flag_err "--tls-verify"

  run buildah pull image1 --authfile=/tmp/somefile
  check_options_flag_err "--authfile=/tmp/somefile"

  run buildah pull image1 -q --cred bla:bla --authfile=/tmp/somefile
  check_options_flag_err "-q"
}
