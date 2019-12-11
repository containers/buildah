#!/usr/bin/env bats

load helpers

@test "push-flags-order-verification" {
  run_buildah 1 push img1 dest1 -q
  check_options_flag_err "-q"

  run_buildah 1 push img1 --tls-verify dest1
  check_options_flag_err "--tls-verify"

  run_buildah 1 push img1 dest1 arg3 --creds user1:pass1
  check_options_flag_err "--creds"

  run_buildah 1 push img1 --creds=user1:pass1 dest1
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

  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
  cid=$output
  run_buildah images -q
  imageid=$output
  run_buildah push --signature-policy ${TESTSDIR}/policy.json $imageid dir:$mytmpdir
}

@test "push with imageid and digest file" {
  mytmpdir=${TESTDIR}/my-dir
  mkdir -p $mytmpdir

  run_buildah from --quiet --pull=false --signature-policy ${TESTSDIR}/policy.json alpine
  cid=$output
  run_buildah images -q
  imageid=$output
  run_buildah push --digestfile=${TESTDIR}/digest.txt --signature-policy ${TESTSDIR}/policy.json $imageid dir:$mytmpdir
  cat ${TESTDIR}/digest.txt
  test -s ${TESTDIR}/digest.txt
}

@test "push without destination" {
  run_buildah pull --signature-policy ${TESTSDIR}/policy.json busybox
  run_buildah 1 push --signature-policy ${TESTSDIR}/policy.json busybox
  expect_output --substring "docker://busybox"
}

@test "push should fail with nonexist authfile" {
  run_buildah from --quiet --pull --signature-policy ${TESTSDIR}/policy.json alpine
  cid=$output
  run_buildah images -q
  imageid=$output
  run_buildah 1 push --signature-policy ${TESTSDIR}/policy.json --authfile /tmp/nonexsit $imageid dir:${TESTDIR}/my-tmp-dir
}

@test "push-denied-by-registry-sources" {
  export BUILD_REGISTRY_SOURCES='{"blockedRegistries": ["registry.example.com"]}'

  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json --quiet busybox
  cid=$output
  run_buildah 1 commit --signature-policy ${TESTSDIR}/policy.json ${cid} docker://registry.example.com/busierbox
  expect_output --substring 'commit to registry at "registry.example.com" denied by policy: it is in the blocked registries list'

  run_buildah pull --signature-policy ${TESTSDIR}/policy.json --quiet busybox
  run_buildah 1 push --signature-policy ${TESTSDIR}/policy.json busybox docker://registry.example.com/evenbusierbox
  expect_output --substring 'push to registry at "registry.example.com" denied by policy: it is in the blocked registries list'

  export BUILD_REGISTRY_SOURCES='{"allowedRegistries": ["some-other-registry.example.com"]}'

  run_buildah from --quiet --signature-policy ${TESTSDIR}/policy.json --quiet busybox
  cid=$output
  run_buildah 1 commit --signature-policy ${TESTSDIR}/policy.json ${cid} docker://registry.example.com/busierbox
  expect_output --substring 'commit to registry at "registry.example.com" denied by policy: not in allowed registries list'

  run_buildah pull --signature-policy ${TESTSDIR}/policy.json --quiet busybox
  run_buildah 1 push --signature-policy ${TESTSDIR}/policy.json busybox docker://registry.example.com/evenbusierbox
  expect_output --substring 'push to registry at "registry.example.com" denied by policy: not in allowed registries list'
}


@test "buildah push image to containers-storage" {
  run_buildah pull --signature-policy ${TESTSDIR}/policy.json busybox
  run_buildah push --signature-policy ${TESTSDIR}/policy.json busybox containers-storage:newimage:latest
  run_buildah images
  expect_output --substring "newimage"
}

@test "buildah push image to docker-archive and oci-archive" {
  run_buildah pull --signature-policy ${TESTSDIR}/policy.json busybox
  for dest in docker-archive oci-archive; do
    mkdir ${TESTDIR}/tmp
    run_buildah push --signature-policy ${TESTSDIR}/policy.json busybox $dest:${TESTDIR}/tmp/busybox.tar:latest
    ls ${TESTDIR}/tmp/busybox.tar
    rm -rf ${TESTDIR}/tmp
  done
}

@test "buildah push image to docker and docker registry" {
  run which docker
  if [[ $status -ne 0 ]]; then
    skip "docker is not installed"
  fi

  run_buildah pull --signature-policy ${TESTSDIR}/policy.json busybox
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
