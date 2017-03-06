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

@test "commit" {
	createrandom ${TMPDIR}/randomfile
	createrandom ${TMPDIR}/other-randomfile

	cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json --image alpine)
	root=$(buildah mount --name=$cid)
	cp ${TMPDIR}/randomfile $root/randomfile
	buildah unmount --name=$cid
	buildah commit --signature-policy ${TESTSDIR}/policy.json --name=$cid --output=containers-storage:new-image
	buildah delete --name=$cid

	newcid=$(buildah from --image new-image)
	newroot=$(buildah mount --name=$newcid)
	test -s $newroot/randomfile
	cmp ${TMPDIR}/randomfile $newroot/randomfile
	cp ${TMPDIR}/other-randomfile $newroot/other-randomfile
	buildah commit --signature-policy ${TESTSDIR}/policy.json --name=$newcid --output=containers-storage:other-new-image
	buildah unmount --name=$newcid
	buildah delete --name=$newcid

	othernewcid=$(buildah from --image other-new-image)
	othernewroot=$(buildah mount --name=$othernewcid)
	test -s $othernewroot/randomfile
	cmp ${TMPDIR}/randomfile $othernewroot/randomfile
	test -s $othernewroot/other-randomfile
	cmp ${TMPDIR}/other-randomfile $othernewroot/other-randomfile
	buildah delete --name=$othernewcid
}
