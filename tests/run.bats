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

	buildah config --entrypoint "" $cid
	buildah config --cmd pwd $cid
	run buildah --debug=false run $cid
	[ "$status" -eq 0 ]
	[ "$output" = /tmp ]

	buildah config --entrypoint echo $cid
	run buildah --debug=false run $cid
	[ "$status" -eq 0 ]
	[ "$output" = pwd ]

	buildah config --cmd "" $cid
	run buildah --debug=false run $cid
	[ "$status" -eq 0 ]
	[ "$output" = "" ]

	buildah config --entrypoint "" $cid
	run buildah --debug=false run $cid echo that-other-thing
	[ "$status" -eq 0 ]
	[ "$output" = that-other-thing ]

	buildah config --cmd echo $cid
	run buildah --debug=false run $cid echo that-other-thing
	[ "$status" -eq 0 ]
	[ "$output" = that-other-thing ]

	buildah config --entrypoint echo $cid
	run buildah --debug=false run $cid that-other-thing
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
