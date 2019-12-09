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
  _prefetch alpine busybox
  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
  cid1=$output
  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json busybox
  cid2=$output
  run_buildah images
  expect_line_count 3
}

@test "images all test" {
  _prefetch alpine
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t test ${TESTSDIR}/bud/use-layers
  run_buildah images
  expect_line_count 3

  run_buildah images -a
  expect_line_count 8

  # create a no name image which should show up when doing buildah images without the --all flag
  run_buildah bud --signature-policy ${TESTSDIR}/policy.json ${TESTSDIR}/bud/use-layers
  run_buildah images
  expect_line_count 4
}

@test "images filter test" {
  _prefetch k8s.gcr.io/pause busybox
  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json k8s.gcr.io/pause
  cid1=$output
  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json busybox
  cid2=$output
  run_buildah images --noheading --filter since=k8s.gcr.io/pause
  expect_line_count 1
}

@test "images format test" {
  _prefetch alpine busybox
  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
  cid1=$output
  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json busybox
  cid2=$output
  run_buildah images --format "{{.Name}}"
  expect_line_count 2
}

@test "images noheading test" {
  _prefetch alpine busybox
  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
  cid1=$output
  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json busybox
  cid2=$output
  run_buildah images --noheading
  expect_line_count 2
}

@test "images quiet test" {
  _prefetch alpine busybox
  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
  cid1=$output
  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json busybox
  cid2=$output
  run_buildah images --quiet
  expect_line_count 2
}

@test "images no-trunc test" {
  _prefetch alpine busybox
  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
  cid1=$output
  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json busybox
  cid2=$output
  run_buildah images -q --no-trunc
  expect_line_count 2
  expect_output --substring --from="${lines[0]}" "sha256"
}

@test "images json test" {
  _prefetch alpine busybox
  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
  cid1=$output
  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json busybox
  cid2=$output
  run_buildah images --json
  expect_line_count 30

  run_buildah images --json alpine
  expect_line_count 16
}

@test "images json dup test" {
  run_buildah from --signature-policy ${TESTSDIR}/policy.json scratch
  cid=$output
  run_buildah commit --signature-policy ${TESTSDIR}/policy.json $cid test
  run_buildah tag test new-name

  run_buildah images --json
  expect_output --substring '"id": '
}

@test "images json valid" {
  run_buildah from --signature-policy ${TESTSDIR}/policy.json scratch
  cid1=$output
  run_buildah from --signature-policy ${TESTSDIR}/policy.json scratch
  cid2=$output
  run_buildah commit --signature-policy ${TESTSDIR}/policy.json $cid1 test
  run_buildah commit --signature-policy ${TESTSDIR}/policy.json $cid2 test2

  run_buildah images --json
  run python3 -m json.tool <<< "$output"
  [ "${status}" -eq 0 ]
}

@test "specify an existing image" {
  _prefetch alpine busybox
  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
  cid1=$output
  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json busybox
  cid2=$output
  run_buildah images alpine
  expect_line_count 2
}

@test "specify a nonexistent image" {
  run_buildah 1 images alpine
  expect_output --from="${lines[0]}" "No such image alpine"
  expect_line_count 1
}

@test "Test dangling images" {
  run_buildah from --signature-policy ${TESTSDIR}/policy.json scratch
  cid=$output
  run_buildah commit --signature-policy ${TESTSDIR}/policy.json $cid test
  run_buildah commit --signature-policy ${TESTSDIR}/policy.json $cid test
  run_buildah images
  expect_line_count 3

  run_buildah images --filter dangling=true
  expect_output --substring " <none> "
  expect_line_count 2

  run_buildah images --filter dangling=false
  expect_output --substring " latest "
  expect_line_count 2
}

@test "image digest test" {
  _prefetch busybox
  run_buildah pull --signature-policy ${TESTSDIR}/policy.json busybox
  run_buildah images --digests
  expect_output --substring "sha256:"
}
