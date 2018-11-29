#!/bin/bash
# test_buildah_authentication
# A script to be run at the command line with Buildah installed.
# This currently needs to be run as root and Docker must be
# installed on the system.
# This will test the code and should be run with this command:
#
# /bin/bash -v test_buildah_authentication.sh

########
# System setup - Create dir for creds and start Docker
########
mkdir -p /root/auth
systemctl restart docker

########
# Create creds and store in /root/auth/htpasswd
########
registry=$(buildah from registry:2)
buildah run $registry -- htpasswd -Bbn testuser testpassword > /root/auth/htpasswd

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
# Buildah from (pull) using certs and cached Docker creds.
# Should show two alpine images and containers when done.
########
ctrid=$(buildah from --cert-dir /root/auth localhost:5000/my-alpine) 

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
# Push using only certs, this should FAIL.
########
buildah push --cert-dir /root/auth --tls-verify=true alpine docker://localhost:5000/my-alpine

########
# Push using creds, certs and no transport (docker://), this should work.
########
buildah push --cert-dir ~/auth --tls-verify=true --creds=testuser:testpassword alpine localhost:5000/my-alpine

########
# Push using a bad password , this should FAIL.
########
buildah push --cert-dir ~/auth --tls-verify=true --creds=testuser:badpassword alpine localhost:5000/my-alpine

########
# No creds anywhere, only the certificate, this should FAIL.
########
buildah from --cert-dir /root/auth  --tls-verify=true localhost:5000/my-alpine

########
# From with creds and certs, this should work
########
ctrid=$(buildah from --cert-dir /root/auth  --tls-verify=true --creds=testuser:testpassword localhost:5000/my-alpine)

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
# No credentials, this should FAIL.
########
buildah commit --cert-dir /root/auth  --tls-verify=true alpine-working-container docker://localhost:5000/my-commit-alpine

########
# This should work, writing image in registry.  Will not create an image locally.
########
buildah commit --cert-dir /root/auth  --tls-verify=true --creds=testuser:testpassword  alpine-working-container docker://localhost:5000/my-commit-alpine

########
# Use bad password on from/pull, this should FAIL
########
buildah from --pull-always --cert-dir /root/auth  --tls-verify=true --creds=testuser:badpassword localhost:5000/my-commit-alpine

########
# Pull the new image that we just commited
########
buildah from --pull-always --cert-dir /root/auth --tls-verify=true --creds=testuser:testpassword localhost:5000/my-commit-alpine

########
# Show stuff
########
docker ps --all

docker images

buildah containers

buildah images

########
# Create Dockerfile 
########
FILE=./Dockerfile
/bin/cat <<EOM >$FILE
FROM localhost:5000/my-commit-alpine
EOM
chmod +x $FILE

########
# Clean up Buildah
########
buildah rm --all
buildah rmi -f $(buildah --debug=false images -q)

########
# Try Buildah bud with creds but no auth, this should FAIL 
########
buildah bud -f ./Dockerfile --tls-verify=true --creds=testuser:testpassword

########
# Try Buildah bud with creds and auth, this should work 
########
buildah bud -f ./Dockerfile --cert-dir /root/auth  --tls-verify=true --creds=testuser:testpassword

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
read -p "Press enter to continue and clean up all"

rm -f ./Dockerfile
rm -rf ${TESTDIR}/auth
docker rm -f $(docker ps --all -q)
docker rmi -f $(docker images -q)
buildah rm $(buildah containers -q)
buildah rmi -f $(buildah --debug=false images -q)
