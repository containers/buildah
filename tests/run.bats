#!/usr/bin/env bats

load helpers

@test "run" {
	if ! which runc ; then
		skip
	fi
	createrandom ${TESTDIR}/randomfile
	cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
	root=$(buildah mount $cid)
	buildah config $cid --workingdir /tmp
	run buildah --debug=false run $cid pwd
	[ "$output" = /tmp ]
	buildah config $cid --workingdir /root
	run buildah --debug=false run        $cid pwd
	[ "$output" = /root ]
	cp ${TESTDIR}/randomfile $root/tmp/
	buildah run        $cid cp /tmp/randomfile /tmp/other-randomfile
	test -s $root/tmp/other-randomfile
	cmp ${TESTDIR}/randomfile $root/tmp/other-randomfile
	run buildah run $cid echo -n test
	[ $status != 0 ]
	run buildah run $cid echo -- -n test
	[ $status != 0 ]
	run buildah run $cid -- echo -n -- test
	[ "$output" = "-- test" ]
	run buildah run $cid -- echo -- -n test --
	[ "$output" = "-- -n -- test --" ]
	run buildah run $cid -- echo -n "test"
	[ "$output" = "test" ]

	buildah unmount $cid
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
	[ "$output" = /tmp ]

	buildah config $cid --entrypoint echo
	run buildah --debug=false run $cid
	[ "$output" = pwd ]

	buildah config $cid --cmd ""
	run buildah --debug=false run $cid
	[ "$output" = "" ]

	buildah config $cid --entrypoint ""
	run buildah --debug=false run $cid echo that-other-thing
	[ "$output" = that-other-thing ]

	buildah config $cid --cmd echo
	run buildah --debug=false run $cid echo that-other-thing
	[ "$output" = that-other-thing ]

	buildah config $cid --entrypoint echo
	run buildah --debug=false run $cid echo that-other-thing
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
	testgroup=jimbogroup
	testuid=$RANDOM
	testgid=$RANDOM
	testgroupid=$RANDOM
	echo "$testuser:x:$testuid:$testgid:Jimbo Jenkins:/home/$testuser:/bin/sh" >> $root/etc/passwd
	echo "$testgroup:x:$testgroupid:" >> $root/etc/group

	buildah config $cid -u ""
	buildah run -- $cid id
	run buildah --debug=false run -- $cid id -u
	[ "$output" = 0 ]
	run buildah --debug=false run -- $cid id -g
	[ "$output" = 0 ]

	buildah config $cid -u ${testuser}
	buildah run -- $cid id
	run buildah --debug=false run -- $cid id -u
	[ "$output" = $testuid ]
	run buildah --debug=false run -- $cid id -g
	[ "$output" = $testgid ]

	buildah config $cid -u ${testuid}
	buildah run -- $cid id
	run buildah --debug=false run -- $cid id -u
	[ "$output" = $testuid ]
	run buildah --debug=false run -- $cid id -g
	[ "$output" = $testgid ]

	buildah config $cid -u ${testuser}:${testgroup}
	buildah run -- $cid id
	run buildah --debug=false run -- $cid id -u
	[ "$output" = $testuid ]
	run buildah --debug=false run -- $cid id -g
	[ "$output" = $testgroupid ]

	buildah config $cid -u ${testuid}:${testgroup}
	buildah run -- $cid id
	run buildah --debug=false run -- $cid id -u
	[ "$output" = $testuid ]
	run buildah --debug=false run -- $cid id -g
	[ "$output" = $testgroupid ]

	buildah config $cid -u ${testuser}:${testgroupid}
	buildah run -- $cid id
	run buildah --debug=false run -- $cid id -u
	[ "$output" = $testuid ]
	run buildah --debug=false run -- $cid id -g
	[ "$output" = $testgroupid ]

	buildah config $cid -u ${testuid}:${testgroupid}
	buildah run -- $cid id
	run buildah --debug=false run -- $cid id -u
	[ "$output" = $testuid ]
	run buildah --debug=false run -- $cid id -g
	[ "$output" = $testgroupid ]

	buildah unmount $cid
	buildah rm $cid
}
