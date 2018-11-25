# buildah-info "1" "November 2018" "Buildah"

## NAME
buildah\-info - Display Buildah system information.

## SYNOPSIS
**buildah info** [*options*]

## DESCRIPTION
The information displayed pertains to the host and current storage statistics which is useful when reporting issues.

## OPTIONS
**--debug, -D**
Show additional information.

**--format** *template*

Use *template* as a Go template when formatting the output.

## EXAMPLE
Run buildah info response:
```
$ buildah info
{
    "host": {
        "Distribution": {
            "distribution": "ubuntu",
            "version": "18.04"
        },
        "MemTotal": 16702980096,
        "MenFree": 309428224,
        "SwapFree": 2146693120,
        "SwapTotal": 2147479552,
        "arch": "amd64",
        "cpus": 4,
        "hostname": "localhost.localdomain",
        "kernel": "4.15.0-36-generic",
        "os": "linux",
        "rootless": false,
        "uptime": "91h 30m 59.9s (Approximately 3.79 days)"
    },
    "store": {
        "ContainerStore": {
            "number": 2
        },
        "GraphDriverName": "overlay",
        "GraphOptions": [
            "overlay.override_kernel_check=true"
        ],
        "GraphRoot": "/var/lib/containers/storage",
        "GraphStatus": {
            "Backing Filesystem": "extfs",
            "Native Overlay Diff": "true",
            "Supports d_type": "true"
        },
        "ImageStore": {
            "number": 1
        },
        "RunRoot": "/var/run/containers/storage"
    }
}
```

Run buildah info and retrieve only the store information:
```
$ buildah info --format={{".store"}}
map[GraphOptions:[overlay.override_kernel_check=true] GraphStatus:map[Backing Filesystem:extfs Supports d_type:true Native Overlay Diff:true] ImageStore:map[number:1] ContainerStore:map[number:2] GraphRoot:/var/lib/containers/storage RunRoot:/var/run/containers/storage GraphDriverName:overlay]
```

## SEE ALSO
buildah(1)
