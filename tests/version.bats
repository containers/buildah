#!/usr/bin/env bats

load helpers

@test "buildah version test" {
	run_buildah version
}

@test "buildah version current in .spec file Version" {
	if [ ! -d "${TESTSDIR}/../contrib/rpm" ]; then
		skip "No source dir available"
	fi
	run_buildah version | awk '/^Version:/ { print $NF }'
	bversion=$output
	rversion=$(cat ${TESTSDIR}/../contrib/rpm/buildah.spec | awk '/^Version:/ { print $NF }')
	test "${bversion}" = "${rversion}" -o "${bversion}" = "${rversion}-dev"
}

@test "buildah version current in .spec file changelog" {
	if [ ! -d "${TESTSDIR}/../contrib/rpm" ]; then
		skip "No source dir available"
	fi
	run_buildah version | awk '/^Version:/ { print $NF }'
	bversion=$output
	grep -A1 ^%changelog ${TESTSDIR}/../contrib/rpm/buildah.spec | grep -q " ${bversion}-"
}
