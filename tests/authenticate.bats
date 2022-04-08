#!/usr/bin/env bats

load helpers

@test "authenticate: login/logout" {
  start_registry testuserfoo testpassword

  run_buildah 0 login --cert-dir $REGISTRY_DIR --username testuserfoo --password testpassword localhost:$REGISTRY_PORT

  run_buildah 0 logout localhost:$REGISTRY_PORT
}

@test "authenticate: login/logout should succeed with XDG_RUNTIME_DIR unset" {
  unset XDG_RUNTIME_DIR

  start_registry testuserfoo testpassword

  run_buildah 0 login --cert-dir $REGISTRY_DIR --username testuserfoo --password testpassword localhost:$REGISTRY_PORT

  run_buildah 0 logout localhost:$REGISTRY_PORT
}

@test "authenticate: logout should fail with nonexistent authfile" {
  start_registry testuserfoo testpassword

  run_buildah 0 login --cert-dir $REGISTRY_DIR --username testuserfoo --password testpassword localhost:$REGISTRY_PORT

  run_buildah 125 logout --authfile /tmp/nonexistent localhost:$REGISTRY_PORT
  expect_output "checking authfile: stat /tmp/nonexistent: no such file or directory"

  run_buildah 0 logout localhost:$REGISTRY_PORT
}

@test "authenticate: cert and credentials" {
  _prefetch alpine

  testuser="testuser$RANDOM"
  testpassword="testpassword$RANDOM"
  start_registry "$testuser" "$testpassword"

  # Basic test: should pass
  run_buildah push --cert-dir $REGISTRY_DIR --signature-policy ${TESTSDIR}/policy.json --tls-verify=false --creds "$testuser":"$testpassword" alpine localhost:$REGISTRY_PORT/my-alpine
  expect_output --substring "Writing manifest to image destination"

  # With tls-verify=true, should fail due to self-signed cert
  run_buildah 125 push --signature-policy ${TESTSDIR}/policy.json --tls-verify=true alpine localhost:$REGISTRY_PORT/my-alpine
  expect_output --substring " x509: certificate signed by unknown authority" \
                "push with --tls-verify=true"

  # wrong credentials: should fail
  run_buildah 125 from --cert-dir $REGISTRY_DIR --signature-policy ${TESTSDIR}/policy.json --creds baduser:badpassword localhost:$REGISTRY_PORT/my-alpine
  expect_output --substring "unauthorized: authentication required"
  run_buildah 125 from --cert-dir $REGISTRY_DIR --signature-policy ${TESTSDIR}/policy.json --creds "$testuser":badpassword localhost:$REGISTRY_PORT/my-alpine
  expect_output --substring "unauthorized: authentication required"
  run_buildah 125 from --cert-dir $REGISTRY_DIR --signature-policy ${TESTSDIR}/policy.json --creds baduser:"$testpassword" localhost:$REGISTRY_PORT/my-alpine
  expect_output --substring "unauthorized: authentication required"

  # This should work
  run_buildah from --cert-dir $REGISTRY_DIR --name "my-alpine-work-ctr" --signature-policy ${TESTSDIR}/policy.json --creds "$testuser":"$testpassword" localhost:$REGISTRY_PORT/my-alpine
  expect_output --from="${lines[-1]}" "my-alpine-work-ctr"

  # Create Dockerfile for bud tests
  mkdir -p ${TESTDIR}/dockerdir
  DOCKERFILE=${TESTDIR}/dockerdir/Dockerfile
  /bin/cat <<EOM >$DOCKERFILE
FROM localhost:$REGISTRY_PORT/my-alpine
EOM

  # Remove containers and images before bud tests
  run_buildah rm --all
  run_buildah rmi -f --all

  # bud test bad password should fail
  run_buildah 125 bud -f $DOCKERFILE --signature-policy ${TESTSDIR}/policy.json --tls-verify=false --creds="$testuser":badpassword
  expect_output --substring "unauthorized: authentication required" \
                "buildah bud with wrong credentials"

  # bud test this should work
  run_buildah bud -f $DOCKERFILE --signature-policy ${TESTSDIR}/policy.json --tls-verify=false --creds="$testuser":"$testpassword" .
  expect_output --from="${lines[0]}" "STEP 1/1: FROM localhost:$REGISTRY_PORT/my-alpine"
  expect_output --substring "Writing manifest to image destination"
}


@test "authenticate: with --tls-verify=true" {
  _prefetch alpine

  start_registry

  # Push with correct credentials: should pass
  run_buildah push --signature-policy ${TESTSDIR}/policy.json --tls-verify=true --cert-dir=$REGISTRY_DIR --creds testuser:testpassword alpine localhost:$REGISTRY_PORT/my-alpine
  expect_output --substring "Writing manifest to image destination"

  # Push with wrong credentials: should fail
  run_buildah 125 push --signature-policy ${TESTSDIR}/policy.json --tls-verify=true --cert-dir=$REGISTRY_DIR --creds testuser:WRONGPASSWORD alpine localhost:$REGISTRY_PORT/my-alpine
  expect_output --substring "unauthorized: authentication required"

  # Make sure we can fetch it
  run_buildah from --pull-always --cert-dir=$REGISTRY_DIR --tls-verify=true --creds=testuser:testpassword localhost:$REGISTRY_PORT/my-alpine
  expect_output --from="${lines[-1]}" "localhost-working-container"
  cid="${lines[-1]}"

  # Commit with correct credentials
  run_buildah run $cid touch testfile
  run_buildah commit --signature-policy ${TESTSDIR}/policy.json --cert-dir=$REGISTRY_DIR --tls-verify=true --creds=testuser:testpassword $cid docker://localhost:$REGISTRY_PORT/my-alpine

  # Create Dockerfile for bud tests
  mkdir -p ${TESTDIR}/dockerdir
  DOCKERFILE=${TESTDIR}/dockerdir/Dockerfile
  /bin/cat <<EOM >$DOCKERFILE
FROM localhost:$REGISTRY_PORT/my-alpine
RUN rm testfile
EOM

  # Remove containers and images before bud tests
  run_buildah rm --all
  run_buildah rmi -f --all

  # bud with correct credentials
  run_buildah bud -f $DOCKERFILE --signature-policy ${TESTSDIR}/policy.json --cert-dir=$REGISTRY_DIR --tls-verify=true --creds=testuser:testpassword .
  expect_output --from="${lines[0]}" "STEP 1/2: FROM localhost:$REGISTRY_PORT/my-alpine"
  expect_output --substring "Writing manifest to image destination"
}


@test "authenticate: with cached (not command-line) credentials" {
  _prefetch alpine

  start_registry

  run_buildah 0 login --tls-verify=false --username testuser --password testpassword localhost:$REGISTRY_PORT
  expect_output "Login Succeeded!"

  # After login, push should pass
  run_buildah push --signature-policy ${TESTSDIR}/policy.json --tls-verify=false alpine localhost:$REGISTRY_PORT/my-alpine
  expect_output --substring "Storing signatures"

  run_buildah 125 login --tls-verify=false --username testuser --password WRONGPASSWORD localhost:$REGISTRY_PORT
  expect_output 'error logging into "localhost:'"$REGISTRY_PORT"'": invalid username/password' \
                "buildah login, wrong credentials"

  run_buildah 0 logout localhost:$REGISTRY_PORT
  expect_output "Removed login credentials for localhost:$REGISTRY_PORT"

  run_buildah 125 push --signature-policy ${TESTSDIR}/policy.json --tls-verify=false alpine localhost:$REGISTRY_PORT/my-alpine
  expect_output --substring "unauthorized: authentication required" \
                "buildah push after buildah logout"
}
