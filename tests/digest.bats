#!/usr/bin/env bats

load helpers

fromreftest() {
  _prefetch $1
  run_buildah from --quiet --pull --signature-policy ${TESTSDIR}/policy.json $1
  cid=$output
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
  fromreftest k8s.gcr.io/pause@sha256:bbeaef1d40778579b7b86543fe03e1ec041428a50d21f7a7b25630e357ec9247
}

@test "from-by-digest-s1-a-discarded-layer" {
  fromreftest libpod/whalesay@sha256:2413c2ffc29fb01d51c27a91b804079995d6037eed9e4b632249fce8c8708eb4
}

@test "from-by-tag-s1" {
  fromreftest k8s.gcr.io/pause:0.8.0
}

@test "from-by-digest-s2" {
  fromreftest alpine@sha256:e9cec9aec697d8b9d450edd32860ecd363f2f3174c8338beb5f809422d182c63
}

@test "from-by-tag-s2" {
  fromreftest alpine:2.6
}

@test "from-by-repo-only-s2" {
  fromreftest alpine
}
