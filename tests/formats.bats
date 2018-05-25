#!/usr/bin/env bats

load helpers

@test "write-formats" {
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

@test "bud-formats" {
  buildah build-using-dockerfile --signature-policy ${TESTSDIR}/policy.json -t scratch-image-default -f Dockerfile bud/from-scratch
  buildah build-using-dockerfile --format dockerv2 --signature-policy ${TESTSDIR}/policy.json -t scratch-image-docker -f Dockerfile bud/from-scratch
  buildah build-using-dockerfile --format ociv1 --signature-policy ${TESTSDIR}/policy.json -t scratch-image-oci -f Dockerfile bud/from-scratch
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
