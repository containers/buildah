#!/usr/bin/env bats

load helpers

@test "selinux test" {
  if ! which selinuxenabled ; then
    skip "No selinuxenabled"
   elif ! /usr/sbin/selinuxenabled; then
     skip "selinux is disabled"
  fi
  image=alpine
  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json $image)
  firstlabel=$(buildah --debug=false run $cid cat /proc/1/attr/current)
  run buildah --debug=false run $cid cat /proc/1/attr/current
  [ "$status" -eq 0 ]
  [ "$output" == $firstlabel ]

  cid1=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json $image)
  run buildah --debug=false run $cid1 cat /proc/1/attr/current
  [ "$status" -eq 0 ]
  [ "$output" != $firstlabel ]

  buildah rm $cid
  buildah rm $cid1
}

