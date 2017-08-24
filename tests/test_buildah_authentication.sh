#!/bin/bash
# test_buildah_authentication
# A script to be run at the command line with Buildah installed.
# This will test the code and should be run with this command:
#
# /bin/bash -v test_buildah_authentication.sh

########
# Create creds and store in /root/auth/htpasswd
########
docker run --entrypoint htpasswd registry:2 -Bbn testuser testpassword > /root/auth/htpasswd

########
# Create certificate via openssl
########
openssl req -newkey rsa:4096 -nodes -sha256 -keyout /root/auth/domain.key -x509 -days 2 -out /root/auth/domain.crt -subj "/C=US/ST=Foo/L=Bar/O=Red Hat, Inc./CN=localhost"

########
# Skopeo and buildah both require *.cert file
########
cp /root/auth/domain.crt /root/auth/domain.cert

########
# Create a private registry that uses certificate and creds file
########
docker run -d -p 5000:5000 --name registry -v /root/auth:/root/auth:Z -e "REGISTRY_AUTH=htpasswd" -e "REGISTRY_AUTH_HTPASSWD_REALM=Registry Realm" -e REGISTRY_AUTH_HTPASSWD_PATH=/root/auth/htpasswd -e REGISTRY_HTTP_TLS_CERTIFICATE=/root/auth/domain.crt -e REGISTRY_HTTP_TLS_KEY=/root/auth/domain.key registry:2

########
# Pull alpine
########
buildah from alpine

buildah containers

buildah images

########
# Log into docker on local repo
########
docker login localhost:5000 --username testuser --password testpassword

########
# Push to the local repo using cached Docker creds.
########
buildah push --cert-dir /root/auth alpine docker://localhost:5000/my-alpine

########
# Show stuff
########
docker ps --all

docker images

buildah containers

buildah images

########
# Buildah pulls using certs and cached Docker creds.
# Should show two alpine images and containers when done.
########
ctrid=$(buildah from localhost:5000/my-alpine --cert-dir /root/auth)

buildah containers

buildah images

########
# Clean up Buildah
########
buildah rm $ctrid
buildah rmi -f localhost:5000/my-alpine:latest

########
# Show stuff
########
docker ps --all

docker images

buildah containers

buildah images

########
# Log out of local repo
########
docker logout localhost:5000

########
# Push using only certs, this should fail.
########
buildah push --cert-dir /root/auth --tls-verify=true alpine docker://localhost:5000/my-alpine

########
# Push using creds and certs, this should work.
########
buildah push --cert-dir ~/auth --tls-verify=true --creds=testuser:testpassword alpine docker://localhost:5000/my-alpine

########
# This should fail, no creds anywhere, only the certificate
########
buildah from localhost:5000/my-alpine --cert-dir /root/auth  --tls-verify=true

########
# Log in with creds, this should work
########
ctrid=$(buildah from localhost:5000/my-alpine --cert-dir /root/auth  --tls-verify=true --creds=testuser:testpassword)

########
# Show stuff
########
docker ps --all

docker images

buildah containers

buildah images

########
# Clean up Buildah
########
buildah rm $ctrid
buildah rmi -f $(buildah --debug=false images -q)

########
# Pull alpine
########
buildah from alpine

########
# Show stuff
########
docker ps --all

docker images

buildah containers

buildah images

########
# Let's test commit
########

########
# No credentials, this should fail.
########
buildah commit --cert-dir /root/auth  --tls-verify=true alpine-working-container docker://localhost:5000/my-commit-alpine

########
# This should work, writing image in registry.  Will not create an image locally.
########
buildah commit --cert-dir /root/auth  --tls-verify=true --creds=testuser:testpassword  alpine-working-container docker://localhost:5000/my-commit-alpine

########
# Pull the new image that we just commited
########
buildah from localhost:5000/my-commit-alpine --cert-dir /root/auth  --tls-verify=true --creds=testuser:testpassword

########
# Show stuff
########
docker ps --all

docker images

buildah containers

buildah images

########
# Clean up
########
rm -rf ${TESTDIR}/auth
docker rm -f $(docker ps --all -q)
docker rmi -f $(docker images -q)
buildah rm $(buildah containers -q)
buildah rmi -f $(buildah --debug=false images -q)
