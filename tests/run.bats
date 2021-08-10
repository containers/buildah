#!/usr/bin/env bats

load helpers

@test "run" {
	skip_if_no_runtime

	_prefetch alpine
	${OCI} --version
	createrandom ${TESTDIR}/randomfile
	run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
	cid=$output
	run_buildah mount $cid
	root=$output
	run_buildah config --workingdir /tmp $cid
	run_buildah run $cid pwd
	expect_output "/tmp"
	run_buildah config --workingdir /root $cid
	run_buildah run        $cid pwd
	expect_output "/root"
	cp ${TESTDIR}/randomfile $root/tmp/
	run_buildah run        $cid cp /tmp/randomfile /tmp/other-randomfile
	test -s $root/tmp/other-randomfile
	cmp ${TESTDIR}/randomfile $root/tmp/other-randomfile

	seq 100000 | buildah run $cid -- sh -c 'while read i; do echo $i; done'
}

@test "run--args" {
	skip_if_no_runtime

	_prefetch alpine
	run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
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
	run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
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
	run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
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
	run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
	cid=$output
	run_buildah config --env foo=foo $cid
	# Ensure foo=foo from `buildah config`
	run_buildah run $cid -- /bin/sh -c 'echo $foo'
	expect_output "foo"
	# Ensure foo=bar from --env override
	run_buildah run --env foo=bar $cid -- /bin/sh -c 'echo $foo'
	expect_output "bar"
	# Ensure that the --env override did not persist
	run_buildah run $cid -- /bin/sh -c 'echo $foo'
	expect_output "foo"
}

@test "run --hostname" {
	skip_if_no_runtime

	_prefetch alpine
	${OCI} --version
	run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
	cid=$output
	run_buildah run $cid hostname
	[ "$output" != "foobar" ]
	run_buildah run --hostname foobar $cid hostname
	expect_output "foobar"
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
	run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
	cid=$output
	mkdir -p ${TESTDIR}/was-empty
	# As a baseline, this should succeed.
	run_buildah run -v ${TESTDIR}/was-empty:/var/not-empty${zflag:+:${zflag}}     $cid touch /var/not-empty/testfile
	# Parsing options that with comma, this should succeed.
	run_buildah run -v ${TESTDIR}/was-empty:/var/not-empty:rw,rshared${zflag:+,${zflag}}     $cid touch /var/not-empty/testfile
	# If we're parsing the options at all, this should be read-only, so it should fail.
	run_buildah 1 run -v ${TESTDIR}/was-empty:/var/not-empty:ro${zflag:+,${zflag}} $cid touch /var/not-empty/testfile
	# Even if the parent directory doesn't exist yet, this should succeed.
	run_buildah run -v ${TESTDIR}/was-empty:/var/multi-level/subdirectory        $cid touch /var/multi-level/subdirectory/testfile
	# And check the same for file volumes.
	run_buildah run -v ${TESTDIR}/was-empty/testfile:/var/different-multi-level/subdirectory/testfile        $cid touch /var/different-multi-level/subdirectory/testfile
	# And check the same for file volumes.
	# Make sure directories show up inside of container on builtin mounts
	run_buildah run -v ${TESTDIR}/was-empty:/run/secrets/testdir $cid ls -ld /run/secrets/testdir
}

@test "run --volume with U flag" {
  skip_if_no_runtime

  # Create source volume.
  mkdir ${TESTDIR}/testdata

  # Create the container.
  _prefetch alpine
  run_buildah from --signature-policy ${TESTSDIR}/policy.json alpine
  ctr="$output"

  # Test user can create file in the mounted volume.
  run_buildah run --user 888:888 --volume ${TESTDIR}/testdata:/mnt:z,U "$ctr" touch /mnt/testfile1.txt

  # Test created file has correct UID and GID ownership.
  run_buildah run --user 888:888 --volume ${TESTDIR}/testdata:/mnt:z,U "$ctr" stat -c "%u:%g" /mnt/testfile1.txt
  expect_output "888:888"
}

@test "run --workingdir" {
	skip_if_no_runtime

	_prefetch alpine
	run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
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
	run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
	cid=$output
	mkdir -p ${TESTDIR}/was:empty
	# As a baseline, this should succeed.
	run_buildah run --mount type=tmpfs,dst=/var/tmpfs-not-empty                                           $cid touch /var/tmpfs-not-empty/testfile
	run_buildah run --mount type=bind,src=${TESTDIR}/was:empty,dst=/var/not-empty${zflag:+,${zflag}}      $cid touch /var/not-empty/testfile
	# If we're parsing the options at all, this should be read-only, so it should fail.
	run_buildah 1 run --mount type=bind,src=${TESTDIR}/was:empty,dst=/var/not-empty,ro${zflag:+,${zflag}} $cid touch /var/not-empty/testfile
	# Even if the parent directory doesn't exist yet, this should succeed.
	run_buildah run --mount type=bind,src=${TESTDIR}/was:empty,dst=/var/multi-level/subdirectory          $cid touch /var/multi-level/subdirectory/testfile
	# And check the same for file volumes.
	run_buildah run --mount type=bind,src=${TESTDIR}/was:empty/testfile,dst=/var/different-multi-level/subdirectory/testfile        $cid touch /var/different-multi-level/subdirectory/testfile
}

@test "run symlinks" {
	skip_if_no_runtime

	${OCI} --version
	_prefetch alpine
	run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
	cid=$output
	mkdir -p ${TESTDIR}/tmp
	ln -s tmp ${TESTDIR}/tmp2
	export TMPDIR=${TESTDIR}/tmp2
	run_buildah run $cid id
}

@test "run --cap-add/--cap-drop" {
	skip_if_no_runtime

	${OCI} --version
	_prefetch alpine
	run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
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
	skip_if_no_runtime

	# we need to not use the list of limits that are set in our default
	# ${TESTSDIR}/containers.conf for the sake of other tests, and override
	# any that might be picked up from system-wide configuration
	echo '[containers]' > ${TESTDIR}/containers.conf
	echo 'default_ulimits = []' >> ${TESTDIR}/containers.conf
	export CONTAINERS_CONF=${TESTDIR}/containers.conf

	_prefetch alpine
	maxpids=$(cat /proc/sys/kernel/pid_max)
	run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
	cid=$output
	run_buildah run $cid awk '/open files/{print $4}' /proc/self/limits
	expect_output 1024 "limits: open files (unlimited)"
	run_buildah run $cid awk '/processes/{print $3}' /proc/self/limits
	expect_output ${maxpids} "limits: processes (unlimited)"
	run_buildah rm $cid

	run_buildah from --quiet --ulimit nofile=300:400 --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
	cid=$output
	run_buildah run $cid awk '/open files/{print $4}' /proc/self/limits
	expect_output "300" "limits: open files (w/file limit)"
	run_buildah run $cid awk '/processes/{print $3}' /proc/self/limits
	expect_output ${maxpids} "limits: processes (w/file limit)"
	run_buildah rm $cid

	run_buildah from --quiet --ulimit nproc=100:200 --ulimit nofile=300:400 --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
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
	run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json quay.io/libpod/registry:volume_omitted
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
	run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
	cid=$output
	run_buildah 42 run ${cid} sh -c 'exit 42'
}

@test "run-exit-status on non executable" {
	skip_if_no_runtime

	_prefetch alpine
	run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
	cid=$output
	run_buildah 1 run ${cid} /etc
}

@test "Verify /run/.containerenv exist" {
	skip_if_no_runtime

	_prefetch alpine
	run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
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
	run_buildah from --quiet --pull=false --device /dev/fuse --signature-policy ${TESTSDIR}/policy.json alpine
	cid=$output
	run_buildah 0 run ${cid} ls /dev/fuse

	run_buildah from --quiet --pull=false --device /dev/fuse:/dev/fuse:rm --signature-policy ${TESTSDIR}/policy.json alpine
	cid=$output
	run_buildah 0 run ${cid} ls /dev/fuse

	run_buildah from --quiet --pull=false --device /dev/fuse:/dev/fuse:rwm --signature-policy ${TESTSDIR}/policy.json alpine
	cid=$output
	run_buildah 0 run ${cid} ls /dev/fuse

}

@test "run-device-Rename" {
	skip_if_no_runtime
	skip_if_chroot
	skip_if_rootless

	_prefetch alpine
	run_buildah from --quiet --pull=false --device /dev/fuse:/dev/fuse1 --signature-policy ${TESTSDIR}/policy.json alpine
	cid=$output
	run_buildah 0 run ${cid} ls /dev/fuse1
}

@test "run check /etc/hosts" {
	skip_if_no_runtime

	${OCI} --version
	_prefetch debian

	run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json debian
	cid=$output
	run_buildah 125 run --isolation=chroot --network=bogus $cid cat /etc/hosts
	expect_output "checking network namespace: stat bogus: no such file or directory"
	run_buildah run --isolation=chroot --network=container $cid cat /etc/hosts
	expect_output --substring "# Generated by Buildah"
	m=$(buildah mount $cid)
	run cat $m/etc/hosts
	[ "$status" -eq 0 ]
	expect_output --substring ""
	run_buildah rm -a

	run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json debian
	cid=$output
	run_buildah run --isolation=chroot --network=host $cid cat /etc/hosts
	expect_output --substring "# Generated by Buildah"
	m=$(buildah mount $cid)
	run cat $m/etc/hosts
	[ "$status" -eq 0 ]
	expect_output --substring ""
	run_buildah rm -a

	run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json debian
	cid=$output
	run_buildah run --isolation=chroot --network=none $cid sh -c 'echo "110.110.110.0 fake_host" >> /etc/hosts; cat /etc/hosts'
	expect_output "110.110.110.0 fake_host"
	m=$(buildah mount $cid)
	run cat $m/etc/hosts
	[ "$status" -eq 0 ]
	expect_output "110.110.110.0 fake_host"
	run_buildah rm -a
}

@test "run check /etc/resolv.conf" {
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
	run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
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
	run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
	cid=$output
	run_buildah run --isolation=chroot --network=host $cid grep nameserver /etc/resolv.conf
	assert "$nameservers" "Container nameservers match the host nameservers"
	if ! is_rootless; then
		run_buildah mount $cid
		assert "$output" != ""
		assert "$(< $output/etc/resolv.conf)" = "" "resolv.conf is empty"
	fi
	run_buildah rm -a

	run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
	cid=$output
	run_buildah run --isolation=chroot --network=none $cid sh -c 'echo "nameserver 110.110.0.110" >> /etc/resolv.conf; cat /etc/resolv.conf'
	expect_output "nameserver 110.110.0.110"
	if ! is_rootless; then
		run_buildah mount $cid
		assert "$output" != ""
		assert "$(< $output/etc/resolv.conf)" =~ "^nameserver 110.110.0.110" "Nameserver is set in the image resolv.conf file"
	fi
	run_buildah rm -a
}

@test "run --user" {
	skip_if_no_runtime

	_prefetch alpine
	run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
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
	cat > ${TESTDIR}/seccomp.json << _EOF
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
	run_buildah from --security-opt seccomp=${TESTDIR}/seccomp.json --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
	cid=$output

	local found_runtime=

	if [ -n "$(command -v runc)" ]; then
		found_runtime=y
		run_buildah ? run --runtime=runc --runtime-flag=debug $cid true
		if [ "$status" -eq 0 ]; then
			[ -n "$output" ]
		else
			# runc fully supports cgroup v2 (unified mode) since v1.0.0-rc93.
			# older runc doesn't work on cgroup v2.
			expect_output --substring "this version of runc doesn't work on cgroups v2" "should fail by unsupportability for cgroupv2"
		fi
	fi

	if [ -n "$(command -v crun)" ]; then
		found_runtime=y
		run_buildah run --runtime=crun --runtime-flag=debug $cid true
		[ -n "$output" ]
	fi

	if [ -z "${found_runtime}" ]; then
		skip "Did not find 'runc' nor 'crun' in \$PATH - could not run this test!"
	fi

}

@test "run --terminal" {
	skip_if_no_runtime

	_prefetch alpine
	run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
	cid=$output
	run_buildah run --terminal=true $cid ls --color=auto
	colored="$output"
	run_buildah run --terminal=false $cid ls --color=auto
	uncolored="$output"
	[ "$colored" != "$uncolored" ]
}
