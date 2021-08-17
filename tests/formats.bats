#!/usr/bin/env bats

load helpers

###################
#  check_imgtype  #  shortcut for running 'imgtype' and verifying image
###################
function check_imgtype() {
  # First argument: image name
  image="$1"

  # Second argument: expected image type, 'oci' or 'docker'
  imgtype_oci="application/vnd.oci.image.manifest.v1+json"
  imgtype_dkr="application/vnd.docker.distribution.manifest.v2+json"

  expect=""
  case "$2" in
      oci)    want=$imgtype_oci; reject=$imgtype_dkr;;
      docker) want=$imgtype_dkr; reject=$imgtype_oci;;
      *)      die "Internal error: unknown image type '$2'";;
  esac

  # First test: run imgtype with expected type, confirm exit 0 + no output
  echo "\$ imgtype -expected-manifest-type $want $image"
  run imgtype -expected-manifest-type $want $image
  echo "$output"
  if [[ $status -ne 0 ]]; then
    die "exit status is $status (expected 0)"
  fi
  expect_output "" "Checking imagetype($image) == $2"

  # Second test: the converse. Run imgtype with the WRONG expected type,
  # confirm error message and exit status 1
  echo "\$ imgtype -expected-manifest-type $reject $image [opposite test]"
  run imgtype -expected-manifest-type $reject $image
  echo "$output"
  if [[ $status -ne 1 ]]; then
    die "exit status is $status (expected 1)"
  fi

  # Can't embed entire string because the '+' sign is interpreted as regex
  expect_output --substring \
                "level=error msg=\"expected .* type \\\\\".*, got " \
                "Checking imagetype($image) == $2"
}


@test "write-formats" {
  run_buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json scratch
  cid=$output
  run_buildah commit --signature-policy ${TESTSDIR}/policy.json $cid scratch-image-default
  run_buildah commit --format docker --signature-policy ${TESTSDIR}/policy.json $cid scratch-image-docker
  run_buildah commit --format oci --signature-policy ${TESTSDIR}/policy.json $cid scratch-image-oci

  check_imgtype scratch-image-default oci
  check_imgtype scratch-image-oci     oci
  check_imgtype scratch-image-docker  docker
}

@test "bud-formats" {
  run_buildah build-using-dockerfile --signature-policy ${TESTSDIR}/policy.json -t scratch-image-default -f Containerfile ${TESTSDIR}/bud/from-scratch
  run_buildah build-using-dockerfile --format docker --signature-policy ${TESTSDIR}/policy.json -t scratch-image-docker -f Containerfile ${TESTSDIR}/bud/from-scratch
  run_buildah build-using-dockerfile --format oci --signature-policy ${TESTSDIR}/policy.json -t scratch-image-oci -f Containerfile ${TESTSDIR}/bud/from-scratch

  check_imgtype scratch-image-default oci
  check_imgtype scratch-image-oci     oci
  check_imgtype scratch-image-docker  docker
}
