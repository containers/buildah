#!/usr/bin/env bats

load helpers

@test "push" {
  touch ${TESTDIR}/reference-time-file
  for source in scratch scratch-image; do
    cid=$(buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json ${source})
    for format in "" docker oci ; do
      mkdir -p ${TESTDIR}/committed${format:+.${format}}
      buildah commit ${format:+--format ${format}} --reference-time ${TESTDIR}/reference-time-file --signature-policy ${TESTSDIR}/policy.json "$cid" scratch-image${format:+-${format}}
      buildah commit ${format:+--format ${format}} --reference-time ${TESTDIR}/reference-time-file --signature-policy ${TESTSDIR}/policy.json "$cid" dir:${TESTDIR}/committed${format:+.${format}}
      mkdir -p ${TESTDIR}/pushed${format:+.${format}}
      buildah push --signature-policy ${TESTSDIR}/policy.json scratch-image${format:+-${format}} dir:${TESTDIR}/pushed${format:+.${format}}
      diff -u ${TESTDIR}/committed${format:+.${format}}/manifest.json ${TESTDIR}/pushed${format:+.${format}}/manifest.json
      [ "$output" = "" ]
    done
    buildah rm "$cid"
  done
}

@test "push with manifest type conversion" {
  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  run buildah push --signature-policy ${TESTSDIR}/policy.json --format oci alpine dir:my-dir
  echo "$output"
  [ "$status" -eq 0 ]
  manifest=$(cat my-dir/manifest.json)
  run grep "application/vnd.oci.image.config.v1+json" <<< "$manifest"
  echo "$output"
  [ "$status" -eq 0 ]
  run buildah push --signature-policy ${TESTSDIR}/policy.json --format v2s2 alpine dir:my-dir
  echo "$output"
  [ "$status" -eq 0 ]
  run grep "application/vnd.docker.distribution.manifest.v2+json" my-dir/manifest.json
  echo "$output"
  [ "$status" -eq 0 ]
  buildah rm "$cid"
  buildah rmi alpine
  rm -rf my-dir
}
