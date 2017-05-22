#!/usr/bin/env bats

load helpers

@test "write-formats" {
  buildimgtype
  cid=$(buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json scratch)
  buildah commit --signature-policy ${TESTSDIR}/policy.json $cid scratch-image-default
  buildah commit --format dockerv2 --signature-policy ${TESTSDIR}/policy.json $cid scratch-image-docker
  buildah commit --format ociv1 --signature-policy ${TESTSDIR}/policy.json $cid scratch-image-oci
  imgtype -expected-manifest-type application/vnd.oci.image.manifest.v1+json scratch-image-default
  imgtype -expected-manifest-type application/vnd.oci.image.manifest.v1+json scratch-image-oci
  imgtype -expected-manifest-type application/vnd.docker.distribution.manifest.v2+json scratch-image-docker
  run imgtype -expected-manifest-type application/vnd.docker.distribution.manifest.v2+json scratch-image-default
  [ "$status" -ne 0 ]
  run imgtype -expected-manifest-type application/vnd.docker.distribution.manifest.v2+json scratch-image-oci
  [ "$status" -ne 0 ]
  run imgtype -expected-manifest-type application/vnd.oci.image.manifest.v1+json scratch-image-docker
  [ "$status" -ne 0 ]
}
