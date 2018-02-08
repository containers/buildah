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
	buildah config $cid --workingdir /tmp
	run buildah --debug=false run $cid pwd
	[ "$status" -eq 0 ]
	[ "$output" = /tmp ]
	buildah config $cid --workingdir /root
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
	run buildah --debug=false run $cid echo -n test
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
	buildah config $cid --workingdir /tmp

	buildah config $cid --entrypoint ""
	buildah config $cid --cmd pwd
	run buildah --debug=false run $cid
	[ "$status" -eq 0 ]
	[ "$output" = /tmp ]

	buildah config $cid --entrypoint echo
	run buildah --debug=false run $cid
	[ "$status" -eq 0 ]
	[ "$output" = pwd ]

	buildah config $cid --cmd ""
	run buildah --debug=false run $cid
	[ "$status" -eq 0 ]
	[ "$output" = "" ]

	buildah config $cid --entrypoint ""
	run buildah --debug=false run $cid echo that-other-thing
	[ "$status" -eq 0 ]
	[ "$output" = that-other-thing ]

	buildah config $cid --cmd echo
	run buildah --debug=false run $cid echo that-other-thing
	[ "$status" -eq 0 ]
	[ "$output" = that-other-thing ]

	buildah config $cid --entrypoint echo
	run buildah --debug=false run $cid echo that-other-thing
	[ "$status" -eq 0 ]
	[ "$output" = that-other-thing ]

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

	buildah config $cid -u ""
	buildah run -- $cid id
	run buildah --debug=false run -- $cid id -u
	[ "$status" -eq 0 ]
	[ "$output" = 0 ]
	run buildah --debug=false run -- $cid id -g
	[ "$status" -eq 0 ]
	[ "$output" = 0 ]

	buildah config $cid -u ${testuser}
	buildah run -- $cid id
	run buildah --debug=false run -- $cid id -u
	[ "$status" -eq 0 ]
	[ "$output" = $testuid ]
	run buildah --debug=false run -- $cid id -g
	[ "$status" -eq 0 ]
	[ "$output" = $testgid ]

	buildah config $cid -u ${testuid}
	buildah run -- $cid id
	run buildah --debug=false run -- $cid id -u
	[ "$status" -eq 0 ]
	[ "$output" = $testuid ]
	run buildah --debug=false run -- $cid id -g
	[ "$status" -eq 0 ]
	[ "$output" = $testgid ]

	buildah config $cid -u ${testuser}:${testgroup}
	buildah run -- $cid id
	run buildah --debug=false run -- $cid id -u
	[ "$status" -eq 0 ]
	[ "$output" = $testuid ]
	run buildah --debug=false run -- $cid id -g
	[ "$status" -eq 0 ]
	[ "$output" = $testgroupid ]

	buildah config $cid -u ${testuid}:${testgroup}
	buildah run -- $cid id
	run buildah --debug=false run -- $cid id -u
	[ "$status" -eq 0 ]
	[ "$output" = $testuid ]
	run buildah --debug=false run -- $cid id -g
	[ "$status" -eq 0 ]
	[ "$output" = $testgroupid ]

	buildah config $cid -u ${testotheruid}:${testgroup}
	buildah run -- $cid id
	run buildah --debug=false run -- $cid id -u
	[ "$status" -eq 0 ]
	[ "$output" = $testotheruid ]
	run buildah --debug=false run -- $cid id -g
	[ "$status" -eq 0 ]
	[ "$output" = $testgroupid ]

	buildah config $cid -u ${testotheruid}
	buildah run -- $cid id
	run buildah --debug=false run -- $cid id -u
	[ "$status" -eq 0 ]
	[ "$output" = $testotheruid ]
	run buildah --debug=false run -- $cid id -g
	[ "$status" -eq 0 ]
	[ "$output" = 0 ]

	buildah config $cid -u ${testuser}:${testgroupid}
	buildah run -- $cid id
	run buildah --debug=false run -- $cid id -u
	[ "$status" -eq 0 ]
	[ "$output" = $testuid ]
	run buildah --debug=false run -- $cid id -g
	[ "$status" -eq 0 ]
	[ "$output" = $testgroupid ]

	buildah config $cid -u ${testuid}:${testgroupid}
	buildah run -- $cid id
	run buildah --debug=false run -- $cid id -u
	[ "$status" -eq 0 ]
	[ "$output" = $testuid ]
	run buildah --debug=false run -- $cid id -g
	[ "$status" -eq 0 ]
	[ "$output" = $testgroupid ]

	buildah config $cid -u ${testbogususer}
	run buildah --debug=false run -- $cid id -u
	[ "$status" -ne 0 ]
	[[ "$output" =~ "unknown user" ]]
	run buildah --debug=false run -- $cid id -g
	[ "$status" -ne 0 ]
	[[ "$output" =~ "unknown user" ]]

	ln -vsf /etc/passwd $root/etc/passwd
	buildah config $cid -u ${testuser}:${testgroup}
	run buildah --debug=false run -- $cid id -u
	echo "$output"
	[ "$status" -ne 0 ]
	[[ "$output" =~ "unknown user" ]]

	buildah unmount $cid
	buildah rm $cid
}

@test "run --hostname" {
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

@test "run cpu-period test" {
	cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
    run buildah run --cpu-period=5000 $cid cat /sys/fs/cgroup/cpu/cpu.cfs_period_us
    echo $output
    [ "$status" -eq 0 ]
    [[ "$output" =~ "5000" ]]
	buildah rm $cid
}

@test "run cpu-quota test" {
	cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
    run buildah run --cpu-quota=5000 $cid cat /sys/fs/cgroup/cpu/cpu.cfs_quota_us
    echo "$output"
    [ "$status" -eq 0 ]
    [[ "$output" =~ 5000 ]]
	buildah rm $cid
}

@test "run cpu-shares test" {
	cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
    run buildah run --cpu-shares=2 $cid cat /sys/fs/cgroup/cpu/cpu.shares
    echo "$output"
    [ "$status" -eq 0 ]
    [[ "$output" =~ 2 ]]
	buildah rm $cid
}

@test "run cpuset-cpus test" {
	cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
    run buildah run --cpuset-cpus=0 $cid cat /sys/fs/cgroup/cpuset/cpuset.cpus
    echo "$output"
    [ "$status" -eq 0 ]
    [[ "$output" =~ 0 ]]
	buildah rm $cid
}

@test "run cpuset-mems test" {
	cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
    run buildah run --cpuset-mems=0 $cid cat /sys/fs/cgroup/cpuset/cpuset.mems
    echo "$output"
    [ "$status" -eq 0 ]
    [[ "$output" =~ 0 ]]
	buildah rm $cid
}

@test "run memory test" {
	cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
    run buildah run --memory=40m $cid cat /sys/fs/cgroup/memory/memory.limit_in_bytes
    echo $output
    [ "$status" -eq 0 ]
    [[ "$output" =~ 41943040 ]]
	buildah rm $cid
}

