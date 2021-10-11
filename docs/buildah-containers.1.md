# buildah-containers "1" "March 2017" "buildah"

## NAME
buildah\-containers - List the working containers and their base images.

## SYNOPSIS
**buildah containers** [*options*]

## DESCRIPTION
Lists containers which appear to be Buildah working containers, their names and
IDs, and the names and IDs of the images from which they were initialized.

## OPTIONS

**--all**, **-a**

List information about all containers, including those which were not created
by and are not being used by Buildah.  Containers created by Buildah are
denoted with an '*' in the 'BUILDER' column.

**--filter**, **-f**

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

**--noheading**, **-n**

Omit the table headings from the listing of containers.

**--notruncate**

Do not truncate IDs and image names in the output.

**--quiet**, **-q**

Displays only the container IDs.

## EXAMPLE

buildah containers
```
CONTAINER ID  BUILDER  IMAGE ID     IMAGE NAME                       CONTAINER NAME
ccf84de04b80     *     53ce4390f2ad registry.access.redhat.com/ub... ubi8-working-container
45be1d806fc5     *     16ea53ea7c65 docker.io/library/busybox:latest busybox-working-container
```

buildah containers --quiet
```
ccf84de04b80c309ce6586997c79a769033dc4129db903c1882bc24a058438b8
45be1d806fc533fcfc2beee77e424d87e5990d3ce9214d6b374677d6630bba07
```

buildah containers -q --noheading --notruncate
```
ccf84de04b80c309ce6586997c79a769033dc4129db903c1882bc24a058438b8
45be1d806fc533fcfc2beee77e424d87e5990d3ce9214d6b374677d6630bba07
```

buildah containers --json
```
[
    {
        "id": "ccf84de04b80c309ce6586997c79a769033dc4129db903c1882bc24a058438b8",
        "builder": true,
        "imageid": "53ce4390f2adb1681eb1a90ec8b48c49c015e0a8d336c197637e7f65e365fa9e",
        "imagename": "registry.access.redhat.com/ubi8:latest",
        "containername": "ubi8-working-container"
    },
    {
        "id": "45be1d806fc533fcfc2beee77e424d87e5990d3ce9214d6b374677d6630bba07",
        "builder": true,
        "imageid": "16ea53ea7c652456803632d67517b78a4f9075a10bfdc4fc6b7b4cbf2bc98497",
        "imagename": "docker.io/library/busybox:latest",
        "containername": "busybox-working-container"
    }
]
```

buildah containers --format "{{.ContainerID}} {{.ContainerName}}"
```
ccf84de04b80c309ce6586997c79a769033dc4129db903c1882bc24a058438b8   ubi8-working-container
45be1d806fc533fcfc2beee77e424d87e5990d3ce9214d6b374677d6630bba07   busybox-working-container
```

buildah containers --format "Container ID: {{.ContainerID}}"
```
Container   ID:   ccf84de04b80c309ce6586997c79a769033dc4129db903c1882bc24a058438b8
Container   ID:   45be1d806fc533fcfc2beee77e424d87e5990d3ce9214d6b374677d6630bba07
```

buildah containers --filter ancestor=ubuntu
```
CONTAINER ID  BUILDER  IMAGE ID     IMAGE NAME                       CONTAINER NAME
fbfd3505376e     *     0ff04b2e7b63 docker.io/library/ubuntu:latest  ubuntu-working-container
```

## SEE ALSO
buildah(1)
