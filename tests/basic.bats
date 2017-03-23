#!/usr/bin/env bats

load helpers

@test "from" {
	cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json --image alpine)
	buildah delete $cid
	cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json         alpine)
	buildah delete $cid
	cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json         alpine  i-love-naming-things)
	buildah delete i-love-naming-things
}

@test "from-defaultpull" {
	cid=$(buildah from --signature-policy ${TESTSDIR}/policy.json --image alpine)
	buildah delete        $cid
}

@test "from-nopull" {
	run buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json --image alpine
	[ "$status" -eq 1 ]
}

@test "mount" {
	cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json --image alpine)
	root=$(buildah mount --name=$cid)
	buildah unmount --name=$cid
	root=$(buildah mount        $cid)
	buildah unmount        $cid
	buildah delete $cid
}

@test "by-name" {
	cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json --image alpine --name=alpine-working-image-for-test)
	root=$(buildah mount --name=alpine-working-image-for-test)
	buildah unmount --name=alpine-working-image-for-test
	buildah delete alpine-working-image-for-test
}

@test "by-root" {
	cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json --image alpine)
	root=$(buildah mount --name=$cid)
	buildah unmount --root=$root
	buildah delete --root=$root $cid
}

@test "commit" {
	createrandom ${TESTDIR}/randomfile
	createrandom ${TESTDIR}/other-randomfile

	cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json --image alpine)
	root=$(buildah mount --name=$cid)
	cp ${TESTDIR}/randomfile $root/randomfile
	buildah unmount --name=$cid
	buildah commit --signature-policy ${TESTSDIR}/policy.json --name=$cid --output=containers-storage:new-image
	buildah delete $cid

	newcid=$(buildah from --image new-image)
	newroot=$(buildah mount --name=$newcid)
	test -s $newroot/randomfile
	cmp ${TESTDIR}/randomfile $newroot/randomfile
	cp ${TESTDIR}/other-randomfile $newroot/other-randomfile
	buildah commit --signature-policy ${TESTSDIR}/policy.json --name=$newcid --output=containers-storage:other-new-image
	# Not an allowed ordering of arguments and flags.  Check that it's rejected.
	run buildah commit --signature-policy ${TESTSDIR}/policy.json        $newcid --output=containers-storage:rejected-new-image
	[ "$status" -eq 1 ]
	buildah commit --signature-policy ${TESTSDIR}/policy.json                --output=containers-storage:another-new-image      $newcid
	buildah commit --signature-policy ${TESTSDIR}/policy.json        $newcid          containers-storage:yet-another-new-image
	buildah unmount --name=$newcid
	buildah delete $newcid

	othernewcid=$(buildah from --image other-new-image)
	othernewroot=$(buildah mount --name=$othernewcid)
	test -s $othernewroot/randomfile
	cmp ${TESTDIR}/randomfile $othernewroot/randomfile
	test -s $othernewroot/other-randomfile
	cmp ${TESTDIR}/other-randomfile $othernewroot/other-randomfile
	buildah delete $othernewcid

	anothernewcid=$(buildah from --image another-new-image)
	anothernewroot=$(buildah mount --name=$anothernewcid)
	test -s $anothernewroot/randomfile
	cmp ${TESTDIR}/randomfile $anothernewroot/randomfile
	test -s $anothernewroot/other-randomfile
	cmp ${TESTDIR}/other-randomfile $anothernewroot/other-randomfile
	buildah delete $anothernewcid

	yetanothernewcid=$(buildah from --image yet-another-new-image)
	yetanothernewroot=$(buildah mount --name=$yetanothernewcid)
	test -s $yetanothernewroot/randomfile
	cmp ${TESTDIR}/randomfile $yetanothernewroot/randomfile
	test -s $yetanothernewroot/other-randomfile
	cmp ${TESTDIR}/other-randomfile $yetanothernewroot/other-randomfile
	buildah delete $yetanothernewcid
}
