#!/usr/bin/env bats

load helpers

@test "commit-to-from-elsewhere" {
  elsewhere=${TESTDIR}/elsewhere-img
  mkdir -p ${elsewhere}

  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json scratch)
  buildah commit --signature-policy ${TESTSDIR}/policy.json $cid dir:${elsewhere}
  buildah rm $cid

  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json dir:${elsewhere})
  buildah rm $cid
  buildah rmi ${elsewhere}
  [ "$cid" = elsewhere-img-working-container ]

  cid=$(buildah from --pull-always --signature-policy ${TESTSDIR}/policy.json dir:${elsewhere})
  buildah rm $cid
  buildah rmi ${elsewhere}
  [ "$cid" = `basename ${elsewhere}`-working-container ]

  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json scratch)
  buildah commit --signature-policy ${TESTSDIR}/policy.json $cid dir:${elsewhere}
  buildah rm $cid

  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json dir:${elsewhere})
  buildah rm $cid
  buildah rmi ${elsewhere}
  [ "$cid" = elsewhere-img-working-container ]

  cid=$(buildah from --pull-always --signature-policy ${TESTSDIR}/policy.json dir:${elsewhere})
  buildah rm $cid
  buildah rmi ${elsewhere}
  [ "$cid" = `basename ${elsewhere}`-working-container ]
}
