#!/usr/bin/env bats

load helpers

@test "add-local" {
	createrandom ${TMPDIR}/randomfile
	createrandom ${TMPDIR}/other-randomfile

	cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json --image alpine)
	root=$(buildah mount --name=$cid)
	mkdir $root/subdir $root/other-subdir
	# Copy a file to the working directory
	buildah config --workingdir=/ --name=$cid
	buildah add --name=$cid ${TMPDIR}/randomfile
	# Copy a file to a specific subdirectory
	buildah add --name=$cid --dest=/subdir ${TMPDIR}/randomfile
	# Copy a file two files to a specific subdirectory
	buildah add --name=$cid --dest=/other-subdir ${TMPDIR}/randomfile ${TMPDIR}/other-randomfile
	# Copy a file two files to a specific location, created as a subdirectory
	buildah add --name=$cid --dest=/notthereyet-subdir ${TMPDIR}/randomfile ${TMPDIR}/other-randomfile
	# Copy a file to a different working directory
	buildah config --workingdir=/cwd --name=$cid
	buildah add --name=$cid ${TMPDIR}/randomfile
	buildah unmount --name=$cid
	buildah commit --signature-policy ${TESTSDIR}/policy.json --name=$cid --output=containers-storage:new-image
	buildah delete --name=$cid

	newcid=$(buildah from --image new-image)
	newroot=$(buildah mount --name=$newcid)
	test -s $newroot/randomfile
	cmp ${TMPDIR}/randomfile $newroot/randomfile
	test -s $newroot/subdir/randomfile
	cmp ${TMPDIR}/randomfile $newroot/subdir/randomfile
	test -s $newroot/other-subdir/randomfile
	cmp ${TMPDIR}/randomfile $newroot/other-subdir/randomfile
	test -s $newroot/other-subdir/other-randomfile
	cmp ${TMPDIR}/other-randomfile $newroot/other-subdir/other-randomfile
	test -d $newroot/notthereyet-subdir
	test -s $newroot/notthereyet-subdir/randomfile
	cmp ${TMPDIR}/randomfile $newroot/notthereyet-subdir/randomfile
	test -s $newroot/notthereyet-subdir/other-randomfile
	cmp ${TMPDIR}/other-randomfile $newroot/notthereyet-subdir/other-randomfile
	test -d $newroot/cwd
	test -s $newroot/cwd/randomfile
	cmp ${TMPDIR}/randomfile $newroot/cwd/randomfile
	buildah delete --name=$newcid
}
