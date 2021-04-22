package libimage

import (
	"context"
	"path/filepath"
	"sort"
	"time"

	libimageTypes "github.com/containers/common/libimage/types"
	"github.com/containers/image/v5/docker/reference"
	"github.com/containers/image/v5/manifest"
	storageTransport "github.com/containers/image/v5/storage"
	"github.com/containers/image/v5/types"
	"github.com/containers/storage"
	"github.com/hashicorp/go-multierror"
	"github.com/opencontainers/go-digest"
	ociv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Image represents an image in the containers storage and allows for further
// operations and data manipulation.
type Image struct {
	// Backwards pointer to the runtime.
	runtime *Runtime

	// Counterpart in the local containers storage.
	storageImage *storage.Image

	// Image reference to the containers storage.
	storageReference types.ImageReference

	// All fields in the below structure are cached.  They may be cleared
	// at any time.  When adding a new field, please make sure to clear
	// it in `(*Image).reload()`.
	cached struct {
		// Image source.  Cached for performance reasons.
		imageSource types.ImageSource
		// Inspect data we get from containers/image.
		partialInspectData *types.ImageInspectInfo
		// Fully assembled image data.
		completeInspectData *libimageTypes.ImageData
		// Corresponding OCI image.
		ociv1Image *ociv1.Image
	}
}

// reload the image and pessimitically clear all cached data.
func (i *Image) reload() error {
	logrus.Tracef("Reloading image %s", i.ID())
	img, err := i.runtime.store.Image(i.ID())
	if err != nil {
		return errors.Wrap(err, "error reloading image")
	}
	i.storageImage = img
	i.cached.imageSource = nil
	i.cached.partialInspectData = nil
	i.cached.completeInspectData = nil
	i.cached.ociv1Image = nil
	return nil
}

// Names returns associated names with the image which may be a mix of tags and
// digests.
func (i *Image) Names() []string {
	return i.storageImage.Names
}

// StorageImage returns the underlying storage.Image.
func (i *Image) StorageImage() *storage.Image {
	return i.storageImage
}

// NamesHistory returns a string array of names previously associated with the
// image, which may be a mixture of tags and digests.
func (i *Image) NamesHistory() []string {
	return i.storageImage.NamesHistory
}

// ID returns the ID of the image.
func (i *Image) ID() string {
	return i.storageImage.ID
}

// Digest is a digest value that we can use to locate the image, if one was
// specified at creation-time.
func (i *Image) Digest() digest.Digest {
	return i.storageImage.Digest
}

// Digests is a list of digest values of the image's manifests, and possibly a
// manually-specified value, that we can use to locate the image.  If Digest is
// set, its value is also in this list.
func (i *Image) Digests() []digest.Digest {
	return i.storageImage.Digests
}

// IsReadOnly returns whether the image is set read only.
func (i *Image) IsReadOnly() bool {
	return i.storageImage.ReadOnly
}

// IsDangling returns true if the image is dangling.  An image is considered
// dangling if no names are associated with it in the containers storage.
func (i *Image) IsDangling() bool {
	return len(i.Names()) == 0
}

// IsIntermediate returns true if the image is an intermediate image, that is
// a dangling image without children.
func (i *Image) IsIntermediate(ctx context.Context) (bool, error) {
	// If the image has tags, it's not an intermediate one.
	if !i.IsDangling() {
		return false, nil
	}
	children, err := i.getChildren(ctx, false)
	if err != nil {
		return false, err
	}
	// No tags, no children -> intermediate!
	return len(children) != 0, nil
}

// Created returns the time the image was created.
func (i *Image) Created() time.Time {
	return i.storageImage.Created
}

// Labels returns the label of the image.
func (i *Image) Labels(ctx context.Context) (map[string]string, error) {
	data, err := i.inspectInfo(ctx)
	if err != nil {
		isManifestList, listErr := i.isManifestList(ctx)
		if listErr != nil {
			err = errors.Wrapf(err, "fallback error checking whether image is a manifest list: %v", err)
		} else if isManifestList {
			logrus.Debugf("Ignoring error: cannot return labels for manifest list or image index %s", i.ID())
			return nil, nil
		}
		return nil, err
	}

	return data.Labels, nil
}

// TopLayer returns the top layer id as a string
func (i *Image) TopLayer() string {
	return i.storageImage.TopLayer
}

// Parent returns the parent image or nil if there is none
func (i *Image) Parent(ctx context.Context) (*Image, error) {
	tree, err := i.runtime.layerTree()
	if err != nil {
		return nil, err
	}
	return tree.parent(ctx, i)
}

// HasChildren returns indicates if the image has children.
func (i *Image) HasChildren(ctx context.Context) (bool, error) {
	children, err := i.getChildren(ctx, false)
	if err != nil {
		return false, err
	}
	return len(children) > 0, nil
}

// Children returns the image's children.
func (i *Image) Children(ctx context.Context) ([]*Image, error) {
	children, err := i.getChildren(ctx, true)
	if err != nil {
		return nil, err
	}
	return children, nil
}

// getChildren returns a list of imageIDs that depend on the image. If all is
// false, only the first child image is returned.
func (i *Image) getChildren(ctx context.Context, all bool) ([]*Image, error) {
	tree, err := i.runtime.layerTree()
	if err != nil {
		return nil, err
	}

	return tree.children(ctx, i, all)
}

// Containers returns a list of containers using the image.
func (i *Image) Containers() ([]string, error) {
	var containerIDs []string
	containers, err := i.runtime.store.Containers()
	if err != nil {
		return nil, err
	}
	imageID := i.ID()
	for i := range containers {
		if containers[i].ImageID == imageID {
			containerIDs = append(containerIDs, containers[i].ID)
		}
	}
	return containerIDs, nil
}

// removeContainers removes all containers using the image.
func (i *Image) removeContainers(fn RemoveContainerFunc) error {
	// Execute the custom removal func if specified.
	if fn != nil {
		logrus.Debugf("Removing containers of image %s with custom removal function", i.ID())
		return fn(i.ID())
	}

	containers, err := i.Containers()
	if err != nil {
		return err
	}

	logrus.Debugf("Removing containers of image %s from the local containers storage", i.ID())
	var multiE error
	for _, cID := range containers {
		if err := i.runtime.store.DeleteContainer(cID); err != nil {
			// If the container does not exist anymore, we're good.
			if errors.Cause(err) != storage.ErrContainerUnknown {
				multiE = multierror.Append(multiE, err)
			}
		}
	}

	return multiE
}

// RemoveContainerFunc allows for customizing the removal of containers using
// an image specified by imageID.
type RemoveContainerFunc func(imageID string) error

// RemoveImageOptions allow for customizing image removal.
type RemoveImageOptions struct {
	// Force will remove all containers from the local storage that are
	// using a removed image.  Use RemoveContainerFunc for a custom logic.
	// If set, all child images will be removed as well.
	Force bool
	// RemoveContainerFunc allows for a custom logic for removing
	// containers using a specific image.  By default, all containers in
	// the local containers storage will be removed (if Force is set).
	RemoveContainerFunc RemoveContainerFunc
}

// Remove removes the image along with all dangling parent images that no other
// image depends on.  The image must not be set read-only and not be used by
// containers.  Callers must make sure to remove containers before image
// removal and may use `(*Image).Containers()` to get a list of containers
// using the image.
//
// If the image is used by containers return storage.ErrImageUsedByContainer.
// Use force to remove these containers.
func (i *Image) Remove(ctx context.Context, options *RemoveImageOptions) error {
	logrus.Debugf("Removing image %s", i.ID())
	if i.IsReadOnly() {
		return errors.Errorf("cannot remove read-only image %q", i.ID())
	}

	if options == nil {
		options = &RemoveImageOptions{}
	}

	if options.Force {
		if err := i.removeContainers(options.RemoveContainerFunc); err != nil {
			return err
		}
	}

	// If there's a dangling parent that no other image depends on, remove
	// it recursively.
	parent, err := i.Parent(ctx)
	if err != nil {
		return err
	}

	if _, err := i.runtime.store.DeleteImage(i.ID(), true); err != nil {
		return err
	}
	delete(i.runtime.imageIDmap, i.ID())

	if parent == nil || !parent.IsDangling() {
		return nil
	}

	return parent.Remove(ctx, options)
}

// Tag the image with the specified name and store it in the local containers
// storage.  The name is normalized according to the rules of NormalizeName.
func (i *Image) Tag(name string) error {
	ref, err := NormalizeName(name)
	if err != nil {
		return errors.Wrapf(err, "error normalizing name %q", name)
	}

	logrus.Debugf("Tagging image %s with %q", i.ID(), ref.String())

	newNames := append(i.Names(), ref.String())
	if err := i.runtime.store.SetNames(i.ID(), newNames); err != nil {
		return err
	}

	return i.reload()
}

// Untag the image with the specified name and make the change persistent in
// the local containers storage.  The name is normalized according to the rules
// of NormalizeName.
func (i *Image) Untag(name string) error {
	ref, err := NormalizeName(name)
	if err != nil {
		return errors.Wrapf(err, "error normalizing name %q", name)
	}
	name = ref.String()

	removedName := false
	newNames := []string{}
	for _, n := range i.Names() {
		if n == name {
			removedName = true
			continue
		}
		newNames = append(newNames, n)
	}

	if !removedName {
		return nil
	}

	logrus.Debugf("Untagging %q from image %s", ref.String(), i.ID())

	if err := i.runtime.store.SetNames(i.ID(), newNames); err != nil {
		return err
	}

	return i.reload()
}

// RepoTags returns a string slice of repotags associated with the image.
func (i *Image) RepoTags() ([]string, error) {
	namedTagged, err := i.NamedTaggedRepoTags()
	if err != nil {
		return nil, err
	}
	repoTags := make([]string, len(namedTagged))
	for i := range namedTagged {
		repoTags[i] = namedTagged[i].String()
	}
	return repoTags, nil
}

// NammedTaggedRepoTags returns the repotags associated with the image as a
// slice of reference.NamedTagged.
func (i *Image) NamedTaggedRepoTags() ([]reference.NamedTagged, error) {
	var repoTags []reference.NamedTagged
	for _, name := range i.Names() {
		named, err := reference.ParseNormalizedNamed(name)
		if err != nil {
			return nil, err
		}
		if tagged, isTagged := named.(reference.NamedTagged); isTagged {
			repoTags = append(repoTags, tagged)
		}
	}
	return repoTags, nil
}

// RepoDigests returns a string array of repodigests associated with the image
func (i *Image) RepoDigests() ([]string, error) {
	var repoDigests []string
	added := make(map[string]struct{})

	for _, name := range i.Names() {
		for _, imageDigest := range append(i.Digests(), i.Digest()) {
			if imageDigest == "" {
				continue
			}

			named, err := reference.ParseNormalizedNamed(name)
			if err != nil {
				return nil, err
			}

			canonical, err := reference.WithDigest(reference.TrimNamed(named), imageDigest)
			if err != nil {
				return nil, err
			}

			if _, alreadyInList := added[canonical.String()]; !alreadyInList {
				repoDigests = append(repoDigests, canonical.String())
				added[canonical.String()] = struct{}{}
			}
		}
	}
	sort.Strings(repoDigests)
	return repoDigests, nil
}

// Mount the image with the specified mount options and label, both of which
// are directly passed down to the containers storage.  Returns the fully
// evaluated path to the mount point.
func (i *Image) Mount(ctx context.Context, mountOptions []string, mountLabel string) (string, error) {
	mountPoint, err := i.runtime.store.MountImage(i.ID(), mountOptions, mountLabel)
	if err != nil {
		return "", err
	}
	mountPoint, err = filepath.EvalSymlinks(mountPoint)
	if err != nil {
		return "", err
	}
	logrus.Debugf("Mounted image %s at %q", i.ID(), mountPoint)
	return mountPoint, nil
}

// Unmount the image.  Use force to ignore the reference counter and forcefully
// unmount.
func (i *Image) Unmount(force bool) error {
	logrus.Debugf("Unmounted image %s", i.ID())
	_, err := i.runtime.store.UnmountImage(i.ID(), force)
	return err
}

// MountPoint returns the fully-evaluated mount point of the image.  If the
// image isn't mounted, an empty string is returned.
func (i *Image) MountPoint() (string, error) {
	counter, err := i.runtime.store.Mounted(i.TopLayer())
	if err != nil {
		return "", err
	}

	if counter == 0 {
		return "", nil
	}

	layer, err := i.runtime.store.Layer(i.TopLayer())
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(layer.MountPoint)
}

// Size computes the size of the image layers and associated data.
func (i *Image) Size() (int64, error) {
	return i.runtime.store.ImageSize(i.ID())
}

// HasDifferentDigest returns true if the image specified by `remoteRef` has a
// different digest than the local one.  This check can be useful to check for
// updates on remote registries.
func (i *Image) HasDifferentDigest(ctx context.Context, remoteRef types.ImageReference) (bool, error) {
	// We need to account for the arch that the image uses.  It seems
	// common on ARM to tweak this option to pull the correct image.  See
	// github.com/containers/podman/issues/6613.
	inspectInfo, err := i.inspectInfo(ctx)
	if err != nil {
		return false, err
	}

	sys := i.runtime.systemContext
	sys.ArchitectureChoice = inspectInfo.Architecture
	// OS and variant may not be set, so let's check to avoid accidental
	// overrides of the runtime settings.
	if inspectInfo.Os != "" {
		sys.OSChoice = inspectInfo.Os
	}
	if inspectInfo.Variant != "" {
		sys.VariantChoice = inspectInfo.Variant
	}

	remoteImg, err := remoteRef.NewImage(ctx, &sys)
	if err != nil {
		return false, err
	}

	rawManifest, _, err := remoteImg.Manifest(ctx)
	if err != nil {
		return false, err
	}

	remoteDigest, err := manifest.Digest(rawManifest)
	if err != nil {
		return false, err
	}

	return i.Digest().String() != remoteDigest.String(), nil
}

// driverData gets the driver data from the store on a layer
func (i *Image) driverData() (*libimageTypes.DriverData, error) {
	store := i.runtime.store
	layerID := i.TopLayer()
	driver, err := store.GraphDriver()
	if err != nil {
		return nil, err
	}
	metaData, err := driver.Metadata(layerID)
	if err != nil {
		return nil, err
	}
	if mountTimes, err := store.Mounted(layerID); mountTimes == 0 || err != nil {
		delete(metaData, "MergedDir")
	}
	return &libimageTypes.DriverData{
		Name: driver.String(),
		Data: metaData,
	}, nil
}

// StorageReference returns the image's reference to the containers storage
// using the image ID.
func (i *Image) StorageReference() (types.ImageReference, error) {
	if i.storageReference != nil {
		return i.storageReference, nil
	}
	ref, err := storageTransport.Transport.ParseStoreReference(i.runtime.store, "@"+i.ID())
	if err != nil {
		return nil, err
	}
	i.storageReference = ref
	return ref, nil
}

// isManifestList returns true if the image is a manifest list (Docker) or an
// image index (OCI).  This information may be useful to make certain execution
// paths more robust.
// NOTE: please use this function only to optimize specific execution paths.
// In general, errors should only be suppressed when necessary.
func (i *Image) isManifestList(ctx context.Context) (bool, error) {
	ref, err := i.StorageReference()
	if err != nil {
		return false, err
	}
	imgRef, err := ref.NewImageSource(ctx, &i.runtime.systemContext)
	if err != nil {
		return false, err
	}
	_, manifestType, err := imgRef.GetManifest(ctx, nil)
	if err != nil {
		return false, err
	}
	return manifest.MIMETypeIsMultiImage(manifestType), nil
}

// source returns the possibly cached image reference.
func (i *Image) source(ctx context.Context) (types.ImageSource, error) {
	if i.cached.imageSource != nil {
		return i.cached.imageSource, nil
	}
	ref, err := i.StorageReference()
	if err != nil {
		return nil, err
	}
	src, err := ref.NewImageSource(ctx, &i.runtime.systemContext)
	if err != nil {
		return nil, err
	}
	i.cached.imageSource = src
	return src, nil
}

// getImageDigest creates an image object and uses the hex value of the digest as the image ID
// for parsing the store reference
func getImageDigest(ctx context.Context, src types.ImageReference, sys *types.SystemContext) (string, error) {
	newImg, err := src.NewImage(ctx, sys)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := newImg.Close(); err != nil {
			logrus.Errorf("failed to close image: %q", err)
		}
	}()
	imageDigest := newImg.ConfigInfo().Digest
	if err = imageDigest.Validate(); err != nil {
		return "", errors.Wrapf(err, "error getting config info")
	}
	return "@" + imageDigest.Hex(), nil
}
