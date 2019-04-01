#!/usr/bin/env bats

load helpers

@test "rename" {
  new_name=test-container
  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  old_name=$(buildah containers --format "{{.ContainerName}}")
  buildah rename ${cid} ${new_name}

  run_buildah containers --format "{{.ContainerName}}"
  is "$output" ".*test-container" "buildah containers"

  run_buildah --debug=false containers --quiet -f name=${old_name}
  is "$output" "" "old_name no longer in buildah containers"

  buildah rm ${new_name}
  [ "$status" -eq 0 ]
}

@test "rename same name as current name" {
  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  run_buildah 1 --debug=false rename ${cid} ${cid}
  is "$output" 'renaming a container with the same name as its current name' "output of buildah rename"

  buildah rm $cid
  buildah rmi -f alpine
}

@test "rename same name as other container name" {
  cid1=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  cid2=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json busybox)
  run_buildah 1 --debug=false rename ${cid1} ${cid2}
  is "$output" ".* already in use by " "output of buildah rename"

  buildah rm $cid1 $cid2
  buildah rmi -f alpine busybox
}
