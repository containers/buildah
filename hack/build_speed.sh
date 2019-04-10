#! /bin/sh

# The main goal of this script is test and time builds using Buildah or Docker.
# We hope to use it to help optimize Buildah build performance
#
# It takes two options
# First option tells the type of the container image
# build to do.  Valid options are:
#       Docker  - docker build
#       Buildah - buildah bud
#       Both    - Do docker build followed by buildah bud
#
# Second Option specifies a directory or cleanup
# The script will 'find' files beginning with Dockerfile, for each Dockerfile
# it finds it will run a build with the Dockerfile and directory for the
# context. When it does the builds, it will call time on them to show how
# long the builds take.  The created image name will be a combination of the
# lowercased Directory name that the Dockerfile was found in plus the lower
# cased dockerfile name.
#
# if the second field is cleanup, the script will remove all images from the
# specified builder.
#
# The script does not check for conflicts on nameing.
#
# Outputs file:
#
#
# cat /tmp/build_speed.json
# {
#   "/usr/share/fedora-dockerfiles/redis/Dockerfile": {
#     "docker": {
#       "command": "docker build -f /usr/share/fedora-dockerfiles/redis/Dockerfile -t redis_dockerfile /usr/share/fedora-dockerfiles/redis",
#       "real": "3:28.70"
#     },
#     "buildah": {
#       "command": "buildah bud --layers -f /usr/share/fedora-dockerfiles/redis/Dockerfile -t redis_dockerfile /usr/share/fedora-dockerfiles/redis",
#       "real": "2:55.48"
#     }
#   }
# }
#
# Examples uses
#     ./build_speed.sh Docker ~/MyImages
#     ./build_speed.sh Both /usr/share/fedora-dockerfiles/django/Dockerfile

#totalsfile=$(mktemp /tmp/buildspeedXXX.json)
totalsfile=/tmp/build_speed.json
commaDockerfile=""

echo -n '{' > $totalsfile
Dockerfiles() {
    find -L $1 -name Dockerfile\*
}

Buildah() {
    Name=$1
    Dockerfile=$2
    Context=$3
    echo buildah bud --layers -f ${Dockerfile} -t ${Name} ${Context}
    Time buildah bud --layers -f ${Dockerfile} -t ${Name} ${Context}
}

Time() {
    outfile=$(mktemp /tmp/buildspeedXXX)
    /usr/bin/time -o $outfile --f "%E" $@
    echo "{\"engine\": \"$1\", \"command\": \"$@\", \"real\": \"$(cat ${outfile})\"}"
    echo -n "${comma}\"$1\": {\"command\": \"$@\", \"real\": \"$(cat ${outfile})\"}" >> $totalsfile
    comma=","
    rm -f $outfile
}

Docker() {
    Name=$1
    Dockerfile=$2
    Context=$3
    echo docker build -f ${Dockerfile} -t ${Name} ${Context}
    Time docker build -f ${Dockerfile} -t ${Name} ${Context}
}

Both() {
    comma=""
    echo -n "${commaDockerfile}\"$2\": {" >> $totalsfile
    commaDockerfile=","
    Docker $1 $2 $3
    Buildah $1 $2 $3
    echo -n "}" >> $totalsfile
}

Docker_cleanup() {
    docker rmi --force $(docker images  -q) 
}

Buildah_cleanup() {
    buildah rmi --force --all
}

Both_cleanup() {
    Docker_cleanup
    Buildah_cleanup
}

Cmd=${1?Missing CMD argument}
Path=${2?Missing PATH argument}

case "$Cmd" in
    Docker)   ;;
    Buildah)  ;;
    Both)     ;;
    *)     echo "Invalid command '$Cmd'; must be Buildah, Docker, or Both"; exit 1;;
esac


if [ "$Path" == "cleanup" ]; then
    ${Cmd}_cleanup
    exit 0
fi

for i in $(Dockerfiles ${Path});do
    name=$(basename $(dirname $i) | sed -e 's/\(.*\)/\L\1/')
    name=${name}_$(basename $i | sed -e 's/\(.*\)/\L\1/')
    echo ${Cmd} ${name} $i $(dirname $i)
    ${Cmd} ${name} $i $(dirname $i)
done

echo '}'>>$totalsfile
echo cat $totalsfile
cat $totalsfile
