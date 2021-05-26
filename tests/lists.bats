#!/usr/bin/env bats

load helpers

IMAGE_LIST=docker://k8s.gcr.io/pause:3.1
IMAGE_LIST_DIGEST=docker://k8s.gcr.io/pause@sha256:f78411e19d84a252e53bff71a4407a5686c46983a2c2eeed83929b888179acea
IMAGE_LIST_INSTANCE=docker://k8s.gcr.io/pause@sha256:f365626a556e58189fc21d099fc64603db0f440bff07f77c740989515c544a39
IMAGE_LIST_AMD64_INSTANCE_DIGEST=sha256:59eec8837a4d942cc19a52b8c09ea75121acc38114a2c68b98983ce9356b8610
IMAGE_LIST_ARM_INSTANCE_DIGEST=sha256:c84b0a3a07b628bc4d62e5047d0f8dff80f7c00979e1e28a821a033ecda8fe53
IMAGE_LIST_ARM64_INSTANCE_DIGEST=sha256:f365626a556e58189fc21d099fc64603db0f440bff07f77c740989515c544a39
IMAGE_LIST_PPC64LE_INSTANCE_DIGEST=sha256:bcf9771c0b505e68c65440474179592ffdfa98790eb54ffbf129969c5e429990
IMAGE_LIST_S390X_INSTANCE_DIGEST=sha256:882a20ee0df7399a445285361d38b711c299ca093af978217112c73803546d5e

@test "manifest-create" {
    run_buildah manifest create foo
}

@test "manifest-inspect-id" {
    run_buildah manifest create foo
    cid=$output
    run_buildah manifest inspect $cid
}

@test "manifest-add" {
    run_buildah manifest create foo
    run_buildah manifest add foo ${IMAGE_LIST}
    run_buildah manifest rm foo
}

@test "manifest-add local image" {
    target=scratch-image
    run_buildah bud --signature-policy ${TESTSDIR}/policy.json -t ${target} ${TESTSDIR}/bud/from-scratch
    run_buildah manifest create foo
    run_buildah manifest add foo ${target}
    run_buildah manifest rm foo
}

@test "manifest-add-one" {
    run_buildah manifest create foo
    run_buildah manifest add --arch=arm64 foo ${IMAGE_LIST_INSTANCE}
    run_buildah manifest inspect foo
    expect_output --substring ${IMAGE_LIST_ARM64_INSTANCE_DIGEST}
    run_buildah 125 inspect --type image foo
    expect_output --substring "no image found"
    run_buildah inspect foo
    expect_output --substring ${IMAGE_LIST_ARM64_INSTANCE_DIGEST}
}

@test "manifest-add-all" {
    run_buildah manifest create foo
    run_buildah manifest add --all foo ${IMAGE_LIST}
    run_buildah manifest inspect foo
    expect_output --substring ${IMAGE_LIST_AMD64_INSTANCE_DIGEST}
    expect_output --substring ${IMAGE_LIST_ARM_INSTANCE_DIGEST}
    expect_output --substring ${IMAGE_LIST_ARM64_INSTANCE_DIGEST}
    expect_output --substring ${IMAGE_LIST_PPC64LE_INSTANCE_DIGEST}
    expect_output --substring ${IMAGE_LIST_S390X_INSTANCE_DIGEST}
}

@test "manifest-remove" {
    run_buildah manifest create foo
    run_buildah manifest add --all foo ${IMAGE_LIST}
    run_buildah manifest inspect foo
    expect_output --substring ${IMAGE_LIST_ARM64_INSTANCE_DIGEST}
    run_buildah manifest remove foo ${IMAGE_LIST_ARM64_INSTANCE_DIGEST}
    run_buildah manifest inspect foo
    expect_output --substring ${IMAGE_LIST_AMD64_INSTANCE_DIGEST}
    expect_output --substring ${IMAGE_LIST_ARM_INSTANCE_DIGEST}
    expect_output --substring ${IMAGE_LIST_PPC64LE_INSTANCE_DIGEST}
    expect_output --substring ${IMAGE_LIST_S390X_INSTANCE_DIGEST}
    run grep ${IMAGE_LIST_ARM64_INSTANCE_DIGEST} <<< "$output"
    [ $status -ne 0 ]
}

@test "manifest-remove-not-found" {
    run_buildah manifest create foo
    run_buildah manifest add foo ${IMAGE_LIST}
    run_buildah 125 manifest remove foo sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef
}

@test "manifest-rm failures" {
    run_buildah 125 manifest rm foo1
    expect_output --substring "foo1: image not known"
}

@test "manifest-push" {
    run_buildah manifest create foo
    run_buildah manifest add --all foo ${IMAGE_LIST}
    run_buildah manifest push --signature-policy ${TESTSDIR}/policy.json foo dir:${TESTDIR}/pushed
    case "$(go env GOARCH 2> /dev/null)" in
	    amd64) IMAGE_LIST_EXPECTED_INSTANCE_DIGEST=${IMAGE_LIST_AMD64_INSTANCE_DIGEST} ;;
	    arm64) IMAGE_LIST_EXPECTED_INSTANCE_DIGEST=${IMAGE_LIST_ARM64_INSTANCE_DIGEST} ;;
	    arm) IMAGE_LIST_EXPECTED_INSTANCE_DIGEST=${IMAGE_LIST_ARM_INSTANCE_DIGEST} ;;
	    ppc64le) IMAGE_LIST_EXPECTED_INSTANCE_DIGEST=${IMAGE_LIST_PPC64LE_INSTANCE_DIGEST} ;;
	    s390x) IMAGE_LIST_EXPECTED_INSTANCE_DIGEST=${IMAGE_LIST_S390X_INSTANCE_DIGEST} ;;
	    *) skip "current arch \"$(go env GOARCH 2> /dev/null)\" not present in manifest list" ;;
    esac
    run grep ${IMAGE_LIST_EXPECTED_INSTANCE_DIGEST##sha256} ${TESTDIR}/pushed/manifest.json
    [ $status -eq 0 ]
}

@test "manifest-push-all" {
    run_buildah manifest create foo
    run_buildah manifest add --all foo ${IMAGE_LIST}
    run_buildah manifest push --signature-policy ${TESTSDIR}/policy.json --all foo dir:${TESTDIR}/pushed
    run sha256sum ${TESTDIR}/pushed/*
    expect_output --substring ${IMAGE_LIST_AMD64_INSTANCE_DIGEST##sha256:}
    expect_output --substring ${IMAGE_LIST_ARM_INSTANCE_DIGEST##sha256:}
    expect_output --substring ${IMAGE_LIST_ARM64_INSTANCE_DIGEST##sha256:}
    expect_output --substring ${IMAGE_LIST_PPC64LE_INSTANCE_DIGEST##sha256:}
    expect_output --substring ${IMAGE_LIST_S390X_INSTANCE_DIGEST##sha256:}
}

@test "manifest-push-purge" {
    run_buildah manifest create foo
    run_buildah manifest add --arch=arm64 foo ${IMAGE_LIST}
    run_buildah manifest inspect foo
    run_buildah manifest push --signature-policy ${TESTSDIR}/policy.json --purge foo dir:${TESTDIR}/pushed
    run_buildah 125 manifest inspect foo
}

@test "manifest-push-rm" {
    run_buildah manifest create foo
    run_buildah manifest add --arch=arm64 foo ${IMAGE_LIST}
    run_buildah manifest inspect foo
    run_buildah manifest push --signature-policy ${TESTSDIR}/policy.json --rm foo dir:${TESTDIR}/pushed
    run_buildah 125 manifest inspect foo
}

@test "manifest-push should fail with nonexistent authfile" {
    run_buildah manifest create foo
    run_buildah manifest add --arch=arm64 foo ${IMAGE_LIST}
    run_buildah manifest inspect foo
    run_buildah 125 manifest push --authfile /tmp/nonexistent --signature-policy ${TESTSDIR}/policy.json --purge foo dir:${TESTDIR}/pushed

}

@test "manifest-push with nonexistent REGISTRY_AUTH_FILE: succeeds" {
  # This field should be ignored
  export REGISTRY_AUTH_FILE=/tmp/nonexistent
    run_buildah manifest create foo
    run_buildah manifest add --arch=arm64 foo ${IMAGE_LIST}
    run_buildah manifest inspect foo
    run_buildah manifest push --signature-policy ${TESTSDIR}/policy.json --purge foo dir:${TESTDIR}/pushed
}

@test "manifest-from-tag" {
    run_buildah from --signature-policy ${TESTSDIR}/policy.json --name test-container ${IMAGE_LIST}
    run_buildah inspect --format ''{{.OCIv1.Architecture}}' ${IMAGE_LIST}
    expect_output --substring $(go env GOARCH)
    run_buildah inspect --format ''{{.OCIv1.Architecture}}' test-container
    expect_output --substring $(go env GOARCH)
}

@test "manifest-from-digest" {
    run_buildah from --signature-policy ${TESTSDIR}/policy.json --name test-container ${IMAGE_LIST_DIGEST}
    run_buildah inspect --format ''{{.OCIv1.Architecture}}' ${IMAGE_LIST_DIGEST}
    expect_output --substring $(go env GOARCH)
    run_buildah inspect --format ''{{.OCIv1.Architecture}}' test-container
    expect_output --substring $(go env GOARCH)
}

@test "manifest-from-instance" {
    run_buildah from --signature-policy ${TESTSDIR}/policy.json --name test-container ${IMAGE_LIST_INSTANCE}
    run_buildah inspect --format ''{{.OCIv1.Architecture}}' ${IMAGE_LIST_INSTANCE}
    expect_output --substring arm64
    run_buildah inspect --format ''{{.OCIv1.Architecture}}' test-container
    expect_output --substring arm64
}

@test "manifest-no-matching-instance" {
    # Check that local images which we can't load the config and history for
    # don't just break multi-layer builds.
    #
    # Create a test list with some stuff in it.
    run_buildah manifest create test-list
    run_buildah manifest add --all test-list ${IMAGE_LIST}
    # Remove the entry for the current arch from the list.
    arch=$(go env GOARCH)
    run_buildah manifest inspect test-list
    archinstance=$(jq -r '.manifests|map(select(.platform.architecture=="'$arch'"))[].digest' <<< "$output")
    run_buildah manifest remove test-list $archinstance
    # Try to build using the build cache.
    mkdir ${TESTDIR}/build
    echo 'much content, wow.' > ${TESTDIR}/build/content.txt
    echo 'FROM scratch' > ${TESTDIR}/build/Dockerfile
    echo 'ADD content.txt /' >> ${TESTDIR}/build/Dockerfile
    run_buildah bud --layers --iidfile image-id.txt ${TESTDIR}/build
    # Make sure we can add the new image to the list.
    run_buildah manifest add test-list $(cat image-id.txt)
}
