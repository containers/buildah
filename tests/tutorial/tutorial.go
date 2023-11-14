package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/containers/buildah"
	"github.com/containers/buildah/define"
	"github.com/containers/buildah/imagebuildah"
	"github.com/containers/storage"
	"github.com/containers/storage/pkg/unshare"
	specs "github.com/opencontainers/runtime-spec/specs-go"
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

	buildStore, err := storage.GetStore(buildStoreOptions)
	if err != nil {
		panic(err)
	}
	defer func() {
		if _, err := buildStore.Shutdown(false); err != nil {
			if !errors.Is(err, storage.ErrLayerUsedByContainer) {
				fmt.Printf("failed to shutdown storage: %q", err)
			}
		}
	}()

	d, err := os.MkdirTemp("", "")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(d)
	dockerfile := filepath.Join(d, "Dockerfile")
	f, err := os.Create(dockerfile)
	if err != nil {
		panic(err)
	}
	fmt.Fprintf(f, "FROM quay.io/libpod/alpine\nRUN echo CUT START; find /sys/fs/cgroup -print | sort ; echo CUT END")
	f.Close()

	buildOptions := define.BuildOptions{
		ContextDirectory: d,
		NamespaceOptions: []define.NamespaceOption{
			{Name: string(specs.NetworkNamespace), Host: true},
		},
	}

	_, _, err = imagebuildah.BuildDockerfiles(context.TODO(), buildStore, buildOptions, dockerfile)
	if err != nil {
		panic(err)
	}
}
