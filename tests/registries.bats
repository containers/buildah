#!/usr/bin/env bats

load helpers

@test "registries" {
  registrypair() {
    image=$1
    imagename=$2

    # Clean up.
    for id in $(buildah containers -q) ; do
      run_buildah rm ${id}
    done
    for id in $(buildah images -q) ; do
      run_buildah rmi ${id}
    done

    # Create a container by specifying the image with one name.
    run_buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json $image

    # Create a container by specifying the image with another name.
    run_buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json $imagename

    # Get their image IDs.  They should be the same one.
    lastid=
    for cid in $(buildah containers -q) ; do
      run_buildah inspect -f "{{.FromImageID}}" $cid
      expect_line_count 1
      if [ "$lastid" != "" ] ; then
        expect_output "$lastid"
      fi
      lastid="$output"
    done

    # A quick bit of troubleshooting help.
    run_buildah images

    # Clean up.
    for id in $(buildah containers -q) ; do
      run_buildah rm ${id}
    done
    for id in $(buildah images -q) ; do
      run_buildah rmi ${id}
    done
  }
  # Test with pairs of short and fully-qualified names that should be the same image.
  registrypair busybox docker.io/busybox
  registrypair docker.io/busybox busybox
  registrypair busybox docker.io/library/busybox
  registrypair docker.io/library/busybox busybox
  registrypair fedora-minimal registry.fedoraproject.org/fedora-minimal
  registrypair registry.fedoraproject.org/fedora-minimal fedora-minimal
}
