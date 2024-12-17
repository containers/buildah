![buildah logo](../../logos/buildah-logo_large.png)

# Buildah Tutorial 4

## Include Buildah in your build tool

The purpose of this tutorial is to demonstrate how to include Buildah as a library in your build tool.

You can take advantage of all features provided by Buildah, like using Dockerfiles and building using rootless mode.

In this tutorial I'll show you how to create a simple CLI tool that creates an image containing NodeJS and a JS main file.

## Bootstrap the project and install the dependencies

Bootstrap the installation of development dependencies of Buildah by following the [Building from scratch](https://github.com/slinkydeveloper/buildah/blob/main/install.md#building-from-scratch) instructions and in particular creating a directory for the Buildah project by completing the instructions in the [Installation from GitHub](https://github.com/containers/buildah/blob/main/install.md#installation-from-github) section of that page.

Now let's bootstrap our project. Assuming you are in the directory of the project, run the following to initialize the go modules:

```shell
go mod init
```

Next, we should import Buildah as a dependency. However, make sure that you have the following
developer packages installed:

```shell
dnf install btrfs-progs-devel gpgme-devel passt
```

Depending on your Linux distribution, the names of the packages can be slightly different. For instance, on
OpenSUSE it would be

```shell
zypper install libbtrfs-devel libgpgme-devel passt
```

On Debian and Ubuntu, it would be

```shell
apt install libbtrfs-dev libgpgme-dev passt
```

Now import Buildah as a dependency:

```shell
go get github.com/containers/buildah
```

## Build the image

Now you can develop your application. To access to the build features of Buildah, you need to instantiate a `buildah.Builder`. This struct has methods to configure the build, define the build steps and run it.

To instantiate a `Builder`, you need a `storage.Store` (the Store interface found in [store.go](https://github.com/containers/storage/blob/main/store.go)) from [`github.com/containers/storage`](https://github.com/containers/storage), where the intermediate and result images will be stored:

```go
buildStoreOptions, err := storage.DefaultStoreOptions()
buildStore, err := storage.GetStore(buildStoreOptions)
```

Define the builder options:

```go
builderOpts := buildah.BuilderOptions{
  FromImage: "docker.io/library/node:12-alpine", // base image
}
```

Now instantiate the `Builder`:

```go
builder, err := buildah.NewBuilder(context.TODO(), buildStore, builderOpts)
```

Let's add our JS file (assuming it is in your local directory with name `script.js`):

```go
err = builder.Add("/home/node/", false, buildah.AddAndCopyOptions{}, "script.js")
```

And configure the command to run:

```go
builder.SetCmd([]string{"node", "/home/node/script.js"})
```

Before completing the build, create the image reference:

```go
imageRef, err := is.Transport.ParseStoreReference(buildStore, "docker.io/myusername/my-image")
```

Now you can commit the container to create the image:

```go
imageId, _, _, err := builder.Commit(context.TODO(), imageRef, buildah.CommitOptions{})
```

## Supplying defaults for Run()

If you need to run a command as part of the build, you'll have to dig up a couple of defaults that aren't picked up automatically:
```go
conf, err := config.Default()
capabilitiesForRoot, err := conf.Capabilities("root", nil, nil)
isolation, err := parse.IsolationOption("")
```

## Rootless mode

To enable rootless mode, import `github.com/containers/storage/pkg/unshare` and add this code at the beginning of your main() method:

```go
if buildah.InitReexec() {
  return
}
unshare.MaybeReexecUsingUserNamespace(false)
```

This code ensures that your application is re-executed in a user namespace where it has root privileges. The reexec mechanism which `buildah.InitReexec()` sets up is also used for various other internal functions, so it should always be called.

## Complete code

```go
package main

import (
  "context"
  "fmt"

  "github.com/containers/buildah"
  "github.com/containers/buildah/pkg/parse"
  "github.com/containers/common/pkg/config"
  is "github.com/containers/image/v5/storage"
  "github.com/containers/storage"
  "github.com/containers/storage/pkg/unshare"
)

func main() {
  if buildah.InitReexec() {
    return
  }
  unshare.MaybeReexecUsingUserNamespace(false)

  buildStoreOptions, err := storage.DefaultStoreOptions()
  if err != nil {
    panic(err)
  }

  conf, err := config.Default()
  if err != nil {
    panic(err)
  }
  capabilitiesForRoot, err := conf.Capabilities("root", nil, nil)
  if err != nil {
    panic(err)
  }

  buildStore, err := storage.GetStore(buildStoreOptions)
  if err != nil {
    panic(err)
  }
  defer buildStore.Shutdown(false)

  builderOpts := buildah.BuilderOptions{
    FromImage:        "docker.io/library/node:12-alpine",
    Capabilities:     capabilitiesForRoot,
  }

  builder, err := buildah.NewBuilder(context.TODO(), buildStore, builderOpts)
  if err != nil {
    panic(err)
  }
  defer builder.Delete()

  err = builder.Add("/home/node/", false, buildah.AddAndCopyOptions{}, "script.js")
  if err != nil {
    panic(err)
  }

  isolation, err := parse.IsolationOption("")
  if err != nil {
    panic(err)
  }

  err = builder.Run([]string{"sh", "-c", "date > /home/node/build-date.txt"}, buildah.RunOptions{Isolation: isolation, Terminal: buildah.WithoutTerminal})
  if err != nil {
    panic(err)
  }

  builder.SetCmd([]string{"node", "/home/node/script.js"})

  imageRef, err := is.Transport.ParseStoreReference(buildStore, "docker.io/myusername/my-image")
  if err != nil {
    panic(err)
  }

  imageId, _, _, err := builder.Commit(context.TODO(), imageRef, buildah.CommitOptions{})
  if err != nil {
    panic(err)
  }

  fmt.Printf("Image built! %s\n", imageId)
}
```
