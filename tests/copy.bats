#!/usr/bin/env bats

load helpers

@test "copy-local-plain" {
	createrandom ${TESTDIR}/randomfile
	createrandom ${TESTDIR}/other-randomfile

	cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json --image alpine)
	root=$(buildah mount --name=$cid)
	buildah config --name=$cid --workingdir /
	buildah copy --name=$cid ${TESTDIR}/randomfile
	buildah copy        $cid ${TESTDIR}/other-randomfile
	buildah unmount --name=$cid
	buildah commit --signature-policy ${TESTSDIR}/policy.json --name=$cid --output=containers-storage:new-image
	buildah delete --name=$cid

	newcid=$(buildah from --image new-image)
	newroot=$(buildah mount --name=$newcid)
	test -s $newroot/randomfile
	cmp ${TESTDIR}/randomfile $newroot/randomfile
	test -s $newroot/other-randomfile
	cmp ${TESTDIR}/other-randomfile $newroot/other-randomfile
	buildah delete --name=$newcid
}
