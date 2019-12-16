#!/usr/bin/env bats

load helpers

@test "run" {
	skip_if_no_runtime

	runc --version
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

	run_buildah unmount $cid
	run_buildah rm $cid
}

@test "run--args" {
	skip_if_no_runtime

	run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
	cid=$output

	# This should fail, because buildah run doesn't have a -n flag.
	run_buildah 1 run -n $cid echo test

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

	run_buildah rm $cid
}

@test "run-cmd" {
	skip_if_no_runtime

	run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
	cid=$output
	run_buildah config --workingdir /tmp $cid


	# Configured entrypoint/cmd shouldn't modify behaviour of run with no arguments

	# empty entrypoint, configured cmd, empty run arguments
	run_buildah config --entrypoint "" $cid
	run_buildah config --cmd pwd $cid
	run_buildah 1 run $cid
	expect_output --substring "command must be specified" "empty entrypoint, cmd, no args"

	# empty entrypoint, configured cmd, empty run arguments, end parsing option
	run_buildah config --entrypoint "" $cid
	run_buildah config --cmd pwd $cid
	run_buildah 1 run $cid --
	expect_output --substring "command must be specified" "empty entrypoint, cmd, no args, --"

	# configured entrypoint, empty cmd, empty run arguments
	run_buildah config --entrypoint pwd $cid
	run_buildah config --cmd "" $cid
	run_buildah 1 run $cid
	expect_output --substring "command must be specified" "entrypoint, empty cmd, no args"

	# configured entrypoint, empty cmd, empty run arguments, end parsing option
	run_buildah config --entrypoint pwd $cid
	run_buildah config --cmd "" $cid
	run_buildah 1 run $cid --
	expect_output --substring "command must be specified" "entrypoint, empty cmd, no args, --"

	# configured entrypoint only, empty run arguments
	run_buildah config --entrypoint pwd $cid
	run_buildah 1 run $cid
	expect_output --substring "command must be specified" "entrypoint, no args"

	# configured entrypoint only, empty run arguments, end parsing option
	run_buildah config --entrypoint pwd $cid
	run_buildah 1 run $cid --
	expect_output --substring "command must be specified" "entrypoint, no args, --"

	# configured cmd only, empty run arguments
	run_buildah config --cmd pwd $cid
	run_buildah 1 run $cid
	expect_output --substring "command must be specified" "cmd, no args"

	# configured cmd only, empty run arguments, end parsing option
	run_buildah config --cmd pwd $cid
	run_buildah 1 run $cid --
	expect_output --substring "command must be specified" "cmd, no args, --"

	# configured entrypoint, configured cmd, empty run arguments
	run_buildah config --entrypoint "pwd" $cid
	run_buildah config --cmd "whoami" $cid
	run_buildah 1 run $cid
	expect_output --substring "command must be specified" "entrypoint, cmd, no args"

	# configured entrypoint, configured cmd, empty run arguments, end parsing option
	run_buildah config --entrypoint "pwd" $cid
	run_buildah config --cmd "whoami" $cid
	run_buildah 1 run $cid --
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

	run_buildah rm $cid
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
	run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
	cid=$output
	run_buildah mount $cid
	root=$output

	testuser=jimbo
	testbogususer=nosuchuser
	testgroup=jimbogroup
	testuid=$RANDOM
	testotheruid=$RANDOM
	testgid=$RANDOM
	testgroupid=$RANDOM
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
        run_buildah 1 run -- $cid id -u
        expect_output --substring "unknown user" "id -u (bogus user)"
        run_buildah 1 run -- $cid id -g
        expect_output --substring "unknown user" "id -g (bogus user)"

	ln -vsf /etc/passwd $root/etc/passwd
	run_buildah config -u ${testuser}:${testgroup} $cid
	run_buildah 1 run -- $cid id -u
	echo "$output"
	expect_output --substring "unknown user" "run as unknown user"

	run_buildah unmount $cid
	run_buildah rm $cid
}

@test "run --hostname" {
	skip_if_rootless
	skip_if_no_runtime

	runc --version
	run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
	cid=$output
	run_buildah run $cid hostname
	[ "$output" != "foobar" ]
	run_buildah run --hostname foobar $cid hostname
	expect_output "foobar"
	run_buildah rm $cid
}

@test "run --volume" {
	skip_if_no_runtime

	zflag=
	if which selinuxenabled > /dev/null 2> /dev/null ; then
		if selinuxenabled ; then
			zflag=z
		fi
	fi
	runc --version
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
}

@test "run --mount" {
	skip_if_no_runtime

	zflag=
	if which selinuxenabled > /dev/null 2> /dev/null ; then
		if selinuxenabled ; then
			zflag=z
		fi
	fi
	runc --version
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

	runc --version
	run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
	cid=$output
	mkdir -p ${TESTDIR}/tmp
	ln -s tmp ${TESTDIR}/tmp2
	export TMPDIR=${TESTDIR}/tmp2
	run_buildah run $cid id
}

@test "run --cap-add/--cap-drop" {
	skip_if_no_runtime

	runc --version
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

	run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
	cid=$output
	run_buildah run $cid awk '/open files/{print $4}' /proc/self/limits
	expect_output "1048576" "limits: open files (unlimited)"
	run_buildah run $cid awk '/processes/{print $3}' /proc/self/limits
	expect_output "1048576" "limits: processes (unlimited)"
	run_buildah rm $cid

	run_buildah from --quiet --ulimit nofile=300:400 --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
	cid=$output
	run_buildah run $cid awk '/open files/{print $4}' /proc/self/limits
	expect_output "300" "limits: open files (w/file limit)"
	run_buildah run $cid awk '/processes/{print $3}' /proc/self/limits
	expect_output "1048576" "limits: processes (w/file limit)"
	run_buildah rm $cid

	run_buildah from --quiet --ulimit nproc=100:200 --ulimit nofile=300:400 --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
	cid=$output
	run_buildah run $cid awk '/open files/{print $4}' /proc/self/limits
	expect_output "300" "limits: open files (w/file & proc limits)"
	run_buildah run $cid awk '/processes/{print $3}' /proc/self/limits
	expect_output "100" "limits: processes (w/file & proc limits)"
	run_buildah rm $cid
}

@test "run-builtin-volume-omitted" {
	# This image is known to include a volume, but not include the mountpoint
	# in the image.
	run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json docker.io/library/registry@sha256:a25e4660ed5226bdb59a5e555083e08ded157b1218282840e55d25add0223390
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

	run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
	cid=$output
	run_buildah 42 run ${cid} sh -c 'exit 42'
}

@test "Verify /run/.containerenv exist" {
	skip_if_no_runtime

	run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
	cid=$output
	# test a standard mount to /run/.containerenv
	run_buildah run $cid ls -1 /run/.containerenv
	expect_output --substring "/run/.containerenv"
}

@test "run-device" {
	skip_if_no_runtime

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

	run_buildah from --quiet --pull=false --device /dev/fuse:/dev/fuse1 --signature-policy ${TESTSDIR}/policy.json alpine
	cid=$output
	run_buildah 0 run ${cid} ls /dev/fuse1
}
