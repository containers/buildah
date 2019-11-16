#!/usr/bin/env bash 

# docker-compatibility-demo.sh
# author : ipbabble
# Assumptions install buildah, podman & docker
# Do NOT start the docker daemon
# Set some of the variables below

demoimg=dockercompatibilitydemo
quayuser=ipbabble
myname="William Henry"
distro=fedora
distrorelease=30
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
echo -e "Using ${red}RED${reset} to introduce Docker commands"

echo -e "Building an image called ${demoimg}"
read -p "${green}Start of the script${reset}"

set -x
read -p "${green}Create a new container on disk from ${distro}${reset}"
newcontainer=$(buildah from ${distro})
read -p "${green}Update packages and clean all ${reset}"
buildah run $newcontainer -- ${pkgmgr} -y update && ${pkgmgr} -y clean all
read -p "${green}Install nginx${reset}"
buildah run $newcontainer -- ${pkgmgr} -y install nginx && ${pkgmgr} -y clean all 
read -p "${green}Make some nginx config and home page changes ${reset}"
buildah run $newcontainer bash -c 'echo "daemon off;" >> /etc/nginx/nginx.conf'
buildah run $newcontainer bash -c 'echo "nginx on OCI Fedora image, built using Buildah" > /usr/share/nginx/html/index.html'
read -p "${green}Use buildah config to expose the port and set the entrypoint${reset}"
buildah config --port 80 --entrypoint /usr/sbin/nginx $newcontainer
read -p "${green}Set other meta data using buildah config${reset}"
buildah config --created-by "${quayuser}"  $newcontainer
buildah config --author "${myname}" --label name=$demoimg $newcontainer
read -p "${green}Inspect the container image meta data${yellow}"
buildah inspect $newcontainer
read -p "${green}Commit the container to an OCI image called ${demoimg}.${reset}"
buildah commit $newcontainer $demoimg
read -p "${green}List the images we have.${reset}"
buildah images
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
read -p "${cyan}Start the Docker daemon. Using restart in case it is already started${reset}"
systemctl restart docker
read -p "${red}List the Docker images in the repository - should be empty${reset}"
docker images                                                                          
read -p "${blue}Push the image to the local Docker repository using docker-daemon${reset}"
podman push $demoimg docker-daemon:$quayuser/dockercompatibilitydemo:latest       
read -p "${red}List the Docker images in the repository${reset}"
docker images                                                                          
read -p "${red}Start the container from the new Docker repo image${reset}"
dockercontainer=$(docker run -d -p 80:80 $quayuser/$demoimg)
read -p "${cyan}Check that nginx is up and running with our new page${reset}"
curl localhost
read -p "${red}Stop the container and rm it${reset}"
docker stop $dockercontainer                                                                            
docker rm $dockercontainer                                                                            
docker rmi $demoimg
read -p "${cyan}Stop Docker${reset}"
systemctl stop docker
echo -e "${red}We are done!${reset}"
