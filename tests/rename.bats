#!/usr/bin/env bats

load helpers

@test "rename" {
  new_name=test-container
  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  old_name=$(buildah containers --format "{{.ContainerName}}")
  buildah rename ${cid} ${new_name}

  run buildah containers --format "{{.ContainerName}}" 
  [ "$status" -eq 0 ]
  [[ "$output" =~ "test-container" ]]

  run buildah --debug=false containers -f name=${old_name}
  [ "$status" -eq 0 ]
  [[ "$output" =~ "" ]]

  buildah rm ${new_name}
  [ "$status" -eq 0 ]
}
