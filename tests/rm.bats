#!/usr/bin/env bats

load helpers

@test "remove multiple containers errors" {
  run buildah --debug=false rm mycontainer1 mycontainer2 mycontainer3
  [ "${lines[0]}" == "error removing container \"mycontainer1\": error reading build container: container not known" ]
  [ "${lines[1]}" == "error removing container \"mycontainer2\": error reading build container: container not known" ]
  [ "${lines[2]}" == "error removing container \"mycontainer3\": error reading build container: container not known" ]
  [ $(wc -l <<< "$output") -eq 3 ]
  [ "${status}" -eq 1 ]
}
