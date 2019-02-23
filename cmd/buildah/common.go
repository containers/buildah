package main

import (
	"context"
	"os"
	"syscall"
	"time"

	"github.com/containers/buildah"
	"github.com/containers/buildah/util"
	is "github.com/containers/image/storage"
	"github.com/containers/image/types"
	lu "github.com/containers/libpod/pkg/util"
	"github.com/containers/storage"
	digest "github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var needToShutdownStore = false

func getStore(c *cobra.Command) (storage.Store, error) {
	options, _, err := lu.GetDefaultStoreOptions()
	if err != nil {
		return nil, err
	}
	if c.Flag("root").Changed || c.Flag("runroot").Changed {
		options.GraphRoot = globalFlagResults.Root
		options.RunRoot = globalFlagResults.RunRoot
	}
	if err := os.Setenv("XDG_RUNTIME_DIR", options.RunRoot); err != nil {
		return nil, errors.New("could not set XDG_RUNTIME_DIR")
	}

	if c.Flag("storage-driver").Changed {
		options.GraphDriverName = globalFlagResults.StorageDriver
		// If any options setup in config, these should be dropped if user overrode the driver
		options.GraphDriverOptions = []string{}
	}
	if c.Flag("storage-opt").Changed {
		if len(globalFlagResults.StorageOpts) > 0 {
			options.GraphDriverOptions = globalFlagResults.StorageOpts
		}
	}

	// For uid/gid mappings, first we check the global definitions
	if len(globalFlagResults.UserNSUID) > 0 || len(globalFlagResults.UserNSGID) > 0 {
		if !(len(globalFlagResults.UserNSUID) > 0 && len(globalFlagResults.UserNSGID) > 0) {
			return nil, errors.Errorf("--userns-uid-map and --userns-gid-map must be used together")
		}
		uopts := globalFlagResults.UserNSUID
		gopts := globalFlagResults.UserNSGID
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
	}

	// If a subcommand has the flags, check if they are set; if so, override the global values
	localUIDMapFlag := c.Flags().Lookup("userns-uid-map")
	localGIDMapFlag := c.Flags().Lookup("userns-gid-map")
	if localUIDMapFlag != nil && localGIDMapFlag != nil && (localUIDMapFlag.Changed || localGIDMapFlag.Changed) {
		if !(localUIDMapFlag.Changed && localGIDMapFlag.Changed) {
			return nil, errors.Errorf("--userns-uid-map and --userns-gid-map must be used together")
		}
		// We know that the flags are both !nil and have been changed (i.e. have values)
		uopts, _ := c.Flags().GetStringSlice("userns-uid-map")
		gopts, _ := c.Flags().GetStringSlice("userns-gid-map")
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
	}

	oldUmask := syscall.Umask(0022)
	if (oldUmask & ^0022) != 0 {
		logrus.Debugf("umask value too restrictive.  Forcing it to 022")
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

func getUserFlags() pflag.FlagSet {
	fs := pflag.FlagSet{}
	fs.String("user", "", "`user[:group]` to run the command as")
	return fs
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

func getFormat(format string) (string, error) {
	switch format {
	case buildah.OCI:
		return buildah.OCIv1ImageManifest, nil
	case buildah.DOCKER:
		return buildah.Dockerv2ImageManifest, nil
	default:
		return "", errors.Errorf("unrecognized image type %q", format)
	}
}

// Tail returns a string slice after the first element unless there are
// not enough elements, then it returns an empty slice.  This is to replace
// the urfavecli Tail method for args
func Tail(a []string) []string {
	if len(a) >= 2 {
		return a[1:]
	}
	return []string{}
}

// UsageTemplate returns the usage template for podman commands
// This blocks the desplaying of the global options. The main podman
// command should not use this.
func UsageTemplate() string {
	return `Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}

  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
  {{.Example}}{{end}}{{if .HasAvailableSubCommands}}

Available Commands:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}
{{end}}
`
}
