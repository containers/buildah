#!/usr/bin/env bats

load helpers

fromreftest() {
  run_buildah from --quiet --pull --signature-policy ${TESTSDIR}/policy.json $1
  cid=$output

  # If image includes '_v2sN', verify that image is schema version N
  expected_schemaversion=$(expr "$1" : '.*_v2s\([0-9]\)')
  if [ -n "$expected_schemaversion" ]; then
      actual_schemaversion=$(imgtype -expected-manifest-type '*' -show-manifest $1 | jq .schemaVersion)
      expect_output --from="$actual_schemaversion" "$expected_schemaversion" \
                    ".schemaversion of $1"
  fi

  # This is all we test: basically, that buildah doesn't crash when pushing
  pushdir=${TESTDIR}/fromreftest
  mkdir -p ${pushdir}/{1,2,3}
  run_buildah push --signature-policy ${TESTSDIR}/policy.json $1 dir:${pushdir}/1
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
  IMG=quay.io/libpod/testdigest_v2s1_with_dups@sha256:70d6c767101c907aa251c21fded459c0f1a481685eb764ca2f7a6162e24dd81a

  fromreftest ${IMG}

  # Verify that image meets our expectations (duplicate layers)
  # Surprisingly, we do this after fromreftest, not before, because fromreftest
  # has to pull the image for us.
  dups=$(imgtype -expected-manifest-type '*' -show-manifest ${IMG} | jq .fsLayers|grep blobSum|sort|uniq -cd)
  if [[ -z "$dups" ]]; then
      die "Image ${IMG} does not have any duplicate layers (expected: one dup)"
  fi
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
