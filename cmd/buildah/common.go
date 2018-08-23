package main

import (
	"context"
	"os"
	"strings"
	"time"

	is "github.com/containers/image/storage"
	"github.com/containers/image/types"
	"github.com/containers/storage"
	digest "github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
	"github.com/projectatomic/buildah"
	"github.com/projectatomic/buildah/util"
	"github.com/urfave/cli"
)

var needToShutdownStore = false

func getStore(c *cli.Context) (storage.Store, error) {
	options := storage.DefaultStoreOptions
	if c.GlobalIsSet("root") || c.GlobalIsSet("runroot") {
		options.GraphRoot = c.GlobalString("root")
		options.RunRoot = c.GlobalString("runroot")
	}
	if c.GlobalIsSet("storage-driver") {
		options.GraphDriverName = c.GlobalString("storage-driver")
		// If any options setup in config, these should be dropped if user overrode the driver
		options.GraphDriverOptions = []string{}
	}
	if c.GlobalIsSet("storage-opt") {
		opts := c.GlobalStringSlice("storage-opt")
		if len(opts) > 0 {
			options.GraphDriverOptions = opts
		}
	}
	if c.GlobalIsSet("userns-uid-map") && c.GlobalIsSet("userns-gid-map") {
		uopts := c.GlobalStringSlice("userns-uid-map")
		gopts := c.GlobalStringSlice("userns-gid-map")
		if len(uopts) == 0 {
			return nil, errors.New("--userns-uid-map used with no mappings?")
		}
		if len(gopts) == 0 {
			return nil, errors.New("--userns-gid-map used with no mappings?")
		}
		uidmap, gidmap, err := util.ParseIDMappings(uopts, gopts)
		if err != nil {
			return nil, err
		}
		options.UIDMap = uidmap
		options.GIDMap = gidmap
	} else if c.GlobalIsSet("userns-uid-map") {
		return nil, errors.Errorf("--userns-uid-map requires --userns-gid-map")
	} else if c.GlobalIsSet("userns-gid-map") {
		return nil, errors.Errorf("--userns-gid-map requires --userns-uid-map")
	}
	store, err := storage.GetStore(options)
	if store != nil {
		is.Transport.SetStore(store)
	}
	needToShutdownStore = true
	return store, err
}

func openBuilder(ctx context.Context, store storage.Store, name string) (builder *buildah.Builder, err error) {
	if name != "" {
		builder, err = buildah.OpenBuilder(store, name)
		if os.IsNotExist(err) {
			options := buildah.ImportOptions{
				Container: name,
			}
			builder, err = buildah.ImportBuilder(ctx, store, options)
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

func openImage(ctx context.Context, sc *types.SystemContext, store storage.Store, name string) (builder *buildah.Builder, err error) {
	options := buildah.ImportFromImageOptions{
		Image:         name,
		SystemContext: sc,
	}
	builder, err = buildah.ImportBuilderFromImage(ctx, store, options)
	if err != nil {
		return nil, errors.Wrapf(err, "error reading image")
	}
	if builder == nil {
		return nil, errors.Errorf("error mocking up build configuration")
	}
	return builder, nil
}

func getDateAndDigestAndSize(ctx context.Context, image storage.Image, store storage.Store) (time.Time, string, int64, error) {
	created := time.Time{}
	is.Transport.SetStore(store)
	storeRef, err := is.Transport.ParseStoreReference(store, image.ID)
	if err != nil {
		return created, "", -1, err
	}
	img, err := storeRef.NewImage(ctx, nil)
	if err != nil {
		return created, "", -1, err
	}
	defer img.Close()
	imgSize, sizeErr := img.Size()
	if sizeErr != nil {
		imgSize = -1
	}
	manifest, _, manifestErr := img.Manifest(ctx)
	manifestDigest := ""
	if manifestErr == nil && len(manifest) > 0 {
		manifestDigest = digest.Canonical.FromBytes(manifest).String()
	}
	inspectInfo, inspectErr := img.Inspect(ctx)
	if inspectErr == nil && inspectInfo != nil {
		created = *inspectInfo.Created
	}
	if sizeErr != nil {
		err = sizeErr
	} else if manifestErr != nil {
		err = manifestErr
	} else if inspectErr != nil {
		err = inspectErr
	}
	return created, manifestDigest, imgSize, err
}

// getContext returns a context.TODO
func getContext() context.Context {
	return context.TODO()
}

var userFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "user",
		Usage: "`user[:group]` to run the command as",
	},
}

func defaultFormat() string {
	format := os.Getenv("BUILDAH_FORMAT")
	if format != "" {
		return format
	}
	return buildah.OCI
}

// imageIsParent goes through the layers in the store and checks if i.TopLayer is
// the parent of any other layer in store. Double check that image with that
// layer exists as well.
func imageIsParent(store storage.Store, topLayer string) (bool, error) {
	children, err := getChildren(store, topLayer)
	if err != nil {
		return false, err
	}
	return len(children) > 0, nil
}

// getParent returns the image ID of the parent. Return nil if a parent is not found.
func getParent(store storage.Store, topLayer string) (*storage.Image, error) {
	images, err := store.Images()
	if err != nil {
		return nil, errors.Wrapf(err, "unable to retrieve images from store")
	}
	layer, err := store.Layer(topLayer)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to retrieve layers from store")
	}
	for _, img := range images {
		if img.TopLayer == layer.Parent {
			return &img, nil
		}
	}
	return nil, nil
}

// getChildren returns a list of the imageIDs that depend on the image
func getChildren(store storage.Store, topLayer string) ([]string, error) {
	var children []string
	images, err := store.Images()
	if err != nil {
		return nil, errors.Wrapf(err, "unable to retrieve images from store")
	}
	layers, err := store.Layers()
	if err != nil {
		return nil, errors.Wrapf(err, "unable to retrieve layers from store")
	}

	for _, layer := range layers {
		if layer.Parent == topLayer {
			if imageID := getImageOfTopLayer(images, layer.ID); len(imageID) > 0 {
				children = append(children, imageID...)
			}
		}
	}
	return children, nil
}

// getImageOfTopLayer returns the image ID where layer is the top layer of the image
func getImageOfTopLayer(images []storage.Image, layer string) []string {
	var matches []string
	for _, img := range images {
		if img.TopLayer == layer {
			matches = append(matches, img.ID)
		}
	}
	return matches
}

func getFormat(c *cli.Context) (string, error) {
	format := strings.ToLower(c.String("format"))
	if strings.HasPrefix(format, buildah.OCI) {
		return buildah.OCIv1ImageManifest, nil
	}

	if strings.HasPrefix(format, buildah.DOCKER) {
		return buildah.Dockerv2ImageManifest, nil
	}
	return "", errors.Errorf("unrecognized image type %q", format)
}
