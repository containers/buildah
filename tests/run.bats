#!/usr/bin/env bats

load helpers

@test "run" {
	if ! which runc ; then
		skip
	fi
	runc --version
	createrandom ${TESTDIR}/randomfile
	cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
	root=$(buildah mount $cid)
	buildah config --workingdir /tmp $cid
	run buildah --debug=false run $cid pwd
	[ "$status" -eq 0 ]
	[ "$output" = /tmp ]
	buildah config --workingdir /root $cid
	run buildah --debug=false run        $cid pwd
	[ "$status" -eq 0 ]
	[ "$output" = /root ]
	cp ${TESTDIR}/randomfile $root/tmp/
	buildah run        $cid cp /tmp/randomfile /tmp/other-randomfile
	test -s $root/tmp/other-randomfile
	cmp ${TESTDIR}/randomfile $root/tmp/other-randomfile

	buildah unmount $cid
	buildah rm $cid
}

@test "run--args" {
	if ! which runc ; then
		skip
	fi
	cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)

	# This should fail, because buildah run doesn't have a -n flag.
	run buildah --debug=false run -n $cid echo test
	[ "$status" -ne 0 ]

	# This should succeed, because buildah run stops caring at the --, which is preserved as part of the command.
	run buildah --debug=false run $cid echo -- -n test
	[ "$status" -eq 0 ]
	echo :"$output":
	[ "$output" = "-- -n test" ]

	# This should succeed, because buildah run stops caring at the --, which is not part of the command.
	run buildah --debug=false run $cid -- echo -n -- test
	[ "$status" -eq 0 ]
	echo :"$output":
	[ "$output" = "-- test" ]

	# This should succeed, because buildah run stops caring at the --.
	run buildah --debug=false run $cid -- echo -- -n test --
	[ "$status" -eq 0 ]
	echo :"$output":
	[ "$output" = "-- -n test --" ]

	# This should succeed, because buildah run stops caring at the --.
	run buildah --debug=false run $cid -- echo -n "test"
	[ "$status" -eq 0 ]
	echo :"$output":
	[ "$output" = "test" ]

	buildah rm $cid
}

@test "run-cmd" {
	if ! which runc ; then
		skip
	fi
	cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
	buildah config --workingdir /tmp $cid


	# Configured entrypoint/cmd shouldn't modify behaviour of run with no arguments

	# empty entrypoint, configured cmd, empty run arguments
	buildah config --entrypoint "" $cid
	buildah config --cmd pwd $cid
	run buildah --debug=false run $cid
	[ "$status" -eq 1 ]
	[ "$output" = "command must be specified" ]
	
	# empty entrypoint, configured cmd, empty run arguments, end parsing option
	buildah config --entrypoint "" $cid
	buildah config --cmd pwd $cid
	run buildah --debug=false run $cid --
	[ "$status" -eq 1 ]
	[ "$output" = "command must be specified" ]

	# configured entrypoint, empty cmd, empty run arguments
	buildah config --entrypoint pwd $cid
	buildah config --cmd "" $cid
	run buildah --debug=false run $cid
	[ "$status" -eq 1 ]
	[ "$output" = "command must be specified" ]
	
	# configured entrypoint, empty cmd, empty run arguments, end parsing option
	buildah config --entrypoint pwd $cid
	buildah config --cmd "" $cid
	run buildah --debug=false run $cid --
	[ "$status" -eq 1 ]
	[ "$output" = "command must be specified" ]

	# configured entrypoint only, empty run arguments
	buildah config --entrypoint pwd $cid
	run buildah --debug=false run $cid
	[ "$status" -eq 1 ]
	[ "$output" = "command must be specified" ]
	
	# configured entrypoint only, empty run arguments, end parsing option
	buildah config --entrypoint pwd $cid
	run buildah --debug=false run $cid --
	[ "$status" -eq 1 ]
	[ "$output" = "command must be specified" ]

	# cofigured cmd only, empty run arguments
	buildah config --cmd pwd $cid
	run buildah --debug=false run $cid
	[ "$status" -eq 1 ]
	[ "$output" = "command must be specified" ]

	# cofigured cmd only, empty run arguments, end parsing option
	buildah config --cmd pwd $cid
	run buildah --debug=false run $cid --
	[ "$status" -eq 1 ]
	[ "$output" = "command must be specified" ]

	# configured entrypoint, configured cmd, empty run arguments
	buildah config --entrypoint "pwd" $cid
	buildah config --cmd "whoami" $cid
	run buildah --debug=false run $cid
	[ "$status" -eq 1 ]
	[ "$output" = "command must be specified" ]
	
	# configured entrypoint, configured cmd, empty run arguments, end parsing option
	buildah config --entrypoint "pwd" $cid
	buildah config --cmd "whoami" $cid
	run buildah --debug=false run $cid --
	[ "$status" -eq 1 ]
	[ "$output" = "command must be specified" ]


	# Configured entrypoint/cmd shouldn't modify behaviour of run with argument
	# Note: entrypoint and cmd can be invalid in below tests as they should never execute

	# empty entrypoint, configured cmd, configured run arguments
	buildah config --entrypoint "" $cid
	buildah config --cmd "/invalid/cmd" $cid
	run buildah --debug=false run $cid -- pwd
	[ "$status" -eq 0 ]
	[ "$output" = "/tmp" ]

        # configured entrypoint, empty cmd, configured run arguments
        buildah config --entrypoint "/invalid/entrypoint" $cid
        buildah config --cmd "" $cid
        run buildah --debug=false run $cid -- pwd
        [ "$status" -eq 0 ]
        [ "$output" = "/tmp" ]

        # configured entrypoint only, configured run arguments
        buildah config --entrypoint "/invalid/entrypoint" $cid
        run buildah --debug=false run $cid -- pwd
        [ "$status" -eq 0 ]
        [ "$output" = "/tmp" ]

        # cofigured cmd only, configured run arguments
        buildah config --cmd "/invalid/cmd" $cid
        run buildah --debug=false run $cid -- pwd
        [ "$status" -eq 0 ]
        [ "$output" = "/tmp" ]

        # configured entrypoint, configured cmd, configured run arguments
        buildah config --entrypoint "/invalid/entrypoint" $cid
        buildah config --cmd "/invalid/cmd" $cid
        run buildah --debug=false run $cid -- pwd
        [ "$status" -eq 0 ]
        [ "$output" = "/tmp" ]

	buildah rm $cid
}

@test "run-user" {
	if ! which runc ; then
		skip
	fi
	eval $(go env)
	echo CGO_ENABLED=${CGO_ENABLED}
	if test "$CGO_ENABLED" -ne 1; then
		skip
	fi
	cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
	root=$(buildah mount $cid)

	testuser=jimbo
	testbogususer=nosuchuser
	testgroup=jimbogroup
	testuid=$RANDOM
	testotheruid=$RANDOM
	testgid=$RANDOM
	testgroupid=$RANDOM
	echo "$testuser:x:$testuid:$testgid:Jimbo Jenkins:/home/$testuser:/bin/sh" >> $root/etc/passwd
	echo "$testgroup:x:$testgroupid:" >> $root/etc/group

	buildah config -u "" $cid
	buildah run -- $cid id
	run buildah --debug=false run -- $cid id -u
	[ "$status" -eq 0 ]
	[ "$output" = 0 ]
	run buildah --debug=false run -- $cid id -g
	[ "$status" -eq 0 ]
	[ "$output" = 0 ]

	buildah config -u ${testuser} $cid
	buildah run -- $cid id
	run buildah --debug=false run -- $cid id -u
	[ "$status" -eq 0 ]
	[ "$output" = $testuid ]
	run buildah --debug=false run -- $cid id -g
	[ "$status" -eq 0 ]
	[ "$output" = $testgid ]

	buildah config -u ${testuid} $cid
	buildah run -- $cid id
	run buildah --debug=false run -- $cid id -u
	[ "$status" -eq 0 ]
	[ "$output" = $testuid ]
	run buildah --debug=false run -- $cid id -g
	[ "$status" -eq 0 ]
	[ "$output" = $testgid ]

	buildah config -u ${testuser}:${testgroup} $cid
	buildah run -- $cid id
	run buildah --debug=false run -- $cid id -u
	[ "$status" -eq 0 ]
	[ "$output" = $testuid ]
	run buildah --debug=false run -- $cid id -g
	[ "$status" -eq 0 ]
	[ "$output" = $testgroupid ]

	buildah config -u ${testuid}:${testgroup} $cid
	buildah run -- $cid id
	run buildah --debug=false run -- $cid id -u
	[ "$status" -eq 0 ]
	[ "$output" = $testuid ]
	run buildah --debug=false run -- $cid id -g
	[ "$status" -eq 0 ]
	[ "$output" = $testgroupid ]

	buildah config -u ${testotheruid}:${testgroup} $cid
	buildah run -- $cid id
	run buildah --debug=false run -- $cid id -u
	[ "$status" -eq 0 ]
	[ "$output" = $testotheruid ]
	run buildah --debug=false run -- $cid id -g
	[ "$status" -eq 0 ]
	[ "$output" = $testgroupid ]

	buildah config -u ${testotheruid} $cid
	buildah run -- $cid id
	run buildah --debug=false run -- $cid id -u
	[ "$status" -eq 0 ]
	[ "$output" = $testotheruid ]
	run buildah --debug=false run -- $cid id -g
	[ "$status" -eq 0 ]
	[ "$output" = 0 ]

	buildah config -u ${testuser}:${testgroupid} $cid
	buildah run -- $cid id
	run buildah --debug=false run -- $cid id -u
	[ "$status" -eq 0 ]
	[ "$output" = $testuid ]
	run buildah --debug=false run -- $cid id -g
	[ "$status" -eq 0 ]
	[ "$output" = $testgroupid ]

	buildah config -u ${testuid}:${testgroupid} $cid
	buildah run -- $cid id
	run buildah --debug=false run -- $cid id -u
	[ "$status" -eq 0 ]
	[ "$output" = $testuid ]
	run buildah --debug=false run -- $cid id -g
	[ "$status" -eq 0 ]
	[ "$output" = $testgroupid ]

	buildah config -u ${testbogususer} $cid
	run buildah --debug=false run -- $cid id -u
	[ "$status" -ne 0 ]
	[[ "$output" =~ "unknown user" ]]
	run buildah --debug=false run -- $cid id -g
	[ "$status" -ne 0 ]
	[[ "$output" =~ "unknown user" ]]

	ln -vsf /etc/passwd $root/etc/passwd
	buildah config -u ${testuser}:${testgroup} $cid
	run buildah --debug=false run -- $cid id -u
	echo "$output"
	[ "$status" -ne 0 ]
	[[ "$output" =~ "unknown user" ]]

	buildah unmount $cid
	buildah rm $cid
}

@test "run --hostname" {
	if test "$BUILDAH_ISOLATION" = "rootless" ; then
		skip
	fi
	if ! which runc ; then
		skip
	fi
	runc --version
	cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
	run buildah --debug=false run $cid hostname
	echo "$output"
	[ "$status" -eq 0 ]
	[ "$output" != "foobar" ]
	run buildah --debug=false run --hostname foobar $cid hostname
	echo "$output"
	[ "$status" -eq 0 ]
	[ "$output" = "foobar" ]
	buildah rm $cid
}

@test "run --volume" {
	if ! which runc ; then
		skip
	fi
	zflag=
	if which selinuxenabled > /dev/null 2> /dev/null ; then
		if selinuxenabled ; then
			zflag=z
		fi
	fi
	runc --version
	cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
	mkdir -p ${TESTDIR}/was-empty
	# As a baseline, this should succeed.
	run buildah --debug=false run -v ${TESTDIR}/was-empty:/var/not-empty${zflag:+:${zflag}}     $cid touch /var/not-empty/testfile
	echo "$output"
	[ "$status" -eq 0 ]
	# If we're parsing the options at all, this should be read-only, so it should fail.
	run buildah --debug=false run -v ${TESTDIR}/was-empty:/var/not-empty:ro${zflag:+,${zflag}} $cid touch /var/not-empty/testfile
	echo "$output"
	[ "$status" -ne 0 ]
}

@test "run symlinks" {
	if ! which runc ; then
		skip
	fi
	runc --version
	cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
	mkdir -p ${TESTDIR}/tmp
	ln -s tmp ${TESTDIR}/tmp2
	export TMPDIR=${TESTDIR}/tmp2
	run buildah --debug=false run $cid id
	echo "$output"
	[ "$status" -eq 0 ]
}

@test "run --cap-add/--cap-drop" {
	if ! which runc ; then
		skip
	fi
	runc --version
	cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
	# Try with default caps.
	run buildah --debug=false run $cid grep ^CapEff /proc/self/status
	echo "$output"
	[ "$status" -eq 0 ]
	defaultcaps="$output"
	# Try adding DAC_OVERRIDE.
	run buildah --debug=false run --cap-add CAP_DAC_OVERRIDE $cid grep ^CapEff /proc/self/status
	echo "$output"
	[ "$status" -eq 0 ]
	addedcaps="$output"
	# Try dropping DAC_OVERRIDE.
	run buildah --debug=false run --cap-drop CAP_DAC_OVERRIDE $cid grep ^CapEff /proc/self/status
	echo "$output"
	[ "$status" -eq 0 ]
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
	if ! which runc ; then
		skip
	fi
	cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
	run buildah --debug=false run $cid awk '/open files/{print $4}' /proc/self/limits
	[ "$status" -eq 0 ]
	[ "$output" = 1048576 ]
	echo $output
	run buildah --debug=false run $cid awk '/processes/{print $3}' /proc/self/limits
	[ "$status" -eq 0 ]
	[ "$output" = 1048576 ]
	echo $output
	buildah rm $cid

	cid=$(buildah from --ulimit nofile=300:400 --pull --signature-policy ${TESTSDIR}/policy.json alpine)
	run buildah --debug=false run $cid awk '/open files/{print $4}' /proc/self/limits
	[ "$status" -eq 0 ]
	[ "$output" = 300 ]
	echo $output
	run buildah --debug=false run $cid awk '/processes/{print $3}' /proc/self/limits
	echo $output
	[ "$status" -eq 0 ]
	[ "$output" = 1048576 ]
	buildah rm $cid

	cid=$(buildah from --ulimit nproc=100:200 --ulimit nofile=300:400 --pull --signature-policy ${TESTSDIR}/policy.json alpine)
	run buildah --debug=false run $cid awk '/open files/{print $4}' /proc/self/limits
	[ "$status" -eq 0 ]
	[ "$output" = 300 ]
	echo $output
	run buildah --debug=false run $cid awk '/processes/{print $3}' /proc/self/limits
	[ "$status" -eq 0 ]
	[ "$output" = 100 ]
	echo $output
	buildah rm $cid
}
