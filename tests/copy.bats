#!/usr/bin/env bats

load helpers

@test "copy-local-plain" {
	createrandom ${TMPDIR}/randomfile
	createrandom ${TMPDIR}/other-randomfile

	cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json --image alpine)
	root=$(buildah mount --name=$cid)
	buildah config --name=$cid --workingdir /
	buildah copy --name=$cid ${TMPDIR}/randomfile
	buildah unmount --name=$cid
	buildah commit --signature-policy ${TESTSDIR}/policy.json --name=$cid --output=containers-storage:new-image
	buildah delete --name=$cid

	newcid=$(buildah from --image new-image)
	newroot=$(buildah mount --name=$newcid)
	test -s $newroot/randomfile
	cmp ${TMPDIR}/randomfile $newroot/randomfile
	buildah delete --name=$newcid
}
