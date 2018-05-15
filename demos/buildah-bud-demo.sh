#!/bin/bash 

# buildah-bud-demo.sh
# author : ipbabble
# Assumptions install buildah, podman & docker
# Do NOT start the docker deamon
# Set some of the variables below

demoimg=buildahbuddemo
quayuser=ipbabble
myname="William Henry"
distro=fedora
distrorelease=28
pkgmgr=dnf   # switch to yum if using yum 

#Setting up some colors for helping read the demo output
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
echo -e "Using ${red}RED${reset} to introduce Docker commands"

echo -e "Building an image called ${demoimg}"
read -p "${green}Start of the script${reset}"

set -x
DOCKERFILE=./Dockerfile
/bin/cat <<EOM >$DOCKERFILE
FROM docker://docker.io/fedora:latest
MAINTAINER ${myname}


RUN dnf -y update; dnf -y clean all
RUN dnf -y install nginx --setopt install_weak_deps=false; dnf -y clean all
RUN echo "daemon off;" >> /etc/nginx/nginx.conf
RUN echo "nginx on Fedora" > /usr/share/nginx/html/index.html

EXPOSE 80

CMD [ "/usr/sbin/nginx" ]
EOM
read -p "${cyan}Display the Dockerfile:${reset}"
cat $DOCKERFILE
read -p "${green}Create a new container image from Dockerfile${reset}"
buildah bud -t $demoimg .
read -p "${green}List the images we have.${reset}"
buildah images
read -p "${green}Inspect the container image meta data${yellow}"
buildah inspect --type image $demoimg
read -p "${blue}Run the container using Podman.${reset}"
containernum=$(podman run -d -p 80:80 $demoimg) 
read -p "${cyan}Check that nginx is up and running with our new page${reset}"
curl localhost
read -p "${blue}Stop the container and rm it${reset}"
podman ps                                                                                   
podman stop $containernum                                                                            
podman rm $containernum                                                                              
read -p "${cyan}Check that nginx is down${reset}"
curl localhost
read -p "${cyan}Start the Docker daemon. Using restart incase it is already started${reset}"
systemctl restart docker
read -p "${red}List the Docker images in the repository - should be empty${reset}"
docker images                                                                          
read -p "${blue}Push the image to the local Docker repository using docker-daemon${reset}"
podman push $demoimg docker-daemon:$quayuser/${demoimg}:latest       
read -p "${red}List the Docker images in the repository${reset}"
docker images                                                                          
read -p "${red}Start the container from the new Docker repo image${reset}"
dockercontainer=$(docker run -d -p 80:80 $quayuser/$demoimg)
read -p "${cyan}Check that nginx is up and running with our new page${reset}"
curl localhost
read -p "${red}Stop the container and remove it and the image${reset}"
docker stop $dockercontainer                                                                            
docker rm $dockercontainer                                                                            
docker rmi $demoimg
read -p "${cyan}Stop Docker${reset}"
systemctl stop docker
echo -e "${red}We are done!${reset}"