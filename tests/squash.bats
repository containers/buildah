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
	createrandom ${TEST_SCRATCH_DIR}/randomfile
	run_buildah from scratch
	cid=$output
	image=stage0
	remove=(8 5)
	for stage in $(seq 10) ; do
		run_buildah copy "$cid" ${TEST_SCRATCH_DIR}/randomfile /layer${stage}
		image=stage${stage}
		if test $stage -eq ${remove[0]} ; then
			run_buildah mount "$cid"
			mountpoint=$output
			rm -f ${mountpoint}/layer${remove[1]}
		fi
		run_buildah commit $WITH_POLICY_JSON --rm "$cid" ${image}
                check_lengths $image $stage
		run_buildah from --quiet ${image}
		cid=$output
	done
	run_buildah commit $WITH_POLICY_JSON --rm --squash "$cid" squashed

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
		cmp $mountpoint/layer${stage} ${TEST_SCRATCH_DIR}/randomfile
	done
}

@test "squash-using-dockerfile" {
	createrandom ${TEST_SCRATCH_DIR}/randomfile
	image=stage0
	from=scratch
	for stage in $(seq 10) ; do
		mkdir -p ${TEST_SCRATCH_DIR}/stage${stage}
		echo FROM ${from} > ${TEST_SCRATCH_DIR}/stage${stage}/Dockerfile
		cp ${TEST_SCRATCH_DIR}/randomfile ${TEST_SCRATCH_DIR}/stage${stage}/
		echo COPY randomfile /layer${stage} >> ${TEST_SCRATCH_DIR}/stage${stage}/Dockerfile
		image=stage${stage}
		from=${image}
		run_buildah build-using-dockerfile $WITH_POLICY_JSON -t ${image} ${TEST_SCRATCH_DIR}/stage${stage}
                check_lengths $image $stage
	done

	mkdir -p ${TEST_SCRATCH_DIR}/squashed
	echo FROM ${from} > ${TEST_SCRATCH_DIR}/squashed/Dockerfile
	cp ${TEST_SCRATCH_DIR}/randomfile ${TEST_SCRATCH_DIR}/squashed/
	echo COPY randomfile /layer-squashed >> ${TEST_SCRATCH_DIR}/stage${stage}/Dockerfile
	run_buildah build-using-dockerfile $WITH_POLICY_JSON --squash -t squashed ${TEST_SCRATCH_DIR}/squashed

        check_lengths squashed 1

	run_buildah from --quiet squashed
	cid=$output
	run_buildah mount $cid
	mountpoint=$output
	for stage in $(seq 10) ; do
		cmp $mountpoint/layer${stage} ${TEST_SCRATCH_DIR}/randomfile
	done

	run_buildah build-using-dockerfile $WITH_POLICY_JSON --squash --layers -t squashed ${TEST_SCRATCH_DIR}/squashed
	run_buildah inspect -t image -f '{{len .Docker.RootFS.DiffIDs}}' squashed
        expect_output "1" "len(DiffIDs) - simple image"

	echo FROM ${from} > ${TEST_SCRATCH_DIR}/squashed/Dockerfile
	run_buildah build-using-dockerfile $WITH_POLICY_JSON --squash -t squashed ${TEST_SCRATCH_DIR}/squashed
	run_buildah inspect -t image -f '{{len .Docker.RootFS.DiffIDs}}' squashed
        expect_output "1" "len(DiffIDs) - image with FROM"

	echo USER root >> ${TEST_SCRATCH_DIR}/squashed/Dockerfile
	run_buildah build-using-dockerfile $WITH_POLICY_JSON --squash -t squashed ${TEST_SCRATCH_DIR}/squashed
	run_buildah inspect -t image -f '{{len .Docker.RootFS.DiffIDs}}' squashed
        expect_output "1" "len(DiffIDs) - image with FROM and USER"

	echo COPY file / >> ${TEST_SCRATCH_DIR}/squashed/Dockerfile
	echo COPY file / > ${TEST_SCRATCH_DIR}/squashed/file
	run_buildah build-using-dockerfile $WITH_POLICY_JSON --squash -t squashed ${TEST_SCRATCH_DIR}/squashed
	run_buildah inspect -t image -f '{{len .Docker.RootFS.DiffIDs}}' squashed
        expect_output "1" "len(DiffIDs) - image with FROM, USER, and 2xCOPY"

	echo FROM ${from} > ${TEST_SCRATCH_DIR}/squashed/Dockerfile
	run_buildah build-using-dockerfile $WITH_POLICY_JSON --squash --layers -t squashed ${TEST_SCRATCH_DIR}/squashed
	run_buildah inspect -t image -f '{{len .Docker.RootFS.DiffIDs}}' squashed
        expect_output "1" "len(DiffIDs) - image with FROM (--layers)"

	echo USER root >> ${TEST_SCRATCH_DIR}/squashed/Dockerfile
	run_buildah build-using-dockerfile $WITH_POLICY_JSON --squash -t squashed ${TEST_SCRATCH_DIR}/squashed
	run_buildah inspect -t image -f '{{len .Docker.RootFS.DiffIDs}}' squashed
        expect_output "1" "len(DiffIDs) - image with FROM and USER (--layers)"

	echo COPY file / >> ${TEST_SCRATCH_DIR}/squashed/Dockerfile
	echo COPY file / > ${TEST_SCRATCH_DIR}/squashed/file
	run_buildah build-using-dockerfile $WITH_POLICY_JSON --squash -t squashed ${TEST_SCRATCH_DIR}/squashed
	run_buildah inspect -t image -f '{{len .Docker.RootFS.DiffIDs}}' squashed
        expect_output "1" "len(DiffIDs) - image with FROM, USER, and 2xCOPY (--layers)"

	run_buildah build-using-dockerfile $WITH_POLICY_JSON --squash --format docker -t squashed ${TEST_SCRATCH_DIR}/squashed
	run_buildah inspect -t image -f '{{.Docker.Parent}}' squashed
        expect_output "" "should have no parent image set"
}


@test "bud-squash-should-use-cache" {
  _prefetch alpine
  # populate cache from simple build
  run_buildah build --layers -t test $WITH_POLICY_JSON -f $BUDFILES/layers-squash/Dockerfile.multi-stage
  # create another squashed build and check if we are using cache for everything.
  # instead of last instruction in last stage
  run_buildah build --layers --squash -t testsquash $WITH_POLICY_JSON -f $BUDFILES/layers-squash/Dockerfile.multi-stage
  expect_output --substring "Using cache"
  run_buildah inspect -t image -f '{{len .Docker.RootFS.DiffIDs}}' testsquash
  expect_output "1" "image built with --squash should only include 1 layer"
  run_buildah rmi -f testsquash
  run_buildah rmi -f test
}

# Test build with --squash and --layers and verify number of layers and content inside image
@test "bud-squash-should-use-cache and verify content inside image" {
  mkdir -p ${TEST_SCRATCH_DIR}/bud/platform

  cat > ${TEST_SCRATCH_DIR}/bud/platform/Dockerfile << _EOF
FROM busybox
RUN touch hello
ADD . /data
RUN echo hey && mkdir water
_EOF

  # Build a first image with --layers and --squash and populate build cache
  run_buildah build $WITH_POLICY_JSON --squash --layers -t one -f ${TEST_SCRATCH_DIR}/bud/platform/Dockerfile ${TEST_SCRATCH_DIR}/bud/platform
  run_buildah inspect -t image -f '{{len .Docker.RootFS.DiffIDs}}' one
  expect_output "1" "image built with --squash should only include 1 layer"
  # Build again and verify if cache is being used
  run_buildah build $WITH_POLICY_JSON --squash --layers -t two -f ${TEST_SCRATCH_DIR}/bud/platform/Dockerfile ${TEST_SCRATCH_DIR}/bud/platform
  expect_output --substring "Using cache"
  run_buildah inspect -t image -f '{{len .Docker.RootFS.DiffIDs}}' two
  expect_output "1" "image built with --squash should only include 1 layer"
  run_buildah from two
  run_buildah run two-working-container ls
  expect_output --substring "water"
  expect_output --substring "data"
  expect_output --substring "hello"
}
