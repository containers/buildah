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

@test "rename same name as current name" {
  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  run buildah --debug=false rename ${cid} ${cid}
  [ "$status" -eq 1 ]
  [[ "$output" =~ "" ]]

  buildah rm $cid
  buildah rmi -f alpine
}

@test "rename same name as other container name" {
  cid1=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  cid2=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json busybox)
  run buildah --debug=false rename ${cid1} ${cid2}
  [ "$status" -eq 1 ]
  [[ "$output" =~ "" ]]

  buildah rm $cid1 $cid2
  buildah rmi -f alpine busybox
}
