package main

import (
	"fmt"
	"os"

	is "github.com/containers/image/storage"
	"github.com/containers/storage/storage"
	"github.com/nalind/buildah"
	"github.com/urfave/cli"
)

func getStore(c *cli.Context) (storage.Store, error) {
	options := storage.DefaultStoreOptions
	if c.GlobalIsSet("root") || c.GlobalIsSet("runroot") {
		options.GraphRoot = c.GlobalString("root")
		options.RunRoot = c.GlobalString("runroot")
	}
	if c.GlobalIsSet("storage-driver") {
		options.GraphDriverName = c.GlobalString("storage-driver")
	}
	if c.GlobalIsSet("storage-options") {
		opts := c.GlobalStringSlice("storage-options")
		if len(opts) > 0 {
			options.GraphDriverOptions = opts
		}
	}
	store, err := storage.GetStore(options)
	if store != nil {
		is.Transport.SetStore(store)
	}
	return store, err
}

func openBuilder(store storage.Store, name, root, link string) (builder *buildah.Builder, err error) {
	if name != "" {
		builder, err = buildah.OpenBuilder(store, name)
		if os.IsNotExist(err) {
			options := buildah.ImportOptions{
				Container: name,
			}
			builder, err = buildah.ImportBuilder(store, options)
		}
	}
	if root != "" {
		builder, err = buildah.OpenBuilderByPath(store, root)
	}
	if link != "" {
		builder, err = buildah.OpenBuilderByPath(store, link)
	}
	if err != nil {
		return nil, fmt.Errorf("error reading build container: %v", err)
	}
	if builder == nil {
		return nil, fmt.Errorf("error finding build container")
	}
	return builder, nil
}
