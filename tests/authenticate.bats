#!/usr/bin/env bats

load helpers

@test "from-authenticate-cert-and-creds" {

  buildah from --pull --signature-policy ${TESTSDIR}/policy.json alpine

  registry=$(buildah from registry:2 --signature-policy ${TESTSDIR}/policy.json)
  mkdir -p  ${TESTDIR}/auth

  # Set policy .json file 
  mnt=$(buildah mount $registry)
  mkdir -p $mnt/${TESTSDIR}
  mkdir -p $mnt/etc/containers
  cp ${TESTSDIR}/policy.json $mnt/${TESTSDIR}/policy.json
  cp ${TESTSDIR}/policy.json $mnt/etc/containers/policy.json

  # Create creds and store in ${TESTDIR}/auth/htpasswd
  buildah run $registry -- htpasswd -Bbn testuser testpassword > ${TESTDIR}/auth/htpasswd
  # Create certifcate via openssl
  openssl req -newkey rsa:4096 -nodes -sha256 -keyout ${TESTDIR}/auth/domain.key -x509 -days 2 -out ${TESTDIR}/auth/domain.crt -subj "/C=US/ST=Foo/L=Bar/O=Red Hat, Inc./CN=localhost"
  # Skopeo and buildah both require *.cert file
  cp ${TESTDIR}/auth/domain.crt ${TESTDIR}/auth/domain.cert

  # Create a private registry that uses certificate and creds file
  buildah config --env REGISTRY_AUTH=htpasswd --env "REGISTRY_AUTH_HTPASSWD_REALM=Registry Realm" --env REGISTRY_AUTH_HTPASSWD_PATH=${TESTDIR}/auth/htpasswd --env REGISTRY_HTTP_TLS_CERTIFICATE=${TESTDIR}/auth/domain.crt --env REGISTRY_HTTP_TLS_KEY=${TESTDIR}/auth/domain.key $registry

  #Fork the registry and then test against it.  3>- closes the open fd of the subprocess
  (buildah run $registry ${TESTSDIR}/policy.json -v ${TESTDIR}/auth:${TESTDIR}/auth:Z) 3>- & 
  REGISTRY_PID=$!

  # This should fail
  run buildah push --signature-policy ${TESTSDIR}/policy.json --cert-dir ${TESTDIR}/auth --creds testuser:badpassword alpine localhost:5000/my-alpine
  [ "$status" -ne 0 ]

  # This should work
  buildah push --signature-policy ${TESTSDIR}/policy.json --cert-dir ${TESTDIR}/auth --creds testuser:testpassword alpine docker://localhost:5000/my-alpine

  # This should fail
  run ctrid=$(buildah from localhost:5000/my-alpine --signature-policy ${TESTSDIR}/policy.json --cert-dir ${TESTDIR}/auth --tls-verify=true)
  [ "$status" -ne 0 ]

  # This should work
  ctrid=$(buildah from localhost:5000/my-alpine --signature-policy ${TESTSDIR}/policy.json --cert-dir ${TESTDIR}/auth --tls-verify=true --creds=testuser:testpassword)

  # Clean up
  rm -rf ${TESTDIR}/auth
  buildah rm $ctrid
  buildah rmi -f $(buildah --debug=false images -q)
  kill -9 $REGISTRY_PID
}
