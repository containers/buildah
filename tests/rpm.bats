#!/usr/bin/env bats

load helpers

@test "rpm-build" {
	if ! which runc ; then
		skip
	fi

	# Build a container to use for building the binaries.
	image=fedora:26
	cid=$(buildah --debug=false from --pull --signature-policy ${TESTSDIR}/policy.json $image)
	root=$(buildah --debug=false mount $cid)
	commit=$(git log --format=%H -n 1)
	shortcommit=$(echo ${commit} | cut -c-7)
	mkdir -p ${root}/rpmbuild/{SOURCES,SPECS}

	# Build the tarball.
	(cd ..; git archive --format tar.gz --prefix=buildah-${commit}/ ${commit}) > ${root}/rpmbuild/SOURCES/buildah-${shortcommit}.tar.gz

	# Update the .spec file with the commit ID.
	sed s:REPLACEWITHCOMMITID:${commit}:g ${TESTSDIR}/../contrib/rpm/buildah.spec > ${root}/rpmbuild/SPECS/buildah.spec

	# Install build dependencies and build binary packages.
	buildah --debug=false run $cid -- dnf -y install 'dnf-command(builddep)' rpm-build
	buildah --debug=false run $cid -- dnf -y builddep --spec rpmbuild/SPECS/buildah.spec
	buildah --debug=false run $cid -- rpmbuild --define "_topdir /rpmbuild" -ba /rpmbuild/SPECS/buildah.spec

	# Build a second new container.
	cid2=$(buildah --debug=false from --pull --signature-policy ${TESTSDIR}/policy.json fedora:26)
	root2=$(buildah --debug=false mount $cid2)

	# Copy the binary packages from the first container to the second one, and build a list of
	# their filenames relative to the root of the second container.
	rpms=
	mkdir -p ${root2}/packages
	for rpm in ${root}/rpmbuild/RPMS/*/*.rpm ; do
		cp $rpm ${root2}/packages/
		rpms="$rpms "/packages/$(basename $rpm)
	done

	# Install the binary packages into the second container.
	buildah --debug=false run $cid2 -- dnf -y install $rpms

	# Run the binary package and compare its self-identified version to the one we tried to build.
	id=$(buildah --debug=false run $cid2 -- buildah version | awk '/^Git Commit:/ { print $NF }')
	bv=$(buildah --debug=false run $cid2 -- buildah version | awk '/^Version:/ { print $NF }')
	rv=$(buildah --debug=false run $cid2 -- rpm -q --queryformat '%{version}' buildah)
	echo "short commit: $shortcommit"
	echo "id: $id"
	echo "buildah version: $bv"
	echo "buildah rpm version: $rv"
	test $shortcommit = $id
	test $bv = $rv

	# Clean up.
	buildah --debug=false rm $cid $cid2
}
