#!/usr/bin/env bats

load helpers

@test "tag" {
  cid=$(buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json scratch)
  run_buildah commit --signature-policy ${TESTSDIR}/policy.json "$cid" scratch-image
  run_buildah 1 inspect --type image tagged-image
  run_buildah tag scratch-image tagged-image tagged-also-image named-image
  run_buildah inspect --type image tagged-image
  run_buildah inspect --type image tagged-also-image
  run_buildah inspect --type image named-image
}
