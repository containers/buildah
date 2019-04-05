#!/usr/bin/env bats

load helpers

@test "push-flags-order-verification" {
  run_buildah 1 push img1 dest1 -q
  check_options_flag_err "-q"

  run_buildah 1 push img1 --tls-verify dest1
  check_options_flag_err "--tls-verify"

  run_buildah 1 push img1 dest1 arg3 --creds user1:pass1
  check_options_flag_err "--creds"

  run_buildah 1 push img1 --creds=user1:pass1 dest1
  check_options_flag_err "--creds=user1:pass1"
}

@test "push" {
  touch ${TESTDIR}/reference-time-file
  for source in scratch scratch-image; do
    cid=$(buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json ${source})
    for format in "" docker oci ; do
      mkdir -p ${TESTDIR}/committed${format:+.${format}}
      # Force no compression to generate what we push.
      buildah commit -D ${format:+--format ${format}} --reference-time ${TESTDIR}/reference-time-file --signature-policy ${TESTSDIR}/policy.json "$cid" scratch-image${format:+-${format}}
      buildah commit -D ${format:+--format ${format}} --reference-time ${TESTDIR}/reference-time-file --signature-policy ${TESTSDIR}/policy.json "$cid" dir:${TESTDIR}/committed${format:+.${format}}
      mkdir -p ${TESTDIR}/pushed${format:+.${format}}
      buildah push -D --signature-policy ${TESTSDIR}/policy.json scratch-image${format:+-${format}} dir:${TESTDIR}/pushed${format:+.${format}}
      # Reencode the manifest to lose variations due to different encoders or definitions of structures.
      imgtype -expected-manifest-type "*" -rebuild-manifest -show-manifest dir:${TESTDIR}/committed${format:+.${format}} > ${TESTDIR}/manifest.committed${format:+.${format}}
      imgtype -expected-manifest-type "*" -rebuild-manifest -show-manifest dir:${TESTDIR}/pushed${format:+.${format}} > ${TESTDIR}/manifest.pushed${format:+.${format}}
      diff -u ${TESTDIR}/manifest.committed${format:+.${format}} ${TESTDIR}/manifest.pushed${format:+.${format}}
      [ "$output" = "" ]
    done
    buildah rm "$cid"
  done
}

@test "push with manifest type conversion" {
  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  run_buildah push --signature-policy ${TESTSDIR}/policy.json --format oci alpine dir:my-dir
  manifest=$(cat my-dir/manifest.json)
  run grep "application/vnd.oci.image.config.v1+json" <<< "$manifest"
  echo "$output"
  [ "$status" -eq 0 ]
  run_buildah push --signature-policy ${TESTSDIR}/policy.json --format v2s2 alpine dir:my-dir
  run grep "application/vnd.docker.distribution.manifest.v2+json" my-dir/manifest.json
  echo "$output"
  [ "$status" -eq 0 ]
  buildah rm "$cid"
  buildah rmi alpine
  rm -rf my-dir
}

@test "push with imageid" {
  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  imageid=$(buildah images -q)
  run_buildah push --signature-policy ${TESTSDIR}/policy.json $imageid dir:my-dir
  buildah rm "$cid"
  buildah rmi alpine
  rm -rf my-dir
}

@test "push without destination" {
  buildah pull --signature-policy ${TESTSDIR}/policy.json busybox
  run_buildah 1 push --signature-policy ${TESTSDIR}/policy.json busybox
  expect_output --substring "docker://busybox"
  buildah rmi busybox
}
