![buildah logo](../../logos/buildah-logo_large.png)

# Buildah Tutorial 4
## Include Buildah in your build tool

The purpose of this tutorial is to demonstrate how to include Buildah as a library in your build tool.

You can take advantage of all features provided by Buildah, like using Dockerfiles and building using rootless mode.

In this tutorial I'll show you how to create a simple CLI tool that creates an image containing NodeJS and a JS main file.

## Bootstrap the project and install the dependencies

Bootstrap the installation of development dependencies of Buildah by following the [Building from scratch](https://github.com/slinkydeveloper/buildah/blob/master/install.md#building-from-scratch) instructions and in particular creating a directory for the Buildah project by completing the instructions in the [Installation from GitHub](https://github.com/containers/buildah/blob/master/install.md#installation-from-github) section of that page.

Now let's bootstrap our project. Assuming you are in the directory of the project, run:


```shell
go mod init
```

To initialize the go modules. Now import Buildah as a dependency:

```shell
go get github.com/containers/buildah
```

## Build the image

Now you can develop your application. To access to the build features of Buildah, you need to instantiate `buildah.Builder`. This struct has methods to configure the build, define the build steps and run it.

To instantiate a `Builder`, you need a `storage.Store` (the Store interface found in [store.go](https://github.com/containers/storage/blob/master/store.go)) from [`github.com/containers/storage`](https://github.com/containers/storage), where the intermediate and result images will be stored:

```go
buildStoreOptions, err := storage.DefaultStoreOptions(unshare.IsRootless(), unshare.GetRootlessUID())
buildStore, err := storage.GetStore(buildStoreOptions)
```

Define the builder options:

```go
builderOpts := buildah.BuilderOptions{
    FromImage:        "node:12-alpine", // Starting image
    Isolation:        buildah.IsolationChroot, // Isolation environment
    CommonBuildOpts:  &buildah.CommonBuildOptions{},
    ConfigureNetwork: buildah.NetworkDefault,
    SystemContext: 	  &types.SystemContext {},
}
```

Now instantiate the `Builder`:

```go
builder, err := buildah.NewBuilder(context.TODO(), buildStore, builderOpts)
```

Let's add our JS file (assuming is in your local directory with name `script.js`):

```go
err = builder.Add("/home/node/", false, buildah.AddAndCopyOptions{}, "script.js")
```

And configure the command to run:

```go
builder.SetCmd([]string{"node", "/home/node/script.js"})
```

Before completing the build, create the image reference:

```go
imageRef, err := alltransports.ParseImageName("docker.io/myusername/my-image")
```

Now you can run commit the build:

```go
imageId, _, _, err := builder.Commit(context.TODO(), imageRef, buildah.CommitOptions{})
```

## Rootless mode

To enable rootless mode, import `github.com/containers/buildah/pkg/unshare` and add this code at the beginning of your main method:

```go
if buildah.InitReexec() {
    return
}
unshare.MaybeReexecUsingUserNamespace(false)
```

This code ensures that your application is re executed in an isolated environment with root privileges.

## Complete code

```go
package main

import (
	"context"
	"fmt"
	"github.com/containers/buildah"
	"github.com/containers/buildah/pkg/unshare"
	"github.com/containers/image/v4/transports/alltransports"
	"github.com/containers/image/v4/types"
	"github.com/containers/storage"
)

func main() {
	if buildah.InitReexec() {
		return
	}
	unshare.MaybeReexecUsingUserNamespace(false)

	buildStoreOptions, err := storage.DefaultStoreOptions(unshare.IsRootless(), unshare.GetRootlessUID())

	if err != nil {
		panic(err)
	}

	buildStore, err := storage.GetStore(buildStoreOptions)

	if err != nil {
		panic(err)
	}

	opts := buildah.BuilderOptions{
		FromImage:        "node:12-alpine",
		Isolation:        buildah.IsolationChroot,
		CommonBuildOpts:  &buildah.CommonBuildOptions{},
		ConfigureNetwork: buildah.NetworkDefault,
		SystemContext: 	  &types.SystemContext {},
	}

	builder, err := buildah.NewBuilder(context.TODO(), buildStore, opts)

	if err != nil {
		panic(err)
	}

	err = builder.Add("/home/node/", false, buildah.AddAndCopyOptions{}, "script.js")

	if err != nil {
		panic(err)
	}

	builder.SetCmd([]string{"node", "/home/node/script.js"})

	imageRef, err := alltransports.ParseImageName("docker.io/myusername/my-image")

	imageId, _, _, err := builder.Commit(context.TODO(), imageRef, buildah.CommitOptions{})

	fmt.Printf("Image built! %s\n", imageId)
}
```
