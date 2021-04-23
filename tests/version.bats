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

@test "buildah version current in package" {
	if [ -n "$(command -v rpm)" ]; then
		run rpm -qfi ${BUILDAH_BINARY}
		[ $status -eq 0 ] || skip "buildah binary is not owned by package"
		rversion=$(awk '/^Version/ { print $NF }' <<< "$output")
	elif [ -n "$(command -v dpkg)" ]; then
		run dpkg --search ${BUILDAH_BINARY}
		[ $status -eq 0 ] || skip "buildah binary is not owned by package"
		package=$(awk -F : '{ print $1 }' <<< "${output}")
		run dpkg --status $package
		[ $status -eq 0 ] || skip "buildah binary is not owned by package"
		rversion=$(awk '/^Version/ { print $NF }' <<< "$output")
		rversion=${rversion%%-*}
	else
		skip "No supported package manager"
	fi

	run_buildah version
	bversion=$(awk '/^Version:/ { print $NF }' <<< "$output")
	echo "bversion=${bversion}"
	echo "rversion=${rversion}"
	test "${bversion}" = "${rversion}" -o "${bversion}" = "${rversion}-dev"
}
