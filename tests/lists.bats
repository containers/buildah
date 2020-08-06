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
}

@test "manifest-add-one" {
    run_buildah manifest create foo
    run_buildah manifest add --override-arch=arm64 foo ${IMAGE_LIST_INSTANCE}
    run_buildah 125 inspect foo
    expect_output --substring "does not exist"
    run_buildah manifest inspect foo
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
    run_buildah manifest add --override-arch=arm64 foo ${IMAGE_LIST}
    run_buildah manifest inspect foo
    run_buildah manifest push --signature-policy ${TESTSDIR}/policy.json --purge foo dir:${TESTDIR}/pushed
    run_buildah 125 manifest inspect foo
}

@test "manifest-push should fail with nonexist authfile" {
    run_buildah manifest create foo
    run_buildah manifest add --override-arch=arm64 foo ${IMAGE_LIST}
    run_buildah manifest inspect foo
    run_buildah 125 manifest push --authfile /tmp/nonexist --signature-policy ${TESTSDIR}/policy.json --purge foo dir:${TESTDIR}/pushed

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
