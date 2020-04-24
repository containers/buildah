#!/usr/bin/env bats

load helpers

@test "pull-flags-order-verification" {
  run_buildah 125 pull image1 --tls-verify
  check_options_flag_err "--tls-verify"

  run_buildah 125 pull image1 --authfile=/tmp/somefile
  check_options_flag_err "--authfile=/tmp/somefile"

  run_buildah 125 pull image1 -q --cred bla:bla --authfile=/tmp/somefile
  check_options_flag_err "-q"
}

@test "pull-blocked" {
  run_buildah 125 --registries-conf ${TESTSDIR}/registries.conf.block pull --signature-policy ${TESTSDIR}/policy.json docker.io/alpine
  expect_output --substring "is blocked by configuration"

  run_buildah --registries-conf ${TESTSDIR}/registries.conf       pull --signature-policy ${TESTSDIR}/policy.json docker.io/alpine
}

@test "pull-from-registry" {
  run_buildah pull --registries-conf ${TESTSDIR}/registries.conf --signature-policy ${TESTSDIR}/policy.json busybox:glibc
  run_buildah pull --registries-conf ${TESTSDIR}/registries.conf --signature-policy ${TESTSDIR}/policy.json busybox
  run_buildah images --format "{{.Name}}:{{.Tag}}"
  expect_output --substring "busybox:glibc"
  expect_output --substring "busybox:latest"

  run_buildah pull --registries-conf ${TESTSDIR}/registries.conf --signature-policy ${TESTSDIR}/policy.json quay.io/libpod/alpine_nginx:latest
  run_buildah images --format "{{.Name}}:{{.Tag}}"
  expect_output --substring "alpine_nginx:latest"

  run_buildah rmi quay.io/libpod/alpine_nginx:latest
  run_buildah pull --registries-conf ${TESTSDIR}/registries.conf --signature-policy ${TESTSDIR}/policy.json quay.io/libpod/alpine_nginx
  run_buildah images --format "{{.Name}}:{{.Tag}}"
  expect_output --substring "alpine_nginx:latest"

  run_buildah pull --registries-conf ${TESTSDIR}/registries.conf --signature-policy ${TESTSDIR}/policy.json alpine@sha256:1072e499f3f655a032e88542330cf75b02e7bdf673278f701d7ba61629ee3ebe
  run_buildah 125 pull --registries-conf ${TESTSDIR}/registries.conf --signature-policy ${TESTSDIR}/policy.json fakeimage/fortest
  run_buildah images --format "{{.Name}}:{{.Tag}}"
  [[ ! "$output" =~ "fakeimage/fortest" ]]
}

@test "pull-from-docker-archive" {
  run_buildah pull --signature-policy ${TESTSDIR}/policy.json alpine
  run_buildah push --signature-policy ${TESTSDIR}/policy.json docker.io/library/alpine:latest docker-archive:${TESTDIR}/alp.tar:alpine:latest
  run_buildah rmi alpine
  run_buildah pull --signature-policy ${TESTSDIR}/policy.json docker-archive:${TESTDIR}/alp.tar
  run_buildah images --format "{{.Name}}:{{.Tag}}"
  expect_output --substring "alpine"
  run_buildah 125 pull --all-tags --signature-policy ${TESTSDIR}/policy.json docker-archive:${TESTDIR}/alp.tar
  expect_output "Non-docker transport is not supported, for --all-tags pulling"
}

@test "pull-from-oci-archive" {
  run_buildah pull --signature-policy ${TESTSDIR}/policy.json alpine
  run_buildah push --signature-policy ${TESTSDIR}/policy.json docker.io/library/alpine:latest oci-archive:${TESTDIR}/alp.tar:alpine
  run_buildah rmi alpine
  run_buildah pull --signature-policy ${TESTSDIR}/policy.json oci-archive:${TESTDIR}/alp.tar
  run_buildah images --format "{{.Name}}:{{.Tag}}"
  expect_output --substring "alpine"
  run_buildah 125 pull --all-tags --signature-policy ${TESTSDIR}/policy.json oci-archive:${TESTDIR}/alp.tar
  expect_output "Non-docker transport is not supported, for --all-tags pulling"
}

@test "pull-from-local-directory" {
  mkdir ${TESTDIR}/buildahtest
  run_buildah pull --signature-policy ${TESTSDIR}/policy.json alpine
  run_buildah push --signature-policy ${TESTSDIR}/policy.json docker.io/library/alpine:latest dir:${TESTDIR}/buildahtest
  run_buildah rmi alpine
  run_buildah pull --signature-policy ${TESTSDIR}/policy.json dir:${TESTDIR}/buildahtest
  run_buildah images --format "{{.Name}}:{{.Tag}}"
  expect_output --substring "localhost${TESTDIR}/buildahtest:latest"
  run_buildah 125 pull --all-tags --signature-policy ${TESTSDIR}/policy.json dir:${TESTDIR}/buildahtest
  expect_output "Non-docker transport is not supported, for --all-tags pulling"
}

@test "pull-from-docker-deamon" {
  run systemctl status docker
  if [[ ! "$output" =~ "active (running)" ]]
  then
     skip "Skip the test as docker services is not running"
  fi

  run systemctl start docker
  echo "$output"
  [ "$status" -eq 0 ]
  run docker pull alpine
  echo "$output"
  [ "$status" -eq 0 ]
  run_buildah pull --signature-policy ${TESTSDIR}/policy.json docker-daemon:docker.io/library/alpine:latest
  run_buildah images --format "{{.Name}}:{{.Tag}}"
  expect_output --substring "alpine:latest"
  run_buildah rmi alpine
  run_buildah 125 pull --all-tags --signature-policy ${TESTSDIR}/policy.json docker-daemon:docker.io/library/alpine:latest
  expect_output --substring "Non-docker transport is not supported, for --all-tags pulling"
}

@test "pull-all-tags" {
  declare -a tags=(0.9 0.9.1 1.1 alpha beta gamma2.0 latest)

  # setup: pull alpine, and push it repeatedly to localhost using those tags
  opts="--signature-policy ${TESTSDIR}/policy.json --tls-verify=false --creds testuser:testpassword"
  run_buildah pull --quiet --signature-policy ${TESTSDIR}/policy.json alpine
  for tag in "${tags[@]}"; do
      run_buildah push $opts alpine localhost:5000/myalpine:$tag
  done

  run_buildah images -q
  expect_line_count 1 "There's only one actual image ID"
  alpine_iid=$output

  # Remove it, and confirm.
  run_buildah rmi alpine
  run_buildah images -q
  expect_output "" "After buildah rmi, there are no locally stored images"

  # Now pull with --all-tags, and confirm that we see all expected tag strings
  run_buildah pull $opts --all-tags localhost:5000/myalpine
  for tag in "${tags[@]}"; do
      expect_output --substring "Pulling localhost:5000/myalpine:$tag"
  done

  # Confirm that 'images -a' lists all of them. <Brackets> help confirm
  # that tag names are exact, e.g we don't confuse 0.9 and 0.9.1
  run_buildah images -a --format '<{{.Tag}}>'
  expect_line_count "${#tags[@]}" "number of tagged images"
  for tag in "${tags[@]}"; do
      expect_output --substring "<$tag>"
  done

  # Finally, make sure that there's actually one and exactly one image
  run_buildah images -q
  expect_output $alpine_iid "Pulled image has the same IID as original alpine"
}

@test "pull-from-oci-directory" {
  run_buildah pull --signature-policy ${TESTSDIR}/policy.json alpine
  run_buildah push --signature-policy ${TESTSDIR}/policy.json docker.io/library/alpine:latest oci:${TESTDIR}/alpine
  run_buildah rmi alpine
  run_buildah pull --signature-policy ${TESTSDIR}/policy.json oci:${TESTDIR}/alpine
  run_buildah images --format "{{.Name}}:{{.Tag}}"
  expect_output --substring "localhost${TESTDIR}/alpine:latest"
  run_buildah 125 pull --all-tags --signature-policy ${TESTSDIR}/policy.json oci:${TESTDIR}/alpine
  expect_output "Non-docker transport is not supported, for --all-tags pulling"
}

@test "pull-denied-by-registry-sources" {
  export BUILD_REGISTRY_SOURCES='{"blockedRegistries": ["docker.io"]}'

  run_buildah 125 pull --signature-policy ${TESTSDIR}/policy.json --registries-conf ${TESTSDIR}/registries.conf.hub --quiet busybox
  expect_output --substring 'pull from registry at "docker.io" denied by policy: it is in the blocked registries list'

  run_buildah 125 pull --signature-policy ${TESTSDIR}/policy.json --registries-conf ${TESTSDIR}/registries.conf.hub --quiet busybox
  expect_output --substring 'pull from registry at "docker.io" denied by policy: it is in the blocked registries list'

  export BUILD_REGISTRY_SOURCES='{"allowedRegistries": ["some-other-registry.example.com"]}'

  run_buildah 125 pull --signature-policy ${TESTSDIR}/policy.json --registries-conf ${TESTSDIR}/registries.conf.hub --quiet busybox
  expect_output --substring 'pull from registry at "docker.io" denied by policy: not in allowed registries list'

  run_buildah 125 pull --signature-policy ${TESTSDIR}/policy.json --registries-conf ${TESTSDIR}/registries.conf.hub --quiet busybox
  expect_output --substring 'pull from registry at "docker.io" denied by policy: not in allowed registries list'
}

@test "pull should fail with nonexist authfile" {
  run_buildah 125 pull --authfile /tmp/nonexist --signature-policy ${TESTSDIR}/policy.json alpine
}
