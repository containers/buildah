#!/usr/bin/env bats

load helpers

@test "add-local-plain" {
	createrandom ${TESTDIR}/randomfile
	createrandom ${TESTDIR}/other-randomfile

	cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json --image alpine)
	root=$(buildah mount --name=$cid)
	mkdir $root/subdir $root/other-subdir
	# Copy a file to the working directory
	buildah config --workingdir=/ --name=$cid
	buildah add --name=$cid ${TESTDIR}/randomfile
	# Copy a file to a specific subdirectory
	buildah add --name=$cid --dest=/subdir ${TESTDIR}/randomfile
	# Copy a file two files to a specific subdirectory
	buildah add --name=$cid --dest=/other-subdir ${TESTDIR}/randomfile ${TESTDIR}/other-randomfile
	# Copy a file two files to a specific location, created as a subdirectory
	buildah add --name=$cid --dest=/notthereyet-subdir ${TESTDIR}/randomfile ${TESTDIR}/other-randomfile
	# Copy a file to a different working directory
	buildah config --workingdir=/cwd --name=$cid
	buildah add --name=$cid ${TESTDIR}/randomfile
	buildah unmount --name=$cid
	buildah commit --signature-policy ${TESTSDIR}/policy.json --name=$cid --output=containers-storage:new-image
	buildah delete --name=$cid

	newcid=$(buildah from --image new-image)
	newroot=$(buildah mount --name=$newcid)
	test -s $newroot/randomfile
	cmp ${TESTDIR}/randomfile $newroot/randomfile
	test -s $newroot/subdir/randomfile
	cmp ${TESTDIR}/randomfile $newroot/subdir/randomfile
	test -s $newroot/other-subdir/randomfile
	cmp ${TESTDIR}/randomfile $newroot/other-subdir/randomfile
	test -s $newroot/other-subdir/other-randomfile
	cmp ${TESTDIR}/other-randomfile $newroot/other-subdir/other-randomfile
	test -d $newroot/notthereyet-subdir
	test -s $newroot/notthereyet-subdir/randomfile
	cmp ${TESTDIR}/randomfile $newroot/notthereyet-subdir/randomfile
	test -s $newroot/notthereyet-subdir/other-randomfile
	cmp ${TESTDIR}/other-randomfile $newroot/notthereyet-subdir/other-randomfile
	test -d $newroot/cwd
	test -s $newroot/cwd/randomfile
	cmp ${TESTDIR}/randomfile $newroot/cwd/randomfile
	buildah delete --name=$newcid
}

@test "add-local-archive" {
	createrandom ${TESTDIR}/randomfile
	createrandom ${TESTDIR}/other-randomfile

	cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json --image alpine)
	root=$(buildah mount --name=$cid)
	dd if=/dev/urandom bs=1024 count=4 of=${TESTDIR}/random1
	dd if=/dev/urandom bs=1024 count=4 of=${TESTDIR}/random2
	tar -c -C ${TESTDIR}    -f ${TESTDIR}/tarball1.tar random1 random2
	mkdir ${TESTDIR}/tarball2
	dd if=/dev/urandom bs=1024 count=4 of=${TESTDIR}/tarball2/tarball2.random1
	dd if=/dev/urandom bs=1024 count=4 of=${TESTDIR}/tarball2/tarball2.random2
	tar -c -C ${TESTDIR} -z -f ${TESTDIR}/tarball2.tar.gz  tarball2
	mkdir ${TESTDIR}/tarball3
	dd if=/dev/urandom bs=1024 count=4 of=${TESTDIR}/tarball3/tarball3.random1
	dd if=/dev/urandom bs=1024 count=4 of=${TESTDIR}/tarball3/tarball3.random2
	tar -c -C ${TESTDIR} -j -f ${TESTDIR}/tarball3.tar.bz2 tarball3
	mkdir ${TESTDIR}/tarball4
	dd if=/dev/urandom bs=1024 count=4 of=${TESTDIR}/tarball4/tarball4.random1
	dd if=/dev/urandom bs=1024 count=4 of=${TESTDIR}/tarball4/tarball4.random2
	tar -c -C ${TESTDIR} -j -f ${TESTDIR}/tarball4.tar.bz2 tarball4
	# Add the files to the working directory, which should extract them all.
	buildah config --workingdir=/ --name=$cid
	buildah add --name=$cid ${TESTDIR}/tarball1.tar
	buildah add --name=$cid ${TESTDIR}/tarball2.tar.gz
	buildah add --name=$cid ${TESTDIR}/tarball3.tar.bz2
	buildah add        $cid ${TESTDIR}/tarball4.tar.bz2
	buildah unmount --name=$cid
	buildah commit --signature-policy ${TESTSDIR}/policy.json --name=$cid --output=containers-storage:new-image
	buildah delete --name=$cid

	newcid=$(buildah from --image new-image)
	newroot=$(buildah mount --name=$newcid)
	test -s $newroot/random1
	cmp ${TESTDIR}/random1 $newroot/random1
	test -s $newroot/random2
	cmp ${TESTDIR}/random2 $newroot/random2
	test -s $newroot/tarball2/tarball2.random1
	cmp ${TESTDIR}/tarball2/tarball2.random1 $newroot/tarball2/tarball2.random1
	test -s $newroot/tarball2/tarball2.random2
	cmp ${TESTDIR}/tarball2/tarball2.random2 $newroot/tarball2/tarball2.random2
	test -s $newroot/tarball3/tarball3.random1
	cmp ${TESTDIR}/tarball3/tarball3.random1 $newroot/tarball3/tarball3.random1
	test -s $newroot/tarball3/tarball3.random2
	cmp ${TESTDIR}/tarball3/tarball3.random2 $newroot/tarball3/tarball3.random2
	test -s $newroot/tarball4/tarball4.random1
	cmp ${TESTDIR}/tarball4/tarball4.random1 $newroot/tarball4/tarball4.random1
	test -s $newroot/tarball4/tarball4.random2
	cmp ${TESTDIR}/tarball4/tarball4.random2 $newroot/tarball4/tarball4.random2
	buildah delete --name=$newcid
}
