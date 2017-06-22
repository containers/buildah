#!/usr/bin/env bats

load helpers


@test "buildah version test" {
	run buildah version
	echo "$output"
	[ "$status" -eq 0 ]

}
