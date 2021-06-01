#!/usr/bin/env bats

load helpers

@test "authenticate: login/logout" {
  run_buildah 0 login --username testuserfoo --password testpassword docker.io

  run_buildah 0 logout docker.io
}

@test "authenticate: login/logout should succeed with XDG_RUNTIME_DIR unset" {
  unset XDG_RUNTIME_DIR
  run_buildah 0 login --username testuserfoo --password testpassword docker.io

  run_buildah 0 logout docker.io
}

@test "authenticate: logout should fail with nonexistent authfile" {
  run_buildah 0 login --username testuserfoo --password testpassword docker.io

  run_buildah 125 logout --authfile /tmp/nonexistent docker.io
  expect_output "checking authfile: stat /tmp/nonexistent: no such file or directory"

  run_buildah 0 logout docker.io
}

@test "authenticate: cert and credentials" {

  _prefetch alpine

  # Basic test: should pass
  run_buildah push --signature-policy ${TESTSDIR}/policy.json --tls-verify=false --creds testuser:testpassword alpine localhost:5000/my-alpine
  expect_output --substring "Writing manifest to image destination"

  # With tls-verify=true, should fail due to self-signed cert
  # The magic GODEBUG is needed for RHEL on 2021-01-20. Without it,
  # we get the following error instead of 'unknown authority':
  #   x509: certificate relies on legacy Common Name field, use SANs or [...]
  # It is possible that this is a temporary workaround, and Go
  # may remove it without notice. We'll deal with that then.
  GODEBUG=x509ignoreCN=0 run_buildah 125 push  --signature-policy ${TESTSDIR}/policy.json --tls-verify=true alpine localhost:5000/my-alpine
  expect_output --substring " x509: certificate signed by unknown authority" \
                "push with --tls-verify=true"

  # wrong credentials: should fail
  run_buildah 125 from --signature-policy ${TESTSDIR}/policy.json --tls-verify=false --creds baduser:badpassword localhost:5000/my-alpine
  expect_output --substring "unauthorized: authentication required"

  # This should work
  run_buildah from --name "my-alpine-work-ctr" --signature-policy ${TESTSDIR}/policy.json --tls-verify=false --creds testuser:testpassword localhost:5000/my-alpine
  expect_output --from="${lines[-1]}" "my-alpine-work-ctr"

  # Create Dockerfile for bud tests
  mkdir -p ${TESTDIR}/dockerdir
  DOCKERFILE=${TESTDIR}/dockerdir/Dockerfile
  /bin/cat <<EOM >$DOCKERFILE
FROM localhost:5000/my-alpine
EOM

  # Remove containers and images before bud tests
  run_buildah rm --all
  run_buildah rmi -f --all

  # bud test bad password should fail
  run_buildah 125 bud -f $DOCKERFILE --signature-policy ${TESTSDIR}/policy.json --tls-verify=false --creds=testuser:badpassword
  expect_output --substring "unauthorized: authentication required" \
                "buildah bud with wrong credentials"

  # bud test this should work
  run_buildah bud -f $DOCKERFILE --signature-policy ${TESTSDIR}/policy.json --tls-verify=false --creds=testuser:testpassword .
  expect_output --from="${lines[0]}" "STEP 1/1: FROM localhost:5000/my-alpine"
  expect_output --substring "Writing manifest to image destination"
}


@test "authenticate: with --tls-verify=true" {
  if [ -z "$BUILDAH_AUTHDIR" ]; then
    # Special case: in Cirrus, the registry auth dir is hardcoded
    if [ -n "$CIRRUS_CI" -a -e "$HOME/auth/domain.cert" ]; then
      BUILDAH_AUTHDIR="$HOME/auth"
    else
      skip "\$BUILDAH_AUTHDIR undefined"
    fi
  fi

  _prefetch alpine

  # Push with correct credentials: should pass
  run_buildah push --signature-policy ${TESTSDIR}/policy.json --tls-verify=true --cert-dir=$BUILDAH_AUTHDIR --creds testuser:testpassword alpine localhost:5000/my-alpine
  expect_output --substring "Writing manifest to image destination"

  # Push with wrong credentials: should fail
  run_buildah 125 push --signature-policy ${TESTSDIR}/policy.json --tls-verify=true --cert-dir=$BUILDAH_AUTHDIR --creds testuser:WRONGPASSWORD alpine localhost:5000/my-alpine
  expect_output --substring "unauthorized: authentication required"

  # Make sure we can fetch it
  run_buildah from --pull-always --cert-dir=$BUILDAH_AUTHDIR --tls-verify=true --creds=testuser:testpassword localhost:5000/my-alpine
  expect_output --from="${lines[-1]}" "localhost-working-container"
  cid="${lines[-1]}"

  # Commit with correct credentials
  run_buildah run $cid touch testfile
  run_buildah commit --signature-policy ${TESTSDIR}/policy.json --cert-dir=$BUILDAH_AUTHDIR --tls-verify=true --creds=testuser:testpassword $cid docker://localhost:5000/my-alpine

  # Create Dockerfile for bud tests
  mkdir -p ${TESTDIR}/dockerdir
  DOCKERFILE=${TESTDIR}/dockerdir/Dockerfile
  /bin/cat <<EOM >$DOCKERFILE
FROM localhost:5000/my-alpine
RUN rm testfile
EOM

  # Remove containers and images before bud tests
  run_buildah rm --all
  run_buildah rmi -f --all

  # bud with correct credentials
  run_buildah bud -f $DOCKERFILE --signature-policy ${TESTSDIR}/policy.json --cert-dir=$BUILDAH_AUTHDIR --tls-verify=true --creds=testuser:testpassword .
  expect_output --from="${lines[0]}" "STEP 1/2: FROM localhost:5000/my-alpine"
  expect_output --substring "Writing manifest to image destination"
}


@test "authenticate: with cached (not command-line) credentials" {
  _prefetch alpine

  run_buildah 0 login --tls-verify=false --username testuser --password testpassword localhost:5000
  expect_output "Login Succeeded!"

  # After login, push should pass
  run_buildah push --signature-policy ${TESTSDIR}/policy.json --tls-verify=false alpine localhost:5000/my-alpine
  expect_output --substring "Storing signatures"

  run_buildah 125 login --tls-verify=false --username testuser --password WRONGPASSWORD localhost:5000
  expect_output 'error logging into "localhost:5000": invalid username/password' \
                "buildah login, wrong credentials"

  run_buildah 0 logout localhost:5000
  expect_output "Removed login credentials for localhost:5000"

  run_buildah 125 push --signature-policy ${TESTSDIR}/policy.json --tls-verify=false alpine localhost:5000/my-alpine
  expect_output --substring "unauthorized: authentication required" \
                "buildah push after buildah logout"
}
