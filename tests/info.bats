#!/usr/bin/env bats

load helpers

@test "info" {
  run_buildah info | grep "host" | wc -l
  out1=$output
  [ "$out1" -ne "0" ]

  run_buildah info --format='{{.store}}'
  out2=$output
  [ "$out2" != "" ]
}
