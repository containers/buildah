#!/usr/bin/env bats

load helpers

fromreftest() {
  local img=$1

  run_buildah from --quiet --pull $WITH_POLICY_JSON $img
  cid=$output

  # If image includes '_v2sN', verify that image is schema version N
  local expected_schemaversion=$(expr "$img" : '.*_v2s\([0-9]\)')
  if [ -n "$expected_schemaversion" ]; then
      actual_schemaversion=$(imgtype -expected-manifest-type '*' -show-manifest $img | jq .schemaVersion)
      expect_output --from="$actual_schemaversion" "$expected_schemaversion" \
                    ".schemaversion of $img"
  fi

  # This is all we test: basically, that buildah doesn't crash when pushing
  pushdir=${TEST_SCRATCH_DIR}/fromreftest
  mkdir -p ${pushdir}/{1,2,3}
  run_buildah push $WITH_POLICY_JSON $img dir:${pushdir}/1
  run_buildah commit $WITH_POLICY_JSON $cid new-image
  run_buildah push $WITH_POLICY_JSON new-image dir:${pushdir}/2
  run_buildah rmi new-image
  run_buildah commit $WITH_POLICY_JSON $cid dir:${pushdir}/3

  run_buildah rm $cid
  rm -fr ${pushdir}
}

@test "from-by-digest-s2" {
  skip_if_rootless_environment
  fromreftest quay.io/libpod/testdigest_v2s2@sha256:755f4d90b3716e2bf57060d249e2cd61c9ac089b1233465c5c2cb2d7ee550fdb
}

@test "from-by-tag-s2" {
  skip_if_rootless_environment
  fromreftest quay.io/libpod/testdigest_v2s2:20200210
}

@test "from-by-repo-only-s2" {
  skip_if_rootless_environment
  fromreftest quay.io/libpod/testdigest_v2s2
}
