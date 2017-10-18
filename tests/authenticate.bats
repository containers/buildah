#!/usr/bin/env bats

load helpers

@test "from-authenticate-cert-and-creds" {

  buildah from --pull --name "alpine" --signature-policy ${TESTSDIR}/policy.json alpine
  run buildah push --signature-policy ${TESTSDIR}/policy.json --tls-verify=false --creds testuser:testpassword alpine localhost:5000/my-alpine
  echo "$output"
  [ "$status" -eq 0 ]

  # This should fail
  run buildah push localhost:5000/my-alpine --signature-policy ${TESTSDIR}/policy.json --tls-verify=true
  [ "$status" -ne 0 ]

  # This should fail
  run buildah from localhost:5000/my-alpine --signature-policy ${TESTSDIR}/policy.json --tls-verify=false --creds baduser:badpassword
  [ "$status" -ne 0 ]

  # This should work
  run buildah from localhost:5000/my-alpine --name "my-alpine" --signature-policy ${TESTSDIR}/policy.json --tls-verify=false --creds testuser:testpassword
  [ "$status" -eq 0 ]

  # Clean up
  buildah rm my-alpine
  buildah rm alpine 
  buildah rmi -f $(buildah --debug=false images -q)
}
