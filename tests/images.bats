#!/usr/bin/env bats

load helpers

@test "images-flags-order-verification" {
  run_buildah images --all

  run_buildah 1 images img1 -n
  check_options_flag_err "-n"

  run_buildah 1 images img1 --filter="service=redis" img2
  check_options_flag_err "--filter=service=redis"

  run_buildah 1 images img1 img2 img3 -q
  check_options_flag_err "-q"
}

@test "images" {
  cid1=$(buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json alpine)
  cid2=$(buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json busybox)
  run_buildah --log-level=error images
  expect_line_count 3
  buildah rm -a
  buildah rmi -a -f
}

@test "images all test" {
  buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t test ${TESTSDIR}/bud/use-layers
  run_buildah --log-level=error images
  expect_line_count 3

  run_buildah --log-level=error images -a
  expect_line_count 8

  # create a no name image which should show up when doing buildah images without the --all flag
  buildah bud --signature-policy ${TESTSDIR}/policy.json ${TESTSDIR}/bud/use-layers
  run_buildah --log-level=error images
  expect_line_count 4

  buildah rmi -a -f
}

@test "images filter test" {
  cid1=$(buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json k8s.gcr.io/pause)
  cid2=$(buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json busybox)
  run_buildah --log-level=error images --noheading --filter since=k8s.gcr.io/pause
  expect_line_count 1
  buildah rm -a
  buildah rmi -a -f
}

@test "images format test" {
  cid1=$(buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json alpine)
  cid2=$(buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json busybox)
  run_buildah --log-level=error images --format "{{.Name}}"
  expect_line_count 2
  buildah rm -a
  buildah rmi -a -f
}

@test "images noheading test" {
  cid1=$(buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json alpine)
  cid2=$(buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json busybox)
  run_buildah --log-level=error images --noheading
  expect_line_count 2
  buildah rm -a
  buildah rmi -a -f
}

@test "images quiet test" {
  cid1=$(buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json alpine)
  cid2=$(buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json busybox)
  run_buildah --log-level=error images --quiet
  expect_line_count 2
  buildah rm -a
  buildah rmi -a -f
}

@test "images no-trunc test" {
  cid1=$(buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json alpine)
  cid2=$(buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json busybox)
  run_buildah --log-level=error images -q --no-trunc
  expect_line_count 2
  expect_output --substring --from="${lines[0]}" "sha256"
  buildah rm -a
  buildah rmi -a -f
}

@test "images json test" {
  cid1=$(buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json alpine)
  cid2=$(buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json busybox)
  run_buildah --log-level=error images --json
  expect_line_count 32

  run_buildah --log-level=error images --json alpine
  expect_line_count 17
  buildah rm -a
  buildah rmi -a -f
}

@test "images json dup test" {
  cid=$(buildah from --signature-policy ${TESTSDIR}/policy.json scratch)
  buildah commit --signature-policy ${TESTSDIR}/policy.json $cid test
  buildah tag test new-name

  run_buildah --log-level=error images --json
  [ $(grep '"id": "' <<< "$output" | wc -l) -eq 1 ]

  buildah rm -a
  buildah rmi -a -f
}

@test "images json valid" {
  cid1=$(buildah from --signature-policy ${TESTSDIR}/policy.json scratch)
  cid2=$(buildah from --signature-policy ${TESTSDIR}/policy.json scratch)
  buildah commit --signature-policy ${TESTSDIR}/policy.json $cid1 test
  buildah commit --signature-policy ${TESTSDIR}/policy.json $cid2 test2

  run_buildah --log-level=error images --json
  run python3 -m json.tool <<< "$output"
  [ "${status}" -eq 0 ]

  buildah rm -a
  buildah rmi -a -f
}

@test "specify an existing image" {
  cid1=$(buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json alpine)
  cid2=$(buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json busybox)
  run_buildah --log-level=error images alpine
  expect_line_count 2
  buildah rm -a
  buildah rmi -a -f
}

@test "specify a nonexistent image" {
  run_buildah 1 --log-level=error images alpine
  expect_output --from="${lines[0]}" "No such image alpine"
  expect_line_count 1
}

@test "Test dangling images" {
  cid=$(buildah from --signature-policy ${TESTSDIR}/policy.json scratch)
  buildah commit --signature-policy ${TESTSDIR}/policy.json $cid test
  buildah commit --signature-policy ${TESTSDIR}/policy.json $cid test
  run_buildah --log-level=error images
  expect_line_count 3

  run_buildah --log-level=error images --filter dangling=true
  expect_output --substring " <none> "
  expect_line_count 2

  run_buildah --log-level=error images --filter dangling=false
  expect_output --substring " latest "
  expect_line_count 2

  buildah rm -a
  buildah rmi -a -f
}
