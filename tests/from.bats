#!/usr/bin/env bats

load helpers

@test "commit-to-from-elsewhere" {
  elsewhere=${TESTDIR}/elsewhere-img
  mkdir -p ${elsewhere}

  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json scratch)
  buildah commit --signature-policy ${TESTSDIR}/policy.json $cid dir:${elsewhere}
  buildah rm $cid

  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json dir:${elsewhere})
  buildah rm $cid
  [ "$cid" = elsewhere-img-working-container ]

  cid=$(buildah from --pull-always --signature-policy ${TESTSDIR}/policy.json dir:${elsewhere})
  buildah rm $cid
  [ "$cid" = `basename ${elsewhere}`-working-container ]

  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json scratch)
  buildah commit --signature-policy ${TESTSDIR}/policy.json $cid dir:${elsewhere}
  buildah rm $cid

  cid=$(buildah from --pull --signature-policy ${TESTSDIR}/policy.json dir:${elsewhere})
  buildah rm $cid
  [ "$cid" = elsewhere-img-working-container ]

  cid=$(buildah from --pull-always --signature-policy ${TESTSDIR}/policy.json dir:${elsewhere})
  buildah rm $cid
  [ "$cid" = `basename ${elsewhere}`-working-container ]
}

@test "from-authenticate-cert" {

  mkdir -p ${TESTDIR}/auth
  # Create certifcate via openssl
  openssl req -newkey rsa:4096 -nodes -sha256 -keyout ${TESTDIR}/auth/domain.key -x509 -days 2 -out ${TESTDIR}/auth/domain.crt -subj "/C=US/ST=Foo/L=Bar/O=Red Hat, Inc./CN=localhost"
  # Skopeo and buildah both require *.cert file
  cp ${TESTDIR}/auth/domain.crt ${TESTDIR}/auth/domain.cert

  # Create a private registry that uses certificate and creds file
#  docker run -d -p 5000:5000 --name registry -v ${TESTDIR}/auth:${TESTDIR}/auth:Z -e REGISTRY_HTTP_TLS_CERTIFICATE=${TESTDIR}/auth/domain.crt -e REGISTRY_HTTP_TLS_KEY=${TESTDIR}/auth/domain.key registry:2

  # When more buildah auth is in place convert the below.
#  docker pull alpine
#  docker tag alpine localhost:5000/my-alpine
#  docker push localhost:5000/my-alpine

#  ctrid=$(buildah from localhost:5000/my-alpine --cert-dir ${TESTDIR}/auth)
#  buildah rm $ctrid
#  buildah rmi -f $(buildah --debug=false images -q)

  # This should work
#  ctrid=$(buildah from localhost:5000/my-alpine --cert-dir ${TESTDIR}/auth  --tls-verify true)

  rm -rf ${TESTDIR}/auth

  # This should fail
  run ctrid=$(buildah from localhost:5000/my-alpine --cert-dir ${TESTDIR}/auth  --tls-verify true)
  [ "$status" -ne 0 ]

  # Clean up
#  docker rm -f $(docker ps --all -q)
#  docker rmi -f localhost:5000/my-alpine
#  docker rmi -f $(docker images -q)
#  buildah rm $ctrid
#  buildah rmi -f $(buildah --debug=false images -q)
}

@test "from-authenticate-cert-and-creds" {
  mkdir -p  ${TESTDIR}/auth
  # Create creds and store in ${TESTDIR}/auth/htpasswd
#  docker run --entrypoint htpasswd registry:2 -Bbn testuser testpassword > ${TESTDIR}/auth/htpasswd
  # Create certifcate via openssl
  openssl req -newkey rsa:4096 -nodes -sha256 -keyout ${TESTDIR}/auth/domain.key -x509 -days 2 -out ${TESTDIR}/auth/domain.crt -subj "/C=US/ST=Foo/L=Bar/O=Red Hat, Inc./CN=localhost"
  # Skopeo and buildah both require *.cert file
  cp ${TESTDIR}/auth/domain.crt ${TESTDIR}/auth/domain.cert

  # Create a private registry that uses certificate and creds file
#  docker run -d -p 5000:5000 --name registry -v ${TESTDIR}/auth:${TESTDIR}/auth:Z -e "REGISTRY_AUTH=htpasswd" -e "REGISTRY_AUTH_HTPASSWD_REALM=Registry Realm" -e REGISTRY_AUTH_HTPASSWD_PATH=${TESTDIR}/auth/htpasswd -e REGISTRY_HTTP_TLS_CERTIFICATE=${TESTDIR}/auth/domain.crt -e REGISTRY_HTTP_TLS_KEY=${TESTDIR}/auth/domain.key registry:2

  # When more buildah auth is in place convert the below.
#  docker pull alpine
#  docker login localhost:5000 --username testuser --password testpassword
#  docker tag alpine localhost:5000/my-alpine
#  docker push localhost:5000/my-alpine

#  ctrid=$(buildah from localhost:5000/my-alpine --cert-dir ${TESTDIR}/auth)
#  buildah rm $ctrid
#  buildah rmi -f $(buildah --debug=false images -q)

#  docker logout localhost:5000

  # This should fail
  run ctrid=$(buildah from localhost:5000/my-alpine --cert-dir ${TESTDIR}/auth  --tls-verify true)
  [ "$status" -ne 0 ]

  # This should work
#  ctrid=$(buildah from localhost:5000/my-alpine --cert-dir ${TESTDIR}/auth  --tls-verify true --creds=testuser:testpassword)

  # Clean up
  rm -rf ${TESTDIR}/auth
#  docker rm -f $(docker ps --all -q)
#  docker rmi -f localhost:5000/my-alpine
#  docker rmi -f $(docker images -q)
#  buildah rm $ctrid
#  buildah rmi -f $(buildah --debug=false images -q)
}

@test "from-tagged-image" {
  # Github #396: Make sure the container name starts with the correct image even when it's tagged.
  cid=$(buildah from --pull=false --signature-policy ${TESTSDIR}/policy.json scratch)
  buildah commit --signature-policy ${TESTSDIR}/policy.json "$cid" scratch2
  buildah rm $cid
  buildah tag scratch2 scratch3
  cid=$(buildah from --signature-policy ${TESTSDIR}/policy.json scratch3)
  [ "$cid" == scratch3-working-container ]
  buildah rm ${cid}
  buildah rmi scratch2 scratch3

  # Github https://github.com/projectatomic/buildah/issues/396#issuecomment-360949396
  cid=$(buildah from --pull=true --signature-policy ${TESTSDIR}/policy.json alpine)
  buildah rm $cid
  buildah tag alpine alpine2
  cid=$(buildah from --signature-policy ${TESTSDIR}/policy.json docker.io/alpine2)
  [ "$cid" == alpine2-working-container ]
  buildah rm ${cid}
  buildah rmi alpine alpine2
}

@test "from cpu-period test" {
  cid=$(buildah from --cpu-period=5000 --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  run buildah run $cid cat /sys/fs/cgroup/cpu/cpu.cfs_period_us
  echo $output
  [ "$status" -eq 0 ]
  [[ "$output" =~ "5000" ]]
  buildah rm $cid
}

@test "from cpu-quota test" {
  cid=$(buildah from --cpu-quota=5000 --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  run buildah run $cid cat /sys/fs/cgroup/cpu/cpu.cfs_quota_us
  echo "$output"
  [ "$status" -eq 0 ]
  [[ "$output" =~ 5000 ]]
  buildah rm $cid
}

@test "from cpu-shares test" {
  cid=$(buildah from --cpu-shares=2 --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  run buildah run $cid cat /sys/fs/cgroup/cpu/cpu.shares
  echo "$output"
  [ "$status" -eq 0 ]
  [[ "$output" =~ 2 ]]
  buildah rm $cid
}

@test "from cpuset-cpus test" {
  cid=$(buildah from --cpuset-cpus=0 --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  run buildah run $cid cat /sys/fs/cgroup/cpuset/cpuset.cpus
  echo "$output"
  [ "$status" -eq 0 ]
  [[ "$output" =~ 0 ]]
  buildah rm $cid
}

@test "from cpuset-mems test" {
  cid=$(buildah from --cpuset-mems=0 --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  run buildah run $cid cat /sys/fs/cgroup/cpuset/cpuset.mems
  echo "$output"
  [ "$status" -eq 0 ]
  [[ "$output" =~ 0 ]]
  buildah rm $cid
}

@test "from memory test" {
  cid=$(buildah from --memory=40m --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  run buildah run $cid cat /sys/fs/cgroup/memory/memory.limit_in_bytes
  echo $output
  [ "$status" -eq 0 ]
  [[ "$output" =~ 41943040 ]]
  buildah rm $cid
}

@test "from volume test" {
  cid=$(buildah from --volume=${TESTDIR}:/myvol --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  run buildah run $cid -- cat /proc/mounts
  echo $output
  [ "$status" -eq 0 ]
  [[ "$output" =~ /myvol ]]
  buildah rm $cid
}

@test "from volume ro test" {
  cid=$(buildah from --volume=${TESTDIR}:/myvol:ro --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  run buildah run $cid -- cat /proc/mounts
  echo $output
  [ "$status" -eq 0 ]
  [[ "$output" =~ /myvol ]]
  buildah rm $cid
}

@test "from shm-size test" {
  cid=$(buildah from --shm-size=80m --pull --signature-policy ${TESTSDIR}/policy.json alpine)
  run buildah run $cid -- df -h
  echo $output
  [ "$status" -eq 0 ]
  [[ "$output" =~ 80 ]]
  buildah rm $cid
}
