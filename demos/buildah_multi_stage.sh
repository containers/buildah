#!/bin/bash 

# author : tsweeney (based on ipbabble's other demos) 
# Based on Alex Ellis blog (https://blog.alexellis.io/mutli-stage-docker-builds)
# Assumptions install buildah and podman
# Set some of the variables below

demoimg=mymultidemo
quayuser=myquauuser
myname=MyName
distrorelease=28
pkgmgr=dnf   # switch to yum if using yum 

#Setting up some colors for helping read the demo output
bold=$(tput bold)
red=$(tput setaf 1)
green=$(tput setaf 2)
yellow=$(tput setaf 3)
blue=$(tput setaf 4)
cyan=$(tput setaf 6)
reset=$(tput sgr0)

echo -e "Using ${green}GREEN${reset} to introduce Buildah steps"
echo -e "Using ${yellow}YELLOW${reset} to introduce code"
echo -e "Using ${blue}BLUE${reset} to introduce Podman steps"
echo -e "Using ${cyan}CYAN${reset} to introduce bash commands"
echo -e "Building an image called ${demoimg}"
read -p "${green}Start of the script${reset}"

set -x
read -p "${yellow}Create Dockerfile.multi${reset}"
FILE=./Dockerfile.multi
/bin/cat <<EOM >$FILE
FROM golang:1.7.3 as builder
WORKDIR /go/src/github.com/alexellis/href-counter/
RUN go get -d -v golang.org/x/net/html
COPY app.go  .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o app .
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /go/src/github.com/alexellis/href-counter/app    .
CMD ["./app"]
EOM
chmod +x $FILE
read -p "${yellow}Let's look at our Dockerfile.multi${reset}"
cat ./Dockerfile.multi
read -p "${yellow}Pull app.go from GitHub${reset}"
curl https://raw.githubusercontent.com/alexellis/href-counter/master/app.go > app.go
read -p "${green}Create a new image on disk from Dockerfile.multi${reset}"
newcontainer=$(buildah bud -t multifromfile:latest -f ./Dockerfile.multi .)
read -p "${blue}Run the multifromfile container${reset}"
podman run --network=host -e url=https://www.alexellis.io/ multifromfile:latest 
podman run --network=host -e url=https://www.alexellis.io/ multifromfile:latest
read -p "${green}Let's check the size of the images${reset}"
buildah images
read -p "${green}Let's clear out our containers${reset}"
buildah rm -a

read -p "${green}Let's build the container with Buildah, first GoLang${reset}"
buildcntr=$(buildah from golang:1.7.3)
read -p "${green}Let's mount the container getting the root directory${reset}"
buildmnt=$(buildah mount $buildcntr) 
read -p "${green}Let's get x/net/html into the container${reset}"
buildah run $buildcntr go get -d -v golang.org/x/net/html 
read -p "${yellow}Copy app.go into the container${reset}"
cp app.go $buildmnt/go
read -p "${green}Build app.go inside the container${reset}"
buildah run $buildcntr /bin/sh -c "CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o app ."

read -p "${green}Build new image to run application in production${reset}"
rtcntr=$(buildah from alpine:latest)
read -p "${green}Mount the new images root fs${reset}"
rtmnt=$(buildah mount $rtcntr)
read -p "${green}Install required packages${reset}"
buildah run $rtcntr apk --no-cache add ca-certificates
read -p "${yellow}Copy the app from the previous container${reset}"
cp $buildmnt/go/app $rtmnt
read -p "${yellow}Set the CMD for the container${reset}"
buildah config --cmd ./app $rtcntr
read -p "${yellow}Unmount and commit the rtimg${reset}"
buildah unmount $rtcntr
buildah commit $rtcntr multifrombuildah:latest

read -p "${blue}Run the multifrombuildah container${reset}"
podman run --network=host -e url=https://www.alexellis.io/ multifrombuildah:latest 
podman run --network=host -e url=https://www.alexellis.io/ multifrombuildah:latest

read -p "${green}Let's check the size of the images${reset}"
buildah images
read -p "${green}Let's clear out our containers${reset}"
buildah rm -a
read -p "${green}Let's clear out our images${reset}"
buildah rmi -a -f
read -p "${green}Let's remove app.go and Dockerfile.multi${reset}"
rm ./app.go ./Dockerfile.multi

echo -e "${red}We are done!${reset}"
