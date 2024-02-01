#!/usr/bin/env bats

load helpers

@test "run" {
	skip_if_no_runtime

	_prefetch alpine
	${OCI} --version
	createrandom ${TEST_SCRATCH_DIR}/randomfile
	run_buildah from --quiet --pull=false $WITH_POLICY_JSON alpine
	cid=$output
	run_buildah mount $cid
	root=$output
	run_buildah config --workingdir /tmp $cid
	run_buildah run $cid pwd
	expect_output "/tmp"
	run_buildah config --workingdir /root $cid
	run_buildah run        $cid pwd
	expect_output "/root"
	cp ${TEST_SCRATCH_DIR}/randomfile $root/tmp/
	run_buildah run        $cid cp /tmp/randomfile /tmp/other-randomfile
	test -s $root/tmp/other-randomfile
	cmp ${TEST_SCRATCH_DIR}/randomfile $root/tmp/other-randomfile

	seq 100000 | buildah run $cid -- sh -c 'while read i; do echo $i; done'
}

@test "run--args" {
	skip_if_no_runtime

	_prefetch alpine
	run_buildah from --quiet --pull=false $WITH_POLICY_JSON alpine
	cid=$output

	# This should fail, because buildah run doesn't have a -n flag.
	run_buildah 125 run -n $cid echo test

	# This should succeed, because buildah run stops caring at the --, which is preserved as part of the command.
	run_buildah run $cid echo -- -n test
	expect_output -- "-- -n test"

	# This should succeed, because buildah run stops caring at the --, which is not part of the command.
	run_buildah run $cid -- echo -n -- test
	expect_output -- "-- test"

	# This should succeed, because buildah run stops caring at the --.
	run_buildah run $cid -- echo -- -n test --
	expect_output -- "-- -n test --"

	# This should succeed, because buildah run stops caring at the --.
	run_buildah run $cid -- echo -n "test"
	expect_output "test"
}

@test "run-cmd" {
	skip_if_no_runtime

	_prefetch alpine
	run_buildah from --quiet --pull=false $WITH_POLICY_JSON alpine
	cid=$output
	run_buildah config --workingdir /tmp $cid


	# Configured entrypoint/cmd shouldn't modify behaviour of run with no arguments

	# empty entrypoint, configured cmd, empty run arguments
	run_buildah config --entrypoint "" $cid
	run_buildah config --cmd pwd $cid
	run_buildah 125 run $cid
	expect_output --substring "command must be specified" "empty entrypoint, cmd, no args"

	# empty entrypoint, configured cmd, empty run arguments, end parsing option
	run_buildah config --entrypoint "" $cid
	run_buildah config --cmd pwd $cid
	run_buildah 125 run $cid --
	expect_output --substring "command must be specified" "empty entrypoint, cmd, no args, --"

	# configured entrypoint, empty cmd, empty run arguments
	run_buildah config --entrypoint pwd $cid
	run_buildah config --cmd "" $cid
	run_buildah 125 run $cid
	expect_output --substring "command must be specified" "entrypoint, empty cmd, no args"

	# configured entrypoint, empty cmd, empty run arguments, end parsing option
	run_buildah config --entrypoint pwd $cid
	run_buildah config --cmd "" $cid
	run_buildah 125 run $cid --
	expect_output --substring "command must be specified" "entrypoint, empty cmd, no args, --"

	# configured entrypoint only, empty run arguments
	run_buildah config --entrypoint pwd $cid
	run_buildah 125 run $cid
	expect_output --substring "command must be specified" "entrypoint, no args"

	# configured entrypoint only, empty run arguments, end parsing option
	run_buildah config --entrypoint pwd $cid
	run_buildah 125 run $cid --
	expect_output --substring "command must be specified" "entrypoint, no args, --"

	# configured cmd only, empty run arguments
	run_buildah config --cmd pwd $cid
	run_buildah 125 run $cid
	expect_output --substring "command must be specified" "cmd, no args"

	# configured cmd only, empty run arguments, end parsing option
	run_buildah config --cmd pwd $cid
	run_buildah 125 run $cid --
	expect_output --substring "command must be specified" "cmd, no args, --"

	# configured entrypoint, configured cmd, empty run arguments
	run_buildah config --entrypoint "pwd" $cid
	run_buildah config --cmd "whoami" $cid
	run_buildah 125 run $cid
	expect_output --substring "command must be specified" "entrypoint, cmd, no args"

	# configured entrypoint, configured cmd, empty run arguments, end parsing option
	run_buildah config --entrypoint "pwd" $cid
	run_buildah config --cmd "whoami" $cid
	run_buildah 125 run $cid --
	expect_output --substring "command must be specified"  "entrypoint, cmd, no args"


	# Configured entrypoint/cmd shouldn't modify behaviour of run with argument
	# Note: entrypoint and cmd can be invalid in below tests as they should never execute

	# empty entrypoint, configured cmd, configured run arguments
	run_buildah config --entrypoint "" $cid
	run_buildah config --cmd "/invalid/cmd" $cid
	run_buildah run $cid -- pwd
	expect_output "/tmp" "empty entrypoint, invalid cmd, pwd"

        # configured entrypoint, empty cmd, configured run arguments
        run_buildah config --entrypoint "/invalid/entrypoint" $cid
        run_buildah config --cmd "" $cid
        run_buildah run $cid -- pwd
	expect_output "/tmp" "invalid entrypoint, empty cmd, pwd"

        # configured entrypoint only, configured run arguments
        run_buildah config --entrypoint "/invalid/entrypoint" $cid
        run_buildah run $cid -- pwd
	expect_output "/tmp" "invalid entrypoint, no cmd(??), pwd"

        # configured cmd only, configured run arguments
        run_buildah config --cmd "/invalid/cmd" $cid
        run_buildah run $cid -- pwd
	expect_output "/tmp" "invalid cmd, no entrypoint(??), pwd"

        # configured entrypoint, configured cmd, configured run arguments
        run_buildah config --entrypoint "/invalid/entrypoint" $cid
        run_buildah config --cmd "/invalid/cmd" $cid
        run_buildah run $cid -- pwd
	expect_output "/tmp" "invalid cmd & entrypoint, pwd"
}

# Helper for run-user test. Generates a UID or GID that is not present
# in the given idfile (mounted /etc/passwd or /etc/group)
function random_unused_id() {
    local idfile=$1

    while :;do
        id=$RANDOM
        if ! fgrep -q :$id: $idfile; then
            echo $id
            return
        fi
    done
}

function configure_and_check_user() {
    local setting=$1
    local expect_u=$2
    local expect_g=$3

    run_buildah config -u "$setting" $cid
    run_buildah run -- $cid id -u
    expect_output "$expect_u" "id -u ($setting)"

    run_buildah run -- $cid id -g
    expect_output "$expect_g" "id -g ($setting)"
}

@test "run-user" {
	skip_if_no_runtime

	eval $(go env)
	echo CGO_ENABLED=${CGO_ENABLED}
	if test "$CGO_ENABLED" -ne 1; then
		skip "CGO_ENABLED = '$CGO_ENABLED'"
	fi
	_prefetch alpine
	run_buildah from --quiet --pull=false $WITH_POLICY_JSON alpine
	cid=$output
	run_buildah mount $cid
	root=$output

	testuser=jimbo
	testbogususer=nosuchuser
	testgroup=jimbogroup
	testuid=$(random_unused_id $root/etc/passwd)
	testotheruid=$(random_unused_id $root/etc/passwd)
	testgid=$(random_unused_id $root/etc/group)
	testgroupid=$(random_unused_id $root/etc/group)
	echo "$testuser:x:$testuid:$testgid:Jimbo Jenkins:/home/$testuser:/bin/sh" >> $root/etc/passwd
	echo "$testgroup:x:$testgroupid:" >> $root/etc/group

        configure_and_check_user ""                             0             0
        configure_and_check_user "${testuser}"                  $testuid      $testgid
        configure_and_check_user "${testuid}"                   $testuid      $testgid
        configure_and_check_user "${testuser}:${testgroup}"     $testuid      $testgroupid
        configure_and_check_user "${testuid}:${testgroup}"      $testuid      $testgroupid
        configure_and_check_user "${testotheruid}:${testgroup}" $testotheruid $testgroupid
        configure_and_check_user "${testotheruid}"              $testotheruid 0
        configure_and_check_user "${testuser}:${testgroupid}"   $testuid      $testgroupid
        configure_and_check_user "${testuid}:${testgroupid}"    $testuid      $testgroupid

        run_buildah config -u ${testbogususer} $cid
        run_buildah 125 run -- $cid id -u
        expect_output --substring "unknown user" "id -u (bogus user)"
        run_buildah 125 run -- $cid id -g
        expect_output --substring "unknown user" "id -g (bogus user)"

	ln -vsf /etc/passwd $root/etc/passwd
	run_buildah config -u ${testuser}:${testgroup} $cid
	run_buildah 125 run -- $cid id -u
	echo "$output"
	expect_output --substring "unknown user" "run as unknown user"
}

@test "run --env" {
	skip_if_no_runtime

	_prefetch alpine
	run_buildah from --quiet --pull=false $WITH_POLICY_JSON alpine
	cid=$output
	run_buildah config --env foo=foo $cid

	# Ensure foo=foo from `buildah config`
	run_buildah run $cid -- /bin/sh -c 'echo $foo'
	expect_output "foo"

	# Ensure foo=bar from --env override
	run_buildah run --env foo=bar $cid -- /bin/sh -c 'echo $foo'
	expect_output "bar"

	# Reference foo=baz from process environment
	foo=baz run_buildah run --env foo $cid -- /bin/sh -c 'echo $foo'
	expect_output "baz"

	# Ensure that the --env override did not persist
	run_buildah run $cid -- /bin/sh -c 'echo $foo'
	expect_output "foo"
}

@test "run --group-add" {
	skip_if_no_runtime
        id=$RANDOM

	_prefetch alpine
	run_buildah from --group-add $id --quiet --pull=false $WITH_POLICY_JSON alpine
	cid=$output
	run_buildah run $cid id -G
	expect_output --substring "$id"

	if is_rootless && has_supplemental_groups; then
	   run_buildah from --group-add keep-groups --quiet --pull=false $WITH_POLICY_JSON alpine
	   cid=$output
	   run_buildah run $cid id -G
	   expect_output --substring "65534"
	fi
}

@test "run --hostname" {
	skip_if_no_runtime

	_prefetch alpine
	${OCI} --version
	run_buildah from --quiet --pull=false $WITH_POLICY_JSON alpine
	cid=$output
	run_buildah run $cid hostname
	[ "$output" != "foobar" ]
	run_buildah run --hostname foobar $cid hostname
	expect_output "foobar"
}

@test "run should also override /etc/hostname" {
	skip_if_no_runtime

	_prefetch alpine
	${OCI} --version
	run_buildah from --quiet --pull=false $WITH_POLICY_JSON alpine
	cid=$output
	run_buildah run --hostname foobar $cid hostname
	expect_output "foobar"
	hostname=$output
	run_buildah run --hostname foobar $cid cat /etc/hostname
	expect_output $hostname
	run_buildah from --quiet --pull=false $WITH_POLICY_JSON alpine
	cid=$output
	run_buildah inspect --format "{{ .ContainerID }}" $cid
	id=$output
	run_buildah run $cid cat /etc/hostname
	expect_output "${id:0:12}"
	run_buildah run --no-hostname $cid cat /etc/hostname
	expect_output 'localhost'
}

@test "run --volume" {
	skip_if_no_runtime

	zflag=
	if which selinuxenabled > /dev/null 2> /dev/null ; then
		if selinuxenabled ; then
			zflag=z
		fi
	fi
	${OCI} --version
	_prefetch alpine
	run_buildah from --quiet --pull=false $WITH_POLICY_JSON alpine
	cid=$output
	mkdir -p ${TEST_SCRATCH_DIR}/was-empty
	# As a baseline, this should succeed.
	run_buildah run -v ${TEST_SCRATCH_DIR}/was-empty:/var/not-empty${zflag:+:${zflag}}     $cid touch /var/not-empty/testfile
	# Parsing options that with comma, this should succeed.
	run_buildah run -v ${TEST_SCRATCH_DIR}/was-empty:/var/not-empty:rw,rshared${zflag:+,${zflag}}     $cid touch /var/not-empty/testfile
	# If we're parsing the options at all, this should be read-only, so it should fail.
	run_buildah 1 run -v ${TEST_SCRATCH_DIR}/was-empty:/var/not-empty:ro${zflag:+,${zflag}} $cid touch /var/not-empty/testfile
	# Even if the parent directory doesn't exist yet, this should succeed.
	run_buildah run -v ${TEST_SCRATCH_DIR}/was-empty:/var/multi-level/subdirectory        $cid touch /var/multi-level/subdirectory/testfile
	# And check the same for file volumes.
	run_buildah run -v ${TEST_SCRATCH_DIR}/was-empty/testfile:/var/different-multi-level/subdirectory/testfile        $cid touch /var/different-multi-level/subdirectory/testfile
	# And check the same for file volumes.
	# Make sure directories show up inside of container on builtin mounts
	run_buildah run -v ${TEST_SCRATCH_DIR}/was-empty:/run/secrets/testdir $cid ls -ld /run/secrets/testdir
}

@test "run overlay --volume with custom upper and workdir" {
	skip_if_no_runtime

	zflag=
	if which selinuxenabled > /dev/null 2> /dev/null ; then
		if selinuxenabled ; then
			zflag=z
		fi
	fi
	${OCI} --version
	_prefetch alpine
	run_buildah from --quiet --pull=false $WITH_POLICY_JSON alpine
	cid=$output
	mkdir -p ${TEST_SCRATCH_DIR}/upperdir
	mkdir -p ${TEST_SCRATCH_DIR}/workdir
	mkdir -p ${TEST_SCRATCH_DIR}/lower

	echo 'hello' >> ${TEST_SCRATCH_DIR}/lower/hello

	# As a baseline, this should succeed.
	run_buildah run -v ${TEST_SCRATCH_DIR}/lower:/test:O,upperdir=${TEST_SCRATCH_DIR}/upperdir,workdir=${TEST_SCRATCH_DIR}/workdir${zflag:+:${zflag}}  $cid cat /test/hello
	expect_output "hello"
	run_buildah run -v ${TEST_SCRATCH_DIR}/lower:/test:O,upperdir=${TEST_SCRATCH_DIR}/upperdir,workdir=${TEST_SCRATCH_DIR}/workdir${zflag:+:${zflag}}  $cid sh -c 'echo "world" > /test/world'

	#upper dir should persist content
	result="$(cat ${TEST_SCRATCH_DIR}/upperdir/world)"
	test "$result" == "world"
}

@test "run --volume with U flag" {
  skip_if_no_runtime

  # Create source volume.
  mkdir ${TEST_SCRATCH_DIR}/testdata

  # Create the container.
  _prefetch alpine
  run_buildah from --quiet --pull=false $WITH_POLICY_JSON alpine
  ctr="$output"

  # Test user can create file in the mounted volume.
  run_buildah run --user 888:888 --volume ${TEST_SCRATCH_DIR}/testdata:/mnt:z,U "$ctr" touch /mnt/testfile1.txt

  # Test created file has correct UID and GID ownership.
  run_buildah run --user 888:888 --volume ${TEST_SCRATCH_DIR}/testdata:/mnt:z,U "$ctr" stat -c "%u:%g" /mnt/testfile1.txt
  expect_output "888:888"
}

@test "run --user and verify gid in supplemental groups" {
  skip_if_no_runtime

  # Create the container.
  _prefetch alpine
  run_buildah from --quiet --pull=false $WITH_POLICY_JSON alpine
  ctr="$output"

  # Run with uid:gid 1000:1000 and verify if gid is present in additional groups
  run_buildah run --user 1000:1000 "$ctr" cat /proc/self/status
  # gid 1000 must be in additional/supplemental groups
  expect_output --substring "Groups:	1000 "
}

@test "run --workingdir" {
	skip_if_no_runtime

	_prefetch alpine
	run_buildah from --quiet --pull=false $WITH_POLICY_JSON alpine
	cid=$output
	run_buildah run $cid pwd
	expect_output "/"
	run_buildah run --workingdir /bin $cid pwd
	expect_output "/bin"
	# Ensure the /bin workingdir override did not persist
	run_buildah run $cid pwd
	expect_output "/"
}

@test "run --mount" {
	skip_if_no_runtime

	zflag=
	if which selinuxenabled > /dev/null 2> /dev/null ; then
		if selinuxenabled ; then
			zflag=z
		fi
	fi
	${OCI} --version
	_prefetch alpine
	run_buildah from --quiet --pull=false $WITH_POLICY_JSON alpine
	cid=$output
	mkdir -p ${TEST_SCRATCH_DIR}/was:empty
	# As a baseline, this should succeed.
	run_buildah run --mount type=tmpfs,dst=/var/tmpfs-not-empty                                           $cid touch /var/tmpfs-not-empty/testfile
	run_buildah run --mount type=bind,src=${TEST_SCRATCH_DIR}/was:empty,dst=/var/not-empty,rw${zflag:+,${zflag}}      $cid touch /var/not-empty/testfile
	# If we're parsing the options at all, this should be read-only, so it should fail.
	run_buildah 1 run --mount type=bind,src=${TEST_SCRATCH_DIR}/was:empty,dst=/var/not-empty,ro${zflag:+,${zflag}} $cid touch /var/not-empty/testfile
	# Even if the parent directory doesn't exist yet, this should succeed.
	run_buildah run --mount type=bind,src=${TEST_SCRATCH_DIR}/was:empty,dst=/var/multi-level/subdirectory,rw          $cid touch /var/multi-level/subdirectory/testfile
	# And check the same for file volumes.
	run_buildah run --mount type=bind,src=${TEST_SCRATCH_DIR}/was:empty/testfile,dst=/var/different-multi-level/subdirectory/testfile,rw        $cid touch /var/different-multi-level/subdirectory/testfile
}

@test "run --mount=type=bind with from like buildkit" {
	skip_if_no_runtime
	zflag=
	if which selinuxenabled > /dev/null 2> /dev/null ; then
		if selinuxenabled ; then
			skip "skip if selinux enabled, since stages have different selinux label"
		fi
	fi
	run_buildah build -t buildkitbase $WITH_POLICY_JSON -f $BUDFILES/buildkit-mount-from/Dockerfilebuildkitbase $BUDFILES/buildkit-mount-from/
	_prefetch alpine
	run_buildah from --quiet --pull=false $WITH_POLICY_JSON alpine
	cid=$output
	run_buildah run --mount type=bind,source=.,from=buildkitbase,target=/test,z  $cid cat /test/hello
	expect_output --substring "hello"
	run_buildah rmi -f buildkitbase
}

@test "run --mount=type=cache like buildkit" {
	skip_if_no_runtime
	zflag=
	if which selinuxenabled > /dev/null 2> /dev/null ; then
		if selinuxenabled ; then
			skip "skip if selinux enabled, since stages have different selinux label"
		fi
	fi
	_prefetch alpine
	run_buildah from --quiet --pull=false $WITH_POLICY_JSON alpine
	cid=$output
	run_buildah run --mount type=cache,target=/test,z  $cid sh -c 'echo "hello" > /test/hello && cat /test/hello'
	run_buildah run --mount type=cache,target=/test,z  $cid cat /test/hello
	expect_output --substring "hello"
}

@test "run symlinks" {
	skip_if_no_runtime

	${OCI} --version
	_prefetch alpine
	run_buildah from --quiet --pull=false $WITH_POLICY_JSON alpine
	cid=$output
	mkdir -p ${TEST_SCRATCH_DIR}/tmp
	ln -s tmp ${TEST_SCRATCH_DIR}/tmp2
	export TMPDIR=${TEST_SCRATCH_DIR}/tmp2
	run_buildah run $cid id
}

@test "run --cap-add/--cap-drop" {
	skip_if_no_runtime

	${OCI} --version
	_prefetch alpine
	run_buildah from --quiet --pull=false $WITH_POLICY_JSON alpine
	cid=$output
	# Try with default caps.
	run_buildah run $cid grep ^CapEff /proc/self/status
	defaultcaps="$output"
	# Try adding DAC_OVERRIDE.
	run_buildah run --cap-add CAP_DAC_OVERRIDE $cid grep ^CapEff /proc/self/status
	addedcaps="$output"
	# Try dropping DAC_OVERRIDE.
	run_buildah run --cap-drop CAP_DAC_OVERRIDE $cid grep ^CapEff /proc/self/status
	droppedcaps="$output"
	# Okay, now the "dropped" and "added" should be different.
	test "$addedcaps" != "$droppedcaps"
	# And one or the other should be different from the default, with the other being the same.
	if test "$defaultcaps" == "$addedcaps" ; then
		test "$defaultcaps" != "$droppedcaps"
	fi
	if test "$defaultcaps" == "$droppedcaps" ; then
		test "$defaultcaps" != "$addedcaps"
	fi
}

@test "Check if containers run with correct open files/processes limits" {
        skip_if_rootless_environment
	skip_if_no_runtime

	# we need to not use the list of limits that are set in our default
	# ${TEST_SOURCES}/containers.conf for the sake of other tests, and override
	# any that might be picked up from system-wide configuration
	echo '[containers]' > ${TEST_SCRATCH_DIR}/containers.conf
	echo 'default_ulimits = []' >> ${TEST_SCRATCH_DIR}/containers.conf
	export CONTAINERS_CONF=${TEST_SCRATCH_DIR}/containers.conf

	_prefetch alpine
	maxpids=$(cat /proc/sys/kernel/pid_max)
	run_buildah from --quiet --pull=false $WITH_POLICY_JSON alpine
	cid=$output
	run_buildah run $cid awk '/open files/{print $4}' /proc/self/limits
	expect_output 1024 "limits: open files (unlimited)"
	run_buildah run $cid awk '/processes/{print $3}' /proc/self/limits
	expect_output ${maxpids} "limits: processes (unlimited)"
	run_buildah rm $cid

	run_buildah from --quiet --ulimit nofile=300:400 --pull=false $WITH_POLICY_JSON alpine
	cid=$output
	run_buildah run $cid awk '/open files/{print $4}' /proc/self/limits
	expect_output "300" "limits: open files (w/file limit)"
	run_buildah run $cid awk '/processes/{print $3}' /proc/self/limits
	expect_output ${maxpids} "limits: processes (w/file limit)"
	run_buildah rm $cid

	run_buildah from --quiet --ulimit nproc=100:200 --ulimit nofile=300:400 --pull=false $WITH_POLICY_JSON alpine
	cid=$output
	run_buildah run $cid awk '/open files/{print $4}' /proc/self/limits
	expect_output "300" "limits: open files (w/file & proc limits)"
	run_buildah run $cid awk '/processes/{print $3}' /proc/self/limits
	expect_output "100" "limits: processes (w/file & proc limits)"

	unset CONTAINERS_CONF
}

@test "run-builtin-volume-omitted" {
	# This image is known to include a volume, but not include the mountpoint
	# in the image.
	run_buildah from --quiet --pull=false $WITH_POLICY_JSON quay.io/libpod/registry:volume_omitted
	cid=$output
	run_buildah mount $cid
	mnt=$output
	# By default, the mountpoint should not be there.
	run test -d "$mnt"/var/lib/registry
	echo "$output"
	[ "$status" -ne 0 ]
	# We'll create the mountpoint for "run".
	run_buildah run $cid ls -1 /var/lib
        expect_output --substring "registry"

	# Double-check that the mountpoint is there.
	test -d "$mnt"/var/lib/registry
}

@test "run-exit-status" {
	skip_if_no_runtime

	_prefetch alpine
	run_buildah from --quiet --pull=false $WITH_POLICY_JSON alpine
	cid=$output
	run_buildah 42 run ${cid} sh -c 'exit 42'
}

@test "run-exit-status on non executable" {
	skip_if_no_runtime

	_prefetch alpine
	run_buildah from --quiet --pull=false $WITH_POLICY_JSON alpine
	cid=$output
	run_buildah 1 run ${cid} /etc
}

@test "Verify /run/.containerenv exist" {
        skip_if_rootless_environment
	skip_if_no_runtime

	_prefetch alpine
	run_buildah from --quiet --pull=false $WITH_POLICY_JSON alpine
	cid=$output
	# test a standard mount to /run/.containerenv
	run_buildah run $cid ls -1 /run/.containerenv
	expect_output --substring "/run/.containerenv"

	run_buildah run $cid sh -c '. /run/.containerenv; echo $engine'
	expect_output --substring "buildah"

	run_buildah run $cid sh -c '. /run/.containerenv; echo $name'
	expect_output "alpine-working-container"

	run_buildah run $cid sh -c '. /run/.containerenv; echo $image'
	expect_output --substring "alpine:latest"

	rootless=0
	if ["$(id -u)" -ne 0 ]; then
		rootless=1
	fi

	run_buildah run $cid sh -c '. /run/.containerenv; echo $rootless'
	expect_output ${rootless}
}

@test "run-device" {
	skip_if_no_runtime

	_prefetch alpine
	run_buildah from --quiet --pull=false --device /dev/fuse $WITH_POLICY_JSON alpine
	cid=$output
	run_buildah 0 run ${cid} ls /dev/fuse

	run_buildah from --quiet --pull=false --device /dev/fuse:/dev/fuse:rm $WITH_POLICY_JSON alpine
	cid=$output
	run_buildah 0 run ${cid} ls /dev/fuse

	run_buildah from --quiet --pull=false --device /dev/fuse:/dev/fuse:rwm $WITH_POLICY_JSON alpine
	cid=$output
	run_buildah 0 run ${cid} ls /dev/fuse

}

@test "run-device-Rename" {
	skip_if_rootless_environment
	skip_if_no_runtime
	skip_if_chroot
	skip_if_rootless

	_prefetch alpine
	run_buildah from --quiet --pull=false --device /dev/fuse:/dev/fuse1 $WITH_POLICY_JSON alpine
	cid=$output
	run_buildah 0 run ${cid} ls /dev/fuse1
}

@test "run check /etc/hosts" {
        skip_if_rootless_environment
	skip_if_no_runtime
	skip_if_in_container

	${OCI} --version
	_prefetch debian

	local hostname=h-$(random_string)

	run_buildah from --quiet --pull=false $WITH_POLICY_JSON debian
	cid=$output
	run_buildah 125 run --network=bogus $cid cat /etc/hosts
	expect_output --substring "unable to find network with name or ID bogus: network not found"
	run_buildah run --hostname $hostname $cid cat /etc/hosts
	expect_output --substring "(10.88.*|10.0.2.100)[[:blank:]]$hostname $cid"
	ip=$(hostname -I | cut -f 1 -d " ")
	expect_output --substring "$ip.*host.containers.internal"

	hosts="127.0.0.5 host1
127.0.0.6 host2"
	base_hosts_file="$TEST_SCRATCH_DIR/base_hosts"
	echo "$hosts" > "$base_hosts_file"
	containers_conf_file="$TEST_SCRATCH_DIR/containers.conf"
	echo -e "[containers]\nbase_hosts_file = \"$base_hosts_file\"" > "$containers_conf_file"
	CONTAINERS_CONF="$containers_conf_file" run_buildah run --hostname $hostname $cid cat /etc/hosts
	expect_output --substring "127.0.0.5[[:blank:]]host1"
	expect_output --substring "127.0.0.6[[:blank:]]host2"
	expect_output --substring "(10.88.*|10.0.2.100)[[:blank:]]$hostname $cid"

	# now check that hostname from base file is not overwritten
	CONTAINERS_CONF="$containers_conf_file" run_buildah run --hostname host1 $cid cat /etc/hosts
	expect_output --substring "127.0.0.5[[:blank:]]host1"
	expect_output --substring "127.0.0.6[[:blank:]]host2"
	expect_output --substring "(10.88.*|10.0.2.100)[[:blank:]]$cid"
	assert "$output" !~ "(10.88.*|10.0.2.100)[[:blank:]]host1 $cid" "Container IP should not contain host1"

	# check slirp4netns sets correct hostname with another cidr
	run_buildah run --network slirp4netns:cidr=192.168.2.0/24 --hostname $hostname $cid cat /etc/hosts
	expect_output --substring "192.168.2.100[[:blank:]]$hostname $cid"

	run_buildah run --network=container $cid cat /etc/hosts
	m=$(buildah mount $cid)
	run cat $m/etc/hosts
	[ "$status" -eq 0 ]
	expect_output --substring ""
	run_buildah rm -a

	run_buildah from --quiet --pull=false $WITH_POLICY_JSON debian
	cid=$output
	run_buildah run --network=host --hostname $hostname $cid cat /etc/hosts
	assert "$output" =~ "$ip[[:blank:]]$hostname"
	hostOutput=$output
	m=$(buildah mount $cid)
	run cat $m/etc/hosts
	[ "$status" -eq 0 ]
	expect_output --substring ""
	run_buildah run --network=host --no-hosts $cid cat /etc/hosts
	[ "$output" != "$hostOutput" ]
	# --isolation chroot implies host networking so check for the correct hosts entry
	run_buildah run --isolation chroot --hostname $hostname $cid cat /etc/hosts
	assert "$output" =~ "$ip[[:blank:]]$hostname"
	run_buildah rm -a

	run_buildah from --quiet --pull=false $WITH_POLICY_JSON debian
	cid=$output
	run_buildah run --network=none $cid sh -c 'echo "110.110.110.0 fake_host" >> /etc/hosts; cat /etc/hosts'
	expect_output "110.110.110.0 fake_host"
	m=$(buildah mount $cid)
	run cat $m/etc/hosts
	[ "$status" -eq 0 ]
	expect_output "110.110.110.0 fake_host"
	run_buildah rm -a
}

@test "run check /etc/hosts with --network pasta" {
	skip_if_no_runtime
	skip_if_chroot
	skip_if_root_environment "pasta only works rootless"

	# FIXME: unskip when we have a new pasta version with:
	# https://archives.passt.top/passt-dev/20230623082531.25947-2-pholzing@redhat.com/
	skip "pasta bug prevents this from working"

	run_buildah from --quiet --pull=false $WITH_POLICY_JSON debian
	cid=$output

	local hostname=h-$(random_string)
	ip=$(hostname -I | cut -f 1 -d " ")
	run_buildah run --network pasta --hostname $hostname $cid cat /etc/hosts
	assert "$output" =~ "$ip[[:blank:]]$hostname $cid" "--network pasta adds correct hostname"

	# check with containers.conf setting
	echo -e "[network]\ndefault_rootless_network_cmd = \"pasta\"" > ${TEST_SCRATCH_DIR}/containers.conf
	CONTAINERS_CONF_OVERRIDE=${TEST_SCRATCH_DIR}/containers.conf run_buildah run --hostname $hostname $cid cat /etc/hosts
	assert "$output" =~ "$ip[[:blank:]]$hostname $cid" "default_rootless_network_cmd = \"pasta\" works"
}

@test "run check /etc/resolv.conf" {
        skip_if_rootless_environment
	skip_if_no_runtime

	${OCI} --version
	_prefetch alpine

	# Make sure to read the correct /etc/resolv.conf file in case of systemd-resolved.
	resolve_file=$(readlink -f /etc/resolv.conf)
	if [[ "$resolve_file" == "/run/systemd/resolve/stub-resolv.conf" ]]; then
		resolve_file="/run/systemd/resolve/resolv.conf"
	fi

	run grep nameserver $resolve_file
	# filter out 127... nameservers
	run grep -v "nameserver 127." <<< "$output"
	nameservers="$output"
	# in case of rootless add extra slirp4netns nameserver
	if is_rootless; then
		nameservers="nameserver 10.0.2.3
$output"
	fi
	run_buildah from --quiet --pull=false $WITH_POLICY_JSON alpine
	cid=$output
	run_buildah run --network=private $cid grep nameserver /etc/resolv.conf
	# check that no 127... nameserver is in resolv.conf
	assert "$output" !~ "^nameserver 127." "Container contains local nameserver"
	assert "$nameservers" "Container nameservers match correct host nameservers"
	if ! is_rootless; then
		run_buildah mount $cid
		assert "$output" != ""
		assert "$(< $output/etc/resolv.conf)" = "" "resolv.conf is empty"
	fi
	run_buildah rm -a

	run grep nameserver /etc/resolv.conf
	nameservers="$output"
	run_buildah from --quiet --pull=false $WITH_POLICY_JSON alpine
	cid=$output
	run_buildah run --isolation=chroot --network=host $cid grep nameserver /etc/resolv.conf
	assert "$nameservers" "Container nameservers match the host nameservers"
	if ! is_rootless; then
		run_buildah mount $cid
		assert "$output" != ""
		assert "$(< $output/etc/resolv.conf)" = "" "resolv.conf is empty"
	fi
	run_buildah rm -a

	run_buildah from --quiet --pull=false $WITH_POLICY_JSON alpine
	cid=$output
	run_buildah 125 run --isolation=chroot --network=none $cid sh -c 'echo "nameserver 110.110.0.110" >> /etc/resolv.conf; cat /etc/resolv.conf'
        expect_output --substring "cannot set --network other than host with --isolation chroot"
	run_buildah rm -a
}

@test "run --network=none and --isolation chroot must conflict" {
	skip_if_no_runtime

	run_buildah from --quiet --pull=false $WITH_POLICY_JSON alpine
	cid=$output
	# should fail by default
	run_buildah 125 run --isolation=chroot --network=none $cid wget google.com
        expect_output --substring "cannot set --network other than host with --isolation chroot"
}

@test "run --network=private must mount a fresh /sys" {
	skip_if_no_runtime

	run_buildah from --quiet --pull=false $WITH_POLICY_JSON alpine
	cid=$output
        # verify there is no /sys/kernel/security in the container, that would mean /sys
        # was bind mounted from the host.
	run_buildah 1 run --network=private $cid grep /sys/kernel/security /proc/self/mountinfo
}

@test "run --network should override build --network" {
	skip_if_no_runtime

	run_buildah from --network=none --quiet --pull=false $WITH_POLICY_JSON alpine
	cid=$output
	# should fail by default
	run_buildah 1 run $cid wget google.com
	expect_output --substring "bad"
	# try pinging external website
	run_buildah run --network=private $cid wget google.com
	expect_output --substring "index.html"
	run_buildah rm -a
}

@test "run --user" {
	skip_if_no_runtime

	_prefetch alpine
	run_buildah from --quiet --pull=false $WITH_POLICY_JSON alpine
	cid=$output
	run_buildah run --user sync $cid whoami
	expect_output "sync"
	run_buildah 125 run --user noexist $cid whoami
	expect_output --substring "unknown user error"
}

@test "run --runtime --runtime-flag" {
	skip_if_in_container
	skip_if_no_runtime

	_prefetch alpine

	# Use seccomp to make crun output a warning message because crun writes few logs.
	cat > ${TEST_SCRATCH_DIR}/seccomp.json << _EOF
{
    "defaultAction": "SCMP_ACT_ALLOW",
    "syscalls": [
        {
	        "name": "unknown",
			"action": "SCMP_ACT_KILL"
	    }
    ]
}
_EOF
	run_buildah from --security-opt seccomp=${TEST_SCRATCH_DIR}/seccomp.json --quiet --pull=false $WITH_POLICY_JSON alpine
	cid=$output

	local found_runtime=

	if [ -n "$(command -v runc)" ]; then
		found_runtime=y
		run_buildah '?' run --runtime=runc --runtime-flag=debug $cid true
		if [ "$status" -eq 0 ]; then
			assert "$output" != "" "Output from running 'true' with --runtime-flag=debug"
		else
			# runc fully supports cgroup v2 (unified mode) since v1.0.0-rc93.
			# older runc doesn't work on cgroup v2.
			expect_output --substring "this version of runc doesn't work on cgroups v2" "should fail by unsupportability for cgroupv2"
		fi
	fi

	if [ -n "$(command -v crun)" ]; then
		found_runtime=y
		run_buildah run --runtime=crun --runtime-flag=log=${TEST_SCRATCH_DIR}/oci-log $cid true
		if test \! -e ${TEST_SCRATCH_DIR}/oci-log; then
			die "the expected file ${TEST_SCRATCH_DIR}/oci-log was not created"
		fi
	fi

	if [ -z "${found_runtime}" ]; then
		skip "Did not find 'runc' nor 'crun' in \$PATH - could not run this test!"
	fi

}

@test "run --terminal" {
	skip_if_no_runtime

	_prefetch alpine
	run_buildah from --quiet --pull=false $WITH_POLICY_JSON alpine
	cid=$output
	run_buildah run --terminal=true $cid ls --color=auto
	colored="$output"
	run_buildah run --terminal=false $cid ls --color=auto
	uncolored="$output"
	[ "$colored" != "$uncolored" ]
}

@test "rootless on cgroupv2 and systemd runs under user.slice" {
	skip_if_no_runtime
	skip_if_cgroupsv1
	skip_if_in_container
	skip_if_root_environment
	if test "$DBUS_SESSION_BUS_ADDRESS" = ""; then
		skip "$test does not work when DBUS_SESSION_BUS_ADDRESS is not defined"
	fi
	_prefetch alpine

	run_buildah from --quiet --pull=false $WITH_POLICY_JSON alpine
	cid=$output
	run_buildah run --cgroupns=host $cid cat /proc/self/cgroup
	expect_output --substring "/user.slice/"
}

@test "run-inheritable-capabilities" {
	skip_if_no_runtime

	_prefetch alpine

	run_buildah from --quiet --pull=false $WITH_POLICY_JSON alpine
	cid=$output
	run_buildah run $cid grep ^CapInh: /proc/self/status
	expect_output "CapInh:	0000000000000000"
	run_buildah run --cap-add=ALL $cid grep ^CapInh: /proc/self/status
	expect_output "CapInh:	0000000000000000"
}

@test "run masks" {
	skip_if_no_runtime

	_prefetch alpine

	run_buildah from --quiet --pull=false $WITH_POLICY_JSON alpine
	cid=$output
	for mask in /proc/acpi /proc/kcore /proc/keys /proc/latency_stats /proc/sched_debug /proc/scsi /proc/timer_list /proc/timer_stats /sys/dev/block /sys/devices/virtual/powercap /sys/firmware /sys/fs/selinux; do
	        if test -d $mask; then
		   run_buildah run $cid ls $mask
		   expect_output "" "Directories should be empty"
		fi
		if test -f $mask; then
		   run_buildah run $cid cat $mask
		   expect_output "" "Directories should be empty"
		fi
	done
}

@test "empty run statement doesn't crash" {
	skip_if_no_runtime

	_prefetch alpine

	cd ${TEST_SCRATCH_DIR}

	printf 'FROM alpine\nRUN \\\n echo && echo' > Dockerfile
	run_buildah bud --pull=false --layers .

	printf 'FROM alpine\nRUN\n echo && echo' > Dockerfile
	run_buildah ? bud --pull=false --layers .
        expect_output --substring -- "-c requires an argument"
}
