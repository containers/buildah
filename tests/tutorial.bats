#!/usr/bin/env bats

load helpers

@test "tutorial-cgroups" {
	# confidence check for the sake of packages that consume our library
	skip_if_no_runtime
	skip_if_cgroupsv1
	skip_if_rootless_environment
	skip_if_chroot

	_prefetch quay.io/libpod/alpine
	run ${TUTORIAL_BINARY}
	buildoutput="$output"
	# shouldn't have the "root" scope in our cgroups
	echo "build output:"
	echo "${output}"
	! grep -q init.scope <<< "$buildoutput"
	run sed -e '0,/^CUT START/d' -e '/^CUT END/,//d' <<< "$buildoutput"
	# should've found a /sys/fs/cgroup with stuff in it
	echo "contents of /sys/fs/cgroup:"
	echo "${output}"
	echo "number of lines: ${#lines[@]}"
	test "${#lines[@]}" -gt 2
}
