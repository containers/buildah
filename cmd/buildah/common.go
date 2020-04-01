package main

import (
	"context"
	"os"
	"time"

	"github.com/containers/buildah"
	"github.com/containers/buildah/pkg/umask"
	"github.com/containers/image/v5/image"
	"github.com/containers/image/v5/manifest"
	is "github.com/containers/image/v5/storage"
	"github.com/containers/image/v5/types"
	encconfig "github.com/containers/ocicrypt/config"
	enchelpers "github.com/containers/ocicrypt/helpers"
	"github.com/containers/storage"
	"github.com/containers/storage/pkg/unshare"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	// configuration, including customizations made in containers.conf
	needToShutdownStore = false
)

const (
	maxPullPushRetries = 3
	pullPushRetryDelay = 2 * time.Second
)

func getStore(c *cobra.Command) (storage.Store, error) {
	options, err := storage.DefaultStoreOptions(unshare.IsRootless(), unshare.GetRootlessUID())
	if err != nil {
		return nil, err
	}
	if c.Flag("root").Changed || c.Flag("runroot").Changed {
		options.GraphRoot = globalFlagResults.Root
		options.RunRoot = globalFlagResults.RunRoot
	}

	if err := setXDGRuntimeDir(); err != nil {
		return nil, err
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

	// Do not allow to mount a graphdriver that is not vfs if we are creating the userns as part
	// of the mount command.
	// Differently, allow the mount if we are already in a userns, as the mount point will still
	// be accessible once "buildah mount" exits.
	if os.Geteuid() != 0 && options.GraphDriverName != "vfs" {
		return nil, errors.Errorf("cannot mount using driver %s in rootless mode. You need to run it in a `buildah unshare` session", options.GraphDriverName)
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
		uidmap, gidmap, err := unshare.ParseIDMappings(uopts, gopts)
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
		uidmap, gidmap, err := unshare.ParseIDMappings(uopts, gopts)
		if err != nil {
			return nil, err
		}
		options.UIDMap = uidmap
		options.GIDMap = gidmap
	}

	umask.CheckUmask()

	store, err := storage.GetStore(options)
	if store != nil {
		is.Transport.SetStore(store)
	}
	needToShutdownStore = true
	return store, err
}

// setXDGRuntimeDir sets XDG_RUNTIME_DIR when if it is unset under rootless
func setXDGRuntimeDir() error {
	if unshare.IsRootless() && os.Getenv("XDG_RUNTIME_DIR") == "" {
		runtimeDir, err := storage.GetRootlessRuntimeDir(unshare.GetRootlessUID())
		if err != nil {
			return err
		}
		if err := os.Setenv("XDG_RUNTIME_DIR", runtimeDir); err != nil {
			return errors.New("could not set XDG_RUNTIME_DIR")
		}
	}
	return nil
}

func openBuilder(ctx context.Context, store storage.Store, name string) (builder *buildah.Builder, err error) {
	if name != "" {
		builder, err = buildah.OpenBuilder(store, name)
		if os.IsNotExist(errors.Cause(err)) {
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

func getDateAndDigestAndSize(ctx context.Context, sys *types.SystemContext, store storage.Store, storeImage storage.Image) (time.Time, string, int64, error) {
	created := time.Time{}
	is.Transport.SetStore(store)
	storeRef, err := is.Transport.ParseStoreReference(store, storeImage.ID)
	if err != nil {
		return created, "", -1, err
	}
	img, err := storeRef.NewImageSource(ctx, nil)
	if err != nil {
		return created, "", -1, err
	}
	defer img.Close()
	imgSize, sizeErr := store.ImageSize(storeImage.ID)
	if sizeErr != nil {
		imgSize = -1
	}
	manifestBytes, _, manifestErr := img.GetManifest(ctx, nil)
	manifestDigest := ""
	if manifestErr == nil && len(manifestBytes) > 0 {
		mDigest, err := manifest.Digest(manifestBytes)
		manifestErr = err
		if manifestErr == nil {
			manifestDigest = mDigest.String()
		}
	}
	inspectable, inspectableErr := image.FromUnparsedImage(ctx, sys, image.UnparsedInstance(img, nil))
	if inspectableErr == nil && inspectable != nil {
		inspectInfo, inspectErr := inspectable.Inspect(ctx)
		if inspectErr == nil && inspectInfo != nil && inspectInfo.Created != nil {
			created = *inspectInfo.Created
		}
	}
	if sizeErr != nil {
		err = sizeErr
	} else if manifestErr != nil {
		err = manifestErr
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
func imageIsParent(ctx context.Context, sc *types.SystemContext, store storage.Store, image *storage.Image) (bool, error) {
	children, err := getChildren(ctx, sc, store, image, 1)
	if err != nil {
		return false, err
	}
	return len(children) > 0, nil
}

func getImageConfig(ctx context.Context, sc *types.SystemContext, store storage.Store, imageID string) (*imgspecv1.Image, error) {
	ref, err := is.Transport.ParseStoreReference(store, imageID)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to parse reference to image %q", imageID)
	}
	image, err := ref.NewImage(ctx, sc)
	if err != nil {
		if img, err2 := store.Image(imageID); err2 == nil && img.ID == imageID {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "unable to open image %q", imageID)
	}
	config, err := image.OCIConfig(ctx)
	defer image.Close()
	if err != nil {
		return nil, errors.Wrapf(err, "unable to read configuration from image %q", imageID)
	}
	return config, nil
}

func historiesDiffer(a, b []imgspecv1.History) bool {
	if len(a) != len(b) {
		return true
	}
	i := 0
	for i < len(a) {
		if a[i].Created == nil && b[i].Created != nil {
			break
		}
		if a[i].Created != nil && b[i].Created == nil {
			break
		}
		if a[i].Created != nil && b[i].Created != nil && !a[i].Created.Equal(*(b[i].Created)) {
			break
		}
		if a[i].CreatedBy != b[i].CreatedBy {
			break
		}
		if a[i].Author != b[i].Author {
			break
		}
		if a[i].Comment != b[i].Comment {
			break
		}
		if a[i].EmptyLayer != b[i].EmptyLayer {
			break
		}
		i++
	}
	return i != len(a)
}

// getParent returns the image's parent image. Return nil if a parent is not found.
func getParent(ctx context.Context, sc *types.SystemContext, store storage.Store, child *storage.Image) (*storage.Image, error) {
	images, err := store.Images()
	if err != nil {
		return nil, errors.Wrapf(err, "unable to retrieve image list from store")
	}
	var childTopLayer *storage.Layer
	if child.TopLayer != "" {
		childTopLayer, err = store.Layer(child.TopLayer)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to retrieve information about layer %s from store", child.TopLayer)
		}
	}
	childConfig, err := getImageConfig(ctx, sc, store, child.ID)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to read configuration from image %q", child.ID)
	}
	if childConfig == nil {
		return nil, nil
	}
	for _, parent := range images {
		if parent.ID == child.ID {
			continue
		}
		if childTopLayer != nil && parent.TopLayer != childTopLayer.Parent && parent.TopLayer != childTopLayer.ID {
			continue
		}
		parentConfig, err := getImageConfig(ctx, sc, store, parent.ID)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to read configuration from image %q", parent.ID)
		}
		if parentConfig == nil {
			continue
		}
		if len(parentConfig.History)+1 != len(childConfig.History) {
			continue
		}
		if len(parentConfig.RootFS.DiffIDs) > 0 {
			if len(childConfig.RootFS.DiffIDs) < len(parentConfig.RootFS.DiffIDs) {
				continue
			}
			childUsesAllParentLayers := true
			for i := range parentConfig.RootFS.DiffIDs {
				if childConfig.RootFS.DiffIDs[i] != parentConfig.RootFS.DiffIDs[i] {
					childUsesAllParentLayers = false
					break
				}
			}
			if !childUsesAllParentLayers {
				continue
			}
		}
		if historiesDiffer(parentConfig.History, childConfig.History[:len(parentConfig.History)]) {
			continue
		}
		return &parent, nil
	}
	return nil, nil
}

// getChildren returns a list of the imageIDs that depend on the image
func getChildren(ctx context.Context, sc *types.SystemContext, store storage.Store, parent *storage.Image, max int) ([]string, error) {
	var children []string
	images, err := store.Images()
	if err != nil {
		return nil, errors.Wrapf(err, "unable to retrieve images from store")
	}
	parentConfig, err := getImageConfig(ctx, sc, store, parent.ID)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to read configuration from image %q", parent.ID)
	}
	if parentConfig == nil {
		return nil, nil
	}
	for _, child := range images {
		if child.ID == parent.ID {
			continue
		}
		var childTopLayer *storage.Layer
		if child.TopLayer != "" {
			childTopLayer, err = store.Layer(child.TopLayer)
			if err != nil {
				return nil, errors.Wrapf(err, "unable to retrieve information about layer %q from store", child.TopLayer)
			}
			if childTopLayer.Parent != parent.TopLayer && childTopLayer.ID != parent.TopLayer {
				continue
			}
		}
		childConfig, err := getImageConfig(ctx, sc, store, child.ID)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to read configuration from image %q", child.ID)
		}
		if childConfig == nil {
			continue
		}
		if len(parentConfig.History)+1 != len(childConfig.History) {
			continue
		}
		if historiesDiffer(parentConfig.History, childConfig.History[:len(parentConfig.History)]) {
			continue
		}
		children = append(children, child.ID)
		if max > 0 && len(children) >= max {
			break
		}
	}
	return children, nil
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

func getDecryptConfig(decryptionKeys []string) (*encconfig.DecryptConfig, error) {
	decConfig := &encconfig.DecryptConfig{}
	if len(decryptionKeys) > 0 {
		// decryption
		dcc, err := enchelpers.CreateCryptoConfig([]string{}, decryptionKeys)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid decryption keys")
		}
		cc := encconfig.CombineCryptoConfigs([]encconfig.CryptoConfig{dcc})
		decConfig = cc.DecryptConfig
	}

	return decConfig, nil
}

func getEncryptConfig(encryptionKeys []string, encryptLayers []int) (*encconfig.EncryptConfig, *[]int, error) {
	var encLayers *[]int
	var encConfig *encconfig.EncryptConfig

	if len(encryptionKeys) > 0 {
		// encryption
		encLayers = &encryptLayers
		ecc, err := enchelpers.CreateCryptoConfig(encryptionKeys, []string{})
		if err != nil {
			return nil, nil, errors.Wrapf(err, "invalid encryption keys")
		}
		cc := encconfig.CombineCryptoConfigs([]encconfig.CryptoConfig{ecc})
		encConfig = cc.EncryptConfig
	}
	return encConfig, encLayers, nil
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
