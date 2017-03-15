#!/usr/bin/env bats

load helpers

@test "from" {
	cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json --image alpine)
	buildah delete --name=$cid
}

@test "mount" {
	cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json --image alpine)
	root=$(buildah mount --name=$cid)
	buildah unmount --name=$cid
	buildah delete --name=$cid
}

@test "by-name" {
	cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json --image alpine --name=alpine-working-image-for-test)
	root=$(buildah mount --name=alpine-working-image-for-test)
	buildah unmount --name=alpine-working-image-for-test
	buildah delete --name=alpine-working-image-for-test
}

@test "by-root" {
	cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json --image alpine)
	root=$(buildah mount --name=$cid)
	buildah unmount --root=$root
	buildah delete --root=$root
}

@test "commit" {
	createrandom ${TESTDIR}/randomfile
	createrandom ${TESTDIR}/other-randomfile

	cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json --image alpine)
	root=$(buildah mount --name=$cid)
	cp ${TESTDIR}/randomfile $root/randomfile
	buildah unmount --name=$cid
	buildah commit --signature-policy ${TESTSDIR}/policy.json --name=$cid --output=containers-storage:new-image
	buildah delete --name=$cid

	newcid=$(buildah from --image new-image)
	newroot=$(buildah mount --name=$newcid)
	test -s $newroot/randomfile
	cmp ${TESTDIR}/randomfile $newroot/randomfile
	cp ${TESTDIR}/other-randomfile $newroot/other-randomfile
	buildah commit --signature-policy ${TESTSDIR}/policy.json --name=$newcid --output=containers-storage:other-new-image
	buildah unmount --name=$newcid
	buildah delete --name=$newcid

	othernewcid=$(buildah from --image other-new-image)
	othernewroot=$(buildah mount --name=$othernewcid)
	test -s $othernewroot/randomfile
	cmp ${TESTDIR}/randomfile $othernewroot/randomfile
	test -s $othernewroot/other-randomfile
	cmp ${TESTDIR}/other-randomfile $othernewroot/other-randomfile
	buildah delete --name=$othernewcid
}
