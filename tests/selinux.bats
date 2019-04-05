#!/usr/bin/env bats

load helpers

@test "selinux test" {
  if ! which selinuxenabled > /dev/null 2> /dev/null ; then
    skip 'selinuxenabled command not found in $PATH'
  elif ! selinuxenabled ; then
    skip "selinux is disabled"
  fi

  image=alpine

  # Create a container and read its context as a baseline.
  cid=$(buildah --debug=false from --quiet --signature-policy ${TESTSDIR}/policy.json $image)
  run_buildah --debug=false run $cid sh -c 'tr \\0 \\n < /proc/self/attr/current'
  [ "$output" != "" ]
  firstlabel="$output"

  # Ensure that we label the same container consistently across multiple "run" instructions.
  run_buildah --debug=false run $cid sh -c 'tr \\0 \\n < /proc/self/attr/current'
  expect_output "$firstlabel" "label of second container == first"

  # Ensure that different containers get different labels.
  cid1=$(buildah --debug=false from --quiet --signature-policy ${TESTSDIR}/policy.json $image)
  run_buildah --debug=false run $cid1 sh -c 'tr \\0 \\n < /proc/self/attr/current'
  if [ "$output" = "$firstlabel" ]; then
      die "Second container has the same label as first (both '$output')"
  fi
}

@test "selinux spc" {
  if ! which selinuxenabled > /dev/null 2> /dev/null ; then
    skip "No selinuxenabled"
  elif ! selinuxenabled ; then
    skip "selinux is disabled"
  fi

  image=alpine

  firstlabel=$(id -Z)
  # Running from installed RPM?
  if [ "$(secon --file $BUILDAH_BINARY -t)" = "bin_t" ]; then
      firstlabel="unconfined_u:system_r:spc_t:s0-s0:c0.c1023"
  fi
  # Create a container and read its context as a baseline.
  cid=$(buildah --debug=false from --security-opt label=disable --quiet --signature-policy ${TESTSDIR}/policy.json $image)
  run_buildah --debug=false run $cid sh -c 'tr \\0 \\n < /proc/self/attr/current'
  expect_output "$firstlabel" "container context matches our own"
}

@test "selinux specific level" {
  if ! which selinuxenabled > /dev/null 2> /dev/null ; then
    skip "No selinuxenabled"
  elif ! selinuxenabled ; then
    skip "selinux is disabled"
  fi

  image=alpine

  firstlabel="system_u:system_r:container_t:s0:c1,c2"
  # Create a container and read its context as a baseline.
  cid=$(buildah --debug=false from --security-opt label="level:s0:c1,c2" --quiet --signature-policy ${TESTSDIR}/policy.json $image)

  # Inspect image
  run_buildah --debug=false inspect  --format '{{.ProcessLabel}}' $cid
  expect_output "$firstlabel"

  # Check actual running context
  run_buildah --debug=false run $cid sh -c 'tr \\0 \\n < /proc/self/attr/current'
  expect_output "$firstlabel" "running container context"
}
