#!/usr/bin/env bats

load helpers

@test "pull-flags-order-verification" {
  run_buildah 1 pull image1 --tls-verify
  check_options_flag_err "--tls-verify"

  run_buildah 1 pull image1 --authfile=/tmp/somefile
  check_options_flag_err "--authfile=/tmp/somefile"

  run_buildah 1 pull image1 -q --cred bla:bla --authfile=/tmp/somefile
  check_options_flag_err "-q"
}

@test "pull-blocked" {
  run_buildah 1 --registries-conf ${TESTSDIR}/registries.conf.block pull --signature-policy ${TESTSDIR}/policy.json docker.io/alpine
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
  run_buildah 1 pull --registries-conf ${TESTSDIR}/registries.conf --signature-policy ${TESTSDIR}/policy.json fakeimage/fortest
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
  run_buildah 1 pull --all-tags --signature-policy ${TESTSDIR}/policy.json docker-archive:${TESTDIR}/alp.tar
  run rm -rf ${TESTDIR}/alp.tar
  echo "$output"
  [ "$status" -eq 0 ]
}

@test "pull-from-oci-archive" {
  run_buildah pull --signature-policy ${TESTSDIR}/policy.json alpine
  run_buildah push --signature-policy ${TESTSDIR}/policy.json docker.io/library/alpine:latest oci-archive:${TESTDIR}/alp.tar:alpine
  run_buildah rmi alpine
  run_buildah pull --signature-policy ${TESTSDIR}/policy.json oci-archive:${TESTDIR}/alp.tar
  run_buildah images --format "{{.Name}}:{{.Tag}}"
  expect_output --substring "alpine"
  run_buildah 1 pull --all-tags --signature-policy ${TESTSDIR}/policy.json oci-archive:${TESTDIR}/alp.tar
  run rm -rf ${TESTDIR}/alp.tar
  echo "$output"
  [ "$status" -eq 0 ]
}

@test "pull-from-local-directory" {
  mkdir ${TESTDIR}/buildahtest
  run_buildah pull --signature-policy ${TESTSDIR}/policy.json alpine
  run_buildah push --signature-policy ${TESTSDIR}/policy.json docker.io/library/alpine:latest dir:${TESTDIR}/buildahtest
  run_buildah rmi alpine
  run_buildah pull --signature-policy ${TESTSDIR}/policy.json dir:${TESTDIR}/buildahtest
  run_buildah images --format "{{.Name}}:{{.Tag}}"
  expect_output --substring "buildahtest"
  run_buildah 1 pull --all-tags --signature-policy ${TESTSDIR}/policy.json dir:${TESTDIR}/buildahtest
  run rm -rf ${TESTDIR}/buildahtest
  echo "$output"
  [ "$status" -eq 0 ]
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
  run_buildah 1 pull --all-tags --signature-policy ${TESTSDIR}/policy.json docker-daemon:docker.io/library/alpine:latest
  run docker rmi -f alpine:latest
  echo "$output"
  [ "$status" -eq 0 ]
}

@test "pull-from-ostree" {
  run command -v ostree
  if [[ $status -ne 0 ]]
  then
     skip "Skip the test as ostree command is not avaible"
  fi

  mkdir ${TESTDIR}/ostree-repo
  run ostree --repo=${TESTDIR}/ostree-repo init
  echo "$output"
  [ "$status" -eq 0 ]
  run_buildah pull --registries-conf ${TESTSDIR}/registries.conf --signature-policy ${TESTSDIR}/policy.json alpine
  run_buildah push --signature-policy ${TESTSDIR}/policy.json alpine ostree:alpine@${TESTDIR}/ostree-repo
  run_buildah rmi alpine
  run_buildah pull --signature-policy ${TESTSDIR}/policy.json ostree:alpine@${TESTDIR}/ostree-repo
  run_buildah images --format "{{.Name}}:{{.Tag}}"
  expect_output --substring "ostree-repo:latest"
}


@test "pull-all-tags" {
  run_buildah pull --signature-policy ${TESTSDIR}/policy.json --all-tags alpine
  expect_output --substring "alpine:latest"

  run_buildah images -q
  [ $(wc -l <<< "$output") -ge 3 ]
}

@test "pull-from-oci-directory" {
  run_buildah pull --signature-policy ${TESTSDIR}/policy.json alpine
  run_buildah push --signature-policy ${TESTSDIR}/policy.json docker.io/library/alpine:latest oci:${TESTDIR}/alpine
  run_buildah rmi alpine
  run_buildah pull --signature-policy ${TESTSDIR}/policy.json oci:${TESTDIR}/alpine
  run_buildah images --format "{{.Name}}:{{.Tag}}"
  expect_output --substring "alpine"
  run_buildah 1 pull --all-tags --signature-policy ${TESTSDIR}/policy.json oci:${TESTDIR}/alpine
  run rm -rf ${TESTDIR}/alpine
  echo "$output"
  [ "$status" -eq 0 ]
}

@test "pull-with-alltags-from-registry" {
  run_buildah pull --all-tags --registries-conf ${TESTSDIR}/registries.conf --signature-policy ${TESTSDIR}/policy.json quay.io/libpod/alpine_nginx
}
