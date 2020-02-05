#!/usr/bin/env bats

load helpers

@test "login/logout" {
  run_buildah 0 login --username testuserfoo --password testpassword docker.io

  run_buildah 0 logout docker.io
}

@test "login/logout should succeed with XDG_RUNTIME_DIR unset" {
  unset XDG_RUNTIME_DIR
  run_buildah 0 login --username testuserfoo --password testpassword docker.io

  run_buildah 0 logout docker.io
}

@test "logout should fail with nonexist authfile" {
  run_buildah 0 login --username testuserfoo --password testpassword docker.io

  run_buildah 1 logout --authfile /tmp/nonexist docker.io

  run_buildah 0 logout docker.io
}

@test "from-authenticate-cert-and-creds" {

  run_buildah from --pull --name "alpine" --signature-policy ${TESTSDIR}/policy.json alpine
  run_buildah push --signature-policy ${TESTSDIR}/policy.json --tls-verify=false --creds testuser:testpassword alpine localhost:5000/my-alpine

  # This should fail
  run_buildah 1 push  --signature-policy ${TESTSDIR}/policy.json --tls-verify=true localhost:5000/my-alpine

  # This should fail
  run_buildah 1 from --signature-policy ${TESTSDIR}/policy.json --tls-verify=false --creds baduser:badpassword localhost:5000/my-alpine

  # This should work
  run_buildah from --name "my-alpine" --signature-policy ${TESTSDIR}/policy.json --tls-verify=false --creds testuser:testpassword localhost:5000/my-alpine

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
  run_buildah 1 bud -f $DOCKERFILE --signature-policy ${TESTSDIR}/policy.json --tls-verify=false --creds=testuser:badpassword

  # bud test this should work
  run_buildah bud -f $DOCKERFILE --signature-policy ${TESTSDIR}/policy.json --tls-verify=false --creds=testuser:testpassword .
}
