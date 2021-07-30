#!/usr/bin/env bats

load helpers

@test "selinux test" {
  if ! which selinuxenabled > /dev/null 2> /dev/null ; then
    skip 'selinuxenabled command not found in $PATH'
  elif ! selinuxenabled ; then
    skip "selinux is disabled"
  fi

  image=alpine
  _prefetch $image

  # Create a container and read its context as a baseline.
  run_buildah from --quiet --quiet --signature-policy ${TESTSDIR}/policy.json $image
  cid=$output
  run_buildah run $cid sh -c 'tr \\0 \\n < /proc/self/attr/current'
  [ "$output" != "" ]
  firstlabel="$output"

  # Ensure that we label the same container consistently across multiple "run" instructions.
  run_buildah run $cid sh -c 'tr \\0 \\n < /proc/self/attr/current'
  expect_output "$firstlabel" "label of second container == first"

  # Ensure that different containers get different labels.
  run_buildah from --quiet --quiet --signature-policy ${TESTSDIR}/policy.json $image
  cid1=$output
  run_buildah run $cid1 sh -c 'tr \\0 \\n < /proc/self/attr/current'
  assert "$output" != "$firstlabel" \
         "Second container has the same label as first (both '$output')"
}

@test "selinux spc" {
  if ! which selinuxenabled > /dev/null 2> /dev/null ; then
    skip "No selinuxenabled"
  elif ! selinuxenabled ; then
    skip "selinux is disabled"
  fi

  image=alpine
  _prefetch $image

  # Create a container and read its context as a baseline.
  run_buildah from --quiet --security-opt label=disable --quiet --signature-policy ${TESTSDIR}/policy.json $image
  cid=$output
  run_buildah run $cid sh -c 'tr \\0 \\n < /proc/self/attr/current'
  context=$output

  # Role and Type should always be constant. (We don't check user)
  role=$(awk -F: '{print $2}' <<<$context)
  expect_output --from="$role" "system_r" "SELinux role"

  type=$(awk -F: '{print $3}' <<<$context)
  expect_output --from="$type" "spc_t" "SELinux type"

  # Range should match that of the invoking process
  my_range=$(id -Z |awk -F: '{print $4 ":" $5}')
  container_range=$(awk -F: '{print $4 ":" $5}' <<<$context)
  expect_output --from="$container_range" "$my_range" "SELinux range: container matches process"
}

@test "selinux specific level" {
  if ! which selinuxenabled > /dev/null 2> /dev/null ; then
    skip "No selinuxenabled"
  elif ! selinuxenabled ; then
    skip "selinux is disabled"
  fi

  image=alpine
  _prefetch $image

  firstlabel="system_u:system_r:container_t:s0:c1,c2"
  # Create a container and read its context as a baseline.
  run_buildah from --quiet --security-opt label="level:s0:c1,c2" --quiet --signature-policy ${TESTSDIR}/policy.json $image
  cid=$output

  # Inspect image
  run_buildah inspect  --format '{{.ProcessLabel}}' $cid
  expect_output "$firstlabel"

  # Check actual running context
  run_buildah run $cid sh -c 'tr \\0 \\n < /proc/self/attr/current'
  expect_output "$firstlabel" "running container context"
}
