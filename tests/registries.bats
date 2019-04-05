#!/usr/bin/env bats

load helpers

@test "registries" {
  registrypair() {
    image=$1
    imagename=$2

    # Clean up.
    for id in $(buildah --debug=false containers -q) ; do
      buildah rm ${id}
    done
    for id in $(buildah --debug=false images -q) ; do
      buildah rmi ${id}
    done

    # Create a container by specifying the image with one name.
    buildah from --pull --signature-policy ${TESTSDIR}/policy.json $image

    # Create a container by specifying the image with another name.
    buildah from --pull --signature-policy ${TESTSDIR}/policy.json $imagename

    # Get their image IDs.  They should be the same one.
    lastid=
    for cid in $(buildah --debug=false containers -q) ; do
      run_buildah --debug=false inspect -f "{{.FromImageID}}" $cid
      expect_line_count 1
      if [ "$lastid" != "" ] ; then
        expect_output "$lastid"
      fi
      lastid="$output"
    done

    # A quick bit of troubleshooting help.
    run_buildah images

    # Clean up.
    for id in $(buildah --debug=false containers -q) ; do
      buildah rm ${id}
    done
    for id in $(buildah --debug=false images -q) ; do
      buildah rmi ${id}
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
