package buildah

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/containers/storage"
	"github.com/containers/storage/pkg/ioutils"
	"github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/projectatomic/buildah/docker"
)

const (
	// Package is the name of this package, used in help output and to
	// identify working containers.
	Package = "buildah"
	// Version for the Package
	Version       = "0.0.1"
	containerType = Package + " 0.0.1"
	stateFile     = Package + ".json"
)

const (
	// PullIfMissing is one of the values that BuilderOptions.PullPolicy
	// can take, signalling that the source image should be pulled from a
	// registry if a local copy of it is not already present.
	PullIfMissing = iota
	// PullAlways is one of the values that BuilderOptions.PullPolicy can
	// take, signalling that a fresh, possibly updated, copy of the image
	// should be pulled from a registry before the build proceeds.
	PullAlways
	// PullNever is one of the values that BuilderOptions.PullPolicy can
	// take, signalling that the source image should not be pulled from a
	// registry if a local copy of it is not already present.
	PullNever
)

// Builder objects are used to represent containers which are being used to
// build images.  They also carry potential updates which will be applied to
// the image's configuration when the container's contents are used to build an
// image.
type Builder struct {
	store storage.Store

	// Type is used to help identify a build container's metadata.  It
	// should not be modified.
	Type string `json:"type"`
	// FromImage is the name of the source image which was used to create
	// the container, if one was used.  It should not be modified.
	FromImage string `json:"image,omitempty"`
	// FromImageID is the ID of the source image which was used to create
	// the container, if one was used.  It should not be modified.
	FromImageID string `json:"image-id"`
	// Config is the source image's configuration.  It should not be
	// modified.
	Config []byte `json:"config,omitempty"`
	// Manifest is the source image's manifest.  It should not be modified.
	Manifest []byte `json:"manifest,omitempty"`

	// Container is the name of the build container.  It should not be modified.
	Container string `json:"container-name,omitempty"`
	// ContainerID is the ID of the build container.  It should not be modified.
	ContainerID string `json:"container-id,omitempty"`
	// MountPoint is the last location where the container's root
	// filesystem was mounted.  It should not be modified.
	MountPoint string `json:"mountpoint,omitempty"`
	// Mounts is a list of places where the container's root filesystem has
	// been mounted.  It should not be modified.
	Mounts []string `json:"mounts,omitempty"`

	// ImageAnnotations is a set of key-value pairs which is stored in the
	// image's manifest.
	ImageAnnotations map[string]string `json:"annotations,omitempty"`
	// ImageCreatedBy is a description of how this container was built.
	ImageCreatedBy string `json:"created-by,omitempty"`

	// Image metadata and runtime settings, in multiple formats.
	OCIv1  v1.Image     `json:"ociv1,omitempty"`
	Docker docker.Image `json:"docker,omitempty"`
}

// BuilderOptions are used to initialize a new Builder.
type BuilderOptions struct {
	// FromImage is the name of the image which should be used as the
	// starting point for the container.  It can be set to an empty value
	// or "scratch" to indicate that the container should not be based on
	// an image.
	FromImage string
	// Container is a desired name for the build container.
	Container string
	// PullPolicy decides whether or not we should pull the image that
	// we're using as a base image.  It should be PullIfMissing,
	// PullAlways, or PullNever.
	PullPolicy int
	// Registry is a value which is prepended to the image's name, if it
	// needs to be pulled and the image name alone can not be resolved to a
	// reference to a source image.
	Registry string
	// Mount signals to NewBuilder() that the container should be mounted
	// immediately.
	Mount bool
	// SignaturePolicyPath specifies an override location for the signature
	// policy which should be used for verifying the new image as it is
	// being written.  Except in specific circumstances, no value should be
	// specified, indicating that the shared, system-wide default policy
	// should be used.
	SignaturePolicyPath string
	// ReportWriter is an io.Writer which will be used to log the reading
	// of the source image from a registry, if we end up pulling the image.
	ReportWriter io.Writer
}

// ImportOptions are used to initialize a Builder from an existing container
// which was created elsewhere.
type ImportOptions struct {
	// Container is the name of the build container.
	Container string
	// SignaturePolicyPath specifies an override location for the signature
	// policy which should be used for verifying the new image as it is
	// being written.  Except in specific circumstances, no value should be
	// specified, indicating that the shared, system-wide default policy
	// should be used.
	SignaturePolicyPath string
}

// ImportFromImageOptions are used to initialize a Builder from an image.
type ImportFromImageOptions struct {
	// Image is the name or ID of the image we'd like to examine.
	Image string
	// SignaturePolicyPath specifies an override location for the signature
	// policy which should be used for verifying the new image as it is
	// being written.  Except in specific circumstances, no value should be
	// specified, indicating that the shared, system-wide default policy
	// should be used.
	SignaturePolicyPath string
}

// NewBuilder creates a new build container.
func NewBuilder(store storage.Store, options BuilderOptions) (*Builder, error) {
	return newBuilder(store, options)
}

// ImportBuilder creates a new build configuration using an already-present
// container.
func ImportBuilder(store storage.Store, options ImportOptions) (*Builder, error) {
	return importBuilder(store, options)
}

// ImportBuilderFromImage creates a new builder configuration using an image.
// The returned object can be modified and examined, but it can not be saved
// or committed because it is not associated with a working container.
func ImportBuilderFromImage(store storage.Store, options ImportFromImageOptions) (*Builder, error) {
	return importBuilderFromImage(store, options)
}

// OpenBuilder loads information about a build container given its name or ID.
func OpenBuilder(store storage.Store, container string) (*Builder, error) {
	cdir, err := store.ContainerDirectory(container)
	if err != nil {
		return nil, err
	}
	buildstate, err := ioutil.ReadFile(filepath.Join(cdir, stateFile))
	if err != nil {
		return nil, err
	}
	b := &Builder{}
	err = json.Unmarshal(buildstate, &b)
	if err != nil {
		return nil, err
	}
	if b.Type != containerType {
		return nil, fmt.Errorf("container is not a %s container", Package)
	}
	b.store = store
	b.fixupConfig()
	return b, nil
}

// OpenBuilderByPath loads information about a build container given a
// path to the container's root filesystem
func OpenBuilderByPath(store storage.Store, path string) (*Builder, error) {
	containers, err := store.Containers()
	if err != nil {
		return nil, err
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	builderMatchesPath := func(b *Builder, path string) bool {
		if b.MountPoint == path {
			return true
		}
		for _, m := range b.Mounts {
			if m == path {
				return true
			}
		}
		return false
	}
	for _, container := range containers {
		cdir, err := store.ContainerDirectory(container.ID)
		if err != nil {
			return nil, err
		}
		buildstate, err := ioutil.ReadFile(filepath.Join(cdir, stateFile))
		if err != nil {
			return nil, err
		}
		b := &Builder{}
		err = json.Unmarshal(buildstate, &b)
		if err == nil && b.Type == containerType && builderMatchesPath(b, abs) {
			b.store = store
			b.fixupConfig()
			return b, nil
		}
	}
	return nil, storage.ErrContainerUnknown
}

// OpenAllBuilders loads all containers which have a state file that we use in
// their data directory, typically so that they can be listed.
func OpenAllBuilders(store storage.Store) (builders []*Builder, err error) {
	containers, err := store.Containers()
	if err != nil {
		return nil, err
	}
	for _, container := range containers {
		cdir, err := store.ContainerDirectory(container.ID)
		if err != nil {
			return nil, err
		}
		buildstate, err := ioutil.ReadFile(filepath.Join(cdir, stateFile))
		if err != nil && os.IsNotExist(err) {
			continue
		}
		b := &Builder{}
		err = json.Unmarshal(buildstate, &b)
		if err == nil && b.Type == containerType {
			b.store = store
			b.fixupConfig()
			builders = append(builders, b)
		}
	}
	return builders, nil
}

// Save saves the builder's current state to the build container's metadata.
// This should not need to be called directly, as other methods of the Builder
// object take care of saving their state.
func (b *Builder) Save() error {
	buildstate, err := json.Marshal(b)
	if err != nil {
		return err
	}
	cdir, err := b.store.ContainerDirectory(b.ContainerID)
	if err != nil {
		return err
	}
	return ioutils.AtomicWriteFile(filepath.Join(cdir, stateFile), buildstate, 0600)
}
