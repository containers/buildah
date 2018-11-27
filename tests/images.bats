#!/usr/bin/env bats

load helpers

@test "images-flags-order-verification" {
  run buildah images --all
  [ $status -eq 0 ]

  run buildah images img1 -n
  check_options_flag_err "-n"

  run buildah images img1 --filter="service=redis" img2
  check_options_flag_err "--filter=service=redis"

  run buildah images img1 img2 img3 -q 
  check_options_flag_err "-q"
}

@test "images" {
  cid1=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  cid2=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json busybox)
  run buildah --debug=false images
  [ $(wc -l <<< "$output") -eq 3 ]
  [ "${status}" -eq 0 ]
  buildah rm -a
  buildah rmi -a -f
}

@test "images all test" {
  buildah bud --signature-policy ${TESTSDIR}/policy.json --layers -t test ${TESTSDIR}/bud/use-layers
  run buildah --debug=false images
  [ $(wc -l <<< "$output") -eq 3 ]
  [ "${status}" -eq 0 ]
  run buildah --debug=false images -a
  [ $(wc -l <<< "$output") -eq 8 ]

  # create a no name image which should show up when doing buildah images without the --all flag
  buildah bud --signature-policy ${TESTSDIR}/policy.json ${TESTSDIR}/bud/use-layers
  run buildah --debug=false images
  [ $(wc -l <<< "$output") -eq 4 ]

  buildah rmi -a -f
}

@test "images filter test" {
  cid1=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json kubernetes/pause)
  cid2=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json busybox)
  run buildah --debug=false images --noheading --filter since=kubernetes/pause
  echo "$output"
  [ $(wc -l <<< "$output") -eq 1 ]
  [ "${status}" -eq 0 ]
  buildah rm -a
  buildah rmi -a -f
}

@test "images format test" {
  cid1=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  cid2=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json busybox)
  run buildah --debug=false images --format "{{.Name}}"
  [ $(wc -l <<< "$output") -eq 2 ]
  [ "${status}" -eq 0 ]
  buildah rm -a
  buildah rmi -a -f
}

@test "images noheading test" {
  cid1=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  cid2=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json busybox)
  run buildah --debug=false images --noheading
  [ $(wc -l <<< "$output") -eq 2 ]
  [ "${status}" -eq 0 ]
  buildah rm -a
  buildah rmi -a -f
}

@test "images quiet test" {
  cid1=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  cid2=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json busybox)
  run buildah --debug=false images --quiet
  [ $(wc -l <<< "$output") -eq 2 ]
  [ "${status}" -eq 0 ]
  buildah rm -a
  buildah rmi -a -f
}

@test "images json test" {
  cid1=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  cid2=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json busybox)
  run buildah --debug=false images --json
  [ $(wc -l <<< "$output") -eq 12 ]
  [ "${status}" -eq 0 ]
 
  run buildah --debug=false images --json alpine
  [ $(wc -l <<< "$output") -eq 6 ]
  [ "${status}" -eq 0 ]
  buildah rm -a
  buildah rmi -a -f
}

@test "specify an existing image" {
  cid1=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  cid2=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json busybox)
  run buildah --debug=false images alpine
  [ $(wc -l <<< "$output") -eq 2 ]
  [ "${status}" -eq 0 ]
  buildah rm -a
  buildah rmi -a -f
}

@test "specify a nonexistent image" {
  run buildah --debug=false images alpine
  [ "${lines[0]}" == "No such image alpine" ]
  [ $(wc -l <<< "$output") -eq 1 ]
  [ "${status}" -eq 1 ]
}

@test "Test dangling images" {
  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json scratch)
  buildah commit --signature-policy ${TESTSDIR}/policy.json $cid test
  buildah commit --signature-policy ${TESTSDIR}/policy.json $cid test
  run buildah --debug=false images 
  [ $(wc -l <<< "$output") -eq 3 ]
  [ "${status}" -eq 0 ]
  run buildah --debug=false images --filter dangling=true
  [[ $output =~ " <none> " ]]
  [ $(wc -l <<< "$output") -eq 2 ]
  [ "${status}" -eq 0 ]
  run buildah --debug=false images --filter dangling=false
  [[ $output =~ " latest " ]]
  [ $(wc -l <<< "$output") -eq 2 ]
  [ "${status}" -eq 0 ]
  buildah rm -a
  buildah rmi -a -f
}
