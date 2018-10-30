#!/usr/bin/env bats

load helpers

@test "blobcache-pull" {
	blobcachedir=${TESTDIR}/cache
	mkdir -p ${blobcachedir}
	# Pull an image using a fresh directory for the blob cache.
	run buildah pull --blob-cache=${blobcachedir} --signature-policy ${TESTSDIR}/policy.json docker.io/kubernetes/pause
	echo "$output"
	[ "$status" -eq 0 ]
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
	run buildah from --blob-cache=${blobcachedir} --signature-policy ${TESTSDIR}/policy.json docker.io/kubernetes/pause
	echo "$output"
	[ "$status" -eq 0 ]
	# Check that we dropped some files in there.
	run find ${blobcachedir} -type f
	echo "$output"
	[ "$status" -eq 0 ]
	[ "${#lines[*]}" -gt 0 ]
}

@test "blobcache-commit" {
	blobcachedir=${TESTDIR}/cache
	mkdir -p ${blobcachedir}
	# Pull an image using a fresh directory for the blob cache.
	run buildah --debug=false from --quiet --blob-cache=${blobcachedir} --signature-policy ${TESTSDIR}/policy.json docker.io/kubernetes/pause
	echo "$output"
	ctr="$output"
	[ "$status" -eq 0 ]
	run buildah add ${ctr} ${TESTSDIR}/bud/add-file/file /
	echo "$output"
	[ "$status" -eq 0 ]
	# Commit the image without using the blob cache.
	doomeddir=${TESTDIR}/doomed
	mkdir -p ${doomeddir}
	ls -l ${blobcachedir}
	echo buildah commit --signature-policy ${TESTSDIR}/policy.json ${ctr} dir:${doomeddir}
	run buildah commit --signature-policy ${TESTSDIR}/policy.json ${ctr} dir:${doomeddir}
	echo "$output"
	[ "$status" -eq 0 ]
	# Look for layer blobs in the destination that match the ones in the cache.
	matched=0
	unmatched=0
	for content in ${doomeddir}/* ; do
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
	[ ${matched} -eq 0 ] # nothing should match items in the cache
	[ ${unmatched} -eq 6 ] # nothing should match items in the cache
	# Commit the image using the blob cache.
	destdir=${TESTDIR}/dest
	mkdir -p ${destdir}
	ls -l ${blobcachedir}
	echo buildah commit --signature-policy ${TESTSDIR}/policy.json --blob-cache=${blobcachedir} ${ctr} dir:${destdir}
	run buildah commit --signature-policy ${TESTSDIR}/policy.json --blob-cache=${blobcachedir} ${ctr} dir:${destdir}
	echo "$output"
	[ "$status" -eq 0 ]
	# Look for layer blobs in the destination that match the ones in the cache.
	matched=0
	unmatched=0
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
	[ ${matched} -eq 5 ] # the base layers, our new layer, our config, and manifest should match items in the cache
	[ ${unmatched} -eq 1 ] # the version shouldn't match an item in the cache
}

@test "blobcache-push" {
	target=targetimage
	blobcachedir=${TESTDIR}/cache
	mkdir -p ${blobcachedir}
	# Pull an image using a fresh directory for the blob cache.
	run buildah --debug=false from --quiet --blob-cache=${blobcachedir} --signature-policy ${TESTSDIR}/policy.json docker.io/kubernetes/pause
	echo "$output"
	ctr="$output"
	[ "$status" -eq 0 ]
	run buildah add ${ctr} ${TESTSDIR}/bud/add-file/file /
	echo "$output"
	[ "$status" -eq 0 ]
	# Commit the image using the blob cache.
	ls -l ${blobcachedir}
	echo buildah commit --signature-policy ${TESTSDIR}/policy.json --blob-cache=${blobcachedir} ${ctr} ${target}
	run buildah commit --signature-policy ${TESTSDIR}/policy.json --blob-cache=${blobcachedir} ${ctr} ${target}
	echo "$output"
	[ "$status" -eq 0 ]
	# Try to push the image without the blob cache.
	doomeddir=${TESTDIR}/doomed
	mkdir -p ${doomeddir}
	ls -l ${blobcachedir}
	echo buildah push --signature-policy ${TESTSDIR}/policy.json ${target} dir:${doomeddir}
	run buildah push --signature-policy ${TESTSDIR}/policy.json ${target} dir:${doomeddir}
	echo "$output"
	[ "$status" -eq 0 ]
	# Look for layer blobs in the doomed copy that match the ones in the cache.
	matched=0
	unmatched=0
	for content in ${doomeddir}/* ; do
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
	[ ${matched} -eq 2 ] # basically, only the config and our new layer should be the same as anything in the cache
	[ ${unmatched} -eq 4 ] # none of the version, manifest, and base layers should match items in the cache, since the base layers were recompressed
	# Now try to push the image using the blob cache.
	destdir=${TESTDIR}/dest
	mkdir -p ${destdir}
	ls -l ${blobcachedir}
	echo buildah push --signature-policy ${TESTSDIR}/policy.json --blob-cache=${blobcachedir} ${target} dir:${destdir}
	run buildah push --signature-policy ${TESTSDIR}/policy.json --blob-cache=${blobcachedir} ${target} dir:${destdir}
	echo "$output"
	[ "$status" -eq 0 ]
	# Look for layer blobs in the destination that match the ones in the cache.
	matched=0
	unmatched=0
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
	[ ${matched} -eq 5 ] # the base image's layers, our new layer, the config, and the manifest should already have been cached
	[ ${unmatched} -eq 1 ] # expected mismatch is only the "version"
}

@test "blobcache-build-compressed-using-dockerfile-explicit-push" {
	blobcachedir=${TESTDIR}/cache
	mkdir -p ${blobcachedir}
	target=new-image
	# Build an image while pulling the base image.
	run buildah build-using-dockerfile -t ${target} --pull-always --blob-cache=${blobcachedir} --signature-policy ${TESTSDIR}/policy.json ${TESTSDIR}/bud/add-file
	echo "$output"
	[ "$status" -eq 0 ]
	# Try to push the image without the blob cache.
	doomeddir=${TESTDIR}/doomed
	mkdir -p ${doomeddir}
	run buildah push --signature-policy ${TESTSDIR}/policy.json ${target} dir:${doomeddir}
	echo "$output"
	[ "$status" -eq 0 ]
	# Look for layer blobs in the destination that match the ones in the cache.
	matched=0
	unmatched=0
	for content in ${doomeddir}/* ; do
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
	[ ${matched} -eq 4 ] # the base layer, our new layer, config, and manifest should all match the cache
	[ ${unmatched} -eq 1 ] # the only mismatch should be "version"
	# Now try to push the image using the blob cache.
	destdir=${TESTDIR}/dest
	mkdir -p ${destdir}
	run buildah push --signature-policy ${TESTSDIR}/policy.json --blob-cache=${blobcachedir} ${target} dir:${destdir}
	echo "$output"
	[ "$status" -eq 0 ]
	# Look for layer blobs in the destination that match the ones in the cache.
	matched=0
	unmatched=0
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
	[ ${matched} -eq 4 ] # the config, the base layer, our new layer, and the manifest should match the cache
	[ ${unmatched} -eq 1 ] # the only expected mismatch should be "version"
}

@test "blobcache-build-uncompressed-using-dockerfile-explicit-push" {
	blobcachedir=${TESTDIR}/cache
	mkdir -p ${blobcachedir}
	target=new-image
	# Build an image while pulling the base image.
	run buildah build-using-dockerfile -t ${target} -D --pull-always --blob-cache=${blobcachedir} --signature-policy ${TESTSDIR}/policy.json ${TESTSDIR}/bud/add-file
	echo "$output"
	[ "$status" -eq 0 ]
	# Try to push the image without the blob cache.
	doomeddir=${TESTDIR}/doomed
	mkdir -p ${doomeddir}
	run buildah push --signature-policy ${TESTSDIR}/policy.json ${target} dir:${doomeddir}
	echo "$output"
	[ "$status" -eq 0 ]
	# Look for layer blobs in the destination that match the ones in the cache.
	matched=0
	unmatched=0
	for content in ${doomeddir}/* ; do
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
	[ ${matched} -eq 2 ] # our new layer (written at build-time) and config should match the cache
	[ ${unmatched} -eq 3 ] # expected mismatches should be "version", and the base layers, which had to be recompressed
	# Now try to push the image using the blob cache.
	destdir=${TESTDIR}/dest
	mkdir -p ${destdir}
	run buildah push --signature-policy ${TESTSDIR}/policy.json --blob-cache=${blobcachedir} ${target} dir:${destdir}
	echo "$output"
	[ "$status" -eq 0 ]
	# Look for layer blobs in the destination that match the ones in the cache.
	matched=0
	unmatched=0
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
	[ ${matched} -eq 2 ] # the config and previously-compressed base layer should match the cache
	[ ${unmatched} -eq 3 ] # expected "version", our new layer, and the manifest to mismatch
}

@test "blobcache-build-compressed-using-dockerfile-implicit-push" {
	blobcachedir=${TESTDIR}/cache
	mkdir -p ${blobcachedir}
	target=new-image
	destdir=${TESTDIR}/dest
	mkdir -p ${destdir}
	# Build an image while pulling the base image, implicitly pushing while writing.
	run buildah build-using-dockerfile -t dir:${destdir} --pull-always --blob-cache=${blobcachedir} --signature-policy ${TESTSDIR}/policy.json ${TESTSDIR}/bud/add-file
	echo "$output"
	[ "$status" -eq 0 ]
	# Look for layer blobs in the destination that match the ones in the cache.
	matched=0
	unmatched=0
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
	[ ${matched} -eq 4 ] # the layers from the base image, our layer, our config, and our manifest should match items in the cache
	[ ${unmatched} -eq 1 ] # expect only the "version" to not match the cache
}

@test "blobcache-build-uncompressed-using-dockerfile-implicit-push" {
	blobcachedir=${TESTDIR}/cache
	mkdir -p ${blobcachedir}
	target=new-image
	destdir=${TESTDIR}/dest
	mkdir -p ${destdir}
	# Build an image while pulling the base image, implicitly pushing while writing.
	run buildah build-using-dockerfile -t dir:${destdir} -D --pull-always --blob-cache=${blobcachedir} --signature-policy ${TESTSDIR}/policy.json ${TESTSDIR}/bud/add-file
	echo "$output"
	[ "$status" -eq 0 ]
	# Look for layer blobs in the destination that match the ones in the cache.
	matched=0
	unmatched=0
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
	[ ${matched} -eq 4 ] # the layers from the base image, our layer, our config, and our manifest should match items in the cache
	[ ${unmatched} -eq 1 ] # expect only the "version" to not match the cache
}
