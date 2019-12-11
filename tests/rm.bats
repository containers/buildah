#!/usr/bin/env bats

load helpers

@test "rm-flags-order-verification" {
  run_buildah 1 rm cnt1 -a
  check_options_flag_err "-a"

  run_buildah 1 rm cnt1 --all cnt2
  check_options_flag_err "--all"
}

@test "remove multiple containers errors" {
  run_buildah 1 rm mycontainer1 mycontainer2 mycontainer3
  expect_output --from="${lines[0]}" "error removing container \"mycontainer1\": error reading build container: container not known" "output line 1"
  expect_output --from="${lines[1]}" "error removing container \"mycontainer2\": error reading build container: container not known" "output line 2"
  expect_output --from="${lines[2]}" "error removing container \"mycontainer3\": error reading build container: container not known" "output line 3"
  expect_line_count 3
}

@test "remove one container" {
  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
  cid=$output
  run_buildah rm "$cid"
}

@test "remove multiple containers" {
  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json alpine
  cid2=$output
  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json busybox
  cid3=$output
  run_buildah rm "$cid2" "$cid3"
}

@test "remove all containers" {
  run_buildah from --signature-policy ${TESTSDIR}/policy.json scratch
  cid1=$output
  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json alpine
  cid2=$output
  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json busybox
  cid3=$output
  run_buildah rm -a
}

@test "use conflicting commands to remove containers" {
  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
  cid=$output
  run_buildah 1 rm -a "$cid"
  expect_output --substring "when using the --all switch, you may not pass any containers names or IDs"
}
