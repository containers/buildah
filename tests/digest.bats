#!/usr/bin/env bats

load helpers

fromreftest() {
  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json $1)
  pushdir=${TESTDIR}/fromreftest
  mkdir -p ${pushdir}/{1,2,3}
  buildah push --signature-policy ${TESTSDIR}/policy.json $1 dir:${pushdir}/1
  buildah commit --signature-policy ${TESTSDIR}/policy.json $cid new-image
  buildah push --signature-policy ${TESTSDIR}/policy.json new-image dir:${pushdir}/2
  buildah rmi new-image
  buildah commit --signature-policy ${TESTSDIR}/policy.json $cid dir:${pushdir}/3
  buildah rm $cid
  rm -fr ${pushdir}
}

@test "from-by-digest-s1" {
  fromreftest kubernetes/pause@sha256:f8cd50c5a287dd8c5f226cf69c60c737d34ed43726c14b8a746d9de2d23eda2b
}

@test "from-by-digest-s1-a-discarded-layer" {
  fromreftest docker/whalesay@sha256:178598e51a26abbc958b8a2e48825c90bc22e641de3d31e18aaf55f3258ba93b
}

@test "from-by-tag-s1" {
  fromreftest kubernetes/pause:go
}

@test "from-by-repo-only-s1" {
  fromreftest kubernetes/pause
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
