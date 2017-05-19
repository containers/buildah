#!/usr/bin/env bats

load helpers

@test "tag" {
  cid=$(buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json scratch)
  run buildah commit --signature-policy ${TESTSDIR}/policy.json "$cid" scratch-image
  [ "$status" -eq 0 ]
  run buildah inspect --type image tagged-image
  [ "$status" -ne 0 ]
  run buildah tag scratch-image tagged-image tagged-also-image named-image
  [ "$status" -eq 0 ]
  run buildah inspect --type image tagged-image
  [ "$status" -eq 0 ]
  run buildah inspect --type image tagged-also-image
  [ "$status" -eq 0 ]
  run buildah inspect --type image named-image
  [ "$status" -eq 0 ]
}
