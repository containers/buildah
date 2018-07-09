#!/usr/bin/env bats

load helpers

@test "rename" {
  new_name=test-container
  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  buildah rename ${cid} ${new_name}
  run buildah --debug=false containers -f name=${new_name}
  [ "$status" -eq 0 ]
  buildah rm ${new_name}
  [ "$status" -eq 0 ]
}
