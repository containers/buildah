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
  run buildah pull --signature-policy ${TESTSDIR}/policy.json busybox:glibc
  echo "$output"
  [ "$status" -eq 0 ]
  run buildah pull --signature-policy ${TESTSDIR}/policy.json busybox
  echo "$output"
  [ "$status" -eq 0 ]
  run buildah images --format "{{.Name}}:{{.Tag}}"
  echo "$output"
  [[ "$output" =~ "busybox:glibc" ]]
  [[ "$output" =~ "busybox:latest" ]]
  run buildah pull --signature-policy ${TESTSDIR}/policy.json quay.io/libpod/alpine_nginx:latest
  echo "$output"
  [ "$status" -eq 0 ]
  run buildah images --format "{{.Name}}:{{.Tag}}"
  echo "$output"
  [[ "$output" =~ "alpine_nginx:latest" ]]
  run buildah rmi quay.io/libpod/alpine_nginx:latest
  echo "$output"
  [ "$status" -eq 0 ]
  run buildah pull --signature-policy ${TESTSDIR}/policy.json quay.io/libpod/alpine_nginx
  echo "$output"
  [ "$status" -eq 0 ]
  run buildah images --format "{{.Name}}:{{.Tag}}"
  echo "$output"
  [[ "$output" =~ "alpine_nginx:latest" ]]
  run buildah pull --signature-policy ${TESTSDIR}/policy.json alpine@sha256:1072e499f3f655a032e88542330cf75b02e7bdf673278f701d7ba61629ee3ebe
  echo "$output"
  [ "$status" -eq 0 ]
  run buildah pull --signature-policy ${TESTSDIR}/policy.json fakeimage/fortest
  echo "$output"
  [ "$status" -ne 0 ]
  run buildah images --format "{{.Name}}:{{.Tag}}"
  echo "$output"
  [[ ! "$output" =~ "fakeimage/fortest" ]]
}

@test "pull-from-docerk-archive" {
  run buildah pull --signature-policy ${TESTSDIR}/policy.json alpine
  echo "$output"
  [ "$status" -eq 0 ]
  run buildah push --signature-policy ${TESTSDIR}/policy.json docker.io/library/alpine:latest docker-archive:/tmp/alp.tar:alpine:latest
  echo "$output"
  [ "$status" -eq 0 ]
  run buildah rmi alpine
  echo "$output"
  [ "$status" -eq 0 ]
  run buildah pull --signature-policy ${TESTSDIR}/policy.json docker-archive:/tmp/alp.tar
  echo "$output"
  [ "$status" -eq 0 ]
  run buildah images --format "{{.Name}}:{{.Tag}}"
  echo "$output"
  [[ "$output" =~ "alpine" ]]
  run rm -f /tmp/alp.tar
  echo "$output"
  [ "$status" -eq 0 ]
}

@test "pull-from-oci-archive" {
  run buildah pull --signature-policy ${TESTSDIR}/policy.json alpine
  echo "$output"
  [ "$status" -eq 0 ]
  run buildah push --signature-policy ${TESTSDIR}/policy.json docker.io/library/alpine:latest oci-archive:/tmp/alp.tar:alpine
  echo "$output"
  [ "$status" -eq 0 ]
  run buildah rmi alpine
  echo "$output"
  [ "$status" -eq 0 ]
  run buildah pull --signature-policy ${TESTSDIR}/policy.json oci-archive:/tmp/alp.tar
  echo "$output"
  [ "$status" -eq 0 ]
  run buildah images --format "{{.Name}}:{{.Tag}}"
  echo "$output"
  [[ "$output" =~ "alpine" ]]
  run rm -f /tmp/alp.tar
  echo "$output"
  [ "$status" -eq 0 ]
}

@test "pull-from-local-directory" {
  run mkdir /tmp/buildahtest
  run buildah pull --signature-policy ${TESTSDIR}/policy.json alpine
  echo "$output"
  [ "$status" -eq 0 ]
  run buildah push --signature-policy ${TESTSDIR}/policy.json docker.io/library/alpine:latest dir:/tmp/buildahtest
  echo "$output"
  [ "$status" -eq 0 ]
  run buildah rmi alpine
  echo "$output"
  [ "$status" -eq 0 ]
  run buildah pull --signature-policy ${TESTSDIR}/policy.json dir:/tmp/buildahtest
  echo "$output"
  [ "$status" -eq 0 ]
  run buildah images --format "{{.Name}}:{{.Tag}}"
  echo "$output"
  [[ "$output" =~ "buildahtest" ]]
  run rm -rf /tmp/buildahtest
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
  run docker pull alpine
  echo "$output"
  [ "$status" -eq 0 ]
  run buildah rmi alpine
  echo "$output"
  run buildah pull --signature-policy ${TESTSDIR}/policy.json docker-daemon:docker.io/alpine:latest
  echo "$output"
  [ "$status" -eq 0 ]
  run buildah images --format "{{.Name}}:{{.Tag}}"
  echo "$output"
  [[ "$output" =~ "alpine:latest" ]]
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

  run mkdir /tmp/ostree-repo
  run ostree --repo=/tmp/ostree-repo init
  echo "$output"
  run buildah pull --signature-policy ${TESTSDIR}/policy.json alpine
  echo "$output"
  [ "$status" -eq 0 ]
  run buildah push --signature-policy ${TESTSDIR}/policy.json alpine ostree:alpine@/tmp/ostree-repo
  echo "$output"
  [ "$status" -eq 0 ]
  run buildah rmi alpine
  echo "$output"
  run buildah pull --signature-policy ${TESTSDIR}/policy.json ostree:alpine@/tmp/ostree-repo
  echo "$output"
  [ "$status" -eq 0 ]
  run buildah images --format "{{.Name}}:{{.Tag}}"
  echo "$output"
  [[ "$output" =~ "ostree-repo:latest" ]]
  run rm -rf /tmp/ostree-repo
  echo "$output"
  [ "$status" -eq 0 ]
}
