#!/usr/bin/env bats

load helpers

@test "buildah version test" {
	run buildah version
	echo "$output"
	[ "$status" -eq 0 ]
}

@test "buildah version up to date in .spec file" {
    skip "cni doesnt version the same"
	run buildah version
	[ "$status" -eq 0 ]
	bversion=$(echo "$output" | awk '/^Version:/ { print $NF }')
	rversion=$(cat ${TESTSDIR}/../contrib/rpm/buildah.spec | awk '/^Version:/ { print $NF }')
	test "$bversion" = "$rversion"
}
