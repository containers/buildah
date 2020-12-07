#!/usr/bin/env bats

load helpers

@test "registries" {
  registrypair() {
    image1=$1
    image2=$2

    # Create a container by specifying the image with one name.
    run_buildah --retry from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json $image1
    cid1=$output

    # Create a container by specifying the image with another name.
    run_buildah --retry from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json $image2
    cid2=$output

    # Get their image IDs.  They should be the same one.
    run_buildah inspect -f "{{.FromImageID}}" $cid1
    iid1=$output
    run_buildah inspect -f "{{.FromImageID}}" $cid2
    expect_output $iid1 "$image2.FromImageID == $image1.FromImageID"

    # Clean up.
    run_buildah rm -a
    run_buildah rmi -a
  }
  # Test with pairs of short and fully-qualified names that should be the same image.
  registrypair busybox           docker.io/busybox
  registrypair busybox           docker.io/library/busybox
  registrypair fedora-minimal:32 registry.fedoraproject.org/fedora-minimal:32
}
