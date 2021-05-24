#!/usr/bin/env bats

load helpers


function check_lengths() {
  local image=$1
  local expect=$2

  # matrix test: check given .Docker.* and .OCIv1.* fields in image
  for which in Docker OCIv1; do
    for field in RootFS.DiffIDs History; do
      run_buildah inspect -t image -f "{{len .$which.$field}}" $image
      expect_output "$expect"
    done
  done
}

@test "squash" {
	createrandom ${TESTDIR}/randomfile
	run_buildah from scratch
	cid=$output
	image=stage0
	remove=(8 5)
	for stage in $(seq 10) ; do
		run_buildah copy "$cid" ${TESTDIR}/randomfile /layer${stage}
		image=stage${stage}
		if test $stage -eq ${remove[0]} ; then
			run_buildah mount "$cid"
			mountpoint=$output
			rm -f ${mountpoint}/layer${remove[1]}
		fi
		run_buildah commit --signature-policy ${TESTSDIR}/policy.json --rm "$cid" ${image}
                check_lengths $image $stage
		run_buildah from --quiet ${image}
		cid=$output
	done
	run_buildah commit --signature-policy ${TESTSDIR}/policy.json --rm --squash "$cid" squashed

        check_lengths squashed 1

	run_buildah from --quiet squashed
	cid=$output
	run_buildah mount $cid
	mountpoint=$output
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
		run_buildah build-using-dockerfile --signature-policy ${TESTSDIR}/policy.json -t ${image} ${TESTDIR}/stage${stage}
                check_lengths $image $stage
	done

	mkdir -p ${TESTDIR}/squashed
	echo FROM ${from} > ${TESTDIR}/squashed/Dockerfile
	cp ${TESTDIR}/randomfile ${TESTDIR}/squashed/
	echo COPY randomfile /layer-squashed >> ${TESTDIR}/stage${stage}/Dockerfile
	run_buildah build-using-dockerfile --signature-policy ${TESTSDIR}/policy.json --squash -t squashed ${TESTDIR}/squashed

        check_lengths squashed 1

	run_buildah from --quiet squashed
	cid=$output
	run_buildah mount $cid
	mountpoint=$output
	for stage in $(seq 10) ; do
		cmp $mountpoint/layer${stage} ${TESTDIR}/randomfile
	done

	run_buildah build-using-dockerfile --signature-policy ${TESTSDIR}/policy.json --squash --layers -t squashed ${TESTDIR}/squashed
	run_buildah inspect -t image -f '{{len .Docker.RootFS.DiffIDs}}' squashed
        expect_output "1" "len(DiffIDs) - simple image"

	echo FROM ${from} > ${TESTDIR}/squashed/Dockerfile
	run_buildah build-using-dockerfile --signature-policy ${TESTSDIR}/policy.json --squash -t squashed ${TESTDIR}/squashed
	run_buildah inspect -t image -f '{{len .Docker.RootFS.DiffIDs}}' squashed
        expect_output "1" "len(DiffIDs) - image with FROM"

	echo USER root >> ${TESTDIR}/squashed/Dockerfile
	run_buildah build-using-dockerfile --signature-policy ${TESTSDIR}/policy.json --squash -t squashed ${TESTDIR}/squashed
	run_buildah inspect -t image -f '{{len .Docker.RootFS.DiffIDs}}' squashed
        expect_output "1" "len(DiffIDs) - image with FROM and USER"

	echo COPY file / >> ${TESTDIR}/squashed/Dockerfile
	echo COPY file / > ${TESTDIR}/squashed/file
	run_buildah build-using-dockerfile --signature-policy ${TESTSDIR}/policy.json --squash -t squashed ${TESTDIR}/squashed
	run_buildah inspect -t image -f '{{len .Docker.RootFS.DiffIDs}}' squashed
        expect_output "1" "len(DiffIDs) - image with FROM, USER, and 2xCOPY"

	echo FROM ${from} > ${TESTDIR}/squashed/Dockerfile
	run_buildah build-using-dockerfile --signature-policy ${TESTSDIR}/policy.json --squash --layers -t squashed ${TESTDIR}/squashed
	run_buildah inspect -t image -f '{{len .Docker.RootFS.DiffIDs}}' squashed
        expect_output "1" "len(DiffIDs) - image with FROM (--layers)"

	echo USER root >> ${TESTDIR}/squashed/Dockerfile
	run_buildah build-using-dockerfile --signature-policy ${TESTSDIR}/policy.json --squash -t squashed ${TESTDIR}/squashed
	run_buildah inspect -t image -f '{{len .Docker.RootFS.DiffIDs}}' squashed
        expect_output "1" "len(DiffIDs) - image with FROM and USER (--layers)"

	echo COPY file / >> ${TESTDIR}/squashed/Dockerfile
	echo COPY file / > ${TESTDIR}/squashed/file
	run_buildah build-using-dockerfile --signature-policy ${TESTSDIR}/policy.json --squash -t squashed ${TESTDIR}/squashed
	run_buildah inspect -t image -f '{{len .Docker.RootFS.DiffIDs}}' squashed
        expect_output "1" "len(DiffIDs) - image with FROM, USER, and 2xCOPY (--layers)"

	run_buildah build-using-dockerfile --signature-policy ${TESTSDIR}/policy.json --squash --format docker -t squashed ${TESTDIR}/squashed
	run_buildah inspect -t image -f '{{.Docker.Parent}}' squashed
        expect_output "" "should have no parent image set"
}
