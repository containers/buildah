#!/usr/bin/env bats

load helpers

@test "buildah version test" {
	run_buildah version
}

@test "buildah version current in .spec file Version" {
	if [ ! -d "${TESTSDIR}/../contrib/rpm" ]; then
		skip "No source dir available"
	fi
	bversion=$(buildah version | awk '/^Version:/ { print $NF }' | sed 's/-dev/dev/')
	rversion=$(cat ${TESTSDIR}/../contrib/rpm/buildah.spec | awk '/^Version:/ { print $NF }')
	run test "${bversion}" = "${rversion}"
	[ "$status" -eq 0 ]
}

@test "buildah version current in .spec file changelog" {
	if [ ! -d "${TESTSDIR}/../contrib/rpm" ]; then
		skip "No source dir available"
	fi
	bversion=$(buildah version | awk '/^Version:/ { print $NF }' | sed 's/-dev/dev/')
	run bash -c "grep -A1 ^%changelog ${TESTSDIR}/../contrib/rpm/buildah.spec | grep -q \" ${bversion}-[0-9]\+$\""
	[ "$status" -eq 0 ]
}
