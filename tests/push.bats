#!/usr/bin/env bats

load helpers

@test "push-flags-order-verification" {
  run_buildah 125 push img1 dest1 -q
  check_options_flag_err "-q"

  run_buildah 125 push img1 --tls-verify dest1
  check_options_flag_err "--tls-verify"

  run_buildah 125 push img1 dest1 arg3 --creds user1:pass1
  check_options_flag_err "--creds"

  run_buildah 125 push img1 --creds=user1:pass1 dest1
  check_options_flag_err "--creds=user1:pass1"
}

@test "push" {
  touch ${TESTDIR}/reference-time-file
  for source in scratch scratch-image; do
    run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json ${source}
    cid=$output
    for format in "" docker oci ; do
      mkdir -p ${TESTDIR}/committed${format:+.${format}}
      # Force no compression to generate what we push.
      run_buildah commit -D ${format:+--format ${format}} --reference-time ${TESTDIR}/reference-time-file --signature-policy ${TESTSDIR}/policy.json "$cid" scratch-image${format:+-${format}}
      run_buildah commit -D ${format:+--format ${format}} --reference-time ${TESTDIR}/reference-time-file --signature-policy ${TESTSDIR}/policy.json "$cid" dir:${TESTDIR}/committed${format:+.${format}}
      mkdir -p ${TESTDIR}/pushed${format:+.${format}}
      run_buildah push -D --signature-policy ${TESTSDIR}/policy.json scratch-image${format:+-${format}} dir:${TESTDIR}/pushed${format:+.${format}}
      # Re-encode the manifest to lose variations due to different encoders or definitions of structures.
      imgtype -expected-manifest-type "*" -rebuild-manifest -show-manifest dir:${TESTDIR}/committed${format:+.${format}} > ${TESTDIR}/manifest.committed${format:+.${format}}
      imgtype -expected-manifest-type "*" -rebuild-manifest -show-manifest dir:${TESTDIR}/pushed${format:+.${format}} > ${TESTDIR}/manifest.pushed${format:+.${format}}
      diff -u ${TESTDIR}/manifest.committed${format:+.${format}} ${TESTDIR}/manifest.pushed${format:+.${format}}
    done
    run_buildah rm "$cid"
  done
}

@test "push with manifest type conversion" {
  mytmpdir=${TESTDIR}/my-dir
  mkdir -p $mytmpdir

  _prefetch alpine
  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
  cid=$output
  run_buildah push --signature-policy ${TESTSDIR}/policy.json --format oci alpine dir:$mytmpdir
  run cat $mytmpdir/manifest.json
  expect_output --substring "application/vnd.oci.image.config.v1\\+json"

  run_buildah push --signature-policy ${TESTSDIR}/policy.json --format v2s2 alpine dir:$mytmpdir
  run cat $mytmpdir/manifest.json
  expect_output --substring "application/vnd.docker.distribution.manifest.v2\\+json"
}

@test "push with imageid" {
  mytmpdir=${TESTDIR}/my-dir
  mkdir -p $mytmpdir

  _prefetch alpine
  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
  cid=$output
  run_buildah images -q
  imageid=$output
  run_buildah push --signature-policy ${TESTSDIR}/policy.json $imageid dir:$mytmpdir
}

@test "push with imageid and digest file" {
  mytmpdir=${TESTDIR}/my-dir
  mkdir -p $mytmpdir

  _prefetch alpine
  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
  cid=$output
  run_buildah images -q
  imageid=$output
  run_buildah push --digestfile=${TESTDIR}/digest.txt --signature-policy ${TESTSDIR}/policy.json $imageid dir:$mytmpdir
  cat ${TESTDIR}/digest.txt
  test -s ${TESTDIR}/digest.txt
}

@test "push without destination" {
  _prefetch busybox
  run_buildah pull --signature-policy ${TESTSDIR}/policy.json busybox
  run_buildah 125 push --signature-policy ${TESTSDIR}/policy.json busybox
  expect_output --substring "busybox"
}

@test "push should fail with nonexistent authfile" {
  _prefetch alpine
  run_buildah from --quiet --pull --signature-policy ${TESTSDIR}/policy.json alpine
  cid=$output
  run_buildah images -q
  imageid=$output
  run_buildah 125 push --signature-policy ${TESTSDIR}/policy.json --authfile /tmp/nonexistent $imageid dir:${TESTDIR}/my-tmp-dir
}

@test "pull with nonexistent REGISTRY_AUTH_FILE: succeeds" {
  # This field should be ignored
  export REGISTRY_AUTH_FILE=/tmp/nonexistent
  _prefetch alpine
  run_buildah from --quiet --pull --signature-policy ${TESTSDIR}/policy.json alpine
  cid=$output
  run_buildah images -q
  imageid=$output
  run_buildah push --signature-policy ${TESTSDIR}/policy.json $imageid dir:${TESTDIR}/my-tmp-dir
}

@test "push-denied-by-registry-sources" {
  _prefetch busybox

  export BUILD_REGISTRY_SOURCES='{"blockedRegistries": ["registry.example.com"]}'

  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json --quiet busybox
  cid=$output
  run_buildah 125 commit --signature-policy ${TESTSDIR}/policy.json ${cid} docker://registry.example.com/busierbox
  expect_output --substring 'commit to registry at "registry.example.com" denied by policy: it is in the blocked registries list'

  run_buildah pull --signature-policy ${TESTSDIR}/policy.json --quiet busybox
  run_buildah 125 push --signature-policy ${TESTSDIR}/policy.json busybox docker://registry.example.com/evenbusierbox

  export BUILD_REGISTRY_SOURCES='{"allowedRegistries": ["some-other-registry.example.com"]}'

  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json --quiet busybox
  cid=$output
  run_buildah 125 commit --signature-policy ${TESTSDIR}/policy.json ${cid} docker://registry.example.com/busierbox
  expect_output --substring 'commit to registry at "registry.example.com" denied by policy: not in allowed registries list'

  run_buildah pull --signature-policy ${TESTSDIR}/policy.json --quiet busybox
  run_buildah 125 push --signature-policy ${TESTSDIR}/policy.json busybox docker://registry.example.com/evenbusierbox
  expect_output --substring 'registry "registry.example.com" denied by policy: not in allowed registries list'
}


@test "buildah push image to containers-storage" {
  _prefetch busybox
  run_buildah push --signature-policy ${TESTSDIR}/policy.json busybox containers-storage:newimage:latest
  run_buildah images
  expect_output --substring "newimage"
}

@test "buildah push image to docker-archive and oci-archive" {
  _prefetch busybox
  for dest in docker-archive oci-archive; do
    mkdir ${TESTDIR}/tmp
    run_buildah push --signature-policy ${TESTSDIR}/policy.json busybox $dest:${TESTDIR}/tmp/busybox.tar:latest
    ls ${TESTDIR}/tmp/busybox.tar
    rm -rf ${TESTDIR}/tmp
  done
}

@test "buildah push image to docker and docker registry" {
  skip_if_no_docker

  _prefetch busybox
  run_buildah push --signature-policy ${TESTSDIR}/policy.json busybox docker-daemon:buildah/busybox:latest
  run docker images
  expect_output --substring "buildah/busybox"
  docker rmi buildah/busybox

  run_buildah push --signature-policy ${TESTSDIR}/policy.json --tls-verify=false --creds testuser:testpassword docker.io/busybox:latest docker://localhost:5000/buildah/busybox:latest
  docker login localhost:5000 --username testuser --password testpassword
  docker pull localhost:5000/buildah/busybox:latest
  output=$(docker images)
  expect_output --substring "buildah/busybox"
  docker rmi localhost:5000/buildah/busybox:latest
  docker logout localhost:5000
}

@test "buildah oci encrypt and push local oci" {
  _prefetch busybox
  mkdir ${TESTDIR}/tmp
  openssl genrsa -out ${TESTDIR}/tmp/mykey.pem 1024
  openssl rsa -in ${TESTDIR}/tmp/mykey.pem -pubout > ${TESTDIR}/tmp/mykey.pub
  run_buildah push --signature-policy ${TESTSDIR}/policy.json --encryption-key jwe:${TESTDIR}/tmp/mykey.pub busybox oci:${TESTDIR}/tmp/busybox_enc
  imgtype  -show-manifest oci:${TESTDIR}/tmp/busybox_enc | grep "+encrypted"
  rm -rf ${TESTDIR}/tmp
}

@test "buildah oci encrypt and push registry" {
  _prefetch busybox
  mkdir ${TESTDIR}/tmp
  openssl genrsa -out ${TESTDIR}/tmp/mykey.pem 1024
  openssl rsa -in ${TESTDIR}/tmp/mykey.pem -pubout > ${TESTDIR}/tmp/mykey.pub
  run_buildah push --signature-policy ${TESTSDIR}/policy.json --tls-verify=false --creds testuser:testpassword --encryption-key jwe:${TESTDIR}/tmp/mykey.pub busybox docker://localhost:5000/buildah/busybox_encrypted:latest
  # this test, just checks the ability to push an image
  # there is no good way to test the details of the image unless with ./buildah pull, test will be in pull.bats
  rm -rf ${TESTDIR}/tmp
}

@test "buildah push to registry allowed by BUILD_REGISTRY_SOURCES" {
  _prefetch busybox
  export BUILD_REGISTRY_SOURCES='{"insecureRegistries": ["localhost:5000"]}'

  run_buildah 125 push --creds testuser:testpassword  --signature-policy ${TESTSDIR}/policy.json --tls-verify=true busybox docker://localhost:5000/buildah/busybox:latest
  expect_output --substring "can't require tls verification on an insecured registry"

  run_buildah push --creds testuser:testpassword  --signature-policy ${TESTSDIR}/policy.json busybox docker://localhost:5000/buildah/busybox:latest
}

@test "push with authfile" {
  _prefetch busybox
  mkdir ${TESTDIR}/tmp
  run_buildah login --authfile ${TESTDIR}/tmp/test.auth --username testuser --password testpassword --tls-verify=false localhost:5000
  run_buildah push --authfile ${TESTDIR}/tmp/test.auth --signature-policy ${TESTSDIR}/policy.json --tls-verify=false busybox docker://localhost:5000/buildah/busybox:latest
  expect_output --substring "Copying"
}

@test "push with --quiet" {
  mytmpdir=${TESTDIR}/my-dir
  mkdir -p $mytmpdir

  _prefetch alpine
  run_buildah push --quiet --signature-policy ${TESTSDIR}/policy.json alpine dir:$mytmpdir
  expect_output ""
}
