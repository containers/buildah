#!/usr/bin/env bats

load helpers

fromreftest() {
  local img=$1

  run_buildah from --quiet --pull --signature-policy ${TESTSDIR}/policy.json $img
  cid=$output

  # If image includes '_v2sN', verify that image is schema version N
  local expected_schemaversion=$(expr "$img" : '.*_v2s\([0-9]\)')
  if [ -n "$expected_schemaversion" ]; then
      actual_schemaversion=$(imgtype -expected-manifest-type '*' -show-manifest $img | jq .schemaVersion)
      expect_output --from="$actual_schemaversion" "$expected_schemaversion" \
                    ".schemaversion of $img"
  fi

  # This is all we test: basically, that buildah doesn't crash when pushing
  pushdir=${TESTDIR}/fromreftest
  mkdir -p ${pushdir}/{1,2,3}
  run_buildah push --signature-policy ${TESTSDIR}/policy.json $img dir:${pushdir}/1
  run_buildah commit --signature-policy ${TESTSDIR}/policy.json $cid new-image
  run_buildah push --signature-policy ${TESTSDIR}/policy.json new-image dir:${pushdir}/2
  run_buildah rmi new-image
  run_buildah commit --signature-policy ${TESTSDIR}/policy.json $cid dir:${pushdir}/3

  run_buildah rm $cid
  rm -fr ${pushdir}
}

@test "from-by-digest-s1" {
  fromreftest quay.io/libpod/testdigest_v2s1@sha256:816563225d7baae4782653efc9410579341754fe32cbe20f7600b39fc37d8ec7
}

@test "from-by-digest-s1-a-discarded-layer" {
  IMG=quay.io/libpod/testdigest_v2s1_with_dups@sha256:2c619fffbed29d8677e246798333e7d1b288333cb61c020575f6372c76fdbb52

  fromreftest ${IMG}

  # Verify that image meets our expectations (duplicate layers)
  # Surprisingly, we do this after fromreftest, not before, because fromreftest
  # has to pull the image for us.
  #
  # Check that the first and second .fsLayers and .history elements are dups
  local manifest=$(imgtype -expected-manifest-type '*' -show-manifest ${IMG})
  for element in fsLayers history; do
      local first=$(jq ".${element}[0]" <<<"$manifest")
      local second=$(jq ".${element}[1]" <<<"$manifest")
      expect_output --from="$second" "$first" "${IMG}: .${element}[1] == [0]"
  done
}

@test "from-by-tag-s1" {
  fromreftest quay.io/libpod/testdigest_v2s1:20200210
}

@test "from-by-digest-s2" {
  fromreftest quay.io/libpod/testdigest_v2s2@sha256:755f4d90b3716e2bf57060d249e2cd61c9ac089b1233465c5c2cb2d7ee550fdb
}

@test "from-by-tag-s2" {
  fromreftest quay.io/libpod/testdigest_v2s2:20200210
}

@test "from-by-repo-only-s2" {
  fromreftest quay.io/libpod/testdigest_v2s2
}
