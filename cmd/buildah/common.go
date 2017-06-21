package main

import (
	"encoding/json"
	"os"
	"strings"
	"time"

	is "github.com/containers/image/storage"
	"github.com/containers/image/types"
	"github.com/containers/storage"
	"github.com/pkg/errors"
	"github.com/projectatomic/buildah"
	"github.com/urfave/cli"
)

type imageMetadata struct {
	Tag            string              `json:"tag"`
	CreatedTime    time.Time           `json:"created-time"`
	ID             string              `json:"id"`
	Blobs          []types.BlobInfo    `json:"blob-list"`
	Layers         map[string][]string `json:"layers"`
	SignatureSizes []string            `json:"signature-sizes"`
}

var needToShutdownStore = false

func getStore(c *cli.Context) (storage.Store, error) {
	options := storage.DefaultStoreOptions
	if c.GlobalIsSet("root") || c.GlobalIsSet("runroot") {
		options.GraphRoot = c.GlobalString("root")
		options.RunRoot = c.GlobalString("runroot")
	}
	if c.GlobalIsSet("storage-driver") {
		options.GraphDriverName = c.GlobalString("storage-driver")
	}
	if c.GlobalIsSet("storage-opt") {
		opts := c.GlobalStringSlice("storage-opt")
		if len(opts) > 0 {
			options.GraphDriverOptions = opts
		}
	}
	store, err := storage.GetStore(options)
	if store != nil {
		is.Transport.SetStore(store)
	}
	needToShutdownStore = true
	return store, err
}

func openBuilder(store storage.Store, name string) (builder *buildah.Builder, err error) {
	if name != "" {
		builder, err = buildah.OpenBuilder(store, name)
		if os.IsNotExist(err) {
			options := buildah.ImportOptions{
				Container: name,
			}
			builder, err = buildah.ImportBuilder(store, options)
		}
	}
	if err != nil {
		return nil, errors.Wrapf(err, "error reading build container")
	}
	if builder == nil {
		return nil, errors.Errorf("error finding build container")
	}
	return builder, nil
}

func openBuilders(store storage.Store) (builders []*buildah.Builder, err error) {
	return buildah.OpenAllBuilders(store)
}

func openImage(store storage.Store, name string) (builder *buildah.Builder, err error) {
	options := buildah.ImportFromImageOptions{
		Image: name,
	}
	builder, err = buildah.ImportBuilderFromImage(store, options)
	if err != nil {
		return nil, errors.Wrapf(err, "error reading image")
	}
	if builder == nil {
		return nil, errors.Errorf("error mocking up build configuration")
	}
	return builder, nil
}

func parseMetadata(image storage.Image) (imageMetadata, error) {
	var im imageMetadata

	dec := json.NewDecoder(strings.NewReader(image.Metadata))
	if err := dec.Decode(&im); err != nil {
		return imageMetadata{}, err
	}
	return im, nil
}

func getSize(image storage.Image, store storage.Store) (int64, error) {

	is.Transport.SetStore(store)
	storeRef, err := is.Transport.ParseStoreReference(store, "@"+image.ID)
	if err != nil {
		return -1, err
	}
	img, err := storeRef.NewImage(nil)
	if err != nil {
		return -1, err
	}
	imgSize, err := img.Size()
	if err != nil {
		return -1, err
	}
	return imgSize, nil
}
