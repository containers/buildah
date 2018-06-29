# buildah-containers "1" "March 2017" "buildah"

## NAME
buildah\-containers - List the working containers and their base images.

## SYNOPSIS
**buildah containers** [*options*]

## DESCRIPTION
Lists containers which appear to be Buildah working containers, their names and
IDs, and the names and IDs of the images from which they were initialized.

## OPTIONS

**--all, -a**

List information about all containers, including those which were not created
by and are not being used by Buildah.  Containers created by Buildah are
denoted with an '*' in the 'BUILDER' column.

**--filter, -f**

Filter output based on conditions provided.

Valid filters are listed below:

| **Filter**      | **Description**                                                     |
| --------------- | ------------------------------------------------------------------- |
| id              | [ID] Container's ID                                                 |
| name            | [Name] Container's name                                             |
| ancestor        | [ImageName] Image or descendant used to create container            |

**--format**

Pretty-print containers using a Go template.

Valid placeholders for the Go template are listed below:

| **Placeholder** | **Description**                          |
| --------------- | -----------------------------------------|
| .ContainerID    | Container ID                             |
| .Builder        | Whether container was created by buildah |
| .ImageID        | Image ID                                 |
| .ImageName      | Image name                               |
| .ContainerName  | Container name                           |

**--json**

Output in JSON format.

**--noheading, -n**

Omit the table headings from the listing of containers.

**--notruncate**

Do not truncate IDs in output.

**--quiet, -q**

Displays only the container IDs.

## EXAMPLE

buildah containers
```
CONTAINER ID  BUILDER  IMAGE ID     IMAGE NAME                       CONTAINER NAME
29bdb522fc62     *     3fd9065eaf02 docker.io/library/alpine:latest  alpine-working-container
c6b04237ac8e     *     f9b6f7f7b9d3 docker.io/library/busybox:latest busybox-working-container
```

buildah containers --quiet
```
29bdb522fc62d43fca0c1a0f11cfc6dfcfed169cf6cf25f928ebca1a612ff5b0
c6b04237ac8e9d435ec9cf0e7eda91e302f2db9ef908418522c2d666352281eb
```

buildah containers -q --noheading --notruncate
```
29bdb522fc62d43fca0c1a0f11cfc6dfcfed169cf6cf25f928ebca1a612ff5b0
c6b04237ac8e9d435ec9cf0e7eda91e302f2db9ef908418522c2d666352281eb
```

buildah containers --json
```
[
    {
        "id": "29bdb522fc62d43fca0c1a0f11cfc6dfcfed169cf6cf25f928ebca1a612ff5b0",
        "builder": true,
        "imageid": "3fd9065eaf02feaf94d68376da52541925650b81698c53c6824d92ff63f98353",
        "imagename": "docker.io/library/alpine:latest",
        "containername": "alpine-working-container"
    },
    {
        "id": "c6b04237ac8e9d435ec9cf0e7eda91e302f2db9ef908418522c2d666352281eb",
        "builder": true,
        "imageid": "f9b6f7f7b9d34113f66e16a9da3e921a580937aec98da344b852ca540aaa2242",
        "imagename": "docker.io/library/busybox:latest",
        "containername": "busybox-working-container"
    }
]
```

buildah containers --format "{{.ContainerID}} {{.ContainerName}}"
```
3fbeaa87e583ee7a3e6787b2d3af961ef21946a0c01a08938e4f52d53cce4c04 myalpine-working-container
fbfd3505376ee639c3ed50f9d32b78445cd59198a1dfcacf2e7958cda2516d5c ubuntu-working-container
```

buildah containers --format "Container ID: {{.ContainerID}}"
```
Container ID: 3fbeaa87e583ee7a3e6787b2d3af961ef21946a0c01a08938e4f52d53cce4c04
Container ID: fbfd3505376ee639c3ed50f9d32b78445cd59198a1dfcacf2e7958cda2516d5c
```

buildah containers --filter ancestor=ubuntu
```
CONTAINER ID  BUILDER  IMAGE ID     IMAGE NAME                       CONTAINER NAME
fbfd3505376e     *     0ff04b2e7b63 docker.io/library/ubuntu:latest  ubuntu-working-container
```

## SEE ALSO
buildah(1)

