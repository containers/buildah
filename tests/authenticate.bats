#!/usr/bin/env bats

load helpers

@test "authenticate: login/logout" {
  start_registry testuserfoo testpassword

  run_buildah 0 login --cert-dir $REGISTRY_DIR --username testuserfoo --password testpassword localhost:$REGISTRY_PORT

  run_buildah 0 logout localhost:$REGISTRY_PORT
}

@test "authenticate: with stdin" {
  start_registry testuserfoo testpassword
  run_buildah 0 login localhost:$REGISTRY_PORT --cert-dir $REGISTRY_DIR --username testuserfoo --password-stdin <<< testpassword
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
  assert "$output" =~ "Error: credential file is not accessible: (faccessat|stat) /tmp/nonexistent: no such file or directory"

  run_buildah 125 logout --compat-auth-file /tmp/nonexistent localhost:$REGISTRY_PORT
  assert "$output" =~ "Error: credential file is not accessible: (faccessat|stat) /tmp/nonexistent: no such file or directory"

  run_buildah 0 logout localhost:$REGISTRY_PORT
}

@test "authenticate: logout should fail with inconsistent authfiles" {
  ambiguous_file=${TEST_SCRATCH_DIR}/ambiguous-auth.json
  echo '{}' > $ambiguous_file # To make sure we are not hitting the “file not found” path

  # We don’t start a real registry; login should never get that far.
  run_buildah 125 login --authfile "$ambiguous_file" --compat-auth-file "$ambiguous_file" localhost:5000
  expect_output "Error: options for paths to the credential file and to the Docker-compatible credential file can not be set simultaneously"

  run_buildah 125 logout --authfile "$ambiguous_file" --compat-auth-file "$ambiguous_file" localhost:5000
  expect_output "Error: options for paths to the credential file and to the Docker-compatible credential file can not be set simultaneously"
}

@test "authenticate: cert and credentials" {
  _prefetch alpine

  testuser="testuser$RANDOM"
  testpassword="testpassword$RANDOM"
  start_registry "$testuser" "$testpassword"

  # Basic test: should pass
  run_buildah push --cert-dir $REGISTRY_DIR $WITH_POLICY_JSON --tls-verify=false --creds "$testuser":"$testpassword" alpine localhost:$REGISTRY_PORT/my-alpine
  expect_output --substring "Writing manifest to image destination"

  # With tls-verify=true, should fail due to self-signed cert
  run_buildah 125 push $WITH_POLICY_JSON --tls-verify=true alpine localhost:$REGISTRY_PORT/my-alpine
  expect_output --substring " x509: certificate signed by unknown authority" \
                "push with --tls-verify=true"

  # wrong credentials: should fail
  run_buildah 125 from --cert-dir $REGISTRY_DIR $WITH_POLICY_JSON --creds baduser:badpassword localhost:$REGISTRY_PORT/my-alpine
  expect_output --substring "authentication required"
  run_buildah 125 from --cert-dir $REGISTRY_DIR $WITH_POLICY_JSON --creds "$testuser":badpassword localhost:$REGISTRY_PORT/my-alpine
  expect_output --substring "authentication required"
  run_buildah 125 from --cert-dir $REGISTRY_DIR $WITH_POLICY_JSON --creds baduser:"$testpassword" localhost:$REGISTRY_PORT/my-alpine
  expect_output --substring "authentication required"

  # This should work
  run_buildah from --cert-dir $REGISTRY_DIR --name "my-alpine-work-ctr" $WITH_POLICY_JSON --creds "$testuser":"$testpassword" localhost:$REGISTRY_PORT/my-alpine
  expect_output --from="${lines[-1]}" "my-alpine-work-ctr"

  # Create Dockerfile for bud tests
  mkdir -p ${TEST_SCRATCH_DIR}/dockerdir
  DOCKERFILE=${TEST_SCRATCH_DIR}/dockerdir/Dockerfile
  /bin/cat <<EOM >$DOCKERFILE
FROM localhost:$REGISTRY_PORT/my-alpine
EOM

  # Remove containers and images before bud tests
  run_buildah rm --all
  run_buildah rmi -f --all

  # bud test bad password should fail
  run_buildah 125 bud -f $DOCKERFILE $WITH_POLICY_JSON --tls-verify=false --creds="$testuser":badpassword
  expect_output --substring "authentication required" \
                "buildah bud with wrong credentials"

  # bud test this should work
  run_buildah bud -f $DOCKERFILE $WITH_POLICY_JSON --tls-verify=false --creds="$testuser":"$testpassword" .
  expect_output --from="${lines[0]}" "STEP 1/1: FROM localhost:$REGISTRY_PORT/my-alpine"
  expect_output --substring "Writing manifest to image destination"
}


@test "authenticate: with --tls-verify=true" {
  _prefetch alpine

  start_registry

  # Push with correct credentials: should pass
  run_buildah push $WITH_POLICY_JSON --tls-verify=true --cert-dir=$REGISTRY_DIR --creds testuser:testpassword alpine localhost:$REGISTRY_PORT/my-alpine
  expect_output --substring "Writing manifest to image destination"

  # Push with wrong credentials: should fail
  run_buildah 125 push $WITH_POLICY_JSON --tls-verify=true --cert-dir=$REGISTRY_DIR --creds testuser:WRONGPASSWORD alpine localhost:$REGISTRY_PORT/my-alpine
  expect_output --substring "authentication required"

  # Make sure we can fetch it
  run_buildah from --pull-always --cert-dir=$REGISTRY_DIR --tls-verify=true --creds=testuser:testpassword localhost:$REGISTRY_PORT/my-alpine
  expect_output --from="${lines[-1]}" "localhost-working-container"
  cid="${lines[-1]}"

  # Commit with correct credentials
  run_buildah run $cid touch testfile
  run_buildah commit $WITH_POLICY_JSON --cert-dir=$REGISTRY_DIR --tls-verify=true --creds=testuser:testpassword $cid docker://localhost:$REGISTRY_PORT/my-alpine

  # Create Dockerfile for bud tests
  mkdir -p ${TEST_SCRATCH_DIR}/dockerdir
  DOCKERFILE=${TEST_SCRATCH_DIR}/dockerdir/Dockerfile
  /bin/cat <<EOM >$DOCKERFILE
FROM localhost:$REGISTRY_PORT/my-alpine
RUN rm testfile
EOM

  # Remove containers and images before bud tests
  run_buildah rm --all
  run_buildah rmi -f --all

  # bud with correct credentials
  run_buildah bud -f $DOCKERFILE $WITH_POLICY_JSON --cert-dir=$REGISTRY_DIR --tls-verify=true --creds=testuser:testpassword .
  expect_output --from="${lines[0]}" "STEP 1/2: FROM localhost:$REGISTRY_PORT/my-alpine"
  expect_output --substring "Writing manifest to image destination"
}


@test "authenticate: with cached (not command-line) credentials" {
  _prefetch alpine

  start_registry

  run_buildah 0 login --tls-verify=false --username testuser --password testpassword localhost:$REGISTRY_PORT
  expect_output "Login Succeeded!"

  # After login, push should pass
  run_buildah push $WITH_POLICY_JSON --tls-verify=false alpine localhost:$REGISTRY_PORT/my-alpine
  expect_output --substring "Writing manifest to image destination"

  run_buildah 125 login --tls-verify=false --username testuser --password WRONGPASSWORD localhost:$REGISTRY_PORT
  expect_output --substring 'logging into "localhost:'"$REGISTRY_PORT"'": invalid username/password' \
                "buildah login, wrong credentials"

  run_buildah 0 logout localhost:$REGISTRY_PORT
  expect_output "Removed login credentials for localhost:$REGISTRY_PORT"

  run_buildah 125 push $WITH_POLICY_JSON --tls-verify=false alpine localhost:$REGISTRY_PORT/my-alpine
  expect_output --substring "authentication required" \
                "buildah push after buildah logout"
}
