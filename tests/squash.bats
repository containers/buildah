#!/usr/bin/env bats

load helpers


function check_lengths() {
  local image=$1
  local expect=$2

  # matrix test: check given .Docker.* and .OCIv1.* fields in image
  for which in Docker OCIv1; do
    for field in RootFS.DiffIDs History; do
      run_buildah --debug=false inspect -t image -f "{{len .$which.$field}}" $image
      expect_output "$expect"
    done
  done
}

@test "squash" {
	createrandom ${TESTDIR}/randomfile
	cid=$(buildah from scratch)
	image=stage0
	remove=(8 5)
	for stage in $(seq 10) ; do
		buildah copy "$cid" ${TESTDIR}/randomfile /layer${stage}
		image=stage${stage}
		if test $stage -eq ${remove[0]} ; then
			mountpoint=$(buildah mount "$cid")
			rm -f ${mountpoint}/layer${remove[1]}
		fi
		buildah commit --signature-policy ${TESTSDIR}/policy.json --rm "$cid" ${image}
                check_lengths $image $stage
		cid=$(buildah from ${image})
	done
	buildah commit --signature-policy ${TESTSDIR}/policy.json --rm --squash "$cid" squashed

        check_lengths squashed 1

	cid=$(buildah from squashed)
	mountpoint=$(buildah mount $cid)
	for stage in $(seq 10) ; do
		if test $stage -eq ${remove[1]} ; then
			if test -e $mountpoint/layer${remove[1]} ; then
				echo file /layer${remove[1]} should not be there
				exit 1
			fi
			continue
		fi
		cmp $mountpoint/layer${stage} ${TESTDIR}/randomfile
	done
}

@test "squash-using-dockerfile" {
	createrandom ${TESTDIR}/randomfile
	image=stage0
	from=scratch
	for stage in $(seq 10) ; do
		mkdir -p ${TESTDIR}/stage${stage}
		echo FROM ${from} > ${TESTDIR}/stage${stage}/Dockerfile
		cp ${TESTDIR}/randomfile ${TESTDIR}/stage${stage}/
		echo COPY randomfile /layer${stage} >> ${TESTDIR}/stage${stage}/Dockerfile
		image=stage${stage}
		from=${image}
		buildah build-using-dockerfile --signature-policy ${TESTSDIR}/policy.json -t ${image} ${TESTDIR}/stage${stage}
                check_lengths $image $stage
	done

	mkdir -p ${TESTDIR}/squashed
	echo FROM ${from} > ${TESTDIR}/squashed/Dockerfile
	cp ${TESTDIR}/randomfile ${TESTDIR}/squashed/
	echo COPY randomfile /layer-squashed >> ${TESTDIR}/stage${stage}/Dockerfile
	buildah build-using-dockerfile --signature-policy ${TESTSDIR}/policy.json --squash -t squashed ${TESTDIR}/squashed

        check_lengths squashed 1

	cid=$(buildah from squashed)
	mountpoint=$(buildah mount $cid)
	for stage in $(seq 10) ; do
		cmp $mountpoint/layer${stage} ${TESTDIR}/randomfile
	done

	buildah build-using-dockerfile --signature-policy ${TESTSDIR}/policy.json --squash --layers -t squashed ${TESTDIR}/squashed
	run_buildah --debug=false inspect -t image -f '{{len .Docker.RootFS.DiffIDs}}' squashed
	[ "$output" -eq 1 ]

	echo FROM ${from} > ${TESTDIR}/squashed/Dockerfile
	buildah build-using-dockerfile --signature-policy ${TESTSDIR}/policy.json --squash -t squashed ${TESTDIR}/squashed
	run_buildah --debug=false inspect -t image -f '{{len .Docker.RootFS.DiffIDs}}' squashed
	[ "$output" -eq 1 ]

	echo FROM ${from} > ${TESTDIR}/squashed/Dockerfile
	buildah build-using-dockerfile --signature-policy ${TESTSDIR}/policy.json --squash --layers -t squashed ${TESTDIR}/squashed
	run_buildah --debug=false inspect -t image -f '{{len .Docker.RootFS.DiffIDs}}' squashed
	[ "$output" -eq 1 ]
}
