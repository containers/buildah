#!/usr/bin/env bats

load helpers

@test "selinux test" {
  if ! which selinuxenabled > /dev/null 2> /dev/null ; then
    skip "No selinuxenabled"
  elif ! selinuxenabled ; then
    skip "selinux is disabled"
  fi

  image=alpine

  # Create a container and read its context as a baseline.
  cid=$(buildah --debug=false from --quiet --signature-policy ${TESTSDIR}/policy.json $image)
  run buildah --debug=false run $cid sh -c 'tr \\0 \\n < /proc/self/attr/current'
  echo "$output"
  [ "$status" -eq 0 ]
  [ "$output" != "" ]
  firstlabel="$output"

  # Ensure that we label the same container consistently across multiple "run" instructions.
  run buildah --debug=false run $cid sh -c 'tr \\0 \\n < /proc/self/attr/current'
  echo "$output"
  [ "$status" -eq 0 ]
  [ "$output" == "$firstlabel" ]

  # Ensure that different containers get different labels.
  cid1=$(buildah --debug=false from --quiet --signature-policy ${TESTSDIR}/policy.json $image)
  run buildah --debug=false run $cid1 sh -c 'tr \\0 \\n < /proc/self/attr/current'
  echo "$output"
  [ "$status" -eq 0 ]
  [ "$output" != "$firstlabel" ]
}
