#!/usr/bin/env bats

load helpers

@test "rm-flags-order-verification" {
  run_buildah 1 rm cnt1 -a
  check_options_flag_err "-a"

  run_buildah 1 rm cnt1 --all cnt2
  check_options_flag_err "--all"
}

@test "remove multiple containers errors" {
  run_buildah 1 --debug=false rm mycontainer1 mycontainer2 mycontainer3
  expect_output --from="${lines[0]}" "error removing container \"mycontainer1\": error reading build container: container not known" "output line 1"
  expect_output --from="${lines[1]}" "error removing container \"mycontainer2\": error reading build container: container not known" "output line 2"
  expect_output --from="${lines[2]}" "error removing container \"mycontainer3\": error reading build container: container not known" "output line 3"
  expect_line_count 3
}

@test "remove one container" {
  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  run_buildah --debug=false rm "$cid"
  run_buildah rmi alpine
}

@test "remove multiple containers" {
  cid2=$(buildah from --signature-policy ${TESTSDIR}/policy.json alpine)
  cid3=$(buildah from --signature-policy ${TESTSDIR}/policy.json busybox)
  run_buildah --debug=false rm "$cid2" "$cid3"
  run_buildah rmi alpine busybox
}

@test "remove all containers" {
  cid1=$(buildah from --signature-policy ${TESTSDIR}/policy.json scratch)
  cid2=$(buildah from --signature-policy ${TESTSDIR}/policy.json alpine)
  cid3=$(buildah from --signature-policy ${TESTSDIR}/policy.json busybox)
  run_buildah --debug=false rm -a
  run_buildah rmi --all
}

@test "use conflicting commands to remove containers" {
  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  run_buildah 1 --debug=false rm -a "$cid"
  expect_output --substring "when using the --all switch, you may not pass any containers names or IDs"
  run_buildah rm "$cid"
  run_buildah rmi alpine
}
