#!/bin/bash

# author : ipbabble
# Assumptions install buildah and podman
# login to Quay.io using buildah if you want to see the image push
#   otherwise it will just fail the last step and no biggy.
#   buildah login quay.io
# Set some of the variables below

demoimg=myshdemo
quayuser=ipbabble
myname=WilliamHenry
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
read -p "${green}Create a new container on disk from scratch${reset}"
newcontainer=$(buildah from scratch)
read -p "${green}Mount the root directory of the new scratch container${reset}"
scratchmnt=$(buildah mount $newcontainer)
read -p "${cyan}Lets see what is in scratchmnt${reset}"
ls $scratchmnt
echo -e "${red}Note that the root of the scratch container is EMPTY!${reset}"
read -p "${cyan}Time to install some basic bash capabilities: coreutils and bash packages${reset}"
if [ "$pkgmgr" == "dnf" ]; then
	$pkgmgr install --installroot $scratchmnt --release ${distrorelease} bash coreutils --setopt install_weak_deps=false -y
elif [ "$pkgmgr" == "yum" ]; then
	$pkgmgr install --installroot $scratchmnt --releasever ${distrorelease} bash coreutils  -y
else
	echo -e "${red}[Error] Unknown package manager ${pkgmgr}${reset}"
fi

read -p "${cyan}Clean up the packages${reset}"
$pkgmgr clean --installroot $scratchmnt all
read -p "${green}Run the shell and see what is inside. When your done, type ${red}exit${green} and return.${reset}"
buildah run $newcontainer bash
read -p "${cyan}Let's look at the program${yellow}"
FILE=./runecho.sh
/bin/cat <<EOM >$FILE
#!/bin/bash
for i in {1..9};
do
    echo "This is a new cloud native container using Buildah [" \$i "]"
done
EOM
chmod +x $FILE
cat $FILE
read -p "${green}Copy program into the container and run ls to see it is there${reset}"
buildah copy $newcontainer $FILE /usr/bin
ls -al $scratchmnt/usr/bin/*.sh
read -p "${green}Run the container using Buildah${reset}"
buildah run $newcontainer /usr/bin/runecho.sh
read -p "${green}Make the container run the program by default when container is run${reset}"
buildah config --entrypoint /usr/bin/runecho.sh $newcontainer
read -p "${green}Set some config information for the container image${reset}"
buildah config --author "${myname}" --created-by "${quayuser}" --label name=${demoimg} $newcontainer
read -p "${green}Inspect the meta data${yellow}"
buildah inspect $newcontainer
read -p "${green}Unmount the container and commit to an image called ${demoimg}.${reset}"
buildah unmount $newcontainer
buildah commit $newcontainer $demoimg
read -p "${green}List the images we have.${reset}"
buildah images
read -p "${blue}Run the container using Podman.${reset}"
podman run -t $demoimg
read -p "${green}Make sure you are already logged into your account on Quay.io. Or use Quay creds.${reset}"
buildah push $demoimg docker://quay.io/$quayuser/$demoimg
echo -e "${red}We are done!${reset}"
