#!/usr/bin/env bats

load helpers

@test "pull-flags-order-verification" {
  run buildah pull image1 --tls-verify
  echo "$output"
  check_options_flag_err "--tls-verify"

  run buildah pull image1 --authfile=/tmp/somefile
  echo "$output"
  check_options_flag_err "--authfile=/tmp/somefile"

  run buildah pull image1 -q --cred bla:bla --authfile=/tmp/somefile
  echo "$output"
  check_options_flag_err "-q"
}

@test "pull-blocked" {
  run buildah --registries-conf ${TESTSDIR}/registries.conf.block pull --signature-policy ${TESTSDIR}/policy.json docker.io/alpine
  echo "$output"
  [ "$status" -ne 0 ]
  [[ "$output" =~ "is blocked by configuration" ]]
  run buildah --registries-conf ${TESTSDIR}/registries.conf       pull --signature-policy ${TESTSDIR}/policy.json docker.io/alpine
  echo "$output"
  [ "$status" -eq 0 ]
}

@test "pull-from-registry" {
  run buildah pull --registries-conf ${TESTSDIR}/registries.conf --signature-policy ${TESTSDIR}/policy.json busybox:glibc
  echo "$output"
  [ "$status" -eq 0 ]
  run buildah pull --registries-conf ${TESTSDIR}/registries.conf --signature-policy ${TESTSDIR}/policy.json busybox
  echo "$output"
  [ "$status" -eq 0 ]
  run buildah images --format "{{.Name}}:{{.Tag}}"
  echo "$output"
  [ "$status" -eq 0 ]
  [[ "$output" =~ "busybox:glibc" ]]
  [[ "$output" =~ "busybox:latest" ]]
  run buildah pull --registries-conf ${TESTSDIR}/registries.conf --signature-policy ${TESTSDIR}/policy.json quay.io/libpod/alpine_nginx:latest
  echo "$output"
  [ "$status" -eq 0 ]
  run buildah images --format "{{.Name}}:{{.Tag}}"
  echo "$output"
  [ "$status" -eq 0 ]
  [[ "$output" =~ "alpine_nginx:latest" ]]
  run buildah rmi quay.io/libpod/alpine_nginx:latest
  echo "$output"
  [ "$status" -eq 0 ]
  run buildah pull --registries-conf ${TESTSDIR}/registries.conf --signature-policy ${TESTSDIR}/policy.json quay.io/libpod/alpine_nginx
  echo "$output"
  [ "$status" -eq 0 ]
  run buildah images --format "{{.Name}}:{{.Tag}}"
  echo "$output"
  [ "$status" -eq 0 ]
  [[ "$output" =~ "alpine_nginx:latest" ]]
  run buildah pull --registries-conf ${TESTSDIR}/registries.conf --signature-policy ${TESTSDIR}/policy.json alpine@sha256:1072e499f3f655a032e88542330cf75b02e7bdf673278f701d7ba61629ee3ebe
  echo "$output"
  [ "$status" -eq 0 ]
  run buildah pull --registries-conf ${TESTSDIR}/registries.conf --signature-policy ${TESTSDIR}/policy.json fakeimage/fortest
  echo "$output"
  [ "$status" -ne 0 ]
  run buildah images --format "{{.Name}}:{{.Tag}}"
  echo "$output"
  [ "$status" -eq 0 ]
  [[ ! "$output" =~ "fakeimage/fortest" ]]
}

@test "pull-from-docker-archive" {
  run buildah pull --signature-policy ${TESTSDIR}/policy.json alpine
  echo "$output"
  [ "$status" -eq 0 ]
  run buildah push --signature-policy ${TESTSDIR}/policy.json docker.io/library/alpine:latest docker-archive:${TESTDIR}/alp.tar:alpine:latest
  echo "$output"
  [ "$status" -eq 0 ]
  run buildah rmi alpine
  echo "$output"
  [ "$status" -eq 0 ]
  run buildah pull --signature-policy ${TESTSDIR}/policy.json docker-archive:${TESTDIR}/alp.tar
  echo "$output"
  [ "$status" -eq 0 ]
  run buildah images --format "{{.Name}}:{{.Tag}}"
  echo "$output"
  [ "$status" -eq 0 ]
  [[ "$output" =~ "alpine" ]]
  run buildah pull --all-tags --signature-policy ${TESTSDIR}/policy.json docker-archive:${TESTDIR}/alp.tar
  echo "$output"
  [ "$status" -ne 0 ]
  run rm -rf ${TESTDIR}/alp.tar
  echo "$output"
  [ "$status" -eq 0 ]
}

@test "pull-from-oci-archive" {
  run buildah pull --signature-policy ${TESTSDIR}/policy.json alpine
  echo "$output"
  [ "$status" -eq 0 ]
  run buildah push --signature-policy ${TESTSDIR}/policy.json docker.io/library/alpine:latest oci-archive:${TESTDIR}/alp.tar:alpine
  echo "$output"
  [ "$status" -eq 0 ]
  run buildah rmi alpine
  echo "$output"
  [ "$status" -eq 0 ]
  run buildah pull --signature-policy ${TESTSDIR}/policy.json oci-archive:${TESTDIR}/alp.tar
  echo "$output"
  [ "$status" -eq 0 ]
  run buildah images --format "{{.Name}}:{{.Tag}}"
  echo "$output"
  [ "$status" -eq 0 ]
  [[ "$output" =~ "alpine" ]]
  run buildah pull --all-tags --signature-policy ${TESTSDIR}/policy.json oci-archive:${TESTDIR}/alp.tar
  echo "$output"
  [ "$status" -ne 0 ]
  run rm -rf ${TESTDIR}/alp.tar
  echo "$output"
  [ "$status" -eq 0 ]
}

@test "pull-from-local-directory" {
  mkdir ${TESTDIR}/buildahtest
  run buildah pull --signature-policy ${TESTSDIR}/policy.json alpine
  echo "$output"
  [ "$status" -eq 0 ]
  run buildah push --signature-policy ${TESTSDIR}/policy.json docker.io/library/alpine:latest dir:${TESTDIR}/buildahtest
  echo "$output"
  [ "$status" -eq 0 ]
  run buildah rmi alpine
  echo "$output"
  [ "$status" -eq 0 ]
  run buildah pull --signature-policy ${TESTSDIR}/policy.json dir:${TESTDIR}/buildahtest
  echo "$output"
  [ "$status" -eq 0 ]
  run buildah images --format "{{.Name}}:{{.Tag}}"
  echo "$output"
  [ "$status" -eq 0 ]
  [[ "$output" =~ "buildahtest" ]]
  run buildah pull --all-tags --signature-policy ${TESTSDIR}/policy.json dir:${TESTDIR}/buildahtest
  echo "$output"
  [ "$status" -ne 0 ]
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
  run buildah pull --signature-policy ${TESTSDIR}/policy.json docker-daemon:docker.io/library/alpine:latest
  echo "$output"
  [ "$status" -eq 0 ]
  run buildah images --format "{{.Name}}:{{.Tag}}"
  echo "$output"
  [ "$status" -eq 0 ]
  [[ "$output" =~ "alpine:latest" ]]
  run buildah rmi alpine
  echo "$output"
  [ "$status" -eq 0 ]
  run buildah pull --all-tags --signature-policy ${TESTSDIR}/policy.json docker-daemon:docker.io/library/alpine:latest
  echo "$output"
  [ "$status" -ne 0 ]
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
  run buildah pull --registries-conf ${TESTSDIR}/registries.conf --signature-policy ${TESTSDIR}/policy.json alpine
  echo "$output"
  [ "$status" -eq 0 ]
  run buildah push --signature-policy ${TESTSDIR}/policy.json alpine ostree:alpine@${TESTDIR}/ostree-repo
  echo "$output"
  [ "$status" -eq 0 ]
  run buildah rmi alpine
  echo "$output"
  [ "$status" -eq 0 ]
  run buildah pull --signature-policy ${TESTSDIR}/policy.json ostree:alpine@${TESTDIR}/ostree-repo
  echo "$output"
  [ "$status" -eq 0 ]
  run buildah images --format "{{.Name}}:{{.Tag}}"
  echo "$output"
  [ "$status" -eq 0 ]
  [[ "$output" =~ "ostree-repo:latest" ]]
}


@test "pull-all-tags" {
  run buildah pull --signature-policy ${TESTSDIR}/policy.json --all-tags alpine
  echo "$output"
  [[ "$output" =~ "alpine:latest" ]]
  [ "$status" -eq 0 ]

  run buildah images -q
  echo "$output"
  [ "$status" -eq 0 ]
  [ $(wc -l <<< "$output") -ge 3 ]
}

@test "pull-from-oci-directory" {
  run buildah pull --signature-policy ${TESTSDIR}/policy.json alpine
  echo "$output"
  [ "$status" -eq 0 ]
  run buildah push --signature-policy ${TESTSDIR}/policy.json docker.io/library/alpine:latest oci:${TESTDIR}/alpine
  echo "$output"
  [ "$status" -eq 0 ]
  run buildah rmi alpine
  echo "$output"
  [ "$status" -eq 0 ]
  run buildah pull --signature-policy ${TESTSDIR}/policy.json oci:${TESTDIR}/alpine
  echo "$output"
  [ "$status" -eq 0 ]
  run buildah images --format "{{.Name}}:{{.Tag}}"
  echo "$output"
  [ "$status" -eq 0 ]
  [[ "$output" =~ "alpine" ]]
  run buildah pull --all-tags --signature-policy ${TESTSDIR}/policy.json oci:${TESTDIR}/alpine
  echo "$output"
  [ "$status" -ne 0 ]
  run rm -rf ${TESTDIR}/alpine
  echo "$output"
  [ "$status" -eq 0 ]
}

@test "pull-with-alltags-from-registry" {
  run buildah pull --all-tags --registries-conf ${TESTSDIR}/registries.conf --signature-policy ${TESTSDIR}/policy.json quay.io/libpod/alpine_nginx
  echo "$output"
  [ "$status" -eq 0 ]
}
