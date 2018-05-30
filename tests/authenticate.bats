#!/usr/bin/env bats

load helpers

@test "from-authenticate-cert-and-creds" {

  buildah from --pull --name "alpine" --signature-policy ${TESTSDIR}/policy.json alpine
  run buildah push --signature-policy ${TESTSDIR}/policy.json --tls-verify=false --creds testuser:testpassword alpine localhost:5000/my-alpine
  echo "$output"
  [ "$status" -eq 0 ]

  # This should fail
  run buildah push  --signature-policy ${TESTSDIR}/policy.json --tls-verify=true localhost:5000/my-alpine
  [ "$status" -ne 0 ]

  # This should fail
  run buildah from --signature-policy ${TESTSDIR}/policy.json --tls-verify=false --creds baduser:badpassword localhost:5000/my-alpine
  [ "$status" -ne 0 ]

  # This should work
  run buildah from --name "my-alpine" --signature-policy ${TESTSDIR}/policy.json --tls-verify=false --creds testuser:testpassword localhost:5000/my-alpine
  [ "$status" -eq 0 ]

  # Create Dockerfile for bud tests
  FILE=./Dockerfile
  /bin/cat <<EOM >$FILE
FROM localhost:5000/my-alpine
EOM
  chmod +x $FILE

  # Remove containers and images before bud tests
  buildah rm --all
  buildah rmi -f --all

  # bud test bad password should fail
  run buildah bud -f ./Dockerfile --signature-policy ${TESTSDIR}/policy.json --tls-verify=false --creds=testuser:badpassword
  [ "$status" -ne 0 ]

  # bud test this should work
  run buildah bud -f ./Dockerfile --signature-policy ${TESTSDIR}/policy.json --tls-verify=false --creds=testuser:testpassword .
  echo $status
  [ "$status" -eq 0 ]

  # Clean up
  rm -f ./Dockerfile
  buildah rm -a
  buildah rmi -f --all
}
