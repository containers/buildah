package libimage

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/containers/image/v5/docker/reference"
	"github.com/containers/image/v5/pkg/shortnames"
	storageTransport "github.com/containers/image/v5/storage"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/image/v5/types"
	"github.com/containers/storage"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// RuntimeOptions allow for creating a customized Runtime.
type RuntimeOptions struct {
	SystemContext *types.SystemContext
}

// setRegistriesConfPath sets the registries.conf path for the specified context.
func setRegistriesConfPath(systemContext *types.SystemContext) {
	if systemContext.SystemRegistriesConfPath != "" {
		return
	}
	if envOverride, ok := os.LookupEnv("CONTAINERS_REGISTRIES_CONF"); ok {
		systemContext.SystemRegistriesConfPath = envOverride
		return
	}
	if envOverride, ok := os.LookupEnv("REGISTRIES_CONFIG_PATH"); ok {
		systemContext.SystemRegistriesConfPath = envOverride
		return
	}
}

// Runtime is responsible for image management and storing them in a containers
// storage.
type Runtime struct {
	// Underlying storage store.
	store storage.Store
	// Global system context.  No pointer to simplify copying and modifying
	// it.
	systemContext types.SystemContext
	// maps an image ID to an Image pointer.  Allows for aggressive
	// caching.
	imageIDmap map[string]*Image
}

// RuntimeFromStore returns a Runtime for the specified store.
func RuntimeFromStore(store storage.Store, options *RuntimeOptions) (*Runtime, error) {
	if options == nil {
		options = &RuntimeOptions{}
	}

	var systemContext types.SystemContext
	if options.SystemContext != nil {
		systemContext = *options.SystemContext
	} else {
		systemContext = types.SystemContext{}
	}

	setRegistriesConfPath(&systemContext)

	if systemContext.BlobInfoCacheDir == "" {
		systemContext.BlobInfoCacheDir = filepath.Join(store.GraphRoot(), "cache")
	}

	return &Runtime{
		store:         store,
		systemContext: systemContext,
		imageIDmap:    make(map[string]*Image),
	}, nil
}

// RuntimeFromStoreOptions returns a return for the specified store options.
func RuntimeFromStoreOptions(runtimeOptions *RuntimeOptions, storeOptions *storage.StoreOptions) (*Runtime, error) {
	if storeOptions == nil {
		storeOptions = &storage.StoreOptions{}
	}
	store, err := storage.GetStore(*storeOptions)
	if err != nil {
		return nil, err
	}
	storageTransport.Transport.SetStore(store)
	return RuntimeFromStore(store, runtimeOptions)
}

// Shutdown attempts to free any kernel resources which are being used by the
// underlying driver.  If "force" is true, any mounted (i.e., in use) layers
// are unmounted beforehand.  If "force" is not true, then layers being in use
// is considered to be an error condition.
func (r *Runtime) Shutdown(force bool) error {
	_, err := r.store.Shutdown(force)
	return err
}

// storageToImage transforms a storage.Image to an Image.
func (r *Runtime) storageToImage(storageImage *storage.Image, ref types.ImageReference) *Image {
	image, exists := r.imageIDmap[storageImage.ID]
	if exists {
		return image
	}
	image = &Image{
		runtime:          r,
		storageImage:     storageImage,
		storageReference: ref,
	}
	r.imageIDmap[storageImage.ID] = image
	return image
}

// Exists returns true if the specicifed image exists in the local containers
// storage.
func (r *Runtime) Exists(name string) (bool, error) {
	image, _, err := r.LookupImage(name, nil)
	return image != nil, err
}

// LookupImageOptions allow for customizing local image lookups.
type LookupImageOptions struct {
	// If set, the image will be purely looked up by name.  No matching to
	// the current platform will be performed.  This can be helpful when
	// the platform does not matter, for instance, for image removal.
	IgnorePlatform bool
}

// Lookup Image looks up `name` in the local container storage matching the
// specified SystemContext.  Returns the image and the name it has been found
// with.  Returns nil if no image has been found.  Note that name may also use
// the `containers-storage:` prefix used to refer to the containers-storage
// transport.
//
// If the specified name uses the `containers-storage` transport, the resolved
// name is empty.
func (r *Runtime) LookupImage(name string, options *LookupImageOptions) (*Image, string, error) {
	logrus.Debugf("Looking up image %q in local containers storage", name)

	if options == nil {
		options = &LookupImageOptions{}
	}

	// If needed extract the name sans transport.
	storageRef, err := alltransports.ParseImageName(name)
	if err == nil {
		if storageRef.Transport().Name() != storageTransport.Transport.Name() {
			return nil, "", errors.Errorf("unsupported transport %q for looking up local images", storageRef.Transport().Name())
		}
		img, err := storageTransport.Transport.GetStoreImage(r.store, storageRef)
		if err != nil {
			return nil, "", err
		}
		logrus.Debugf("Found image %q in local containers storage (%s)", name, storageRef.StringWithinTransport())
		return r.storageToImage(img, storageRef), "", nil
	}

	byDigest := false
	if strings.HasPrefix(name, "sha256:") {
		byDigest = true
		name = strings.TrimPrefix(name, "sha256:")
	}

	// Anonymouns function to lookup the provided image in the storage and
	// check whether it's matching the system context.
	findImage := func(input string) (*Image, error) {
		img, err := r.store.Image(input)
		if err != nil && errors.Cause(err) != storage.ErrImageUnknown {
			return nil, err
		}
		if img == nil {
			return nil, nil
		}
		ref, err := storageTransport.Transport.ParseStoreReference(r.store, img.ID)
		if err != nil {
			return nil, err
		}

		if options.IgnorePlatform {
			logrus.Debugf("Found image %q as %q in local containers storage", name, input)
			return r.storageToImage(img, ref), nil
		}

		matches, err := imageReferenceMatchesContext(context.Background(), ref, &r.systemContext)
		if err != nil {
			return nil, err
		}
		if !matches {
			return nil, nil
		}
		// Also print the string within the storage transport.  That
		// may aid in debugging when using additional stores since we
		// see explicitly where the store is and which driver (options)
		// are used.
		logrus.Debugf("Found image %q as %q in local containers storage (%s)", name, input, ref.StringWithinTransport())
		return r.storageToImage(img, ref), nil
	}

	// First, check if we have an exact match in the storage. Maybe an ID
	// or a fully-qualified image name.
	img, err := findImage(name)
	if err != nil {
		return nil, "", err
	}
	if img != nil {
		return img, name, nil
	}

	// If the name clearly referred to a local image, there's nothing we can
	// do anymore.
	if storageRef != nil || byDigest {
		return nil, "", nil
	}

	// Second, try out the candidates as resolved by shortnames. This takes
	// "localhost/" prefixed images into account as well.
	candidates, err := shortnames.ResolveLocally(&r.systemContext, name)
	if err != nil {
		return nil, "", err
	}
	// Backwards compat: normalize to docker.io as some users may very well
	// rely on that.
	dockerNamed, err := reference.ParseDockerRef(name)
	if err != nil {
		return nil, "", errors.Wrap(err, "error normalizing to docker.io")
	}

	candidates = append(candidates, dockerNamed)
	for _, candidate := range candidates {
		img, err := findImage(candidate.String())
		if err != nil {
			return nil, "", err
		}
		if img != nil {
			return img, candidate.String(), err
		}
	}

	return nil, "", nil
}

// imageReferenceMatchesContext return true if the specified reference matches
// the platform (os, arch, variant) as specified by the system context.
func imageReferenceMatchesContext(ctx context.Context, ref types.ImageReference, sys *types.SystemContext) (bool, error) {
	if sys == nil {
		return true, nil
	}
	img, err := ref.NewImage(ctx, sys)
	if err != nil {
		return false, err
	}
	defer img.Close()
	data, err := img.Inspect(ctx)
	if err != nil {
		return false, err
	}
	osChoice := sys.OSChoice
	if osChoice == "" {
		osChoice = runtime.GOOS
	}
	arch := sys.ArchitectureChoice
	if arch == "" {
		arch = runtime.GOARCH
	}
	if osChoice == data.Os && arch == data.Architecture {
		if sys.VariantChoice == "" || sys.VariantChoice == data.Variant {
			return true, nil
		}
	}
	return false, nil
}

// ListImagesOptions allow for customizing listing images.
type ListImagesOptions struct {
	// Filters to filter the listed images.  Supported filters are
	// * after,before,since=image
	// * dangling=true,false
	// * intermediate=true,false (useful for pruning images)
	// * id=id
	// * label=key[=value]
	// * readonly=true,false
	// * reference=name[:tag] (wildcards allowed)
	Filters []string
}

// ListImages lists images in the local container storage.  If names are
// specified, only images with the specified names are looked up and filtered.
func (r *Runtime) ListImages(ctx context.Context, names []string, options *ListImagesOptions) ([]*Image, error) {
	if options == nil {
		options = &ListImagesOptions{}
	}

	var images []*Image
	if len(names) > 0 {
		lookupOpts := LookupImageOptions{IgnorePlatform: true}
		for _, name := range names {
			image, _, err := r.LookupImage(name, &lookupOpts)
			if err != nil {
				return nil, err
			}
			if image == nil {
				return nil, errors.Wrap(storage.ErrImageUnknown, name)
			}
			images = append(images, image)
		}
	} else {
		storageImages, err := r.store.Images()
		if err != nil {
			return nil, err
		}
		for i := range storageImages {
			images = append(images, r.storageToImage(&storageImages[i], nil))
		}
	}

	var filters []filterFunc
	if len(options.Filters) > 0 {
		compiledFilters, err := r.compileImageFilters(ctx, options.Filters)
		if err != nil {
			return nil, err
		}
		filters = append(filters, compiledFilters...)
	}

	return filterImages(images, filters)
}

// RemoveImagesOptions allow for customizing image removal.
type RemoveImagesOptions struct {
	RemoveImageOptions

	// Filters to filter the removed images.  Supported filters are
	// * after,before,since=image
	// * dangling=true,false
	// * intermediate=true,false (useful for pruning images)
	// * id=id
	// * label=key[=value]
	// * readonly=true,false
	// * reference=name[:tag] (wildcards allowed)
	Filters []string
}

// RemoveImages removes images specified by names.  All images are expected to
// exist in the local containers storage.
//
// If an image has more names than one name, the image will be untagged with
// the specified name.  RemoveImages returns a slice of untagged and removed
// images.
func (r *Runtime) RemoveImages(ctx context.Context, names []string, options *RemoveImagesOptions) (untagged, removed []string, rmError error) {
	if options == nil {
		options = &RemoveImagesOptions{}
	}

	// deleteMe bundles an image with a possibly empty string value it has
	// been looked up with.  The string value is required to implement the
	// untagging logic.
	type deleteMe struct {
		image *Image
		name  string
	}

	var images []*deleteMe
	switch {
	case len(names) > 0:
		lookupOptions := LookupImageOptions{IgnorePlatform: true}
		for _, name := range names {
			img, resolvedName, err := r.LookupImage(name, &lookupOptions)
			if err != nil {
				return nil, nil, err
			}
			if img == nil {
				return nil, nil, errors.Wrap(storage.ErrImageUnknown, name)
			}
			images = append(images, &deleteMe{image: img, name: resolvedName})
		}
		if len(images) == 0 {
			return nil, nil, errors.New("no images found")
		}

	case len(options.Filters) > 0:
		filteredImages, err := r.ListImages(ctx, nil, &ListImagesOptions{Filters: options.Filters})
		if err != nil {
			return nil, nil, err
		}
		for _, img := range filteredImages {
			images = append(images, &deleteMe{image: img})
		}
	}

	// Now remove the images.
	for _, delete := range images {
		numNames := len(delete.image.Names())

		skipRemove := false
		if len(names) > 0 {
			hasChildren, err := delete.image.HasChildren(ctx)
			if err != nil {
				rmError = multierror.Append(rmError, err)
				continue
			}
			skipRemove = hasChildren
		}

		if delete.name != "" {
			untagged = append(untagged, delete.name)
		}

		mustUntag := !options.Force && delete.name != "" && (numNames > 1 || skipRemove)
		if mustUntag {
			if err := delete.image.Untag(delete.name); err != nil {
				rmError = multierror.Append(rmError, err)
				continue
			}
			// If the untag did not reduce the image names, name
			// must have been an ID in which case we should throw
			// an error. UNLESS there is only one tag left.
			newNumNames := len(delete.image.Names())
			if newNumNames == numNames && newNumNames != 1 {
				err := errors.Errorf("unable to delete image %q by ID with more than one tag (%s): use force removal", delete.image.ID(), delete.image.Names())
				rmError = multierror.Append(rmError, err)
				continue
			}

			// If we deleted the last tag/name, we can continue
			// removing the image.  Otherwise, we mark it as
			// untagged and need to continue.
			if newNumNames >= 1 || skipRemove {
				continue
			}
		}

		if err := delete.image.Remove(ctx, &options.RemoveImageOptions); err != nil {
			// If the image does not exist (anymore) we are good.
			// We already performed a presence check in the image
			// look up when `names` are specified.
			if errors.Cause(err) != storage.ErrImageUnknown {
				rmError = multierror.Append(rmError, err)
				continue
			}
		}
		removed = append(removed, delete.image.ID())
	}

	return untagged, removed, rmError
}
