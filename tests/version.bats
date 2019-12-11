#!/usr/bin/env bats

load helpers

@test "buildah version test" {
	run_buildah version
}

@test "buildah version current in .spec file Version" {
	if [ ! -d "${TESTSDIR}/../contrib/rpm" ]; then
		skip "No source dir available"
	fi
	run_buildah version
	bversion=$(awk '/^Version:/ { print $NF }' <<< "$output")
	rversion=$(awk '/^Version:/ { print $NF }' < ${TESTSDIR}/../contrib/rpm/buildah.spec)
	echo "bversion=${bversion}"
	echo "rversion=${rversion}"
	test "${bversion}" = "${rversion}" -o "${bversion}" = "${rversion}-dev"
}

@test "buildah version current in .spec file changelog" {
	if [ ! -d "${TESTSDIR}/../contrib/rpm" ]; then
		skip "No source dir available"
	fi
	run_buildah version
	bversion=$(awk '/^Version:/ { print $NF }' <<< "$output")
	grep -A1 ^%changelog ${TESTSDIR}/../contrib/rpm/buildah.spec | grep -q " ${bversion}-"
}
