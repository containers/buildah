#!/usr/bin/env bats

load helpers

@test "blobcache-pull" {
	blobcachedir=${TESTDIR}/cache
	mkdir -p ${blobcachedir}
	# Pull an image using a fresh directory for the blob cache.
	run_buildah pull --blob-cache=${blobcachedir} --signature-policy ${TESTSDIR}/policy.json k8s.gcr.io/pause
	# Check that we dropped some files in there.
	run find ${blobcachedir} -type f
	echo "$output"
	[ "$status" -eq 0 ]
	[ "${#lines[@]}" -gt 0 ]
}

@test "blobcache-from" {
	blobcachedir=${TESTDIR}/cache
	mkdir -p ${blobcachedir}
	# Pull an image using a fresh directory for the blob cache.
	run_buildah from --blob-cache=${blobcachedir} --signature-policy ${TESTSDIR}/policy.json k8s.gcr.io/pause
	# Check that we dropped some files in there.
	run find ${blobcachedir} -type f
	echo "$output"
	[ "$status" -eq 0 ]
	[ "${#lines[*]}" -gt 0 ]
}


function _check_matches() {
	local destdir="$1"
	local blobcachedir="$2"

	# Look for layer blobs in the destination that match the ones in the cache.
	local matched=0
	local unmatched=0
	for content in ${destdir}/* ; do
		match=false
		for blob in ${blobcachedir}/* ; do
			if cmp -s ${content} ${blob} ; then
				echo $(file ${blob}) and ${content} have the same contents, was cached
				match=true
				break
			fi
		done
		if ${match} ; then
			matched=$(( ${matched} + 1 ))
		else
			unmatched=$(( ${unmatched} + 1 ))
			echo ${content} was not cached
		fi
	done

        expect_output --from="$matched"   "$3"  "$4 should match"
        expect_output --from="$unmatched" "$5"  "$6 should not match"
}

@test "blobcache-commit" {
	blobcachedir=${TESTDIR}/cache
	mkdir -p ${blobcachedir}
	# Pull an image using a fresh directory for the blob cache.
	run_buildah from --quiet --blob-cache=${blobcachedir} --signature-policy ${TESTSDIR}/policy.json k8s.gcr.io/pause
	ctr="$output"
	run_buildah add ${ctr} ${TESTSDIR}/bud/add-file/file /
	# Commit the image without using the blob cache, using compression so that uncompressed blobs
	# in the cache which we inherited from our base image won't be matched.
	doomeddir=${TESTDIR}/doomed
	mkdir -p ${doomeddir}
	run_buildah commit --signature-policy ${TESTSDIR}/policy.json --disable-compression=false ${ctr} dir:${doomeddir}
        _check_matches $doomeddir $blobcachedir \
                       0 "nothing" \
                       6 "everything"

	# Commit the image using the blob cache, again using compression.  We'll have recorded the
	# compressed digests that match the uncompressed digests the last time around, so we should
	# get some matches this time.
	destdir=${TESTDIR}/dest
	mkdir -p ${destdir}
	ls -l ${blobcachedir}
	run_buildah commit --signature-policy ${TESTSDIR}/policy.json --blob-cache=${blobcachedir} --disable-compression=false ${ctr} dir:${destdir}
	_check_matches $destdir $blobcachedir \
                       5 "base layers, new layer, config, and manifest" \
                       1 "version"
}

@test "blobcache-push" {
	target=targetimage
	blobcachedir=${TESTDIR}/cache
	mkdir -p ${blobcachedir}
	# Pull an image using a fresh directory for the blob cache.
	run_buildah from --quiet --blob-cache=${blobcachedir} --signature-policy ${TESTSDIR}/policy.json k8s.gcr.io/pause
	ctr="$output"
	run_buildah add ${ctr} ${TESTSDIR}/bud/add-file/file /
	# Commit the image using the blob cache.
	ls -l ${blobcachedir}
	run_buildah commit --signature-policy ${TESTSDIR}/policy.json --blob-cache=${blobcachedir} --disable-compression=false ${ctr} ${target}
	# Try to push the image without the blob cache.
	doomeddir=${TESTDIR}/doomed
	mkdir -p ${doomeddir}
	ls -l ${blobcachedir}
	run_buildah push --signature-policy ${TESTSDIR}/policy.json ${target} dir:${doomeddir}
        _check_matches $doomeddir $blobcachedir \
                       2 "only config and new layer" \
                       4 "version, manifest, base layers"

	# Now try to push the image using the blob cache.
	destdir=${TESTDIR}/dest
	mkdir -p ${destdir}
	ls -l ${blobcachedir}

	run_buildah push --signature-policy ${TESTSDIR}/policy.json --blob-cache=${blobcachedir} ${target} dir:${destdir}
        _check_matches $destdir $blobcachedir \
                       5 "base image layers, new layer, config, and manifest" \
                       1 "version"
}

@test "blobcache-build-compressed-using-dockerfile-explicit-push" {
	blobcachedir=${TESTDIR}/cache
	mkdir -p ${blobcachedir}
	target=new-image
	# Build an image while pulling the base image.  Compress the layers so that they get added
	# to the blob cache in their compressed forms.
	run_buildah build-using-dockerfile -t ${target} --pull-always --signature-policy ${TESTSDIR}/policy.json --blob-cache=${blobcachedir} --disable-compression=false ${TESTSDIR}/bud/add-file
	# Now try to push the image using the blob cache.  The blob cache will only suggest the
	# compressed version of a blob if it's been told that we want to compress things, so
	# we also request compression here to avoid having the copy logic just compress the
	# uncompressed copy again.
	destdir=${TESTDIR}/dest
	mkdir -p ${destdir}
	run_buildah push --signature-policy ${TESTSDIR}/policy.json --blob-cache=${blobcachedir} --disable-compression=false ${target} dir:${destdir}
        _check_matches $destdir $blobcachedir \
                       4 "config, base layer, new layer, and manifest" \
                       1 "version"
}

@test "blobcache-build-uncompressed-using-dockerfile-explicit-push" {
	blobcachedir=${TESTDIR}/cache
	mkdir -p ${blobcachedir}
	target=new-image
	# Build an image while pulling the base image.
	run_buildah build-using-dockerfile -t ${target} -D --pull-always --blob-cache=${blobcachedir} --signature-policy ${TESTSDIR}/policy.json ${TESTSDIR}/bud/add-file
	# Now try to push the image using the blob cache.
	destdir=${TESTDIR}/dest
	mkdir -p ${destdir}
	run_buildah push --signature-policy ${TESTSDIR}/policy.json --blob-cache=${blobcachedir} ${target} dir:${destdir}
	_check_matches $destdir $blobcachedir \
                       2 "config and previously-compressed base layer" \
                       3 "version, new layer, and manifest"
}

@test "blobcache-build-compressed-using-dockerfile-implicit-push" {
	blobcachedir=${TESTDIR}/cache
	mkdir -p ${blobcachedir}
	target=new-image
	destdir=${TESTDIR}/dest
	mkdir -p ${destdir}
	# Build an image while pulling the base image, implicitly pushing while writing.
	run_buildah build-using-dockerfile -t dir:${destdir} --pull-always --blob-cache=${blobcachedir} --signature-policy ${TESTSDIR}/policy.json ${TESTSDIR}/bud/add-file
        _check_matches $destdir $blobcachedir \
                       4 "base image, layer, config, and manifest" \
                       1 "version"
}

@test "blobcache-build-uncompressed-using-dockerfile-implicit-push" {
	blobcachedir=${TESTDIR}/cache
	mkdir -p ${blobcachedir}
	target=new-image
	destdir=${TESTDIR}/dest
	mkdir -p ${destdir}
	# Build an image while pulling the base image, implicitly pushing while writing.
	run_buildah build-using-dockerfile -t dir:${destdir} -D --pull-always --blob-cache=${blobcachedir} --signature-policy ${TESTSDIR}/policy.json ${TESTSDIR}/bud/add-file
        _check_matches $destdir $blobcachedir \
                       4 "base image, our layer, config, and manifest" \
                       1 "version"
}
