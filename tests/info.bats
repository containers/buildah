#!/usr/bin/env bats

load helpers

@test "info" {
  out1=$(buildah info | grep "host" | wc -l)
  [ "$out1" -ne "0" ]

  out2=$(buildah info --format='{{.store}}')
  [ "$out2" != "" ]
}
